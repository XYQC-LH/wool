package scheduler

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

// HealthTracker 健康追踪器接口
type HealthTracker interface {
	// RecordRequest 记录请求结果
	RecordRequest(ctx context.Context, providerID uint, success bool, latencyMs int64, inputTokens, outputTokens int64, cost decimal.Decimal) error
	// UpdateHealthScore 更新健康分数
	UpdateHealthScore(ctx context.Context, providerID uint) error
	// GetHealthScore 获取健康分数
	GetHealthScore(ctx context.Context, providerID uint) (float64, error)
	// GetProviderHealth 获取源头健康详情
	GetProviderHealth(ctx context.Context, providerID uint) (*ProviderHealth, error)
	// GetModelHealth 获取模型健康概览
	GetModelHealth(ctx context.Context, operation string, modelID string) (*ModelHealth, error)
	// GetHealthSummary 获取健康摘要
	GetHealthSummary(ctx context.Context) (*HealthSummary, error)
	// StartBackgroundUpdater 启动后台健康分数更新器
	StartBackgroundUpdater(ctx context.Context, interval time.Duration)
	// StopBackgroundUpdater 停止后台更新器
	StopBackgroundUpdater()
}

// ProviderHealth 源头健康详情
type ProviderHealth struct {
	ProviderID      uint       `json:"provider_id"`
	ProviderName    string     `json:"provider_name,omitempty"`
	HealthScore     float64    `json:"health_score"`
	SuccessRate     float64    `json:"success_rate"`
	AvgLatencyMs    float64    `json:"avg_latency_ms"`
	TotalRequests   int64      `json:"total_requests"`
	SuccessRequests int64      `json:"success_requests"`
	FailedRequests  int64      `json:"failed_requests"`
	CircuitState    string     `json:"circuit_state"`
	LastSuccessAt   *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt   *time.Time `json:"last_failure_at,omitempty"`
	IsHealthy       bool       `json:"is_healthy"`
}

// ModelHealth 模型健康概览
type ModelHealth struct {
	Operation        string           `json:"operation"`
	ModelID          string           `json:"model_id"`
	ModelName        string           `json:"model_name,omitempty"`
	TotalProviders   int              `json:"total_providers"`
	HealthyProviders int              `json:"healthy_providers"`
	AvgHealthScore   float64          `json:"avg_health_score"`
	AvgSuccessRate   float64          `json:"avg_success_rate"`
	AvgLatencyMs     float64          `json:"avg_latency_ms"`
	Providers        []ProviderHealth `json:"providers,omitempty"`
}

// HealthConfig 健康追踪配置
type HealthConfig struct {
	// 健康分数计算权重
	SuccessRateWeight float64 // 成功率权重
	LatencyWeight     float64 // 延迟权重
	RecentWeight      float64 // 近期数据权重

	// 健康阈值
	HealthyThreshold   float64 // 健康阈值（分数高于此值视为健康）
	UnhealthyThreshold float64 // 不健康阈值（分数低于此值视为不健康）

	// 延迟基准
	TargetLatencyMs float64 // 目标延迟（毫秒）
	MaxLatencyMs    float64 // 最大可接受延迟（毫秒）

	// 时间窗口
	RecentWindowMinutes int // 近期数据窗口（分钟）
}

// DefaultHealthConfig 默认健康配置
func DefaultHealthConfig() *HealthConfig {
	return &HealthConfig{
		SuccessRateWeight:   0.5,
		LatencyWeight:       0.3,
		RecentWeight:        0.2,
		HealthyThreshold:    80.0,
		UnhealthyThreshold:  50.0,
		TargetLatencyMs:     1000,
		MaxLatencyMs:        10000,
		RecentWindowMinutes: 30,
	}
}

// healthTracker 健康追踪器实现
type healthTracker struct {
	providerRepo repository.ModelProviderRepository
	metricsRepo  repository.ProviderMetricsRepository
	config       *HealthConfig

	// 后台更新器
	stopCh   chan struct{}
	stopOnce sync.Once
	running  bool
	mu       sync.RWMutex
}

