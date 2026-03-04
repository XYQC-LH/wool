package scheduler

import (
	"context"
	"fmt"
	"math"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/shopspring/decimal"
)

// CostCalculator 成本计算器接口
// 统一管理成本计算逻辑
type CostCalculator interface {
	// CalculateCost 计算请求成本
	CalculateCost(modelID string, promptTokens, completionTokens int) (decimal.Decimal, error)
	// CalculateProviderCost 计算Provider成本
	CalculateProviderCost(provider *model.ModelProvider, promptTokens, completionTokens int) decimal.Decimal
	// EstimateCost 估算请求成本
	EstimateCost(req *ChatCompletionRequest) (decimal.Decimal, error)
	// EstimatePromptTokens 估算Prompt tokens
	EstimatePromptTokens(req *ChatCompletionRequest) int
	// EstimateCompletionTokens 估算Completion tokens
	EstimateCompletionTokens(req *ChatCompletionRequest) int
	// GetModelPricing 获取模型定价
	GetModelPricing(modelID string) (*model.ModelPricing, error)
	// CalculateProfit 计算利润
	CalculateProfit(upstreamCost, downstreamCost decimal.Decimal) decimal.Decimal
	// CalculateMargin 计算利润率
	CalculateMargin(upstreamCost, downstreamCost decimal.Decimal) float64
	// GetCostBreakdown 获取成本明细
	GetCostBreakdown(modelID string, promptTokens, completionTokens int) (*CostBreakdown, error)
	// AnalyzeCosts 分析成本
	AnalyzeCosts(modelID string, startTime, endTime time.Time) (*CostAnalysis, error)
	// GetCostOptimizationSuggestions 获取成本优化建议
	GetCostOptimizationSuggestions(modelID string) ([]*CostOptimizationSuggestion, error)
}

