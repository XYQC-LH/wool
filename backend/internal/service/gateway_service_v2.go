package service

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	"nexus-api/internal/service/scheduler"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/sync/singleflight"
	"golang.org/x/net/websocket"
)

// gatewayServiceV2 增强版 Gateway 服务实现
type gatewayServiceV2 struct {
	channelService    ChannelService
	tokenService      TokenService
	userRepo          repository.UserRepository
	logRepo           repository.LogRepository
	modelRepo         repository.ModelRepository
	resourceRepo      repository.ResourceAccountRepository
	channelRepo       repository.ChannelRepository
	httpClient        *http.Client
	providerRepo      repository.ModelProviderRepository
	instanceRepo      repository.ProviderInstanceRepository
	providerSelector  scheduler.ProviderSelector
	circuitBreaker    scheduler.CircuitBreaker
	cascadeController scheduler.CascadeController
	cascadeMetrics    *scheduler.CascadeMetrics
	instanceScheduler scheduler.InstanceScheduler
	errorClassifier   scheduler.ErrorClassifier
	streamGuard       scheduler.StreamGuard
	healthTracker     scheduler.HealthTracker
	costCalculator    scheduler.CostCalculator
	tenantBillingHook TenantBillingHook
	requestDeduper    singleflight.Group
}

// NewGatewayServiceV2 创建增强版 Gateway 服务
func NewGatewayServiceV2(
	channelService ChannelService,
	tokenService TokenService,
	userRepo repository.UserRepository,
	logRepo repository.LogRepository,
	modelRepo repository.ModelRepository,
	resourceRepo repository.ResourceAccountRepository,
	channelRepo repository.ChannelRepository,
	providerRepo repository.ModelProviderRepository,
	instanceRepo repository.ProviderInstanceRepository,
	providerSelector scheduler.ProviderSelector,
	instanceScheduler scheduler.InstanceScheduler,
	cascadeController scheduler.CascadeController,
	circuitBreaker scheduler.CircuitBreaker,
	streamGuard scheduler.StreamGuard,
	healthTracker scheduler.HealthTracker,
	costCalculator scheduler.CostCalculator,
	errorClassifier scheduler.ErrorClassifier,
	tenantBillingHook TenantBillingHook,
) GatewayService {
	cascadeMetrics := scheduler.NewCascadeMetrics()

	return &gatewayServiceV2{
		channelService:    channelService,
		tokenService:      tokenService,
		userRepo:          userRepo,
		logRepo:           logRepo,
		modelRepo:         modelRepo,
		resourceRepo:      resourceRepo,
		channelRepo:       channelRepo,
		httpClient:        &http.Client{Timeout: 5 * time.Minute},
		providerRepo:      providerRepo,
		instanceRepo:      instanceRepo,
		providerSelector:  providerSelector,
		circuitBreaker:    circuitBreaker,
		cascadeController: cascadeController,
		cascadeMetrics:    cascadeMetrics,
		instanceScheduler: instanceScheduler,
		errorClassifier:   errorClassifier,
		streamGuard:       streamGuard,
		healthTracker:     healthTracker,
		costCalculator:    costCalculator,
		tenantBillingHook: tenantBillingHook,
	}
}

// HandleChatCompletion 处理聊天完成请求
func (s *gatewayServiceV2) HandleChatCompletion(req *ChatCompletionRequest, token *model.Token) (*ChatCompletionResponse, error) {
	startTime := time.Now()
	ctx := context.Background()

	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		return nil, err
	}

	// 转换为scheduler.ChatCompletionRequest
	maxTokens := 0
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	temperature := 0.7
	if req.Temperature != nil {
		temperature = *req.Temperature
	}
	topP := 1.0
	if req.TopP != nil {
		topP = *req.TopP
	}
	var stop []string
	if req.Stop != nil {
		if s, ok := req.Stop.(string); ok {
			stop = []string{s}
		} else if s, ok := req.Stop.([]string); ok {
			stop = s
		}
	}
	presencePenalty := 0.0
	if req.PresencePenalty != nil {
		presencePenalty = *req.PresencePenalty
	}
	frequencyPenalty := 0.0
	if req.FrequencyPenalty != nil {
		frequencyPenalty = *req.FrequencyPenalty
	}

	schedulerReq := &scheduler.ChatCompletionRequest{
		Model:            req.Model,
		Messages:         convertMessages(req.Messages),
		MaxTokens:        maxTokens,
		Temperature:      temperature,
		TopP:             topP,
		Stream:           req.Stream,
		Stop:             stop,
		PresencePenalty:  presencePenalty,
		FrequencyPenalty: frequencyPenalty,
	}

	estimatedCost, err := s.costCalculator.EstimateCost(schedulerReq)
	if err != nil {
		return nil, err
	}

	if err := s.checkTenantQuota(ctx, token, estimatedCost); err != nil {
		return nil, err
	}

	if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
		return nil, &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
	}

	if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
		return nil, &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
	}

	operation := model.OperationChatCompletions
	ctx = s.applyGatewayTrafficHints(ctx, operation, req.Model, scheduler.TrafficHints{
		SessionID:      req.SessionID,
		TrafficTag:     req.TrafficTag,
		ExperimentID:   req.ExperimentID,
		IdempotencyKey: req.IdempotencyKey,
		ForceCanary:    req.ForceCanary,
	})

	control := gatewayRequestControl{
		Operation:      operation,
		ModelID:        req.Model,
		IdempotencyKey: req.IdempotencyKey,
		EnableCache:    req.EnableCache,
		BypassCache:    req.BypassCache,
		DisableDedup:   req.DisableDedup,
	}

	return s.executeWithTrafficOptimization(ctx, token, req, control, func(execCtx context.Context) (*ChatCompletionResponse, error) {
		return s.handleChatCompletionWithScheduler(execCtx, req, token, modelPricing, startTime)
	})
}

