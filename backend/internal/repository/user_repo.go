package repository

import (
	"errors"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// UserRepository 用户仓库接口
type UserRepository interface {
	Create(user *model.User) error
	GetByID(id uuid.UUID) (*model.User, error)
	GetByUsername(username string) (*model.User, error)
	GetByEmail(email string) (*model.User, error)
	Update(user *model.User) error
	Delete(id uuid.UUID) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.User, int64, error)
	UpdateBalance(id uuid.UUID, amount decimal.Decimal) error
	DeductBalance(id uuid.UUID, amount decimal.Decimal) error
	UpdateLastLogin(id uuid.UUID) error
	CountByStatus(status model.UserStatus) (int64, error)
	GetActiveUsersCount(since time.Time) (int64, error)
}

// userRepository 用户仓库实现
type userRepository struct {
	db *gorm.DB
}

// NewUserRepository 创建用户仓库
func NewUserRepository(db *gorm.DB) UserRepository {
	return &userRepository{db: db}
}

// Create 创建用户
func (r *userRepository) Create(user *model.User) error {
	return r.db.Create(user).Error
}

// GetByID 根据 ID 获取用户
func (r *userRepository) GetByID(id uuid.UUID) (*model.User, error) {
	// 先从缓存获取
	cacheKey := cache.KeyUserPrefix + id.String()
	var user model.User
	if err := cache.Get(cacheKey, &user); err == nil {
		return &user, nil
	}

	// 从数据库获取
	if err := r.db.Where("id = ?", id).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}

	// 缓存用户信息
	_ = cache.Set(cacheKey, &user, 10*time.Minute)

	return &user, nil
}

// GetByUsername 根据用户名获取用户
func (r *userRepository) GetByUsername(username string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("username = ?", username).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// GetByEmail 根据邮箱获取用户
func (r *userRepository) GetByEmail(email string) (*model.User, error) {
	var user model.User
	if err := r.db.Where("email = ?", email).First(&user).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &user, nil
}

// Update 更新用户
func (r *userRepository) Update(user *model.User) error {
	err := r.db.Save(user).Error
	if err == nil {
		// 清除缓存
		_ = cache.Delete(cache.KeyUserPrefix + user.ID.String())
	}
	return err
}

// Delete 删除用户
func (r *userRepository) Delete(id uuid.UUID) error {
	err := r.db.Delete(&model.User{}, "id = ?", id).Error
	if err == nil {
		// 清除缓存
		_ = cache.Delete(cache.KeyUserPrefix + id.String())
	}
	return err
}

// List 获取用户列表
func (r *userRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.User, int64, error) {
	var users []*model.User
	var total int64

	query := r.db.Model(&model.User{})

	// 应用过滤条件
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if role, ok := filters["role"]; ok {
		query = query.Where("role = ?", role)
	}
	if keyword, ok := filters["keyword"]; ok && keyword != "" {
		query = query.Where("username LIKE ? OR email LIKE ?", "%"+keyword.(string)+"%", "%"+keyword.(string)+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&users).Error; err != nil {
		return nil, 0, err
	}

	return users, total, nil
}

// UpdateBalance 更新余额（增加）
func (r *userRepository) UpdateBalance(id uuid.UUID, amount decimal.Decimal) error {
	err := r.db.Model(&model.User{}).
		Where("id = ?", id).
		Update("balance", gorm.Expr("balance + ?", amount)).Error
	if err == nil {
		// 清除缓存
		_ = cache.Delete(cache.KeyUserPrefix + id.String())
	}
	return err
}

// DeductBalance 扣除余额
func (r *userRepository) DeductBalance(id uuid.UUID, amount decimal.Decimal) error {
	result := r.db.Model(&model.User{}).
		Where("id = ? AND balance >= ?", id, amount).
		Update("balance", gorm.Expr("balance - ?", amount))

	if result.Error != nil {
		return result.Error
	}

	if result.RowsAffected == 0 {
		return ErrInsufficientBalance
	}

	// 清除缓存
	_ = cache.Delete(cache.KeyUserPrefix + id.String())

	return nil
}

// UpdateLastLogin 更新最后登录时间
func (r *userRepository) UpdateLastLogin(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.User{}).
		Where("id = ?", id).
		Update("last_login_at", now).Error
}

// CountByStatus 按状态统计用户数
func (r *userRepository) CountByStatus(status model.UserStatus) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).Where("status = ?", status).Count(&count).Error
	return count, err
}

// GetActiveUsersCount 获取活跃用户数
func (r *userRepository) GetActiveUsersCount(since time.Time) (int64, error) {
	var count int64
	err := r.db.Model(&model.User{}).
		Where("last_login_at >= ? AND status = ?", since, model.UserStatusActive).
		Count(&count).Error
	return count, err
}
