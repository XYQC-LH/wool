package model

import (
	"time"
)

// InstanceStatus 实例状态
type InstanceStatus string

const (
	InstanceStatusActive   InstanceStatus = "active"
	InstanceStatusDisabled InstanceStatus = "disabled"
	InstanceStatusCooling  InstanceStatus = "cooling"
)

// InstanceType 实例类型
type InstanceType string

const (
	InstanceTypeAPIKey          InstanceType = "api_key"
	InstanceTypeResourceAccount InstanceType = "resource_account"
	InstanceTypeSession         InstanceType = "session"
)

// ProviderInstance 源头实例 - 源头组内的可用执行单元
// 用于支持号池场景，一个 ModelProvider 可以有多个 ProviderInstance
type ProviderInstance struct {
	ID         uint `gorm:"primaryKey;autoIncrement" json:"id"`
	ProviderID uint `gorm:"not null;index" json:"provider_id"`

	// 基本信息
	Name         string       `gorm:"type:varchar(100);not null" json:"name"`
	InstanceType InstanceType `gorm:"type:varchar(20);not null" json:"instance_type"`

	// 关联资源账户（仅号池场景需要）
	ResourceAccountID *uint `gorm:"index" json:"resource_account_id,omitempty"`

	// 组内调度配置
	Weight int            `gorm:"default:1" json:"weight"`
	Status InstanceStatus `gorm:"type:varchar(20);default:'active'" json:"status"`

	// 容量/限流配置（0 表示不限制）
	MaxConcurrency int `gorm:"default:0" json:"max_concurrency"`
	RPMLimit       int `gorm:"default:0" json:"rpm_limit"`
	TPMLimit       int `gorm:"default:0" json:"tpm_limit"`

	// 统计数据
	TotalRequests   int64 `gorm:"default:0" json:"total_requests"`
	SuccessRequests int64 `gorm:"default:0" json:"success_requests"`
	FailedRequests  int64 `gorm:"default:0" json:"failed_requests"`
	TotalLatency    int64 `gorm:"default:0" json:"total_latency"`

	// 时间戳
	CreatedAt time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Provider        *ModelProvider   `gorm:"foreignKey:ProviderID" json:"provider,omitempty"`
	ResourceAccount *ResourceAccount `gorm:"foreignKey:ResourceAccountID" json:"resource_account,omitempty"`
}

// TableName 表名
func (ProviderInstance) TableName() string {
	return "provider_instances"
}

// IsAvailable 检查实例是否可用
func (pi *ProviderInstance) IsAvailable() bool {
	return pi.Status == InstanceStatusActive
}

// GetSuccessRate 获取成功率
func (pi *ProviderInstance) GetSuccessRate() float64 {
	if pi.TotalRequests == 0 {
		return 100.0
	}
	return float64(pi.SuccessRequests) / float64(pi.TotalRequests) * 100
}

// GetAvgLatency 获取平均延迟（毫秒）
func (pi *ProviderInstance) GetAvgLatency() int {
	if pi.TotalRequests == 0 {
		return 0
	}
	return int(pi.TotalLatency / pi.TotalRequests)
}

