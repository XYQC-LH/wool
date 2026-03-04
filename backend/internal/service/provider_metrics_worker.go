package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"strings"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

type ProviderMetricsWorkerConfig struct {
	PollInterval time.Duration
	Lookback     time.Duration
	Granularity  model.MetricGranularity
}

func DefaultProviderMetricsWorkerConfig() ProviderMetricsWorkerConfig {
	return ProviderMetricsWorkerConfig{
		PollInterval: 1 * time.Minute,
		Lookback:     10 * time.Minute,
		Granularity:  model.MetricGranularityMinute,
	}
}

// ProviderMetricsWorker 用于把 logs 聚合写入 provider_metrics（低频写，用于展示/分析）。
//
// 设计说明（KISS）：
// - 当前只做按 provider 聚合；provider=operation+model_id+channel_id 的组合（见 model_providers）。
// - operation 从 logs.metadata.operation 读取；为空时按 chat.completions 兼容旧数据。
type ProviderMetricsWorker struct {
	db          *gorm.DB
	metricsRepo repository.ProviderMetricsRepository
	cfg         ProviderMetricsWorkerConfig
}

func NewProviderMetricsWorker(db *gorm.DB, metricsRepo repository.ProviderMetricsRepository, cfg ProviderMetricsWorkerConfig) (*ProviderMetricsWorker, error) {
	if db == nil {
		return nil, fmt.Errorf("db 不能为空")
	}
	if metricsRepo == nil {
		return nil, fmt.Errorf("metricsRepo 不能为空")
	}

	if cfg.PollInterval <= 0 {
		cfg.PollInterval = DefaultProviderMetricsWorkerConfig().PollInterval
	}
	if cfg.Lookback <= 0 {
		cfg.Lookback = DefaultProviderMetricsWorkerConfig().Lookback
	}
	if cfg.Granularity == "" {
		cfg.Granularity = DefaultProviderMetricsWorkerConfig().Granularity
	}

	return &ProviderMetricsWorker{
		db:          db,
		metricsRepo: metricsRepo,
		cfg:         cfg,
	}, nil
}

func (w *ProviderMetricsWorker) Run(ctx context.Context) error {
	if ctx == nil {
		ctx = context.Background()
	}

	log.Printf("[metrics-worker] started poll=%s lookback=%s granularity=%s", w.cfg.PollInterval, w.cfg.Lookback, w.cfg.Granularity)

	ticker := time.NewTicker(w.cfg.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			log.Printf("[metrics-worker] stopping: %v", ctx.Err())
			return nil
		case <-ticker.C:
			if err := w.runOnce(ctx); err != nil {
				log.Printf("[metrics-worker] runOnce error: %v", err)
			}
		}
	}
}

func (w *ProviderMetricsWorker) runOnce(ctx context.Context) error {
	if w == nil || w.db == nil || w.metricsRepo == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	end := time.Now().Truncate(time.Minute)
	start := end.Add(-w.cfg.Lookback).Truncate(time.Minute)
	if !start.Before(end) {
		return nil
	}

	rows, err := w.aggregateProviderMetrics(ctx, w.cfg.Granularity, start, end)
	if err != nil {
		return err
	}

	for _, row := range rows {
		updates := map[string]interface{}{
			"request_count":  int(row.RequestCount),
			"success_count":  int(row.SuccessCount),
			"failure_count":  int(row.FailureCount),
			"timeout_count":  int(row.TimeoutCount),
			"avg_latency_ms": int(math.Round(row.AvgLatencyMs)),
			"min_latency_ms": int(row.MinLatencyMs),
			"max_latency_ms": int(row.MaxLatencyMs),
			"p50_latency_ms": int(math.Round(row.P50LatencyMs)),
			"p95_latency_ms": int(math.Round(row.P95LatencyMs)),
			"p99_latency_ms": int(math.Round(row.P99LatencyMs)),
			"input_tokens":   row.InputTokens,
			"output_tokens":  row.OutputTokens,
			"actual_cost":    row.ActualCost,
			"revenue":        row.Revenue,
			"profit":         row.Profit,
		}

		if err := w.metricsRepo.UpsertMetrics(ctx, row.ProviderID, row.MetricTime, w.cfg.Granularity, updates); err != nil {
			return err
		}
	}

	return nil
}

