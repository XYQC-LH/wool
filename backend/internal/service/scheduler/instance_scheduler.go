package scheduler

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

// InstanceScheduler 实例调度器接口
// 在ProviderGroup下选择具体的ProviderInstance
type InstanceScheduler interface {
	// SelectInstance 选择实例
	SelectInstance(ctx context.Context, providerID uint, operation string) (*model.ProviderInstance, error)
	// SelectInstanceWithRetry 选择实例（带重试）
	SelectInstanceWithRetry(ctx context.Context, providerID uint, operation string, maxRetries int) (*model.ProviderInstance, error)
	// RecordInstanceResult 记录实例执行结果
	RecordInstanceResult(ctx context.Context, instanceID uint, success bool, latency time.Duration) error
	// GetInstanceStats 获取实例统计
	GetInstanceStats(ctx context.Context, instanceID uint) (*InstanceStats, error)
	// GetProviderInstanceStats 获取Provider下所有实例的统计
	GetProviderInstanceStats(ctx context.Context, providerID uint) (*ProviderInstanceStats, error)
	// AcquireInstanceSlot 获取实例槽位（并发控制）
	AcquireInstanceSlot(ctx context.Context, instanceID uint) (bool, error)
	// AcquireInstance 获取指定实例（校验状态/熔断/限流/并发，并占用并发槽位）
	AcquireInstance(ctx context.Context, instanceID uint, operation string) (*model.ProviderInstance, error)
	// ReleaseInstanceSlot 释放实例槽位
	ReleaseInstanceSlot(ctx context.Context, instanceID uint) error
}

// InstanceStats 实例统计
type InstanceStats struct {
	InstanceID         uint       `json:"instance_id"`
	TotalRequests      int64      `json:"total_requests"`
	SuccessRequests    int64      `json:"success_requests"`
	FailedRequests     int64      `json:"failed_requests"`
	SuccessRate        float64    `json:"success_rate"`
	AvgLatencyMs       int        `json:"avg_latency_ms"`
	CurrentConcurrency int64      `json:"current_concurrency"`
	LastUsedAt         *time.Time `json:"last_used_at,omitempty"`
}

// ProviderInstanceStats Provider实例统计
type ProviderInstanceStats struct {
	ProviderID      uint            `json:"provider_id"`
	TotalInstances  int64           `json:"total_instances"`
	ActiveInstances int64           `json:"active_instances"`
	InstanceStats   []InstanceStats `json:"instance_stats"`
}

// InstanceSchedulerConfig 实例调度器配置
type InstanceSchedulerConfig struct {
	EnableConcurrencyControl bool                      // 是否启用并发控制
	EnableRateLimit          bool                      // 是否启用速率限制
	EnableCircuitBreaker     bool                      // 是否启用实例级熔断
	CircuitFailureThreshold  int64                     // 连续失败阈值（达到即熔断）
	CircuitOpenSeconds       int                       // 熔断打开时长（秒）
	DefaultMaxConcurrency    int                       // 默认最大并发数
	DefaultRPMLimit          int64                     // 默认RPM限制
	DefaultTPMLimit          int64                     // 默认TPM限制
	SelectionStrategy        InstanceSelectionStrategy // 选择策略
}

// InstanceSelectionStrategy 实例选择策略
type InstanceSelectionStrategy string

const (
	StrategyWeightedRoundRobin InstanceSelectionStrategy = "weighted_round_robin" // 加权轮询
	StrategyLeastConnections   InstanceSelectionStrategy = "least_connections"    // 最少连接
	StrategyBestHealth         InstanceSelectionStrategy = "best_health"          // 最佳健康度
	StrategyRandom             InstanceSelectionStrategy = "random"               // 随机
	StrategyAdaptiveInstance   InstanceSelectionStrategy = "adaptive"             // 自适应
)

// DefaultInstanceSchedulerConfig 默认实例调度器配置
func DefaultInstanceSchedulerConfig() *InstanceSchedulerConfig {
	return &InstanceSchedulerConfig{
		EnableConcurrencyControl: true,
		EnableRateLimit:          true,
		EnableCircuitBreaker:     true,
		CircuitFailureThreshold:  3,
		CircuitOpenSeconds:       30,
		DefaultMaxConcurrency:    10,
		DefaultRPMLimit:          100,
		DefaultTPMLimit:          10000,
		SelectionStrategy:        StrategyAdaptiveInstance,
	}
}

