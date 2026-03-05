package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	"nexus-api/internal/service/scheduler"

	"github.com/shopspring/decimal"
)

// ModelProviderService 妯″瀷婧愬ご鏈嶅姟鎺ュ彛
type ModelProviderService interface {
	// 鍩虹 CRUD
	Create(ctx context.Context, req *CreateProviderRequest) (*model.ModelProvider, error)
	Update(ctx context.Context, id uint, req *UpdateProviderRequest) (*model.ModelProvider, error)
	Delete(ctx context.Context, id uint) error
	GetByID(ctx context.Context, id uint) (*model.ModelProviderResponse, error)

	// 鏌ヨ
	List(ctx context.Context, params *model.ProviderQueryParams) ([]*model.ModelProviderResponse, int64, error)
	GetByModelID(ctx context.Context, operation string, modelID string) ([]*model.ModelProviderResponse, error)
	GetByChannelID(ctx context.Context, channelID uint) ([]*model.ModelProviderResponse, error)

	// 内部使用：获取带关联的实体（用于测试/调度等需要访问 Channel/BaseURL 的场景）
	GetEntityByID(ctx context.Context, id uint) (*model.ModelProvider, error)

	// 鐘舵€佺鐞?
	Enable(ctx context.Context, id uint) error
	Disable(ctx context.Context, id uint) error
	BatchUpdateStatus(ctx context.Context, ids []uint, status model.ProviderStatus) error

	// 鐔旀柇绠＄悊
	OpenCircuit(ctx context.Context, id uint, duration time.Duration, reason string) error
	CloseCircuit(ctx context.Context, id uint) error
	GetCircuitInfo(ctx context.Context, id uint) (*scheduler.CircuitInfo, error)

	// 鍋ュ悍绠＄悊
	GetProviderHealth(ctx context.Context, id uint) (*scheduler.ProviderHealth, error)
	GetModelHealth(ctx context.Context, operation string, modelID string) (*scheduler.ModelHealth, error)
	GetHealthSummary(ctx context.Context) (*scheduler.HealthSummary, error)

	// 璋冨害鐩稿叧
	GetAvailableProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error)
	SelectBestProvider(ctx context.Context, operation string, modelID string, strategy scheduler.SelectionStrategy) (*model.ModelProvider, error)

	// 鎵归噺鎿嶄綔
	BatchCreate(ctx context.Context, reqs []*CreateProviderRequest) ([]*model.ModelProvider, error)
	SyncFromChannelModels(ctx context.Context, channelID uint) error
}

// CreateProviderRequest 鍒涘缓婧愬ご璇锋眰
type CreateProviderRequest struct {
	Operation                 string               `json:"operation,omitempty"`
	ModelID                   string               `json:"model_id" binding:"required"`
	ChannelID                 uint                 `json:"channel_id" binding:"required"`
	UpstreamModelName         string               `json:"upstream_model_name" binding:"required,min=1,max=100"`
	ActualCostPer1kInput      decimal.Decimal      `json:"actual_cost_per_1k_input" binding:"required"`
	ActualCostPer1kOutput     decimal.Decimal      `json:"actual_cost_per_1k_output" binding:"required"`
	IsCostPriority            *bool                `json:"is_cost_priority,omitempty"`
	Priority                  int                  `json:"priority"`
	Weight                    int                  `json:"weight"`
	ConnectTimeoutMs          int                  `json:"connect_timeout_ms"`
	AttemptTimeoutMs          int                  `json:"attempt_timeout_ms"`
	StreamFirstChunkTimeoutMs int                  `json:"stream_first_chunk_timeout_ms"`
	FailureThreshold          int                  `json:"failure_threshold"`
	RecoveryTimeoutSeconds    int                  `json:"recovery_timeout_seconds"`
	Status                    model.ProviderStatus `json:"status"`
}

