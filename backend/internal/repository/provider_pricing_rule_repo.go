package repository

import (
	"context"
	"errors"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ProviderPricingRuleRepository 多模态计费规则仓库接口
type ProviderPricingRuleRepository interface {
	Create(ctx context.Context, rule *model.ProviderPricingRule) error
	Update(ctx context.Context, rule *model.ProviderPricingRule) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ProviderPricingRule, error)
	GetByProviderOperationUnit(ctx context.Context, providerID uint, operation string, unit string) (*model.ProviderPricingRule, error)
	List(ctx context.Context, providerID uint, operation string, page, pageSize int) ([]*model.ProviderPricingRule, int64, error)
}

type providerPricingRuleRepository struct {
	db *gorm.DB
}

func NewProviderPricingRuleRepository(db *gorm.DB) ProviderPricingRuleRepository {
	return &providerPricingRuleRepository{db: db}
}

func (r *providerPricingRuleRepository) Create(ctx context.Context, rule *model.ProviderPricingRule) error {
	return r.db.WithContext(ctx).Create(rule).Error
}

func (r *providerPricingRuleRepository) Update(ctx context.Context, rule *model.ProviderPricingRule) error {
	return r.db.WithContext(ctx).Save(rule).Error
}

func (r *providerPricingRuleRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ProviderPricingRule{}, id).Error
}

func (r *providerPricingRuleRepository) GetByID(ctx context.Context, id uint) (*model.ProviderPricingRule, error) {
	var rule model.ProviderPricingRule
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

func (r *providerPricingRuleRepository) GetByProviderOperationUnit(ctx context.Context, providerID uint, operation string, unit string) (*model.ProviderPricingRule, error) {
	var rule model.ProviderPricingRule
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND operation = ? AND unit = ?", providerID, model.NormalizeOperation(operation), unit).
		First(&rule).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &rule, nil
}

func (r *providerPricingRuleRepository) List(ctx context.Context, providerID uint, operation string, page, pageSize int) ([]*model.ProviderPricingRule, int64, error) {
	var list []*model.ProviderPricingRule
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ProviderPricingRule{})
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
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
		Order("provider_id ASC, operation ASC, unit ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}
