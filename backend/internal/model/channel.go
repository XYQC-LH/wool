package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// ChannelType 渠道类型
type ChannelType string

const (
	ChannelTypeOfficial          ChannelType = "official"
	ChannelTypeReverseEngineered ChannelType = "reverse_engineered"
	ChannelTypeProxy             ChannelType = "proxy"
)

// ChannelStatus 渠道状态
type ChannelStatus string

const (
	ChannelStatusHealthy  ChannelStatus = "healthy"
	ChannelStatusDegraded ChannelStatus = "degraded"
	ChannelStatusDown     ChannelStatus = "down"
	ChannelStatusDisabled ChannelStatus = "disabled"
)

// Channel 渠道模型
type Channel struct {
	ID              uint            `gorm:"primaryKey;autoIncrement" json:"id"`
	Name            string          `gorm:"type:varchar(100);not null" json:"name"`
	Type            ChannelType     `gorm:"type:varchar(20);not null" json:"type"`
	BaseURL         string          `gorm:"type:varchar(500);not null" json:"base_url"`
	APIKey          string          `gorm:"type:text" json:"-"`
	Models          StringArray     `gorm:"type:jsonb" json:"models"`
	ModelMapping    JSON            `gorm:"type:jsonb" json:"model_mapping"`
	Weight          int             `gorm:"default:1" json:"weight"`
	Priority        int             `gorm:"default:0" json:"priority"`
	Status          ChannelStatus   `gorm:"type:varchar(20);default:'healthy'" json:"status"`
	Latency         int             `gorm:"default:0" json:"latency"`
	ErrorCount      int             `gorm:"default:0" json:"error_count"`
	MaxConcurrent   int             `gorm:"default:100" json:"max_concurrent"`
	RateLimit       int             `gorm:"default:60" json:"rate_limit"`
	TimeoutMs       int             `gorm:"default:30000" json:"timeout_ms"`
	RetryCount      int             `gorm:"default:3" json:"retry_count"`
	Config          JSON            `gorm:"type:jsonb;default:'{}'" json:"config"`
	LastTestAt      *time.Time      `json:"last_test_at,omitempty"`
	LastTestLatency *int            `json:"last_test_latency,omitempty"`
	SuccessRate     decimal.Decimal `gorm:"type:decimal(5,2);default:100.00" json:"success_rate"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt       time.Time       `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	ChannelModels    []ChannelModel    `gorm:"foreignKey:ChannelID" json:"channel_models,omitempty"`
	ResourceAccounts []ResourceAccount `gorm:"foreignKey:ChannelID" json:"resource_accounts,omitempty"`
}

// TableName 表名
func (Channel) TableName() string {
	return "channels"
}

// IsAvailable 检查渠道是否可用
func (c *Channel) IsAvailable() bool {
	return c.Status == ChannelStatusHealthy || c.Status == ChannelStatusDegraded
}

// ChannelResponse 渠道响应结构
type ChannelResponse struct {
	ID              uint            `json:"id"`
	Name            string          `json:"name"`
	Type            ChannelType     `json:"type"`
	BaseURL         string          `json:"base_url"`
	Weight          int             `json:"weight"`
	Priority        int             `json:"priority"`
	Status          ChannelStatus   `json:"status"`
	Latency         int             `json:"latency"`
	ErrorCount      int             `json:"error_count"`
	MaxConcurrent   int             `json:"max_concurrent"`
	RateLimit       int             `json:"rate_limit"`
	TimeoutMs       int             `json:"timeout_ms"`
	RetryCount      int             `json:"retry_count"`
	LastTestAt      *time.Time      `json:"last_test_at,omitempty"`
	LastTestLatency *int            `json:"last_test_latency,omitempty"`
	SuccessRate     decimal.Decimal `json:"success_rate"`
	Models          []string        `json:"models,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ToResponse 转换为响应结构
func (c *Channel) ToResponse() *ChannelResponse {
	return &ChannelResponse{
		ID:              c.ID,
		Name:            c.Name,
		Type:            c.Type,
		BaseURL:         c.BaseURL,
		Weight:          c.Weight,
		Priority:        c.Priority,
		Status:          c.Status,
		Latency:         c.Latency,
		ErrorCount:      c.ErrorCount,
		MaxConcurrent:   c.MaxConcurrent,
		RateLimit:       c.RateLimit,
		TimeoutMs:       c.TimeoutMs,
		RetryCount:      c.RetryCount,
		LastTestAt:      c.LastTestAt,
		LastTestLatency: c.LastTestLatency,
		SuccessRate:     c.SuccessRate,
		Models:          c.Models,
		CreatedAt:       c.CreatedAt,
	}
}

// CreateChannelRequest 创建渠道请求
type CreateChannelRequest struct {
	Name          string                      `json:"name" binding:"required,min=1,max=100"`
	Type          ChannelType                 `json:"type" binding:"required,oneof=official reverse_engineered proxy"`
	BaseURL       string                      `json:"base_url" binding:"required,url"`
	APIKey        string                      `json:"api_key"`
	Models        []string                    `json:"models"`
	Weight        int                         `json:"weight" binding:"min=0"`
	Priority      int                         `json:"priority" binding:"min=0"`
	MaxConcurrent int                         `json:"max_concurrent" binding:"min=1"`
	RateLimit     int                         `json:"rate_limit" binding:"min=1"`
	TimeoutMs     int                         `json:"timeout_ms" binding:"min=1000"`
	RetryCount    int                         `json:"retry_count" binding:"min=0,max=10"`
	Config        map[string]interface{}      `json:"config,omitempty"`
	ModelMappings []CreateChannelModelRequest `json:"model_mappings,omitempty"`
}

// UpdateChannelRequest 更新渠道请求
type UpdateChannelRequest struct {
	Name          *string                `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Type          ChannelType            `json:"type,omitempty"`
	BaseURL       *string                `json:"base_url,omitempty" binding:"omitempty,url"`
	APIKey        *string                `json:"api_key,omitempty"`
	Models        []string               `json:"models,omitempty"`
	Weight        *int                   `json:"weight,omitempty" binding:"omitempty,min=0"`
	Priority      *int                   `json:"priority,omitempty" binding:"omitempty,min=0"`
	Status        ChannelStatus          `json:"status,omitempty"`
	MaxConcurrent *int                   `json:"max_concurrent,omitempty" binding:"omitempty,min=1"`
	RateLimit     *int                   `json:"rate_limit,omitempty" binding:"omitempty,min=1"`
	TimeoutMs     *int                   `json:"timeout_ms,omitempty" binding:"omitempty,min=1000"`
	RetryCount    *int                   `json:"retry_count,omitempty" binding:"omitempty,min=0,max=10"`
	Config        map[string]interface{} `json:"config,omitempty"`
}

// TestChannelResponse 测试渠道响应
type TestChannelResponse struct {
	Status    ChannelStatus `json:"status"`
	LatencyMs int           `json:"latency_ms"`
	Response  string        `json:"response"`
	Error     string        `json:"error,omitempty"`
}
