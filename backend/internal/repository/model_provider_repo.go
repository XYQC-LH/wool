package repository

import (
	"context"
	"errors"
	"time"

	"nexus-api/internal/model"

	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// ModelProviderRepository 模型源头仓库接口
type ModelProviderRepository interface {
	// 基础 CRUD
	Create(ctx context.Context, provider *model.ModelProvider) error
	Update(ctx context.Context, provider *model.ModelProvider) error
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ModelProvider, error)

	// 查询方法
	GetByModelID(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error)
	GetByChannelID(ctx context.Context, channelID uint) ([]*model.ModelProvider, error)
	GetByModelAndChannel(ctx context.Context, operation string, modelID string, channelID uint) (*model.ModelProvider, error)
	GetActiveProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error)
	GetAvailableProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error)
	List(ctx context.Context, params *model.ProviderQueryParams) ([]*model.ModelProvider, int64, error)

	// 熔断相关
	IncrementFailure(ctx context.Context, id uint) error
	ResetFailure(ctx context.Context, id uint) error
	OpenCircuit(ctx context.Context, id uint, duration time.Duration) error
	CloseCircuit(ctx context.Context, id uint) error
	HalfOpenCircuit(ctx context.Context, id uint) error
	GetCircuitState(ctx context.Context, id uint) (model.CircuitState, error)

	// 统计相关
	IncrementStats(ctx context.Context, id uint, success bool, latencyMs int64, inputTokens, outputTokens int64, cost decimal.Decimal) error
	UpdateHealthScore(ctx context.Context, id uint, score float64) error
	GetMetrics(ctx context.Context, id uint, startTime, endTime time.Time) (*model.ProviderMetrics, error)

	// 批量操作
	BatchCreate(ctx context.Context, providers []*model.ModelProvider) error
	BatchUpdateStatus(ctx context.Context, ids []uint, status model.ProviderStatus) error
	DeleteByModelID(ctx context.Context, operation string, modelID string) error
	DeleteByChannelID(ctx context.Context, channelID uint) error
}

// modelProviderRepository 模型源头仓库实现
type modelProviderRepository struct {
	db *gorm.DB
}

// NewModelProviderRepository 创建模型源头仓库
func NewModelProviderRepository(db *gorm.DB) ModelProviderRepository {
	return &modelProviderRepository{db: db}
}

// Create 创建模型源头
func (r *modelProviderRepository) Create(ctx context.Context, provider *model.ModelProvider) error {
	return r.db.WithContext(ctx).Create(provider).Error
}

// Update 更新模型源头
func (r *modelProviderRepository) Update(ctx context.Context, provider *model.ModelProvider) error {
	return r.db.WithContext(ctx).Save(provider).Error
}

// Delete 删除模型源头
func (r *modelProviderRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&model.ModelProvider{}, id).Error
}

// GetByID 根据ID获取模型源头
func (r *modelProviderRepository) GetByID(ctx context.Context, id uint) (*model.ModelProvider, error) {
	var provider model.ModelProvider
	err := r.db.WithContext(ctx).
		Preload("Model").
		Preload("Channel").
		First(&provider, id).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

// GetByModelID 根据模型ID获取所有源头
func (r *modelProviderRepository) GetByModelID(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	var providers []*model.ModelProvider
	err := r.db.WithContext(ctx).
		Preload("Model").
		Preload("Channel").
		Where("operation = ? AND model_id = ?", operation, modelID).
		Order("priority ASC, actual_cost_per_1k_input ASC").
		Find(&providers).Error
	return providers, err
}

// GetByChannelID 根据渠道ID获取所有源头
func (r *modelProviderRepository) GetByChannelID(ctx context.Context, channelID uint) ([]*model.ModelProvider, error) {
	var providers []*model.ModelProvider
	err := r.db.WithContext(ctx).
		Preload("Model").
		Preload("Channel").
		Where("channel_id = ?", channelID).
		Find(&providers).Error
	return providers, err
}

// GetByModelAndChannel 根据模型ID和渠道ID获取源头
func (r *modelProviderRepository) GetByModelAndChannel(ctx context.Context, operation string, modelID string, channelID uint) (*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	var provider model.ModelProvider
	err := r.db.WithContext(ctx).
		Preload("Model").
		Preload("Channel").
		Where("operation = ? AND model_id = ? AND channel_id = ?", operation, modelID, channelID).
		First(&provider).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &provider, nil
}

// GetActiveProviders 获取活跃的源头（状态为 active）
func (r *modelProviderRepository) GetActiveProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	var providers []*model.ModelProvider
	err := r.db.WithContext(ctx).
		Preload("Model").
		Preload("Channel").
		Where("operation = ? AND model_id = ? AND status = ?", operation, modelID, model.ProviderStatusActive).
		Order("priority ASC, actual_cost_per_1k_input ASC").
		Find(&providers).Error
	return providers, err
}

