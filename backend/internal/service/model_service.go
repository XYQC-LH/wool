package service

import (
	"errors"
	"fmt"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/shopspring/decimal"
)

// ModelService 模型服务接口
type ModelService interface {
	// 用户端接口
	ListEnabled() ([]*model.ModelResponse, error)
	ListPublic() ([]*model.Model, error)
	GetByID(id string) (*model.ModelResponse, error)

	// 管理员接口
	AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.ModelResponse, *model.Pagination, error)
	AdminGetByID(id string) (*model.ModelResponse, error)
	Create(req *model.CreateModelRequest) (*model.ModelResponse, error)
	Update(id string, req *model.UpdateModelRequest) (*model.ModelResponse, error)
	Delete(id string) error
	UpdateStatus(id string, enabled bool) error
}

// modelService 模型服务实现
type modelService struct {
	modelRepo repository.ModelRepository
}

// NewModelService 创建模型服务
func NewModelService(modelRepo repository.ModelRepository) ModelService {
	return &modelService{
		modelRepo: modelRepo,
	}
}

// ListEnabled 获取启用的模型列表
func (s *modelService) ListEnabled() ([]*model.ModelResponse, error) {
	models, err := s.modelRepo.ListEnabled()
	if err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}

	responses := make([]*model.ModelResponse, len(models))
	for i, m := range models {
		responses[i] = m.ToResponse()
	}

	return responses, nil
}

// ListPublic 获取公开的模型列表（返回原始模型对象）
func (s *modelService) ListPublic() ([]*model.Model, error) {
	models, err := s.modelRepo.ListEnabled()
	if err != nil {
		return nil, fmt.Errorf("获取模型列表失败: %w", err)
	}

	return models, nil
}

// GetByID 获取模型详情
func (s *modelService) GetByID(id string) (*model.ModelResponse, error) {
	m, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("获取模型失败: %w", err)
	}
	if m == nil {
		return nil, errors.New("模型不存在")
	}

	return m.ToResponse(), nil
}

// AdminList 管理员获取模型列表
func (s *modelService) AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.ModelResponse, *model.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	models, total, err := s.modelRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("获取模型列表失败: %w", err)
	}

	responses := make([]*model.ModelResponse, len(models))
	for i, m := range models {
		responses[i] = m.ToResponse()
	}

	pagination := model.NewPagination(page, pageSize, total)

	return responses, pagination, nil
}

// AdminGetByID 管理员获取模型详情
func (s *modelService) AdminGetByID(id string) (*model.ModelResponse, error) {
	m, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("获取模型失败: %w", err)
	}
	if m == nil {
		return nil, errors.New("模型不存在")
	}

	return m.ToResponse(), nil
}

// Create 创建模型
func (s *modelService) Create(req *model.CreateModelRequest) (*model.ModelResponse, error) {
	// 检查模型名称是否已存在
	existing, _ := s.modelRepo.GetByID(req.Name)
	if existing != nil {
		return nil, errors.New("模型名称已存在")
	}

	m := &model.Model{
		ID:          req.Name,
		Name:        req.Name,
		DisplayName: req.DisplayName,
		Provider:    req.Provider,
		InputPrice:  decimal.NewFromFloat(req.InputPrice),
		OutputPrice: decimal.NewFromFloat(req.OutputPrice),
		MaxTokens:   req.MaxTokens,
		MaxContext:  req.MaxContext,
		Status:      model.ModelStatusActive,
		Enabled:     true,
		Description: req.Description,
	}

	if err := s.modelRepo.Create(m); err != nil {
		return nil, fmt.Errorf("创建模型失败: %w", err)
	}

	return m.ToResponse(), nil
}

// Update 更新模型
func (s *modelService) Update(id string, req *model.UpdateModelRequest) (*model.ModelResponse, error) {
	m, err := s.modelRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("获取模型失败: %w", err)
	}
	if m == nil {
		return nil, errors.New("模型不存在")
	}

	// 更新字段
	if req.DisplayName != nil {
		m.DisplayName = *req.DisplayName
	}
	if req.InputPrice != nil {
		m.InputPrice = decimal.NewFromFloat(*req.InputPrice)
	}
	if req.OutputPrice != nil {
		m.OutputPrice = decimal.NewFromFloat(*req.OutputPrice)
	}
	if req.MaxTokens != nil {
		m.MaxTokens = *req.MaxTokens
	}
	if req.MaxContext != nil {
		m.MaxContext = *req.MaxContext
	}
	if req.Status != nil {
		m.Status = *req.Status
	}
	if req.Description != nil {
		m.Description = req.Description
	}

	if err := s.modelRepo.Update(m); err != nil {
		return nil, fmt.Errorf("更新模型失败: %w", err)
	}

	return m.ToResponse(), nil
}

// Delete 删除模型
func (s *modelService) Delete(id string) error {
	if err := s.modelRepo.Delete(id); err != nil {
		return fmt.Errorf("删除模型失败: %w", err)
	}
	return nil
}

// UpdateStatus 更新模型状态
func (s *modelService) UpdateStatus(id string, enabled bool) error {
	if err := s.modelRepo.UpdateStatus(id, enabled); err != nil {
		return fmt.Errorf("更新模型状态失败: %w", err)
	}
	return nil
}
