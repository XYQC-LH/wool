package model

import (
	"time"

	"github.com/google/uuid"
)

// SystemSetting 系统设置（按 section 存储 JSON 配置）
// - section: general/security/notification/system
// - data: JSONB 数据
// - updated_by: 最后更新人（管理员用户 ID）
type SystemSetting struct {
	Section   string     `gorm:"type:varchar(50);primaryKey" json:"section"`
	Data      JSON       `gorm:"type:jsonb;not null;default:'{}'" json:"data"`
	UpdatedBy *uuid.UUID `gorm:"type:uuid" json:"updated_by,omitempty"`
	CreatedAt time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SystemSetting) TableName() string {
	return "system_settings"
}
