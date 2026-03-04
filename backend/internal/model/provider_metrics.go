package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// MetricGranularity 指标粒度
type MetricGranularity string

const (
	MetricGranularityMinute MetricGranularity = "minute"
	MetricGranularityHour   MetricGranularity = "hour"
	MetricGranularityDay    MetricGranularity = "day"
)

// ProviderMetrics 源头指标表 - 记录每个源头的详细调度指标
type ProviderMetrics struct {
	ID         uint `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID uint `gorm:"not null;index" json:"provider_id"`

	// 时间维度
	MetricTime  time.Time         `gorm:"not null;index" json:"metric_time"` // 指标时间（按分钟/小时聚合）
	Granularity MetricGranularity `gorm:"type:varchar(10);not null" json:"granularity"`

	// 请求指标
	RequestCount int `gorm:"default:0" json:"request_count"` // 请求数
	SuccessCount int `gorm:"default:0" json:"success_count"` // 成功数
	FailureCount int `gorm:"default:0" json:"failure_count"` // 失败数
	TimeoutCount int `gorm:"default:0" json:"timeout_count"` // 超时数

	// 延迟指标
	AvgLatencyMs int `gorm:"default:0" json:"avg_latency_ms"` // 平均延迟
	MinLatencyMs int `gorm:"default:0" json:"min_latency_ms"` // 最小延迟
	MaxLatencyMs int `gorm:"default:0" json:"max_latency_ms"` // 最大延迟
	P50LatencyMs int `gorm:"default:0" json:"p50_latency_ms"` // P50延迟
	P95LatencyMs int `gorm:"default:0" json:"p95_latency_ms"` // P95延迟
	P99LatencyMs int `gorm:"default:0" json:"p99_latency_ms"` // P99延迟

	// Token指标
	InputTokens  int64 `gorm:"default:0" json:"input_tokens"`  // 输入Token数
	OutputTokens int64 `gorm:"default:0" json:"output_tokens"` // 输出Token数

	// 成本指标
	ActualCost decimal.Decimal `gorm:"type:decimal(12,6);default:0" json:"actual_cost"` // 实际成本
	Revenue    decimal.Decimal `gorm:"type:decimal(12,6);default:0" json:"revenue"`     // 收入
	Profit     decimal.Decimal `gorm:"type:decimal(12,6);default:0" json:"profit"`      // 利润

	// 熔断指标
	CircuitOpenCount           int `gorm:"default:0" json:"circuit_open_count"`            // 熔断次数
	CircuitOpenDurationSeconds int `gorm:"default:0" json:"circuit_open_duration_seconds"` // 熔断总时长

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Provider *ModelProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

// TableName 表名
func (ProviderMetrics) TableName() string {
	return "provider_metrics"
}

// GetSuccessRate 获取成功率
func (pm *ProviderMetrics) GetSuccessRate() float64 {
	if pm.RequestCount == 0 {
		return 100.0
	}
	return float64(pm.SuccessCount) / float64(pm.RequestCount) * 100
}

// GetFailureRate 获取失败率
func (pm *ProviderMetrics) GetFailureRate() float64 {
	if pm.RequestCount == 0 {
		return 0.0
	}
	return float64(pm.FailureCount) / float64(pm.RequestCount) * 100
}

// GetTimeoutRate 获取超时率
func (pm *ProviderMetrics) GetTimeoutRate() float64 {
	if pm.RequestCount == 0 {
		return 0.0
	}
	return float64(pm.TimeoutCount) / float64(pm.RequestCount) * 100
}

// ProviderMetricsResponse 源头指标响应结构
type ProviderMetricsResponse struct {
	ID                         uint              `json:"id"`
	ProviderID                 uint              `json:"provider_id"`
	ProviderName               string            `json:"provider_name,omitempty"`
	MetricTime                 time.Time         `json:"metric_time"`
	Granularity                MetricGranularity `json:"granularity"`
	RequestCount               int               `json:"request_count"`
	SuccessCount               int               `json:"success_count"`
	FailureCount               int               `json:"failure_count"`
	TimeoutCount               int               `json:"timeout_count"`
	SuccessRate                float64           `json:"success_rate"`
	FailureRate                float64           `json:"failure_rate"`
	TimeoutRate                float64           `json:"timeout_rate"`
	AvgLatencyMs               int               `json:"avg_latency_ms"`
	MinLatencyMs               int               `json:"min_latency_ms"`
	MaxLatencyMs               int               `json:"max_latency_ms"`
	P50LatencyMs               int               `json:"p50_latency_ms"`
	P95LatencyMs               int               `json:"p95_latency_ms"`
	P99LatencyMs               int               `json:"p99_latency_ms"`
	InputTokens                int64             `json:"input_tokens"`
	OutputTokens               int64             `json:"output_tokens"`
	TotalTokens                int64             `json:"total_tokens"`
	ActualCost                 decimal.Decimal   `json:"actual_cost"`
	Revenue                    decimal.Decimal   `json:"revenue"`
	Profit                     decimal.Decimal   `json:"profit"`
	CircuitOpenCount           int               `json:"circuit_open_count"`
	CircuitOpenDurationSeconds int               `json:"circuit_open_duration_seconds"`
	CreatedAt                  time.Time         `json:"created_at"`
}

// ToResponse 转换为响应结构
func (pm *ProviderMetrics) ToResponse() *ProviderMetricsResponse {
	resp := &ProviderMetricsResponse{
		ID:                         pm.ID,
		ProviderID:                 pm.ProviderID,
		MetricTime:                 pm.MetricTime,
		Granularity:                pm.Granularity,
		RequestCount:               pm.RequestCount,
		SuccessCount:               pm.SuccessCount,
		FailureCount:               pm.FailureCount,
		TimeoutCount:               pm.TimeoutCount,
		SuccessRate:                pm.GetSuccessRate(),
		FailureRate:                pm.GetFailureRate(),
		TimeoutRate:                pm.GetTimeoutRate(),
		AvgLatencyMs:               pm.AvgLatencyMs,
		MinLatencyMs:               pm.MinLatencyMs,
		MaxLatencyMs:               pm.MaxLatencyMs,
		P50LatencyMs:               pm.P50LatencyMs,
		P95LatencyMs:               pm.P95LatencyMs,
		P99LatencyMs:               pm.P99LatencyMs,
		InputTokens:                pm.InputTokens,
		OutputTokens:               pm.OutputTokens,
		TotalTokens:                pm.InputTokens + pm.OutputTokens,
		ActualCost:                 pm.ActualCost,
		Revenue:                    pm.Revenue,
		Profit:                     pm.Profit,
		CircuitOpenCount:           pm.CircuitOpenCount,
		CircuitOpenDurationSeconds: pm.CircuitOpenDurationSeconds,
		CreatedAt:                  pm.CreatedAt,
	}

	if pm.Provider != nil && pm.Provider.Channel != nil {
		resp.ProviderName = pm.Provider.Channel.Name
	}

	return resp
}

// MetricsQueryParams 指标查询参数
type MetricsQueryParams struct {
	ProviderID  uint              `form:"provider_id"`
	ModelID     string            `form:"model_id"`
	Granularity MetricGranularity `form:"granularity"`
	StartTime   *time.Time        `form:"start_time"`
	EndTime     *time.Time        `form:"end_time"`
	Page        int               `form:"page"`
	PageSize    int               `form:"page_size"`
}

// AggregatedMetrics 聚合指标
type AggregatedMetrics struct {
	TotalRequests     int64           `json:"total_requests"`
	TotalSuccess      int64           `json:"total_success"`
	TotalFailure      int64           `json:"total_failure"`
	TotalTimeout      int64           `json:"total_timeout"`
	SuccessRate       float64         `json:"success_rate"`
	AvgLatencyMs      float64         `json:"avg_latency_ms"`
	P99LatencyMs      int             `json:"p99_latency_ms"`
	TotalInputTokens  int64           `json:"total_input_tokens"`
	TotalOutputTokens int64           `json:"total_output_tokens"`
	TotalCost         decimal.Decimal `json:"total_cost"`
	TotalRevenue      decimal.Decimal `json:"total_revenue"`
	TotalProfit       decimal.Decimal `json:"total_profit"`
	ProfitMargin      float64         `json:"profit_margin"`
}

// TimeSeriesMetric 时间序列指标
type TimeSeriesMetric struct {
	Time         time.Time `json:"time"`
	RequestCount int64     `json:"request_count"`
	SuccessRate  float64   `json:"success_rate"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
}