// NewHealthTracker 创建健康追踪器
func NewHealthTracker(
	providerRepo repository.ModelProviderRepository,
	metricsRepo repository.ProviderMetricsRepository,
	config *HealthConfig,
) HealthTracker {
	if config == nil {
		config = DefaultHealthConfig()
	}
	return &healthTracker{
		providerRepo: providerRepo,
		metricsRepo:  metricsRepo,
		config:       config,
		stopCh:       make(chan struct{}),
	}
}

// RecordRequest 记录请求结果
func (h *healthTracker) RecordRequest(ctx context.Context, providerID uint, success bool, latencyMs int64, inputTokens, outputTokens int64, cost decimal.Decimal) error {
	// 更新源头统计
	if err := h.providerRepo.IncrementStats(ctx, providerID, success, latencyMs, inputTokens, outputTokens, cost); err != nil {
		return err
	}

	// 记录近期表现（Redis 滚动窗口）
	h.recordRecentWindow(ctx, providerID, success)

	// 异步更新健康分数
	go func() {
		_ = h.UpdateHealthScore(context.Background(), providerID)
	}()

	return nil
}

// UpdateHealthScore 更新健康分数
// ⭐ 核心方法：计算综合健康分数
func (h *healthTracker) UpdateHealthScore(ctx context.Context, providerID uint) error {
	provider, err := h.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return err
	}
	if provider == nil {
		return nil
	}

	// 计算健康分数
	score := h.calculateHealthScore(provider)

	// 更新数据库
	return h.providerRepo.UpdateHealthScore(ctx, providerID, score)
}

// calculateHealthScore 计算健康分数
func (h *healthTracker) calculateHealthScore(provider *model.ModelProvider) float64 {
	// 1. 计算成功率分数 (0-100)
	successRateScore := h.calculateSuccessRateScore(provider)

	// 2. 计算延迟分数 (0-100)
	latencyScore := h.calculateLatencyScore(provider)

	// 3. 计算近期表现分数 (0-100)
	recentScore := h.calculateRecentScore(provider)

	// 4. 综合计算
	totalScore := successRateScore*h.config.SuccessRateWeight +
		latencyScore*h.config.LatencyWeight +
		recentScore*h.config.RecentWeight

	// 5. 熔断状态惩罚
	if provider.CircuitState == model.CircuitStateOpen {
		totalScore *= 0.5 // 熔断状态分数减半
	} else if provider.CircuitState == model.CircuitStateHalfOpen {
		totalScore *= 0.8 // 半开状态分数减少20%
	}

	// 确保分数在 0-100 范围内
	if totalScore < 0 {
		totalScore = 0
	}
	if totalScore > 100 {
		totalScore = 100
	}

	return totalScore
}

// calculateSuccessRateScore 计算成功率分数
func (h *healthTracker) calculateSuccessRateScore(provider *model.ModelProvider) float64 {
	if provider.TotalRequests == 0 {
		return 100.0 // 没有请求时默认满分
	}

	successRate := float64(provider.SuccessRequests) / float64(provider.TotalRequests) * 100
	return successRate
}

// calculateLatencyScore 计算延迟分数
func (h *healthTracker) calculateLatencyScore(provider *model.ModelProvider) float64 {
	if provider.TotalRequests == 0 {
		return 100.0 // 没有请求时默认满分
	}

	avgLatency := float64(provider.TotalLatency) / float64(provider.TotalRequests)

	// 使用对数函数平滑延迟分数
	if avgLatency <= h.config.TargetLatencyMs {
		return 100.0
	}

	if avgLatency >= h.config.MaxLatencyMs {
		return 0.0
	}

	// 线性插值
	ratio := (avgLatency - h.config.TargetLatencyMs) / (h.config.MaxLatencyMs - h.config.TargetLatencyMs)
	return 100.0 * (1 - ratio)
}

