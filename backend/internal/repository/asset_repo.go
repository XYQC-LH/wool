package repository

import (
	"errors"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AssetRepository 资源仓储接口
type AssetRepository interface {
	Create(asset *model.Asset) error
	GetByID(id uuid.UUID) (*model.Asset, error)
	ListSite(page, pageSize int) ([]*model.Asset, int64, error)
	ListByUserID(userID uuid.UUID, page, pageSize int) ([]*model.Asset, int64, error)
}

type assetRepository struct {
	db *gorm.DB
}

func NewAssetRepository(db *gorm.DB) AssetRepository {
	return &assetRepository{db: db}
}

func (r *assetRepository) Create(asset *model.Asset) error {
	return r.db.Create(asset).Error
}

func (r *assetRepository) GetByID(id uuid.UUID) (*model.Asset, error) {
	var asset model.Asset
	if err := r.db.Where("id = ? AND status <> ?", id, model.AssetStatusDeleted).First(&asset).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &asset, nil
}

func (r *assetRepository) ListSite(page, pageSize int) ([]*model.Asset, int64, error) {
	return r.list(func(db *gorm.DB) *gorm.DB {
		return db.Where("owner_type = ? AND status <> ?", model.AssetOwnerTypeSite, model.AssetStatusDeleted)
	}, page, pageSize)
}

func (r *assetRepository) ListByUserID(userID uuid.UUID, page, pageSize int) ([]*model.Asset, int64, error) {
	return r.list(func(db *gorm.DB) *gorm.DB {
		return db.Where("owner_type = ? AND user_id = ? AND status <> ?", model.AssetOwnerTypeUser, userID, model.AssetStatusDeleted)
	}, page, pageSize)
}

func (r *assetRepository) list(scoped func(db *gorm.DB) *gorm.DB, page, pageSize int) ([]*model.Asset, int64, error) {
	var assets []*model.Asset
	var total int64

	query := scoped(r.db.Model(&model.Asset{}))
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&assets).Error; err != nil {
		return nil, 0, err
	}

	return assets, total, nil
}
