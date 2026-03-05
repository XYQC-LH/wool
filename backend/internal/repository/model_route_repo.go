package repository

import (
	"context"
	"errors"
	"strings"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

type ModelRouteRepository interface {
	Create(ctx context.Context, route *model.ModelRoute) error
	Update(ctx context.Context, route *model.ModelRoute) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ModelRoute, error)
	GetByModelAndProvider(ctx context.Context, operation string, modelID string, providerID uint) (*model.ModelRoute, error)
	List(ctx context.Context, operation string, modelID string, providerID uint, isEnabled *bool, page, pageSize int) ([]*model.ModelRoute, int64, error)
	BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error
	BatchUpdatePriority(ctx context.Context, updates map[uint]int) error
	GetStats(ctx context.Context, operation string, modelID string) (*model.ModelRouteStats, error)
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

func (r *modelRouteRepository) GetByID(ctx context.Context, id uint) (*model.ModelRoute, error) {
	var route model.ModelRoute
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Model").
		Preload("Provider.Channel").
		First(&route, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &route, nil
}

func (r *modelRouteRepository) GetByModelAndProvider(ctx context.Context, operation string, modelID string, providerID uint) (*model.ModelRoute, error) {
	var route model.ModelRoute
	err := r.db.WithContext(ctx).
		Where("operation = ? AND model_id = ? AND provider_id = ?", model.NormalizeOperation(operation), strings.TrimSpace(modelID), providerID).
		First(&route).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &route, nil
}

func (r *modelRouteRepository) List(ctx context.Context, operation string, modelID string, providerID uint, isEnabled *bool, page, pageSize int) ([]*model.ModelRoute, int64, error) {
	var list []*model.ModelRoute
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ModelRoute{})
	operation = strings.TrimSpace(operation)
	modelID = strings.TrimSpace(modelID)
	if operation != "" {
		query = query.Where("operation = ?", model.NormalizeOperation(operation))
	}
	if modelID != "" {
		query = query.Where("model_id = ?", modelID)
	}
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	if isEnabled != nil {
		query = query.Where("is_enabled = ?", *isEnabled)
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
		Preload("Provider").
		Preload("Provider.Model").
		Preload("Provider.Channel").
		Order("priority ASC, id ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *modelRouteRepository) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.ModelRoute{}).
		Where("id IN ?", ids).
		Update("is_enabled", enabled).Error
}

func (r *modelRouteRepository) BatchUpdatePriority(ctx context.Context, updates map[uint]int) error {
	if len(updates) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for id, priority := range updates {
			if err := tx.Model(&model.ModelRoute{}).Where("id = ?", id).Update("priority", priority).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

func (r *modelRouteRepository) GetStats(ctx context.Context, operation string, modelID string) (*model.ModelRouteStats, error) {
	stats := &model.ModelRouteStats{}
	operation = strings.TrimSpace(operation)
	modelID = strings.TrimSpace(modelID)

	buildQuery := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&model.ModelRoute{})
		if operation != "" {
			query = query.Where("operation = ?", model.NormalizeOperation(operation))
		}
		if modelID != "" {
			query = query.Where("model_id = ?", modelID)
		}
		return query
	}

	if err := buildQuery().Count(&stats.TotalRoutes).Error; err != nil {
		return nil, err
	}
	if err := buildQuery().Where("is_enabled = ?", true).Count(&stats.EnabledRoutes).Error; err != nil {
		return nil, err
	}
	if err := buildQuery().Where("is_enabled = ?", false).Count(&stats.DisabledRoutes).Error; err != nil {
		return nil, err
	}

	var distinctModels int64
	if err := buildQuery().Distinct("model_id").Count(&distinctModels).Error; err != nil {
		return nil, err
	}
	stats.DistinctModels = distinctModels

	var distinctProviders int64
	if err := buildQuery().Distinct("provider_id").Count(&distinctProviders).Error; err != nil {
		return nil, err
	}
	stats.DistinctProviders = distinctProviders

	return stats, nil
}

func (r *modelRouteRepository) GetByRouteKey(ctx context.Context, operation string, modelID string) ([]*model.ModelRoute, error) {
	var routes []*model.ModelRoute
	operation = model.NormalizeOperation(operation)
	modelID = strings.TrimSpace(modelID)
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Model").
		Preload("Provider.Channel").
		Where("operation = ? AND model_id = ? AND is_enabled = ?", operation, modelID, true).
		Order("priority ASC, id ASC").
		Find(&routes).Error
	return routes, err
}
