package scheduler

import (
	"context"
	"sync"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

// CircuitBreaker 熔断器接口
type CircuitBreaker interface {
	// CanExecute 检查是否可以执行请求
	CanExecute(ctx context.Context, providerID uint) (bool, error)
	// RecordSuccess 记录成功
	RecordSuccess(ctx context.Context, providerID uint) error
	// RecordFailure 记录失败
	RecordFailure(ctx context.Context, providerID uint, err error) error
	// GetState 获取熔断状态
	GetState(ctx context.Context, providerID uint) (model.CircuitState, error)
	// ForceOpen 强制打开熔断器
	ForceOpen(ctx context.Context, providerID uint, duration time.Duration, reason string) error
	// ForceClose 强制关闭熔断器
	ForceClose(ctx context.Context, providerID uint) error
	// GetCircuitInfo 获取熔断器信息
	GetCircuitInfo(ctx context.Context, providerID uint) (*CircuitInfo, error)
}

// CircuitInfo 熔断器信息
type CircuitInfo struct {
	ProviderID       uint               `json:"provider_id"`
	State            model.CircuitState `json:"state"`
	FailureCount     int                `json:"failure_count"`
	FailureThreshold int                `json:"failure_threshold"`
	LastFailureAt    *time.Time         `json:"last_failure_at,omitempty"`
	OpenUntil        *time.Time         `json:"open_until,omitempty"`
	RecoveryTimeout  int                `json:"recovery_timeout_seconds"`
}

// CircuitBreakerConfig 熔断器配置
type CircuitBreakerConfig struct {
	DefaultFailureThreshold int           // 默认失败阈值
	DefaultRecoveryTimeout  time.Duration // 默认恢复超时
	HalfOpenMaxRequests     int           // 半开状态最大请求数
	SuccessThresholdToClose int           // 关闭熔断器所需的连续成功数
}

// DefaultCircuitBreakerConfig 默认熔断器配置
func DefaultCircuitBreakerConfig() *CircuitBreakerConfig {
	return &CircuitBreakerConfig{
		DefaultFailureThreshold: 5,
		DefaultRecoveryTimeout:  30 * time.Second,
		HalfOpenMaxRequests:     3,
		SuccessThresholdToClose: 2,
	}
}

// circuitBreaker 熔断器实现
type circuitBreaker struct {
	providerRepo repository.ModelProviderRepository
	metricsRepo  repository.ProviderMetricsRepository
	stateStore   RuntimeStateStore
	config       *CircuitBreakerConfig
}

// halfOpenCounter 半开状态计数器
type halfOpenCounter struct {
	requests  int
	successes int
	mu        sync.Mutex
}

// NewCircuitBreaker 创建熔断器
func NewCircuitBreaker(
	providerRepo repository.ModelProviderRepository,
	metricsRepo repository.ProviderMetricsRepository,
	stateStore RuntimeStateStore,
	config *CircuitBreakerConfig,
) CircuitBreaker {
	if config == nil {
		config = DefaultCircuitBreakerConfig()
	}
	return &circuitBreaker{
		providerRepo: providerRepo,
		metricsRepo:  metricsRepo,
		stateStore:   stateStore,
		config:       config,
	}
}

// CanExecute 检查是否可以执行请求
// ⭐ 核心方法：实现熔断器状态机
func (cb *circuitBreaker) CanExecute(ctx context.Context, providerID uint) (bool, error) {
	state, err := cb.stateStore.GetCircuitState(ctx, providerID)
	if err != nil {
		return false, err
	}

	switch state {
	case model.CircuitStateClosed:
		// 关闭状态：允许执行
		return true, nil

	case model.CircuitStateOpen:
		// 打开状态：不允许执行
		return false, nil

	case model.CircuitStateHalfOpen:
		// 尽量同步 DB 状态，便于管理端展示（半开状态是低频状态，写入成本可接受）
		_ = cb.providerRepo.HalfOpenCircuit(ctx, providerID)
		// 半开状态：限制请求数
		return cb.canExecuteHalfOpen(ctx, providerID), nil

	default:
		// 未知状态，默认允许
		return true, nil
	}
}

// canExecuteHalfOpen 检查半开状态是否可以执行
func (cb *circuitBreaker) canExecuteHalfOpen(ctx context.Context, providerID uint) bool {
	counter, err := cb.stateStore.GetHalfOpenCounter(ctx, providerID)
	if err != nil {
		return false
	}

	if counter.Requests >= cb.config.HalfOpenMaxRequests {
		return false
	}

	// 增加请求计数
	counter.Requests++
	_ = cb.stateStore.SetHalfOpenCounter(ctx, providerID, counter, 5*time.Minute)
	return true
}

// RecordSuccess 记录成功
func (cb *circuitBreaker) RecordSuccess(ctx context.Context, providerID uint) error {
	state, err := cb.stateStore.GetCircuitState(ctx, providerID)
	if err != nil {
		return err
	}

	switch state {
	case model.CircuitStateClosed:
		// 关闭状态：重置失败计数
		return cb.stateStore.ResetFailureCount(ctx, providerID)

	case model.CircuitStateHalfOpen:
		// 半开状态：检查是否可以关闭熔断器
		return cb.handleHalfOpenSuccess(ctx, providerID)

	default:
		return nil
	}
}

// handleHalfOpenSuccess 处理半开状态的成功
func (cb *circuitBreaker) handleHalfOpenSuccess(ctx context.Context, providerID uint) error {
	counter, err := cb.stateStore.GetHalfOpenCounter(ctx, providerID)
	if err != nil {
		return err
	}

	counter.Successes++

	// 检查是否达到关闭阈值
	if counter.Successes >= cb.config.SuccessThresholdToClose {
		// 删除计数器
		_ = cb.stateStore.DeleteHalfOpenCounter(ctx, providerID)

		// 关闭熔断器
		return cb.closeCircuit(ctx, providerID, "半开状态连续成功达到阈值")
	}

	// 更新计数器
	_ = cb.stateStore.SetHalfOpenCounter(ctx, providerID, counter, 5*time.Minute)
	return nil
}

// RecordFailure 记录失败
func (cb *circuitBreaker) RecordFailure(ctx context.Context, providerID uint, failErr error) error {
	state, err := cb.stateStore.GetCircuitState(ctx, providerID)
	if err != nil {
		return err
	}

	// 增加失败计数
	if _, err := cb.stateStore.IncrementFailureCount(ctx, providerID); err != nil {
		return err
	}

	switch state {
	case model.CircuitStateClosed:
		// 关闭状态：检查是否需要打开熔断器
		return cb.checkAndOpenCircuit(ctx, providerID, failErr)

	case model.CircuitStateHalfOpen:
		// 半开状态：立即打开熔断器
		return cb.openCircuitFromHalfOpen(ctx, providerID, failErr)

	default:
		return nil
	}
}

// checkAndOpenCircuit 检查并打开熔断器
func (cb *circuitBreaker) checkAndOpenCircuit(ctx context.Context, providerID uint, failErr error) error {
	provider, err := cb.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return err
	}
	if provider == nil {
		return nil
	}

	// 获取失败阈值
	threshold := provider.FailureThreshold
	if threshold <= 0 {
		threshold = cb.config.DefaultFailureThreshold
	}

	// 获取当前失败计数
	failureCount, err := cb.stateStore.GetFailureCount(ctx, providerID)
	if err != nil {
		return err
	}

	// 检查是否达到阈值
	if failureCount >= int64(threshold) {
		// 获取恢复超时
		recoveryTimeout := time.Duration(provider.RecoveryTimeoutSeconds) * time.Second
		if recoveryTimeout <= 0 {
			recoveryTimeout = cb.config.DefaultRecoveryTimeout
		}

		// 打开熔断器
		reason := "连续失败达到阈值"
		if failErr != nil {
			reason = reason + ": " + failErr.Error()
		}
		return cb.openCircuit(ctx, providerID, recoveryTimeout, reason)
	}

	return nil
}