// instanceScheduler 实例调度器实现
type instanceScheduler struct {
	instanceRepo      repository.ProviderInstanceRepository
	rateLimitRuleRepo repository.ProviderRateLimitRuleRepository
	stateStore        RuntimeStateStore
	config            *InstanceSchedulerConfig
	roundRobinIdx     map[uint]int // ProviderID -> 当前轮询索引
	mu                sync.RWMutex
}

// NewInstanceScheduler 创建实例调度器
func NewInstanceScheduler(
	instanceRepo repository.ProviderInstanceRepository,
	rateLimitRuleRepo repository.ProviderRateLimitRuleRepository,
	stateStore RuntimeStateStore,
	config *InstanceSchedulerConfig,
) InstanceScheduler {
	if config == nil {
		config = DefaultInstanceSchedulerConfig()
	}
	return &instanceScheduler{
		instanceRepo:      instanceRepo,
		rateLimitRuleRepo: rateLimitRuleRepo,
		stateStore:        stateStore,
		config:            config,
		roundRobinIdx:     make(map[uint]int),
	}
}

// SelectInstance 选择实例
// ⭐ 核心方法：根据配置的策略选择实例
func (s *instanceScheduler) SelectInstance(ctx context.Context, providerID uint, operation string) (*model.ProviderInstance, error) {
	return s.SelectInstanceWithRetry(ctx, providerID, operation, 3)
}

// SelectInstanceWithRetry 选择实例（带重试）
func (s *instanceScheduler) SelectInstanceWithRetry(ctx context.Context, providerID uint, operation string, maxRetries int) (*model.ProviderInstance, error) {
	operation = model.NormalizeOperation(operation)
	// 获取所有可用实例
	instances, err := s.instanceRepo.GetAvailableInstances(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("获取实例列表失败: %w", err)
	}

	if len(instances) == 0 {
		return nil, fmt.Errorf("没有可用的实例")
	}

	instances = s.filterCircuitOpenInstances(ctx, instances)
	if len(instances) == 0 {
		return nil, fmt.Errorf("没有可用的实例（均处于熔断中）")
	}

	// 根据策略选择实例
	var selectedInstance *model.ProviderInstance
	var lastError error

	for attempt := 0; attempt < maxRetries; attempt++ {
		selectedInstance, lastError = s.selectByStrategy(ctx, instances)
		if lastError != nil {
			continue
		}

		slotAcquired := false
		releaseSlotIfNeeded := func() {
			if !slotAcquired {
				return
			}
			_ = s.ReleaseInstanceSlot(ctx, selectedInstance.ID)
			slotAcquired = false
		}

		// 检查并发控制
		if s.config.EnableConcurrencyControl {
			acquired, err := s.AcquireInstanceSlot(ctx, selectedInstance.ID)
			if err != nil {
				lastError = err
				continue
			}
			if !acquired {
				// 实例已满，尝试下一个
				lastError = fmt.Errorf("实例 %d 并发已满", selectedInstance.ID)
				continue
			}
			slotAcquired = true
		}

		// 检查速率限制（实例级）
		if s.config.EnableRateLimit {
			allowed, err := s.applyLegacyInstanceRateLimit(ctx, selectedInstance)
			if err != nil {
				releaseSlotIfNeeded()
				lastError = err
				continue
			}
			if !allowed {
				releaseSlotIfNeeded()
				lastError = fmt.Errorf("实例 %d 限流已达到", selectedInstance.ID)
				continue
			}

			if operation != "" {
				allowed, err = s.applyInstanceRateLimitRules(ctx, selectedInstance, operation)
				if err != nil {
					// fail-open：限流检查错误不拦截调度
					allowed = true
				}
				if !allowed {
					releaseSlotIfNeeded()
					lastError = fmt.Errorf("实例 %d 限流已达到", selectedInstance.ID)
					continue
				}
			}
		}

		// 成功选择实例
		return selectedInstance, nil
	}

	return nil, fmt.Errorf("选择实例失败，最后错误: %w", lastError)
}

