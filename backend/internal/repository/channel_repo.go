package repository

import (
	"errors"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ChannelRepository Channel 仓库接口
type ChannelRepository interface {
	Create(channel *model.Channel) error
	GetByID(id uint) (*model.Channel, error)
	Update(channel *model.Channel) error
	Delete(id uint) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.Channel, int64, error)
	ListAll() ([]*model.Channel, error)
	ListHealthy() ([]*model.Channel, error)
	ListByModel(modelID string) ([]*model.Channel, error)
	UpdateStatus(id uint, status model.ChannelStatus) error
	UpdateLatency(id uint, latency int) error
	UpdateLastTest(id uint, testAt time.Time, latency int) error
	IncrementErrorCount(id uint) error
	ResetErrorCount(id uint) error
	GetStats() (*model.ChannelStats, error)
	GetChannelModel(channelID uint, modelID string) (*model.ChannelModel, error)
	GetChannelModels(channelID uint) ([]*model.ChannelModel, error)
	InvalidateCache()
}

// channelRepository Channel 仓库实现
type channelRepository struct {
	db *gorm.DB
}

// NewChannelRepository 创建 Channel 仓库
func NewChannelRepository(db *gorm.DB) ChannelRepository {
	return &channelRepository{db: db}
}

// Create 创建 Channel
func (r *channelRepository) Create(channel *model.Channel) error {
	err := r.db.Create(channel).Error
	if err == nil {
		r.InvalidateCache()
	}
	return err
}

// GetByID 根据 ID 获取 Channel
func (r *channelRepository) GetByID(id uint) (*model.Channel, error) {
	var channel model.Channel

	// 从数据库获取
	if err := r.db.Where("id = ?", id).First(&channel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	return &channel, nil
}

// Update 更新 Channel
func (r *channelRepository) Update(channel *model.Channel) error {
	err := r.db.Save(channel).Error
	if err == nil {
		r.InvalidateCache()
	}
	return err
}

// Delete 删除 Channel
func (r *channelRepository) Delete(id uint) error {
	err := r.db.Delete(&model.Channel{}, "id = ?", id).Error
	if err == nil {
		r.InvalidateCache()
	}
	return err
}

// List 获取 Channel 列表
func (r *channelRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.Channel, int64, error) {
	var channels []*model.Channel
	var total int64

	query := r.db.Model(&model.Channel{})

	// 应用过滤条件
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if channelType, ok := filters["type"]; ok {
		query = query.Where("type = ?", channelType)
	}
	if keyword, ok := filters["keyword"]; ok && keyword != "" {
		query = query.Where("name LIKE ?", "%"+keyword.(string)+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("weight DESC, id ASC").Find(&channels).Error; err != nil {
		return nil, 0, err
	}

	return channels, total, nil
}

// ListAll 获取所有 Channel
func (r *channelRepository) ListAll() ([]*model.Channel, error) {
	var channels []*model.Channel

	// 从数据库获取
	if err := r.db.Order("weight DESC, id ASC").Find(&channels).Error; err != nil {
		return nil, err
	}

	return channels, nil
}

// ListHealthy 获取健康的 Channel
func (r *channelRepository) ListHealthy() ([]*model.Channel, error) {
	var channels []*model.Channel

	// 从数据库获取
	if err := r.db.Where("status = ?", model.ChannelStatusHealthy).
		Order("weight DESC, id ASC").
		Find(&channels).Error; err != nil {
		return nil, err
	}

	return channels, nil
}

// ListByModel 根据模型获取支持的 Channel
func (r *channelRepository) ListByModel(modelID string) ([]*model.Channel, error) {
	var channels []*model.Channel

	// 从数据库获取（使用 JSONB 查询）
	if err := r.db.Where("status = ? AND models ? ?", model.ChannelStatusHealthy, modelID).
		Order("weight DESC, id ASC").
		Find(&channels).Error; err != nil {
		// 如果 JSONB 查询失败，使用 LIKE 查询
		if err := r.db.Where("status = ? AND models LIKE ?", model.ChannelStatusHealthy, "%"+modelID+"%").
			Order("weight DESC, id ASC").
			Find(&channels).Error; err != nil {
			return nil, err
		}
	}

	return channels, nil
}

// UpdateStatus 更新状态
func (r *channelRepository) UpdateStatus(id uint, status model.ChannelStatus) error {
	err := r.db.Model(&model.Channel{}).Where("id = ?", id).Update("status", status).Error
	if err == nil {
		r.InvalidateCache()
	}
	return err
}

// UpdateLatency 更新延迟
func (r *channelRepository) UpdateLatency(id uint, latency int) error {
	return r.db.Model(&model.Channel{}).Where("id = ?", id).Update("latency", latency).Error
}

func (r *channelRepository) UpdateLastTest(id uint, testAt time.Time, latency int) error {
	return r.db.Model(&model.Channel{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"last_test_at":      testAt,
			"last_test_latency": latency,
		}).Error
}

// IncrementErrorCount 增加错误计数
func (r *channelRepository) IncrementErrorCount(id uint) error {
	return r.db.Model(&model.Channel{}).
		Where("id = ?", id).
		Update("error_count", gorm.Expr("error_count + 1")).Error
}

// ResetErrorCount 重置错误计数
func (r *channelRepository) ResetErrorCount(id uint) error {
	return r.db.Model(&model.Channel{}).
		Where("id = ?", id).
		Update("error_count", 0).Error
}

// GetStats 获取统计信息
func (r *channelRepository) GetStats() (*model.ChannelStats, error) {
	var stats model.ChannelStats

	// 统计各状态的数量
	var results []struct {
		Status model.ChannelStatus
		Count  int64
	}

	if err := r.db.Model(&model.Channel{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	for _, result := range results {
		switch result.Status {
		case model.ChannelStatusHealthy:
			stats.SuccessCount = result.Count
		case model.ChannelStatusDown:
			stats.FailCount = result.Count
		}
	}

	stats.RequestCount = stats.SuccessCount + stats.FailCount
	if stats.RequestCount > 0 {
		stats.SuccessRate = float64(stats.SuccessCount) / float64(stats.RequestCount) * 100
	}

	return &stats, nil
}

// GetChannelModel 获取渠道模型映射
func (r *channelRepository) GetChannelModel(channelID uint, modelID string) (*model.ChannelModel, error) {
	var channelModel model.ChannelModel
	if err := r.db.Where("channel_id = ? AND model_id = ?", channelID, modelID).First(&channelModel).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &channelModel, nil
}

// GetChannelModels 获取渠道下的所有模型映射
func (r *channelRepository) GetChannelModels(channelID uint) ([]*model.ChannelModel, error) {
	var channelModels []*model.ChannelModel
	if err := r.db.Where("channel_id = ?", channelID).Find(&channelModels).Error; err != nil {
		return nil, err
	}
	return channelModels, nil
}

// InvalidateCache 清除缓存
func (r *channelRepository) InvalidateCache() {
	_ = cache.Delete(cache.KeyChannelList)
	_ = cache.Delete(cache.KeyHealthyChannels)
}
