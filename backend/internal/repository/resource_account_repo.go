package repository

import (
	"errors"
	"strconv"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// ResourceAccountRepository ResourceAccount 仓库接口
type ResourceAccountRepository interface {
	Create(account *model.ResourceAccount) error
	GetByID(id uint) (*model.ResourceAccount, error)
	Update(account *model.ResourceAccount) error
	Delete(id uint) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.ResourceAccount, int64, error)
	ListByChannelID(channelID uint) ([]*model.ResourceAccount, error)
	ListActive() ([]*model.ResourceAccount, error)
	ListActiveByChannelID(channelID uint) ([]*model.ResourceAccount, error)
	GetRandomActive(channelID uint) (*model.ResourceAccount, error)
	UpdateStatus(id uint, status model.ResourceAccountStatus) error
	UpdateSession(id uint, sessionToken, cookieData string) error
	MarkActive(id uint) error
	MarkError(id uint, errMsg string) error
	GetStats() (*model.ResourcePoolStats, error)
	InvalidateCache(channelID uint)
}

// resourceAccountRepository ResourceAccount 仓库实现
type resourceAccountRepository struct {
	db *gorm.DB
}

// NewResourceAccountRepository 创建 ResourceAccount 仓库
func NewResourceAccountRepository(db *gorm.DB) ResourceAccountRepository {
	return &resourceAccountRepository{db: db}
}

// Create 创建资源账户
func (r *resourceAccountRepository) Create(account *model.ResourceAccount) error {
	err := r.db.Create(account).Error
	if err == nil {
		r.InvalidateCache(account.ChannelID)
	}
	return err
}

