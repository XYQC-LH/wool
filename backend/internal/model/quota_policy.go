package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// QuotaPolicyStatus 配额策略状态
type QuotaPolicyStatus string

const (
	QuotaPolicyStatusActive   QuotaPolicyStatus = "active"
	QuotaPolicyStatusDisabled QuotaPolicyStatus = "disabled"
)

// QuotaPolicy 租户配额策略
type QuotaPolicy struct {
	ID                    uuid.UUID         `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	TenantID              string            `gorm:"type:varchar(64);not null;uniqueIndex" json:"tenant_id"`
	Name                  string            `gorm:"type:varchar(100);not null" json:"name"`
	Description           string            `gorm:"type:text" json:"description,omitempty"`
	DailyRequestLimit     int64             `gorm:"not null;default:0" json:"daily_request_limit"`
	DailyCostLimit        decimal.Decimal   `gorm:"type:decimal(16,6);not null;default:0" json:"daily_cost_limit"`
	AlertThresholdPercent int               `gorm:"not null;default:80" json:"alert_threshold_percent"`
	Status                QuotaPolicyStatus `gorm:"type:varchar(20);not null;default:'active';index" json:"status"`
	CreatedAt             time.Time         `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time         `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (QuotaPolicy) TableName() string {
	return "quota_policies"
}

func (s QuotaPolicyStatus) IsValid() bool {
	switch s {
	case QuotaPolicyStatusActive, QuotaPolicyStatusDisabled:
		return true
	default:
		return false
	}
}

// NormalizeQuotaPolicyStatus 标准化状态值
func NormalizeQuotaPolicyStatus(raw string) QuotaPolicyStatus {
	status := QuotaPolicyStatus(strings.ToLower(strings.TrimSpace(raw)))
	if status.IsValid() {
		return status
	}
	return QuotaPolicyStatusActive
}

// EffectiveThresholdPercent 生效告警阈值（百分比）
func (p *QuotaPolicy) EffectiveThresholdPercent() int {
	if p == nil {
		return 80
	}
	if p.AlertThresholdPercent <= 0 {
		return 80
	}
	if p.AlertThresholdPercent > 100 {
		return 100
	}
	return p.AlertThresholdPercent
}

