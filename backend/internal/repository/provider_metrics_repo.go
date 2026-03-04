package repository

import (
	"context"
	"time"

	"nexus-api/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ProviderMetricsRepository 源头指标仓库接口
type ProviderMetricsRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, metrics *model.ProviderMetrics) error
	BatchCreate(ctx context.Context, metrics []*model.ProviderMetrics) error

	// 查询方法
	GetByProviderID(ctx context.Context, providerID uint, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.ProviderMetrics, error)
	GetByModelID(ctx context.Context, modelID string, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.ProviderMetrics, error)
	GetLatest(ctx context.Context, providerID uint, granularity model.MetricGranularity) (*model.ProviderMetrics, error)

	// 聚合查询
	GetAggregatedMetrics(ctx context.Context, providerID uint, startTime, endTime time.Time) (*model.AggregatedMetrics, error)
	GetModelAggregatedMetrics(ctx context.Context, modelID string, startTime, endTime time.Time) (*model.AggregatedMetrics, error)
	GetTimeSeries(ctx context.Context, providerID uint, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.TimeSeriesMetric, error)
	GetTrafficDistribution(ctx context.Context, modelID string, startTime, endTime time.Time) ([]*model.ProviderTrafficDistribution, error)

	// 更新或插入
	UpsertMetrics(ctx context.Context, providerID uint, metricTime time.Time, granularity model.MetricGranularity, updates map[string]interface{}) error

	// 清理
	DeleteOldMetrics(ctx context.Context, before time.Time, granularity model.MetricGranularity) error

	// 熔断事件
	CreateCircuitEvent(ctx context.Context, event *model.CircuitEventRecord) error
	GetCircuitEvents(ctx context.Context, providerID uint, startTime, endTime time.Time) ([]*model.CircuitEventRecord, error)
	GetRecentCircuitEvents(ctx context.Context, limit int) ([]*model.CircuitEventRecord, error)
}

// providerMetricsRepository 源头指标仓库实现
type providerMetricsRepository struct {
	db *gorm.DB
}

// NewProviderMetricsRepository 创建源头指标仓库
func NewProviderMetricsRepository(db *gorm.DB) ProviderMetricsRepository {
	return &providerMetricsRepository{db: db}
}

// Create 创建指标记录
func (r *providerMetricsRepository) Create(ctx context.Context, metrics *model.ProviderMetrics) error {
	return r.db.WithContext(ctx).Create(metrics).Error
}

// BatchCreate 批量创建指标记录
func (r *providerMetricsRepository) BatchCreate(ctx context.Context, metrics []*model.ProviderMetrics) error {
	if len(metrics) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(metrics, 100).Error
}

// GetByProviderID 根据源头ID获取指标
func (r *providerMetricsRepository) GetByProviderID(ctx context.Context, providerID uint, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.ProviderMetrics, error) {
	var metrics []*model.ProviderMetrics
	err := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Where("granularity = ?", granularity).
		Where("metric_time >= ? AND metric_time <= ?", startTime, endTime).
		Order("metric_time ASC").
		Find(&metrics).Error
	return metrics, err
}

// GetByModelID 根据模型ID获取指标（聚合所有源头）
func (r *providerMetricsRepository) GetByModelID(ctx context.Context, modelID string, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.ProviderMetrics, error) {
	var metrics []*model.ProviderMetrics
	err := r.db.WithContext(ctx).
		Joins("JOIN model_providers ON model_providers.id = provider_metrics.provider_id").
		Where("model_providers.model_id = ?", modelID).
		Where("provider_metrics.granularity = ?", granularity).
		Where("provider_metrics.metric_time >= ? AND provider_metrics.metric_time <= ?", startTime, endTime).
		Order("provider_metrics.metric_time ASC").
		Find(&metrics).Error
	return metrics, err
}

// GetLatest 获取最新的指标记录
func (r *providerMetricsRepository) GetLatest(ctx context.Context, providerID uint, granularity model.MetricGranularity) (*model.ProviderMetrics, error) {
	var metrics model.ProviderMetrics
	err := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Where("granularity = ?", granularity).
		Order("metric_time DESC").
		First(&metrics).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &metrics, nil
}

