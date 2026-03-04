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
	"strings"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	"nexus-api/internal/service/scheduler"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// GatewayService Gateway 服务接口
type GatewayService interface {
	HandleChatCompletion(req *ChatCompletionRequest, token *model.Token) (*ChatCompletionResponse, error)
	HandleChatCompletionStream(req *ChatCompletionRequest, token *model.Token, writer http.ResponseWriter) error
	HandleCompletion(req *CompletionRequest, token *model.Token) (*CompletionResponse, error)
	HandleEmbedding(req *EmbeddingRequest, token *model.Token) (*EmbeddingResponse, error)
	ListModels() (*ModelsResponse, error)
}

// gatewayService Gateway 服务实现
type gatewayService struct {
	channelService ChannelService
	tokenService   TokenService
	userRepo       repository.UserRepository
	logRepo        repository.LogRepository
	modelRepo      repository.ModelRepository
	resourceRepo   repository.ResourceAccountRepository
	channelRepo    repository.ChannelRepository
	httpClient     *http.Client
	// ⭐ 新增：调度组件
	providerSelector     scheduler.ProviderSelector
	instanceScheduler    scheduler.InstanceScheduler
	cascadeController    scheduler.CascadeController
	circuitBreaker       scheduler.CircuitBreaker
	streamGuard          scheduler.StreamGuard
	healthTracker        scheduler.HealthTracker
	costCalculator       scheduler.CostCalculator
	runtimeStateStore    scheduler.RuntimeStateStore
	errorClassifier      scheduler.ErrorClassifier
	modelProviderRepo    repository.ModelProviderRepository
	providerInstanceRepo repository.ProviderInstanceRepository
}

// NewGatewayService 创建 Gateway 服务
func NewGatewayService(
	channelService ChannelService,
	tokenService TokenService,
	userRepo repository.UserRepository,
	logRepo repository.LogRepository,
	modelRepo repository.ModelRepository,
	resourceRepo repository.ResourceAccountRepository,
	channelRepo repository.ChannelRepository,
	// ⭐ 新增：调度组件
	providerSelector scheduler.ProviderSelector,
	instanceScheduler scheduler.InstanceScheduler,
	cascadeController scheduler.CascadeController,
	circuitBreaker scheduler.CircuitBreaker,
	streamGuard scheduler.StreamGuard,
	healthTracker scheduler.HealthTracker,
	costCalculator scheduler.CostCalculator,
	runtimeStateStore scheduler.RuntimeStateStore,
	errorClassifier scheduler.ErrorClassifier,
	modelProviderRepo repository.ModelProviderRepository,
	providerInstanceRepo repository.ProviderInstanceRepository,
) GatewayService {
	return &gatewayService{
		channelService: channelService,
		tokenService:   tokenService,
		userRepo:       userRepo,
		logRepo:        logRepo,
		modelRepo:      modelRepo,
		resourceRepo:   resourceRepo,
		channelRepo:    channelRepo,
		httpClient: &http.Client{
			Timeout: 5 * time.Minute,
		},
		// ⭐ 新增：调度组件
		providerSelector:     providerSelector,
		instanceScheduler:    instanceScheduler,
		cascadeController:    cascadeController,
		circuitBreaker:       circuitBreaker,
		streamGuard:          streamGuard,
		healthTracker:        healthTracker,
		costCalculator:       costCalculator,
		runtimeStateStore:    runtimeStateStore,
		errorClassifier:      errorClassifier,
		modelProviderRepo:    modelProviderRepo,
		providerInstanceRepo: providerInstanceRepo,
	}
}

