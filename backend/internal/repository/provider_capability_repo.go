package repository

import (
	"context"
	"errors"
	"strings"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

type ProviderCapabilityRepository interface {
	Create(ctx context.Context, capability *model.ProviderCapability) error
	Update(ctx context.Context, capability *model.ProviderCapability) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ProviderCapability, error)

	GetByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error)
	GetByProviderAll(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error)
	GetByProviderAndOperation(ctx context.Context, providerID uint, operation string) (*model.ProviderCapability, error)

	List(ctx context.Context, providerID uint, operation string, isEnabled *bool, page, pageSize int) ([]*model.ProviderCapability, int64, error)
	BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error
	GetSummary(ctx context.Context, providerID uint) (*model.ProviderCapabilitySummary, error)
}

type providerCapabilityRepository struct {
	db *gorm.DB
}

func NewProviderCapabilityRepository(db *gorm.DB) ProviderCapabilityRepository {
	return &providerCapabilityRepository{db: db}
}

func (r *providerCapabilityRepository) Create(ctx context.Context, capability *model.ProviderCapability) error {
	return r.db.WithContext(ctx).Create(capability).Error
}

func (r *providerCapabilityRepository) Update(ctx context.Context, capability *model.ProviderCapability) error {
	return r.db.WithContext(ctx).Save(capability).Error
}

func (r *providerCapabilityRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ProviderCapability{}, id).Error
}

func (r *providerCapabilityRepository) GetByID(ctx context.Context, id uint) (*model.ProviderCapability, error) {
	var capability model.ProviderCapability
	err := r.db.WithContext(ctx).First(&capability, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &capability, nil
}

func (r *providerCapabilityRepository) GetByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error) {
	var caps []*model.ProviderCapability
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND is_enabled = ?", providerID, true).
		Find(&caps).Error
	return caps, err
}

func (r *providerCapabilityRepository) GetByProviderAll(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error) {
	var caps []*model.ProviderCapability
	err := r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Order("operation ASC").
		Find(&caps).Error
	return caps, err
}

func (r *providerCapabilityRepository) GetByProviderAndOperation(ctx context.Context, providerID uint, operation string) (*model.ProviderCapability, error) {
	var capability model.ProviderCapability
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND operation = ?", providerID, model.NormalizeOperation(operation)).
		First(&capability).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &capability, nil
}

func (r *providerCapabilityRepository) List(ctx context.Context, providerID uint, operation string, isEnabled *bool, page, pageSize int) ([]*model.ProviderCapability, int64, error) {
	var list []*model.ProviderCapability
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ProviderCapability{})
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}
	operation = model.NormalizeOperation(strings.TrimSpace(operation))
	if operation != "" {
		query = query.Where("operation = ?", operation)
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

	err := query.Order("provider_id ASC, operation ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error
	return list, total, err
}

func (r *providerCapabilityRepository) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.ProviderCapability{}).
		Where("id IN ?", ids).
		Update("is_enabled", enabled).Error
}

func (r *providerCapabilityRepository) GetSummary(ctx context.Context, providerID uint) (*model.ProviderCapabilitySummary, error) {
	summary := &model.ProviderCapabilitySummary{
		ProviderID:         providerID,
		OperationBreakdown: map[string]int{},
	}

	buildQuery := func() *gorm.DB {
		query := r.db.WithContext(ctx).Model(&model.ProviderCapability{})
		if providerID > 0 {
			query = query.Where("provider_id = ?", providerID)
		}
		return query
	}

	if err := buildQuery().Count(&summary.Total).Error; err != nil {
		return nil, err
	}
	if err := buildQuery().Where("is_enabled = ?", true).Count(&summary.Enabled).Error; err != nil {
		return nil, err
	}
	if err := buildQuery().Where("is_enabled = ?", false).Count(&summary.Disabled).Error; err != nil {
		return nil, err
	}

	var rows []struct {
		Operation string
		Count     int
	}
	if err := buildQuery().
		Select("operation, COUNT(*) as count").
		Group("operation").
		Scan(&rows).Error; err != nil {
		return nil, err
	}
	for _, row := range rows {
		op := model.NormalizeOperation(strings.TrimSpace(row.Operation))
		summary.OperationBreakdown[op] = row.Count
	}

	return summary, nil
}