// calculateRecentScore 计算近期表现分数
//
// ⚠️ 当前实现限制：
// - 仅使用数据库中的 LastSuccessAt 和 LastFailureAt 字段
// - 没有使用 Redis 滚动窗口指标
// - 无法准确反映近期的请求成功率波动
//
// 📋 未来改进方向：
// 1. 使用 Redis 滚动窗口存储近期请求记录
//    - 使用 Redis Sorted Set 存储请求时间戳
//    - Key: health:window:{provider_id}
//    - Score: 请求时间戳（毫秒）
//    - Member: 请求结果（success/failure）
// 2. 计算滚动窗口内的成功率
//    - 使用 ZREMRANGEBYSCORE 清理过期数据
//    - 使用 ZCOUNT 统计成功和失败请求数
//    - 计算近期成功率 = 成功请求数 / 总请求数
// 3. 实现滑动窗口算法
//    - 窗口大小：30分钟（可配置）
//    - 自动清理过期数据
//    - 支持多时间粒度（5分钟、15分钟、30分钟）
func (h *healthTracker) calculateRecentScore(provider *model.ModelProvider) float64 {
	client := cache.GetClient()
	if client == nil {
		return h.calculateRecentScoreFallback(provider)
	}

	nowMs := time.Now().UnixMilli()
	keyAll, keySuccess := recentWindowKeys(provider.ID)

	type windowSpec struct {
		minutes int64
		weight  float64
	}

	recentMinutes := int64(h.config.RecentWindowMinutes)
	if recentMinutes <= 0 {
		recentMinutes = 30
	}

	windows := []windowSpec{
		{minutes: 5, weight: 0.5},
		{minutes: 15, weight: 0.3},
		{minutes: recentMinutes, weight: 0.2},
	}

	pipe := client.Pipeline()
	type windowCmds struct {
		total   *redis.IntCmd
		success *redis.IntCmd
		weight  float64
	}
	cmds := make([]windowCmds, 0, len(windows))

	for _, w := range windows {
		if w.minutes <= 0 || w.weight <= 0 {
			continue
		}
		startMs := nowMs - (time.Duration(w.minutes) * time.Minute).Milliseconds()
		min := fmt.Sprintf("%d", startMs)
		max := fmt.Sprintf("%d", nowMs)
		cmds = append(cmds, windowCmds{
			total:   pipe.ZCount(context.Background(), keyAll, min, max),
			success: pipe.ZCount(context.Background(), keySuccess, min, max),
			weight:  w.weight,
		})
	}

	if _, err := pipe.Exec(context.Background()); err != nil {
		return h.calculateRecentScoreFallback(provider)
	}

	var weighted float64
	var usedWeight float64
	for _, c := range cmds {
		total := c.total.Val()
		if total <= 0 {
			continue
		}
		success := c.success.Val()
		rate := float64(success) / float64(total) * 100
		weighted += rate * c.weight
		usedWeight += c.weight
	}

	if usedWeight <= 0 {
		return h.calculateRecentScoreFallback(provider)
	}

	score := weighted / usedWeight
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}
	return score
}

func (h *healthTracker) calculateRecentScoreFallback(provider *model.ModelProvider) float64 {
	now := time.Now()

	if provider.LastSuccessAt != nil {
		timeSinceSuccess := now.Sub(*provider.LastSuccessAt)
		if timeSinceSuccess < time.Duration(h.config.RecentWindowMinutes)*time.Minute {
			return 100.0
		}
	}

	if provider.LastFailureAt != nil {
		timeSinceFailure := now.Sub(*provider.LastFailureAt)
		if timeSinceFailure < 5*time.Minute {
			return 30.0
		}
		if timeSinceFailure < 15*time.Minute {
			return 60.0
		}
	}

	return 80.0
}

func recentWindowKeys(providerID uint) (allKey string, successKey string) {
	prefix := fmt.Sprintf("health:window:%d", providerID)
	return prefix + ":all", prefix + ":success"
}

func (h *healthTracker) recordRecentWindow(ctx context.Context, providerID uint, success bool) {
	client := cache.GetClient()
	if client == nil {
		return
	}

	now := time.Now()
	nowMs := now.UnixMilli()

	recentMinutes := h.config.RecentWindowMinutes
	if recentMinutes <= 0 {
		recentMinutes = 30
	}

	// 支持多窗口（5m/15m/RecentWindowMinutes），保留最大窗口范围的数据
	maxMinutes := recentMinutes
	if maxMinutes < 15 {
		maxMinutes = 15
	}

	window := time.Duration(maxMinutes) * time.Minute
	windowStart := nowMs - window.Milliseconds()

	member := fmt.Sprintf("%d", now.UnixNano())
	score := float64(nowMs)

	allKey, successKey := recentWindowKeys(providerID)

	pipe := client.Pipeline()
	pipe.ZRemRangeByScore(ctx, allKey, "0", fmt.Sprintf("%d", windowStart))
	pipe.ZAdd(ctx, allKey, redis.Z{Score: score, Member: member})
	pipe.Expire(ctx, allKey, window+time.Minute)

	pipe.ZRemRangeByScore(ctx, successKey, "0", fmt.Sprintf("%d", windowStart))
	if success {
		pipe.ZAdd(ctx, successKey, redis.Z{Score: score, Member: member})
	}
	pipe.Expire(ctx, successKey, window+time.Minute)

	_, _ = pipe.Exec(ctx)
}