func (s *gatewayServiceV2) handleChatCompletionWithScheduler(ctx context.Context, req *ChatCompletionRequest, token *model.Token, modelPricing *model.ModelPricing, startTime time.Time) (*ChatCompletionResponse, error) {
	_ = modelPricing

	executor := func(execCtx context.Context, provider *model.ModelProvider) (interface{}, error) {
		return s.executeProviderRequest(execCtx, provider, req)
	}

	promptTokens, completionTokens := s.estimateChatTokens(req)
	rateCtx := scheduler.WithRateLimitUsage(ctx, buildTokenRateLimitUsage(promptTokens, completionTokens))

	result, err := s.cascadeController.ExecuteWithStrategy(rateCtx, model.OperationChatCompletions, req.Model, scheduler.StrategyAdaptive, executor)
	if result != nil {
		s.cascadeMetrics.RecordResult(result)
	}

	if err != nil {
		s.logProviderRequest(token, nil, model.OperationChatCompletions, req.Model, &Usage{}, decimal.Zero, decimal.Zero, int(time.Since(startTime).Milliseconds()), model.LogStatusError, err.Error())
		return nil, err
	}

	chatResp, ok := result.Response.(*ChatCompletionResponse)
	if !ok {
		return nil, fmt.Errorf("响应类型错误")
	}

	if chatResp.Usage != nil {
		cost, _ := s.costCalculator.CalculateCost(req.Model, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens)
		upstreamCost := s.costCalculator.CalculateProviderCost(result.Provider, chatResp.Usage.PromptTokens, chatResp.Usage.CompletionTokens)

		if token.User != nil {
			if err := DeductUserBalance(s.userRepo, token.UserID, cost, &token.User.Balance); err != nil {
				s.logProviderRequest(token, result.Provider, model.OperationChatCompletions, req.Model, chatResp.Usage, decimal.Zero, decimal.Zero, int(result.TotalLatencyMs), model.LogStatusError, err.Error())
				return nil, err
			}
		}
		if token.RemainQuota != nil {
			if err := s.tokenService.DeductQuota(token.ID, cost); err != nil {
				s.logProviderRequest(token, result.Provider, model.OperationChatCompletions, req.Model, chatResp.Usage, decimal.Zero, decimal.Zero, int(result.TotalLatencyMs), model.LogStatusError, err.Error())
				return nil, err
			}
		}
		_ = s.providerRepo.IncrementStats(ctx, result.Provider.ID, true, result.TotalLatencyMs, int64(chatResp.Usage.PromptTokens), int64(chatResp.Usage.CompletionTokens), upstreamCost)
		s.logProviderRequest(token, result.Provider, model.OperationChatCompletions, req.Model, chatResp.Usage, cost, upstreamCost, int(result.TotalLatencyMs), model.LogStatusSuccess, "")
		s.emitTenantBilling(ctx, token, model.OperationChatCompletions, req.Model, chatResp.Usage, cost)
	}

	return chatResp, nil
}