// UpdateProviderRequest 鏇存柊婧愬ご璇锋眰
type UpdateProviderRequest struct {
	ActualCostPer1kInput      *decimal.Decimal      `json:"actual_cost_per_1k_input,omitempty"`
	ActualCostPer1kOutput     *decimal.Decimal      `json:"actual_cost_per_1k_output,omitempty"`
	UpstreamModelName         *string               `json:"upstream_model_name,omitempty"`
	IsCostPriority            *bool                 `json:"is_cost_priority,omitempty"`
	Priority                  *int                  `json:"priority,omitempty"`
	Weight                    *int                  `json:"weight,omitempty"`
	ConnectTimeoutMs          *int                  `json:"connect_timeout_ms,omitempty"`
	AttemptTimeoutMs          *int                  `json:"attempt_timeout_ms,omitempty"`
	StreamFirstChunkTimeoutMs *int                  `json:"stream_first_chunk_timeout_ms,omitempty"`
	FailureThreshold          *int                  `json:"failure_threshold,omitempty"`
	RecoveryTimeoutSeconds    *int                  `json:"recovery_timeout_seconds,omitempty"`
	Status                    *model.ProviderStatus `json:"status,omitempty"`
}

// modelProviderService 妯″瀷婧愬ご鏈嶅姟瀹炵幇
type modelProviderService struct {
	providerRepo   repository.ModelProviderRepository
	metricsRepo    repository.ProviderMetricsRepository
	channelRepo    repository.ChannelRepository
	modelRepo      repository.ModelRepository
	selector       scheduler.ProviderSelector
	circuitBreaker scheduler.CircuitBreaker
	healthTracker  scheduler.HealthTracker
}

// NewModelProviderService 鍒涘缓妯″瀷婧愬ご鏈嶅姟
func NewModelProviderService(
	providerRepo repository.ModelProviderRepository,
	metricsRepo repository.ProviderMetricsRepository,
	channelRepo repository.ChannelRepository,
	modelRepo repository.ModelRepository,
	selector scheduler.ProviderSelector,
	circuitBreaker scheduler.CircuitBreaker,
	healthTracker scheduler.HealthTracker,
) ModelProviderService {
	return &modelProviderService{
		providerRepo:   providerRepo,
		metricsRepo:    metricsRepo,
		channelRepo:    channelRepo,
		modelRepo:      modelRepo,
		selector:       selector,
		circuitBreaker: circuitBreaker,
		healthTracker:  healthTracker,
	}
}

