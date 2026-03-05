package scheduler

import (
	"context"
	"sort"
	"sync"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/shopspring/decimal"
)

// SelectionStrategy 选择策略
type SelectionStrategy string

const (
	// StrategyCostFirst 成本优先（默认）
	StrategyCostFirst SelectionStrategy = "cost_first"
	// StrategyAdaptive 自适应（健康/成本/延迟）
	StrategyAdaptive SelectionStrategy = "adaptive"
	// StrategyLatencyFirst 延迟优先
	StrategyLatencyFirst SelectionStrategy = "latency_first"
	// StrategyHealthFirst 健康度优先
	StrategyHealthFirst SelectionStrategy = "health_first"
	// StrategyRoundRobin 轮询
	StrategyRoundRobin SelectionStrategy = "round_robin"
	// StrategyWeighted 加权随机
	StrategyWeighted SelectionStrategy = "weighted"
)

// ProviderSelector 源头选择器接口
type ProviderSelector interface {
	// SelectProviders 选择可用的源头列表（按策略排序）
	SelectProviders(ctx context.Context, operation string, modelID string, strategy SelectionStrategy) ([]*model.ModelProvider, error)
	// SelectBestProvider 选择最佳源头
	SelectBestProvider(ctx context.Context, operation string, modelID string, strategy SelectionStrategy) (*model.ModelProvider, error)
	// GetProviderCost 获取源头的预估成本
	GetProviderCost(provider *model.ModelProvider, inputTokens, outputTokens int64) decimal.Decimal
	// RefreshCache 刷新缓存
	RefreshCache(ctx context.Context, operation string, modelID string) error
}

// providerSelector 源头选择器实现
type providerSelector struct {
	providerRepo    repository.ModelProviderRepository
	pricingRuleRepo repository.ProviderPricingRuleRepository
	cache           map[string]*providerCache
	cacheMu         sync.RWMutex
	cacheTTL        time.Duration
	roundRobinIdx   map[string]int
	rrMu            sync.Mutex
}

// providerCache 源头缓存
type providerCache struct {
	providers []*model.ModelProvider
	expireAt  time.Time
}

// NewProviderSelector 创建源头选择器
func NewProviderSelector(providerRepo repository.ModelProviderRepository, pricingRuleRepo repository.ProviderPricingRuleRepository) ProviderSelector {
	return &providerSelector{
		providerRepo:    providerRepo,
		pricingRuleRepo: pricingRuleRepo,
		cache:           make(map[string]*providerCache),
		cacheTTL:        30 * time.Second, // 缓存30秒
		roundRobinIdx:   make(map[string]int),
	}
}

// SelectProviders 选择可用的源头列表
func (s *providerSelector) cacheKey(operation string, modelID string) string {
	return model.NormalizeOperation(operation) + ":" + modelID
}

func (s *providerSelector) SelectProviders(ctx context.Context, operation string, modelID string, strategy SelectionStrategy) ([]*model.ModelProvider, error) {
	// 从缓存或数据库获取源头
	providers, err := s.getAvailableProviders(ctx, operation, modelID)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, nil
	}

	// 复制切片以避免修改缓存
	result := make([]*model.ModelProvider, len(providers))
	copy(result, providers)

	// 根据策略排序
	switch strategy {
	case StrategyCostFirst:
		s.sortByCost(ctx, operation, result)
	case StrategyAdaptive:
		s.sortByAdaptive(ctx, operation, result)
		s.applyRoundRobin(s.cacheKey(operation, modelID), result)
	case StrategyLatencyFirst:
		s.sortByLatency(result)
	case StrategyHealthFirst:
		s.sortByHealth(result)
	case StrategyRoundRobin:
		s.applyRoundRobin(s.cacheKey(operation, modelID), result)
	case StrategyWeighted:
		s.applyWeighted(result)
	default:
		s.sortByCost(ctx, operation, result) // 默认成本优先
	}

	return result, nil
}

// SelectBestProvider 选择最佳源头
func (s *providerSelector) SelectBestProvider(ctx context.Context, operation string, modelID string, strategy SelectionStrategy) (*model.ModelProvider, error) {
	providers, err := s.SelectProviders(ctx, operation, modelID, strategy)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, nil
	}

	return providers[0], nil
}

