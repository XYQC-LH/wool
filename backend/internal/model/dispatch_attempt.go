package model

import "time"

// DispatchAttempt 调度尝试审计记录
// 用于记录 Constraints / HealthGate / Scoring / Cascade 的执行结果
type DispatchAttempt struct {
	ID uint `gorm:"primaryKey;autoIncrement" json:"id"`

	RequestID string `gorm:"type:varchar(64);index:idx_dispatch_request" json:"request_id,omitempty"`

	Operation        string `gorm:"type:varchar(50);not null;index:idx_dispatch_route,priority:1" json:"operation"`
	RequestedModelID string `gorm:"type:varchar(100);not null;index:idx_dispatch_route,priority:2" json:"requested_model_id"`
	ResolvedModelID  string `gorm:"type:varchar(100);not null;index" json:"resolved_model_id"`
	RouteModelID     string `gorm:"type:varchar(100);not null;index" json:"route_model_id"`

	ProviderID         uint  `gorm:"not null;index" json:"provider_id"`
	ProviderInstanceID *uint `gorm:"index" json:"provider_instance_id,omitempty"`

	AttemptNo int    `gorm:"not null;default:0" json:"attempt_no"`
	Strategy  string `gorm:"type:varchar(30);not null;default:'cost_first'" json:"strategy"`
	Stage     string `gorm:"type:varchar(20);not null;default:'cascade'" json:"stage"`
	Decision  string `gorm:"type:varchar(20);not null;default:'selected'" json:"decision"`

	Success      bool   `gorm:"not null;default:false;index" json:"success"`
	ErrorType    string `gorm:"type:varchar(30)" json:"error_type,omitempty"`
	ErrorMessage string `gorm:"type:text" json:"error_message,omitempty"`
	LatencyMs    int64  `gorm:"not null;default:0" json:"latency_ms"`

	Metadata JSON `gorm:"type:jsonb" json:"metadata,omitempty"`

	CreatedAt time.Time `gorm:"autoCreateTime;index" json:"created_at"`
}

func (DispatchAttempt) TableName() string {
	return "dispatch_attempts"
}