// GetAggregatedMetrics 获取聚合指标
func (r *providerMetricsRepository) GetAggregatedMetrics(ctx context.Context, providerID uint, startTime, endTime time.Time) (*model.AggregatedMetrics, error) {
	var result struct {
		TotalRequests     int64           `gorm:"column:total_requests"`
		TotalSuccess      int64           `gorm:"column:total_success"`
		TotalFailure      int64           `gorm:"column:total_failure"`
		TotalTimeout      int64           `gorm:"column:total_timeout"`
		AvgLatencyMs      float64         `gorm:"column:avg_latency_ms"`
		MaxP99LatencyMs   int             `gorm:"column:max_p99_latency_ms"`
		TotalInputTokens  int64           `gorm:"column:total_input_tokens"`
		TotalOutputTokens int64           `gorm:"column:total_output_tokens"`
		TotalCost         decimal.Decimal `gorm:"column:total_cost"`
		TotalRevenue      decimal.Decimal `gorm:"column:total_revenue"`
		TotalProfit       decimal.Decimal `gorm:"column:total_profit"`
	}

	err := r.db.WithContext(ctx).
		Model(&model.ProviderMetrics{}).
		Select(`
			COALESCE(SUM(request_count), 0) as total_requests,
			COALESCE(SUM(success_count), 0) as total_success,
			COALESCE(SUM(failure_count), 0) as total_failure,
			COALESCE(SUM(timeout_count), 0) as total_timeout,
			COALESCE(AVG(avg_latency_ms), 0) as avg_latency_ms,
			COALESCE(MAX(p99_latency_ms), 0) as max_p99_latency_ms,
			COALESCE(SUM(input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(actual_cost), 0) as total_cost,
			COALESCE(SUM(revenue), 0) as total_revenue,
			COALESCE(SUM(profit), 0) as total_profit
		`).
		Where("provider_id = ?", providerID).
		Where("metric_time >= ? AND metric_time <= ?", startTime, endTime).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	successRate := float64(0)
	if result.TotalRequests > 0 {
		successRate = float64(result.TotalSuccess) / float64(result.TotalRequests) * 100
	}

	profitMargin := float64(0)
	if !result.TotalRevenue.IsZero() {
		profitMargin = result.TotalProfit.Div(result.TotalRevenue).InexactFloat64() * 100
	}

	return &model.AggregatedMetrics{
		TotalRequests:     result.TotalRequests,
		TotalSuccess:      result.TotalSuccess,
		TotalFailure:      result.TotalFailure,
		TotalTimeout:      result.TotalTimeout,
		SuccessRate:       successRate,
		AvgLatencyMs:      result.AvgLatencyMs,
		P99LatencyMs:      result.MaxP99LatencyMs,
		TotalInputTokens:  result.TotalInputTokens,
		TotalOutputTokens: result.TotalOutputTokens,
		TotalCost:         result.TotalCost,
		TotalRevenue:      result.TotalRevenue,
		TotalProfit:       result.TotalProfit,
		ProfitMargin:      profitMargin,
	}, nil
}