// GetProviderCost 获取源头的预估成本
func (s *providerSelector) GetProviderCost(provider *model.ModelProvider, inputTokens, outputTokens int64) decimal.Decimal {
	if provider == nil {
		return decimal.Zero
	}

	// 计算输入成本
	inputCost := provider.ActualCostPer1kInput.
		Mul(decimal.NewFromInt(inputTokens)).
		Div(decimal.NewFromInt(1000))

	// 计算输出成本
	outputCost := provider.ActualCostPer1kOutput.
		Mul(decimal.NewFromInt(outputTokens)).
		Div(decimal.NewFromInt(1000))

	return inputCost.Add(outputCost)
}

// RefreshCache 刷新缓存
func (s *providerSelector) RefreshCache(ctx context.Context, operation string, modelID string) error {
	operation = model.NormalizeOperation(operation)
	providers, err := s.providerRepo.GetAvailableProviders(ctx, operation, modelID)
	if err != nil {
		return err
	}

	cacheKey := s.cacheKey(operation, modelID)
	s.cacheMu.Lock()
	s.cache[cacheKey] = &providerCache{
		providers: providers,
		expireAt:  time.Now().Add(s.cacheTTL),
	}
	s.cacheMu.Unlock()

	return nil
}

// getAvailableProviders 获取可用源头（带缓存）
func (s *providerSelector) getAvailableProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	// 检查缓存
	operation = model.NormalizeOperation(operation)
	cacheKey := s.cacheKey(operation, modelID)

	s.cacheMu.RLock()
	cached, ok := s.cache[cacheKey]
	s.cacheMu.RUnlock()

	if ok && time.Now().Before(cached.expireAt) {
		return cached.providers, nil
	}

	// 从数据库获取
	providers, err := s.providerRepo.GetAvailableProviders(ctx, operation, modelID)
	if err != nil {
		return nil, err
	}

	// 更新缓存
	s.cacheMu.Lock()
	s.cache[cacheKey] = &providerCache{
		providers: providers,
		expireAt:  time.Now().Add(s.cacheTTL),
	}
	s.cacheMu.Unlock()

	return providers, nil
}

// sortByCost 按成本排序（升序）
// ⭐ 核心排序逻辑：成本优先
//
// 排序优先级：
// 1. 成本最低的优先
// 2. 成本相同时，权重高的优先（负载均衡）
// 3. 权重相同时，健康分高的优先
// 4. 健康分相同时，优先级数字小的优先
func (s *providerSelector) sortByCost(ctx context.Context, operation string, providers []*model.ModelProvider) {
	costMap := make(map[uint]decimal.Decimal, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		costMap[provider.ID] = s.estimateProviderCost(ctx, operation, provider)
	}

	sort.Slice(providers, func(i, j int) bool {
		// 首先按实际成本排序（多模态优先用 provider_pricing_rules 的 CostPerUnit）
		costI := costMap[providers[i].ID]
		costJ := costMap[providers[j].ID]

		cmp := costI.Cmp(costJ)
		if cmp != 0 {
			return cmp < 0 // 成本低的优先
		}

		// ⭐ 修复：成本相同时，按权重排序（权重高的优先）
		// 这样可以在成本相同的 provider 之间实现负载均衡
		if providers[i].Weight != providers[j].Weight {
			return providers[i].Weight > providers[j].Weight // 权重高的优先
		}

		// 权重相同时，按健康分数排序（decimal.Decimal 类型）
		if providers[i].HealthScore.Cmp(providers[j].HealthScore) != 0 {
			return providers[i].HealthScore.GreaterThan(providers[j].HealthScore) // 健康分高的优先
		}

		// 健康分相同时，按优先级排序
		return providers[i].Priority < providers[j].Priority // 优先级数字小的优先
	})
}