// CostBreakdown 成本明细
type CostBreakdown struct {
	ModelID          string          `json:"model_id"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	TotalTokens      int             `json:"total_tokens"`
	UpstreamCost     decimal.Decimal `json:"upstream_cost"`
	DownstreamCost   decimal.Decimal `json:"downstream_cost"`
	Profit           decimal.Decimal `json:"profit"`
	Margin           float64         `json:"margin"`
	CostPer1KTokens  decimal.Decimal `json:"cost_per_1k_tokens"`
}

// CostCalculatorConfig 成本计算器配置
type CostCalculatorConfig struct {
	// 默认token估算比例
	DefaultTokenRatio float64
	// 默认completion tokens估算
	DefaultCompletionTokens int
	// 是否启用缓存
	EnableCache bool
	// 缓存过期时间
	CacheTTL int
}

// DefaultCostCalculatorConfig 默认成本计算器配置
func DefaultCostCalculatorConfig() *CostCalculatorConfig {
	return &CostCalculatorConfig{
		DefaultTokenRatio:       0.25, // 每4个字符约1个token
		DefaultCompletionTokens: 500,
		EnableCache:             true,
		CacheTTL:                3600, // 1小时
	}
}

// costCalculator 成本计算器实现
type costCalculator struct {
	modelRepo    repository.ModelRepository
	providerRepo repository.ModelProviderRepository
	metricsRepo  repository.ProviderMetricsRepository
	config       *CostCalculatorConfig
}

// NewCostCalculator 创建成本计算器
func NewCostCalculator(
	modelRepo repository.ModelRepository,
	providerRepo repository.ModelProviderRepository,
	metricsRepo repository.ProviderMetricsRepository,
	config *CostCalculatorConfig,
) CostCalculator {
	if config == nil {
		config = DefaultCostCalculatorConfig()
	}
	return &costCalculator{
		modelRepo:    modelRepo,
		providerRepo: providerRepo,
		metricsRepo:  metricsRepo,
		config:       config,
	}
}

// CalculateCost 计算请求成本
// ⭐ 核心方法：根据模型定价计算成本
func (cc *costCalculator) CalculateCost(modelID string, promptTokens, completionTokens int) (decimal.Decimal, error) {
	// 获取模型定价
	pricing, err := cc.GetModelPricing(modelID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("获取模型定价失败: %w", err)
	}

	// 计算输入成本
	inputCost := cc.calculateInputCost(pricing, promptTokens)

	// 计算输出成本
	outputCost := cc.calculateOutputCost(pricing, completionTokens)

	// 总成本
	totalCost := inputCost.Add(outputCost)

	return totalCost, nil
}

// CalculateProviderCost 计算Provider成本
// ⭐ 核心方法：根据Provider配置计算上游成本
func (cc *costCalculator) CalculateProviderCost(provider *model.ModelProvider, promptTokens, completionTokens int) decimal.Decimal {
	if provider == nil {
		return decimal.Zero
	}

	// 计算输入成本
	inputCost := provider.ActualCostPer1kInput.
		Mul(decimal.NewFromInt(int64(promptTokens))).
		Div(decimal.NewFromInt(1000))

	// 计算输出成本
	outputCost := provider.ActualCostPer1kOutput.
		Mul(decimal.NewFromInt(int64(completionTokens))).
		Div(decimal.NewFromInt(1000))

	// 总成本
	totalCost := inputCost.Add(outputCost)

	return totalCost
}

// EstimateCost 估算请求成本
// ⭐ 核心方法：估算请求成本（用于预检查）
func (cc *costCalculator) EstimateCost(req *ChatCompletionRequest) (decimal.Decimal, error) {
	// 估算prompt tokens
	promptTokens := cc.EstimatePromptTokens(req)

	// 估算completion tokens
	completionTokens := cc.EstimateCompletionTokens(req)

	// 计算成本
	return cc.CalculateCost(req.Model, promptTokens, completionTokens)
}

// EstimatePromptTokens 估算Prompt tokens
// ⭐ 核心方法：根据消息内容估算tokens
func (cc *costCalculator) EstimatePromptTokens(req *ChatCompletionRequest) int {
	totalChars := 0

	for _, msg := range req.Messages {
		if content, ok := msg["content"].(string); ok {
			totalChars += len(content)
		}
	}

	// 使用配置的比例估算
	tokens := int(float64(totalChars) * cc.config.DefaultTokenRatio)

	return tokens
}

// EstimateCompletionTokens 估算Completion tokens
// ⭐ 核心方法：估算completion tokens
func (cc *costCalculator) EstimateCompletionTokens(req *ChatCompletionRequest) int {
	// 如果有max_tokens，使用max_tokens
	if req.MaxTokens > 0 {
		return req.MaxTokens
	}

	// 否则使用默认值
	return cc.config.DefaultCompletionTokens
}

// GetModelPricing 获取模型定价
func (cc *costCalculator) GetModelPricing(modelID string) (*model.ModelPricing, error) {
	return cc.modelRepo.GetPricing(modelID)
}

// CalculateProfit 计算利润
func (cc *costCalculator) CalculateProfit(upstreamCost, downstreamCost decimal.Decimal) decimal.Decimal {
	return downstreamCost.Sub(upstreamCost)
}

// CalculateMargin 计算利润率
func (cc *costCalculator) CalculateMargin(upstreamCost, downstreamCost decimal.Decimal) float64 {
	if downstreamCost.IsZero() {
		return 0.0
	}

	profit := cc.CalculateProfit(upstreamCost, downstreamCost)
	margin, _ := profit.Div(downstreamCost).Float64()

	return margin * 100 // 转换为百分比
}

// GetCostBreakdown 获取成本明细
// ⭐ 核心方法：提供详细的成本分析
//
// ⚠️ 修复：当没有可用的 Provider 时，返回错误而不是默认值
// 这样可以让调用者明确知道没有可用的 Provider，而不是返回误导性的默认数据
func (cc *costCalculator) GetCostBreakdown(modelID string, promptTokens, completionTokens int) (*CostBreakdown, error) {
	// 计算下游成本
	downstreamCost, err := cc.CalculateCost(modelID, promptTokens, completionTokens)
	if err != nil {
		return nil, err
	}

	// 获取Provider（使用第一个Provider）
	providers, err := cc.providerRepo.GetAvailableProviders(context.Background(), model.OperationChatCompletions, modelID)
	if err != nil {
		return nil, fmt.Errorf("获取可用Provider失败: %w", err)
	}

	if len(providers) == 0 {
		// ⭐ 修复：返回错误而不是默认值
		// 没有可用的 Provider 时，无法计算上游成本和利润
		return nil, fmt.Errorf("没有可用的Provider支持模型: %s", modelID)
	}

	// 使用第一个Provider计算上游成本
	provider := providers[0]
	upstreamCost := cc.CalculateProviderCost(provider, promptTokens, completionTokens)

	// 计算利润和利润率
	profit := cc.CalculateProfit(upstreamCost, downstreamCost)
	margin := cc.CalculateMargin(upstreamCost, downstreamCost)

	breakdown := &CostBreakdown{
		ModelID:          modelID,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      promptTokens + completionTokens,
		UpstreamCost:     upstreamCost,
		DownstreamCost:   downstreamCost,
		Profit:           profit,
		Margin:           margin,
		CostPer1KTokens:  cc.calculateCostPer1K(downstreamCost, promptTokens+completionTokens),
	}

	return breakdown, nil
}

// ==================== 辅助方法 ====================

// calculateInputCost 计算输入成本
func (cc *costCalculator) calculateInputCost(pricing *model.ModelPricing, promptTokens int) decimal.Decimal {
	if pricing == nil || promptTokens <= 0 {
		return decimal.Zero
	}

	return pricing.InputPrice.
		Mul(decimal.NewFromInt(int64(promptTokens))).
		Div(decimal.NewFromInt(int64(pricing.PriceUnit)))
}

// calculateOutputCost 计算输出成本
func (cc *costCalculator) calculateOutputCost(pricing *model.ModelPricing, completionTokens int) decimal.Decimal {
	if pricing == nil || completionTokens <= 0 {
		return decimal.Zero
	}

	return pricing.OutputPrice.
		Mul(decimal.NewFromInt(int64(completionTokens))).
		Div(decimal.NewFromInt(int64(pricing.PriceUnit)))
}

// calculateCostPer1K 计算每1K tokens的成本
func (cc *costCalculator) calculateCostPer1K(cost decimal.Decimal, totalTokens int) decimal.Decimal {
	if totalTokens <= 0 {
		return decimal.Zero
	}

	return cost.Mul(decimal.NewFromInt(1000)).Div(decimal.NewFromInt(int64(totalTokens)))
}

// ==================== 成本优化建议 ====================

// CostOptimizationSuggestion 成本优化建议
type CostOptimizationSuggestion struct {
	ModelID             string          `json:"model_id"`
	CurrentProvider     string          `json:"current_provider"`
	RecommendedProvider string          `json:"recommended_provider"`
	Savings             decimal.Decimal `json:"savings"`
	SavingsPercent      float64         `json:"savings_percent"`
	Reason              string          `json:"reason"`
}

// GetCostOptimizationSuggestions 获取成本优化建议
func (cc *costCalculator) GetCostOptimizationSuggestions(modelID string) ([]*CostOptimizationSuggestion, error) {
	// 获取所有可用的Provider
	providers, err := cc.providerRepo.GetAvailableProviders(context.Background(), model.OperationChatCompletions, modelID)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("没有可用的Provider")
	}

	// 按成本排序
	sortedProviders := make([]*model.ModelProvider, len(providers))
	copy(sortedProviders, providers)

	// 计算每个Provider的平均成本
	type providerWithCost struct {
		provider *model.ModelProvider
		avgCost  decimal.Decimal
	}

	providersWithCost := make([]providerWithCost, 0, len(providers))
	for _, provider := range sortedProviders {
		avgCost := provider.ActualCostPer1kInput.Add(provider.ActualCostPer1kOutput).Div(decimal.NewFromInt(2))
		providersWithCost = append(providersWithCost, providerWithCost{
			provider: provider,
			avgCost:  avgCost,
		})
	}

	// 按成本排序
	for i := 0; i < len(providersWithCost); i++ {
		for j := i + 1; j < len(providersWithCost); j++ {
			if providersWithCost[i].avgCost.GreaterThan(providersWithCost[j].avgCost) {
				providersWithCost[i], providersWithCost[j] = providersWithCost[j], providersWithCost[i]
			}
		}
	}

	// 生成优化建议
	suggestions := make([]*CostOptimizationSuggestion, 0)

	// 找出最便宜的Provider
	if len(providersWithCost) > 1 {
		cheapest := providersWithCost[0]

		for i := 1; i < len(providersWithCost); i++ {
			current := providersWithCost[i]

			if current.avgCost.GreaterThan(cheapest.avgCost) {
				// 计算节省金额
				savings := current.avgCost.Sub(cheapest.avgCost)
				savingsPercent, _ := savings.Div(current.avgCost).Mul(decimal.NewFromInt(100)).Float64()

				suggestion := &CostOptimizationSuggestion{
					ModelID:             modelID,
					CurrentProvider:     cc.getProviderName(current.provider),
					RecommendedProvider: cc.getProviderName(cheapest.provider),
					Savings:             savings,
					SavingsPercent:      math.Abs(savingsPercent),
					Reason:              "使用更便宜的Provider可以降低成本",
				}

				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions, nil
}

// getProviderName 获取Provider名称
func (cc *costCalculator) getProviderName(provider *model.ModelProvider) string {
	if provider.Channel != nil {
		return provider.Channel.Name
	}
	return fmt.Sprintf("Provider-%d", provider.ID)
}

// ==================== 成本分析 ====================

// CostAnalysis 成本分析
type CostAnalysis struct {
	TotalCost         decimal.Decimal `json:"total_cost"`
	TotalUpstreamCost decimal.Decimal `json:"total_upstream_cost"`
	TotalProfit       decimal.Decimal `json:"total_profit"`
	AvgMargin         float64         `json:"avg_margin"`
	TotalTokens       int64           `json:"total_tokens"`
	CostPer1KTokens   decimal.Decimal `json:"cost_per_1k_tokens"`
	TopCostProviders  []ProviderCost  `json:"top_cost_providers"`
}

// ProviderCost Provider成本
type ProviderCost struct {
	ProviderID   uint            `json:"provider_id"`
	ProviderName string          `json:"provider_name"`
	TotalCost    decimal.Decimal `json:"total_cost"`
	RequestCount int64           `json:"request_count"`
	AvgCost      decimal.Decimal `json:"avg_cost"`
}

// AnalyzeCosts 分析成本
// ⭐ 核心方法：分析指定时间范围内的成本数据
func (cc *costCalculator) AnalyzeCosts(modelID string, startTime, endTime time.Time) (*CostAnalysis, error) {
	// 获取所有Provider
	providers, err := cc.providerRepo.GetAvailableProviders(context.Background(), model.OperationChatCompletions, modelID)
	if err != nil {
		return nil, fmt.Errorf("获取Provider失败: %w", err)
	}

	if len(providers) == 0 {
		return &CostAnalysis{
			TotalCost:         decimal.Zero,
			TotalUpstreamCost: decimal.Zero,
			TotalProfit:       decimal.Zero,
			AvgMargin:         0.0,
			TotalTokens:       0,
			CostPer1KTokens:   decimal.Zero,
			TopCostProviders:  []ProviderCost{},
		}, nil
	}

	// 获取模型定价
	pricing, err := cc.GetModelPricing(modelID)
	if err != nil {
		return nil, fmt.Errorf("获取模型定价失败: %w", err)
	}

	// 计算每个Provider的成本
	providerCosts := make([]ProviderCost, 0, len(providers))
	var totalCost decimal.Decimal
	var totalUpstreamCost decimal.Decimal
	var totalProfit decimal.Decimal
	var totalTokens int64
	var totalMargin float64

	for _, provider := range providers {
		// 获取Provider的metrics
		metrics, err := cc.metricsRepo.GetAggregatedMetrics(context.Background(), provider.ID, startTime, endTime)
		if err != nil {
			continue
		}

		if metrics == nil || metrics.TotalRequests == 0 {
			continue
		}

		// 计算上游成本
		upstreamCost := cc.CalculateProviderCost(provider, int(metrics.TotalInputTokens), int(metrics.TotalOutputTokens))

		// 计算下游成本
		downstreamCost := cc.calculateDownstreamCost(pricing, int(metrics.TotalInputTokens), int(metrics.TotalOutputTokens))

		// 计算利润
		profit := cc.CalculateProfit(upstreamCost, downstreamCost)

		// 计算利润率
		margin := cc.CalculateMargin(upstreamCost, downstreamCost)

		// 累计
		totalCost = totalCost.Add(downstreamCost)
		totalUpstreamCost = totalUpstreamCost.Add(upstreamCost)
		totalProfit = totalProfit.Add(profit)
		totalTokens += metrics.TotalInputTokens + metrics.TotalOutputTokens
		totalMargin += margin

		// 添加到Provider成本列表
		avgCost := downstreamCost.Div(decimal.NewFromInt(metrics.TotalRequests))
		providerCosts = append(providerCosts, ProviderCost{
			ProviderID:   provider.ID,
			ProviderName: cc.getProviderName(provider),
			TotalCost:    downstreamCost,
			RequestCount: metrics.TotalRequests,
			AvgCost:      avgCost,
		})
	}

	// 计算平均利润率
	avgMargin := 0.0
	if len(providerCosts) > 0 {
		avgMargin = totalMargin / float64(len(providerCosts))
	}

	// 计算每1K tokens的成本
	costPer1KTokens := decimal.Zero
	if totalTokens > 0 {
		costPer1KTokens = totalCost.Mul(decimal.NewFromInt(1000)).Div(decimal.NewFromInt(totalTokens))
	}

	// 按成本排序Top 5
	topCostProviders := cc.getTopCostProviders(providerCosts, 5)

	return &CostAnalysis{
		TotalCost:         totalCost,
		TotalUpstreamCost: totalUpstreamCost,
		TotalProfit:       totalProfit,
		AvgMargin:         avgMargin,
		TotalTokens:       totalTokens,
		CostPer1KTokens:   costPer1KTokens,
		TopCostProviders:  topCostProviders,
	}, nil
}

// calculateDownstreamCost 计算下游成本
func (cc *costCalculator) calculateDownstreamCost(pricing *model.ModelPricing, promptTokens, completionTokens int) decimal.Decimal {
	if pricing == nil {
		return decimal.Zero
	}

	inputCost := pricing.InputPrice.
		Mul(decimal.NewFromInt(int64(promptTokens))).
		Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	outputCost := pricing.OutputPrice.
		Mul(decimal.NewFromInt(int64(completionTokens))).
		Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	return inputCost.Add(outputCost)
}

// getTopCostProviders 获取成本最高的Provider
func (cc *costCalculator) getTopCostProviders(providerCosts []ProviderCost, limit int) []ProviderCost {
	if len(providerCosts) <= limit {
		return providerCosts
	}

	// 按成本降序排序
	sorted := make([]ProviderCost, len(providerCosts))
	copy(sorted, providerCosts)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].TotalCost.LessThan(sorted[j].TotalCost) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted[:limit]
}
