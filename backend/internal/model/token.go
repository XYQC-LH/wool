package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
)

// TokenStatus Token 状态
type TokenStatus string

const (
	TokenStatusActive   TokenStatus = "active"
	TokenStatusDisabled TokenStatus = "disabled"
	TokenStatusExpired  TokenStatus = "expired"
)

// Token API 密钥模型
type Token struct {
	ID             uuid.UUID        `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Key            string           `gorm:"type:varchar(64);uniqueIndex;not null" json:"key"`
	UserID         uuid.UUID        `gorm:"type:uuid;not null;index" json:"user_id"`
	Name           string           `gorm:"type:varchar(100);not null" json:"name"`
	RemainQuota    *decimal.Decimal `gorm:"type:decimal(12,4)" json:"remain_quota,omitempty"`
	UnlimitedQuota bool             `gorm:"default:false" json:"unlimited_quota"`
	Status         TokenStatus      `gorm:"type:varchar(20);default:'active'" json:"status"`
	AllowedModels  pq.StringArray   `gorm:"type:text[]" json:"allowed_models,omitempty"`
	AllowedIPs     pq.StringArray   `gorm:"type:text[]" json:"allowed_ips,omitempty"`
	RateLimit      *int             `gorm:"type:int" json:"rate_limit,omitempty"`
	LastUsedAt     *time.Time       `json:"last_used_at,omitempty"`
	ExpiresAt      *time.Time       `json:"expires_at,omitempty"`
	CreatedAt      time.Time        `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 表名
func (Token) TableName() string {
	return "tokens"
}

// IsValid 检查 Token 是否有效
func (t *Token) IsValid() bool {
	if t.Status != TokenStatusActive {
		return false
	}
	if t.ExpiresAt != nil && t.ExpiresAt.Before(time.Now()) {
		return false
	}
	return true
}

// IsExpired 检查 Token 是否过期
func (t *Token) IsExpired() bool {
	if t.ExpiresAt == nil {
		return false
	}
	return t.ExpiresAt.Before(time.Now())
}

// HasQuota 检查是否有配额
func (t *Token) HasQuota(amount decimal.Decimal) bool {
	if t.UnlimitedQuota {
		return true
	}
	if t.RemainQuota == nil {
		return true
	}
	return t.RemainQuota.GreaterThanOrEqual(amount)
}

// IsModelAllowed 检查 Token 是否允许访问指定模型（AllowedModels 为空表示不限制）
func (t *Token) IsModelAllowed(modelName string) bool {
	if len(t.AllowedModels) == 0 {
		return true
	}

	for _, allowed := range t.AllowedModels {
		if strings.TrimSpace(allowed) == modelName {
			return true
		}
	}

	return false
}

// IsIPAllowed 检查 Token 是否允许来自指定 IP 的请求（AllowedIPs 为空表示不限制）
func (t *Token) IsIPAllowed(ip string) bool {
	if len(t.AllowedIPs) == 0 {
		return true
	}

	for _, allowed := range t.AllowedIPs {
		if strings.TrimSpace(allowed) == ip {
			return true
		}
	}

	return false
}

// TokenResponse Token 响应结构
type TokenResponse struct {
	ID             uuid.UUID        `json:"id"`
	Key            string           `json:"key"`
	Name           string           `json:"name"`
	RemainQuota    *decimal.Decimal `json:"remain_quota,omitempty"`
	UnlimitedQuota bool             `json:"unlimited_quota"`
	Status         TokenStatus      `json:"status"`
	AllowedModels  []string         `json:"allowed_models,omitempty"`
	AllowedIPs     []string         `json:"allowed_ips,omitempty"`
	RateLimit      *int             `json:"rate_limit,omitempty"`
	LastUsedAt     *time.Time       `json:"last_used_at,omitempty"`
	ExpiresAt      *time.Time       `json:"expires_at,omitempty"`
	CreatedAt      time.Time        `json:"created_at"`
	Usage          *TokenUsageStats `json:"usage,omitempty"`
}

// TokenUsageStats Token 用量统计（用于列表展示等轻量场景）
type TokenUsageStats struct {
	TokenID      uuid.UUID       `json:"-" gorm:"column:token_id"`
	RequestCount int64           `json:"request_count" gorm:"column:request_count"`
	TotalTokens  int64           `json:"total_tokens" gorm:"column:total_tokens"`
	TotalCost    decimal.Decimal `json:"total_cost" gorm:"column:total_cost"`
}

// ToResponse 转换为响应结构
func (t *Token) ToResponse() *TokenResponse {
	// 脱敏处理 Key
	maskedKey := t.Key
	if len(t.Key) > 12 {
		maskedKey = t.Key[:8] + "..." + t.Key[len(t.Key)-4:]
	}

	return &TokenResponse{
		ID:             t.ID,
		Key:            maskedKey,
		Name:           t.Name,
		RemainQuota:    t.RemainQuota,
		UnlimitedQuota: t.UnlimitedQuota,
		Status:         t.Status,
		AllowedModels:  t.AllowedModels,
		AllowedIPs:     t.AllowedIPs,
		RateLimit:      t.RateLimit,
		LastUsedAt:     t.LastUsedAt,
		ExpiresAt:      t.ExpiresAt,
		CreatedAt:      t.CreatedAt,
	}
}

// CreateTokenRequest 创建 Token 请求
type CreateTokenRequest struct {
	Name          string     `json:"name" binding:"required,min=1,max=100"`
	Quota         *float64   `json:"quota,omitempty"`
	ExpiresAt     *time.Time `json:"expires_at,omitempty"`
	AllowedModels []string   `json:"allowed_models,omitempty"`
	AllowedIPs    []string   `json:"allowed_ips,omitempty"`
	RateLimit     *int       `json:"rate_limit,omitempty"`
}

// UpdateTokenRequest 更新 Token 请求
type UpdateTokenRequest struct {
	Name          *string     `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	Status        TokenStatus `json:"status,omitempty"`
	Quota         *float64    `json:"quota,omitempty"`
	ExpiresAt     *time.Time  `json:"expires_at,omitempty"`
	RateLimit     *int        `json:"rate_limit,omitempty"`
	AllowedModels *[]string   `json:"allowed_models,omitempty"`
	AllowedIPs    *[]string   `json:"allowed_ips,omitempty"`
}