// GetHealthScore 获取健康分数
func (h *healthTracker) GetHealthScore(ctx context.Context, providerID uint) (float64, error) {
	provider, err := h.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return 0, err
	}
	if provider == nil {
		return 0, nil
	}
	return provider.HealthScore.InexactFloat64(), nil
}

// GetProviderHealth 获取源头健康详情
func (h *healthTracker) GetProviderHealth(ctx context.Context, providerID uint) (*ProviderHealth, error) {
	provider, err := h.providerRepo.GetByID(ctx, providerID)
	if err != nil {
		return nil, err
	}
	if provider == nil {
		return nil, nil
	}

	// 计算成功率
	successRate := float64(0)
	if provider.TotalRequests > 0 {
		successRate = float64(provider.SuccessRequests) / float64(provider.TotalRequests) * 100
	}

	// 计算平均延迟
	avgLatency := float64(0)
	if provider.TotalRequests > 0 {
		avgLatency = float64(provider.TotalLatency) / float64(provider.TotalRequests)
	}

	// 获取源头名称
	providerName := ""
	if provider.Channel != nil {
		providerName = provider.Channel.Name
	}

	return &ProviderHealth{
		ProviderID:      providerID,
		ProviderName:    providerName,
		HealthScore:     provider.HealthScore.InexactFloat64(),
		SuccessRate:     successRate,
		AvgLatencyMs:    avgLatency,
		TotalRequests:   provider.TotalRequests,
		SuccessRequests: provider.SuccessRequests,
		FailedRequests:  provider.FailedRequests,
		CircuitState:    string(provider.CircuitState),
		LastSuccessAt:   provider.LastSuccessAt,
		LastFailureAt:   provider.LastFailureAt,
		IsHealthy:       provider.HealthScore.GreaterThanOrEqual(decimal.NewFromFloat(h.config.HealthyThreshold)),
	}, nil
}

// GetModelHealth 获取模型健康概览
func (h *healthTracker) GetModelHealth(ctx context.Context, operation string, modelID string) (*ModelHealth, error) {
	operation = model.NormalizeOperation(operation)
	providers, err := h.providerRepo.GetByModelID(ctx, operation, modelID)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return &ModelHealth{
			Operation:        operation,
			ModelID:          modelID,
			TotalProviders:   0,
			HealthyProviders: 0,
		}, nil
	}

	var totalHealthScore, totalSuccessRate, totalLatency float64
	var healthyCount int
	providerHealths := make([]ProviderHealth, 0, len(providers))

	for _, p := range providers {
		// 计算成功率
		successRate := float64(0)
		if p.TotalRequests > 0 {
			successRate = float64(p.SuccessRequests) / float64(p.TotalRequests) * 100
		}

		// 计算平均延迟
		avgLatency := float64(0)
		if p.TotalRequests > 0 {
			avgLatency = float64(p.TotalLatency) / float64(p.TotalRequests)
		}

		// 获取源头名称
		providerName := ""
		if p.Channel != nil {
			providerName = p.Channel.Name
		}

		isHealthy := p.HealthScore.GreaterThanOrEqual(decimal.NewFromFloat(h.config.HealthyThreshold))
		if isHealthy {
			healthyCount++
		}

		totalHealthScore += p.HealthScore.InexactFloat64()
		totalSuccessRate += successRate
		totalLatency += avgLatency

		providerHealths = append(providerHealths, ProviderHealth{
			ProviderID:      p.ID,
			ProviderName:    providerName,
			HealthScore:     p.HealthScore.InexactFloat64(),
			SuccessRate:     successRate,
			AvgLatencyMs:    avgLatency,
			TotalRequests:   p.TotalRequests,
			SuccessRequests: p.SuccessRequests,
			FailedRequests:  p.FailedRequests,
			CircuitState:    string(p.CircuitState),
			LastSuccessAt:   p.LastSuccessAt,
			LastFailureAt:   p.LastFailureAt,
			IsHealthy:       isHealthy,
		})
	}

	count := float64(len(providers))

	return &ModelHealth{
		Operation:        operation,
		ModelID:          modelID,
		TotalProviders:   len(providers),
		HealthyProviders: healthyCount,
		AvgHealthScore:   math.Round(totalHealthScore/count*100) / 100,
		AvgSuccessRate:   math.Round(totalSuccessRate/count*100) / 100,
		AvgLatencyMs:     math.Round(totalLatency/count*100) / 100,
		Providers:        providerHealths,
	}, nil
}