func (s *gatewayServiceV2) executeProviderRequest(ctx context.Context, provider *model.ModelProvider, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if provider.Channel == nil {
		return nil, fmt.Errorf("源头缺少渠道信息")
	}

	if scheduler.SourceTransportFromContext(ctx) == scheduler.SourceTransportWebSocket {
		return s.executeProviderRequestByWebSocket(ctx, provider, req)
	}

	upstreamReq, err := s.buildProviderRequest(ctx, provider, req)
	if err != nil {
		return nil, fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := s.httpClient.Do(upstreamReq)
	if err != nil {
		return nil, fmt.Errorf("上游请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(body))
		return nil, NewUpstreamHTTPError(resp.StatusCode, bodyStr)
	}

	var chatResp ChatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	return &chatResp, nil
}

func (s *gatewayServiceV2) buildProviderRequest(ctx context.Context, provider *model.ModelProvider, req *ChatCompletionRequest) (*http.Request, error) {
	if provider.Channel == nil {
		return nil, fmt.Errorf("源头缺少渠道信息")
	}

	upstreamModel := provider.UpstreamModelName
	if upstreamModel == "" {
		upstreamModel = req.Model
	}

	reqCopy := *req
	reqCopy.Model = upstreamModel

	reqBody, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, err
	}

	transport := scheduler.SourceTransportFromContext(ctx)
	endpoint := "/chat/completions"
	contentType := "application/json"
	switch transport {
	case scheduler.SourceTransportGRPC:
		if configured := strings.TrimSpace(readProviderChannelConfigString(provider, "grpc_gateway_path", "grpc_path")); configured != "" {
			endpoint = configured
		}
		contentType = "application/grpc+json"
	default:
		if configured := strings.TrimSpace(readProviderChannelConfigString(provider, "http_path")); configured != "" {
			endpoint = configured
		}
	}

	upstreamURL, err := buildOpenAIURL(provider.Channel.BaseURL, endpoint)
	if err != nil {
		return nil, err
	}

	upstreamReq, err := http.NewRequestWithContext(ctx, http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	upstreamReq.Header.Set("Content-Type", contentType)
	upstreamReq.Header.Set("Authorization", "Bearer "+provider.Channel.APIKey)
	upstreamReq.Header.Set("X-Source-Transport", string(transport))

	return upstreamReq, nil
}

func (s *gatewayServiceV2) executeProviderRequestByWebSocket(ctx context.Context, provider *model.ModelProvider, req *ChatCompletionRequest) (*ChatCompletionResponse, error) {
	if provider == nil || provider.Channel == nil {
		return nil, fmt.Errorf("源头缺少渠道信息")
	}

	upstreamModel := provider.UpstreamModelName
	if upstreamModel == "" {
		upstreamModel = req.Model
	}

	reqCopy := *req
	reqCopy.Stream = false
	reqCopy.Model = upstreamModel

	wsURL, err := s.resolveProviderWebSocketURL(provider, "/chat/completions")
	if err != nil {
		return nil, err
	}

	conn, err := dialProviderWebSocket(ctx, wsURL, provider.Channel.APIKey)
	if err != nil {
		return nil, fmt.Errorf("建立 WebSocket 上游连接失败: %w", err)
	}
	defer conn.Close()

	if err := websocket.JSON.Send(conn, reqCopy); err != nil {
		return nil, fmt.Errorf("发送 WebSocket 请求失败: %w", err)
	}

	var response ChatCompletionResponse
	if err := websocket.JSON.Receive(conn, &response); err != nil {
		return nil, fmt.Errorf("读取 WebSocket 响应失败: %w", err)
	}
	return &response, nil
}

// calculateProviderCost 已移除，使用costCalculator.CalculateProviderCost替代

// handleChatCompletionLegacy 已移除，完全使用新的调度器

// HandleChatCompletionStream 处理流式聊天完成请求
func (s *gatewayServiceV2) HandleChatCompletionStream(req *ChatCompletionRequest, token *model.Token, writer http.ResponseWriter) error {
	startTime := time.Now()
	ctx := context.Background()

	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		return err
	}

	req.Stream = true
	ctx = s.applyGatewayTrafficHints(ctx, model.OperationChatCompletions, req.Model, scheduler.TrafficHints{
		SessionID:      req.SessionID,
		TrafficTag:     req.TrafficTag,
		ExperimentID:   req.ExperimentID,
		IdempotencyKey: req.IdempotencyKey,
		ForceCanary:    req.ForceCanary,
	})

	return s.handleStreamWithScheduler(ctx, req, token, modelPricing, writer, startTime)
}

func (s *gatewayServiceV2) handleStreamWithScheduler(ctx context.Context, req *ChatCompletionRequest, token *model.Token, modelPricing *model.ModelPricing, writer http.ResponseWriter, startTime time.Time) error {
	_ = modelPricing

	maxTokens := 0
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	schedulerReq := &scheduler.ChatCompletionRequest{
		Model:     req.Model,
		Messages:  convertMessages(req.Messages),
		MaxTokens: maxTokens,
		Stream:    true,
	}

	estimatedPromptTokens := s.costCalculator.EstimatePromptTokens(schedulerReq)
	estimatedCompletionTokens := s.costCalculator.EstimateCompletionTokens(schedulerReq)
	estimatedCost, _ := s.costCalculator.CalculateCost(req.Model, estimatedPromptTokens, estimatedCompletionTokens)

	if err := s.checkTenantQuota(ctx, token, estimatedCost); err != nil {
		return err
	}

	if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
		return &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
	}
	if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
		return &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
	}

	executor := func(execCtx context.Context, provider *model.ModelProvider, onFirstChunk func()) error {
		return s.executeStreamRequest(execCtx, provider, req, writer, onFirstChunk)
	}

	rateCtx := scheduler.WithRateLimitUsage(ctx, buildTokenRateLimitUsage(estimatedPromptTokens, estimatedCompletionTokens))
	result, err := s.cascadeController.ExecuteStreamWithFailover(rateCtx, model.OperationChatCompletions, req.Model, scheduler.StrategyAdaptive, executor)
	if result != nil {
		s.cascadeMetrics.RecordResult(result)
	}
	if err != nil {
		return err
	}
	if result == nil || result.Provider == nil {
		return fmt.Errorf("调度结果缺少源头信息")
	}

	provider := result.Provider
	latency := int(time.Since(startTime).Milliseconds())

	usage := &Usage{
		PromptTokens:     estimatedPromptTokens,
		CompletionTokens: estimatedCompletionTokens,
		TotalTokens:      estimatedPromptTokens + estimatedCompletionTokens,
	}

	cost, _ := s.costCalculator.CalculateCost(req.Model, estimatedPromptTokens, estimatedCompletionTokens)
	upstreamCost := s.costCalculator.CalculateProviderCost(provider, estimatedPromptTokens, estimatedCompletionTokens)

	if token.User != nil {
		_ = s.userRepo.DeductBalance(token.UserID, cost)
	}
	if token.RemainQuota != nil {
		_ = s.tokenService.DeductQuota(token.ID, cost)
	}

	_ = s.healthTracker.RecordRequest(ctx, provider.ID, true, int64(latency), int64(estimatedPromptTokens), int64(estimatedCompletionTokens), upstreamCost)

	if provider.SelectedInstance != nil {
		_ = s.instanceRepo.IncrementStats(ctx, provider.SelectedInstance.ID, true, int64(latency))
	}

	s.logProviderRequest(token, provider, model.OperationChatCompletions, req.Model, usage, cost, upstreamCost, latency, model.LogStatusSuccess, "")
	s.emitTenantBilling(ctx, token, model.OperationChatCompletions, req.Model, usage, cost)
	return nil
}