// selectByStrategy 根据策略选择实例
func (s *instanceScheduler) selectByStrategy(ctx context.Context, instances []*model.ProviderInstance) (*model.ProviderInstance, error) {
	s.mu.RLock()
	strategy := s.config.SelectionStrategy
	s.mu.RUnlock()

	switch strategy {
	case StrategyAdaptiveInstance:
		return s.selectAdaptive(ctx, instances)
	case StrategyWeightedRoundRobin:
		return s.selectWeightedRoundRobin(instances)
	case StrategyLeastConnections:
		return s.selectLeastConnections(ctx, instances)
	case StrategyBestHealth:
		return s.selectBestHealth(instances)
	case StrategyRandom:
		return s.selectRandom(instances)
	default:
		return s.selectWeightedRoundRobin(instances)
	}
}

func (s *instanceScheduler) selectAdaptive(ctx context.Context, instances []*model.ProviderInstance) (*model.ProviderInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("实例列表为空")
	}
	if s == nil {
		return instances[0], nil
	}
	if s.stateStore == nil {
		return s.selectBestHealth(instances)
	}

	type scoredInstance struct {
		instance      *model.ProviderInstance
		score         float64
		concurrency   int64
		healthScore   float64
		latencyScore  float64
		weightScore   float64
		successScore  float64
	}

	// 先收集并发上限，用于归一化
	maxConcurrency := int64(1)
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		currentConcurrency, err := s.stateStore.GetInstanceConcurrency(ctx, inst.ID)
		if err != nil {
			currentConcurrency = 0
		}
		if currentConcurrency > maxConcurrency {
			maxConcurrency = currentConcurrency
		}
	}

	scored := make([]scoredInstance, 0, len(instances))
	for _, inst := range instances {
		if inst == nil {
			continue
		}

		concurrency, err := s.stateStore.GetInstanceConcurrency(ctx, inst.ID)
		if err != nil {
			concurrency = 0
		}

		successRate := inst.GetSuccessRate() / 100.0
		if successRate < 0 {
			successRate = 0
		}
		if successRate > 1 {
			successRate = 1
		}

		latencyMs := inst.GetAvgLatency()
		latencyScore := 1.0 - math.Min(float64(latencyMs)/5000.0, 1.0)

		weight := inst.Weight
		if weight <= 0 {
			weight = 1
		}
		weightScore := math.Min(float64(weight)/100.0, 1.0)

		concurrencyPenalty := 0.0
		if maxConcurrency > 0 {
			concurrencyPenalty = float64(concurrency) / float64(maxConcurrency)
		}
		if concurrencyPenalty < 0 {
			concurrencyPenalty = 0
		}
		if concurrencyPenalty > 1 {
			concurrencyPenalty = 1
		}

		healthScore := successRate*0.6 + latencyScore*0.4
		total := healthScore*0.55 + weightScore*0.15 + (1.0-concurrencyPenalty)*0.30

		scored = append(scored, scoredInstance{
			instance:     inst,
			score:        total,
			concurrency:  concurrency,
			healthScore:  healthScore,
			latencyScore: latencyScore,
			weightScore:  weightScore,
			successScore: successRate,
		})
	}

	if len(scored) == 0 {
		return nil, fmt.Errorf("实例列表为空")
	}

	sort.Slice(scored, func(i, j int) bool {
		if scored[i].score != scored[j].score {
			return scored[i].score > scored[j].score
		}
		if scored[i].concurrency != scored[j].concurrency {
			return scored[i].concurrency < scored[j].concurrency
		}
		if scored[i].instance.Weight != scored[j].instance.Weight {
			return scored[i].instance.Weight > scored[j].instance.Weight
		}
		return scored[i].instance.ID < scored[j].instance.ID
	})

	return scored[0].instance, nil
}

// selectWeightedRoundRobin 加权轮询选择
func (s *instanceScheduler) selectWeightedRoundRobin(instances []*model.ProviderInstance) (*model.ProviderInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("实例列表为空")
	}

	// 计算总权重
	totalWeight := 0
	for _, instance := range instances {
		weight := instance.Weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
	}

	if totalWeight == 0 {
		return instances[0], nil
	}

	// 获取ProviderID（假设所有实例属于同一个Provider）
	providerID := instances[0].ProviderID

	// 获取当前轮询索引
	s.mu.Lock()
	idx, ok := s.roundRobinIdx[providerID]
	if !ok {
		idx = 0
	}
	s.roundRobinIdx[providerID] = (idx + 1) % len(instances)
	s.mu.Unlock()

	// 根据权重选择
	remainingWeight := idx % totalWeight
	for _, instance := range instances {
		weight := instance.Weight
		if weight <= 0 {
			weight = 1
		}
		if remainingWeight < weight {
			return instance, nil
		}
		remainingWeight -= weight
	}

	return instances[0], nil
}