// GetAvailableProviders 获取可用的源头（活跃且未熔断）
// ⭐ 核心方法：按成本优先排序，用于调度
func (r *modelProviderRepository) GetAvailableProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	var providers []*model.ModelProvider
	now := time.Now()

	err := r.db.WithContext(ctx).
		Preload("Model").
		Preload("Channel").
		Where("operation = ? AND model_id = ?", operation, modelID).
		Where("status = ?", model.ProviderStatusActive).
		Where("circuit_state != ? OR (circuit_state = ? AND circuit_open_until < ?)",
									model.CircuitStateOpen, model.CircuitStateOpen, now).
		Order("actual_cost_per_1k_input ASC, priority ASC"). // ⭐ 成本优先排序
		Find(&providers).Error

	return providers, err
}

// List 分页查询源头列表
func (r *modelProviderRepository) List(ctx context.Context, params *model.ProviderQueryParams) ([]*model.ModelProvider, int64, error) {
	var providers []*model.ModelProvider
	var total int64

	query := r.db.WithContext(ctx).Model(&model.ModelProvider{})

	// 应用过滤条件
	if params.Operation != "" {
		query = query.Where("operation = ?", params.Operation)
	}
	if params.ModelID != "" {
		query = query.Where("model_id = ?", params.ModelID)
	}
	if params.ChannelID > 0 {
		query = query.Where("channel_id = ?", params.ChannelID)
	}
	if params.Status != "" {
		query = query.Where("status = ?", params.Status)
	}
	if params.CircuitState != "" {
		query = query.Where("circuit_state = ?", params.CircuitState)
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
		Preload("Model").
		Preload("Channel").
		Order("model_id ASC, actual_cost_per_1k_input ASC").
		Offset(offset).
		Limit(pageSize).
		Find(&providers).Error

	return providers, total, err
}

// IncrementFailure 增加失败计数
func (r *modelProviderRepository) IncrementFailure(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"failure_count":   gorm.Expr("failure_count + 1"),
			"failed_requests": gorm.Expr("failed_requests + 1"),
			"total_requests":  gorm.Expr("total_requests + 1"),
			"last_failure_at": time.Now(),
			"last_used_at":    time.Now(),
		}).Error
}

// ResetFailure 重置失败计数
func (r *modelProviderRepository) ResetFailure(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"failure_count":   0,
			"circuit_state":   model.CircuitStateClosed,
			"last_success_at": time.Now(),
		}).Error
}

// OpenCircuit 打开熔断器
func (r *modelProviderRepository) OpenCircuit(ctx context.Context, id uint, duration time.Duration) error {
	openUntil := time.Now().Add(duration)
	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"circuit_state":      model.CircuitStateOpen,
			"circuit_open_until": openUntil,
		}).Error
}

// CloseCircuit 关闭熔断器
func (r *modelProviderRepository) CloseCircuit(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"circuit_state":      model.CircuitStateClosed,
			"circuit_open_until": nil,
			"failure_count":      0,
		}).Error
}

