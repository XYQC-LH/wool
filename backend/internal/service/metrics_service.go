package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

var ErrInvalidMetricsGranularity = errors.New("无效的 granularity")

// MetricsService 监控指标服务
type MetricsService interface {
	Query(ctx context.Context, query *MetricsQuery) ([]*MetricsQueryItem, *model.Pagination, error)
	GetRealtime(ctx context.Context, window time.Duration) (*RealtimeMetrics, error)
}

// MetricsQuery 指标查询参数
type MetricsQuery struct {
	ProviderID  *uint
	ModelID     string
	Granularity model.MetricGranularity
	StartTime   time.Time
	EndTime     time.Time
	Page        int
	PageSize    int
}

// MetricsQueryItem 指标查询结果
type MetricsQueryItem struct {
	ProviderID    uint                   `json:"provider_id"`
	ModelID       string                 `json:"model_id"`
	ModelName     string                 `json:"model_name"`
	ChannelID     uint                   `json:"channel_id"`
	ChannelName   string                 `json:"channel_name"`
	MetricTime    time.Time              `json:"metric_time"`
	Granularity   model.MetricGranularity `json:"granularity"`
	RequestCount  int                    `json:"request_count"`
	SuccessCount  int                    `json:"success_count"`
	FailureCount  int                    `json:"failure_count"`
	SuccessRate   float64                `json:"success_rate"`
	AvgLatencyMs  int                    `json:"avg_latency_ms"`
	InputTokens   int64                  `json:"input_tokens"`
	OutputTokens  int64                  `json:"output_tokens"`
	TotalTokens   int64                  `json:"total_tokens"`
	ActualCost    float64                `json:"actual_cost"`
	Revenue       float64                `json:"revenue"`
	Profit        float64                `json:"profit"`
}

// RealtimeMetrics 实时监控指标
type RealtimeMetrics struct {
	WindowSeconds int                 `json:"window_seconds"`
	StartTime     time.Time           `json:"start_time"`
	EndTime       time.Time           `json:"end_time"`
	RequestCount  int64               `json:"request_count"`
	SuccessCount  int64               `json:"success_count"`
	FailureCount  int64               `json:"failure_count"`
	SuccessRate   float64             `json:"success_rate"`
	AvgLatencyMs  float64             `json:"avg_latency_ms"`
	TotalTokens   int64               `json:"total_tokens"`
	TotalRevenue  float64             `json:"total_revenue"`
	TotalCost     float64             `json:"total_cost"`
	TotalProfit   float64             `json:"total_profit"`
	ActiveAlerts  int64               `json:"active_alerts"`
	Providers     ProviderHealthStats `json:"providers"`
}

// ProviderHealthStats 源头健康摘要
type ProviderHealthStats struct {
	Total       int64 `json:"total"`
	Active      int64 `json:"active"`
	CircuitOpen int64 `json:"circuit_open"`
}

type MetricsStore interface {
	QueryMetrics(ctx context.Context, query *MetricsQuery) ([]*metricsQueryRow, int64, error)
	QueryRealtimeSummary(ctx context.Context, startTime, endTime time.Time) (*realtimeSummaryRow, error)
	QueryProviderHealth(ctx context.Context) (*providerHealthRow, error)
	CountActiveAlerts(ctx context.Context) (int64, error)
}

type metricsService struct {
	store MetricsStore
}

type gormMetricsStore struct {
	db *gorm.DB
}

type metricsQueryRow struct {
	ProviderID   uint                   `gorm:"column:provider_id"`
	ModelID      string                 `gorm:"column:model_id"`
	ModelName    string                 `gorm:"column:model_name"`
	ChannelID    uint                   `gorm:"column:channel_id"`
	ChannelName  string                 `gorm:"column:channel_name"`
	MetricTime   time.Time              `gorm:"column:metric_time"`
	Granularity  model.MetricGranularity `gorm:"column:granularity"`
	RequestCount int                    `gorm:"column:request_count"`
	SuccessCount int                    `gorm:"column:success_count"`
	FailureCount int                    `gorm:"column:failure_count"`
	AvgLatencyMs int                    `gorm:"column:avg_latency_ms"`
	InputTokens  int64                  `gorm:"column:input_tokens"`
	OutputTokens int64                  `gorm:"column:output_tokens"`
	ActualCost   float64                `gorm:"column:actual_cost"`
	Revenue      float64                `gorm:"column:revenue"`
	Profit       float64                `gorm:"column:profit"`
}

