package repository

import (
	"context"
	"errors"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ProviderRateLimitRuleRepository 多模态限流规则仓库接口
type ProviderRateLimitRuleRepository interface {
	Create(ctx context.Context, rule *model.ProviderRateLimitRule) error
	Update(ctx context.Context, rule *model.ProviderRateLimitRule) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ProviderRateLimitRule, error)
	GetByScopeOperationUnitWindow(ctx context.Context, scope string, providerID uint, instanceID uint, operation string, unit string, windowSeconds int) (*model.ProviderRateLimitRule, error)
	GetByProviderOperationUnitWindow(ctx context.Context, providerID uint, operation string, unit string, windowSeconds int) (*model.ProviderRateLimitRule, error)
	List(ctx context.Context, scope string, providerID uint, instanceID uint, operation string, page, pageSize int) ([]*model.ProviderRateLimitRule, int64, error)
	ListEnabledByScope(ctx context.Context, scope string, providerID uint, instanceID uint, operation string) ([]*model.ProviderRateLimitRule, error)
	ListEnabledByProviderOperation(ctx context.Context, providerID uint, operation string) ([]*model.ProviderRateLimitRule, error)
}

type providerRateLimitRuleRepository struct {
	db *gorm.DB
}

func NewProviderRateLimitRuleRepository(db *gorm.DB) ProviderRateLimitRuleRepository {
	return &providerRateLimitRuleRepository{db: db}
}

func (r *providerRateLimitRuleRepository) Create(ctx context.Context, rule *model.ProviderRateLimitRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *providerRateLimitRuleRepository) Update(ctx context.Context, rule *model.ProviderRateLimitRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *providerRateLimitRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ProviderRateLimitRule{}, id).Error
}

func (r *providerRateLimitRuleRepository) GetByID(ctx context.Context, id uint) (*model.ProviderRateLimitRule, error) {
	var rule model.ProviderRateLimitRule
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Model").
		Preload("Provider.Channel").
		First(&rule, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *providerRateLimitRuleRepository) GetByScopeOperationUnitWindow(ctx context.Context, scope string, providerID uint, instanceID uint, operation string, unit string, windowSeconds int) (*model.ProviderRateLimitRule, error) {
	var rule model.ProviderRateLimitRule
	scope = model.NormalizeRateLimitScope(scope)
	query := r.db.WithContext(ctx).
		Where("scope = ? AND provider_id = ? AND operation = ? AND unit = ? AND window_seconds = ?",
			scope,
			providerID,
			model.NormalizeOperation(operation),
			unit,
			windowSeconds,
		)
	if scope == model.RateLimitScopeInstance {
		query = query.Where("instance_id = ?", instanceID)
	} else {
		query = query.Where("instance_id IS NULL")
	}
	err := query.First(&rule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *providerRateLimitRuleRepository) GetByProviderOperationUnitWindow(ctx context.Context, providerID uint, operation string, unit string, windowSeconds int) (*model.ProviderRateLimitRule, error) {
	return r.GetByScopeOperationUnitWindow(ctx, model.RateLimitScopeProvider, providerID, 0, operation, unit, windowSeconds)
}

func (r *providerRateLimitRuleRepository) List(ctx context.Context, scope string, providerID uint, instanceID uint, operation string, page, pageSize int) ([]*model.ProviderRateLimitRule, int64, error) {
	var list []*model.ProviderRateLimitRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ProviderRateLimitRule{})
	if scope != "" {
		query = query.Where("scope = ?", model.NormalizeRateLimitScope(scope))
	}
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	if instanceID > 0 {
		query = query.Where("instance_id = ?", instanceID)
	} else if scope != "" && model.NormalizeRateLimitScope(scope) == model.RateLimitScopeProvider {
		query = query.Where("instance_id IS NULL")
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
		Preload("Provider").
		Preload("Provider.Model").
		Preload("Provider.Channel").
		Order("scope ASC, provider_id ASC, instance_id ASC, operation ASC, unit ASC, window_seconds ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *providerRateLimitRuleRepository) ListEnabledByScope(ctx context.Context, scope string, providerID uint, instanceID uint, operation string) ([]*model.ProviderRateLimitRule, error) {
	var list []*model.ProviderRateLimitRule
	scope = model.NormalizeRateLimitScope(scope)
	if providerID == 0 {
		return list, nil
	}

	query := r.db.WithContext(ctx).
		Where("scope = ? AND provider_id = ? AND operation = ? AND enabled = ?", scope, providerID, model.NormalizeOperation(operation), true)
	if scope == model.RateLimitScopeInstance {
		if instanceID == 0 {
			return list, nil
		}
		query = query.Where("instance_id = ?", instanceID)
	} else {
		query = query.Where("instance_id IS NULL")
	}
	err := query.Order("unit ASC, window_seconds ASC").Find(&list).Error
	return list, err
}

func (r *providerRateLimitRuleRepository) ListEnabledByProviderOperation(ctx context.Context, providerID uint, operation string) ([]*model.ProviderRateLimitRule, error) {
	return r.ListEnabledByScope(ctx, model.RateLimitScopeProvider, providerID, 0, operation)
}
