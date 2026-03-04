package handler

import (
	"context"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TopologyHandler 模型/源头链路拓扑处理器（用于管理端可视化）。
//
// 设计目标（KISS）：
// - 一次请求返回 “Model -> ProviderGroup(model_providers) -> ProviderInstance(provider_instances)” 的层级结构
// - 提供近窗口 metrics（来自 provider_metrics），用于实时显示负载/成功率
type TopologyHandler struct {
	db *gorm.DB
}

func NewTopologyHandler(db *gorm.DB) *TopologyHandler {
	return &TopologyHandler{db: db}
}

type TopologyProviderMetrics struct {
	WindowSeconds int     `json:"window_seconds"`
	RequestCount  int64   `json:"request_count"`
	SuccessRate   float64 `json:"success_rate"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
}

type TopologyProviderPricingRule struct {
	ID           uint            `json:"id"`
	ProviderID   uint            `json:"provider_id"`
	Operation    string          `json:"operation"`
	Unit         string          `json:"unit"`
	CostPerUnit  decimal.Decimal `json:"cost_per_unit"`
	PricePerUnit decimal.Decimal `json:"price_per_unit"`
	Enabled      bool            `json:"enabled"`
	UpdatedAt    time.Time       `json:"updated_at"`
}

type TopologyInstance struct {
	ID                  uint                 `json:"id"`
	ProviderID          uint                 `json:"provider_id"`
	Name                string               `json:"name"`
	InstanceType        model.InstanceType   `json:"instance_type"`
	Status              model.InstanceStatus `json:"status"`
	Weight              int                  `json:"weight"`
	MaxConcurrency      int                  `json:"max_concurrency"`
	RPMLimit            int                  `json:"rpm_limit"`
	TPMLimit            int                  `json:"tpm_limit"`
	ResourceAccountID   *uint                `json:"resource_account_id,omitempty"`
	ResourceAccountName *string              `json:"resource_account_name,omitempty"`
}

type TopologyProvider struct {
	ID                    uint                      `json:"id"`
	Operation             string                    `json:"operation"`
	ModelID               string                    `json:"model_id"`
	ModelName             string                    `json:"model_name"`
	ChannelID             uint                      `json:"channel_id"`
	ChannelName           string                    `json:"channel_name"`
	UpstreamModelName     string                    `json:"upstream_model_name"`
	ActualCostPer1kInput  decimal.Decimal           `json:"actual_cost_per_1k_input"`
	ActualCostPer1kOutput decimal.Decimal           `json:"actual_cost_per_1k_output"`
	Status                model.ModelProviderStatus `json:"status"`
	CircuitState          model.CircuitState        `json:"circuit_state"`
	HealthScore           decimal.Decimal           `json:"health_score"`
	TotalRequests         int64                     `json:"total_requests"`

	Metrics      *TopologyProviderMetrics      `json:"metrics,omitempty"`
	PricingRules []TopologyProviderPricingRule `json:"pricing_rules,omitempty"`
	Instances    []TopologyInstance            `json:"instances"`
}

type TopologyOperation struct {
	Operation string             `json:"operation"`
	Providers []TopologyProvider `json:"providers"`
}

type TopologyModel struct {
	ID         string              `json:"id"`
	Name       string              `json:"name"`
	Operations []TopologyOperation `json:"operations"`
}

type ModelProviderTopologyResponse struct {
	GeneratedAt time.Time       `json:"generated_at"`
	Models      []TopologyModel `json:"models"`
}

func (h *TopologyHandler) GetModelProviderTopology(c *gin.Context) {
	if h == nil || h.db == nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "db 未初始化"))
		return
	}

	modelID := strings.TrimSpace(c.Query("model_id"))
	operation := strings.TrimSpace(c.Query("operation"))
	if operation != "" {
		operation = model.NormalizeOperation(operation)
	}

	includeInstances := parseBoolQuery(c.DefaultQuery("include_instances", "true"), true)
	includePricingRules := parseBoolQuery(c.DefaultQuery("include_pricing_rules", "false"), false)

	windowSeconds, _ := strconv.Atoi(strings.TrimSpace(c.DefaultQuery("metrics_window_seconds", "300")))
	if windowSeconds <= 0 {
		windowSeconds = 300
	}
	if windowSeconds > 3600 {
		windowSeconds = 3600
	}

	type providerRow struct {
		ID                    uint
		Operation             string
		ModelID               string
		ModelName             string
		ChannelID             uint
		ChannelName           string
		UpstreamModelName     string
		ActualCostPer1kInput  decimal.Decimal
		ActualCostPer1kOutput decimal.Decimal
		Status                model.ModelProviderStatus
		CircuitState          model.CircuitState
		HealthScore           decimal.Decimal
		TotalRequests         int64
	}

	query := h.db.Table("model_providers").
		Select(`
			model_providers.id,
			model_providers.operation,
			model_providers.model_id,
			COALESCE(models.display_name, models.name, '') as model_name,
			model_providers.channel_id,
			COALESCE(channels.name, '') as channel_name,
			COALESCE(model_providers.upstream_model_name, '') as upstream_model_name,
			model_providers.actual_cost_per_1k_input,
			model_providers.actual_cost_per_1k_output,
			model_providers.status,
			model_providers.circuit_state,
			model_providers.health_score,
			model_providers.total_requests
		`).
		Joins("LEFT JOIN models ON models.id = model_providers.model_id").
		Joins("LEFT JOIN channels ON channels.id = model_providers.channel_id")

	if modelID != "" {
		query = query.Where("model_providers.model_id = ?", modelID)
	}
	if operation != "" {
		query = query.Where("model_providers.operation = ?", operation)
	}

	var providers []providerRow
	if err := query.Order("model_providers.model_id ASC, model_providers.operation ASC, channels.name ASC, model_providers.id ASC").Scan(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询 model_providers 失败: "+err.Error()))
		return
	}

	if len(providers) == 0 {
		c.JSON(http.StatusOK, model.SuccessResponse(&ModelProviderTopologyResponse{
			GeneratedAt: time.Now(),
			Models:      []TopologyModel{},
		}))
		return
	}

	providerIDs := make([]uint, 0, len(providers))
	for _, p := range providers {
		providerIDs = append(providerIDs, p.ID)
	}

	instancesByProviderID := map[uint][]TopologyInstance{}
	if includeInstances {
		var err error
		instancesByProviderID, err = h.listInstancesByProviderID(c.Request.Context(), providerIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询 provider_instances 失败: "+err.Error()))
			return
		}
	} else {
		for _, pid := range providerIDs {
			instancesByProviderID[pid] = []TopologyInstance{}
		}
	}

	metricsByProviderID, err := h.aggregateRecentProviderMetrics(c.Request.Context(), providerIDs, windowSeconds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询 provider_metrics 失败: "+err.Error()))
		return
	}

	pricingRulesByProviderID := map[uint][]TopologyProviderPricingRule{}
	if includePricingRules {
		var err error
		pricingRulesByProviderID, err = h.listPricingRulesByProviderID(c.Request.Context(), providerIDs)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询 provider_pricing_rules 失败: "+err.Error()))
			return
		}
	}

	type opBuilder struct {
		Operation string
		Providers []TopologyProvider
	}
	type modelBuilder struct {
		ID         string
		Name       string
		Operations map[string]*opBuilder
	}

	modelBuilders := make(map[string]*modelBuilder, 16)

	for _, p := range providers {
		mb, ok := modelBuilders[p.ModelID]
		if !ok {
			mb = &modelBuilder{
				ID:         p.ModelID,
				Name:       p.ModelName,
				Operations: make(map[string]*opBuilder, 8),
			}
			modelBuilders[p.ModelID] = mb
		}

		op := model.NormalizeOperation(p.Operation)
		ob, ok := mb.Operations[op]
		if !ok {
			ob = &opBuilder{
				Operation: op,
				Providers: []TopologyProvider{},
			}
			mb.Operations[op] = ob
		}

		ob.Providers = append(ob.Providers, TopologyProvider{
			ID:                    p.ID,
			Operation:             op,
			ModelID:               p.ModelID,
			ModelName:             p.ModelName,
			ChannelID:             p.ChannelID,
			ChannelName:           p.ChannelName,
			UpstreamModelName:     p.UpstreamModelName,
			ActualCostPer1kInput:  p.ActualCostPer1kInput,
			ActualCostPer1kOutput: p.ActualCostPer1kOutput,
			Status:                p.Status,
			CircuitState:          p.CircuitState,
			HealthScore:           p.HealthScore,
			TotalRequests:         p.TotalRequests,
			Instances:             instancesByProviderID[p.ID],
			Metrics:               metricsByProviderID[p.ID],
			PricingRules:          pricingRulesByProviderID[p.ID],
		})
	}

	models := make([]TopologyModel, 0, len(modelBuilders))
	for _, mb := range modelBuilders {
		ops := make([]TopologyOperation, 0, len(mb.Operations))
		for _, ob := range mb.Operations {
			// providers 内部按 channel_name 排序（更稳定）
			sort.Slice(ob.Providers, func(i, j int) bool {
				if ob.Providers[i].ChannelName == ob.Providers[j].ChannelName {
					return ob.Providers[i].ID < ob.Providers[j].ID
				}
				return ob.Providers[i].ChannelName < ob.Providers[j].ChannelName
			})

			ops = append(ops, TopologyOperation{
				Operation: ob.Operation,
				Providers: ob.Providers,
			})
		}
		sort.Slice(ops, func(i, j int) bool { return ops[i].Operation < ops[j].Operation })

		models = append(models, TopologyModel{
			ID:         mb.ID,
			Name:       mb.Name,
			Operations: ops,
		})
	}
	sort.Slice(models, func(i, j int) bool { return models[i].ID < models[j].ID })

	c.JSON(http.StatusOK, model.SuccessResponse(&ModelProviderTopologyResponse{
		GeneratedAt: time.Now(),
		Models:      models,
	}))
}

func (h *TopologyHandler) listInstancesByProviderID(ctx context.Context, providerIDs []uint) (map[uint][]TopologyInstance, error) {
	if h == nil || h.db == nil || len(providerIDs) == 0 {
		return map[uint][]TopologyInstance{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	type instanceRow struct {
		ID                  uint                 `gorm:"column:id"`
		ProviderID          uint                 `gorm:"column:provider_id"`
		Name                string               `gorm:"column:name"`
		InstanceType        model.InstanceType   `gorm:"column:instance_type"`
		Status              model.InstanceStatus `gorm:"column:status"`
		Weight              int                  `gorm:"column:weight"`
		MaxConcurrency      int                  `gorm:"column:max_concurrency"`
		RPMLimit            int                  `gorm:"column:rpm_limit"`
		TPMLimit            int                  `gorm:"column:tpm_limit"`
		ResourceAccountID   *uint                `gorm:"column:resource_account_id"`
		ResourceAccountName *string              `gorm:"column:resource_account_name"`
	}

	var rows []instanceRow
	err := h.db.WithContext(ctx).
		Table("provider_instances").
		Select(`
			provider_instances.id,
			provider_instances.provider_id,
			provider_instances.name,
			provider_instances.instance_type,
			provider_instances.status,
			provider_instances.weight,
			provider_instances.max_concurrency,
			provider_instances.rpm_limit,
			provider_instances.tpm_limit,
			provider_instances.resource_account_id,
			resource_accounts.account_name as resource_account_name
		`).
		Joins("LEFT JOIN resource_accounts ON resource_accounts.id = provider_instances.resource_account_id").
		Where("provider_instances.provider_id IN ?", providerIDs).
		Order("provider_instances.provider_id ASC, provider_instances.id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint][]TopologyInstance, 16)
	for _, r := range rows {
		result[r.ProviderID] = append(result[r.ProviderID], TopologyInstance{
			ID:                  r.ID,
			ProviderID:          r.ProviderID,
			Name:                r.Name,
			InstanceType:        r.InstanceType,
			Status:              r.Status,
			Weight:              r.Weight,
			MaxConcurrency:      r.MaxConcurrency,
			RPMLimit:            r.RPMLimit,
			TPMLimit:            r.TPMLimit,
			ResourceAccountID:   r.ResourceAccountID,
			ResourceAccountName: r.ResourceAccountName,
		})
	}

	// 确保无实例的 provider 也有空数组（前端渲染更简单）
	for _, pid := range providerIDs {
		if _, ok := result[pid]; !ok {
			result[pid] = []TopologyInstance{}
		}
	}

	return result, nil
}

func (h *TopologyHandler) aggregateRecentProviderMetrics(ctx context.Context, providerIDs []uint, windowSeconds int) (map[uint]*TopologyProviderMetrics, error) {
	if h == nil || h.db == nil || len(providerIDs) == 0 {
		return map[uint]*TopologyProviderMetrics{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	start := time.Now().Add(-time.Duration(windowSeconds) * time.Second)
	end := time.Now()

	type metricRow struct {
		ProviderID   uint    `gorm:"column:provider_id"`
		RequestCount int64   `gorm:"column:request_count"`
		SuccessCount int64   `gorm:"column:success_count"`
		AvgLatencyMs float64 `gorm:"column:avg_latency_ms"`
	}

	var rows []metricRow
	err := h.db.WithContext(ctx).
		Table("provider_metrics").
		Select(`
			provider_id,
			COALESCE(SUM(request_count), 0) as request_count,
			COALESCE(SUM(success_count), 0) as success_count,
			COALESCE(SUM(avg_latency_ms * request_count) / NULLIF(SUM(request_count), 0), 0) as avg_latency_ms
		`).
		Where("provider_id IN ?", providerIDs).
		Where("granularity = ?", model.MetricGranularityMinute).
		Where("metric_time >= ? AND metric_time <= ?", start, end).
		Group("provider_id").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint]*TopologyProviderMetrics, len(providerIDs))
	for _, r := range rows {
		successRate := float64(0)
		if r.RequestCount > 0 {
			successRate = float64(r.SuccessCount) / float64(r.RequestCount) * 100
		}
		result[r.ProviderID] = &TopologyProviderMetrics{
			WindowSeconds: windowSeconds,
			RequestCount:  r.RequestCount,
			SuccessRate:   successRate,
			AvgLatencyMs:  r.AvgLatencyMs,
		}
	}

	return result, nil
}

func (h *TopologyHandler) listPricingRulesByProviderID(ctx context.Context, providerIDs []uint) (map[uint][]TopologyProviderPricingRule, error) {
	if h == nil || h.db == nil || len(providerIDs) == 0 {
		return map[uint][]TopologyProviderPricingRule{}, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	type ruleRow struct {
		ID           uint            `gorm:"column:id"`
		ProviderID   uint            `gorm:"column:provider_id"`
		Operation    string          `gorm:"column:operation"`
		Unit         string          `gorm:"column:unit"`
		CostPerUnit  decimal.Decimal `gorm:"column:cost_per_unit"`
		PricePerUnit decimal.Decimal `gorm:"column:price_per_unit"`
		Enabled      bool            `gorm:"column:enabled"`
		UpdatedAt    time.Time       `gorm:"column:updated_at"`
	}

	var rows []ruleRow
	err := h.db.WithContext(ctx).
		Table("provider_pricing_rules").
		Select(`
			id,
			provider_id,
			operation,
			unit,
			cost_per_unit,
			price_per_unit,
			enabled,
			updated_at
		`).
		Where("provider_id IN ?", providerIDs).
		Order("provider_id ASC, operation ASC, unit ASC, id ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	result := make(map[uint][]TopologyProviderPricingRule, 16)
	for _, r := range rows {
		result[r.ProviderID] = append(result[r.ProviderID], TopologyProviderPricingRule{
			ID:           r.ID,
			ProviderID:   r.ProviderID,
			Operation:    model.NormalizeOperation(r.Operation),
			Unit:         r.Unit,
			CostPerUnit:  r.CostPerUnit,
			PricePerUnit: r.PricePerUnit,
			Enabled:      r.Enabled,
			UpdatedAt:    r.UpdatedAt,
		})
	}

	for _, pid := range providerIDs {
		if _, ok := result[pid]; !ok {
			result[pid] = []TopologyProviderPricingRule{}
		}
	}

	return result, nil
}

func parseBoolQuery(value string, defaultValue bool) bool {
	v := strings.TrimSpace(strings.ToLower(value))
	if v == "" {
		return defaultValue
	}
	if v == "1" || v == "true" || v == "yes" || v == "y" || v == "on" {
		return true
	}
	if v == "0" || v == "false" || v == "no" || v == "n" || v == "off" {
		return false
	}
	return defaultValue
}

// RegisterTopologyRoutes 注册拓扑相关路由
func RegisterTopologyRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewTopologyHandler(db)

	topology := r.Group("/topology")
	{
		topology.GET("/model-providers", h.GetModelProviderTopology)
	}
}
