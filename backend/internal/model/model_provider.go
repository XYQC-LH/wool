package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// CircuitState 熔断器状态
type CircuitState string

const (
	CircuitStateClosed   CircuitState = "closed"    // 关闭状态（正常）
	CircuitStateOpen     CircuitState = "open"      // 打开状态（熔断中）
	CircuitStateHalfOpen CircuitState = "half_open" // 半开状态（探测中）
)

// ProviderStatus 源头状态
type ProviderStatus string

const (
	ProviderStatusActive   ProviderStatus = "active"
	ProviderStatusDisabled ProviderStatus = "disabled"
	ProviderStatusCooling  ProviderStatus = "cooling"
)

// ModelProviderStatus 模型源头状态（兼容旧代码）
type ModelProviderStatus string

const (
	ModelProviderStatusActive      ModelProviderStatus = "active"
	ModelProviderStatusDisabled    ModelProviderStatus = "disabled"
	ModelProviderStatusCooling     ModelProviderStatus = "cooling"
	ModelProviderStatusCircuitOpen ModelProviderStatus = "circuit_open"
)

// ModelProvider 模型源头 - 模型与渠道的多对多关系，包含成本配置和熔断状态
type ModelProvider struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	Operation string `gorm:"type:varchar(50);not null;default:'chat.completions';uniqueIndex:idx_operation_model_channel,priority:1" json:"operation"`
	ModelID   string `gorm:"type:varchar(100);not null;uniqueIndex:idx_operation_model_channel,priority:2;index" json:"model_id"`
	ChannelID uint   `gorm:"not null;uniqueIndex:idx_operation_model_channel,priority:3;index" json:"channel_id"`

	// ⭐ 成本配置（核心字段）
	ActualCostPer1kInput  decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"actual_cost_per_1k_input"`
	ActualCostPer1kOutput decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"actual_cost_per_1k_output"`

	// 调度配置
	Priority       int  `gorm:"default:0" json:"priority"`            // 手动优先级（可选覆盖成本排序）
	Weight         int  `gorm:"default:1" json:"weight"`              // 权重（同成本时的负载均衡）
	IsCostPriority bool `gorm:"default:true" json:"is_cost_priority"` // 是否启用成本优先排序

	// 状态管理
	Status       ModelProviderStatus `gorm:"type:varchar(20);default:'active'" json:"status"`
	CircuitState CircuitState        `gorm:"type:varchar(20);default:'closed'" json:"circuit_state"` // 熔断器状态
	HealthScore  decimal.Decimal     `gorm:"type:decimal(5,2);default:100.00" json:"health_score"`   // 健康分数 0-100

	// 熔断配置
	FailureCount           int        `gorm:"default:0" json:"failure_count"`             // 连续失败次数
	FailureThreshold       int        `gorm:"default:3" json:"failure_threshold"`         // 熔断阈值
	CircuitOpenUntil       *time.Time `json:"circuit_open_until,omitempty"`               // 熔断恢复时间
	RecoveryTimeoutSeconds int        `gorm:"default:30" json:"recovery_timeout_seconds"` // 熔断恢复等待时间
	HalfOpenRequests       int        `gorm:"default:1" json:"half_open_requests"`        // 半开状态允许的探测请求数

	// 超时配置
	ConnectTimeoutMs          int `gorm:"default:2000" json:"connect_timeout_ms"`            // 建连超时
	AttemptTimeoutMs          int `gorm:"default:15000" json:"attempt_timeout_ms"`           // 单次尝试总超时
	StreamFirstChunkTimeoutMs int `gorm:"default:3000" json:"stream_first_chunk_timeout_ms"` // stream 首包门控超时

	// ⚠️ 统计数据 - 这些字段用于历史查询和报表，运行时数据应从 Redis 读取
	// 高频变化的运行时状态（熔断计数、健康分数、in-flight 等）应存储在 Redis 中
	// 避免高并发下数据库热点行竞争
	TotalRequests   int64           `gorm:"default:0" json:"total_requests"`                // 总请求数（历史累计）
	SuccessRequests int64           `gorm:"default:0" json:"success_requests"`              // 成功请求数（历史累计）
	FailedRequests  int64           `gorm:"default:0" json:"failed_requests"`               // 失败请求数（历史累计）
	TotalLatency    int64           `gorm:"default:0" json:"total_latency"`                 // 总延迟（毫秒，历史累计）
	TotalTokensUsed int64           `gorm:"default:0" json:"total_tokens_used"`             // 总Token使用量（历史累计）
	InputTokens     int64           `gorm:"default:0" json:"input_tokens"`                  // 输入Token数（历史累计）
	OutputTokens    int64           `gorm:"default:0" json:"output_tokens"`                 // 输出Token数（历史累计）
	TotalCost       decimal.Decimal `gorm:"type:decimal(12,4);default:0" json:"total_cost"` // 总成本（历史累计）
	AvgLatencyMs    int             `gorm:"default:0" json:"avg_latency_ms"`                // 平均延迟（历史累计）
	P99LatencyMs    int             `gorm:"default:0" json:"p99_latency_ms"`                // P99延迟（历史累计）

	// ⚠️ 时间戳 - 这些字段用于历史查询，运行时数据应从 Redis 读取
	LastSuccessAt *time.Time `json:"last_success_at,omitempty"`             // 最后成功时间（历史记录）
	LastFailureAt *time.Time `json:"last_failure_at,omitempty"`             // 最后失败时间（历史记录）
	LastUsedAt    *time.Time `json:"last_used_at,omitempty"`                // 最后使用时间（历史记录）
	LastError     string     `gorm:"type:text" json:"last_error,omitempty"` // 最后错误信息（历史记录）

	// 上游模型名称映射
	UpstreamModelName string `gorm:"type:varchar(100);not null" json:"upstream_model_name"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Model            *Model            `gorm:"foreignKey:ModelID" json:"model,omitempty"`
	Channel          *Channel          `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	SelectedInstance *ProviderInstance `gorm:"-" json:"selected_instance,omitempty"` // 运行时选中的实例（不持久化）
}

