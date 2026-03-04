package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// ProviderPricingRule 多模态计费规则表 - 按 operation/单位定义成本与售价（tokens/seconds/images/pixels...）
type ProviderPricingRule struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID uint   `gorm:"not null;index;uniqueIndex:idx_provider_pricing_rule,priority:1" json:"provider_id"`
	Operation  string `gorm:"type:varchar(50);not null;uniqueIndex:idx_provider_pricing_rule,priority:2" json:"operation"`
	Unit       string `gorm:"type:varchar(50);not null;uniqueIndex:idx_provider_pricing_rule,priority:3" json:"unit"`

	// 成本/售价（按 Unit 计价）
	CostPerUnit  decimal.Decimal `gorm:"type:decimal(12,6);not null;default:0" json:"cost_per_unit"`
	PricePerUnit decimal.Decimal `gorm:"type:decimal(12,6);not null;default:0" json:"price_per_unit"`
	Meta         JSON            `gorm:"type:jsonb" json:"meta,omitempty"`

	Enabled bool `gorm:"not null;default:true" json:"enabled"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Provider *ModelProvider `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
}

func (ProviderPricingRule) TableName() string {
	return "provider_pricing_rules"
}
