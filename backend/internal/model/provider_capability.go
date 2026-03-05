package model

import "time"

type ProviderCapability struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID  uint      `gorm:"not null;index;uniqueIndex:idx_provider_capability,priority:1" json:"provider_id"`
	Operation   string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_provider_capability,priority:2" json:"operation"`
	Constraints JSON      `gorm:"type:jsonb" json:"constraints"`
	IsEnabled   bool      `gorm:"not null;default:true" json:"is_enabled"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ProviderCapability) TableName() string { return "provider_capabilities" }

type ProviderCapabilitySummary struct {
	ProviderID          uint           `json:"provider_id,omitempty"`
	Total               int64          `json:"total"`
	Enabled             int64          `json:"enabled"`
	Disabled            int64          `json:"disabled"`
	OperationBreakdown  map[string]int `json:"operation_breakdown"`
}

type ProviderCapabilityValidationResult struct {
	ProviderID   uint   `json:"provider_id"`
	Operation    string `json:"operation"`
	CapabilityID uint   `json:"capability_id,omitempty"`
	Matched      bool   `json:"matched"`
	Reason       string `json:"reason,omitempty"`
}