// Create 鍒涘缓妯″瀷婧愬ご
func (s *modelProviderService) Create(ctx context.Context, req *CreateProviderRequest) (*model.ModelProvider, error) {
	operation := model.NormalizeOperation(req.Operation)
	// 妫€鏌ユā鍨嬫槸鍚﹀瓨鍦?
	if operation == model.OperationChatCompletions {
		modelInfo, err := s.modelRepo.GetByID(req.ModelID)
		if err != nil {
			return nil, fmt.Errorf("鑾峰彇妯″瀷澶辫触: %w", err)
		}
		if modelInfo == nil {
			return nil, fmt.Errorf("妯″瀷涓嶅瓨鍦? %s", req.ModelID)
		}

		// 妫€鏌ユ笭閬撴槸鍚﹀瓨鍦?
	}

	channel, err := s.channelRepo.GetByID(req.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇娓犻亾澶辫触: %w", err)
	}
	if channel == nil {
		return nil, fmt.Errorf("娓犻亾涓嶅瓨鍦? %d", req.ChannelID)
	}

	// 妫€鏌ユ槸鍚﹀凡瀛樺湪鐩稿悓鐨勬簮澶?
	existing, err := s.providerRepo.GetByModelAndChannel(ctx, operation, req.ModelID, req.ChannelID)
	if err != nil {
		return nil, fmt.Errorf("妫€鏌ユ簮澶存槸鍚﹀瓨鍦ㄥけ璐? %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("璇ユā鍨嬪拰娓犻亾鐨勬簮澶村凡瀛樺湪")
	}

	// 璁剧疆榛樿鍊?
	status := req.Status
	if status == "" {
		status = model.ProviderStatusActive
	}

	upstreamModelName := req.UpstreamModelName
	if upstreamModelName == "" {
		upstreamModelName = req.ModelID
	}

	isCostPriority := true
	if req.IsCostPriority != nil {
		isCostPriority = *req.IsCostPriority
	}

	priority := req.Priority
	if priority <= 0 {
		priority = 100
	}

	weight := req.Weight
	if weight <= 0 {
		weight = 100
	}

	connectTimeoutMs := req.ConnectTimeoutMs
	if connectTimeoutMs <= 0 {
		connectTimeoutMs = 2000
	}
	attemptTimeoutMs := req.AttemptTimeoutMs
	if attemptTimeoutMs <= 0 {
		attemptTimeoutMs = 15000
	}
	streamFirstChunkTimeoutMs := req.StreamFirstChunkTimeoutMs
	if streamFirstChunkTimeoutMs <= 0 {
		streamFirstChunkTimeoutMs = 3000
	}

	failureThreshold := req.FailureThreshold
	if failureThreshold <= 0 {
		failureThreshold = 5
	}

	recoveryTimeout := req.RecoveryTimeoutSeconds
	if recoveryTimeout <= 0 {
		recoveryTimeout = 30
	}

	provider := &model.ModelProvider{
		Operation:                 operation,
		ModelID:                   req.ModelID,
		ChannelID:                 req.ChannelID,
		UpstreamModelName:         upstreamModelName,
		ActualCostPer1kInput:      req.ActualCostPer1kInput,
		ActualCostPer1kOutput:     req.ActualCostPer1kOutput,
		Priority:                  priority,
		Weight:                    weight,
		IsCostPriority:            isCostPriority,
		Status:                    model.ModelProviderStatus(status),
		CircuitState:              model.CircuitStateClosed,
		FailureThreshold:          failureThreshold,
		RecoveryTimeoutSeconds:    recoveryTimeout,
		ConnectTimeoutMs:          connectTimeoutMs,
		AttemptTimeoutMs:          attemptTimeoutMs,
		StreamFirstChunkTimeoutMs: streamFirstChunkTimeoutMs,
		HealthScore:               decimal.NewFromInt(100),
	}

	if err := s.providerRepo.Create(ctx, provider); err != nil {
		return nil, fmt.Errorf("鍒涘缓婧愬ご澶辫触: %w", err)
	}

	// 鍒锋柊缂撳瓨
	_ = s.selector.RefreshCache(ctx, operation, req.ModelID)
	InvalidateGatewayResponseCache(operation, req.ModelID)

	return provider, nil
}

func (s *modelProviderService) GetEntityByID(ctx context.Context, id uint) (*model.ModelProvider, error) {
	return s.providerRepo.GetByID(ctx, id)
}

// Update 鏇存柊妯″瀷婧愬ご
func (s *modelProviderService) Update(ctx context.Context, id uint, req *UpdateProviderRequest) (*model.ModelProvider, error) {
	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇婧愬ご澶辫触: %w", err)
	}
	if provider == nil {
		return nil, fmt.Errorf("婧愬ご涓嶅瓨鍦? %d", id)
	}

	// 鏇存柊瀛楁
	if req.ActualCostPer1kInput != nil {
		provider.ActualCostPer1kInput = *req.ActualCostPer1kInput
	}
	if req.ActualCostPer1kOutput != nil {
		provider.ActualCostPer1kOutput = *req.ActualCostPer1kOutput
	}
	if req.UpstreamModelName != nil {
		provider.UpstreamModelName = *req.UpstreamModelName
	}
	if req.IsCostPriority != nil {
		provider.IsCostPriority = *req.IsCostPriority
	}
	if req.Priority != nil {
		provider.Priority = *req.Priority
	}
	if req.Weight != nil {
		provider.Weight = *req.Weight
	}
	if req.ConnectTimeoutMs != nil {
		provider.ConnectTimeoutMs = *req.ConnectTimeoutMs
	}
	if req.AttemptTimeoutMs != nil {
		provider.AttemptTimeoutMs = *req.AttemptTimeoutMs
	}
	if req.StreamFirstChunkTimeoutMs != nil {
		provider.StreamFirstChunkTimeoutMs = *req.StreamFirstChunkTimeoutMs
	}
	if req.FailureThreshold != nil {
		provider.FailureThreshold = *req.FailureThreshold
	}
	if req.RecoveryTimeoutSeconds != nil {
		provider.RecoveryTimeoutSeconds = *req.RecoveryTimeoutSeconds
	}
	if req.Status != nil {
		provider.Status = model.ModelProviderStatus(*req.Status)
	}

	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return nil, fmt.Errorf("鏇存柊婧愬ご澶辫触: %w", err)
	}

	// 鍒锋柊缂撳瓨
	_ = s.selector.RefreshCache(ctx, model.NormalizeOperation(provider.Operation), provider.ModelID)
	InvalidateGatewayResponseCache(provider.Operation, provider.ModelID)

	return provider, nil
}

