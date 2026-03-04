package model

import "time"

// ModelCapability 模型能力表 - 定义模型支持哪些 operation（chat/image/audio/video...）
type ModelCapability struct {
	ID        uint   `gorm:"primaryKey;autoIncrement" json:"id"`
	ModelID   string `gorm:"type:varchar(100);not null;index;uniqueIndex:idx_model_capability,priority:1" json:"model_id"`
	Operation string `gorm:"type:varchar(50);not null;uniqueIndex:idx_model_capability,priority:2" json:"operation"`
	Enabled   bool   `gorm:"not null;default:true" json:"enabled"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Model *Model `gorm:"foreignKey:ModelID;references:ID" json:"model,omitempty"`
}

func (ModelCapability) TableName() string {
	return "model_capabilities"
}
