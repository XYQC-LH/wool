package service

import (
	"context"
	"fmt"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

type modelRouteModelReader interface {
	GetByID(id string) (*model.Model, error)
}

type modelRouteProviderReader interface {
	GetByID(ctx context.Context, id uint) (*model.ModelProvider, error)
}

type ModelRouteService interface {
	Create(ctx context.Context, req *CreateModelRouteRequest) (*model.ModelRoute, error)
	Update(ctx context.Context, id uint, req *UpdateModelRouteRequest) (*model.ModelRoute, error)
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ModelRoute, error)

	List(ctx context.Context, params *ModelRouteQueryParams) ([]*model.ModelRoute, int64, error)
	BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error
	BatchUpdatePriority(ctx context.Context, items []ModelRoutePriorityItem) error
	GetStats(ctx context.Context, operation string, modelID string) (*model.ModelRouteStats, error)
}

type CreateModelRouteRequest struct {
	Operation   string
	ModelID     string
	ProviderID  uint
	Priority    int
	IsEnabled   *bool
	Description string
}

type UpdateModelRouteRequest struct {
	Operation   *string
	ModelID     *string
	ProviderID  *uint
	Priority    *int
	IsEnabled   *bool
	Description *string
}

type ModelRouteQueryParams struct {
	Operation  string
	ModelID    string
	ProviderID uint
	IsEnabled  *bool
	Page       int
	PageSize   int
}

type ModelRoutePriorityItem struct {
	ID       uint
	Priority int
}

type modelRouteService struct {
	routeRepo      repository.ModelRouteRepository
	modelReader    modelRouteModelReader
	providerReader modelRouteProviderReader
}

func NewModelRouteService(
	routeRepo repository.ModelRouteRepository,
	modelReader modelRouteModelReader,
	providerReader modelRouteProviderReader,
) ModelRouteService {
	return &modelRouteService{
		routeRepo:      routeRepo,
		modelReader:    modelReader,
		providerReader: providerReader,
	}
}

func (s *modelRouteService) Create(ctx context.Context, req *CreateModelRouteRequest) (*model.ModelRoute, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	operation := model.NormalizeOperation(req.Operation)
	modelID := strings.TrimSpace(req.ModelID)
	if err := s.validateBinding(ctx, operation, modelID, req.ProviderID); err != nil {
		return nil, err
	}

	existing, err := s.routeRepo.GetByModelAndProvider(ctx, operation, modelID, req.ProviderID)
	if err != nil {
		return nil, fmt.Errorf("检查路由映射失败: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("该模型路由映射已存在")
	}

	enabled := true
	if req.IsEnabled != nil {
		enabled = *req.IsEnabled
	}
	priority := req.Priority
	if priority <= 0 {
		priority = 100
	}

	entity := &model.ModelRoute{
		RouteKey:    model.BuildModelRouteKey(operation, modelID, req.ProviderID),
		Operation:   operation,
		ModelID:     modelID,
		ProviderID:  req.ProviderID,
		Priority:    priority,
		IsEnabled:   enabled,
		Description: strings.TrimSpace(req.Description),
	}

	if err := s.routeRepo.Create(ctx, entity); err != nil {
		return nil, fmt.Errorf("创建路由映射失败: %w", err)
	}
	InvalidateGatewayResponseCache(operation, modelID)
	return s.routeRepo.GetByID(ctx, entity.ID)
}

func (s *modelRouteService) Update(ctx context.Context, id uint, req *UpdateModelRouteRequest) (*model.ModelRoute, error) {
	if req == nil {
		return nil, fmt.Errorf("请求不能为空")
	}

	entity, err := s.routeRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("加载路由映射失败: %w", err)
	}
	if entity == nil {
		return nil, fmt.Errorf("路由映射不存在: %d", id)
	}
	previousOperation := entity.Operation
	previousModelID := entity.ModelID

	operation := model.NormalizeOperation(entity.Operation)
	modelID := strings.TrimSpace(entity.ModelID)
	providerID := entity.ProviderID

	if req.Operation != nil {
		operation = model.NormalizeOperation(*req.Operation)
	}
	if req.ModelID != nil {
		modelID = strings.TrimSpace(*req.ModelID)
	}
	if req.ProviderID != nil {
		providerID = *req.ProviderID
	}

	if err := s.validateBinding(ctx, operation, modelID, providerID); err != nil {
		return nil, err
	}

	if operation != entity.Operation || modelID != entity.ModelID || providerID != entity.ProviderID {
		conflict, err := s.routeRepo.GetByModelAndProvider(ctx, operation, modelID, providerID)
		if err != nil {
			return nil, fmt.Errorf("检查路由映射冲突失败: %w", err)
		}
		if conflict != nil && conflict.ID != entity.ID {
			return nil, fmt.Errorf("该模型路由映射已存在")
		}
	}

	entity.Operation = operation
	entity.ModelID = modelID
	entity.ProviderID = providerID
	entity.RouteKey = model.BuildModelRouteKey(operation, modelID, providerID)

	if req.Priority != nil {
		if *req.Priority <= 0 {
			return nil, fmt.Errorf("priority 必须大于 0")
		}
		entity.Priority = *req.Priority
	}
	if req.IsEnabled != nil {
		entity.IsEnabled = *req.IsEnabled
	}
	if req.Description != nil {
		entity.Description = strings.TrimSpace(*req.Description)
	}

	if err := s.routeRepo.Update(ctx, entity); err != nil {
		return nil, fmt.Errorf("更新路由映射失败: %w", err)
	}
	InvalidateGatewayResponseCache(previousOperation, previousModelID)
	InvalidateGatewayResponseCache(entity.Operation, entity.ModelID)
	return s.routeRepo.GetByID(ctx, entity.ID)
}