// handleStreamLegacy 已移除，完全使用新的调度器

// HandleCompletion 处理完成请求
func (s *gatewayServiceV2) HandleCompletion(req *CompletionRequest, token *model.Token) (*CompletionResponse, error) {
	startTime := time.Now()
	ctx := context.Background()
	estimatedPromptTokens, estimatedCompletionTokens := estimateCompletionTokens(req)
	estimatedCost, _ := s.costCalculator.CalculateCost(req.Model, estimatedPromptTokens, estimatedCompletionTokens)

	if err := s.checkTenantQuota(ctx, token, estimatedCost); err != nil {
		return nil, err
	}

	operation := model.OperationCompletions
	ctx = s.applyGatewayTrafficHints(ctx, operation, req.Model, scheduler.TrafficHints{
		SessionID:      req.SessionID,
		TrafficTag:     req.TrafficTag,
		ExperimentID:   req.ExperimentID,
		IdempotencyKey: req.IdempotencyKey,
		ForceCanary:    req.ForceCanary,
	})

	control := gatewayRequestControl{
		Operation:      operation,
		ModelID:        req.Model,
		IdempotencyKey: req.IdempotencyKey,
		EnableCache:    req.EnableCache,
		BypassCache:    req.BypassCache,
		DisableDedup:   req.DisableDedup,
	}

	return s.executeWithTrafficOptimization(ctx, token, req, control, func(execCtx context.Context) (*CompletionResponse, error) {
		executor := func(callCtx context.Context, provider *model.ModelProvider) (interface{}, error) {
			return s.executeCompletionProviderRequest(callCtx, provider, req)
		}

		usedOperation := operation
		rateCtx := scheduler.WithRateLimitUsage(execCtx, buildTokenRateLimitUsage(estimatedPromptTokens, estimatedCompletionTokens))
		result, err := s.cascadeController.ExecuteWithStrategy(rateCtx, operation, req.Model, scheduler.StrategyAdaptive, executor)
		if err != nil {
			var noProviders *scheduler.NoAvailableProviderError
			if errors.As(err, &noProviders) {
				usedOperation = model.OperationChatCompletions
				result, err = s.cascadeController.ExecuteWithStrategy(rateCtx, usedOperation, req.Model, scheduler.StrategyAdaptive, executor)
			}
		}
		if err != nil {
			s.logProviderRequest(token, nil, usedOperation, req.Model, &Usage{}, decimal.Zero, decimal.Zero, int(time.Since(startTime).Milliseconds()), model.LogStatusError, err.Error())
			return nil, err
		}
		if result == nil || result.Response == nil {
			return nil, fmt.Errorf("调度结果为空")
		}

		completionResp, ok := result.Response.(*CompletionResponse)
		if !ok || completionResp == nil {
			return nil, fmt.Errorf("完成响应解析失败")
		}

		latencyMs := int(time.Since(startTime).Milliseconds())

		usage := completionResp.Usage
		if usage == nil {
			usage = &Usage{}
		}

		cost, _ := s.costCalculator.CalculateCost(req.Model, usage.PromptTokens, usage.CompletionTokens)
		upstreamCost := s.costCalculator.CalculateProviderCost(result.Provider, usage.PromptTokens, usage.CompletionTokens)

		if token.User != nil {
			if err := DeductUserBalance(s.userRepo, token.UserID, cost, &token.User.Balance); err != nil {
				s.logProviderRequest(token, result.Provider, usedOperation, req.Model, usage, decimal.Zero, decimal.Zero, latencyMs, model.LogStatusError, err.Error())
				return nil, err
			}
		}
		if token.RemainQuota != nil {
			if err := s.tokenService.DeductQuota(token.ID, cost); err != nil {
				s.logProviderRequest(token, result.Provider, usedOperation, req.Model, usage, decimal.Zero, decimal.Zero, latencyMs, model.LogStatusError, err.Error())
				return nil, err
			}
		}
		if result.Provider != nil {
			_ = s.providerRepo.IncrementStats(execCtx, result.Provider.ID, true, int64(latencyMs), int64(usage.PromptTokens), int64(usage.CompletionTokens), upstreamCost)
		}
		s.logProviderRequest(token, result.Provider, usedOperation, req.Model, usage, cost, upstreamCost, latencyMs, model.LogStatusSuccess, "")
		s.emitTenantBilling(execCtx, token, usedOperation, req.Model, usage, cost)

		return completionResp, nil
	})
}