type realtimeSummaryRow struct {
	RequestCount int64   `gorm:"column:request_count"`
	SuccessCount int64   `gorm:"column:success_count"`
	FailureCount int64   `gorm:"column:failure_count"`
	AvgLatencyMs float64 `gorm:"column:avg_latency_ms"`
	TotalTokens  int64   `gorm:"column:total_tokens"`
	TotalRevenue float64 `gorm:"column:total_revenue"`
	TotalCost    float64 `gorm:"column:total_cost"`
}

type providerHealthRow struct {
	Total       int64 `gorm:"column:total"`
	Active      int64 `gorm:"column:active"`
	CircuitOpen int64 `gorm:"column:circuit_open"`
}

func NewMetricsService(db *gorm.DB) MetricsService {
	return newMetricsServiceWithStore(newGormMetricsStore(db))
}

func newMetricsServiceWithStore(store MetricsStore) *metricsService {
	return &metricsService{store: store}
}

func newGormMetricsStore(db *gorm.DB) MetricsStore {
	return &gormMetricsStore{db: db}
}

func (s *metricsService) Query(ctx context.Context, query *MetricsQuery) ([]*MetricsQueryItem, *model.Pagination, error) {
	if s == nil || s.store == nil {
		return nil, nil, errors.New("metrics service 未初始化")
	}

	normalized, err := normalizeMetricsQuery(query)
	if err != nil {
		return nil, nil, err
	}

	rows, total, err := s.store.QueryMetrics(ctx, normalized)
	if err != nil {
		return nil, nil, err
	}

	items := make([]*MetricsQueryItem, 0, len(rows))
	for _, row := range rows {
		successRate := float64(0)
		if row.RequestCount > 0 {
			successRate = float64(row.SuccessCount) / float64(row.RequestCount) * 100
		}

		items = append(items, &MetricsQueryItem{
			ProviderID:   row.ProviderID,
			ModelID:      row.ModelID,
			ModelName:    row.ModelName,
			ChannelID:    row.ChannelID,
			ChannelName:  row.ChannelName,
			MetricTime:   row.MetricTime,
			Granularity:  row.Granularity,
			RequestCount: row.RequestCount,
			SuccessCount: row.SuccessCount,
			FailureCount: row.FailureCount,
			SuccessRate:  successRate,
			AvgLatencyMs: row.AvgLatencyMs,
			InputTokens:  row.InputTokens,
			OutputTokens: row.OutputTokens,
			TotalTokens:  row.InputTokens + row.OutputTokens,
			ActualCost:   row.ActualCost,
			Revenue:      row.Revenue,
			Profit:       row.Profit,
		})
	}

	return items, model.NewPagination(normalized.Page, normalized.PageSize, total), nil
}

func (s *metricsService) GetRealtime(ctx context.Context, window time.Duration) (*RealtimeMetrics, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("metrics service 未初始化")
	}

	if window <= 0 {
		window = 5 * time.Minute
	}
	if window > 24*time.Hour {
		window = 24 * time.Hour
	}

	endTime := time.Now()
	startTime := endTime.Add(-window)

	realtimeRow, err := s.store.QueryRealtimeSummary(ctx, startTime, endTime)
	if err != nil {
		return nil, err
	}
	if realtimeRow == nil {
		realtimeRow = &realtimeSummaryRow{}
	}

	providerRow, err := s.store.QueryProviderHealth(ctx)
	if err != nil {
		return nil, err
	}
	if providerRow == nil {
		providerRow = &providerHealthRow{}
	}

	activeAlerts, err := s.store.CountActiveAlerts(ctx)
	if err != nil {
		return nil, err
	}

	successRate := float64(0)
	if realtimeRow.RequestCount > 0 {
		successRate = float64(realtimeRow.SuccessCount) / float64(realtimeRow.RequestCount) * 100
	}

	totalProfit := realtimeRow.TotalRevenue - realtimeRow.TotalCost

	return &RealtimeMetrics{
		WindowSeconds: int(window.Seconds()),
		StartTime:     startTime,
		EndTime:       endTime,
		RequestCount:  realtimeRow.RequestCount,
		SuccessCount:  realtimeRow.SuccessCount,
		FailureCount:  realtimeRow.FailureCount,
		SuccessRate:   successRate,
		AvgLatencyMs:  realtimeRow.AvgLatencyMs,
		TotalTokens:   realtimeRow.TotalTokens,
		TotalRevenue:  realtimeRow.TotalRevenue,
		TotalCost:     realtimeRow.TotalCost,
		TotalProfit:   totalProfit,
		ActiveAlerts:  activeAlerts,
		Providers: ProviderHealthStats{
			Total:       providerRow.Total,
			Active:      providerRow.Active,
			CircuitOpen: providerRow.CircuitOpen,
		},
	}, nil
}