// selectLeastConnections 最少连接选择
func (s *instanceScheduler) selectLeastConnections(ctx context.Context, instances []*model.ProviderInstance) (*model.ProviderInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("实例列表为空")
	}

	// 获取每个实例的当前并发数
	type instanceWithConcurrency struct {
		instance    *model.ProviderInstance
		concurrency int64
	}

	instancesWithConcurrency := make([]instanceWithConcurrency, 0, len(instances))
	for _, instance := range instances {
		concurrency, err := s.stateStore.GetInstanceConcurrency(ctx, instance.ID)
		if err != nil {
			concurrency = 0
		}
		instancesWithConcurrency = append(instancesWithConcurrency, instanceWithConcurrency{
			instance:    instance,
			concurrency: concurrency,
		})
	}

	// 按并发数排序
	sort.Slice(instancesWithConcurrency, func(i, j int) bool {
		return instancesWithConcurrency[i].concurrency < instancesWithConcurrency[j].concurrency
	})

	return instancesWithConcurrency[0].instance, nil
}

// selectBestHealth 最佳健康度选择
func (s *instanceScheduler) selectBestHealth(instances []*model.ProviderInstance) (*model.ProviderInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("实例列表为空")
	}

	// 计算每个实例的健康分数
	type instanceWithScore struct {
		instance *model.ProviderInstance
		score    float64
	}

	instancesWithScore := make([]instanceWithScore, 0, len(instances))
	for _, instance := range instances {
		score := s.calculateHealthScore(instance)
		instancesWithScore = append(instancesWithScore, instanceWithScore{
			instance: instance,
			score:    score,
		})
	}

	// 按健康分数排序
	sort.Slice(instancesWithScore, func(i, j int) bool {
		return instancesWithScore[i].score > instancesWithScore[j].score
	})

	return instancesWithScore[0].instance, nil
}

// calculateHealthScore 计算健康分数
func (s *instanceScheduler) calculateHealthScore(instance *model.ProviderInstance) float64 {
	// 成功率权重：0.5
	// 延迟权重：0.3
	// 权重权重：0.2

	successRate := instance.GetSuccessRate()
	avgLatency := instance.GetAvgLatency()
	weight := instance.Weight
	if weight <= 0 {
		weight = 1
	}

	// 归一化成功率（0-100）
	successScore := successRate / 100.0

	// 归一化延迟（假设最大延迟为5000ms）
	latencyScore := 1.0 - math.Min(float64(avgLatency)/5000.0, 1.0)

	// 归一化权重（假设最大权重为100）
	weightScore := math.Min(float64(weight)/100.0, 1.0)

	// 计算综合分数
	totalScore := successScore*0.5 + latencyScore*0.3 + weightScore*0.2

	return totalScore
}

// selectRandom 随机选择
func (s *instanceScheduler) selectRandom(instances []*model.ProviderInstance) (*model.ProviderInstance, error) {
	if len(instances) == 0 {
		return nil, fmt.Errorf("实例列表为空")
	}

	// 简单随机选择（使用时间戳作为种子）
	now := time.Now().UnixNano()
	idx := int(now) % len(instances)
	if idx < 0 {
		idx = -idx
	}

	return instances[idx], nil
}