// ProviderQueryParams 源头查询参数
type ProviderQueryParams struct {
	Operation    string       `form:"operation"`
	ModelID      string       `form:"model_id"`
	ChannelID    uint         `form:"channel_id"`
	Status       string       `form:"status"`
	CircuitState CircuitState `form:"circuit_state"`
	Page         int          `form:"page"`
	PageSize     int          `form:"page_size"`
}

// TableName 表名
func (ModelProvider) TableName() string {
	return "model_providers"
}

// IsAvailable 检查源头是否可用
func (mp *ModelProvider) IsAvailable() bool {
	if mp.Status != ModelProviderStatusActive {
		return false
	}
	// 检查熔断状态
	if mp.CircuitOpenUntil != nil && time.Now().Before(*mp.CircuitOpenUntil) {
		return false
	}
	return true
}

// IsCircuitOpen 检查熔断器是否打开
func (mp *ModelProvider) IsCircuitOpen() bool {
	if mp.CircuitOpenUntil == nil {
		return false
	}
	return time.Now().Before(*mp.CircuitOpenUntil)
}

// ShouldTryHalfOpen 检查是否应该尝试半开状态
func (mp *ModelProvider) ShouldTryHalfOpen() bool {
	if mp.CircuitOpenUntil == nil {
		return false
	}
	return time.Now().After(*mp.CircuitOpenUntil)
}

// GetSuccessRate 获取成功率
func (mp *ModelProvider) GetSuccessRate() float64 {
	if mp.TotalRequests == 0 {
		return 100.0
	}
	return float64(mp.SuccessRequests) / float64(mp.TotalRequests) * 100
}