// HandleEmbedding 处理嵌入请求
func (s *gatewayServiceV2) HandleEmbedding(req *EmbeddingRequest, token *model.Token) (*EmbeddingResponse, error) {
	startTime := time.Now()
	ctx := context.Background()
	estimatedPromptTokens := estimateEmbeddingTokens(req)
	estimatedCost, _ := s.costCalculator.CalculateCost(req.Model, estimatedPromptTokens, 0)

	if err := s.checkTenantQuota(ctx, token, estimatedCost); err != nil {
		return nil, err
	}

	operation := model.OperationEmbeddings
	ctx = s.applyGatewayTrafficHints(ctx, operation, req.Model, scheduler.TrafficHints{
		SessionID:      req.SessionID,
		TrafficTag:     req.TrafficTag,
		ExperimentID:   req.ExperimentID,
		IdempotencyKey: req.IdempotencyKey,
		ForceCanary:    req.ForceCanary,
	})

	control := gatewayRequestControl{
		Operation:      operation,
		ModelID:        req.Model,
		IdempotencyKey: req.IdempotencyKey,
		EnableCache:    req.EnableCache,
		BypassCache:    req.BypassCache,
		DisableDedup:   req.DisableDedup,
	}

	return s.executeWithTrafficOptimization(ctx, token, req, control, func(execCtx context.Context) (*EmbeddingResponse, error) {
		executor := func(callCtx context.Context, provider *model.ModelProvider) (interface{}, error) {
			return s.executeEmbeddingProviderRequest(callCtx, provider, req)
		}

		usedOperation := operation
		rateCtx := scheduler.WithRateLimitUsage(execCtx, buildTokenRateLimitUsage(estimatedPromptTokens, 0))
		result, err := s.cascadeController.ExecuteWithStrategy(rateCtx, operation, req.Model, scheduler.StrategyAdaptive, executor)
		if err != nil {
			var noProviders *scheduler.NoAvailableProviderError
			if errors.As(err, &noProviders) {
				usedOperation = model.OperationChatCompletions
				result, err = s.cascadeController.ExecuteWithStrategy(rateCtx, usedOperation, req.Model, scheduler.StrategyAdaptive, executor)
			}
		}
		if err != nil {
			s.logProviderRequest(token, nil, usedOperation, req.Model, &Usage{}, decimal.Zero, decimal.Zero, int(time.Since(startTime).Milliseconds()), model.LogStatusError, err.Error())
			return nil, err
		}
		if result == nil || result.Response == nil {
			return nil, fmt.Errorf("调度结果为空")
		}

		embeddingResp, ok := result.Response.(*EmbeddingResponse)
		if !ok || embeddingResp == nil {
			return nil, fmt.Errorf("嵌入响应解析失败")
		}

		latencyMs := int(time.Since(startTime).Milliseconds())

		usage := embeddingResp.Usage
		if usage == nil {
			usage = &Usage{}
		}

		cost, _ := s.costCalculator.CalculateCost(req.Model, usage.PromptTokens, usage.CompletionTokens)
		upstreamCost := s.costCalculator.CalculateProviderCost(result.Provider, usage.PromptTokens, usage.CompletionTokens)

		if token.User != nil {
			if err := DeductUserBalance(s.userRepo, token.UserID, cost, &token.User.Balance); err != nil {
				s.logProviderRequest(token, result.Provider, usedOperation, req.Model, usage, decimal.Zero, decimal.Zero, latencyMs, model.LogStatusError, err.Error())
				return nil, err
			}
		}
		if token.RemainQuota != nil {
			if err := s.tokenService.DeductQuota(token.ID, cost); err != nil {
				s.logProviderRequest(token, result.Provider, usedOperation, req.Model, usage, decimal.Zero, decimal.Zero, latencyMs, model.LogStatusError, err.Error())
				return nil, err
			}
		}
		if result.Provider != nil {
			_ = s.providerRepo.IncrementStats(execCtx, result.Provider.ID, true, int64(latencyMs), int64(usage.PromptTokens), int64(usage.CompletionTokens), upstreamCost)
		}
		s.logProviderRequest(token, result.Provider, usedOperation, req.Model, usage, cost, upstreamCost, latencyMs, model.LogStatusSuccess, "")
		s.emitTenantBilling(execCtx, token, usedOperation, req.Model, usage, cost)

		return embeddingResp, nil
	})
}

