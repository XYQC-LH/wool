package model

import "time"

// ModelAlias 模型别名映射（用于将别名解析为实际模型名）
type ModelAlias struct {
	ID          uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	Alias       string    `gorm:"type:varchar(100);uniqueIndex;not null" json:"alias"`
	TargetModel string    `gorm:"type:varchar(100);not null" json:"target_model"`
	Enabled     bool      `gorm:"not null;default:true" json:"enabled"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (ModelAlias) TableName() string {
	return "model_aliases"
}
