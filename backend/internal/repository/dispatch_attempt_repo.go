package repository

import (
	"context"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// DispatchAttemptRepository 调度尝试审计仓库
type DispatchAttemptRepository interface {
	Create(ctx context.Context, attempt *model.DispatchAttempt) error
	BatchCreate(ctx context.Context, attempts []*model.DispatchAttempt) error
	ListByRequestID(ctx context.Context, requestID string, limit int) ([]*model.DispatchAttempt, error)
}

type dispatchAttemptRepository struct {
	db *gorm.DB
}

func NewDispatchAttemptRepository(db *gorm.DB) DispatchAttemptRepository {
	return &dispatchAttemptRepository{db: db}
}

func (r *dispatchAttemptRepository) Create(ctx context.Context, attempt *model.DispatchAttempt) error {
	if attempt == nil {
		return nil
	}
	return r.db.WithContext(ctx).Create(attempt).Error
}

func (r *dispatchAttemptRepository) BatchCreate(ctx context.Context, attempts []*model.DispatchAttempt) error {
	if len(attempts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(attempts, 100).Error
}

func (r *dispatchAttemptRepository) ListByRequestID(ctx context.Context, requestID string, limit int) ([]*model.DispatchAttempt, error) {
	if limit <= 0 {
		limit = 100
	}

	var attempts []*model.DispatchAttempt
	err := r.db.WithContext(ctx).
		Where("request_id = ?", requestID).
		Order("id ASC").
		Limit(limit).
		Find(&attempts).Error
	return attempts, err
}
