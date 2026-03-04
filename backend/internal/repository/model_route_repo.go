package repository

import (
	"context"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

type ModelRouteRepository interface {
	Create(ctx context.Context, route *model.ModelRoute) error
	Update(ctx context.Context, route *model.ModelRoute) error
	Delete(ctx context.Context, id uint) error
	GetByRouteKey(ctx context.Context, operation string, modelID string) ([]*model.ModelRoute, error)
}

type modelRouteRepository struct {
	db *gorm.DB
}

func NewModelRouteRepository(db *gorm.DB) ModelRouteRepository {
	return &modelRouteRepository{db: db}
}

func (r *modelRouteRepository) Create(ctx context.Context, route *model.ModelRoute) error {
	return r.db.WithContext(ctx).Create(route).Error
}

func (r *modelRouteRepository) Update(ctx context.Context, route *model.ModelRoute) error {
	return r.db.WithContext(ctx).Save(route).Error
}

func (r *modelRouteRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ModelRoute{}, id).Error
}

func (r *modelRouteRepository) GetByRouteKey(ctx context.Context, operation string, modelID string) ([]*model.ModelRoute, error) {
	var routes []*model.ModelRoute
	err := r.db.WithContext(ctx).
		Where("operation = ? AND model_id = ? AND is_enabled = ?", operation, modelID, true).
		Order("priority ASC").
		Find(&routes).Error
	return routes, err
}
