package model

import "time"

type ModelRoute struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	RouteKey    string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_route_key,priority:1" json:"route_key"`
	Operation   string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_route_key,priority:2" json:"operation"`
	ModelID     string    `gorm:"type:varchar(100);not null;uniqueIndex:idx_route_key,priority:3" json:"model_id"`
	Priority    int       `gorm:"not null;default:0" json:"priority"`
	IsEnabled   bool      `gorm:"not null;default:true" json:"is_enabled"`
	Description string    `gorm:"type:text" json:"description,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ModelRoute) TableName() string { return "model_routes" }
