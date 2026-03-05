package repository

import (
	"errors"
	"strings"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// TokenRepository Token 仓库接口
type TokenRepository interface {
	Create(token *model.Token) error
	GetByID(id uuid.UUID) (*model.Token, error)
	GetByKey(key string) (*model.Token, error)
	GetByKeyWithUser(key string) (*model.Token, error)
	Update(token *model.Token) error
	Delete(id uuid.UUID) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.Token, int64, error)
	ListByUserID(userID uuid.UUID, page, pageSize int) ([]*model.Token, int64, error)
	DeductQuota(id uuid.UUID, amount decimal.Decimal) error
	UpdateLastUsed(id uuid.UUID) error
	CountByUserID(userID uuid.UUID) (int64, error)
	InvalidateCache(key string)
}

// List 获取 Token 列表（管理员）
func (r *tokenRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.Token, int64, error) {
	var tokens []*model.Token
	var total int64

	query := r.db.Model(&model.Token{})

	if userID, ok := filters["user_id"]; ok {
		query = query.Where("user_id = ?", userID)
	}

	if status, ok := filters["status"].(string); ok {
		status = strings.TrimSpace(status)
		if status != "" {
			query = query.Where("status = ?", status)
		}
	}

	if keyword, ok := filters["keyword"].(string); ok {
		keyword = strings.TrimSpace(keyword)
		if keyword != "" {
			likeKeyword := "%" + keyword + "%"
			query = query.Where("name LIKE ? OR key LIKE ?", likeKeyword, likeKeyword)
		}
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, 0, err
	}

	return tokens, total, nil
}

// tokenRepository Token 仓库实现
type tokenRepository struct {
	db *gorm.DB
}

// NewTokenRepository 创建 Token 仓库
func NewTokenRepository(db *gorm.DB) TokenRepository {
	return &tokenRepository{db: db}
}

// Create 创建 Token
func (r *tokenRepository) Create(token *model.Token) error {
	return r.db.Create(token).Error
}

// GetByID 根据 ID 获取 Token
func (r *tokenRepository) GetByID(id uuid.UUID) (*model.Token, error) {
	var token model.Token
	if err := r.db.Where("id = ?", id).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &token, nil
}

// GetByKey 根据 Key 获取 Token
func (r *tokenRepository) GetByKey(key string) (*model.Token, error) {
	// 先从缓存获取
	cacheKey := cache.KeyTokenPrefix + key
	var token model.Token
	if err := cache.Get(cacheKey, &token); err == nil {
		return &token, nil
	}

	// 从数据库获取
	if err := r.db.Where("key = ?", key).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// 缓存 Token 信息
	_ = cache.Set(cacheKey, &token, 5*time.Minute)

	return &token, nil
}

// GetByKeyWithUser 根据 Key 获取 Token（包含用户信息）
func (r *tokenRepository) GetByKeyWithUser(key string) (*model.Token, error) {
	// 先从缓存获取
	cacheKey := cache.KeyTokenPrefix + key + ":with_user"
	var token model.Token
	if err := cache.Get(cacheKey, &token); err == nil {
		return &token, nil
	}

	// 从数据库获取
	if err := r.db.Preload("User").Where("key = ?", key).First(&token).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// 缓存 Token 信息
	_ = cache.Set(cacheKey, &token, 5*time.Minute)

	return &token, nil
}

// Update 更新 Token
func (r *tokenRepository) Update(token *model.Token) error {
	err := r.db.Save(token).Error
	if err == nil {
		// 清除缓存
		r.InvalidateCache(token.Key)
	}
	return err
}

// Delete 删除 Token
func (r *tokenRepository) Delete(id uuid.UUID) error {
	// 先获取 Token 以便清除缓存
	var token model.Token
	if err := r.db.Where("id = ?", id).First(&token).Error; err != nil {
		return err
	}

	err := r.db.Delete(&model.Token{}, "id = ?", id).Error
	if err == nil {
		r.InvalidateCache(token.Key)
	}
	return err
}

// ListByUserID 获取用户的 Token 列表
func (r *tokenRepository) ListByUserID(userID uuid.UUID, page, pageSize int) ([]*model.Token, int64, error) {
	var tokens []*model.Token
	var total int64

	query := r.db.Model(&model.Token{}).Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&tokens).Error; err != nil {
		return nil, 0, err
	}

	return tokens, total, nil
}

// DeductQuota 扣除配额
func (r *tokenRepository) DeductQuota(id uuid.UUID, amount decimal.Decimal) error {
	// 先获取 Token
	var token model.Token
	if err := r.db.Where("id = ?", id).First(&token).Error; err != nil {
		return err
	}

	// 如果没有配额限制，直接返回
	if token.RemainQuota == nil {
		return nil
	}

	// 检查配额是否足够
	if token.RemainQuota.LessThan(amount) {
		return ErrInsufficientQuota
	}

	// 扣除配额
	result := r.db.Model(&model.Token{}).
		Where("id = ? AND remain_quota >= ?", id, amount).
		Update("remain_quota", gorm.Expr("remain_quota - ?", amount))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrInsufficientQuota
	}

	// 清除缓存
	r.InvalidateCache(token.Key)

	return nil
}

// UpdateLastUsed 更新最后使用时间
func (r *tokenRepository) UpdateLastUsed(id uuid.UUID) error {
	// 先获取 Key 以便清除缓存（避免 gateway 缓存中的 last_used_at 长时间不更新）
	var token model.Token
	if err := r.db.Select("key").Where("id = ?", id).First(&token).Error; err != nil {
		return err
	}

	now := time.Now()
	err := r.db.Model(&model.Token{}).
		Where("id = ?", id).
		Update("last_used_at", now).Error
	if err == nil {
		r.InvalidateCache(token.Key)
	}
	return err
}

// CountByUserID 统计用户的 Token 数量
func (r *tokenRepository) CountByUserID(userID uuid.UUID) (int64, error) {
	var count int64
	err := r.db.Model(&model.Token{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// InvalidateCache 清除缓存
func (r *tokenRepository) InvalidateCache(key string) {
	_ = cache.Delete(cache.KeyTokenPrefix + key)
	_ = cache.Delete(cache.KeyTokenPrefix + key + ":with_user")
}