// RecordInstanceResult 记录实例执行结果
func (s *instanceScheduler) RecordInstanceResult(ctx context.Context, instanceID uint, success bool, latency time.Duration) error {
	// 更新数据库统计
	err := s.instanceRepo.IncrementStats(ctx, instanceID, success, int64(latency.Milliseconds()))
	if err != nil {
		return fmt.Errorf("更新实例统计失败: %w", err)
	}

	if s.stateStore != nil && s.config != nil && s.config.EnableCircuitBreaker {
		openTTL := time.Duration(s.config.CircuitOpenSeconds) * time.Second
		if openTTL <= 0 {
			openTTL = 30 * time.Second
		}

		if success {
			_ = s.stateStore.ResetInstanceFailureCount(ctx, instanceID)
			_ = s.stateStore.SetInstanceCircuitState(ctx, instanceID, model.CircuitStateClosed, openTTL)
		} else {
			count, incErr := s.stateStore.IncrementInstanceFailureCount(ctx, instanceID)
			if incErr == nil && s.config.CircuitFailureThreshold > 0 && count >= s.config.CircuitFailureThreshold {
				_ = s.stateStore.SetInstanceCircuitState(ctx, instanceID, model.CircuitStateOpen, openTTL)
			}
		}
	}

	// 更新Redis健康指标
	err = s.stateStore.IncrementHealthMetric(ctx, instanceID, success, int64(latency.Milliseconds()))
	if err != nil {
		// Redis更新失败不影响主流程
		return nil
	}

	return nil
}

// GetInstanceStats 获取实例统计
func (s *instanceScheduler) GetInstanceStats(ctx context.Context, instanceID uint) (*InstanceStats, error) {
	// 从数据库获取实例
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("实例不存在")
	}

	// 获取当前并发数
	concurrency, _ := s.stateStore.GetInstanceConcurrency(ctx, instanceID)

	// 构建统计信息
	stats := &InstanceStats{
		InstanceID:         instance.ID,
		TotalRequests:      instance.TotalRequests,
		SuccessRequests:    instance.SuccessRequests,
		FailedRequests:     instance.FailedRequests,
		SuccessRate:        instance.GetSuccessRate(),
		AvgLatencyMs:       instance.GetAvgLatency(),
		CurrentConcurrency: concurrency,
	}

	return stats, nil
}

// GetProviderInstanceStats 获取Provider下所有实例的统计
func (s *instanceScheduler) GetProviderInstanceStats(ctx context.Context, providerID uint) (*ProviderInstanceStats, error) {
	// 获取所有实例
	instances, err := s.instanceRepo.GetByProviderID(ctx, providerID)
	if err != nil {
		return nil, err
	}

	// 构建统计信息
	stats := &ProviderInstanceStats{
		ProviderID:     providerID,
		TotalInstances: int64(len(instances)),
		InstanceStats:  make([]InstanceStats, 0, len(instances)),
	}

	for _, instance := range instances {
		instanceStats, err := s.GetInstanceStats(ctx, instance.ID)
		if err != nil {
			continue
		}

		if instance.Status == model.InstanceStatusActive {
			stats.ActiveInstances++
		}

		stats.InstanceStats = append(stats.InstanceStats, *instanceStats)
	}

	return stats, nil
}

// AcquireInstanceSlot 获取实例槽位（并发控制）
// ⭐ 修复：传递 MaxConcurrency 参数给 RuntimeStateStore
func (s *instanceScheduler) AcquireInstanceSlot(ctx context.Context, instanceID uint) (bool, error) {
	// 获取实例配置
	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return false, err
	}
	if instance == nil {
		return false, fmt.Errorf("实例不存在")
	}

	// 如果没有配置并发限制，直接返回成功
	if instance.MaxConcurrency <= 0 {
		return true, nil
	}

	// ⭐ 修复：传递 MaxConcurrency 参数，而不是硬编码
	// 使用Redis进行并发控制
	return s.stateStore.AcquireInstanceSlot(ctx, instanceID, int64(instance.MaxConcurrency))
}

