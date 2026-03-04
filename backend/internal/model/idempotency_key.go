package model

import (
	"time"

	"github.com/google/uuid"
)

// IdempotencyKey 幂等键表 - 用于防重复下单/重复创建任务
type IdempotencyKey struct {
	ID        uint      `gorm:"primaryKey;autoIncrement" json:"id"`
	UserID    uuid.UUID `gorm:"type:uuid;not null;index;uniqueIndex:idx_idempotency_user_operation_key,priority:1" json:"user_id"`
	TokenID   uuid.UUID `gorm:"type:uuid;not null;index" json:"token_id"`
	Operation string    `gorm:"type:varchar(50);not null;uniqueIndex:idx_idempotency_user_operation_key,priority:2" json:"operation"`
	Key       string    `gorm:"type:varchar(128);not null;uniqueIndex:idx_idempotency_user_operation_key,priority:3" json:"key"`

	RequestHash  string    `gorm:"type:varchar(64);not null" json:"request_hash"`
	ResourceType string    `gorm:"type:varchar(50);not null;default:'generation_task'" json:"resource_type"`
	ResourceID   uuid.UUID `gorm:"type:uuid;not null;index" json:"resource_id"`

	Status    string     `gorm:"type:varchar(20);not null;default:'completed'" json:"status"`
	ExpiresAt *time.Time `gorm:"index" json:"expires_at,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (IdempotencyKey) TableName() string {
	return "idempotency_keys"
}