// ChatCompletionRequest 聊天完成请求
type ChatCompletionRequest struct {
	Model            string                 `json:"model"`
	Messages         []ChatMessage          `json:"messages"`
	Temperature      *float64               `json:"temperature,omitempty"`
	TopP             *float64               `json:"top_p,omitempty"`
	N                *int                   `json:"n,omitempty"`
	Stream           bool                   `json:"stream,omitempty"`
	Stop             interface{}            `json:"stop,omitempty"`
	MaxTokens        *int                   `json:"max_tokens,omitempty"`
	PresencePenalty  *float64               `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64               `json:"frequency_penalty,omitempty"`
	LogitBias        map[string]float64     `json:"logit_bias,omitempty"`
	User             string                 `json:"user,omitempty"`
	Functions        []interface{}          `json:"functions,omitempty"`
	FunctionCall     interface{}            `json:"function_call,omitempty"`
	Tools            []interface{}          `json:"tools,omitempty"`
	ToolChoice       interface{}            `json:"tool_choice,omitempty"`
	ResponseFormat   map[string]interface{} `json:"response_format,omitempty"`
}

// ChatMessage 聊天消息
type ChatMessage struct {
	Role       string      `json:"role"`
	Content    interface{} `json:"content"`
	Name       string      `json:"name,omitempty"`
	ToolCalls  interface{} `json:"tool_calls,omitempty"`
	ToolCallID string      `json:"tool_call_id,omitempty"`
}

// ChatCompletionResponse 聊天完成响应
type ChatCompletionResponse struct {
	ID                string   `json:"id"`
	Object            string   `json:"object"`
	Created           int64    `json:"created"`
	Model             string   `json:"model"`
	Choices           []Choice `json:"choices"`
	Usage             *Usage   `json:"usage,omitempty"`
	SystemFingerprint string   `json:"system_fingerprint,omitempty"`
}

// Choice 选择
type Choice struct {
	Index        int          `json:"index"`
	Message      *ChatMessage `json:"message,omitempty"`
	Delta        *ChatMessage `json:"delta,omitempty"`
	FinishReason *string      `json:"finish_reason,omitempty"`
}

// Usage 使用量
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

// CompletionRequest 完成请求
type CompletionRequest struct {
	Model            string             `json:"model"`
	Prompt           interface{}        `json:"prompt"`
	Suffix           string             `json:"suffix,omitempty"`
	MaxTokens        *int               `json:"max_tokens,omitempty"`
	Temperature      *float64           `json:"temperature,omitempty"`
	TopP             *float64           `json:"top_p,omitempty"`
	N                *int               `json:"n,omitempty"`
	Stream           bool               `json:"stream,omitempty"`
	Logprobs         *int               `json:"logprobs,omitempty"`
	Echo             bool               `json:"echo,omitempty"`
	Stop             interface{}        `json:"stop,omitempty"`
	PresencePenalty  *float64           `json:"presence_penalty,omitempty"`
	FrequencyPenalty *float64           `json:"frequency_penalty,omitempty"`
	BestOf           *int               `json:"best_of,omitempty"`
	LogitBias        map[string]float64 `json:"logit_bias,omitempty"`
	User             string             `json:"user,omitempty"`
}

// CompletionResponse 完成响应
type CompletionResponse struct {
	ID      string             `json:"id"`
	Object  string             `json:"object"`
	Created int64              `json:"created"`
	Model   string             `json:"model"`
	Choices []CompletionChoice `json:"choices"`
	Usage   *Usage             `json:"usage,omitempty"`
}

// CompletionChoice 完成选择
type CompletionChoice struct {
	Text         string  `json:"text"`
	Index        int     `json:"index"`
	Logprobs     *int    `json:"logprobs,omitempty"`
	FinishReason *string `json:"finish_reason,omitempty"`
}

// EmbeddingRequest 嵌入请求
type EmbeddingRequest struct {
	Model          string      `json:"model"`
	Input          interface{} `json:"input"`
	User           string      `json:"user,omitempty"`
	EncodingFormat string      `json:"encoding_format,omitempty"`
}

// EmbeddingResponse 嵌入响应
type EmbeddingResponse struct {
	Object string          `json:"object"`
	Data   []EmbeddingData `json:"data"`
	Model  string          `json:"model"`
	Usage  *Usage          `json:"usage,omitempty"`
}

// EmbeddingData 嵌入数据
type EmbeddingData struct {
	Object    string    `json:"object"`
	Embedding []float64 `json:"embedding"`
	Index     int       `json:"index"`
}

// ModelsResponse 模型列表响应
type ModelsResponse struct {
	Object string      `json:"object"`
	Data   []ModelData `json:"data"`
}

// ModelData 模型数据
type ModelData struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	OwnedBy string `json:"owned_by"`
}

// HandleChatCompletion 处理聊天完成请求（带failover重试）
func (s *gatewayService) HandleChatCompletion(req *ChatCompletionRequest, token *model.Token) (*ChatCompletionResponse, error) {
	ctx := context.Background()
	startTime := time.Now()

	// 获取模型定价
	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		return nil, err
	}

	// 估算请求成本（用于预校验）
	estimatedCost := s.estimateCost(req, modelPricing)

	// 校验用户余额
	if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
		return nil, &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
	}

	// 校验Token配额
	if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
		return nil, &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
	}

	// ⭐ 使用ProviderSelector按成本优先选择ProviderGroup
	providers, err := s.providerSelector.SelectProviders(ctx, model.OperationChatCompletions, req.Model, scheduler.StrategyCostFirst)
	if err != nil {
		return nil, fmt.Errorf("获取ProviderGroup失败: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("没有可用的ProviderGroup支持该模型")
	}

	var lastError error
	var usedProvider *model.ModelProvider
	var usedInstance *model.ProviderInstance

	// ⭐ 使用CascadeController进行级联failover
	for _, provider := range providers {
		usedProvider = provider

		// ⭐ 检查熔断器
		canExecute, err := s.circuitBreaker.CanExecute(ctx, provider.ID)
		if err != nil {
			lastError = fmt.Errorf("检查熔断器失败: %w", err)
			continue
		}
		if !canExecute {
			lastError = fmt.Errorf("ProviderGroup %d 熔断器已打开", provider.ID)
			continue
		}

		// ⭐ 使用InstanceScheduler选择实例
		instance, err := s.instanceScheduler.SelectInstance(ctx, provider.ID, model.OperationChatCompletions)
		if err != nil {
			lastError = fmt.Errorf("选择实例失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		usedInstance = instance

		// ⭐ 获取渠道信息
		channel, err := s.channelRepo.GetByID(provider.ChannelID)
		if err != nil {
			lastError = fmt.Errorf("获取渠道失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 构建上游请求
		upstreamReq, err := s.buildUpstreamRequest(channel, req)
		if err != nil {
			lastError = err
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 发送请求
		resp, err := s.httpClient.Do(upstreamReq)
		if err != nil {
			lastError = fmt.Errorf("上游请求失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			bodyStr := strings.TrimSpace(string(body))
			lastError = NewUpstreamHTTPError(resp.StatusCode, bodyStr)

			// ⭐ 使用ErrorClassifier分类错误
			if s.errorClassifier.ShouldCircuitBreak(lastError) {
				s.circuitBreaker.RecordFailure(ctx, provider.ID, lastError)
			}
			continue
		}

		// 解析响应
		var chatResp ChatCompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&chatResp); err != nil {
			_ = resp.Body.Close()
			lastError = fmt.Errorf("解析响应失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		_ = resp.Body.Close()

		// 成功！计算延迟
		latency := time.Since(startTime).Milliseconds()

		// ⭐ 记录成功到熔断器
		s.circuitBreaker.RecordSuccess(ctx, provider.ID)

		// ⭐ 更新健康分数
		_ = s.healthTracker.UpdateHealthScore(ctx, provider.ID)

		// ⭐ 更新ProviderGroup统计
		inputTokens := int64(0)
		outputTokens := int64(0)
		if chatResp.Usage != nil {
			inputTokens = int64(chatResp.Usage.PromptTokens)
			outputTokens = int64(chatResp.Usage.CompletionTokens)
		}
		cost, _ := s.costCalculator.CalculateCost(req.Model, int(inputTokens), int(outputTokens))
		_ = s.modelProviderRepo.IncrementStats(ctx, provider.ID, true, int64(latency), inputTokens, outputTokens, cost)

		// ⭐ 更新实例统计
		if usedInstance != nil {
			_ = s.providerInstanceRepo.IncrementStats(ctx, usedInstance.ID, true, int64(latency))
		}

		// 计算费用并扣费
		if chatResp.Usage != nil {
			cost := s.calculateCost(chatResp.Usage, modelPricing)
			upstreamCost := s.calculateUpstreamCost(cost, channel, req.Model)

			// 扣除用户余额
			if token.User != nil {
				if err := DeductUserBalance(s.userRepo, token.UserID, cost, &token.User.Balance); err != nil {
					return nil, err
				}
			}

			// 扣除 Token 配额
			if token.RemainQuota != nil {
				if err := s.tokenService.DeductQuota(token.ID, cost); err != nil {
					return nil, err
				}
			}

			// 记录日志
			s.logRequest(token, channel, req.Model, chatResp.Usage, cost, upstreamCost, int(latency), model.LogStatusSuccess, "")
		}

		return &chatResp, nil
	}

	// 所有ProviderGroup都失败，记录失败日志
	if usedProvider != nil {
		channel, _ := s.channelRepo.GetByID(usedProvider.ChannelID)
		s.logRequest(token, channel, req.Model, &Usage{}, decimal.Zero, decimal.Zero, int(time.Since(startTime).Milliseconds()), model.LogStatusError, lastError.Error())
	}

	return nil, fmt.Errorf("所有ProviderGroup均失败，最后错误: %w", lastError)
}

// HandleChatCompletionStream 处理流式聊天完成请求（带Stream First-Chunk Failover）
func (s *gatewayService) HandleChatCompletionStream(req *ChatCompletionRequest, token *model.Token, writer http.ResponseWriter) error {
	ctx := context.Background()
	startTime := time.Now()

	// 获取模型定价
	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		return err
	}

	// 估算请求成本（用于预校验）
	estimatedCost := s.estimateCost(req, modelPricing)

	// 校验用户余额
	if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
		return &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
	}

	// 校验Token配额
	if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
		return &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
	}

	// ⭐ 使用ProviderSelector按成本优先选择ProviderGroup
	providers, err := s.providerSelector.SelectProviders(ctx, model.OperationChatCompletions, req.Model, scheduler.StrategyCostFirst)
	if err != nil {
		return fmt.Errorf("获取ProviderGroup失败: %w", err)
	}

	if len(providers) == 0 {
		return fmt.Errorf("没有可用的ProviderGroup支持该模型")
	}

	var lastError error
	var usedInstance *model.ProviderInstance
	var usedChannel *model.Channel

	// ⭐ 使用CascadeController进行级联failover
	for _, provider := range providers {

		// ⭐ 检查熔断器
		canExecute, err := s.circuitBreaker.CanExecute(ctx, provider.ID)
		if err != nil {
			lastError = fmt.Errorf("检查熔断器失败: %w", err)
			continue
		}
		if !canExecute {
			lastError = fmt.Errorf("ProviderGroup %d 熔断器已打开", provider.ID)
			continue
		}

		// ⭐ 使用InstanceScheduler选择实例
		instance, err := s.instanceScheduler.SelectInstance(ctx, provider.ID, model.OperationChatCompletions)
		if err != nil {
			lastError = fmt.Errorf("选择实例失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		usedInstance = instance

		// ⭐ 获取渠道信息
		channel, err := s.channelRepo.GetByID(provider.ChannelID)
		if err != nil {
			lastError = fmt.Errorf("获取渠道失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		usedChannel = channel

		// ⭐ 启动Stream Guard
		streamCtx, err := s.streamGuard.StartStream(ctx, provider.ID)
		if err != nil {
			lastError = fmt.Errorf("启动Stream Guard失败: %w", err)
			continue
		}

		// 确保是流式请求
		req.Stream = true

		// 构建上游请求
		upstreamReq, err := s.buildUpstreamRequest(channel, req)
		if err != nil {
			lastError = err
			s.streamGuard.EndStream(ctx, streamCtx, false)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 发送请求
		resp, err := s.httpClient.Do(upstreamReq)
		if err != nil {
			lastError = fmt.Errorf("上游请求失败: %w", err)
			s.streamGuard.EndStream(ctx, streamCtx, false)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			bodyStr := strings.TrimSpace(string(body))
			lastError = NewUpstreamHTTPError(resp.StatusCode, bodyStr)
			s.streamGuard.EndStream(ctx, streamCtx, false)

			// ⭐ 使用ErrorClassifier分类错误
			if s.errorClassifier.ShouldCircuitBreak(lastError) {
				s.circuitBreaker.RecordFailure(ctx, provider.ID, lastError)
			}
			continue
		}
		defer resp.Body.Close()

		// 设置 SSE 响应头
		writer.Header().Set("Content-Type", "text/event-stream")
		writer.Header().Set("Cache-Control", "no-cache")
		writer.Header().Set("Connection", "keep-alive")
		writer.Header().Set("Transfer-Encoding", "chunked")

		flusher, ok := writer.(http.Flusher)
		if !ok {
			s.streamGuard.EndStream(ctx, streamCtx, false)
			return errors.New("streaming not supported")
		}

		// 读取并转发流式响应
		reader := bufio.NewReader(resp.Body)
		var totalPromptTokens, totalCompletionTokens int
		var firstChunkSent bool

		for {
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					break
				}
				s.streamGuard.EndStream(ctx, streamCtx, false)
				return err
			}

			// ⭐ 检查Stream Guard是否允许发送
			if !firstChunkSent {
				locked, err := s.streamGuard.IsLocked(ctx, streamCtx)
				if err != nil {
					s.streamGuard.EndStream(ctx, streamCtx, false)
					return err
				}
				if locked {
					// Stream已被锁定，需要切换到下一个Provider
					s.streamGuard.EndStream(ctx, streamCtx, false)
					break
				}
				// ⭐ 标记第一个chunk已发送
				s.streamGuard.OnFirstChunk(ctx, streamCtx)
				firstChunkSent = true
			}

			// 转发数据
			_, _ = writer.Write(line)
			flusher.Flush()

			// 解析 SSE 数据以统计 Token
			lineStr := strings.TrimSpace(string(line))
			if strings.HasPrefix(lineStr, "data: ") {
				data := strings.TrimPrefix(lineStr, "data: ")
				if data == "[DONE]" {
					break
				}

				var chunk ChatCompletionResponse
				if err := json.Unmarshal([]byte(data), &chunk); err == nil {
					if chunk.Usage != nil {
						totalPromptTokens = chunk.Usage.PromptTokens
						totalCompletionTokens = chunk.Usage.CompletionTokens
					}
				}
			}
		}

		s.streamGuard.EndStream(ctx, streamCtx, true)

		// 如果成功发送了第一个chunk，则直接返回
		if firstChunkSent {
			// 成功！计算延迟
			latency := time.Since(startTime).Milliseconds()

			// ⭐ 记录成功到熔断器
			s.circuitBreaker.RecordSuccess(ctx, provider.ID)

			// ⭐ 更新健康分数
			_ = s.healthTracker.UpdateHealthScore(ctx, provider.ID)

			// ⭐ 更新ProviderGroup统计
			inputTokens := int64(totalPromptTokens)
			outputTokens := int64(totalCompletionTokens)
			cost, _ := s.costCalculator.CalculateCost(req.Model, int(inputTokens), int(outputTokens))
			_ = s.modelProviderRepo.IncrementStats(ctx, provider.ID, true, int64(latency), inputTokens, outputTokens, cost)

			// ⭐ 更新实例统计
			if usedInstance != nil {
				_ = s.providerInstanceRepo.IncrementStats(ctx, usedInstance.ID, true, int64(latency))
			}

			// 如果没有从流中获取到 Token 数，估算
			if totalPromptTokens == 0 {
				totalPromptTokens = s.estimatePromptTokens(req)
			}
			if totalCompletionTokens == 0 {
				totalCompletionTokens = 100 // 默认估算
			}

			// 计算费用并扣费
			usage := &Usage{
				PromptTokens:     totalPromptTokens,
				CompletionTokens: totalCompletionTokens,
				TotalTokens:      totalPromptTokens + totalCompletionTokens,
			}

			cost = s.calculateCost(usage, modelPricing)
			upstreamCost := s.calculateUpstreamCost(cost, channel, req.Model)

			// 扣除用户余额
			if token.User != nil {
				if err := DeductUserBalance(s.userRepo, token.UserID, cost, &token.User.Balance); err != nil {
					return err
				}
			}

			// 扣除 Token 配额
			if token.RemainQuota != nil {
				if err := s.tokenService.DeductQuota(token.ID, cost); err != nil {
					return err
				}
			}

			// 记录日志
			s.logRequest(token, channel, req.Model, usage, cost, upstreamCost, int(latency), model.LogStatusSuccess, "")

			return nil
		}

		// 如果没有成功发送第一个chunk，继续尝试下一个Provider
		continue
	}

	// 所有ProviderGroup都失败，记录失败日志
	if usedChannel != nil {
		s.logRequest(token, usedChannel, req.Model, &Usage{}, decimal.Zero, decimal.Zero, int(time.Since(startTime).Milliseconds()), model.LogStatusError, lastError.Error())
	}

	return fmt.Errorf("所有ProviderGroup均失败，最后错误: %w", lastError)
}

// HandleCompletion 处理完成请求（带failover重试）
func (s *gatewayService) HandleCompletion(req *CompletionRequest, token *model.Token) (*CompletionResponse, error) {
	ctx := context.Background()
	startTime := time.Now()

	// 获取模型定价
	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		return nil, err
	}

	// 估算请求成本（用于预校验）
	estimatedCost := s.estimateCompletionCost(req, modelPricing)

	// 校验用户余额
	if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
		return nil, &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
	}

	// 校验Token配额
	if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
		return nil, &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
	}

	// ⭐ 使用ProviderSelector按成本优先选择ProviderGroup
	providers, err := s.providerSelector.SelectProviders(ctx, model.OperationCompletions, req.Model, scheduler.StrategyCostFirst)
	if err != nil {
		return nil, fmt.Errorf("获取ProviderGroup失败: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("没有可用的ProviderGroup支持该模型")
	}

	var lastError error
	var usedProvider *model.ModelProvider
	var usedInstance *model.ProviderInstance

	// ⭐ 使用CascadeController进行级联failover
	for _, provider := range providers {
		usedProvider = provider

		// ⭐ 检查熔断器
		canExecute, err := s.circuitBreaker.CanExecute(ctx, provider.ID)
		if err != nil {
			lastError = fmt.Errorf("检查熔断器失败: %w", err)
			continue
		}
		if !canExecute {
			lastError = fmt.Errorf("ProviderGroup %d 熔断器已打开", provider.ID)
			continue
		}

		// ⭐ 使用InstanceScheduler选择实例
		instance, err := s.instanceScheduler.SelectInstance(ctx, provider.ID, model.OperationCompletions)
		if err != nil {
			lastError = fmt.Errorf("选择实例失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		usedInstance = instance

		// ⭐ 获取渠道信息
		channel, err := s.channelRepo.GetByID(provider.ChannelID)
		if err != nil {
			lastError = fmt.Errorf("获取渠道失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 构建上游请求
		reqBody, err := json.Marshal(req)
		if err != nil {
			lastError = err
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		upstreamURL, err := buildOpenAIURL(channel.BaseURL, "/completions")
		if err != nil {
			lastError = err
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		upstreamReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
		if err != nil {
			lastError = err
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		upstreamReq.Header.Set("Content-Type", "application/json")
		upstreamReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

		// 发送请求
		resp, err := s.httpClient.Do(upstreamReq)
		if err != nil {
			lastError = fmt.Errorf("上游请求失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			bodyStr := strings.TrimSpace(string(body))
			lastError = NewUpstreamHTTPError(resp.StatusCode, bodyStr)

			// ⭐ 使用ErrorClassifier分类错误
			if s.errorClassifier.ShouldCircuitBreak(lastError) {
				s.circuitBreaker.RecordFailure(ctx, provider.ID, lastError)
			}
			continue
		}

		// 解析响应
		var completionResp CompletionResponse
		if err := json.NewDecoder(resp.Body).Decode(&completionResp); err != nil {
			_ = resp.Body.Close()
			lastError = fmt.Errorf("解析响应失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		_ = resp.Body.Close()

		// 成功！计算延迟
		latency := time.Since(startTime).Milliseconds()

		// ⭐ 记录成功到熔断器
		s.circuitBreaker.RecordSuccess(ctx, provider.ID)

		// ⭐ 更新健康分数
		_ = s.healthTracker.UpdateHealthScore(ctx, provider.ID)

		// ⭐ 更新ProviderGroup统计
		inputTokens := int64(0)
		outputTokens := int64(0)
		if completionResp.Usage != nil {
			inputTokens = int64(completionResp.Usage.PromptTokens)
			outputTokens = int64(completionResp.Usage.CompletionTokens)
		}
		cost, _ := s.costCalculator.CalculateCost(req.Model, int(inputTokens), int(outputTokens))
		_ = s.modelProviderRepo.IncrementStats(ctx, provider.ID, true, int64(latency), inputTokens, outputTokens, cost)

		// ⭐ 更新实例统计
		if usedInstance != nil {
			_ = s.providerInstanceRepo.IncrementStats(ctx, usedInstance.ID, true, int64(latency))
		}

		// 计算费用并扣费
		if completionResp.Usage != nil {
			cost := s.calculateCost(completionResp.Usage, modelPricing)
			upstreamCost := s.calculateUpstreamCost(cost, channel, req.Model)

			// 扣除用户余额
			if token.User != nil {
				if err := DeductUserBalance(s.userRepo, token.UserID, cost, &token.User.Balance); err != nil {
					return nil, err
				}
			}

			// 扣除 Token 配额
			if token.RemainQuota != nil {
				if err := s.tokenService.DeductQuota(token.ID, cost); err != nil {
					return nil, err
				}
			}

			// 记录日志
			s.logRequest(token, channel, req.Model, completionResp.Usage, cost, upstreamCost, int(latency), model.LogStatusSuccess, "")
		}

		return &completionResp, nil
	}

	// 所有ProviderGroup都失败，记录失败日志
	if usedProvider != nil {
		channel, _ := s.channelRepo.GetByID(usedProvider.ChannelID)
		s.logRequest(token, channel, req.Model, &Usage{}, decimal.Zero, decimal.Zero, int(time.Since(startTime).Milliseconds()), model.LogStatusError, lastError.Error())
	}

	return nil, fmt.Errorf("所有ProviderGroup均失败，最后错误: %w", lastError)
}

// HandleEmbedding 处理嵌入请求（带failover重试）
func (s *gatewayService) HandleEmbedding(req *EmbeddingRequest, token *model.Token) (*EmbeddingResponse, error) {
	ctx := context.Background()
	startTime := time.Now()

	// 获取模型定价
	modelPricing, err := s.modelRepo.GetPricing(req.Model)
	if err != nil {
		return nil, err
	}

	// 估算请求成本（用于预校验）
	estimatedCost := s.estimateEmbeddingCost(req, modelPricing)

	// 校验用户余额
	if token.User != nil && token.User.Balance.LessThan(estimatedCost) {
		return nil, &InsufficientFundsError{Needed: estimatedCost, Balance: token.User.Balance}
	}

	// 校验Token配额
	if token.RemainQuota != nil && !token.UnlimitedQuota && token.RemainQuota.LessThan(estimatedCost) {
		return nil, &QuotaExceededError{Needed: estimatedCost, Remaining: *token.RemainQuota}
	}

	// ⭐ 使用ProviderSelector按成本优先选择ProviderGroup
	providers, err := s.providerSelector.SelectProviders(ctx, model.OperationEmbeddings, req.Model, scheduler.StrategyCostFirst)
	if err != nil {
		return nil, fmt.Errorf("获取ProviderGroup失败: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("没有可用的ProviderGroup支持该模型")
	}

	var lastError error
	var usedProvider *model.ModelProvider
	var usedInstance *model.ProviderInstance

	// ⭐ 使用CascadeController进行级联failover
	for _, provider := range providers {
		usedProvider = provider

		// ⭐ 检查熔断器
		canExecute, err := s.circuitBreaker.CanExecute(ctx, provider.ID)
		if err != nil {
			lastError = fmt.Errorf("检查熔断器失败: %w", err)
			continue
		}
		if !canExecute {
			lastError = fmt.Errorf("ProviderGroup %d 熔断器已打开", provider.ID)
			continue
		}

		// ⭐ 使用InstanceScheduler选择实例
		instance, err := s.instanceScheduler.SelectInstance(ctx, provider.ID, model.OperationEmbeddings)
		if err != nil {
			lastError = fmt.Errorf("选择实例失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		usedInstance = instance

		// ⭐ 获取渠道信息
		channel, err := s.channelRepo.GetByID(provider.ChannelID)
		if err != nil {
			lastError = fmt.Errorf("获取渠道失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 构建上游请求
		reqBody, err := json.Marshal(req)
		if err != nil {
			lastError = err
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		upstreamURL, err := buildOpenAIURL(channel.BaseURL, "/embeddings")
		if err != nil {
			lastError = err
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		upstreamReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
		if err != nil {
			lastError = err
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		upstreamReq.Header.Set("Content-Type", "application/json")
		upstreamReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

		// 发送请求
		resp, err := s.httpClient.Do(upstreamReq)
		if err != nil {
			lastError = fmt.Errorf("上游请求失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}

		// 检查响应状态
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			bodyStr := strings.TrimSpace(string(body))
			lastError = NewUpstreamHTTPError(resp.StatusCode, bodyStr)

			// ⭐ 使用ErrorClassifier分类错误
			if s.errorClassifier.ShouldCircuitBreak(lastError) {
				s.circuitBreaker.RecordFailure(ctx, provider.ID, lastError)
			}
			continue
		}

		// 解析响应
		var embeddingResp EmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embeddingResp); err != nil {
			_ = resp.Body.Close()
			lastError = fmt.Errorf("解析响应失败: %w", err)
			s.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			continue
		}
		_ = resp.Body.Close()

		// 成功！计算延迟
		latency := time.Since(startTime).Milliseconds()

		// ⭐ 记录成功到熔断器
		s.circuitBreaker.RecordSuccess(ctx, provider.ID)

		// ⭐ 更新健康分数
		_ = s.healthTracker.UpdateHealthScore(ctx, provider.ID)

		// ⭐ 更新ProviderGroup统计
		inputTokens := int64(0)
		outputTokens := int64(0)
		if embeddingResp.Usage != nil {
			inputTokens = int64(embeddingResp.Usage.PromptTokens)
			outputTokens = int64(embeddingResp.Usage.CompletionTokens)
		}
		cost, _ := s.costCalculator.CalculateCost(req.Model, int(inputTokens), int(outputTokens))
		_ = s.modelProviderRepo.IncrementStats(ctx, provider.ID, true, int64(latency), inputTokens, outputTokens, cost)

		// ⭐ 更新实例统计
		if usedInstance != nil {
			_ = s.providerInstanceRepo.IncrementStats(ctx, usedInstance.ID, true, int64(latency))
		}

		// 计算费用并扣费
		if embeddingResp.Usage != nil {
			cost := s.calculateCost(embeddingResp.Usage, modelPricing)
			upstreamCost := s.calculateUpstreamCost(cost, channel, req.Model)

			// 扣除用户余额
			if token.User != nil {
				if err := DeductUserBalance(s.userRepo, token.UserID, cost, &token.User.Balance); err != nil {
					return nil, err
				}
			}

			// 扣除 Token 配额
			if token.RemainQuota != nil {
				if err := s.tokenService.DeductQuota(token.ID, cost); err != nil {
					return nil, err
				}
			}

			// 记录日志
			s.logRequest(token, channel, req.Model, embeddingResp.Usage, cost, upstreamCost, int(latency), model.LogStatusSuccess, "")
		}

		return &embeddingResp, nil
	}

	// 所有ProviderGroup都失败，记录失败日志
	if usedProvider != nil {
		channel, _ := s.channelRepo.GetByID(usedProvider.ChannelID)
		s.logRequest(token, channel, req.Model, &Usage{}, decimal.Zero, decimal.Zero, int(time.Since(startTime).Milliseconds()), model.LogStatusError, lastError.Error())
	}

	return nil, fmt.Errorf("所有ProviderGroup均失败，最后错误: %w", lastError)
}

// ListModels 列出可用模型
func (s *gatewayService) ListModels() (*ModelsResponse, error) {
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

	return &ModelsResponse{
		Object: "list",
		Data:   data,
	}, nil
}

// buildUpstreamRequest 构建上游请求
func (s *gatewayService) buildUpstreamRequest(channel *model.Channel, req *ChatCompletionRequest) (*http.Request, error) {
	// 模型映射
	upstreamModel := req.Model
	if channel.ModelMapping != nil {
		if mapped, ok := channel.ModelMapping[req.Model]; ok {
			if mappedStr, ok := mapped.(string); ok {
				upstreamModel = mappedStr
			}
		}
	}

	// 创建请求副本并修改模型
	reqCopy := *req
	reqCopy.Model = upstreamModel

	reqBody, err := json.Marshal(reqCopy)
	if err != nil {
		return nil, err
	}

	upstreamURL, err := buildOpenAIURL(channel.BaseURL, "/chat/completions")
	if err != nil {
		return nil, err
	}

	upstreamReq, err := http.NewRequest(http.MethodPost, upstreamURL, bytes.NewReader(reqBody))
	if err != nil {
		return nil, err
	}

	upstreamReq.Header.Set("Content-Type", "application/json")
	upstreamReq.Header.Set("Authorization", "Bearer "+channel.APIKey)

	return upstreamReq, nil
}

// calculateCost 计算用户费用
func (s *gatewayService) calculateCost(usage *Usage, pricing *model.ModelPricing) decimal.Decimal {
	if pricing == nil {
		return decimal.Zero
	}

	// 计算输入费用
	inputCost := pricing.InputPrice.Mul(decimal.NewFromInt(int64(usage.PromptTokens))).Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	// 计算输出费用
	outputCost := pricing.OutputPrice.Mul(decimal.NewFromInt(int64(usage.CompletionTokens))).Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	return inputCost.Add(outputCost)
}

// calculateUpstreamCost 计算上游成本
func (s *gatewayService) calculateUpstreamCost(userCost decimal.Decimal, channel *model.Channel, modelID string) decimal.Decimal {
	if userCost.IsZero() {
		return decimal.Zero
	}

	// 默认上游成本比例（无 channel_models 配置时使用）
	defaultUpstreamCostRatio := decimal.NewFromInt(7).Div(decimal.NewFromInt(10)) // 0.7

	// 获取渠道模型映射信息
	channelModel, err := s.channelRepo.GetChannelModel(channel.ID, modelID)
	if err != nil || channelModel == nil {
		return userCost.Mul(defaultUpstreamCostRatio)
	}

	return userCost.Mul(channelModel.CostRatio)
}

// estimatePromptTokens 估算 Prompt Token 数
func (s *gatewayService) estimatePromptTokens(req *ChatCompletionRequest) int {
	// 简单估算：每个字符约 0.25 个 Token
	totalChars := 0
	for _, msg := range req.Messages {
		if content, ok := msg.Content.(string); ok {
			totalChars += len(content)
		}
	}
	return totalChars / 4
}

// estimateCost 估算聊天完成请求成本
func (s *gatewayService) estimateCost(req *ChatCompletionRequest, pricing *model.ModelPricing) decimal.Decimal {
	// 估算输入 Token 数
	promptTokens := s.estimatePromptTokens(req)

	// 估算输出 Token 数（默认 100）
	completionTokens := 100
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		completionTokens = *req.MaxTokens
	}

	// 计算输入费用
	inputCost := pricing.InputPrice.Mul(decimal.NewFromInt(int64(promptTokens))).Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	// 计算输出费用
	outputCost := pricing.OutputPrice.Mul(decimal.NewFromInt(int64(completionTokens))).Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	return inputCost.Add(outputCost)
}

// estimateCompletionCost 估算完成请求成本
func (s *gatewayService) estimateCompletionCost(req *CompletionRequest, pricing *model.ModelPricing) decimal.Decimal {
	// 估算输入 Token 数
	promptTokens := 100 // 默认估算
	if prompt, ok := req.Prompt.(string); ok {
		promptTokens = len(prompt) / 4
	}

	// 估算输出 Token 数（默认 100）
	completionTokens := 100
	if req.MaxTokens != nil && *req.MaxTokens > 0 {
		completionTokens = *req.MaxTokens
	}

	// 计算输入费用
	inputCost := pricing.InputPrice.Mul(decimal.NewFromInt(int64(promptTokens))).Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	// 计算输出费用
	outputCost := pricing.OutputPrice.Mul(decimal.NewFromInt(int64(completionTokens))).Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	return inputCost.Add(outputCost)
}

// estimateEmbeddingCost 估算嵌入请求成本
func (s *gatewayService) estimateEmbeddingCost(req *EmbeddingRequest, pricing *model.ModelPricing) decimal.Decimal {
	// 估算输入 Token 数
	promptTokens := 100 // 默认估算
	if input, ok := req.Input.(string); ok {
		promptTokens = len(input) / 4
	} else if inputs, ok := req.Input.([]interface{}); ok {
		promptTokens = len(inputs) * 100 // 每个输入估算 100 tokens
	}

	// 计算输入费用（嵌入只有输入费用）
	inputCost := pricing.InputPrice.Mul(decimal.NewFromInt(int64(promptTokens))).Div(decimal.NewFromInt(int64(pricing.PriceUnit)))

	return inputCost
}

// logRequest 记录请求日志
func (s *gatewayService) logRequest(
	token *model.Token,
	channel *model.Channel,
	modelName string,
	usage *Usage,
	cost, upstreamCost decimal.Decimal,
	latency int,
	status model.LogStatus,
	errMsg string,
) {
	log := &model.Log{
		ID:               uuid.New(),
		UserID:           token.UserID,
		TokenID:          token.ID,
		ChannelID:        channel.ID,
		Model:            modelName,
		PromptTokens:     usage.PromptTokens,
		CompletionTokens: usage.CompletionTokens,
		TotalCost:        cost,
		UpstreamCost:     upstreamCost,
		Duration:         latency,
		Status:           status,
		ErrorMessage:     errMsg,
	}

	// 异步记录日志
	go func() {
		_ = s.logRepo.Create(log)
	}()
}