// Delete 鍒犻櫎妯″瀷婧愬ご
func (s *modelProviderService) Delete(ctx context.Context, id uint) error {
	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("鑾峰彇婧愬ご澶辫触: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("婧愬ご涓嶅瓨鍦? %d", id)
	}

	modelID := provider.ModelID
	operation := model.NormalizeOperation(provider.Operation)

	if err := s.providerRepo.Delete(ctx, id); err != nil {
		return fmt.Errorf("鍒犻櫎婧愬ご澶辫触: %w", err)
	}

	// 鍒锋柊缂撳瓨
	_ = s.selector.RefreshCache(ctx, operation, modelID)
	InvalidateGatewayResponseCache(operation, modelID)

	return nil
}

// GetByID 鏍规嵁ID鑾峰彇婧愬ご
func (s *modelProviderService) GetByID(ctx context.Context, id uint) (*model.ModelProviderResponse, error) {
	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇婧愬ご澶辫触: %w", err)
	}
	if provider == nil {
		return nil, nil
	}
	return provider.ToResponse(), nil
}

// List 鍒嗛〉鏌ヨ婧愬ご鍒楄〃
func (s *modelProviderService) List(ctx context.Context, params *model.ProviderQueryParams) ([]*model.ModelProviderResponse, int64, error) {
	providers, total, err := s.providerRepo.List(ctx, params)
	if err != nil {
		return nil, 0, fmt.Errorf("鏌ヨ婧愬ご鍒楄〃澶辫触: %w", err)
	}

	responses := make([]*model.ModelProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	return responses, total, nil
}

// GetByModelID 鏍规嵁妯″瀷ID鑾峰彇婧愬ご鍒楄〃
func (s *modelProviderService) GetByModelID(ctx context.Context, operation string, modelID string) ([]*model.ModelProviderResponse, error) {
	operation = model.NormalizeOperation(operation)
	providers, err := s.providerRepo.GetByModelID(ctx, operation, modelID)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇婧愬ご鍒楄〃澶辫触: %w", err)
	}

	responses := make([]*model.ModelProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	return responses, nil
}

// GetByChannelID 鏍规嵁娓犻亾ID鑾峰彇婧愬ご鍒楄〃
func (s *modelProviderService) GetByChannelID(ctx context.Context, channelID uint) ([]*model.ModelProviderResponse, error) {
	providers, err := s.providerRepo.GetByChannelID(ctx, channelID)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇婧愬ご鍒楄〃澶辫触: %w", err)
	}

	responses := make([]*model.ModelProviderResponse, len(providers))
	for i, p := range providers {
		responses[i] = p.ToResponse()
	}

	return responses, nil
}