// ListModels 列出可用模型
func (s *gatewayServiceV2) ListModels() (*ModelsResponse, error) {
	models, err := s.modelRepo.ListEnabled()
	if err != nil {
		return nil, err
	}

	var data []ModelData
	for _, m := range models {
		data = append(data, ModelData{
			ID:      m.ID,
			Object:  "model",
			Created: m.CreatedAt.Unix(),
			OwnedBy: "nexus-api",
		})
	}

	return &ModelsResponse{Object: "list", Data: data}, nil
}

// 辅助方法

// convertMessages 转换消息格式
func convertMessages(messages []ChatMessage) []map[string]interface{} {
	result := make([]map[string]interface{}, len(messages))
	for i, msg := range messages {
		result[i] = map[string]interface{}{
			"role":    msg.Role,
			"content": msg.Content,
		}
		if msg.Name != "" {
			result[i]["name"] = msg.Name
		}
		if msg.ToolCalls != nil {
			result[i]["tool_calls"] = msg.ToolCalls
		}
		if msg.ToolCallID != "" {
			result[i]["tool_call_id"] = msg.ToolCallID
		}
	}
	return result
}

// executeStreamRequest 执行流式请求
func (s *gatewayServiceV2) executeStreamRequest(ctx context.Context, provider *model.ModelProvider, req *ChatCompletionRequest, writer http.ResponseWriter, onFirstChunk func()) error {
	if scheduler.SourceTransportFromContext(ctx) == scheduler.SourceTransportWebSocket {
		return s.executeStreamRequestByWebSocket(ctx, provider, req, writer, onFirstChunk)
	}

	upstreamReq, err := s.buildProviderRequest(ctx, provider, req)
	if err != nil {
		return fmt.Errorf("构建请求失败: %w", err)
	}

	resp, err := s.httpClient.Do(upstreamReq)
	if err != nil {
		return fmt.Errorf("上游请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(body))
		return NewUpstreamHTTPError(resp.StatusCode, bodyStr)
	}

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	flusher, ok := writer.(http.Flusher)
	if !ok {
		return errors.New("streaming not supported")
	}

	reader := bufio.NewReader(resp.Body)
	firstDataFlushed := false
	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			if err == io.EOF {
				break
			}
			return err
		}
		_, err = writer.Write(line)
		if err != nil {
			return err
		}
		flusher.Flush()

		if !firstDataFlushed {
			lineStr := strings.TrimSpace(string(line))
			if strings.HasPrefix(lineStr, "data:") {
				if onFirstChunk != nil {
					onFirstChunk()
				}
				firstDataFlushed = true
			}
		}
	}

	return nil
}

func (s *gatewayServiceV2) executeStreamRequestByWebSocket(ctx context.Context, provider *model.ModelProvider, req *ChatCompletionRequest, writer http.ResponseWriter, onFirstChunk func()) error {
	if provider == nil || provider.Channel == nil {
		return fmt.Errorf("源头缺少渠道信息")
	}

	upstreamModel := provider.UpstreamModelName
	if upstreamModel == "" {
		upstreamModel = req.Model
	}
	reqCopy := *req
	reqCopy.Stream = true
	reqCopy.Model = upstreamModel

	wsURL, err := s.resolveProviderWebSocketURL(provider, "/chat/completions")
	if err != nil {
		return err
	}

	conn, err := dialProviderWebSocket(ctx, wsURL, provider.Channel.APIKey)
	if err != nil {
		return fmt.Errorf("建立 WebSocket 上游连接失败: %w", err)
	}
	defer conn.Close()

	if err := websocket.JSON.Send(conn, reqCopy); err != nil {
		return fmt.Errorf("发送 WebSocket 请求失败: %w", err)
	}

	writer.Header().Set("Content-Type", "text/event-stream")
	writer.Header().Set("Cache-Control", "no-cache")
	writer.Header().Set("Connection", "keep-alive")

	flusher, ok := writer.(http.Flusher)
	if !ok {
		return errors.New("streaming not supported")
	}

	firstDataFlushed := false
	for {
		var chunk string
		if err := websocket.Message.Receive(conn, &chunk); err != nil {
			if err == io.EOF {
				break
			}
			return err
		}

		if !firstDataFlushed && onFirstChunk != nil {
			onFirstChunk()
			firstDataFlushed = true
		}

		output := strings.TrimSpace(chunk)
		if output == "" {
			continue
		}

		if output == "[DONE]" {
			output = "data: [DONE]"
		} else if !strings.HasPrefix(output, "data:") {
			output = "data: " + output
		}

		if !strings.HasSuffix(output, "\n\n") {
			output += "\n\n"
		}

		if _, err := writer.Write([]byte(output)); err != nil {
			return err
		}
		flusher.Flush()

		if strings.Contains(output, "data: [DONE]") {
			break
		}
	}

	return nil
}

