package service

import (
	"context"
	"fmt"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	"nexus-api/internal/service/scheduler"
)

type providerEntityReader interface {
	GetByID(ctx context.Context, id uint) (*model.ModelProvider, error)
}

type ProviderCapabilityService interface {
	Create(ctx context.Context, req *CreateProviderCapabilityRequest) (*model.ProviderCapability, error)
	Update(ctx context.Context, id uint, req *UpdateProviderCapabilityRequest) (*model.ProviderCapability, error)
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ProviderCapability, error)

	List(ctx context.Context, params *ProviderCapabilityQueryParams) ([]*model.ProviderCapability, int64, error)
	ListByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error)
	BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error
	GetSummary(ctx context.Context, providerID uint) (*model.ProviderCapabilitySummary, error)
	ValidateConstraints(ctx context.Context, req *ValidateProviderConstraintsRequest) (*model.ProviderCapabilityValidationResult, error)
}

type CreateProviderCapabilityRequest struct {
	ProviderID  uint
	Operation   string
	Constraints model.JSON
	IsEnabled   *bool
}

type UpdateProviderCapabilityRequest struct {
	Operation   *string
	Constraints *model.JSON
	IsEnabled   *bool
}

type ProviderCapabilityQueryParams struct {
	ProviderID uint
	Operation  string
	IsEnabled  *bool
	Page       int
	PageSize   int
}

type ValidateProviderConstraintsRequest struct {
	ProviderID  uint
	Operation   string
	Constraints map[string]interface{}
}

type providerCapabilityService struct {
	repo           repository.ProviderCapabilityRepository
	providerReader providerEntityReader
}

func NewProviderCapabilityService(
	repo repository.ProviderCapabilityRepository,
	providerReader providerEntityReader,
) ProviderCapabilityService {
	return &providerCapabilityService{
		repo:           repo,
		providerReader: providerReader,
	}
}

func (s *providerCapabilityService) Create(ctx context.Context, req *CreateProviderCapabilityRequest) (*model.ProviderCapability, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if req.ProviderID == 0 {
		return nil, fmt.Errorf("provider_id 不能为空")
	}
	operation := model.NormalizeOperation(strings.TrimSpace(req.Operation))
	if operation == "" {
		return nil, fmt.Errorf("operation 不能为空")
	}

	provider, err := s.providerReader.GetByID(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("加载源头失败: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("源头不存在: %d", req.ProviderID)
	}

	existing, err := s.repo.GetByProviderAndOperation(ctx, req.ProviderID, operation)
	if err != nil {
		return nil, fmt.Errorf("检查能力配置失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("该源头能力已存在")
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}

	constraints := model.JSON{}
	if req.Constraints != nil {
		constraints = req.Constraints
	}

	entity := &model.ProviderCapability{
		ProviderID:  req.ProviderID,
		Operation:   operation,
		Constraints: constraints,
		IsEnabled:   enabled,
	}
	if err := s.repo.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("创建能力配置失败: %w", err)
	}

	return s.repo.GetByID(ctx, entity.ID)
}

func (s *providerCapabilityService) Update(ctx context.Context, id uint, req *UpdateProviderCapabilityRequest) (*model.ProviderCapability, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	entity, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("加载能力配置失败: %w", err)
	}
	if entity == nil {
		return nil, fmt.Errorf("能力配置不存在: %d", id)
	}

	if req.Operation != nil {
		operation := model.NormalizeOperation(strings.TrimSpace(*req.Operation))
		if operation == "" {
			return nil, fmt.Errorf("operation 不能为空")
		}
		conflict, err := s.repo.GetByProviderAndOperation(ctx, entity.ProviderID, operation)
		if err != nil {
			return nil, fmt.Errorf("检查能力配置冲突失败: %w", err)
		}
		if conflict != nil && conflict.ID != entity.ID {
			return nil, fmt.Errorf("该源头能力已存在")
		}
		entity.Operation = operation
	}

	if req.Constraints != nil {
		entity.Constraints = *req.Constraints
		if entity.Constraints == nil {
			entity.Constraints = model.JSON{}
		}
	}

	if req.IsEnabled != nil {
		entity.IsEnabled = *req.IsEnabled
	}

	if err := s.repo.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("更新能力配置失败: %w", err)
	}
	return s.repo.GetByID(ctx, entity.ID)
}