func (s *modelRouteService) Delete(ctx context.Context, id uint) error {
	existing, _ := s.routeRepo.GetByID(ctx, id)
	if err := s.routeRepo.Delete(ctx, id); err != nil {
		return err
	}
	if existing != nil {
		InvalidateGatewayResponseCache(existing.Operation, existing.ModelID)
	}
	return nil
}

func (s *modelRouteService) GetByID(ctx context.Context, id uint) (*model.ModelRoute, error) {
	return s.routeRepo.GetByID(ctx, id)
}

func (s *modelRouteService) List(ctx context.Context, params *ModelRouteQueryParams) ([]*model.ModelRoute, int64, error) {
	if params == nil {
		params = &ModelRouteQueryParams{}
	}
	return s.routeRepo.List(
		ctx,
		params.Operation,
		strings.TrimSpace(params.ModelID),
		params.ProviderID,
		params.IsEnabled,
		params.Page,
		params.PageSize,
	)
}

func (s *modelRouteService) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	filtered := normalizeRouteIDs(ids)
	if len(filtered) == 0 {
		return fmt.Errorf("ids 不能为空")
	}
	affectedRouteKeys := make(map[string]struct{}, len(filtered))
	for _, id := range filtered {
		existing, err := s.routeRepo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("加载路由映射失败: %w", err)
		}
		if existing == nil {
			return fmt.Errorf("路由映射不存在: %d", id)
		}
		affectedRouteKeys[model.NormalizeOperation(existing.Operation)+":"+strings.TrimSpace(existing.ModelID)] = struct{}{}
	}

	if err := s.routeRepo.BatchUpdateEnabled(ctx, filtered, enabled); err != nil {
		return err
	}
	for key := range affectedRouteKeys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		InvalidateGatewayResponseCache(parts[0], parts[1])
	}
	return nil
}

func (s *modelRouteService) BatchUpdatePriority(ctx context.Context, items []ModelRoutePriorityItem) error {
	if len(items) == 0 {
		return fmt.Errorf("items 不能为空")
	}

	updates := make(map[uint]int, len(items))
	for _, item := range items {
		if item.ID == 0 {
			return fmt.Errorf("id 不能为空")
		}
		if item.Priority <= 0 {
			return fmt.Errorf("priority 必须大于 0")
		}
		updates[item.ID] = item.Priority
	}

	affectedRouteKeys := make(map[string]struct{}, len(items))
	for id := range updates {
		existing, err := s.routeRepo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("加载路由映射失败: %w", err)
		}
		if existing == nil {
			return fmt.Errorf("路由映射不存在: %d", id)
		}
		affectedRouteKeys[model.NormalizeOperation(existing.Operation)+":"+strings.TrimSpace(existing.ModelID)] = struct{}{}
	}

	if err := s.routeRepo.BatchUpdatePriority(ctx, updates); err != nil {
		return err
	}
	for key := range affectedRouteKeys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		InvalidateGatewayResponseCache(parts[0], parts[1])
	}
	return nil
}

func (s *modelRouteService) GetStats(ctx context.Context, operation string, modelID string) (*model.ModelRouteStats, error) {
	return s.routeRepo.GetStats(ctx, operation, strings.TrimSpace(modelID))
}

func (s *modelRouteService) validateBinding(ctx context.Context, operation string, modelID string, providerID uint) error {
	if strings.TrimSpace(modelID) == "" {
		return fmt.Errorf("model_id 不能为空")
	}
	if providerID == 0 {
		return fmt.Errorf("provider_id 不能为空")
	}
	if s.modelReader == nil {
		return fmt.Errorf("model reader 未配置")
	}
	modelEntity, err := s.modelReader.GetByID(modelID)
	if err != nil {
		return fmt.Errorf("加载模型失败: %w", err)
	}
	if modelEntity == nil {
		return fmt.Errorf("模型不存在: %s", modelID)
	}
	if s.providerReader == nil {
		return fmt.Errorf("provider reader 未配置")
	}
	provider, err := s.providerReader.GetByID(ctx, providerID)
	if err != nil {
		return fmt.Errorf("加载源头失败: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("源头不存在: %d", providerID)
	}

	if model.NormalizeOperation(provider.Operation) != model.NormalizeOperation(operation) {
		return fmt.Errorf("源头与 operation 不匹配")
	}
	if strings.TrimSpace(provider.ModelID) != strings.TrimSpace(modelID) {
		return fmt.Errorf("源头与 model_id 不匹配")
	}
	return nil
}

func normalizeRouteIDs(ids []uint) []uint {
	if len(ids) == 0 {
		return nil
	}
	result := make([]uint, 0, len(ids))
	seen := make(map[uint]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		result = append(result, id)
	}
	return result
}