// StartBackgroundUpdater 启动后台健康分数更新器
func (h *healthTracker) StartBackgroundUpdater(ctx context.Context, interval time.Duration) {
	h.mu.Lock()
	if h.running {
		h.mu.Unlock()
		return
	}
	h.running = true
	h.stopCh = make(chan struct{})
	h.mu.Unlock()

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.updateAllHealthScores(ctx)
			case <-h.stopCh:
				return
			case <-ctx.Done():
				return
			}
		}
	}()
}

// StopBackgroundUpdater 停止后台更新器
func (h *healthTracker) StopBackgroundUpdater() {
	h.stopOnce.Do(func() {
		h.mu.Lock()
		defer h.mu.Unlock()
		if h.running {
			close(h.stopCh)
			h.running = false
		}
	})
}

// updateAllHealthScores 更新所有源头的健康分数
func (h *healthTracker) updateAllHealthScores(ctx context.Context) {
	// 获取所有活跃的源头
	params := &model.ProviderQueryParams{
		Status:   string(model.ProviderStatusActive),
		Page:     1,
		PageSize: 1000, // 批量处理
	}

	providers, _, err := h.providerRepo.List(ctx, params)
	if err != nil {
		return
	}

	for _, p := range providers {
		_ = h.UpdateHealthScore(ctx, p.ID)
	}
}

// HealthSummary 健康摘要
type HealthSummary struct {
	TotalProviders     int     `json:"total_providers"`
	HealthyProviders   int     `json:"healthy_providers"`
	UnhealthyProviders int     `json:"unhealthy_providers"`
	CircuitOpenCount   int     `json:"circuit_open_count"`
	AvgHealthScore     float64 `json:"avg_health_score"`
	AvgSuccessRate     float64 `json:"avg_success_rate"`
}

// GetHealthSummary 获取健康摘要
func (h *healthTracker) GetHealthSummary(ctx context.Context) (*HealthSummary, error) {
	params := &model.ProviderQueryParams{
		Status:   string(model.ProviderStatusActive),
		Page:     1,
		PageSize: 1000,
	}

	providers, _, err := h.providerRepo.List(ctx, params)
	if err != nil {
		return nil, err
	}

	summary := &HealthSummary{
		TotalProviders: len(providers),
	}

	if len(providers) == 0 {
		return summary, nil
	}

	var totalHealthScore, totalSuccessRate float64

	for _, p := range providers {
		totalHealthScore += p.HealthScore.InexactFloat64()

		if p.TotalRequests > 0 {
			totalSuccessRate += float64(p.SuccessRequests) / float64(p.TotalRequests) * 100
		} else {
			totalSuccessRate += 100
		}

		if p.HealthScore.GreaterThanOrEqual(decimal.NewFromFloat(h.config.HealthyThreshold)) {
			summary.HealthyProviders++
		} else if p.HealthScore.LessThan(decimal.NewFromFloat(h.config.UnhealthyThreshold)) {
			summary.UnhealthyProviders++
		}

		if p.CircuitState == model.CircuitStateOpen {
			summary.CircuitOpenCount++
		}
	}

	count := float64(len(providers))
	summary.AvgHealthScore = math.Round(totalHealthScore/count*100) / 100
	summary.AvgSuccessRate = math.Round(totalSuccessRate/count*100) / 100

	return summary, nil
}
