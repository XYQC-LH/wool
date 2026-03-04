package model

import (
	"time"

	"gorm.io/gorm"
)

// AnnouncementStatus 公告状态
type AnnouncementStatus string

const (
	AnnouncementStatusDraft     AnnouncementStatus = "draft"
	AnnouncementStatusPublished AnnouncementStatus = "published"
	AnnouncementStatusArchived  AnnouncementStatus = "archived"
)

// AnnouncementType 公告类型
type AnnouncementType string

const (
	AnnouncementTypeInfo    AnnouncementType = "info"
	AnnouncementTypeWarning AnnouncementType = "warning"
	AnnouncementTypeSuccess AnnouncementType = "success"
	AnnouncementTypeError   AnnouncementType = "error"
)

// Announcement 公告模型
type Announcement struct {
	ID          uint               `gorm:"primaryKey;autoIncrement" json:"id"`
	Title       string             `gorm:"type:varchar(200);not null" json:"title"`
	Content     string             `gorm:"type:text;not null" json:"content"`
	Type        AnnouncementType   `gorm:"type:varchar(20);default:'info'" json:"type"`
	Status      AnnouncementStatus `gorm:"type:varchar(20);default:'draft'" json:"status"`
	Priority    int                `gorm:"default:0" json:"priority"` // 优先级，数字越大越靠前
	PublishedAt *time.Time         `json:"published_at,omitempty"`
	ExpiresAt   *time.Time         `json:"expires_at,omitempty"`
	CreatedAt   time.Time          `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time          `gorm:"autoUpdateTime" json:"updated_at"`
}

// TableName 表名
func (Announcement) TableName() string {
	return "announcements"
}

// IsPublished 是否已发布
func (a *Announcement) IsPublished() bool {
	return a.Status == AnnouncementStatusPublished
}

// IsExpired 是否已过期
func (a *Announcement) IsExpired() bool {
	if a.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*a.ExpiresAt)
}

// IsActive 是否活跃（已发布且未过期）
func (a *Announcement) IsActive() bool {
	return a.IsPublished() && !a.IsExpired()
}

// Publish 发布公告
func (a *Announcement) Publish(db *gorm.DB) error {
	now := time.Now()
	return db.Model(a).Updates(map[string]interface{}{
		"status":       AnnouncementStatusPublished,
		"published_at": now,
	}).Error
}

// Archive 归档公告
func (a *Announcement) Archive(db *gorm.DB) error {
	return db.Model(a).Update("status", AnnouncementStatusArchived).Error
}

// AnnouncementResponse 公告响应结构
type AnnouncementResponse struct {
	ID          uint               `json:"id"`
	Title       string             `json:"title"`
	Content     string             `json:"content"`
	Type        AnnouncementType   `json:"type"`
	Status      AnnouncementStatus `json:"status"`
	Priority    int                `json:"priority"`
	PublishedAt *time.Time         `json:"published_at,omitempty"`
	ExpiresAt   *time.Time         `json:"expires_at,omitempty"`
	CreatedAt   time.Time          `json:"created_at"`
}

// ToResponse 转换为响应结构
func (a *Announcement) ToResponse() *AnnouncementResponse {
	return &AnnouncementResponse{
		ID:          a.ID,
		Title:       a.Title,
		Content:     a.Content,
		Type:        a.Type,
		Status:      a.Status,
		Priority:    a.Priority,
		PublishedAt: a.PublishedAt,
		ExpiresAt:   a.ExpiresAt,
		CreatedAt:   a.CreatedAt,
	}
}

// CreateAnnouncementRequest 创建公告请求
type CreateAnnouncementRequest struct {
	Title     string             `json:"title" binding:"required,max=200"`
	Content   string             `json:"content" binding:"required"`
	Type      AnnouncementType   `json:"type" binding:"omitempty,oneof=info warning success error"`
	Status    AnnouncementStatus `json:"status" binding:"omitempty,oneof=draft published"`
	Priority  int                `json:"priority" binding:"omitempty,min=0,max=100"`
	ExpiresAt *time.Time         `json:"expires_at,omitempty"`
}

// UpdateAnnouncementRequest 更新公告请求
type UpdateAnnouncementRequest struct {
	Title     *string            `json:"title,omitempty" binding:"omitempty,max=200"`
	Content   *string            `json:"content,omitempty"`
	Type      AnnouncementType   `json:"type,omitempty" binding:"omitempty,oneof=info warning success error"`
	Status    AnnouncementStatus `json:"status,omitempty" binding:"omitempty,oneof=draft published archived"`
	Priority  *int               `json:"priority,omitempty" binding:"omitempty,min=0,max=100"`
	ExpiresAt *time.Time         `json:"expires_at,omitempty"`
}
