package repository

import (
	"context"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

type ProviderCapabilityRepository interface {
	GetByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error)
}

type providerCapabilityRepository struct {
	db *gorm.DB
}

func NewProviderCapabilityRepository(db *gorm.DB) ProviderCapabilityRepository {
	return &providerCapabilityRepository{db: db}
}

func (r *providerCapabilityRepository) GetByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error) {
	var caps []*model.ProviderCapability
	err := r.db.WithContext(ctx).
		Where("provider_id = ? AND is_enabled = ?", providerID, true).
		Find(&caps).Error
	return caps, err
}