func (s *providerSelector) estimateProviderCost(ctx context.Context, operation string, provider *model.ModelProvider) decimal.Decimal {
	if provider == nil {
		return decimal.Zero
	}

	operation = model.NormalizeOperation(operation)
	if ctx == nil {
		ctx = context.Background()
	}

	switch operation {
	case model.OperationChatCompletions, model.OperationCompletions, model.OperationEmbeddings:
		// token-based：沿用原始成本字段
		return provider.ActualCostPer1kInput.Add(provider.ActualCostPer1kOutput)
	default:
		// 多模态：优先用可配置的 pricing rule 作为“每次请求”的基准成本
		for _, unit := range preferredPricingUnits(operation) {
			if unit == "" {
				continue
			}
			if cost, ok := s.resolvePricingRuleCost(ctx, provider.ID, operation, unit); ok {
				return cost
			}
		}

		// 兜底：仍返回 token 成本字段（允许业务方沿用 old 逻辑）
		return provider.ActualCostPer1kInput.Add(provider.ActualCostPer1kOutput)
	}
}

func preferredPricingUnits(operation string) []string {
	switch model.NormalizeOperation(operation) {
	case model.OperationImagesGenerations:
		return []string{"image", "request"}
	case model.OperationVideosGenerations:
		return []string{"video_second", "request"}
	case model.OperationAudioTranscriptions, model.OperationAudioTranslations, model.OperationAudioSpeech:
		return []string{"request"}
	default:
		return []string{"request"}
	}
}

func (s *providerSelector) resolvePricingRuleCost(ctx context.Context, providerID uint, operation string, unit string) (decimal.Decimal, bool) {
	if s == nil || s.pricingRuleRepo == nil || providerID == 0 {
		return decimal.Zero, false
	}
	if ctx == nil {
		ctx = context.Background()
	}

	rule, err := s.pricingRuleRepo.GetByProviderOperationUnit(ctx, providerID, operation, unit)
	if err != nil || rule == nil || !rule.Enabled {
		return decimal.Zero, false
	}
	if rule.CostPerUnit.LessThan(decimal.Zero) {
		return decimal.Zero, false
	}
	return rule.CostPerUnit, true
}

// sortByLatency 按延迟排序（升序）
func (s *providerSelector) sortByLatency(providers []*model.ModelProvider) {
	sort.Slice(providers, func(i, j int) bool {
		// 使用 AvgLatencyMs 字段或计算平均延迟
		avgLatencyI := int64(providers[i].AvgLatencyMs)
		avgLatencyJ := int64(providers[j].AvgLatencyMs)

		// 如果 AvgLatencyMs 为 0，尝试从 TotalLatency 计算
		if avgLatencyI == 0 && providers[i].TotalRequests > 0 {
			avgLatencyI = providers[i].TotalLatency / providers[i].TotalRequests
		}
		if avgLatencyJ == 0 && providers[j].TotalRequests > 0 {
			avgLatencyJ = providers[j].TotalLatency / providers[j].TotalRequests
		}

		if avgLatencyI != avgLatencyJ {
			return avgLatencyI < avgLatencyJ // 延迟低的优先
		}

		// 延迟相同时，按成本排序
		costI := providers[i].ActualCostPer1kInput.Add(providers[i].ActualCostPer1kOutput)
		costJ := providers[j].ActualCostPer1kInput.Add(providers[j].ActualCostPer1kOutput)
		return costI.Cmp(costJ) < 0
	})
}

