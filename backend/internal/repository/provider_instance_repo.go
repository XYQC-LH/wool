package repository

import (
	"context"
	"errors"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ProviderInstanceRepository 源头实例仓库接口
type ProviderInstanceRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, instance *model.ProviderInstance) error
	Update(ctx context.Context, instance *model.ProviderInstance) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ProviderInstance, error)

	// 查询方法
	GetByProviderID(ctx context.Context, providerID uint) ([]*model.ProviderInstance, error)
	GetAvailableInstances(ctx context.Context, providerID uint) ([]*model.ProviderInstance, error)
	List(ctx context.Context, params *model.ProviderInstanceQueryParams) ([]*model.ProviderInstance, int64, error)

	// 统计相关
	IncrementStats(ctx context.Context, id uint, success bool, latencyMs int64) error
	ResetStats(ctx context.Context, id uint) error

	// 批量操作
	BatchCreate(ctx context.Context, instances []*model.ProviderInstance) error
	BatchUpdateStatus(ctx context.Context, ids []uint, status model.InstanceStatus) error
	DeleteByProviderID(ctx context.Context, providerID uint) error

	// 统计查询
	GetStats(ctx context.Context, providerID uint) (*model.ProviderInstanceStats, error)
}

// providerInstanceRepository 源头实例仓库实现
type providerInstanceRepository struct {
	db *gorm.DB
}

// NewProviderInstanceRepository 创建源头实例仓库
func NewProviderInstanceRepository(db *gorm.DB) ProviderInstanceRepository {
	return &providerInstanceRepository{db: db}
}

// Create 创建源头实例
func (r *providerInstanceRepository) Create(ctx context.Context, instance *model.ProviderInstance) error {
	return r.db.WithContext(ctx).Create(instance).Error
}

// Update 更新源头实例
func (r *providerInstanceRepository) Update(ctx context.Context, instance *model.ProviderInstance) error {
	return r.db.WithContext(ctx).Save(instance).Error
}

// Delete 删除源头实例
func (r *providerInstanceRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ProviderInstance{}, id).Error
}

// GetByID 根据ID获取源头实例
func (r *providerInstanceRepository) GetByID(ctx context.Context, id uint) (*model.ProviderInstance, error) {
	var instance model.ProviderInstance
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Channel").
		Preload("ResourceAccount").
		First(&instance, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &instance, nil
}

// GetByProviderID 根据源头ID获取所有实例
func (r *providerInstanceRepository) GetByProviderID(ctx context.Context, providerID uint) ([]*model.ProviderInstance, error) {
	var instances []*model.ProviderInstance
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Channel").
		Preload("ResourceAccount").
		Where("provider_id = ?", providerID).
		Order("weight DESC").
		Find(&instances).Error
	return instances, err
}

// GetAvailableInstances 获取可用的实例（状态为 active）
// ⭐ 核心方法：用于实例级调度
func (r *providerInstanceRepository) GetAvailableInstances(ctx context.Context, providerID uint) ([]*model.ProviderInstance, error) {
	var instances []*model.ProviderInstance
	err := r.db.WithContext(ctx).
		Preload("Provider").
		Preload("Provider.Channel").
		Preload("ResourceAccount").
		Where("provider_id = ?", providerID).
		Where("status = ?", model.InstanceStatusActive).
		Order("weight DESC").
		Find(&instances).Error
	return instances, err
}

// List 分页查询实例列表
func (r *providerInstanceRepository) List(ctx context.Context, params *model.ProviderInstanceQueryParams) ([]*model.ProviderInstance, int64, error) {
	var instances []*model.ProviderInstance
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ProviderInstance{})

	// 应用过滤条件
	if params.ProviderID > 0 {
		query = query.Where("provider_id = ?", params.ProviderID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.InstanceType != "" {
		query = query.Where("instance_type = ?", params.InstanceType)
	}

	// 计算总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页
	page := params.Page
	pageSize := params.PageSize
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

	// 查询数据
	err := query.
		Preload("Provider").
		Preload("Provider.Channel").
		Preload("ResourceAccount").
		Order("weight DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&instances).Error

	return instances, total, err
}

// IncrementStats 增加统计数据
func (r *providerInstanceRepository) IncrementStats(ctx context.Context, id uint, success bool, latencyMs int64) error {
	updates := map[string]interface{}{
		"total_requests": gorm.Expr("total_requests + 1"),
		"total_latency":  gorm.Expr("total_latency + ?", latencyMs),
	}

	if success {
		updates["success_requests"] = gorm.Expr("success_requests + 1")
	} else {
		updates["failed_requests"] = gorm.Expr("failed_requests + 1")
	}

	return r.db.WithContext(ctx).
		Model(&model.ProviderInstance{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// ResetStats 重置统计数据
func (r *providerInstanceRepository) ResetStats(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.ProviderInstance{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"total_requests":   0,
			"success_requests": 0,
			"failed_requests":  0,
			"total_latency":    0,
		}).Error
}

// BatchCreate 批量创建实例
func (r *providerInstanceRepository) BatchCreate(ctx context.Context, instances []*model.ProviderInstance) error {
	if len(instances) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(instances, 100).Error
}

// BatchUpdateStatus 批量更新状态
func (r *providerInstanceRepository) BatchUpdateStatus(ctx context.Context, ids []uint, status model.InstanceStatus) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.ProviderInstance{}).
		Where("id IN ?", ids).
		Update("status", status).Error
}

// DeleteByProviderID 根据源头ID删除所有实例
func (r *providerInstanceRepository) DeleteByProviderID(ctx context.Context, providerID uint) error {
	return r.db.WithContext(ctx).
		Where("provider_id = ?", providerID).
		Delete(&model.ProviderInstance{}).Error
}

// GetStats 获取源头实例统计
func (r *providerInstanceRepository) GetStats(ctx context.Context, providerID uint) (*model.ProviderInstanceStats, error) {
	var stats model.ProviderInstanceStats

	query := r.db.WithContext(ctx).Model(&model.ProviderInstance{})
	if providerID > 0 {
		query = query.Where("provider_id = ?", providerID)
	}

	// 总数
	if err := query.Count(&stats.TotalInstances).Error; err != nil {
		return nil, err
	}

	// 活跃数
	if err := r.db.WithContext(ctx).Model(&model.ProviderInstance{}).
		Where("status = ?", model.InstanceStatusActive).
		Count(&stats.ActiveInstances).Error; err != nil {
		return nil, err
	}

	// 禁用数
	if err := r.db.WithContext(ctx).Model(&model.ProviderInstance{}).
		Where("status = ?", model.InstanceStatusDisabled).
		Count(&stats.DisabledInstances).Error; err != nil {
		return nil, err
	}

	// 冷却数
	if err := r.db.WithContext(ctx).Model(&model.ProviderInstance{}).
		Where("status = ?", model.InstanceStatusCooling).
		Count(&stats.CoolingInstances).Error; err != nil {
		return nil, err
	}

	// 平均成功率
	var avgResult struct {
		AvgSuccessRate float64
	}
	r.db.WithContext(ctx).Model(&model.ProviderInstance{}).
		Select("AVG(CASE WHEN total_requests > 0 THEN (success_requests * 100.0 / total_requests) ELSE 100 END) as avg_success_rate").
		Scan(&avgResult)
	stats.AvgSuccessRate = avgResult.AvgSuccessRate

	return &stats, nil
}
