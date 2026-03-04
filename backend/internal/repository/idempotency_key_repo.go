package repository

import (
	"context"
	"errors"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// IdempotencyKeyRepository 幂等键仓库接口
type IdempotencyKeyRepository interface {
	Create(ctx context.Context, key *model.IdempotencyKey) error
	GetByUserOperationKey(ctx context.Context, userID uuid.UUID, operation string, key string) (*model.IdempotencyKey, error)
}

type idempotencyKeyRepository struct {
	db *gorm.DB
}

func NewIdempotencyKeyRepository(db *gorm.DB) IdempotencyKeyRepository {
	return &idempotencyKeyRepository{db: db}
}

func (r *idempotencyKeyRepository) Create(ctx context.Context, key *model.IdempotencyKey) error {
	return r.db.WithContext(ctx).Create(key).Error
}

func (r *idempotencyKeyRepository) GetByUserOperationKey(ctx context.Context, userID uuid.UUID, operation string, key string) (*model.IdempotencyKey, error) {
	var record model.IdempotencyKey
	err := r.db.WithContext(ctx).
		Where("user_id = ? AND operation = ? AND key = ?", userID, model.NormalizeOperation(operation), key).
		First(&record).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &record, nil
}