// ModelProviderResponse 模型源头响应结构
type ModelProviderResponse struct {
	ID                        uint                `json:"id"`
	Operation                 string              `json:"operation"`
	ModelID                   string              `json:"model_id"`
	ModelName                 string              `json:"model_name,omitempty"`
	ChannelID                 uint                `json:"channel_id"`
	ChannelName               string              `json:"channel_name,omitempty"`
	ActualCostPer1kInput      decimal.Decimal     `json:"actual_cost_per_1k_input"`
	ActualCostPer1kOutput     decimal.Decimal     `json:"actual_cost_per_1k_output"`
	Priority                  int                 `json:"priority"`
	Weight                    int                 `json:"weight"`
	IsCostPriority            bool                `json:"is_cost_priority"`
	Status                    ModelProviderStatus `json:"status"`
	CircuitState              CircuitState        `json:"circuit_state"`
	HealthScore               decimal.Decimal     `json:"health_score"`
	FailureCount              int                 `json:"failure_count"`
	FailureThreshold          int                 `json:"failure_threshold"`
	CircuitOpenUntil          *time.Time          `json:"circuit_open_until,omitempty"`
	RecoveryTimeoutSeconds    int                 `json:"recovery_timeout_seconds"`
	ConnectTimeoutMs          int                 `json:"connect_timeout_ms"`
	AttemptTimeoutMs          int                 `json:"attempt_timeout_ms"`
	StreamFirstChunkTimeoutMs int                 `json:"stream_first_chunk_timeout_ms"`
	TotalRequests             int64               `json:"total_requests"`
	SuccessRequests           int64               `json:"success_requests"`
	FailedRequests            int64               `json:"failed_requests"`
	SuccessRate               float64             `json:"success_rate"`
	TotalTokensUsed           int64               `json:"total_tokens_used"`
	TotalCost                 decimal.Decimal     `json:"total_cost"`
	AvgLatencyMs              int                 `json:"avg_latency_ms"`
	P99LatencyMs              int                 `json:"p99_latency_ms"`
	LastSuccessAt             *time.Time          `json:"last_success_at,omitempty"`
	LastFailureAt             *time.Time          `json:"last_failure_at,omitempty"`
	LastError                 string              `json:"last_error,omitempty"`
	UpstreamModelName         string              `json:"upstream_model_name"`
	CreatedAt                 time.Time           `json:"created_at"`
	UpdatedAt                 time.Time           `json:"updated_at"`
}

// ToResponse 转换为响应结构
func (mp *ModelProvider) ToResponse() *ModelProviderResponse {
	resp := &ModelProviderResponse{
		ID:                        mp.ID,
		Operation:                 mp.Operation,
		ModelID:                   mp.ModelID,
		ChannelID:                 mp.ChannelID,
		ActualCostPer1kInput:      mp.ActualCostPer1kInput,
		ActualCostPer1kOutput:     mp.ActualCostPer1kOutput,
		Priority:                  mp.Priority,
		Weight:                    mp.Weight,
		IsCostPriority:            mp.IsCostPriority,
		Status:                    mp.Status,
		CircuitState:              mp.CircuitState,
		HealthScore:               mp.HealthScore,
		FailureCount:              mp.FailureCount,
		FailureThreshold:          mp.FailureThreshold,
		CircuitOpenUntil:          mp.CircuitOpenUntil,
		RecoveryTimeoutSeconds:    mp.RecoveryTimeoutSeconds,
		ConnectTimeoutMs:          mp.ConnectTimeoutMs,
		AttemptTimeoutMs:          mp.AttemptTimeoutMs,
		StreamFirstChunkTimeoutMs: mp.StreamFirstChunkTimeoutMs,
		TotalRequests:             mp.TotalRequests,
		SuccessRequests:           mp.SuccessRequests,
		FailedRequests:            mp.FailedRequests,
		SuccessRate:               mp.GetSuccessRate(),
		TotalTokensUsed:           mp.TotalTokensUsed,
		TotalCost:                 mp.TotalCost,
		AvgLatencyMs:              mp.AvgLatencyMs,
		P99LatencyMs:              mp.P99LatencyMs,
		LastSuccessAt:             mp.LastSuccessAt,
		LastFailureAt:             mp.LastFailureAt,
		LastError:                 mp.LastError,
		UpstreamModelName:         mp.UpstreamModelName,
		CreatedAt:                 mp.CreatedAt,
		UpdatedAt:                 mp.UpdatedAt,
	}

	if mp.Model != nil {
		resp.ModelName = mp.Model.Name
	}
	if mp.Channel != nil {
		resp.ChannelName = mp.Channel.Name
	}

	return resp
}