func (s *instanceScheduler) AcquireInstance(ctx context.Context, instanceID uint, operation string) (*model.ProviderInstance, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if instanceID == 0 {
		return nil, fmt.Errorf("instanceID 不能为空")
	}

	instance, err := s.instanceRepo.GetByID(ctx, instanceID)
	if err != nil {
		return nil, err
	}
	if instance == nil {
		return nil, fmt.Errorf("实例不存在")
	}
	if !instance.IsAvailable() {
		return nil, fmt.Errorf("实例不可用")
	}

	if s != nil && s.stateStore != nil && s.config != nil && s.config.EnableCircuitBreaker {
		state, _ := s.stateStore.GetInstanceCircuitState(ctx, instanceID)
		if state == model.CircuitStateOpen {
			return nil, fmt.Errorf("实例处于熔断中")
		}
	}

	slotAcquired := false
	releaseSlotIfNeeded := func() {
		if !slotAcquired {
			return
		}
		_ = s.ReleaseInstanceSlot(ctx, instanceID)
		slotAcquired = false
	}

	if s != nil && s.config != nil && s.config.EnableConcurrencyControl {
		acquired, err := s.AcquireInstanceSlot(ctx, instanceID)
		if err != nil {
			return nil, err
		}
		if !acquired {
			return nil, fmt.Errorf("实例并发已满")
		}
		slotAcquired = true
	}

	if s != nil && s.stateStore != nil && s.config != nil && s.config.EnableRateLimit {
		allowed, err := s.applyLegacyInstanceRateLimit(ctx, instance)
		if err != nil {
			releaseSlotIfNeeded()
			return nil, err
		}
		if !allowed {
			releaseSlotIfNeeded()
			return nil, fmt.Errorf("实例限流已达到")
		}
		if operation != "" {
			allowed, err = s.applyInstanceRateLimitRules(ctx, instance, operation)
			if err != nil {
				allowed = true
			}
			if !allowed {
				releaseSlotIfNeeded()
				return nil, fmt.Errorf("实例限流已达到")
			}
		}
	}

	return instance, nil
}

// ReleaseInstanceSlot 释放实例槽位
func (s *instanceScheduler) ReleaseInstanceSlot(ctx context.Context, instanceID uint) error {
	return s.stateStore.ReleaseInstanceSlot(ctx, instanceID)
}

func (s *instanceScheduler) filterCircuitOpenInstances(ctx context.Context, instances []*model.ProviderInstance) []*model.ProviderInstance {
	if s == nil || s.stateStore == nil || s.config == nil || !s.config.EnableCircuitBreaker {
		return instances
	}
	if len(instances) == 0 {
		return instances
	}

	filtered := make([]*model.ProviderInstance, 0, len(instances))
	for _, inst := range instances {
		if inst == nil {
			continue
		}
		state, err := s.stateStore.GetInstanceCircuitState(ctx, inst.ID)
		if err != nil {
			filtered = append(filtered, inst)
			continue
		}
		if state == model.CircuitStateOpen {
			continue
		}
		filtered = append(filtered, inst)
	}
	return filtered
}

func (s *instanceScheduler) applyLegacyInstanceRateLimit(ctx context.Context, instance *model.ProviderInstance) (bool, error) {
	if s == nil || s.stateStore == nil || instance == nil {
		return true, nil
	}
	if instance.RPMLimit > 0 {
		allowed, err := s.stateStore.CheckInstanceRateLimit(ctx, instance.ID, "rpm", int64(instance.RPMLimit))
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}
	if instance.TPMLimit > 0 {
		allowed, err := s.stateStore.CheckInstanceRateLimit(ctx, instance.ID, "tpm", int64(instance.TPMLimit))
		if err != nil {
			return false, err
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}

func (s *instanceScheduler) applyInstanceRateLimitRules(ctx context.Context, instance *model.ProviderInstance, operation string) (bool, error) {
	if s == nil || s.stateStore == nil || s.rateLimitRuleRepo == nil || instance == nil {
		return true, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	operation = model.NormalizeOperation(operation)
	if operation == "" {
		return true, nil
	}

	rules, err := s.rateLimitRuleRepo.ListEnabledByScope(ctx, model.RateLimitScopeInstance, instance.ProviderID, instance.ID, operation)
	if err != nil {
		return true, err
	}
	for _, rule := range rules {
		if rule == nil || !rule.Enabled {
			continue
		}
		if rule.Limit <= 0 || rule.WindowSeconds <= 0 {
			continue
		}
		unit := strings.ToLower(strings.TrimSpace(rule.Unit))
		increment := resolveRateLimitIncrement(ctx, unit)
		if increment <= 0 {
			continue
		}

		window := time.Duration(rule.WindowSeconds) * time.Second
		allowed, err := s.stateStore.ConsumeInstanceRateLimit(ctx, instance.ID, unit, rule.Limit, window, increment)
		if err != nil {
			return true, err
		}
		if !allowed {
			return false, nil
		}
	}
	return true, nil
}
