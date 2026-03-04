package model

import (
	"strings"
	"time"
)

// ProviderRateLimitRule 多模态限流规则表 - 按 operation/单位定义限额
type ProviderRateLimitRule struct {
	ID        uint `gorm:"primaryKey;autoIncrement" json:"id"`
	Scope     string `gorm:"type:varchar(20);not null;default:'provider';index;uniqueIndex:idx_provider_rate_limit_rule,priority:1" json:"scope"`
	ProviderID uint `gorm:"not null;index;uniqueIndex:idx_provider_rate_limit_rule,priority:2" json:"provider_id"`
	// InstanceID 仅当 scope=instance 时生效
	InstanceID *uint `gorm:"index;uniqueIndex:idx_provider_rate_limit_rule,priority:3" json:"instance_id,omitempty"`
	Operation  string `gorm:"type:varchar(50);not null;uniqueIndex:idx_provider_rate_limit_rule,priority:4" json:"operation"`
	Unit       string `gorm:"type:varchar(50);not null;uniqueIndex:idx_provider_rate_limit_rule,priority:5" json:"unit"`

	Limit         int64 `gorm:"not null;default:0" json:"limit"`
	WindowSeconds int   `gorm:"not null;default:60;uniqueIndex:idx_provider_rate_limit_rule,priority:6" json:"window_seconds"`

	Enabled bool `gorm:"not null;default:true" json:"enabled"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Provider *ModelProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

func (ProviderRateLimitRule) TableName() string {
	return "provider_rate_limit_rules"
}

const (
	RateLimitScopeProvider = "provider"
	RateLimitScopeInstance = "instance"
)

func NormalizeRateLimitScope(scope string) string {
	scope = strings.ToLower(strings.TrimSpace(scope))
	switch scope {
	case RateLimitScopeInstance:
		return RateLimitScopeInstance
	default:
		return RateLimitScopeProvider
	}
}
