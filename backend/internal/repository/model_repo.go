package repository

import (
	"errors"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ModelRepository Model 仓库接口
type ModelRepository interface {
	Create(m *model.Model) error
	GetByID(id string) (*model.Model, error)
	Update(m *model.Model) error
	Delete(id string) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.Model, int64, error)
	ListAll() ([]*model.Model, error)
	ListEnabled() ([]*model.Model, error)
	UpdateStatus(id string, enabled bool) error
	GetPricing(id string) (*model.ModelPricing, error)
	InvalidateCache()
}

// modelRepository Model 仓库实现
type modelRepository struct {
	db *gorm.DB
}

// NewModelRepository 创建 Model 仓库
func NewModelRepository(db *gorm.DB) ModelRepository {
	return &modelRepository{db: db}
}

// Create 创建模型
func (r *modelRepository) Create(m *model.Model) error {
	err := r.db.Create(m).Error
	if err == nil {
		r.InvalidateCache()
	}
	return err
}

// GetByID 根据 ID 获取模型
func (r *modelRepository) GetByID(id string) (*model.Model, error) {
	// 先从缓存获取
	cacheKey := cache.KeyModelPrefix + id
	var m model.Model
	if err := cache.Get(cacheKey, &m); err == nil {
		return &m, nil
	}

	// 从数据库获取
	if err := r.db.Where("id = ?", id).First(&m).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// 缓存模型信息
	_ = cache.Set(cacheKey, &m, 10*time.Minute)

	return &m, nil
}

// Update 更新模型
func (r *modelRepository) Update(m *model.Model) error {
	err := r.db.Save(m).Error
	if err == nil {
		r.InvalidateCache()
		_ = cache.Delete(cache.KeyModelPrefix + m.Name)
	}
	return err
}

// Delete 删除模型
func (r *modelRepository) Delete(id string) error {
	err := r.db.Delete(&model.Model{}, "id = ?", id).Error
	if err == nil {
		r.InvalidateCache()
		_ = cache.Delete(cache.KeyModelPrefix + id)
	}
	return err
}

// List 获取模型列表
func (r *modelRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.Model, int64, error) {
	var models []*model.Model
	var total int64

	query := r.db.Model(&model.Model{})

	// 应用过滤条件
	if enabled, ok := filters["enabled"]; ok {
		query = query.Where("enabled = ?", enabled)
	}
	if modelType, ok := filters["type"]; ok {
		query = query.Where("type = ?", modelType)
	}
	if keyword, ok := filters["keyword"]; ok && keyword != "" {
		pattern := "%" + keyword.(string) + "%"
		query = query.Where(
			"id LIKE ? OR name LIKE ? OR display_name LIKE ? OR provider LIKE ?",
			pattern,
			pattern,
			pattern,
			pattern,
		)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("id ASC").Find(&models).Error; err != nil {
		return nil, 0, err
	}

	return models, total, nil
}

// ListAll 获取所有模型
func (r *modelRepository) ListAll() ([]*model.Model, error) {
	// 先从缓存获取
	var models []*model.Model
	if err := cache.Get(cache.KeyModelList, &models); err == nil {
		return models, nil
	}

	// 从数据库获取
	if err := r.db.Order("id ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	// 缓存列表
	_ = cache.Set(cache.KeyModelList, models, 5*time.Minute)

	return models, nil
}

// ListEnabled 获取启用的模型
func (r *modelRepository) ListEnabled() ([]*model.Model, error) {
	// 先从缓存获取
	cacheKey := cache.KeyModelList + ":enabled"
	var models []*model.Model
	if err := cache.Get(cacheKey, &models); err == nil {
		return models, nil
	}

	// 从数据库获取
	if err := r.db.Where("enabled = ?", true).Order("name ASC").Find(&models).Error; err != nil {
		return nil, err
	}

	// 缓存列表
	_ = cache.Set(cacheKey, models, 5*time.Minute)

	return models, nil
}

// UpdateStatus 更新状态
func (r *modelRepository) UpdateStatus(id string, enabled bool) error {
	err := r.db.Model(&model.Model{}).Where("id = ? OR name = ?", id, id).Update("enabled", enabled).Error
	if err == nil {
		r.InvalidateCache()
		_ = cache.Delete(cache.KeyModelPrefix + id)
	}
	return err
}

// GetPricing 获取定价信息（通过模型名称）
func (r *modelRepository) GetPricing(modelName string) (*model.ModelPricing, error) {
	// 先从缓存获取
	cacheKey := cache.KeyModelPrefix + "pricing:" + modelName
	var pricing model.ModelPricing
	if err := cache.Get(cacheKey, &pricing); err == nil {
		return &pricing, nil
	}

	// 从数据库获取
	var m model.Model
	if err := r.db.Where("name = ? OR id = ?", modelName, modelName).First(&m).Error; err != nil {
		return nil, err
	}

	result := &model.ModelPricing{
		ModelID:         m.ID,
		InputPrice:      m.InputPrice,
		OutputPrice:     m.OutputPrice,
		PriceUnit:       m.PriceUnit,
		ContextLength:   m.ContextLength,
		MaxOutputTokens: m.MaxOutputTokens,
	}

	// 缓存定价信息
	_ = cache.Set(cacheKey, result, 10*time.Minute)

	return result, nil
}

// InvalidateCache 清除缓存
func (r *modelRepository) InvalidateCache() {
	_ = cache.Delete(cache.KeyModelList)
	_ = cache.Delete(cache.KeyModelList + ":enabled")
}
