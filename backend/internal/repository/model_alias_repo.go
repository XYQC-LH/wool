package repository

import (
	"context"
	"errors"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ModelAliasRepository 模型别名仓库
type ModelAliasRepository interface {
	GetByAlias(ctx context.Context, alias string) (*model.ModelAlias, error)
	ListEnabled(ctx context.Context) ([]*model.ModelAlias, error)
	ListByTargetModel(ctx context.Context, targetModel string) ([]*model.ModelAlias, error)
}

type modelAliasRepository struct {
	db *gorm.DB
}

func NewModelAliasRepository(db *gorm.DB) ModelAliasRepository {
	return &modelAliasRepository{db: db}
}

func (r *modelAliasRepository) GetByAlias(ctx context.Context, alias string) (*model.ModelAlias, error) {
	var item model.ModelAlias
	err := r.db.WithContext(ctx).
		Where("alias = ? AND enabled = TRUE", alias).
		First(&item).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (r *modelAliasRepository) ListEnabled(ctx context.Context) ([]*model.ModelAlias, error) {
	var items []*model.ModelAlias
	if err := r.db.WithContext(ctx).
		Where("enabled = TRUE").
		Order("alias ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (r *modelAliasRepository) ListByTargetModel(ctx context.Context, targetModel string) ([]*model.ModelAlias, error) {
	var items []*model.ModelAlias
	if err := r.db.WithContext(ctx).
		Where("target_model = ? AND enabled = TRUE", targetModel).
		Order("alias ASC").
		Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}