// sortByAdaptive 按综合评分排序（健康/成本/延迟/权重）
func (s *providerSelector) sortByAdaptive(ctx context.Context, operation string, providers []*model.ModelProvider) {
	if len(providers) <= 1 {
		return
	}

	type adaptiveScore struct {
		score       float64
		cost        float64
		latencyMs   int64
		healthScore float64
		weightScore float64
	}

	scoreMap := make(map[uint]adaptiveScore, len(providers))

	minCost := 0.0
	maxCost := 0.0
	minLatency := int64(0)
	maxLatency := int64(0)
	maxWeight := 1

	for i, provider := range providers {
		if provider == nil {
			continue
		}

		cost := s.estimateProviderCost(ctx, operation, provider).InexactFloat64()
		if cost < 0 {
			cost = 0
		}

		latencyMs := int64(provider.AvgLatencyMs)
		if latencyMs <= 0 && provider.TotalRequests > 0 {
			latencyMs = provider.TotalLatency / provider.TotalRequests
		}
		if latencyMs < 0 {
			latencyMs = 0
		}

		weight := provider.Weight
		if weight <= 0 {
			weight = 1
		}

		if i == 0 {
			minCost, maxCost = cost, cost
			minLatency, maxLatency = latencyMs, latencyMs
			maxWeight = weight
		} else {
			if cost < minCost {
				minCost = cost
			}
			if cost > maxCost {
				maxCost = cost
			}
			if latencyMs < minLatency {
				minLatency = latencyMs
			}
			if latencyMs > maxLatency {
				maxLatency = latencyMs
			}
			if weight > maxWeight {
				maxWeight = weight
			}
		}

		scoreMap[provider.ID] = adaptiveScore{
			cost:      cost,
			latencyMs: latencyMs,
		}
	}

	costRange := maxCost - minCost
	latencyRange := float64(maxLatency - minLatency)
	maxWeightFloat := float64(maxWeight)
	if maxWeightFloat <= 0 {
		maxWeightFloat = 1
	}

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		raw := scoreMap[provider.ID]
		costNorm := 1.0
		if costRange > 0 {
			costNorm = 1.0 - (raw.cost-minCost)/costRange
		}

		latencyNorm := 1.0
		if latencyRange > 0 {
			latencyNorm = 1.0 - float64(raw.latencyMs-minLatency)/latencyRange
		}

		healthNorm := provider.HealthScore.InexactFloat64() / 100.0
		if healthNorm < 0 {
			healthNorm = 0
		}
		if healthNorm > 1 {
			healthNorm = 1
		}

		weight := provider.Weight
		if weight <= 0 {
			weight = 1
		}
		weightNorm := float64(weight) / maxWeightFloat

		// 权重分配：
		// 成本 35%，延迟 25%，健康 30%，静态权重 10%。
		total := costNorm*0.35 + latencyNorm*0.25 + healthNorm*0.30 + weightNorm*0.10
		raw.score = total
		raw.healthScore = healthNorm
		raw.weightScore = weightNorm
		scoreMap[provider.ID] = raw
	}

	sort.Slice(providers, func(i, j int) bool {
		left := scoreMap[providers[i].ID]
		right := scoreMap[providers[j].ID]

		if left.score != right.score {
			return left.score > right.score
		}
		if left.healthScore != right.healthScore {
			return left.healthScore > right.healthScore
		}
		if left.cost != right.cost {
			return left.cost < right.cost
		}
		if providers[i].Weight != providers[j].Weight {
			return providers[i].Weight > providers[j].Weight
		}
		return providers[i].Priority < providers[j].Priority
	})
}

// sortByHealth 按健康度排序（降序）
func (s *providerSelector) sortByHealth(providers []*model.ModelProvider) {
	sort.Slice(providers, func(i, j int) bool {
		if providers[i].HealthScore.Cmp(providers[j].HealthScore) != 0 {
			return providers[i].HealthScore.GreaterThan(providers[j].HealthScore) // 健康分高的优先（decimal.Decimal 类型）
		}

		// 健康分相同时，按成本排序
		costI := providers[i].ActualCostPer1kInput.Add(providers[i].ActualCostPer1kOutput)
		costJ := providers[j].ActualCostPer1kInput.Add(providers[j].ActualCostPer1kOutput)
		return costI.Cmp(costJ) < 0
	})
}

// applyRoundRobin 应用轮询策略
func (s *providerSelector) applyRoundRobin(cacheKey string, providers []*model.ModelProvider) {
	if len(providers) <= 1 {
		return
	}

	s.rrMu.Lock()
	defer s.rrMu.Unlock()

	idx := s.roundRobinIdx[cacheKey]
	idx = idx % len(providers)

	// 将当前索引的元素移到第一位
	if idx > 0 {
		selected := providers[idx]
		copy(providers[1:idx+1], providers[0:idx])
		providers[0] = selected
	}

	// 更新索引
	s.roundRobinIdx[cacheKey] = (idx + 1) % len(providers)
}