// openCircuitFromHalfOpen 从半开状态打开熔断器
func (cb *circuitBreaker) openCircuitFromHalfOpen(ctx context.Context, providerID uint, failErr error) error {
	// 删除计数器
	_ = cb.stateStore.DeleteHalfOpenCounter(ctx, providerID)

	// 获取恢复超时
	provider, err := cb.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return err
	}

	recoveryTimeout := cb.config.DefaultRecoveryTimeout
	if provider != nil && provider.RecoveryTimeoutSeconds > 0 {
		recoveryTimeout = time.Duration(provider.RecoveryTimeoutSeconds) * time.Second
	}

	// 打开熔断器（使用更长的超时）
	reason := "半开状态失败"
	if failErr != nil {
		reason = reason + ": " + failErr.Error()
	}
	return cb.openCircuit(ctx, providerID, recoveryTimeout*2, reason)
}

// openCircuit 打开熔断器
func (cb *circuitBreaker) openCircuit(ctx context.Context, providerID uint, duration time.Duration, reason string) error {
	// 更新Redis状态
	if err := cb.stateStore.SetCircuitState(ctx, providerID, model.CircuitStateOpen, duration); err != nil {
		return err
	}

	// 同步 DB（供管理端查询/过滤）
	if err := cb.providerRepo.OpenCircuit(ctx, providerID, duration); err != nil {
		return err
	}

	// 记录熔断事件
	event := &model.CircuitEventRecord{
		ProviderID:      providerID,
		EventType:       "open",
		Reason:          reason,
		DurationSeconds: int(duration.Seconds()),
	}
	_ = cb.metricsRepo.CreateCircuitEvent(ctx, event)

	return nil
}