// HalfOpenCircuit 半开熔断器
func (r *modelProviderRepository) HalfOpenCircuit(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"circuit_state": model.CircuitStateHalfOpen,
		}).Error
}

// GetCircuitState 获取熔断状态
func (r *modelProviderRepository) GetCircuitState(ctx context.Context, id uint) (model.CircuitState, error) {
	var provider model.ModelProvider
	err := r.db.WithContext(ctx).
		Select("circuit_state", "circuit_open_until").
		First(&provider, id).Error
	if err != nil {
		return model.CircuitStateClosed, err
	}

	// 检查是否应该从 OPEN 转为 HALF_OPEN
	if provider.CircuitState == model.CircuitStateOpen &&
		provider.CircuitOpenUntil != nil &&
		time.Now().After(*provider.CircuitOpenUntil) {
		// 自动转为半开状态
		_ = r.HalfOpenCircuit(ctx, id)
		return model.CircuitStateHalfOpen, nil
	}

	return provider.CircuitState, nil
}

// IncrementStats 增加统计数据
func (r *modelProviderRepository) IncrementStats(ctx context.Context, id uint, success bool, latencyMs int64, inputTokens, outputTokens int64, cost decimal.Decimal) error {
	updates := map[string]interface{}{
		"total_requests": gorm.Expr("total_requests + 1"),
		"total_latency":  gorm.Expr("total_latency + ?", latencyMs),
		"input_tokens":   gorm.Expr("input_tokens + ?", inputTokens),
		"output_tokens":  gorm.Expr("output_tokens + ?", outputTokens),
		"total_cost":     gorm.Expr("total_cost + ?", cost),
		"last_used_at":   time.Now(),
	}

	if success {
		updates["success_requests"] = gorm.Expr("success_requests + 1")
		updates["last_success_at"] = time.Now()
		updates["failure_count"] = 0 // 成功时重置连续失败计数
	} else {
		updates["failed_requests"] = gorm.Expr("failed_requests + 1")
		updates["failure_count"] = gorm.Expr("failure_count + 1")
		updates["last_failure_at"] = time.Now()
	}

	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateHealthScore 更新健康分数
func (r *modelProviderRepository) UpdateHealthScore(ctx context.Context, id uint, score float64) error {
	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id = ?", id).
		Update("health_score", score).Error
}

// BatchCreate 批量创建源头
func (r *modelProviderRepository) BatchCreate(ctx context.Context, providers []*model.ModelProvider) error {
	if len(providers) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).CreateInBatches(providers, 100).Error
}

// BatchUpdateStatus 批量更新状态
func (r *modelProviderRepository) BatchUpdateStatus(ctx context.Context, ids []uint, status model.ProviderStatus) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Where("id IN ?", ids).
		Update("status", status).Error
}

// DeleteByModelID 根据模型ID删除所有源头
func (r *modelProviderRepository) DeleteByModelID(ctx context.Context, operation string, modelID string) error {
	operation = model.NormalizeOperation(operation)
	return r.db.WithContext(ctx).
		Where("operation = ? AND model_id = ?", operation, modelID).
		Delete(&model.ModelProvider{}).Error
}

// DeleteByChannelID 根据渠道ID删除所有源头
func (r *modelProviderRepository) DeleteByChannelID(ctx context.Context, channelID uint) error {
	return r.db.WithContext(ctx).
		Where("channel_id = ?", channelID).
		Delete(&model.ModelProvider{}).Error
}

// GetMetrics 获取指定时间范围内的指标
func (r *modelProviderRepository) GetMetrics(ctx context.Context, id uint, startTime, endTime time.Time) (*model.ProviderMetrics, error) {
	var metrics model.ProviderMetrics
	err := r.db.WithContext(ctx).
		Model(&model.ModelProvider{}).
		Select(`
			id as provider_id,
			total_requests,
			success_requests,
			failed_requests,
			total_latency,
			input_tokens,
			output_tokens,
			total_cost,
			health_score,
			last_success_at,
			last_failure_at
		`).
		Where("id = ?", id).
		First(&metrics).Error
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}