// Enable 鍚敤婧愬ご
func (s *modelProviderService) Enable(ctx context.Context, id uint) error {
	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("鑾峰彇婧愬ご澶辫触: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("婧愬ご涓嶅瓨鍦? %d", id)
	}

	provider.Status = model.ModelProviderStatusActive
	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return fmt.Errorf("鍚敤婧愬ご澶辫触: %w", err)
	}

	_ = s.selector.RefreshCache(ctx, model.NormalizeOperation(provider.Operation), provider.ModelID)
	InvalidateGatewayResponseCache(provider.Operation, provider.ModelID)
	return nil
}

// Disable 绂佺敤婧愬ご
func (s *modelProviderService) Disable(ctx context.Context, id uint) error {
	provider, err := s.providerRepo.GetByID(ctx, id)
	if err != nil {
		return fmt.Errorf("鑾峰彇婧愬ご澶辫触: %w", err)
	}
	if provider == nil {
		return fmt.Errorf("婧愬ご涓嶅瓨鍦? %d", id)
	}

	provider.Status = model.ModelProviderStatusDisabled
	if err := s.providerRepo.Update(ctx, provider); err != nil {
		return fmt.Errorf("绂佺敤婧愬ご澶辫触: %w", err)
	}

	_ = s.selector.RefreshCache(ctx, model.NormalizeOperation(provider.Operation), provider.ModelID)
	InvalidateGatewayResponseCache(provider.Operation, provider.ModelID)
	return nil
}

// BatchUpdateStatus 鎵归噺鏇存柊鐘舵€?
func (s *modelProviderService) BatchUpdateStatus(ctx context.Context, ids []uint, status model.ProviderStatus) error {
	affectedRouteKeys := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		if id == 0 {
			continue
		}
		provider, err := s.providerRepo.GetByID(ctx, id)
		if err != nil {
			return fmt.Errorf("加载源头失败: %w", err)
		}
		if provider == nil {
			continue
		}
		operation := model.NormalizeOperation(provider.Operation)
		affectedRouteKeys[operation+":"+provider.ModelID] = struct{}{}
	}

	if err := s.providerRepo.BatchUpdateStatus(ctx, ids, status); err != nil {
		return fmt.Errorf("鎵归噺鏇存柊鐘舵€佸け璐? %w", err)
	}
	for key := range affectedRouteKeys {
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		_ = s.selector.RefreshCache(ctx, parts[0], parts[1])
		InvalidateGatewayResponseCache(parts[0], parts[1])
	}
	return nil
}

// OpenCircuit 鎵撳紑鐔旀柇鍣?
func (s *modelProviderService) OpenCircuit(ctx context.Context, id uint, duration time.Duration, reason string) error {
	return s.circuitBreaker.ForceOpen(ctx, id, duration, reason)
}

// CloseCircuit 鍏抽棴鐔旀柇鍣?
func (s *modelProviderService) CloseCircuit(ctx context.Context, id uint) error {
	return s.circuitBreaker.ForceClose(ctx, id)
}

// GetCircuitInfo 鑾峰彇鐔旀柇鍣ㄤ俊鎭?
func (s *modelProviderService) GetCircuitInfo(ctx context.Context, id uint) (*scheduler.CircuitInfo, error) {
	return s.circuitBreaker.GetCircuitInfo(ctx, id)
}

// GetProviderHealth 鑾峰彇婧愬ご鍋ュ悍璇︽儏
func (s *modelProviderService) GetProviderHealth(ctx context.Context, id uint) (*scheduler.ProviderHealth, error) {
	return s.healthTracker.GetProviderHealth(ctx, id)
}

// GetModelHealth 鑾峰彇妯″瀷鍋ュ悍姒傝
func (s *modelProviderService) GetModelHealth(ctx context.Context, operation string, modelID string) (*scheduler.ModelHealth, error) {
	return s.healthTracker.GetModelHealth(ctx, operation, modelID)
}

