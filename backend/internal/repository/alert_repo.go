package repository

import (
	"time"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AlertRepository Alert 仓库接口
type AlertRepository interface {
	Create(alert *model.Alert) error
	GetByID(id uuid.UUID) (*model.Alert, error)
	List(page, pageSize int, filters map[string]interface{}) ([]*model.Alert, int64, error)
	Update(alert *model.Alert) error
	Resolve(id uuid.UUID, resolvedBy uuid.UUID) error
	GetStats() (*model.AlertStats, error)
	GetActiveAlerts() ([]*model.Alert, error)
}

// alertRepository Alert 仓库实现
type alertRepository struct {
	db *gorm.DB
}

// NewAlertRepository 创建 Alert 仓库
func NewAlertRepository(db *gorm.DB) AlertRepository {
	return &alertRepository{db: db}
}

// Create 创建告警
func (r *alertRepository) Create(alert *model.Alert) error {
	return r.db.Create(alert).Error
}

// GetByID 根据 ID 获取告警
func (r *alertRepository) GetByID(id uuid.UUID) (*model.Alert, error) {
	var alert model.Alert
	if err := r.db.Where("id = ?", id).First(&alert).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &alert, nil
}

// List 获取告警列表
func (r *alertRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.Alert, int64, error) {
	var alerts []*model.Alert
	var total int64

	query := r.db.Model(&model.Alert{})

	// 应用过滤条件
	if alertType, ok := filters["type"]; ok && alertType != "" {
		query = query.Where("type = ?", alertType)
	}
	if severity, ok := filters["severity"]; ok && severity != "" {
		query = query.Where("severity = ?", severity)
	}
	if status, ok := filters["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&alerts).Error; err != nil {
		return nil, 0, err
	}

	return alerts, total, nil
}

// Update 更新告警
func (r *alertRepository) Update(alert *model.Alert) error {
	return r.db.Save(alert).Error
}

// Resolve 解决告警
func (r *alertRepository) Resolve(id uuid.UUID, resolvedBy uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.Alert{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":      model.AlertStatusResolved,
			"resolved_at": now,
			"resolved_by": resolvedBy,
		}).Error
}

// GetStats 获取告警统计
func (r *alertRepository) GetStats() (*model.AlertStats, error) {
	var stats model.AlertStats

	// 统计总数
	if err := r.db.Model(&model.Alert{}).Count(&stats.TotalAlerts).Error; err != nil {
		return nil, err
	}

	// 统计活跃告警
	if err := r.db.Model(&model.Alert{}).Where("status = ?", model.AlertStatusActive).Count(&stats.ActiveAlerts).Error; err != nil {
		return nil, err
	}

	// 统计严重告警
	if err := r.db.Model(&model.Alert{}).
		Where("severity = ? AND status = ?", model.AlertSeverityCritical, model.AlertStatusActive).
		Count(&stats.CriticalAlerts).Error; err != nil {
		return nil, err
	}

	// 统计警告告警
	if err := r.db.Model(&model.Alert{}).
		Where("severity = ? AND status = ?", model.AlertSeverityWarning, model.AlertStatusActive).
		Count(&stats.WarningAlerts).Error; err != nil {
		return nil, err
	}

	return &stats, nil
}

// GetActiveAlerts 获取活跃告警
func (r *alertRepository) GetActiveAlerts() ([]*model.Alert, error) {
	var alerts []*model.Alert
	if err := r.db.Where("status = ?", model.AlertStatusActive).
		Order("created_at DESC").
		Find(&alerts).Error; err != nil {
		return nil, err
	}
	return alerts, nil
}
