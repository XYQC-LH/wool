package repository

import (
	"errors"
	"time"

	"nexus-api/internal/model"

	"gorm.io/gorm"
)

// AnnouncementRepository 公告仓库接口
type AnnouncementRepository interface {
	Create(announcement *model.Announcement) error
	GetByID(id uint) (*model.Announcement, error)
	Update(announcement *model.Announcement) error
	Delete(id uint) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.Announcement, int64, error)
	ListActive() ([]*model.Announcement, error)
	Publish(id uint) error
	Archive(id uint) error
}

// announcementRepository 公告仓库实现
type announcementRepository struct {
	db *gorm.DB
}

// NewAnnouncementRepository 创建公告仓库
func NewAnnouncementRepository(db *gorm.DB) AnnouncementRepository {
	return &announcementRepository{db: db}
}

// Create 创建公告
func (r *announcementRepository) Create(announcement *model.Announcement) error {
	return r.db.Create(announcement).Error
}

// GetByID 根据 ID 获取公告
func (r *announcementRepository) GetByID(id uint) (*model.Announcement, error) {
	var announcement model.Announcement
	if err := r.db.Where("id = ?", id).First(&announcement).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &announcement, nil
}

// Update 更新公告
func (r *announcementRepository) Update(announcement *model.Announcement) error {
	return r.db.Save(announcement).Error
}

// Delete 删除公告
func (r *announcementRepository) Delete(id uint) error {
	return r.db.Delete(&model.Announcement{}, "id = ?", id).Error
}

// List 获取公告列表
func (r *announcementRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.Announcement, int64, error) {
	var announcements []*model.Announcement
	var total int64

	query := r.db.Model(&model.Announcement{})

	// 应用过滤条件
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if announcementType, ok := filters["type"]; ok {
		query = query.Where("type = ?", announcementType)
	}
	if keyword, ok := filters["keyword"]; ok && keyword != "" {
		query = query.Where("title LIKE ? OR content LIKE ?", "%"+keyword.(string)+"%", "%"+keyword.(string)+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("priority DESC, created_at DESC").Find(&announcements).Error; err != nil {
		return nil, 0, err
	}

	return announcements, total, nil
}

// ListActive 获取活跃的公告列表（已发布且未过期）
func (r *announcementRepository) ListActive() ([]*model.Announcement, error) {
	var announcements []*model.Announcement
	now := time.Now()

	if err := r.db.Where("status = ? AND (expires_at IS NULL OR expires_at > ?)",
		model.AnnouncementStatusPublished, now).
		Order("priority DESC, created_at DESC").
		Find(&announcements).Error; err != nil {
		return nil, err
	}

	return announcements, nil
}

// Publish 发布公告
func (r *announcementRepository) Publish(id uint) error {
	now := time.Now()
	return r.db.Model(&model.Announcement{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":       model.AnnouncementStatusPublished,
			"published_at": now,
		}).Error
}

// Archive 归档公告
func (r *announcementRepository) Archive(id uint) error {
	return r.db.Model(&model.Announcement{}).
		Where("id = ?", id).
		Update("status", model.AnnouncementStatusArchived).Error
}