func (s *gatewayServiceV2) executeCompletionProviderRequest(execCtx context.Context, provider *model.ModelProvider, req *CompletionRequest) (*CompletionResponse, error) {
	if execCtx == nil {
		execCtx = context.Background()
	}
	if provider == nil || provider.Channel == nil {
		return nil, fmt.Errorf("源头缺少 channel 信息")
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	endpoint := "/completions"
	contentType := "application/json"
	if scheduler.SourceTransportFromContext(execCtx) == scheduler.SourceTransportGRPC {
		if configured := strings.TrimSpace(readProviderChannelConfigString(provider, "grpc_gateway_path", "grpc_path")); configured != "" {
			endpoint = configured
		}
		contentType = "application/grpc+json"
	}

	upstreamURL, err := buildOpenAIURL(provider.Channel.BaseURL, endpoint)
	if err != nil {
		return nil, err
	}

	upstreamReq, err := http.NewRequestWithContext(execCtx, http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	upstreamReq.Header.Set("Content-Type", contentType)
	upstreamReq.Header.Set("Authorization", "Bearer "+provider.Channel.APIKey)
	upstreamReq.Header.Set("X-Source-Transport", string(scheduler.SourceTransportFromContext(execCtx)))

	resp, err := s.httpClient.Do(upstreamReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(body))
		return nil, NewUpstreamHTTPError(resp.StatusCode, bodyStr)
	}

	var completionResp CompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
		return nil, err
	}
	return &completionResp, nil
}

func (s *gatewayServiceV2) executeEmbeddingProviderRequest(execCtx context.Context, provider *model.ModelProvider, req *EmbeddingRequest) (*EmbeddingResponse, error) {
	if execCtx == nil {
		execCtx = context.Background()
	}
	if provider == nil || provider.Channel == nil {
		return nil, fmt.Errorf("源头缺少 channel 信息")
	}

	reqBody, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	endpoint := "/embeddings"
	contentType := "application/json"
	if scheduler.SourceTransportFromContext(execCtx) == scheduler.SourceTransportGRPC {
		if configured := strings.TrimSpace(readProviderChannelConfigString(provider, "grpc_gateway_path", "grpc_path")); configured != "" {
			endpoint = configured
		}
		contentType = "application/grpc+json"
	}

	upstreamURL, err := buildOpenAIURL(provider.Channel.BaseURL, endpoint)
	if err != nil {
		return nil, err
	}

	upstreamReq, err := http.NewRequestWithContext(execCtx, http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}
	upstreamReq.Header.Set("Content-Type", contentType)
	upstreamReq.Header.Set("Authorization", "Bearer "+provider.Channel.APIKey)
	upstreamReq.Header.Set("X-Source-Transport", string(scheduler.SourceTransportFromContext(execCtx)))

	resp, err := s.httpClient.Do(upstreamReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		bodyStr := strings.TrimSpace(string(body))
		return nil, NewUpstreamHTTPError(resp.StatusCode, bodyStr)
	}

	var embeddingResp EmbeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
		return nil, err
	}
	return &embeddingResp, nil
}

func (s *gatewayServiceV2) resolveProviderWebSocketURL(provider *model.ModelProvider, endpoint string) (string, error) {
	if provider == nil || provider.Channel == nil {
		return "", fmt.Errorf("源头缺少渠道信息")
	}

	wsEndpoint := strings.TrimSpace(readProviderChannelConfigString(provider, "websocket_path", "ws_path"))
	if wsEndpoint == "" {
		wsEndpoint = endpoint
	}

	httpURL, err := buildOpenAIURL(provider.Channel.BaseURL, wsEndpoint)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(httpURL)
	if err != nil {
		return "", err
	}

	switch strings.ToLower(strings.TrimSpace(parsed.Scheme)) {
	case "https", "wss":
		parsed.Scheme = "wss"
	case "http", "ws":
		parsed.Scheme = "ws"
	default:
		parsed.Scheme = "wss"
	}

	return parsed.String(), nil
}

func dialProviderWebSocket(ctx context.Context, rawURL string, apiKey string) (*websocket.Conn, error) {
	config, err := websocket.NewConfig(rawURL, "https://nexus-api.local")
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(apiKey) != "" {
		config.Header = http.Header{}
		config.Header.Set("Authorization", "Bearer "+apiKey)
	}

	conn, err := websocket.DialConfig(config)
	if err != nil {
		return nil, err
	}

	if ctx != nil {
		if deadline, ok := ctx.Deadline(); ok {
			_ = conn.SetDeadline(deadline)
		}
		conn.PayloadType = websocket.TextFrame
	}

	return conn, nil
}