func normalizeMetricsQuery(query *MetricsQuery) (*MetricsQuery, error) {
	if query == nil {
		query = &MetricsQuery{}
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}
	if query.PageSize > 200 {
		query.PageSize = 200
	}

	if query.Granularity == "" {
		query.Granularity = model.MetricGranularityMinute
	}

	switch query.Granularity {
	case model.MetricGranularityMinute, model.MetricGranularityHour, model.MetricGranularityDay:
	default:
		return nil, fmt.Errorf("%w: 仅支持 minute/hour/day", ErrInvalidMetricsGranularity)
	}

	if query.EndTime.IsZero() {
		query.EndTime = time.Now()
	}
	if query.StartTime.IsZero() {
		query.StartTime = query.EndTime.Add(-24 * time.Hour)
	}
	if !query.StartTime.Before(query.EndTime) {
		return nil, errors.New("start_time 必须早于 end_time")
	}

	query.ModelID = strings.TrimSpace(query.ModelID)
	return query, nil
}

func (s *gormMetricsStore) QueryMetrics(ctx context.Context, query *MetricsQuery) ([]*metricsQueryRow, int64, error) {
	if s == nil || s.db == nil {
		return nil, 0, errors.New("db 未初始化")
	}

	base := s.db.WithContext(ctx).
		Table("provider_metrics").
		Joins("JOIN model_providers ON model_providers.id = provider_metrics.provider_id").
		Joins("LEFT JOIN models ON models.id = model_providers.model_id").
		Joins("LEFT JOIN channels ON channels.id = model_providers.channel_id")

	if query.ProviderID != nil && *query.ProviderID > 0 {
		base = base.Where("provider_metrics.provider_id = ?", *query.ProviderID)
	}
	if query.ModelID != "" {
		base = base.Where("model_providers.model_id = ?", query.ModelID)
	}
	base = base.Where("provider_metrics.granularity = ?", query.Granularity)
	base = base.Where("provider_metrics.metric_time >= ? AND provider_metrics.metric_time <= ?", query.StartTime, query.EndTime)

	var total int64
	if err := base.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	var rows []*metricsQueryRow
	err := base.
		Select(`
			provider_metrics.provider_id,
			COALESCE(model_providers.model_id, '') as model_id,
			COALESCE(models.name, '') as model_name,
			COALESCE(model_providers.channel_id, 0) as channel_id,
			COALESCE(channels.name, '') as channel_name,
			provider_metrics.metric_time,
			provider_metrics.granularity,
			provider_metrics.request_count,
			provider_metrics.success_count,
			provider_metrics.failure_count,
			provider_metrics.avg_latency_ms,
			provider_metrics.input_tokens,
			provider_metrics.output_tokens,
			COALESCE(provider_metrics.actual_cost, 0)::float8 as actual_cost,
			COALESCE(provider_metrics.revenue, 0)::float8 as revenue,
			COALESCE(provider_metrics.profit, 0)::float8 as profit
		`).
		Order("provider_metrics.metric_time DESC").
		Offset((query.Page - 1) * query.PageSize).
		Limit(query.PageSize).
		Scan(&rows).Error
	if err != nil {
		return nil, 0, err
	}

	return rows, total, nil
}

func (s *gormMetricsStore) QueryRealtimeSummary(ctx context.Context, startTime, endTime time.Time) (*realtimeSummaryRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db 未初始化")
	}

	var row realtimeSummaryRow
	err := s.db.WithContext(ctx).
		Table("logs").
		Select(`
			COUNT(*) as request_count,
			COALESCE(SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END), 0) as success_count,
			COALESCE(SUM(CASE WHEN status <> 'success' THEN 1 ELSE 0 END), 0) as failure_count,
			COALESCE(AVG(duration), 0) as avg_latency_ms,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0)::float8 as total_revenue,
			COALESCE(SUM(upstream_cost), 0)::float8 as total_cost
		`).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *gormMetricsStore) QueryProviderHealth(ctx context.Context) (*providerHealthRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db 未初始化")
	}

	var row providerHealthRow
	err := s.db.WithContext(ctx).
		Table("model_providers").
		Select(`
			COUNT(*) as total,
			COALESCE(SUM(CASE WHEN status = 'active' AND circuit_state = 'closed' THEN 1 ELSE 0 END), 0) as active,
			COALESCE(SUM(CASE WHEN circuit_state = 'open' THEN 1 ELSE 0 END), 0) as circuit_open
		`).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *gormMetricsStore) CountActiveAlerts(ctx context.Context) (int64, error) {
	if s == nil || s.db == nil {
		return 0, errors.New("db 未初始化")
	}

	var total int64
	err := s.db.WithContext(ctx).
		Table("alerts").
		Where("status = ?", model.AlertStatusActive).
		Count(&total).Error
	return total, err
}
