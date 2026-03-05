package model

import (
	"time"

	"github.com/google/uuid"
)

// AlertSeverity 告警严重级别
type AlertSeverity string

const (
	AlertSeverityInfo     AlertSeverity = "info"
	AlertSeverityWarning  AlertSeverity = "warning"
	AlertSeverityError    AlertSeverity = "error"
	AlertSeverityCritical AlertSeverity = "critical"
)

// AlertStatus 告警状态
type AlertStatus string

const (
	AlertStatusActive   AlertStatus = "active"
	AlertStatusResolved AlertStatus = "resolved"
	AlertStatusIgnored  AlertStatus = "ignored"
)

// AlertType 告警类型
type AlertType string

const (
	AlertTypeChannelDown      AlertType = "channel_down"
	AlertTypeChannelHighError AlertType = "channel_high_error"
	AlertTypeLowBalance       AlertType = "low_balance"
	AlertTypeHighLatency      AlertType = "high_latency"
	AlertTypeSystemError      AlertType = "system_error"
	AlertTypeQuotaWarning     AlertType = "quota_warning"
	AlertTypeQuotaExceeded    AlertType = "quota_exceeded"
)

// Alert 告警模型
type Alert struct {
	ID         uuid.UUID     `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Type       AlertType     `gorm:"type:varchar(50);not null;index" json:"type"`
	Severity   AlertSeverity `gorm:"type:varchar(20);not null" json:"severity"`
	Status     AlertStatus   `gorm:"type:varchar(20);default:'active';index" json:"status"`
	Title      string        `gorm:"type:varchar(200);not null" json:"title"`
	Message    string        `gorm:"type:text;not null" json:"message"`
	Metadata   JSON          `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`
	ResolvedBy *uuid.UUID    `json:"resolved_by,omitempty"`
	CreatedAt  time.Time     `gorm:"autoCreateTime;index" json:"created_at"`
	UpdatedAt  time.Time     `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	ResolvedByUser *User `gorm:"foreignKey:ResolvedBy" json:"resolved_by_user,omitempty"`
}

// TableName 表名
func (Alert) TableName() string {
	return "alerts"
}

// AlertResponse 告警响应结构
type AlertResponse struct {
	ID         uuid.UUID     `json:"id"`
	Type       AlertType     `json:"type"`
	Severity   AlertSeverity `json:"severity"`
	Status     AlertStatus   `json:"status"`
	Title      string        `json:"title"`
	Message    string        `json:"message"`
	Metadata   JSON          `json:"metadata,omitempty"`
	ResolvedAt *time.Time    `json:"resolved_at,omitempty"`
	ResolvedBy *uuid.UUID    `json:"resolved_by,omitempty"`
	CreatedAt  time.Time     `json:"created_at"`
}

// ToResponse 转换为响应结构
func (a *Alert) ToResponse() *AlertResponse {
	return &AlertResponse{
		ID:         a.ID,
		Type:       a.Type,
		Severity:   a.Severity,
		Status:     a.Status,
		Title:      a.Title,
		Message:    a.Message,
		Metadata:   a.Metadata,
		ResolvedAt: a.ResolvedAt,
		ResolvedBy: a.ResolvedBy,
		CreatedAt:  a.CreatedAt,
	}
}

// AlertStats 告警统计
type AlertStats struct {
	TotalAlerts    int64 `json:"total_alerts"`
	ActiveAlerts   int64 `json:"active_alerts"`
	CriticalAlerts int64 `json:"critical_alerts"`
	WarningAlerts  int64 `json:"warning_alerts"`
}
