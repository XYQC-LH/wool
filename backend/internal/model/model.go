package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// ModelStatus 模型状态
type ModelStatus string

const (
	ModelStatusActive     ModelStatus = "active"
	ModelStatusDisabled   ModelStatus = "disabled"
	ModelStatusDeprecated ModelStatus = "deprecated"
)

// Model AI 模型定义
type Model struct {
	ID              string          `gorm:"primaryKey;type:varchar(100)" json:"id"`
	Name            string          `gorm:"type:varchar(100);uniqueIndex;not null" json:"name"`
	DisplayName     string          `gorm:"type:varchar(100);not null" json:"display_name"`
	Provider        string          `gorm:"type:varchar(50);not null" json:"provider"`
	InputPrice      decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"input_price"`
	OutputPrice     decimal.Decimal `gorm:"type:decimal(10,6);not null" json:"output_price"`
	PriceUnit       int             `gorm:"default:1000000" json:"price_unit"`
	MaxTokens       int             `gorm:"default:4096" json:"max_tokens"`
	MaxContext      int             `gorm:"default:8192" json:"max_context"`
	ContextLength   int             `gorm:"default:8192" json:"context_length"`
	MaxOutputTokens int             `gorm:"default:4096" json:"max_output_tokens"`
	Status          ModelStatus     `gorm:"type:varchar(20);default:'active'" json:"status"`
	Enabled         bool            `gorm:"default:true" json:"enabled"`
	Description     *string         `gorm:"type:text" json:"description,omitempty"`
	CreatedAt       time.Time       `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	ChannelModels []ChannelModel `gorm:"foreignKey:ModelID" json:"channel_models,omitempty"`
}

// TableName 表名
func (Model) TableName() string {
	return "models"
}

// IsAvailable 检查模型是否可用
func (m *Model) IsAvailable() bool {
	return m.Status == ModelStatusActive
}

// ModelResponse 模型响应结构
type ModelResponse struct {
	ID              string          `json:"id"`
	Name            string          `json:"name"`
	DisplayName     string          `json:"display_name"`
	Provider        string          `json:"provider"`
	InputPrice      decimal.Decimal `json:"input_price"`
	OutputPrice     decimal.Decimal `json:"output_price"`
	PriceUnit       int             `json:"price_unit"`
	MaxTokens       int             `json:"max_tokens"`
	MaxContext      int             `json:"max_context"`
	ContextLength   int             `json:"context_length"`
	MaxOutputTokens int             `json:"max_output_tokens"`
	Status          ModelStatus     `json:"status"`
	Enabled         bool            `json:"enabled"`
	Description     *string         `json:"description,omitempty"`
	CreatedAt       time.Time       `json:"created_at"`
}

// ToResponse 转换为响应结构
func (m *Model) ToResponse() *ModelResponse {
	return &ModelResponse{
		ID:              m.ID,
		Name:            m.Name,
		DisplayName:     m.DisplayName,
		Provider:        m.Provider,
		InputPrice:      m.InputPrice,
		OutputPrice:     m.OutputPrice,
		PriceUnit:       m.PriceUnit,
		MaxTokens:       m.MaxTokens,
		MaxContext:      m.MaxContext,
		ContextLength:   m.ContextLength,
		MaxOutputTokens: m.MaxOutputTokens,
		Status:          m.Status,
		Enabled:         m.Enabled,
		Description:     m.Description,
		CreatedAt:       m.CreatedAt,
	}
}

// CreateModelRequest 创建模型请求
type CreateModelRequest struct {
	Name        string  `json:"name" binding:"required,min=1,max=100"`
	DisplayName string  `json:"display_name" binding:"required,min=1,max=100"`
	Provider    string  `json:"provider" binding:"required,min=1,max=50"`
	InputPrice  float64 `json:"input_price" binding:"required,min=0"`
	OutputPrice float64 `json:"output_price" binding:"required,min=0"`
	MaxTokens   int     `json:"max_tokens" binding:"min=1"`
	MaxContext  int     `json:"max_context" binding:"min=1"`
	Description *string `json:"description,omitempty"`
}

// UpdateModelRequest 更新模型请求
type UpdateModelRequest struct {
	DisplayName *string      `json:"display_name,omitempty" binding:"omitempty,min=1,max=100"`
	InputPrice  *float64     `json:"input_price,omitempty" binding:"omitempty,min=0"`
	OutputPrice *float64     `json:"output_price,omitempty" binding:"omitempty,min=0"`
	MaxTokens   *int         `json:"max_tokens,omitempty" binding:"omitempty,min=1"`
	MaxContext  *int         `json:"max_context,omitempty" binding:"omitempty,min=1"`
	Status      *ModelStatus `json:"status,omitempty"`
	Description *string      `json:"description,omitempty"`
}

// ChannelModel 渠道模型映射
type ChannelModel struct {
	ID                int             `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID         uint            `gorm:"not null;index" json:"channel_id"`
	ModelID           string          `gorm:"type:varchar(100);not null;index" json:"model_id"`
	UpstreamModelName string          `gorm:"type:varchar(100);not null" json:"upstream_model_name"`
	CostRatio         decimal.Decimal `gorm:"type:decimal(5,2);default:1.00" json:"cost_ratio"`
	Status            string          `gorm:"type:varchar(20);default:'active'" json:"status"`

	// 关联
	Channel *Channel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
	Model   *Model   `gorm:"foreignKey:ModelID" json:"model,omitempty"`
}

// TableName 表名
func (ChannelModel) TableName() string {
	return "channel_models"
}

// CreateChannelModelRequest 创建渠道模型映射请求
type CreateChannelModelRequest struct {
	ModelID           int     `json:"model_id" binding:"required"`
	UpstreamModelName string  `json:"upstream_model_name" binding:"required,min=1,max=100"`
	CostRatio         float64 `json:"cost_ratio" binding:"min=0"`
}

// OpenAI 兼容的模型列表响应
type OpenAIModelResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// OpenAIModelListResponse OpenAI 模型列表响应
type OpenAIModelListResponse struct {
	Object string                `json:"object"`
	Data   []OpenAIModelResponse `json:"data"`
}

// ModelPricing 模型定价信息（用于网关计费）
type ModelPricing struct {
	ModelID         string          `json:"model_id"`
	InputPrice      decimal.Decimal `json:"input_price"`
	OutputPrice     decimal.Decimal `json:"output_price"`
	PriceUnit       int             `json:"price_unit"`
	ContextLength   int             `json:"context_length"`
	MaxOutputTokens int             `json:"max_output_tokens"`
}
