package model

import (
	"fmt"
	"strings"
	"time"
)

type ModelRoute struct {
	ID         uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	RouteKey   string `gorm:"type:varchar(160);not null;index:idx_model_route_lookup,priority:1" json:"route_key"`
	Operation  string `gorm:"type:varchar(50);not null;index:idx_model_route_lookup,priority:2;uniqueIndex:idx_model_route_unique,priority:1" json:"operation"`
	ModelID    string `gorm:"type:varchar(100);not null;index:idx_model_route_lookup,priority:3;uniqueIndex:idx_model_route_unique,priority:2" json:"model_id"`
	ProviderID uint   `gorm:"not null;default:0;index;index:idx_model_route_lookup,priority:4;uniqueIndex:idx_model_route_unique,priority:3" json:"provider_id"`
	Priority   int    `gorm:"not null;default:100" json:"priority"`
	IsEnabled  bool   `gorm:"not null;default:true" json:"is_enabled"`

	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	Provider *ModelProvider `gorm:"foreignKey:ProviderID;references:ID" json:"provider,omitempty"`
}

type ModelRouteStats struct {
	TotalRoutes       int64 `json:"total_routes"`
	EnabledRoutes     int64 `json:"enabled_routes"`
	DisabledRoutes    int64 `json:"disabled_routes"`
	DistinctModels    int64 `json:"distinct_models"`
	DistinctProviders int64 `json:"distinct_providers"`
}

func BuildModelRouteKey(operation string, modelID string, providerID uint) string {
	return fmt.Sprintf("%s:%s:%d", NormalizeOperation(operation), strings.TrimSpace(modelID), providerID)
}

func (ModelRoute) TableName() string { return "model_routes" }
