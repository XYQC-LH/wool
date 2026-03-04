package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// LogStatus 日志状态
type LogStatus string

const (
	LogStatusSuccess LogStatus = "success"
	LogStatusError   LogStatus = "error"
	LogStatusPending LogStatus = "pending"
)

// Log 请求日志模型
type Log struct {
	ID               uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID           uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	TokenID          uuid.UUID       `gorm:"type:uuid;index" json:"token_id"`
	TokenKey         *string         `gorm:"type:varchar(64);index" json:"token_key,omitempty"`
	ChannelID        uint            `gorm:"index" json:"channel_id"`
	Model            string          `gorm:"type:varchar(100);not null;index" json:"model"`
	PromptTokens     int             `gorm:"default:0" json:"prompt_tokens"`
	CompletionTokens int             `gorm:"default:0" json:"completion_tokens"`
	TotalCost        decimal.Decimal `gorm:"type:decimal(12,6);default:0" json:"total_cost"`
	UpstreamCost     decimal.Decimal `gorm:"type:decimal(12,6);default:0" json:"upstream_cost"`
	Duration         int             `gorm:"default:0" json:"duration"`
	DurationMs       *int            `json:"duration_ms,omitempty"`
	StatusCode       *int            `json:"status_code,omitempty"`
	Status           LogStatus       `gorm:"type:varchar(20);default:'success'" json:"status"`
	IsStream         bool            `gorm:"default:false" json:"is_stream"`
	ErrorMessage     string          `gorm:"type:text" json:"error_message,omitempty"`
	RequestIP        *string         `gorm:"type:varchar(45)" json:"request_ip,omitempty"`
	UserAgent        *string         `gorm:"type:text" json:"user_agent,omitempty"`
	Metadata         JSON            `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt        time.Time       `gorm:"autoCreateTime;index" json:"created_at"`

	// 关联
	User    *User    `gorm:"foreignKey:UserID" json:"user,omitempty"`
	Channel *Channel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
}

// TableName 表名
func (Log) TableName() string {
	return "logs"
}

// BeforeCreate 创建前钩子
func (l *Log) BeforeCreate(tx *gorm.DB) error {
	if l.ID == uuid.Nil {
		l.ID = uuid.New()
	}
	return nil
}

// TotalTokens 计算总 Token 数
func (l *Log) TotalTokens() int {
	return l.PromptTokens + l.CompletionTokens
}

// Profit 计算利润
func (l *Log) Profit() decimal.Decimal {
	return l.TotalCost.Sub(l.UpstreamCost)
}

// LogResponse 日志响应结构
type LogResponse struct {
	ID               uuid.UUID       `json:"id"`
	TokenKey         *string         `json:"token_key,omitempty"`
	Model            string          `json:"model"`
	PromptTokens     int             `json:"prompt_tokens"`
	CompletionTokens int             `json:"completion_tokens"`
	TotalTokens      int             `json:"total_tokens"`
	TotalCost        decimal.Decimal `json:"total_cost"`
	Duration         int             `json:"duration"`
	DurationMs       *int            `json:"duration_ms,omitempty"`
	StatusCode       *int            `json:"status_code,omitempty"`
	Status           LogStatus       `json:"status"`
	IsStream         bool            `json:"is_stream"`
	ErrorMessage     string          `json:"error_message,omitempty"`
	CreatedAt        time.Time       `json:"created_at"`
}

// ToResponse 转换为响应结构
func (l *Log) ToResponse() *LogResponse {
	var maskedKey *string
	if l.TokenKey != nil && len(*l.TokenKey) > 12 {
		masked := (*l.TokenKey)[:8] + "..." + (*l.TokenKey)[len(*l.TokenKey)-4:]
		maskedKey = &masked
	}

	return &LogResponse{
		ID:               l.ID,
		TokenKey:         maskedKey,
		Model:            l.Model,
		PromptTokens:     l.PromptTokens,
		CompletionTokens: l.CompletionTokens,
		TotalTokens:      l.TotalTokens(),
		TotalCost:        l.TotalCost,
		Duration:         l.Duration,
		DurationMs:       l.DurationMs,
		StatusCode:       l.StatusCode,
		Status:           l.Status,
		IsStream:         l.IsStream,
		ErrorMessage:     l.ErrorMessage,
		CreatedAt:        l.CreatedAt,
	}
}

// AdminLogResponse 管理员日志响应结构
type AdminLogResponse struct {
	ID               uuid.UUID `json:"id"`
	UserID           uuid.UUID `json:"user_id"`
	Username         string    `json:"username,omitempty"`
	TokenKey         *string   `json:"token_key,omitempty"`
	ChannelID        uint      `json:"channel_id"`
	ChannelName      string    `json:"channel_name,omitempty"`
	Model            string    `json:"model"`
	PromptTokens     int       `json:"prompt_tokens"`
	CompletionTokens int       `json:"completion_tokens"`
	TotalTokens      int       `json:"total_tokens"`
	TotalCost        float64   `json:"total_cost"`
	UpstreamCost     float64   `json:"upstream_cost"`
	Profit           float64   `json:"profit"`
	Duration         int       `json:"duration"`
	DurationMs       *int      `json:"duration_ms,omitempty"`
	StatusCode       *int      `json:"status_code,omitempty"`
	Status           LogStatus `json:"status"`
	IsStream         bool      `json:"is_stream"`
	ErrorMessage     string    `json:"error_message,omitempty"`
	RequestIP        *string   `json:"request_ip,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
}

