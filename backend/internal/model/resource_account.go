package model

import (
	"time"

	"gorm.io/gorm"
)

// ResourceAccountStatus 资源账户状态
type ResourceAccountStatus string

const (
	ResourceAccountStatusActive   ResourceAccountStatus = "active"
	ResourceAccountStatusInactive ResourceAccountStatus = "inactive"
	ResourceAccountStatusExpired  ResourceAccountStatus = "expired"
	ResourceAccountStatusBanned   ResourceAccountStatus = "banned"
)

// ResourceAccount 资源账户模型（用于逆向工程账户管理）
type ResourceAccount struct {
	ID           uint                  `gorm:"primaryKey;autoIncrement" json:"id"`
	ChannelID    uint                  `gorm:"not null;index" json:"channel_id"`
	AccountName  string                `gorm:"type:varchar(100);not null" json:"account_name"`
	Credentials  JSON                  `gorm:"type:jsonb;default:'{}'" json:"credentials,omitempty"` // 加密存储的凭证
	SessionToken string                `gorm:"type:text" json:"-"`                                   // 当前会话令牌
	CookieData   string                `gorm:"type:text" json:"-"`                                   // Cookie 数据
	Status       ResourceAccountStatus `gorm:"type:varchar(20);default:'active'" json:"status"`
	LastActiveAt *time.Time            `json:"last_active_at,omitempty"`
	ExpiresAt    *time.Time            `json:"expires_at,omitempty"`
	ErrorCount   int                   `gorm:"default:0" json:"error_count"`
	LastError    string                `gorm:"type:text" json:"last_error,omitempty"`
	Metadata     JSON                  `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt    time.Time             `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time             `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Channel *Channel `gorm:"foreignKey:ChannelID" json:"channel,omitempty"`
}

// TableName 表名
func (ResourceAccount) TableName() string {
	return "resource_accounts"
}

// IsActive 是否活跃
func (r *ResourceAccount) IsActive() bool {
	return r.Status == ResourceAccountStatusActive
}

// IsExpired 是否过期
func (r *ResourceAccount) IsExpired() bool {
	if r.ExpiresAt == nil {
		return false
	}
	return time.Now().After(*r.ExpiresAt)
}

// MarkActive 标记为活跃
func (r *ResourceAccount) MarkActive(db *gorm.DB) error {
	now := time.Now()
	return db.Model(r).Updates(map[string]interface{}{
		"status":         ResourceAccountStatusActive,
		"last_active_at": now,
		"error_count":    0,
		"last_error":     "",
	}).Error
}

// MarkError 标记错误
func (r *ResourceAccount) MarkError(db *gorm.DB, errMsg string) error {
	updates := map[string]interface{}{
		"error_count": gorm.Expr("error_count + 1"),
		"last_error":  errMsg,
	}

	// 如果错误次数超过阈值，标记为不活跃
	if r.ErrorCount >= 4 { // 第5次错误时标记为不活跃
		updates["status"] = ResourceAccountStatusInactive
	}

	return db.Model(r).Updates(updates).Error
}

// UpdateSession 更新会话信息
func (r *ResourceAccount) UpdateSession(db *gorm.DB, sessionToken, cookieData string) error {
	now := time.Now()
	return db.Model(r).Updates(map[string]interface{}{
		"session_token":  sessionToken,
		"cookie_data":    cookieData,
		"last_active_at": now,
		"status":         ResourceAccountStatusActive,
	}).Error
}

// ResourceAccountResponse 资源账户响应结构
type ResourceAccountResponse struct {
	ID           uint                  `json:"id"`
	ChannelID    uint                  `json:"channel_id"`
	ChannelName  string                `json:"channel_name,omitempty"`
	AccountName  string                `json:"account_name"`
	Status       ResourceAccountStatus `json:"status"`
	LastActiveAt *time.Time            `json:"last_active_at,omitempty"`
	ExpiresAt    *time.Time            `json:"expires_at,omitempty"`
	ErrorCount   int                   `json:"error_count"`
	LastError    string                `json:"last_error,omitempty"`
	CreatedAt    time.Time             `json:"created_at"`
	UpdatedAt    time.Time             `json:"updated_at"`
}

// ToResponse 转换为响应结构
func (r *ResourceAccount) ToResponse() *ResourceAccountResponse {
	resp := &ResourceAccountResponse{
		ID:           r.ID,
		ChannelID:    r.ChannelID,
		AccountName:  r.AccountName,
		Status:       r.Status,
		LastActiveAt: r.LastActiveAt,
		ExpiresAt:    r.ExpiresAt,
		ErrorCount:   r.ErrorCount,
		LastError:    r.LastError,
		CreatedAt:    r.CreatedAt,
		UpdatedAt:    r.UpdatedAt,
	}

	if r.Channel != nil {
		resp.ChannelName = r.Channel.Name
	}

	return resp
}

// CreateResourceAccountRequest 创建资源账户请求
type CreateResourceAccountRequest struct {
	ChannelID   uint                  `json:"channel_id" binding:"required"`
	AccountName string                `json:"account_name" binding:"required,max=100"`
	Credentials map[string]string     `json:"credentials" binding:"required"`
	Status      ResourceAccountStatus `json:"status,omitempty" binding:"omitempty,oneof=active inactive expired banned"`
	ExpiresAt   *time.Time            `json:"expires_at,omitempty"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
}

// UpdateResourceAccountRequest 更新资源账户请求
type UpdateResourceAccountRequest struct {
	AccountName *string               `json:"account_name,omitempty" binding:"omitempty,max=100"`
	Credentials map[string]string     `json:"credentials,omitempty"`
	Status      ResourceAccountStatus `json:"status,omitempty" binding:"omitempty,oneof=active inactive expired banned"`
	ExpiresAt   *time.Time            `json:"expires_at,omitempty"`
	Metadata    map[string]any        `json:"metadata,omitempty"`
}

// ResourcePoolStats 资源池统计
type ResourcePoolStats struct {
	TotalAccounts    int64 `json:"total_accounts"`
	ActiveAccounts   int64 `json:"active_accounts"`
	InactiveAccounts int64 `json:"inactive_accounts"`
	ExpiredAccounts  int64 `json:"expired_accounts"`
	BannedAccounts   int64 `json:"banned_accounts"`
}
