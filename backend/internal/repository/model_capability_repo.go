package repository

import (
	"context"
	"errors"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ModelCapabilityRepository 模型能力仓库接口
type ModelCapabilityRepository interface {
	Create(ctx context.Context, capability *model.ModelCapability) error
	Update(ctx context.Context, capability *model.ModelCapability) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ModelCapability, error)
	GetByModelAndOperation(ctx context.Context, modelID string, operation string) (*model.ModelCapability, error)
	List(ctx context.Context, modelID string, operation string, page, pageSize int) ([]*model.ModelCapability, int64, error)
}

type modelCapabilityRepository struct {
	db *gorm.DB
}

func NewModelCapabilityRepository(db *gorm.DB) ModelCapabilityRepository {
	return &modelCapabilityRepository{db: db}
}

func (r *modelCapabilityRepository) Create(ctx context.Context, capability *model.ModelCapability) error {
	return r.db.WithContext(ctx).Create(capability).Error
}

func (r *modelCapabilityRepository) Update(ctx context.Context, capability *model.ModelCapability) error {
	return r.db.WithContext(ctx).Save(capability).Error
}

func (r *modelCapabilityRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ModelCapability{}, id).Error
}

func (r *modelCapabilityRepository) GetByID(ctx context.Context, id uint) (*model.ModelCapability, error) {
	var capability model.ModelCapability
	err := r.db.WithContext(ctx).
		Preload("Model").
		First(&capability, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &capability, nil
}

func (r *modelCapabilityRepository) GetByModelAndOperation(ctx context.Context, modelID string, operation string) (*model.ModelCapability, error) {
	var capability model.ModelCapability
	err := r.db.WithContext(ctx).
		Where("model_id = ? AND operation = ?", modelID, model.NormalizeOperation(operation)).
		First(&capability).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &capability, nil
}

func (r *modelCapabilityRepository) List(ctx context.Context, modelID string, operation string, page, pageSize int) ([]*model.ModelCapability, int64, error) {
	var list []*model.ModelCapability
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ModelCapability{})
	if modelID != "" {
		query = query.Where("model_id = ?", modelID)
	}
	if operation != "" {
		query = query.Where("operation = ?", model.NormalizeOperation(operation))
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	err := query.
		Preload("Model").
		Order("model_id ASC, operation ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}