// GetModelAggregatedMetrics 获取模型级别的聚合指标
func (r *providerMetricsRepository) GetModelAggregatedMetrics(ctx context.Context, modelID string, startTime, endTime time.Time) (*model.AggregatedMetrics, error) {
	var result struct {
		TotalRequests     int64           `gorm:"column:total_requests"`
		TotalSuccess      int64           `gorm:"column:total_success"`
		TotalFailure      int64           `gorm:"column:total_failure"`
		TotalTimeout      int64           `gorm:"column:total_timeout"`
		AvgLatencyMs      float64         `gorm:"column:avg_latency_ms"`
		MaxP99LatencyMs   int             `gorm:"column:max_p99_latency_ms"`
		TotalInputTokens  int64           `gorm:"column:total_input_tokens"`
		TotalOutputTokens int64           `gorm:"column:total_output_tokens"`
		TotalCost         decimal.Decimal `gorm:"column:total_cost"`
		TotalRevenue      decimal.Decimal `gorm:"column:total_revenue"`
		TotalProfit       decimal.Decimal `gorm:"column:total_profit"`
	}

	err := r.db.WithContext(ctx).
		Model(&model.ProviderMetrics{}).
		Joins("JOIN model_providers ON model_providers.id = provider_metrics.provider_id").
		Select(`
			COALESCE(SUM(provider_metrics.request_count), 0) as total_requests,
			COALESCE(SUM(provider_metrics.success_count), 0) as total_success,
			COALESCE(SUM(provider_metrics.failure_count), 0) as total_failure,
			COALESCE(SUM(provider_metrics.timeout_count), 0) as total_timeout,
			COALESCE(AVG(provider_metrics.avg_latency_ms), 0) as avg_latency_ms,
			COALESCE(MAX(provider_metrics.p99_latency_ms), 0) as max_p99_latency_ms,
			COALESCE(SUM(provider_metrics.input_tokens), 0) as total_input_tokens,
			COALESCE(SUM(provider_metrics.output_tokens), 0) as total_output_tokens,
			COALESCE(SUM(provider_metrics.actual_cost), 0) as total_cost,
			COALESCE(SUM(provider_metrics.revenue), 0) as total_revenue,
			COALESCE(SUM(provider_metrics.profit), 0) as total_profit
		`).
		Where("model_providers.model_id = ?", modelID).
		Where("provider_metrics.metric_time >= ? AND provider_metrics.metric_time <= ?", startTime, endTime).
		Scan(&result).Error

	if err != nil {
		return nil, err
	}

	successRate := float64(0)
	if result.TotalRequests > 0 {
		successRate = float64(result.TotalSuccess) / float64(result.TotalRequests) * 100
	}

	profitMargin := float64(0)
	if !result.TotalRevenue.IsZero() {
		profitMargin = result.TotalProfit.Div(result.TotalRevenue).InexactFloat64() * 100
	}

	return &model.AggregatedMetrics{
		TotalRequests:     result.TotalRequests,
		TotalSuccess:      result.TotalSuccess,
		TotalFailure:      result.TotalFailure,
		TotalTimeout:      result.TotalTimeout,
		SuccessRate:       successRate,
		AvgLatencyMs:      result.AvgLatencyMs,
		P99LatencyMs:      result.MaxP99LatencyMs,
		TotalInputTokens:  result.TotalInputTokens,
		TotalOutputTokens: result.TotalOutputTokens,
		TotalCost:         result.TotalCost,
		TotalRevenue:      result.TotalRevenue,
		TotalProfit:       result.TotalProfit,
		ProfitMargin:      profitMargin,
	}, nil
}