type providerMetricsAggregateRow struct {
	ProviderID   uint            `gorm:"column:provider_id"`
	MetricTime   time.Time       `gorm:"column:metric_time"`
	RequestCount int64           `gorm:"column:request_count"`
	SuccessCount int64           `gorm:"column:success_count"`
	FailureCount int64           `gorm:"column:failure_count"`
	TimeoutCount int64           `gorm:"column:timeout_count"`
	AvgLatencyMs float64         `gorm:"column:avg_latency_ms"`
	MinLatencyMs int64           `gorm:"column:min_latency_ms"`
	MaxLatencyMs int64           `gorm:"column:max_latency_ms"`
	P50LatencyMs float64         `gorm:"column:p50_latency_ms"`
	P95LatencyMs float64         `gorm:"column:p95_latency_ms"`
	P99LatencyMs float64         `gorm:"column:p99_latency_ms"`
	InputTokens  int64           `gorm:"column:input_tokens"`
	OutputTokens int64           `gorm:"column:output_tokens"`
	ActualCost   decimal.Decimal `gorm:"column:actual_cost"`
	Revenue      decimal.Decimal `gorm:"column:revenue"`
	Profit       decimal.Decimal `gorm:"column:profit"`
}

func (w *ProviderMetricsWorker) aggregateProviderMetrics(ctx context.Context, granularity model.MetricGranularity, start, end time.Time) ([]providerMetricsAggregateRow, error) {
	if w == nil || w.db == nil {
		return nil, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if !start.Before(end) {
		return []providerMetricsAggregateRow{}, nil
	}

	unit := "minute"
	switch granularity {
	case model.MetricGranularityMinute:
		unit = "minute"
	case model.MetricGranularityHour:
		unit = "hour"
	case model.MetricGranularityDay:
		unit = "day"
	default:
		unit = "minute"
	}

	// 兼容旧日志：未写入 metadata.operation 的场景，默认认为是 chat.completions。
	defaultOperation := model.OperationChatCompletions
	if strings.TrimSpace(defaultOperation) == "" {
		defaultOperation = "chat.completions"
	}

	selectSQL := fmt.Sprintf(`
		model_providers.id as provider_id,
		DATE_TRUNC('%s', logs.created_at) as metric_time,
		COUNT(*) as request_count,
		COALESCE(SUM(CASE WHEN logs.status = 'success' THEN 1 ELSE 0 END), 0) as success_count,
		COALESCE(SUM(CASE WHEN logs.status <> 'success' THEN 1 ELSE 0 END), 0) as failure_count,
		COALESCE(SUM(CASE WHEN logs.status <> 'success' AND (
			COALESCE(logs.status_code, 0) IN (408, 504) OR
			logs.error_message ILIKE '%%timeout%%' OR
			logs.error_message ILIKE '%%超时%%'
		) THEN 1 ELSE 0 END), 0) as timeout_count,
		COALESCE(AVG(logs.duration), 0) as avg_latency_ms,
		COALESCE(MIN(logs.duration), 0) as min_latency_ms,
		COALESCE(MAX(logs.duration), 0) as max_latency_ms,
		COALESCE(PERCENTILE_CONT(0.5) WITHIN GROUP (ORDER BY logs.duration), 0) as p50_latency_ms,
		COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY logs.duration), 0) as p95_latency_ms,
		COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY logs.duration), 0) as p99_latency_ms,
		COALESCE(SUM(logs.prompt_tokens), 0) as input_tokens,
		COALESCE(SUM(logs.completion_tokens), 0) as output_tokens,
		COALESCE(SUM(logs.upstream_cost), 0) as actual_cost,
		COALESCE(SUM(logs.total_cost), 0) as revenue,
		COALESCE(SUM(logs.total_cost - logs.upstream_cost), 0) as profit
	`, unit)

	joinSQL := fmt.Sprintf(`
		JOIN model_providers
		  ON model_providers.channel_id = logs.channel_id
		 AND model_providers.model_id = logs.model
		 AND model_providers.operation = COALESCE(NULLIF(logs.metadata->>'operation',''), '%s')
	`, defaultOperation)

	var rows []providerMetricsAggregateRow
	err := w.db.WithContext(ctx).
		Table("logs").
		Select(selectSQL).
		Joins(joinSQL).
		Where("logs.channel_id <> 0").
		Where("logs.model <> ''").
		Where("logs.created_at >= ? AND logs.created_at < ?", start, end).
		Group(fmt.Sprintf("model_providers.id, DATE_TRUNC('%s', logs.created_at)", unit)).
		Order("metric_time ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}