// CreateModelProviderRequest 创建模型源头请求
type CreateModelProviderRequest struct {
	Operation              string  `json:"operation,omitempty"`
	ModelID                string  `json:"model_id" binding:"required"`
	ChannelID              uint    `json:"channel_id" binding:"required"`
	UpstreamModelName      string  `json:"upstream_model_name" binding:"required,min=1,max=100"`
	ActualCostPer1kInput   float64 `json:"actual_cost_per_1k_input" binding:"required,min=0"`
	ActualCostPer1kOutput  float64 `json:"actual_cost_per_1k_output" binding:"required,min=0"`
	IsCostPriority         *bool   `json:"is_cost_priority,omitempty"`
	Priority               int     `json:"priority" binding:"min=0"`
	Weight                 int     `json:"weight" binding:"min=1"`
	FailureThreshold       int     `json:"failure_threshold" binding:"min=1"`
	RecoveryTimeoutSeconds int     `json:"recovery_timeout_seconds" binding:"min=1"`
	Status                 string  `json:"status,omitempty"`
}

// UpdateModelProviderRequest 更新模型源头请求
type UpdateModelProviderRequest struct {
	Operation              *string              `json:"operation,omitempty"`
	UpstreamModelName      *string              `json:"upstream_model_name,omitempty" binding:"omitempty,min=1,max=100"`
	ActualCostPer1kInput   *float64             `json:"actual_cost_per_1k_input,omitempty" binding:"omitempty,min=0"`
	ActualCostPer1kOutput  *float64             `json:"actual_cost_per_1k_output,omitempty" binding:"omitempty,min=0"`
	IsCostPriority         *bool                `json:"is_cost_priority,omitempty"`
	Priority               *int                 `json:"priority,omitempty" binding:"omitempty,min=0"`
	Weight                 *int                 `json:"weight,omitempty" binding:"omitempty,min=1"`
	FailureThreshold       *int                 `json:"failure_threshold,omitempty" binding:"omitempty,min=1"`
	RecoveryTimeoutSeconds *int                 `json:"recovery_timeout_seconds,omitempty" binding:"omitempty,min=1"`
	Status                 *ModelProviderStatus `json:"status,omitempty"`
}

// ModelProviderStats 模型源头统计
type ModelProviderStats struct {
	TotalProviders   int64   `json:"total_providers"`
	ActiveProviders  int64   `json:"active_providers"`
	CircuitOpenCount int64   `json:"circuit_open_count"`
	AvgSuccessRate   float64 `json:"avg_success_rate"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
}

// DispatchStats 调度统计数据
type DispatchStats struct {
	TotalRequests        int64                    `json:"total_requests"`
	SuccessRate          float64                  `json:"success_rate"`
	AvgLatencyMs         float64                  `json:"avg_latency_ms"`
	ActiveProviders      int64                    `json:"active_providers"`
	CircuitOpenProviders int64                    `json:"circuit_open_providers"`
	Providers            []*ModelProviderResponse `json:"providers"`
}

// DispatchMetrics 调度指标（用于图表）
type DispatchMetrics struct {
	Time         time.Time                  `json:"time"`
	RequestCount int64                      `json:"request_count"`
	SuccessRate  float64                    `json:"success_rate"`
	AvgLatencyMs float64                    `json:"avg_latency_ms"`
	Providers    map[string]*ProviderMetric `json:"providers"`
}

// ProviderMetric 单个源头的指标
type ProviderMetric struct {
	Requests    int64   `json:"requests"`
	SuccessRate float64 `json:"success_rate"`
}

// CircuitEvent 熔断事件
type CircuitEvent struct {
	ID           uint      `json:"id"`
	ProviderID   uint      `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	EventType    string    `json:"event_type"` // open, close, half_open
	Reason       string    `json:"reason"`
	Duration     int       `json:"duration,omitempty"` // 熔断时长（秒）
	CreatedAt    time.Time `json:"created_at"`
}