// GetHealthSummary 鑾峰彇鍋ュ悍鎽樿
func (s *modelProviderService) GetHealthSummary(ctx context.Context) (*scheduler.HealthSummary, error) {
	return s.healthTracker.GetHealthSummary(ctx)
}

// GetAvailableProviders 鑾峰彇鍙敤婧愬ご
func (s *modelProviderService) GetAvailableProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	return s.providerRepo.GetAvailableProviders(ctx, operation, modelID)
}

// SelectBestProvider 閫夋嫨鏈€浣虫簮澶?
func (s *modelProviderService) SelectBestProvider(ctx context.Context, operation string, modelID string, strategy scheduler.SelectionStrategy) (*model.ModelProvider, error) {
	return s.selector.SelectBestProvider(ctx, operation, modelID, strategy)
}

// BatchCreate 鎵归噺鍒涘缓婧愬ご
func (s *modelProviderService) BatchCreate(ctx context.Context, reqs []*CreateProviderRequest) ([]*model.ModelProvider, error) {
	providers := make([]*model.ModelProvider, 0, len(reqs))

	for _, req := range reqs {
		provider, err := s.Create(ctx, req)
		if err != nil {
			// 璁板綍閿欒浣嗙户缁鐞?
			continue
		}
		providers = append(providers, provider)
	}

	return providers, nil
}

// SyncFromChannelModels 浠庢笭閬撴ā鍨嬪悓姝ユ簮澶?
// 杩欎釜鏂规硶鐢ㄤ簬浠庣幇鏈夌殑 channel_models 琛ㄨ縼绉绘暟鎹埌 model_providers 琛?
func (s *modelProviderService) SyncFromChannelModels(ctx context.Context, channelID uint) error {
	// 鑾峰彇娓犻亾淇℃伅
	channel, err := s.channelRepo.GetByID(channelID)
	if err != nil {
		return fmt.Errorf("鑾峰彇娓犻亾澶辫触: %w", err)
	}
	if channel == nil {
		return fmt.Errorf("娓犻亾涓嶅瓨鍦? %d", channelID)
	}

	// 鑾峰彇娓犻亾鏀寔鐨勬ā鍨?
	channelModels, err := s.channelRepo.GetChannelModels(channelID)
	if err != nil {
		return fmt.Errorf("鑾峰彇娓犻亾妯″瀷澶辫触: %w", err)
	}

	// 涓烘瘡涓ā鍨嬪垱寤烘簮澶?
	for _, cm := range channelModels {
		// 妫€鏌ユ槸鍚﹀凡瀛樺湪
		existing, err := s.providerRepo.GetByModelAndChannel(ctx, model.OperationChatCompletions, cm.ModelID, channelID)
		if err != nil {
			continue
		}
		if existing != nil {
			continue // 宸插瓨鍦紝璺宠繃
		}

		// 鑾峰彇妯″瀷瀹氫环淇℃伅
		modelPricing, err := s.modelRepo.GetPricing(cm.ModelID)
		if err != nil || modelPricing == nil {
			continue
		}

		// 璁＄畻瀹為檯鎴愭湰锛堜娇鐢?CostRatio锛?
		actualInputCost := modelPricing.InputPrice.Mul(cm.CostRatio)
		actualOutputCost := modelPricing.OutputPrice.Mul(cm.CostRatio)

		provider := &model.ModelProvider{
			Operation:              model.OperationChatCompletions,
			ModelID:                cm.ModelID,
			ChannelID:              channelID,
			ActualCostPer1kInput:   actualInputCost,
			ActualCostPer1kOutput:  actualOutputCost,
			Priority:               100,
			Weight:                 100,
			Status:                 model.ModelProviderStatusActive,
			CircuitState:           model.CircuitStateClosed,
			FailureThreshold:       5,
			RecoveryTimeoutSeconds: 30,
			HealthScore:            decimal.NewFromInt(100),
		}

		_ = s.providerRepo.Create(ctx, provider)
	}

	return nil
}