// GetTimeSeries 获取时间序列数据
func (r *providerMetricsRepository) GetTimeSeries(ctx context.Context, providerID uint, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.TimeSeriesMetric, error) {
	var results []struct {
		MetricTime   time.Time `gorm:"column:metric_time"`
		RequestCount int64     `gorm:"column:request_count"`
		SuccessCount int64     `gorm:"column:success_count"`
		AvgLatencyMs float64   `gorm:"column:avg_latency_ms"`
	}

	err := r.db.WithContext(ctx).
		Model(&model.ProviderMetrics{}).
		Select("metric_time, request_count, success_count, avg_latency_ms").
		Where("provider_id = ?", providerID).
		Where("granularity = ?", granularity).
		Where("metric_time >= ? AND metric_time <= ?", startTime, endTime).
		Order("metric_time ASC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	timeSeries := make([]*model.TimeSeriesMetric, len(results))
	for i, r := range results {
		successRate := float64(0)
		if r.RequestCount > 0 {
			successRate = float64(r.SuccessCount) / float64(r.RequestCount) * 100
		}
		timeSeries[i] = &model.TimeSeriesMetric{
			Time:         r.MetricTime,
			RequestCount: r.RequestCount,
			SuccessRate:  successRate,
			AvgLatencyMs: r.AvgLatencyMs,
		}
	}

	return timeSeries, nil
}

// GetTrafficDistribution 获取流量分布
func (r *providerMetricsRepository) GetTrafficDistribution(ctx context.Context, modelID string, startTime, endTime time.Time) ([]*model.ProviderTrafficDistribution, error) {
	var results []struct {
		ProviderID   uint   `gorm:"column:provider_id"`
		ProviderName string `gorm:"column:provider_name"`
		RequestCount int64  `gorm:"column:request_count"`
	}

	err := r.db.WithContext(ctx).
		Model(&model.ProviderMetrics{}).
		Joins("JOIN model_providers ON model_providers.id = provider_metrics.provider_id").
		Joins("JOIN channels ON channels.id = model_providers.channel_id").
		Select(`
			provider_metrics.provider_id,
			channels.name as provider_name,
			COALESCE(SUM(provider_metrics.request_count), 0) as request_count
		`).
		Where("model_providers.model_id = ?", modelID).
		Where("provider_metrics.metric_time >= ? AND provider_metrics.metric_time <= ?", startTime, endTime).
		Group("provider_metrics.provider_id, channels.name").
		Order("request_count DESC").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	// 计算总请求数
	var totalRequests int64
	for _, r := range results {
		totalRequests += r.RequestCount
	}

	distribution := make([]*model.ProviderTrafficDistribution, len(results))
	for i, r := range results {
		percentage := float64(0)
		if totalRequests > 0 {
			percentage = float64(r.RequestCount) / float64(totalRequests) * 100
		}
		distribution[i] = &model.ProviderTrafficDistribution{
			ProviderID:   r.ProviderID,
			ProviderName: r.ProviderName,
			RequestCount: r.RequestCount,
			Percentage:   percentage,
		}
	}

	return distribution, nil
}

// UpsertMetrics 更新或插入指标
func (r *providerMetricsRepository) UpsertMetrics(ctx context.Context, providerID uint, metricTime time.Time, granularity model.MetricGranularity, updates map[string]interface{}) error {
	// 先尝试查找现有记录
	var existing model.ProviderMetrics
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND metric_time = ? AND granularity = ?", providerID, metricTime, granularity).
		First(&existing).Error

	if err == gorm.ErrRecordNotFound {
		// 创建新记录
		metrics := &model.ProviderMetrics{
			ProviderID:  providerID,
			MetricTime:  metricTime,
			Granularity: granularity,
		}
		if err := r.db.WithContext(ctx).Create(metrics).Error; err != nil {
			return err
		}
		if len(updates) == 0 {
			return nil
		}
		return r.db.WithContext(ctx).Model(metrics).Updates(updates).Error
	} else if err != nil {
		return err
	}

	// 更新现有记录
	return r.db.WithContext(ctx).
		Model(&existing).
		Updates(updates).Error
}

// DeleteOldMetrics 删除旧指标
func (r *providerMetricsRepository) DeleteOldMetrics(ctx context.Context, before time.Time, granularity model.MetricGranularity) error {
	return r.db.WithContext(ctx).
		Where("metric_time < ? AND granularity = ?", before, granularity).
		Delete(&model.ProviderMetrics{}).Error
}

// CreateCircuitEvent 创建熔断事件
func (r *providerMetricsRepository) CreateCircuitEvent(ctx context.Context, event *model.CircuitEventRecord) error {
	return r.db.WithContext(ctx).Create(event).Error
}

// GetCircuitEvents 获取熔断事件
func (r *providerMetricsRepository) GetCircuitEvents(ctx context.Context, providerID uint, startTime, endTime time.Time) ([]*model.CircuitEventRecord, error) {
	var events []*model.CircuitEventRecord
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Model").
		Preload("Provider.Channel").
		Where("provider_id = ?", providerID).
		Where("created_at >= ? AND created_at <= ?", startTime, endTime).
		Order("created_at DESC").
		Find(&events).Error
	return events, err
}

// GetRecentCircuitEvents 获取最近的熔断事件
func (r *providerMetricsRepository) GetRecentCircuitEvents(ctx context.Context, limit int) ([]*model.CircuitEventRecord, error) {
	var events []*model.CircuitEventRecord
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Model").
		Preload("Provider.Channel").
		Order("created_at DESC").
		Limit(limit).
		Find(&events).Error
	return events, err
}