// ProviderTrafficDistribution 源头流量分布
type ProviderTrafficDistribution struct {
	ProviderID   uint    `json:"provider_id"`
	ProviderName string  `json:"provider_name"`
	RequestCount int64   `json:"request_count"`
	Percentage   float64 `json:"percentage"`
}

// CircuitEventRecord 熔断事件记录
type CircuitEventRecord struct {
	ID               uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID       uint      `gorm:"not null;index" json:"provider_id"`
	EventType        string    `gorm:"type:varchar(20);not null" json:"event_type"` // open, close, half_open
	Reason           string    `gorm:"type:text" json:"reason"`
	FailureCount     int       `gorm:"default:0" json:"failure_count"`
	RecoveryAttempts int       `gorm:"default:0" json:"recovery_attempts"`
	DurationSeconds  int       `gorm:"default:0" json:"duration_seconds"`
	CreatedAt        time.Time `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	Provider *ModelProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

// TableName 表名
func (CircuitEventRecord) TableName() string {
	return "circuit_event_records"
}

// CircuitEventRecordResponse 熔断事件记录响应
type CircuitEventRecordResponse struct {
	ID               uint      `json:"id"`
	ProviderID       uint      `json:"provider_id"`
	ProviderName     string    `json:"provider_name,omitempty"`
	ModelName        string    `json:"model_name,omitempty"`
	ChannelName      string    `json:"channel_name,omitempty"`
	EventType        string    `json:"event_type"`
	Reason           string    `json:"reason"`
	FailureCount     int       `json:"failure_count"`
	RecoveryAttempts int       `json:"recovery_attempts"`
	DurationSeconds  int       `json:"duration_seconds"`
	CreatedAt        time.Time `json:"created_at"`
}

// ToResponse 转换为响应结构
func (cer *CircuitEventRecord) ToResponse() *CircuitEventRecordResponse {
	resp := &CircuitEventRecordResponse{
		ID:               cer.ID,
		ProviderID:       cer.ProviderID,
		EventType:        cer.EventType,
		Reason:           cer.Reason,
		FailureCount:     cer.FailureCount,
		RecoveryAttempts: cer.RecoveryAttempts,
		DurationSeconds:  cer.DurationSeconds,
		CreatedAt:        cer.CreatedAt,
	}

	if cer.Provider != nil {
		if cer.Provider.Model != nil {
			resp.ModelName = cer.Provider.Model.Name
		}
		if cer.Provider.Channel != nil {
			resp.ChannelName = cer.Provider.Channel.Name
			resp.ProviderName = cer.Provider.Channel.Name
		}
	}

	return resp
}