// closeCircuit 关闭熔断器
func (cb *circuitBreaker) closeCircuit(ctx context.Context, providerID uint, reason string) error {
	// 更新Redis状态
	if err := cb.stateStore.SetCircuitState(ctx, providerID, model.CircuitStateClosed, 0); err != nil {
		return err
	}

	// 重置失败计数
	_ = cb.stateStore.ResetFailureCount(ctx, providerID)

	// 同步 DB（供管理端查询/过滤）
	if err := cb.providerRepo.CloseCircuit(ctx, providerID); err != nil {
		return err
	}

	// 记录熔断事件
	event := &model.CircuitEventRecord{
		ProviderID: providerID,
		EventType:  "close",
		Reason:     reason,
	}
	_ = cb.metricsRepo.CreateCircuitEvent(ctx, event)

	return nil
}

// GetState 获取熔断状态
func (cb *circuitBreaker) GetState(ctx context.Context, providerID uint) (model.CircuitState, error) {
	return cb.stateStore.GetCircuitState(ctx, providerID)
}

// ForceOpen 强制打开熔断器
func (cb *circuitBreaker) ForceOpen(ctx context.Context, providerID uint, duration time.Duration, reason string) error {
	if duration <= 0 {
		duration = cb.config.DefaultRecoveryTimeout
	}
	return cb.openCircuit(ctx, providerID, duration, "手动强制打开: "+reason)
}

// ForceClose 强制关闭熔断器
func (cb *circuitBreaker) ForceClose(ctx context.Context, providerID uint) error {
	// 删除计数器
	_ = cb.stateStore.DeleteHalfOpenCounter(ctx, providerID)

	return cb.closeCircuit(ctx, providerID, "手动强制关闭")
}

// GetCircuitInfo 获取熔断器信息
func (cb *circuitBreaker) GetCircuitInfo(ctx context.Context, providerID uint) (*CircuitInfo, error) {
	provider, err := cb.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, nil
	}

	threshold := provider.FailureThreshold
	if threshold <= 0 {
		threshold = cb.config.DefaultFailureThreshold
	}

	recoveryTimeout := provider.RecoveryTimeoutSeconds
	if recoveryTimeout <= 0 {
		recoveryTimeout = int(cb.config.DefaultRecoveryTimeout.Seconds())
	}

	// 获取当前熔断状态
	state, _ := cb.stateStore.GetCircuitState(ctx, providerID)

	// 获取失败计数
	failureCount, _ := cb.stateStore.GetFailureCount(ctx, providerID)

	return &CircuitInfo{
		ProviderID:       providerID,
		State:            state,
		FailureCount:     int(failureCount),
		FailureThreshold: threshold,
		LastFailureAt:    provider.LastFailureAt,
		OpenUntil:        provider.CircuitOpenUntil,
		RecoveryTimeout:  recoveryTimeout,
	}, nil
}

// CircuitBreakerStats 熔断器统计
type CircuitBreakerStats struct {
	TotalProviders   int `json:"total_providers"`
	ClosedCount      int `json:"closed_count"`
	OpenCount        int `json:"open_count"`
	HalfOpenCount    int `json:"half_open_count"`
	RecentOpenEvents int `json:"recent_open_events"`
}

// GetStats 获取熔断器统计（需要在服务层实现）
func (cb *circuitBreaker) GetStats(ctx context.Context) (*CircuitBreakerStats, error) {
	// 这个方法需要在服务层实现，因为需要查询所有源头
	return nil, nil
}