func readProviderChannelConfigString(provider *model.ModelProvider, keys ...string) string {
	if provider == nil || provider.Channel == nil || provider.Channel.Config == nil || len(keys) == 0 {
		return ""
	}

	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		raw, ok := provider.Channel.Config[key]
		if !ok || raw == nil {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

// logProviderRequest 记录Provider请求日志
func (s *gatewayServiceV2) logProviderRequest(token *model.Token, provider *model.ModelProvider, operation string, modelName string, usage *Usage, cost, upstreamCost decimal.Decimal, latency int, status model.LogStatus, errMsg string) {
	if token == nil {
		return
	}
	if usage == nil {
		usage = &Usage{}
	}

	op := model.NormalizeOperation(operation)

	var metadata model.JSON
	if strings.TrimSpace(op) != "" {
		metadata = model.JSON{"operation": op}
	}
	if token != nil {
		tenantID := strings.TrimSpace(token.EffectiveTenantID())
		if tenantID != "" {
			if metadata == nil {
				metadata = model.JSON{}
			}
			metadata["tenant_id"] = tenantID
		}
	}

	log := &model.Log{
		ID:               uuid.New(),
		UserID:           token.UserID,
		TokenID:          token.ID,
		Model:            modelName,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalCost:        cost,
		UpstreamCost:     upstreamCost,
		Duration:         latency,
		Status:           status,
		ErrorMessage:     errMsg,
		Metadata:         metadata,
	}
	if provider != nil && provider.Channel != nil {
		log.ChannelID = provider.Channel.ID
	}
	_ = s.logRepo.Create(log)
}

func (s *gatewayServiceV2) checkTenantQuota(ctx context.Context, token *model.Token, estimatedCost decimal.Decimal) error {
	if s == nil || s.tenantBillingHook == nil || token == nil {
		return nil
	}

	tenantID := strings.TrimSpace(token.EffectiveTenantID())
	if tenantID == "" {
		return nil
	}

	return s.tenantBillingHook.CheckQuota(ctx, tenantID, estimatedCost)
}

func (s *gatewayServiceV2) emitTenantBilling(ctx context.Context, token *model.Token, operation string, modelName string, usage *Usage, cost decimal.Decimal) {
	if s == nil || s.tenantBillingHook == nil || token == nil {
		return
	}
	tenantID := strings.TrimSpace(token.EffectiveTenantID())
	if tenantID == "" {
		return
	}
	if usage == nil {
		usage = &Usage{}
	}

	_ = s.tenantBillingHook.OnBilled(ctx, &TenantBillingEvent{
		TenantID:         tenantID,
		UserID:           token.UserID,
		TokenID:          token.ID,
		Operation:        model.NormalizeOperation(operation),
		Model:            modelName,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		Cost:             cost,
		OccurredAt:       time.Now(),
	})
}

func buildTokenRateLimitUsage(promptTokens, completionTokens int) map[string]int64 {
	usage := map[string]int64{
		"request": 1,
	}
	totalTokens := promptTokens + completionTokens
	if totalTokens > 0 {
		usage["token"] = int64(totalTokens)
		usage["tokens"] = int64(totalTokens)
	}
	return usage
}

func (s *gatewayServiceV2) estimateChatTokens(req *ChatCompletionRequest) (int, int) {
	if s == nil || s.costCalculator == nil || req == nil {
		return 0, 0
	}
	maxTokens := 0
	if req.MaxTokens != nil {
		maxTokens = *req.MaxTokens
	}
	schedulerReq := &scheduler.ChatCompletionRequest{
		Model:     req.Model,
		Messages:  convertMessages(req.Messages),
		MaxTokens: maxTokens,
		Stream:    req.Stream,
	}
	promptTokens := s.costCalculator.EstimatePromptTokens(schedulerReq)
	completionTokens := s.costCalculator.EstimateCompletionTokens(schedulerReq)
	return promptTokens, completionTokens
}

func estimateCompletionTokens(req *CompletionRequest) (int, int) {
	if req == nil {
		return 0, 0
	}
	promptTokens := 100
	switch v := req.Prompt.(type) {
	case string:
		promptTokens = len(v) / 4
	case []string:
		total := 0
		for _, item := range v {
			total += len(item)
		}
		promptTokens = total / 4
	case []interface{}:
		total := 0
		for _, item := range v {
			if item == nil {
				continue
			}
			if s, ok := item.(string); ok {
				total += len(s)
				continue
			}
			total += len(fmt.Sprintf("%v", item))
		}
		promptTokens = total / 4
	}

	completionTokens := 100
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		completionTokens = *req.MaxTokens
	}
	return promptTokens, completionTokens
}

func estimateEmbeddingTokens(req *EmbeddingRequest) int {
	if req == nil {
		return 0
	}
	switch v := req.Input.(type) {
	case string:
		return len(v) / 4
	case []string:
		total := 0
		for _, item := range v {
			total += len(item)
		}
		return total / 4
	case []interface{}:
		total := 0
		for _, item := range v {
			if item == nil {
				continue
			}
			if s, ok := item.(string); ok {
				total += len(s)
				continue
			}
			total += len(fmt.Sprintf("%v", item))
		}
		return total / 4
	default:
		return 0
	}
}
