package repository

import (
	"time"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditLogRepository 审计日志仓库
type AuditLogRepository interface {
	Create(log *model.AuditLog) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLog, int64, error)
}

type auditLogRepository struct {
	db *gorm.DB
}

func NewAuditLogRepository(db *gorm.DB) AuditLogRepository {
	return &auditLogRepository{db: db}
}

func (r *auditLogRepository) Create(log *model.AuditLog) error {
	return r.db.Create(log).Error
}

func (r *auditLogRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLog, int64, error) {
	var logs []*model.AuditLog
	var total int64

	query := r.db.Model(&model.AuditLog{}).Preload("Actor")

	if actorID, ok := filters["actor_id"].(uuid.UUID); ok && actorID != uuid.Nil {
		query = query.Where("actor_id = ?", actorID)
	}
	if action, ok := filters["action"].(string); ok && action != "" {
		query = query.Where("action = ?", action)
	}
	if resource, ok := filters["resource"].(string); ok && resource != "" {
		query = query.Where("resource = ?", resource)
	}
	if method, ok := filters["method"].(string); ok && method != "" {
		query = query.Where("method = ?", method)
	}
	if statusCode, ok := filters["status_code"].(int); ok && statusCode > 0 {
		query = query.Where("status_code = ?", statusCode)
	}
	if success, ok := filters["success"].(bool); ok {
		query = query.Where("success = ?", success)
	}
	if startTime, ok := filters["start_time"].(time.Time); ok && !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if endTime, ok := filters["end_time"].(time.Time); ok && !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}
