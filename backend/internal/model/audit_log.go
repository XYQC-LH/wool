package model

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLog 管理员审计日志
type AuditLog struct {
	ID          uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	ActorID     *uuid.UUID `gorm:"type:uuid;index" json:"actor_id,omitempty"`
	ActorRole   string    `gorm:"type:varchar(32)" json:"actor_role,omitempty"`
	Action      string    `gorm:"type:varchar(128);not null;index" json:"action"`
	Resource    string    `gorm:"type:varchar(64);index" json:"resource,omitempty"`
	Method      string    `gorm:"type:varchar(16);not null;index" json:"method"`
	Path        string    `gorm:"type:varchar(255);not null;index" json:"path"`
	StatusCode  int       `gorm:"index" json:"status_code"`
	Success     bool      `gorm:"index" json:"success"`
	RequestIP   string    `gorm:"type:varchar(64)" json:"request_ip,omitempty"`
	UserAgent   string    `gorm:"type:text" json:"user_agent,omitempty"`
	QueryParams string    `gorm:"type:text" json:"query_params,omitempty"`
	RequestBody string    `gorm:"type:text" json:"request_body,omitempty"`
	ErrorMsg    string    `gorm:"type:text" json:"error_msg,omitempty"`
	Metadata    JSON      `gorm:"type:jsonb;default:'{}'" json:"metadata,omitempty"`
	CreatedAt   time.Time `gorm:"autoCreateTime;index" json:"created_at"`

	Actor *User `gorm:"foreignKey:ActorID" json:"actor,omitempty"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}

func (a *AuditLog) BeforeCreate(tx *gorm.DB) error {
	if a.ID == uuid.Nil {
		a.ID = uuid.New()
	}
	return nil
}

// AuditLogResponse 审计日志响应
type AuditLogResponse struct {
	ID          uuid.UUID  `json:"id"`
	ActorID     *uuid.UUID `json:"actor_id,omitempty"`
	ActorName   string     `json:"actor_name,omitempty"`
	ActorRole   string     `json:"actor_role,omitempty"`
	Action      string     `json:"action"`
	Resource    string     `json:"resource,omitempty"`
	Method      string     `json:"method"`
	Path        string     `json:"path"`
	StatusCode  int        `json:"status_code"`
	Success     bool       `json:"success"`
	RequestIP   string     `json:"request_ip,omitempty"`
	UserAgent   string     `json:"user_agent,omitempty"`
	QueryParams string     `json:"query_params,omitempty"`
	RequestBody string     `json:"request_body,omitempty"`
	ErrorMsg    string     `json:"error_msg,omitempty"`
	Metadata    JSON       `json:"metadata,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (a *AuditLog) ToResponse() *AuditLogResponse {
	resp := &AuditLogResponse{
		ID:          a.ID,
		ActorID:     a.ActorID,
		ActorRole:   a.ActorRole,
		Action:      a.Action,
		Resource:    a.Resource,
		Method:      a.Method,
		Path:        a.Path,
		StatusCode:  a.StatusCode,
		Success:     a.Success,
		RequestIP:   a.RequestIP,
		UserAgent:   a.UserAgent,
		QueryParams: a.QueryParams,
		RequestBody: a.RequestBody,
		ErrorMsg:    a.ErrorMsg,
		Metadata:    a.Metadata,
		CreatedAt:   a.CreatedAt,
	}

	if a.Actor != nil {
		resp.ActorName = a.Actor.Username
	}

	return resp
}