// ProviderInstanceResponse 源头实例响应结构
type ProviderInstanceResponse struct {
	ID                uint           `json:"id"`
	ProviderID        uint           `json:"provider_id"`
	ProviderName      string         `json:"provider_name,omitempty"`
	Name              string         `json:"name"`
	InstanceType      InstanceType   `json:"instance_type"`
	ResourceAccountID *uint          `json:"resource_account_id,omitempty"`
	AccountName       string         `json:"account_name,omitempty"`
	Weight            int            `json:"weight"`
	Status            InstanceStatus `json:"status"`
	MaxConcurrency    int            `json:"max_concurrency"`
	RPMLimit          int            `json:"rpm_limit"`
	TPMLimit          int            `json:"tpm_limit"`
	TotalRequests     int64          `json:"total_requests"`
	SuccessRequests   int64          `json:"success_requests"`
	FailedRequests    int64          `json:"failed_requests"`
	SuccessRate       float64        `json:"success_rate"`
	AvgLatencyMs      int            `json:"avg_latency_ms"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time      `json:"updated_at"`
}

// ToResponse 转换为响应结构
func (pi *ProviderInstance) ToResponse() *ProviderInstanceResponse {
	resp := &ProviderInstanceResponse{
		ID:                pi.ID,
		ProviderID:        pi.ProviderID,
		Name:              pi.Name,
		InstanceType:      pi.InstanceType,
		ResourceAccountID: pi.ResourceAccountID,
		Weight:            pi.Weight,
		Status:            pi.Status,
		MaxConcurrency:    pi.MaxConcurrency,
		RPMLimit:          pi.RPMLimit,
		TPMLimit:          pi.TPMLimit,
		TotalRequests:     pi.TotalRequests,
		SuccessRequests:   pi.SuccessRequests,
		FailedRequests:    pi.FailedRequests,
		SuccessRate:       pi.GetSuccessRate(),
		AvgLatencyMs:      pi.GetAvgLatency(),
		CreatedAt:         pi.CreatedAt,
		UpdatedAt:         pi.UpdatedAt,
	}

	if pi.Provider != nil && pi.Provider.Channel != nil {
		resp.ProviderName = pi.Provider.Channel.Name + " - " + pi.Provider.UpstreamModelName
	}
	if pi.ResourceAccount != nil {
		resp.AccountName = pi.ResourceAccount.AccountName
	}

	return resp
}

// CreateProviderInstanceRequest 创建源头实例请求
type CreateProviderInstanceRequest struct {
	Name              string         `json:"name" binding:"required,min=1,max=100"`
	InstanceType      InstanceType   `json:"instance_type" binding:"required,oneof=api_key resource_account session"`
	ResourceAccountID *uint          `json:"resource_account_id,omitempty"`
	Weight            int            `json:"weight" binding:"min=0"`
	MaxConcurrency    int            `json:"max_concurrency" binding:"min=0"`
	RPMLimit          int            `json:"rpm_limit" binding:"min=0"`
	TPMLimit          int            `json:"tpm_limit" binding:"min=0"`
	Status            InstanceStatus `json:"status,omitempty" binding:"omitempty,oneof=active disabled cooling"`
}

// UpdateProviderInstanceRequest 更新源头实例请求
type UpdateProviderInstanceRequest struct {
	Name              *string         `json:"name,omitempty" binding:"omitempty,min=1,max=100"`
	InstanceType      *InstanceType   `json:"instance_type,omitempty" binding:"omitempty,oneof=api_key resource_account session"`
	ResourceAccountID *uint           `json:"resource_account_id,omitempty"`
	Weight            *int            `json:"weight,omitempty" binding:"omitempty,min=0"`
	MaxConcurrency    *int            `json:"max_concurrency,omitempty" binding:"omitempty,min=0"`
	RPMLimit          *int            `json:"rpm_limit,omitempty" binding:"omitempty,min=0"`
	TPMLimit          *int            `json:"tpm_limit,omitempty" binding:"omitempty,min=0"`
	Status            *InstanceStatus `json:"status,omitempty" binding:"omitempty,oneof=active disabled cooling"`
}

// ProviderInstanceQueryParams 源头实例查询参数
type ProviderInstanceQueryParams struct {
	ProviderID   uint           `form:"provider_id"`
	Status       InstanceStatus `form:"status"`
	InstanceType InstanceType   `form:"instance_type"`
	Page         int            `form:"page"`
	PageSize     int            `form:"page_size"`
}

// ProviderInstanceStats 源头实例统计
type ProviderInstanceStats struct {
	TotalInstances    int64   `json:"total_instances"`
	ActiveInstances   int64   `json:"active_instances"`
	DisabledInstances int64   `json:"disabled_instances"`
	CoolingInstances  int64   `json:"cooling_instances"`
	AvgSuccessRate    float64 `json:"avg_success_rate"`
}