// ToAdminResponse 转换为管理员响应结构
func (l *Log) ToAdminResponse() *AdminLogResponse {
	resp := &AdminLogResponse{
		ID:               l.ID,
		UserID:           l.UserID,
		TokenKey:         l.TokenKey,
		ChannelID:        l.ChannelID,
		Model:            l.Model,
		PromptTokens:     l.PromptTokens,
		CompletionTokens: l.CompletionTokens,
		TotalTokens:      l.TotalTokens(),
		TotalCost:        l.TotalCost.InexactFloat64(),
		UpstreamCost:     l.UpstreamCost.InexactFloat64(),
		Profit:           l.Profit().InexactFloat64(),
		Duration:         l.Duration,
		DurationMs:       l.DurationMs,
		StatusCode:       l.StatusCode,
		Status:           l.Status,
		IsStream:         l.IsStream,
		ErrorMessage:     l.ErrorMessage,
		RequestIP:        l.RequestIP,
		CreatedAt:        l.CreatedAt,
	}

	if l.User != nil {
		resp.Username = l.User.Username
	}
	if l.Channel != nil {
		resp.ChannelName = l.Channel.Name
	}

	return resp
}

// LogQueryParams 日志查询参数
type LogQueryParams struct {
	Page      int        `form:"page" binding:"min=1"`
	PageSize  int        `form:"page_size" binding:"min=1,max=100"`
	StartDate *string    `form:"start_date"`
	EndDate   *string    `form:"end_date"`
	Model     *string    `form:"model"`
	TokenKey  *string    `form:"token_key"`
	Status    *string    `form:"status"` // success, error
	UserID    *uuid.UUID `form:"user_id"`
	ChannelID *int       `form:"channel_id"`
}

// StatisticsResponse 统计响应
type StatisticsResponse struct {
	Summary StatisticsSummary `json:"summary"`
	ByModel []ModelStatistics `json:"by_model"`
	Trend   []TrendStatistics `json:"trend"`
}

// StatisticsSummary 统计摘要
type StatisticsSummary struct {
	TotalRequests int64           `json:"total_requests"`
	TotalTokens   int64           `json:"total_tokens"`
	TotalCost     decimal.Decimal `json:"total_cost"`
	AvgLatencyMs  int             `json:"avg_latency_ms"`
}

// ModelStatistics 模型统计
type ModelStatistics struct {
	Model    string          `json:"model"`
	Requests int64           `json:"requests"`
	Tokens   int64           `json:"tokens"`
	Cost     decimal.Decimal `json:"cost"`
}

// TrendStatistics 趋势统计
type TrendStatistics struct {
	Date     string          `json:"date"`
	Requests int64           `json:"requests"`
	Cost     decimal.Decimal `json:"cost"`
}