// applyWeighted 应用加权随机策略
func (s *providerSelector) applyWeighted(providers []*model.ModelProvider) {
	if len(providers) <= 1 {
		return
	}

	// 计算权重（基于健康分数和成本的综合）
	weights := make([]float64, len(providers))
	totalWeight := 0.0

	for i, p := range providers {
		// 权重 = 健康分数 / (成本 + 1)
		cost := p.ActualCostPer1kInput.Add(p.ActualCostPer1kOutput).InexactFloat64()
		if cost < 0.001 {
			cost = 0.001
		}
		// HealthScore 是 decimal.Decimal 类型
		weights[i] = p.HealthScore.InexactFloat64() / cost
		totalWeight += weights[i]
	}

	// 归一化权重并排序
	type weightedProvider struct {
		provider *model.ModelProvider
		weight   float64
	}

	wp := make([]weightedProvider, len(providers))
	for i, p := range providers {
		wp[i] = weightedProvider{
			provider: p,
			weight:   weights[i] / totalWeight,
		}
	}

	// 按权重降序排序
	sort.Slice(wp, func(i, j int) bool {
		return wp[i].weight > wp[j].weight
	})

	// 更新原切片
	for i, w := range wp {
		providers[i] = w.provider
	}
}

// ProviderScore 源头评分（用于综合评估）
type ProviderScore struct {
	Provider     *model.ModelProvider
	CostScore    float64 // 成本评分（0-100，越低越好）
	LatencyScore float64 // 延迟评分（0-100，越低越好）
	HealthScore  float64 // 健康评分（0-100，越高越好）
	TotalScore   float64 // 综合评分
}

// ScoreProviders 对源头进行评分
func (s *providerSelector) ScoreProviders(providers []*model.ModelProvider) []ProviderScore {
	if len(providers) == 0 {
		return nil
	}

	scores := make([]ProviderScore, len(providers))

	// 找出成本和延迟的最大最小值
	var minCost, maxCost decimal.Decimal
	var minLatency, maxLatency int64

	for i, p := range providers {
		cost := p.ActualCostPer1kInput.Add(p.ActualCostPer1kOutput)
		// 使用 AvgLatencyMs 或从 TotalLatency 计算
		avgLatency := int64(p.AvgLatencyMs)
		if avgLatency == 0 && p.TotalRequests > 0 {
			avgLatency = p.TotalLatency / p.TotalRequests
		}

		if i == 0 {
			minCost, maxCost = cost, cost
			minLatency, maxLatency = avgLatency, avgLatency
		} else {
			if cost.LessThan(minCost) {
				minCost = cost
			}
			if cost.GreaterThan(maxCost) {
				maxCost = cost
			}
			if avgLatency < minLatency {
				minLatency = avgLatency
			}
			if avgLatency > maxLatency {
				maxLatency = avgLatency
			}
		}
	}

	// 计算评分
	costRange := maxCost.Sub(minCost)
	latencyRange := maxLatency - minLatency

	for i, p := range providers {
		cost := p.ActualCostPer1kInput.Add(p.ActualCostPer1kOutput)
		// 使用 AvgLatencyMs 或从 TotalLatency 计算
		avgLatency := int64(p.AvgLatencyMs)
		if avgLatency == 0 && p.TotalRequests > 0 {
			avgLatency = p.TotalLatency / p.TotalRequests
		}

		// 成本评分（归一化到0-100）
		costScore := 0.0
		if !costRange.IsZero() {
			costScore = cost.Sub(minCost).Div(costRange).InexactFloat64() * 100
		}

		// 延迟评分（归一化到0-100）
		latencyScore := 0.0
		if latencyRange > 0 {
			latencyScore = float64(avgLatency-minLatency) / float64(latencyRange) * 100
		}

		// 综合评分（成本40%，延迟30%，健康30%）- HealthScore 是 decimal.Decimal 类型
		totalScore := (100-costScore)*0.4 + (100-latencyScore)*0.3 + p.HealthScore.InexactFloat64()*0.3

		scores[i] = ProviderScore{
			Provider:     p,
			CostScore:    costScore,
			LatencyScore: latencyScore,
			HealthScore:  p.HealthScore.InexactFloat64(),
			TotalScore:   totalScore,
		}
	}

	// 按综合评分排序
	sort.Slice(scores, func(i, j int) bool {
		return scores[i].TotalScore > scores[j].TotalScore
	})

	return scores
}
