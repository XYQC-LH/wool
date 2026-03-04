package service

import (
	"errors"
	"fmt"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

// AnnouncementService 公告服务接口
type AnnouncementService interface {
	// 用户端接口
	ListActive() ([]*model.AnnouncementResponse, error)
	ListPublic() ([]*model.Announcement, error)

	// 管理员接口
	AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AnnouncementResponse, *model.Pagination, error)
	AdminGetByID(id uint) (*model.AnnouncementResponse, error)
	Create(req *model.CreateAnnouncementRequest) (*model.AnnouncementResponse, error)
	Update(id uint, req *model.UpdateAnnouncementRequest) (*model.AnnouncementResponse, error)
	Delete(id uint) error
	Publish(id uint) error
	Archive(id uint) error
}

// announcementService 公告服务实现
type announcementService struct {
	announcementRepo repository.AnnouncementRepository
}

// NewAnnouncementService 创建公告服务
func NewAnnouncementService(announcementRepo repository.AnnouncementRepository) AnnouncementService {
	return &announcementService{
		announcementRepo: announcementRepo,
	}
}

// ListActive 获取活跃的公告列表
func (s *announcementService) ListActive() ([]*model.AnnouncementResponse, error) {
	announcements, err := s.announcementRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("获取公告列表失败: %w", err)
	}

	responses := make([]*model.AnnouncementResponse, len(announcements))
	for i, announcement := range announcements {
		responses[i] = announcement.ToResponse()
	}

	return responses, nil
}

// ListPublic 获取公开的公告列表（返回原始公告对象）
func (s *announcementService) ListPublic() ([]*model.Announcement, error) {
	announcements, err := s.announcementRepo.ListActive()
	if err != nil {
		return nil, fmt.Errorf("获取公告列表失败: %w", err)
	}

	return announcements, nil
}

// AdminList 管理员获取公告列表
func (s *announcementService) AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AnnouncementResponse, *model.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	announcements, total, err := s.announcementRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("获取公告列表失败: %w", err)
	}

	responses := make([]*model.AnnouncementResponse, len(announcements))
	for i, announcement := range announcements {
		responses[i] = announcement.ToResponse()
	}

	pagination := model.NewPagination(page, pageSize, total)

	return responses, pagination, nil
}

// AdminGetByID 管理员获取公告详情
func (s *announcementService) AdminGetByID(id uint) (*model.AnnouncementResponse, error) {
	announcement, err := s.announcementRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("获取公告失败: %w", err)
	}
	if announcement == nil {
		return nil, errors.New("公告不存在")
	}

	return announcement.ToResponse(), nil
}

// Create 创建公告
func (s *announcementService) Create(req *model.CreateAnnouncementRequest) (*model.AnnouncementResponse, error) {
	// 设置默认值
	if req.Type == "" {
		req.Type = model.AnnouncementTypeInfo
	}
	if req.Status == "" {
		req.Status = model.AnnouncementStatusDraft
	}

	announcement := &model.Announcement{
		Title:     req.Title,
		Content:   req.Content,
		Type:      req.Type,
		Status:    req.Status,
		Priority:  req.Priority,
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.announcementRepo.Create(announcement); err != nil {
		return nil, fmt.Errorf("创建公告失败: %w", err)
	}

	return announcement.ToResponse(), nil
}

// Update 更新公告
func (s *announcementService) Update(id uint, req *model.UpdateAnnouncementRequest) (*model.AnnouncementResponse, error) {
	announcement, err := s.announcementRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("获取公告失败: %w", err)
	}
	if announcement == nil {
		return nil, errors.New("公告不存在")
	}

	// 更新字段
	if req.Title != nil {
		announcement.Title = *req.Title
	}
	if req.Content != nil {
		announcement.Content = *req.Content
	}
	if req.Type != "" {
		announcement.Type = req.Type
	}
	if req.Status != "" {
		announcement.Status = req.Status
	}
	if req.Priority != nil {
		announcement.Priority = *req.Priority
	}
	if req.ExpiresAt != nil {
		announcement.ExpiresAt = req.ExpiresAt
	}

	if err := s.announcementRepo.Update(announcement); err != nil {
		return nil, fmt.Errorf("更新公告失败: %w", err)
	}

	return announcement.ToResponse(), nil
}

// Delete 删除公告
func (s *announcementService) Delete(id uint) error {
	if err := s.announcementRepo.Delete(id); err != nil {
		return fmt.Errorf("删除公告失败: %w", err)
	}
	return nil
}

// Publish 发布公告
func (s *announcementService) Publish(id uint) error {
	if err := s.announcementRepo.Publish(id); err != nil {
		return fmt.Errorf("发布公告失败: %w", err)
	}
	return nil
}

// Archive 归档公告
func (s *announcementService) Archive(id uint) error {
	if err := s.announcementRepo.Archive(id); err != nil {
		return fmt.Errorf("归档公告失败: %w", err)
	}
	return nil
}