// GetByID 根据 ID 获取资源账户
func (r *resourceAccountRepository) GetByID(id uint) (*model.ResourceAccount, error) {
	var account model.ResourceAccount
	if err := r.db.Preload("Channel").Where("id = ?", id).First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// Update 更新资源账户
func (r *resourceAccountRepository) Update(account *model.ResourceAccount) error {
	err := r.db.Save(account).Error
	if err == nil {
		r.InvalidateCache(account.ChannelID)
	}
	return err
}

// Delete 删除资源账户
func (r *resourceAccountRepository) Delete(id uint) error {
	// 先获取账户以便清除缓存
	var account model.ResourceAccount
	if err := r.db.Where("id = ?", id).First(&account).Error; err != nil {
		return err
	}

	err := r.db.Delete(&model.ResourceAccount{}, "id = ?", id).Error
	if err == nil {
		r.InvalidateCache(account.ChannelID)
	}
	return err
}

// List 获取资源账户列表
func (r *resourceAccountRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.ResourceAccount, int64, error) {
	var accounts []*model.ResourceAccount
	var total int64

	query := r.db.Model(&model.ResourceAccount{})

	// 应用过滤条件
	if channelID, ok := filters["channel_id"]; ok {
		query = query.Where("channel_id = ?", channelID)
	}
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if keyword, ok := filters["keyword"]; ok && keyword != "" {
		query = query.Where("account_name LIKE ?", "%"+keyword.(string)+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Channel").Offset(offset).Limit(pageSize).Order("id ASC").Find(&accounts).Error; err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// ListByChannelID 根据渠道 ID 获取资源账户列表
func (r *resourceAccountRepository) ListByChannelID(channelID uint) ([]*model.ResourceAccount, error) {
	var accounts []*model.ResourceAccount
	if err := r.db.Where("channel_id = ?", channelID).Order("id ASC").Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// ListActive 获取所有活跃的资源账户
func (r *resourceAccountRepository) ListActive() ([]*model.ResourceAccount, error) {
	var accounts []*model.ResourceAccount
	if err := r.db.Preload("Channel").
		Where("status = ?", model.ResourceAccountStatusActive).
		Order("last_active_at DESC").
		Find(&accounts).Error; err != nil {
		return nil, err
	}
	return accounts, nil
}

// ListActiveByChannelID 根据渠道 ID 获取活跃的资源账户列表
func (r *resourceAccountRepository) ListActiveByChannelID(channelID uint) ([]*model.ResourceAccount, error) {
	// 先从缓存获取
	cacheKey := cache.KeyResourcePool + strconv.FormatUint(uint64(channelID), 10)
	var accounts []*model.ResourceAccount
	if err := cache.Get(cacheKey, &accounts); err == nil {
		return accounts, nil
	}

	// 从数据库获取
	if err := r.db.Where("channel_id = ? AND status = ?", channelID, model.ResourceAccountStatusActive).
		Order("last_active_at DESC").
		Find(&accounts).Error; err != nil {
		return nil, err
	}

	// 缓存列表
	_ = cache.Set(cacheKey, accounts, 1*time.Minute)

	return accounts, nil
}

// GetRandomActive 随机获取一个活跃的资源账户
func (r *resourceAccountRepository) GetRandomActive(channelID uint) (*model.ResourceAccount, error) {
	var account model.ResourceAccount
	if err := r.db.Where("channel_id = ? AND status = ?", channelID, model.ResourceAccountStatusActive).
		Order("RANDOM()").
		First(&account).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &account, nil
}

// UpdateStatus 更新状态
func (r *resourceAccountRepository) UpdateStatus(id uint, status model.ResourceAccountStatus) error {
	// 先获取账户以便清除缓存
	var account model.ResourceAccount
	if err := r.db.Where("id = ?", id).First(&account).Error; err != nil {
		return err
	}

	err := r.db.Model(&model.ResourceAccount{}).Where("id = ?", id).Update("status", status).Error
	if err == nil {
		r.InvalidateCache(account.ChannelID)
	}
	return err
}

// UpdateSession 更新会话信息
func (r *resourceAccountRepository) UpdateSession(id uint, sessionToken, cookieData string) error {
	now := time.Now()
	return r.db.Model(&model.ResourceAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"session_token":  sessionToken,
			"cookie_data":    cookieData,
			"last_active_at": now,
			"status":         model.ResourceAccountStatusActive,
		}).Error
}

// MarkActive 标记为活跃
func (r *resourceAccountRepository) MarkActive(id uint) error {
	now := time.Now()
	return r.db.Model(&model.ResourceAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":         model.ResourceAccountStatusActive,
			"last_active_at": now,
			"error_count":    0,
			"last_error":     "",
		}).Error
}

// MarkError 标记错误
func (r *resourceAccountRepository) MarkError(id uint, errMsg string) error {
	// 先获取当前错误计数
	var account model.ResourceAccount
	if err := r.db.Where("id = ?", id).First(&account).Error; err != nil {
		return err
	}

	updates := map[string]interface{}{
		"error_count": gorm.Expr("error_count + 1"),
		"last_error":  errMsg,
	}

	// 如果错误次数超过阈值，标记为不活跃
	if account.ErrorCount >= 4 { // 第5次错误时标记为不活跃
		updates["status"] = model.ResourceAccountStatusInactive
		r.InvalidateCache(account.ChannelID)
	}

	return r.db.Model(&model.ResourceAccount{}).Where("id = ?", id).Updates(updates).Error
}

// GetStats 获取统计信息
func (r *resourceAccountRepository) GetStats() (*model.ResourcePoolStats, error) {
	var stats model.ResourcePoolStats

	// 统计各状态的数量
	var results []struct {
		Status model.ResourceAccountStatus
		Count  int64
	}

	if err := r.db.Model(&model.ResourceAccount{}).
		Select("status, count(*) as count").
		Group("status").
		Scan(&results).Error; err != nil {
		return nil, err
	}

	for _, result := range results {
		switch result.Status {
		case model.ResourceAccountStatusActive:
			stats.ActiveAccounts = result.Count
		case model.ResourceAccountStatusInactive:
			stats.InactiveAccounts = result.Count
		case model.ResourceAccountStatusExpired:
			stats.ExpiredAccounts = result.Count
		case model.ResourceAccountStatusBanned:
			stats.BannedAccounts = result.Count
		}
		stats.TotalAccounts += result.Count
	}

	return &stats, nil
}

// InvalidateCache 清除缓存
func (r *resourceAccountRepository) InvalidateCache(channelID uint) {
	_ = cache.Delete(cache.KeyResourcePool + strconv.FormatUint(uint64(channelID), 10))
}