func (s *providerCapabilityService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

func (s *providerCapabilityService) GetByID(ctx context.Context, id uint) (*model.ProviderCapability, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *providerCapabilityService) List(ctx context.Context, params *ProviderCapabilityQueryParams) ([]*model.ProviderCapability, int64, error) {
	if params == nil {
		params = &ProviderCapabilityQueryParams{}
	}
	return s.repo.List(ctx, params.ProviderID, params.Operation, params.IsEnabled, params.Page, params.PageSize)
}

func (s *providerCapabilityService) ListByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error) {
	if providerID == 0 {
		return nil, fmt.Errorf("provider_id 不能为空")
	}

	provider, err := s.providerReader.GetByID(ctx, providerID)
	if err != nil {
		return nil, fmt.Errorf("加载源头失败: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("源头不存在: %d", providerID)
	}
	return s.repo.GetByProviderAll(ctx, providerID)
}

func (s *providerCapabilityService) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	return s.repo.BatchUpdateEnabled(ctx, ids, enabled)
}

func (s *providerCapabilityService) GetSummary(ctx context.Context, providerID uint) (*model.ProviderCapabilitySummary, error) {
	return s.repo.GetSummary(ctx, providerID)
}

func (s *providerCapabilityService) ValidateConstraints(ctx context.Context, req *ValidateProviderConstraintsRequest) (*model.ProviderCapabilityValidationResult, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}
	if req.ProviderID == 0 {
		return nil, fmt.Errorf("provider_id 不能为空")
	}

	operation := model.NormalizeOperation(strings.TrimSpace(req.Operation))
	if operation == "" {
		return nil, fmt.Errorf("operation 不能为空")
	}

	provider, err := s.providerReader.GetByID(ctx, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("加载源头失败: %w", err)
	}
	if provider == nil {
		return &model.ProviderCapabilityValidationResult{
			ProviderID: req.ProviderID,
			Operation:  operation,
			Matched:    false,
			Reason:     "provider_not_found",
		}, nil
	}

	capability, err := s.repo.GetByProviderAndOperation(ctx, req.ProviderID, operation)
	if err != nil {
		return nil, fmt.Errorf("加载能力配置失败: %w", err)
	}
	if capability == nil {
		return &model.ProviderCapabilityValidationResult{
			ProviderID: req.ProviderID,
			Operation:  operation,
			Matched:    false,
			Reason:     "capability_not_found",
		}, nil
	}

	result := &model.ProviderCapabilityValidationResult{
		ProviderID:   req.ProviderID,
		Operation:    operation,
		CapabilityID: capability.ID,
		Matched:      false,
	}
	if !capability.IsEnabled {
		result.Reason = "capability_disabled"
		return result, nil
	}

	constraints := req.Constraints
	if len(constraints) == 0 {
		result.Matched = true
		return result, nil
	}

	matcher := scheduler.NewCapabilityMatcher(s.repo)
	filtered, rejects, err := matcher.MatchProviders(
		scheduler.WithCapabilityConstraints(ctx, constraints),
		operation,
		[]*model.ModelProvider{provider},
	)
	if err != nil {
		return nil, fmt.Errorf("能力约束校验失败: %w", err)
	}
	if len(filtered) > 0 {
		result.Matched = true
		return result, nil
	}

	if len(rejects) > 0 {
		result.Reason = rejects[0].Reason
	} else {
		result.Reason = "constraints_not_matched"
	}
	return result, nil
}
