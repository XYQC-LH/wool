package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/shopspring/decimal"
)

// CascadeResult 绾ц仈鎵ц缁撴灉
type CascadeResult struct {
	Success         bool                 `json:"success"`
	Provider        *model.ModelProvider `json:"provider,omitempty"`
	Response        interface{}          `json:"response,omitempty"`
	Error           error                `json:"error,omitempty"`
	AttemptCount    int                  `json:"attempt_count"`
	TotalLatencyMs  int64                `json:"total_latency_ms"`
	UsedProviders   []uint               `json:"used_providers"`
	FailedProviders []FailedAttempt      `json:"failed_providers,omitempty"`
}

// FailedAttempt 澶辫触灏濊瘯璁板綍
type FailedAttempt struct {
	ProviderID   uint   `json:"provider_id"`
	ProviderName string `json:"provider_name,omitempty"`
	Error        string `json:"error"`
	LatencyMs    int64  `json:"latency_ms"`
}

// CascadeController 绾ц仈鎺у埗鍣ㄦ帴鍙?type CascadeController interface {
// Execute 鎵ц绾ц仈璇锋眰
type CascadeController interface {
	Execute(ctx context.Context, operation string, modelID string, executor RequestExecutor) (*CascadeResult, error)
	// ExecuteWithStrategy 浣跨敤鎸囧畾绛栫暐鎵ц绾ц仈璇锋眰
	ExecuteWithStrategy(ctx context.Context, operation string, modelID string, strategy SelectionStrategy, executor RequestExecutor) (*CascadeResult, error)
	ExecuteOnProvider(ctx context.Context, operation string, modelID string, provider *model.ModelProvider, executor RequestExecutor) (*CascadeResult, error)
	// ExecuteStreamWithFailover 鎵ц娴佸紡璇锋眰骞舵敮鎸侀鍖呰秴鏃?failover
	ExecuteStreamWithFailover(ctx context.Context, operation string, modelID string, strategy SelectionStrategy, executor StreamExecutor) (*CascadeResult, error)
	// GetMaxRetries 鑾峰彇鏈€澶ч噸璇曟鏁?	GetMaxRetries() int
	// SetMaxRetries 璁剧疆鏈€澶ч噸璇曟鏁?
	GetMaxRetries() int
	SetMaxRetries(retries int)
}

// CascadeConfig 绾ц仈閰嶇疆
type CascadeConfig struct {
	MaxRetries           int           // 鏈€澶ч噸璇曟鏁?
	RetryDelay           time.Duration // 閲嶈瘯寤惰繜
	Timeout              time.Duration // 鍗曟璇锋眰瓒呮椂
	EnableCircuitBreaker bool          // 鏄惁鍚敤鐔旀柇鍣?
}

// DefaultCascadeConfig 榛樿绾ц仈閰嶇疆
func DefaultCascadeConfig() *CascadeConfig {
	return &CascadeConfig{
		// 0 琛ㄧず灏濊瘯鎵€鏈夊彲鐢ㄦ簮澶达紙鎴愬姛浼樺厛锛?		MaxRetries:        0,
		RetryDelay:           100 * time.Millisecond,
		Timeout:              5 * time.Minute,
		EnableCircuitBreaker: true,
	}
}

type dispatchRequestIDContextKey struct{}

// WithDispatchRequestID 将调度请求ID注入上下文（用于 dispatch_attempts 审计）
func WithDispatchRequestID(ctx context.Context, requestID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if requestID == "" {
		return ctx
	}
	return context.WithValue(ctx, dispatchRequestIDContextKey{}, requestID)
}

func dispatchRequestIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value := ctx.Value(dispatchRequestIDContextKey{})
	requestID, _ := value.(string)
	return requestID
}

// cascadeController 绾ц仈鎺у埗鍣ㄥ疄鐜?
type cascadeController struct {
	selector              ProviderSelector
	circuitBreaker        CircuitBreaker
	instanceScheduler     InstanceScheduler
	instanceRepo          repository.ProviderInstanceRepository
	commitGuard           CommitGuard     // ⭐ CommitGuard：提交点门控（现阶段复用 StreamGuard 首包锁定）
	modelAggregator       ModelAggregator // 猸?鏂板锛氭ā鍨嬭仛鍚堝櫒
	capabilityRepo        repository.ModelCapabilityRepository
	capabilityMatcher     CapabilityMatcher
	routeResolver         RouteResolver
	sourceAdapterRegistry SourceAdapterRegistry
	dispatchAttemptRepo repository.DispatchAttemptRepository
	rateLimiter          ProviderRateLimiter
	providerRepo         repository.ModelProviderRepository
	metricsRepo          repository.ProviderMetricsRepository
	config               *CascadeConfig
	mu                   sync.RWMutex
}

// NewCascadeController 鍒涘缓绾ц仈鎺у埗鍣?
func NewCascadeController(
	selector ProviderSelector,
	circuitBreaker CircuitBreaker,
	providerRepo repository.ModelProviderRepository,
	metricsRepo repository.ProviderMetricsRepository,
	instanceScheduler InstanceScheduler,
	instanceRepo repository.ProviderInstanceRepository,
	commitGuard CommitGuard, // ⭐ CommitGuard
	modelAggregator ModelAggregator, // 猸?鏂板鍙傛暟
	capabilityRepo repository.ModelCapabilityRepository,
	capabilityMatcher CapabilityMatcher,
	routeResolver RouteResolver,
	dispatchAttemptRepo repository.DispatchAttemptRepository,
	rateLimiter ProviderRateLimiter,
	config *CascadeConfig,
	sourceAdapterRegistries ...SourceAdapterRegistry,
) CascadeController {
	if config == nil {
		config = DefaultCascadeConfig()
	}
	var sourceAdapterRegistry SourceAdapterRegistry
	if len(sourceAdapterRegistries) > 0 {
		sourceAdapterRegistry = sourceAdapterRegistries[0]
	}
	return &cascadeController{
		selector:              selector,
		circuitBreaker:        circuitBreaker,
		instanceScheduler:     instanceScheduler,
		instanceRepo:          instanceRepo,
		commitGuard:           commitGuard,     // ⭐ CommitGuard
		modelAggregator:       modelAggregator, // 猸?鏂板
		capabilityRepo:        capabilityRepo,
		capabilityMatcher:     capabilityMatcher,
		routeResolver:         routeResolver,
		sourceAdapterRegistry: sourceAdapterRegistry,
		dispatchAttemptRepo: dispatchAttemptRepo,
		rateLimiter:          rateLimiter,
		providerRepo:         providerRepo,
		metricsRepo:          metricsRepo,
		config:               config,
	}
}

// Execute 鎵ц绾ц仈璇锋眰锛堜娇鐢ㄩ粯璁ゆ垚鏈紭鍏堢瓥鐣ワ級
func (c *cascadeController) Execute(ctx context.Context, operation string, modelID string, executor RequestExecutor) (*CascadeResult, error) {
	return c.ExecuteWithStrategy(ctx, operation, modelID, StrategyCostFirst, executor)
}

// ExecuteWithStrategy 浣跨敤鎸囧畾绛栫暐鎵ц绾ц仈璇锋眰
// 猸?鏍稿績鏂规硶锛氬疄鐜扮骇鑱?failover 閫昏緫
func (c *cascadeController) ExecuteWithStrategy(ctx context.Context, operation string, modelID string, strategy SelectionStrategy, executor RequestExecutor) (*CascadeResult, error) {
	operation = model.NormalizeOperation(operation)
	startTime := time.Now()

	resolvedModelID, providers, err := c.buildProvidersForDispatch(ctx, operation, modelID, strategy)
	if err != nil {
		return nil, err
	}
	if len(providers) == 0 {
		return nil, &NoAvailableProviderError{Operation: operation, ModelID: modelID}
	}

	result := &CascadeResult{
		UsedProviders:   make([]uint, 0),
		FailedProviders: make([]FailedAttempt, 0),
	}

	c.mu.RLock()
	maxRetries := c.config.MaxRetries
	retryDelay := c.config.RetryDelay
	enableCircuitBreaker := c.config.EnableCircuitBreaker
	c.mu.RUnlock()

	// 闄愬埗閲嶈瘯娆℃暟涓嶈秴杩囧彲鐢ㄦ簮澶存暟
	if maxRetries <= 0 || maxRetries > len(providers) {
		maxRetries = len(providers)
	}

	var lastError error

	// 绾ц仈灏濊瘯姣忎釜婧愬ご
	for attempt := 0; attempt < maxRetries && attempt < len(providers); attempt++ {
		provider := providers[attempt]
		result.UsedProviders = append(result.UsedProviders, provider.ID)
		result.AttemptCount++

		// 妫€鏌ョ啍鏂櫒鐘舵€?
		if enableCircuitBreaker && c.circuitBreaker != nil {
			canExecute, err := c.circuitBreaker.CanExecute(ctx, provider.ID)
			if err != nil {
				// 鐔旀柇鍣ㄦ鏌ュけ璐ワ紝璁板綍浣嗙户缁皾璇?
				lastError = fmt.Errorf("鐔旀柇鍣ㄦ鏌ュけ璐? %w", err)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID: provider.ID,
					Error:      lastError.Error(),
					LatencyMs:  0,
				})
				c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
					RequestID:         dispatchRequestIDFromContext(ctx),
					Operation:         operation,
					RequestedModelID:  modelID,
					ResolvedModelID:   resolvedModelID,
					RouteModelID:      provider.ModelID,
					ProviderID:        provider.ID,
					AttemptNo:         attempt + 1,
					Strategy:          string(strategy),
					Stage:             "cascade",
					Decision:          "rejected",
					Success:           false,
					ErrorType:         "circuit_check_error",
					ErrorMessage:      lastError.Error(),
					LatencyMs:         0,
				})
				continue
			}
			if !canExecute {
				// 鐔旀柇鍣ㄦ墦寮€锛岃烦杩囨婧愬ご
				lastError = fmt.Errorf("婧愬ご %d 鐔旀柇鍣ㄦ墦寮€", provider.ID)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID: provider.ID,
					Error:      "鐔旀柇鍣ㄦ墦寮€",
					LatencyMs:  0,
				})
				c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
					RequestID:         dispatchRequestIDFromContext(ctx),
					Operation:         operation,
					RequestedModelID:  modelID,
					ResolvedModelID:   resolvedModelID,
					RouteModelID:      provider.ModelID,
					ProviderID:        provider.ID,
					AttemptNo:         attempt + 1,
					Strategy:          string(strategy),
					Stage:             "health_gate",
					Decision:          "rejected",
					Success:           false,
					ErrorType:         "circuit_open",
					ErrorMessage:      "circuit_open",
					LatencyMs:         0,
				})
				continue
			}
		}

		if c.rateLimiter != nil {
			allowed, err := c.rateLimiter.Allow(ctx, provider.ID, operation)
			if err == nil && !allowed {
				lastError = &ProviderRateLimitedError{ProviderID: provider.ID, Operation: operation}
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID:   provider.ID,
					ProviderName: c.getProviderName(provider),
					Error:        "rate_limit",
					LatencyMs:    0,
				})
				c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
					RequestID:         dispatchRequestIDFromContext(ctx),
					Operation:         operation,
					RequestedModelID:  modelID,
					ResolvedModelID:   resolvedModelID,
					RouteModelID:      provider.ModelID,
					ProviderID:        provider.ID,
					AttemptNo:         attempt + 1,
					Strategy:          string(strategy),
					Stage:             "health_gate",
					Decision:          "rejected",
					Success:           false,
					ErrorType:         "rate_limit",
					ErrorMessage:      "rate_limit",
					LatencyMs:         0,
				})
				continue
			}
		}

		// 鎵ц璇锋眰锛堜娇鐢↖nstanceScheduler閫夋嫨瀹炰緥锛?
		attemptStart := time.Now()
		response, err := c.executeWithInstanceSelection(ctx, provider, operation, executor)
		attemptLatency := time.Since(attemptStart).Milliseconds()

		if err != nil {
			// 璇锋眰澶辫触
			lastError = err
			result.FailedProviders = append(result.FailedProviders, FailedAttempt{
				ProviderID:   provider.ID,
				ProviderName: c.getProviderName(provider),
				Error:        err.Error(),
				LatencyMs:    attemptLatency,
			})

			// 璁板綍澶辫触鍒扮啍鏂櫒
			if enableCircuitBreaker && c.circuitBreaker != nil {
				_ = c.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			}

			// 鏇存柊婧愬ご缁熻
			_ = c.providerRepo.IncrementStats(ctx, provider.ID, false, attemptLatency, 0, 0, decimal.Zero)
			c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
				RequestID:         dispatchRequestIDFromContext(ctx),
				Operation:         operation,
				RequestedModelID:  modelID,
				ResolvedModelID:   resolvedModelID,
				RouteModelID:      provider.ModelID,
				ProviderID:        provider.ID,
				AttemptNo:         attempt + 1,
				Strategy:          string(strategy),
				Stage:             "cascade",
				Decision:          "failed",
				Success:           false,
				ErrorType:         "execute_error",
				ErrorMessage:      err.Error(),
				LatencyMs:         attemptLatency,
			})

			// 閲嶈瘯寤惰繜
			if attempt < maxRetries-1 && retryDelay > 0 {
				time.Sleep(retryDelay)
			}

			continue
		}

		// 璇锋眰鎴愬姛
		result.Success = true
		result.Provider = provider
		result.Response = response
		result.TotalLatencyMs = time.Since(startTime).Milliseconds()

		// 璁板綍鎴愬姛鍒扮啍鏂櫒
		if enableCircuitBreaker && c.circuitBreaker != nil {
			_ = c.circuitBreaker.RecordSuccess(ctx, provider.ID)
		}
		c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
			RequestID:         dispatchRequestIDFromContext(ctx),
			Operation:         operation,
			RequestedModelID:  modelID,
			ResolvedModelID:   resolvedModelID,
			RouteModelID:      provider.ModelID,
			ProviderID:        provider.ID,
			AttemptNo:         attempt + 1,
			Strategy:          string(strategy),
			Stage:             "cascade",
			Decision:          "succeeded",
			Success:           true,
			LatencyMs:         attemptLatency,
		})

		return result, nil
	}

	// 鎵€鏈夊皾璇曢兘澶辫触
	result.TotalLatencyMs = time.Since(startTime).Milliseconds()
	result.Error = lastError

	return result, fmt.Errorf("鎵€鏈夋簮澶村潎澶辫触锛屾渶鍚庨敊璇? %w", lastError)
}

func (c *cascadeController) ExecuteOnProvider(ctx context.Context, operation string, modelID string, provider *model.ModelProvider, executor RequestExecutor) (*CascadeResult, error) {
	operation = model.NormalizeOperation(operation)
	startTime := time.Now()

	if provider == nil {
		return nil, fmt.Errorf("provider 不能为空")
	}

	resolvedModelID, err := c.modelAggregator.ResolveModelAlias(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("解析模型别名失败: %w", err)
	}
	if err := c.ensureModelCapability(ctx, resolvedModelID, operation); err != nil {
		return nil, err
	}

	result := &CascadeResult{
		UsedProviders:   []uint{provider.ID},
		FailedProviders: make([]FailedAttempt, 0),
		AttemptCount:    1,
	}

	c.mu.RLock()
	enableCircuitBreaker := c.config.EnableCircuitBreaker
	c.mu.RUnlock()

	if enableCircuitBreaker && c.circuitBreaker != nil {
		canExecute, err := c.circuitBreaker.CanExecute(ctx, provider.ID)
		if err != nil {
			c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
				RequestID:         dispatchRequestIDFromContext(ctx),
				Operation:         operation,
				RequestedModelID:  modelID,
				ResolvedModelID:   resolvedModelID,
				RouteModelID:      provider.ModelID,
				ProviderID:        provider.ID,
				AttemptNo:         1,
				Strategy:          string(StrategyCostFirst),
				Stage:             "health_gate",
				Decision:          "rejected",
				Success:           false,
				ErrorType:         "circuit_check_error",
				ErrorMessage:      err.Error(),
			})
			return nil, fmt.Errorf("熔断器检查失败: %w", err)
		}
		if !canExecute {
			c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
				RequestID:         dispatchRequestIDFromContext(ctx),
				Operation:         operation,
				RequestedModelID:  modelID,
				ResolvedModelID:   resolvedModelID,
				RouteModelID:      provider.ModelID,
				ProviderID:        provider.ID,
				AttemptNo:         1,
				Strategy:          string(StrategyCostFirst),
				Stage:             "health_gate",
				Decision:          "rejected",
				Success:           false,
				ErrorType:         "circuit_open",
				ErrorMessage:      "circuit_open",
			})
			return nil, fmt.Errorf("源头 %d 熔断器打开", provider.ID)
		}
	}

	if c.rateLimiter != nil {
		allowed, err := c.rateLimiter.Allow(ctx, provider.ID, operation)
		if err == nil && !allowed {
			c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
				RequestID:         dispatchRequestIDFromContext(ctx),
				Operation:         operation,
				RequestedModelID:  modelID,
				ResolvedModelID:   resolvedModelID,
				RouteModelID:      provider.ModelID,
				ProviderID:        provider.ID,
				AttemptNo:         1,
				Strategy:          string(StrategyCostFirst),
				Stage:             "health_gate",
				Decision:          "rejected",
				Success:           false,
				ErrorType:         "rate_limit",
				ErrorMessage:      "rate_limit",
			})
			return nil, &ProviderRateLimitedError{ProviderID: provider.ID, Operation: operation}
		}
	}

	attemptStart := time.Now()
	response, err := c.executeWithInstanceSelection(ctx, provider, operation, executor)
	attemptLatency := time.Since(attemptStart).Milliseconds()
	result.TotalLatencyMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Success = false
		result.Error = err
		result.FailedProviders = append(result.FailedProviders, FailedAttempt{
			ProviderID:   provider.ID,
			ProviderName: c.getProviderName(provider),
			Error:        err.Error(),
			LatencyMs:    attemptLatency,
		})

		if enableCircuitBreaker && c.circuitBreaker != nil {
			_ = c.circuitBreaker.RecordFailure(ctx, provider.ID, err)
		}
		_ = c.providerRepo.IncrementStats(ctx, provider.ID, false, attemptLatency, 0, 0, decimal.Zero)
		c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
			RequestID:         dispatchRequestIDFromContext(ctx),
			Operation:         operation,
			RequestedModelID:  modelID,
			ResolvedModelID:   resolvedModelID,
			RouteModelID:      provider.ModelID,
			ProviderID:        provider.ID,
			AttemptNo:         1,
			Strategy:          string(StrategyCostFirst),
			Stage:             "cascade",
			Decision:          "failed",
			Success:           false,
			ErrorType:         "execute_error",
			ErrorMessage:      err.Error(),
			LatencyMs:         attemptLatency,
		})
		return result, err
	}

	result.Success = true
	result.Provider = provider
	result.Response = response

	if enableCircuitBreaker && c.circuitBreaker != nil {
		_ = c.circuitBreaker.RecordSuccess(ctx, provider.ID)
	}
	c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
		RequestID:         dispatchRequestIDFromContext(ctx),
		Operation:         operation,
		RequestedModelID:  modelID,
		ResolvedModelID:   resolvedModelID,
		RouteModelID:      provider.ModelID,
		ProviderID:        provider.ID,
		AttemptNo:         1,
		Strategy:          string(StrategyCostFirst),
		Stage:             "cascade",
		Decision:          "succeeded",
		Success:           true,
		LatencyMs:         attemptLatency,
	})

	return result, nil
}

// executeWithInstanceSelection 甯﹀疄渚嬮€夋嫨鐨勮姹傛墽琛?
func (c *cascadeController) executeWithInstanceSelection(ctx context.Context, provider *model.ModelProvider, operation string, executor RequestExecutor) (interface{}, error) {
	c.mu.RLock()
	timeout := c.config.Timeout
	c.mu.RUnlock()

	// 鍒涘缓甯﹁秴鏃剁殑涓婁笅鏂?
	timeoutCtx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	// 浣跨敤 channel 鎺ユ敹缁撴灉
	type result struct {
		response interface{}
		err      error
	}

	resultCh := make(chan result, 1)

	go func() {
		c.attachSelectedInstance(timeoutCtx, provider, operation)

		// 鎵ц璇锋眰
		response, err := c.executeRequestViaAdapter(timeoutCtx, operation, provider, executor)
		resultCh <- result{response: response, err: err}
	}()

	select {
	case <-timeoutCtx.Done():
		if timeoutCtx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("璇锋眰瓒呮椂")
		}
		return nil, timeoutCtx.Err()
	case r := <-resultCh:
		return r.response, r.err
	}
}

func (c *cascadeController) executeRequestViaAdapter(ctx context.Context, operation string, provider *model.ModelProvider, executor RequestExecutor) (interface{}, error) {
	if executor == nil {
		return nil, fmt.Errorf("RequestExecutor 不能为空")
	}
	if c == nil || c.sourceAdapterRegistry == nil {
		return executor(ctx, provider)
	}

	adapter := c.sourceAdapterRegistry.Resolve(operation, provider)
	if adapter == nil {
		return executor(ctx, provider)
	}

	return adapter.Execute(ctx, &AdapterRequest{
		Operation: operation,
		Provider:  provider,
		Executor:  executor,
	})
}

// getProviderName 鑾峰彇婧愬ご鍚嶇О
func (c *cascadeController) getProviderName(provider *model.ModelProvider) string {
	if provider.Channel != nil {
		return provider.Channel.Name
	}
	return fmt.Sprintf("Provider-%d", provider.ID)
}

func (c *cascadeController) buildProvidersForDispatch(ctx context.Context, operation string, requestModelID string, strategy SelectionStrategy) (string, []*model.ModelProvider, error) {
	if ctx == nil {
		ctx = context.Background()
	}

	resolvedModelID := requestModelID
	var err error
	if c.modelAggregator != nil {
		resolvedModelID, err = c.modelAggregator.ResolveModelAlias(ctx, requestModelID)
		if err != nil {
			return "", nil, fmt.Errorf("解析模型别名失败: %w", err)
		}
	}
	if err := c.ensureModelCapability(ctx, resolvedModelID, operation); err != nil {
		return "", nil, err
	}

	routeModels, err := c.resolveRouteModels(ctx, operation, resolvedModelID)
	if err != nil {
		return "", nil, err
	}
	if c.providerRepo == nil {
		return "", nil, fmt.Errorf("provider repository 未配置")
	}

	merged := make([]*model.ModelProvider, 0)
	seenProviders := make(map[uint]struct{})
	totalConstraintReject := 0
	totalSourceCandidates := 0

	for _, routeModelID := range routeModels {
		providers, err := c.providerRepo.GetByModelID(ctx, operation, routeModelID)
		if err != nil {
			return "", nil, fmt.Errorf("获取路由源头失败(model=%s): %w", routeModelID, err)
		}
		if len(providers) == 0 {
			continue
		}

		totalSourceCandidates += len(providers)
		matchedProviders := providers
		if c.capabilityMatcher != nil {
			filtered, rejected, err := c.capabilityMatcher.MatchProviders(ctx, operation, providers)
			if err != nil {
				return "", nil, fmt.Errorf("能力约束匹配失败(model=%s): %w", routeModelID, err)
			}
			totalConstraintReject += len(rejected)
			matchedProviders = filtered
		}
		if len(matchedProviders) == 0 {
			continue
		}

		healthyProviders := c.applyHealthGate(matchedProviders)
		if len(healthyProviders) == 0 {
			continue
		}

		ranked := c.rankProviders(operation, strategy, healthyProviders)
		for _, provider := range ranked {
			if provider == nil {
				continue
			}
			if _, exists := seenProviders[provider.ID]; exists {
				continue
			}
			seenProviders[provider.ID] = struct{}{}
			merged = append(merged, provider)
		}
	}

	if len(merged) == 0 && totalSourceCandidates > 0 && totalConstraintReject > 0 && totalConstraintReject >= totalSourceCandidates {
		return resolvedModelID, nil, &ModelOperationNotSupportedError{
			Operation: operation,
			ModelID:   resolvedModelID,
		}
	}

	return resolvedModelID, merged, nil
}

func (c *cascadeController) resolveRouteModels(ctx context.Context, operation string, resolvedModelID string) ([]string, error) {
	if c.routeResolver == nil {
		return []string{resolvedModelID}, nil
	}

	routeModels, err := c.routeResolver.ResolveRouteModels(ctx, operation, resolvedModelID)
	if err != nil {
		return nil, fmt.Errorf("解析路由失败: %w", err)
	}
	if len(routeModels) == 0 {
		return []string{resolvedModelID}, nil
	}
	return routeModels, nil
}

func (c *cascadeController) applyHealthGate(providers []*model.ModelProvider) []*model.ModelProvider {
	if len(providers) == 0 {
		return nil
	}

	now := time.Now()
	filtered := make([]*model.ModelProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		if provider.Status != model.ModelProviderStatusActive {
			continue
		}
		if provider.CircuitState == model.CircuitStateOpen {
			if provider.CircuitOpenUntil == nil || now.Before(*provider.CircuitOpenUntil) {
				continue
			}
		}
		filtered = append(filtered, provider)
	}

	return filtered
}

func (c *cascadeController) rankProviders(operation string, strategy SelectionStrategy, providers []*model.ModelProvider) []*model.ModelProvider {
	if len(providers) <= 1 {
		result := make([]*model.ModelProvider, len(providers))
		copy(result, providers)
		return result
	}

	ranked := make([]*model.ModelProvider, len(providers))
	copy(ranked, providers)

	selector, ok := c.selector.(*providerSelector)
	if !ok {
		return ranked
	}

	switch strategy {
	case StrategyLatencyFirst:
		selector.sortByLatency(ranked)
	case StrategyHealthFirst:
		selector.sortByHealth(ranked)
	case StrategyRoundRobin:
		selector.applyRoundRobin(operation, ranked)
	case StrategyWeighted:
		selector.applyWeighted(ranked)
	default:
		selector.sortByCost(context.Background(), operation, ranked)
	}

	return ranked
}

func (c *cascadeController) recordDispatchAttempt(ctx context.Context, attempt *model.DispatchAttempt) {
	if c == nil || c.dispatchAttemptRepo == nil || attempt == nil {
		return
	}
	_ = c.dispatchAttemptRepo.Create(ctx, attempt)
}

func (c *cascadeController) ensureModelCapability(ctx context.Context, modelID string, operation string) error {
	if c == nil || c.capabilityRepo == nil {
		return nil
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if modelID == "" || operation == "" {
		return nil
	}

	capability, err := c.capabilityRepo.GetByModelAndOperation(ctx, modelID, operation)
	if err != nil {
		return fmt.Errorf("获取模型能力失败: %w", err)
	}
	if capability == nil {
		// 向后兼容：未配置能力记录时不拦截
		return nil
	}
	if !capability.Enabled {
		return &ModelOperationNotSupportedError{
			Operation: operation,
			ModelID:   modelID,
		}
	}
	return nil
}

// GetMaxRetries 鑾峰彇鏈€澶ч噸璇曟鏁?
func (c *cascadeController) GetMaxRetries() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.config.MaxRetries
}

// SetMaxRetries 璁剧疆鏈€澶ч噸璇曟鏁?
func (c *cascadeController) SetMaxRetries(retries int) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if retries >= 0 {
		c.config.MaxRetries = retries
	}
}

/*
// ExecuteStreamWithFailover 鎵ц娴佸紡璇锋眰骞舵敮鎸侀鍖呰秴鏃?failover
// 猸?鏍稿績鏂规硶锛氬疄鐜版祦寮忚姹傜殑棣栧寘瓒呮椂 failover 鏈哄埗
// 鏍规嵁鏋舵瀯璁捐锛?
// - 棣栧寘鍒拌揪鍓嶏細鍏佽 failover 鍒颁笅涓€涓簮澶?
// - 棣栧寘鍒拌揪鍚庯細閿佸畾褰撳墠婧愬ご锛屼笉鍐嶅垏鎹?
func (c *cascadeController) ExecuteStreamWithFailover(ctx context.Context, modelID string, strategy SelectionStrategy, executor StreamExecutor) (*CascadeResult, error) {
	startTime := time.Now()

	// 猸?姝ラ2: ModelAggregator - 瑙ｆ瀽妯″瀷鍒悕骞惰幏鍙?ProviderGroup 鍒楄〃
	resolvedModelID, err := c.modelAggregator.ResolveModelAlias(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("瑙ｆ瀽妯″瀷鍒悕澶辫触: %w", err)
	}

	// 鑾峰彇鎺掑簭鍚庣殑婧愬ご鍒楄〃
	providers, err := c.selector.SelectProviders(ctx, operation, resolvedModelID, strategy)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇婧愬ご鍒楄〃澶辫触: %w", err)
	}

	if len(providers) == 0 {
		return nil, fmt.Errorf("娌℃湁鍙敤鐨勬簮澶存敮鎸佹ā鍨? %s", modelID)
	}

	result := &CascadeResult{
		UsedProviders:   make([]uint, 0),
		FailedProviders: make([]FailedAttempt, 0),
	}

	c.mu.RLock()
	maxRetries := c.config.MaxRetries
	retryDelay := c.config.RetryDelay
	enableCircuitBreaker := c.config.EnableCircuitBreaker
	c.mu.RUnlock()

	// 闄愬埗閲嶈瘯娆℃暟涓嶈秴杩囧彲鐢ㄦ簮澶存暟
	if maxRetries > len(providers) {
		maxRetries = len(providers)
	}

	var lastError error

	// 绾ц仈灏濊瘯姣忎釜婧愬ご
	for attempt := 0; attempt < maxRetries && attempt < len(providers); attempt++ {
		provider := providers[attempt]
		result.UsedProviders = append(result.UsedProviders, provider.ID)
		result.AttemptCount++

		// 妫€鏌ョ啍鏂櫒鐘舵€?
		if enableCircuitBreaker {
			canExecute, err := c.circuitBreaker.CanExecute(ctx, provider.ID)
			if err != nil {
				// 鐔旀柇鍣ㄦ鏌ュけ璐ワ紝璁板綍浣嗙户缁皾璇?
				lastError = fmt.Errorf("鐔旀柇鍣ㄦ鏌ュけ璐? %w", err)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID: provider.ID,
					Error:      lastError.Error(),
					LatencyMs:  0,
				})
				continue
			}
			if !canExecute {
				// 鐔旀柇鍣ㄦ墦寮€锛岃烦杩囨婧愬ご
				lastError = fmt.Errorf("婧愬ご %d 鐔旀柇鍣ㄦ墦寮€", provider.ID)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID: provider.ID,
					Error:      "鐔旀柇鍣ㄦ墦寮€",
					LatencyMs:  0,
				})
				continue
			}
		}

		// 猸?浣跨敤 StreamGuard 鎵ц娴佸紡璇锋眰锛屽甫棣栧寘瓒呮椂淇濇姢
		attemptStart := time.Now()
		canFailover, err := c.executeStreamWithGuard(ctx, provider, operation, executor)
		attemptLatency := time.Since(attemptStart).Milliseconds()

		if err != nil {
			// 璇锋眰澶辫触
			lastError = err
			result.FailedProviders = append(result.FailedProviders, FailedAttempt{
				ProviderID:   provider.ID,
				ProviderName: c.getProviderName(provider),
				Error:        err.Error(),
				LatencyMs:    attemptLatency,
			})

			// 璁板綍澶辫触鍒扮啍鏂櫒
			if enableCircuitBreaker {
				_ = c.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			}

			// 鏇存柊婧愬ご缁熻
			_ = c.providerRepo.IncrementStats(ctx, provider.ID, false, attemptLatency, 0, 0, decimal.Zero)

			// 閲嶈瘯寤惰繜
			if attempt < maxRetries-1 && retryDelay > 0 {
				time.Sleep(retryDelay)
			}

			continue
		}

		// 璇锋眰鎴愬姛
		result.Success = true
		result.Provider = provider
		result.Response = response
		result.TotalLatencyMs = time.Since(startTime).Milliseconds()

		// 璁板綍鎴愬姛鍒扮啍鏂櫒
		if enableCircuitBreaker {
			_ = c.circuitBreaker.RecordSuccess(ctx, provider.ID)
		}

		return result, nil
	}

	// 鎵€鏈夊皾璇曢兘澶辫触
	result.TotalLatencyMs = time.Since(startTime).Milliseconds()
	result.Error = lastError

	return result, fmt.Errorf("鎵€鏈夋簮澶村潎澶辫触锛屾渶鍚庨敊璇? %w", lastError)
}

// executeStreamWithGuard 浣跨敤 StreamGuard 鎵ц娴佸紡璇锋眰
// 猸?鏍稿績鏂规硶锛氬疄鐜伴鍖呰秴鏃朵繚鎶ゆ満鍒?
// 鏍规嵁鏋舵瀯璁捐锛?
// - 棣栧寘鍒拌揪鍓嶏細鍏佽 failover 鍒颁笅涓€涓簮澶?
// - 棣栧寘鍒拌揪鍚庯細閿佸畾褰撳墠婧愬ご锛屼笉鍐嶅垏鎹?
// - 棣栧寘瓒呮椂鏃堕棿锛?绉掞紙FirstChunkTimeout锛?
func (c *cascadeController) executeRequestWithGuard(ctx context.Context, provider *model.ModelProvider, executor RequestExecutor) (interface{}, error) {
	if c.commitGuard == nil {
		// 濡傛灉娌℃湁 StreamGuard锛岀洿鎺ユ墽琛?
		return c.executeWithInstanceSelection(ctx, provider, executor)
	}

	// 寮€濮嬫祦寮忚姹?
	streamCtx, err := c.commitGuard.StartStream(ctx, provider.ID)
	if err != nil {
		return nil, fmt.Errorf("鍚姩娴佸紡璇锋眰澶辫触: %w", err)
	}

	// 猸?淇锛氬疄鐜伴鍖呰秴鏃舵娴嬪拰 failover 閫昏緫
	// 鍒涘缓甯﹁秴鏃剁殑涓婁笅鏂囷紝鐢ㄤ簬妫€娴嬮鍖呰秴鏃?
	c.mu.RLock()
	timeout := c.config.Timeout
	c.mu.RUnlock()

	// 浣跨敤 channel 鎺ユ敹缁撴灉
	type result struct {
		response interface{}
		err      error
	}

	resultCh := make(chan result, 1)
	firstChunkCh := make(chan struct{}, 1) // 棣栧寘鍒拌揪淇″彿

	// 鍚姩 goroutine 鎵ц璇锋眰
	go func() {
		// 鎵ц璇锋眰
		response, err := c.executeWithInstanceSelection(ctx, provider, executor)

		// 棣栧瓧鑺傚埌杈惧悗閿佸畾
		if err == nil {
			_ = c.commitGuard.OnFirstChunk(ctx, streamCtx)
		}

		resultCh <- result{response: response, err: err}
	}()

	// 猸?鍚姩棣栧寘瓒呮椂妫€娴?goroutine
	go func() {
		// 绛夊緟棣栧寘鍒拌揪淇″彿锛堣繖閲岀畝鍖栧鐞嗭紝瀹為檯搴旇浠?executor 涓娴嬮鍖咃級
		// 鐢变簬 executor 鏄閮ㄤ紶鍏ョ殑锛屾垜浠棤娉曠洿鎺ユ娴嬮鍖?
		// 杩欓噷浣跨敤涓€涓畝鍖栫殑鏂规锛氱瓑寰呬竴娈垫椂闂村悗鍙戦€侀鍖呬俊鍙?
		// 瀹為檯瀹炵幇涓紝executor 搴旇鍦ㄩ鍖呭埌杈炬椂璋冪敤 OnFirstChunk

		// 鑾峰彇棣栧寘瓒呮椂閰嶇疆
		firstChunkTimeout := 3 * time.Second // 榛樿3绉?

		// 绛夊緟棣栧寘瓒呮椂
		select {
		case <-time.After(firstChunkTimeout):
			// 棣栧寘瓒呮椂锛屾鏌ユ槸鍚﹀凡缁忛攣瀹?
			locked, _ := c.commitGuard.IsLocked(ctx, streamCtx)
			if !locked {
				// 鏈攣瀹氾紝璇存槑棣栧寘鏈埌杈撅紝瑙﹀彂瓒呮椂
				// 缁撴潫娴佸紡璇锋眰
				_ = c.commitGuard.EndStream(ctx, streamCtx, false)
			}
		case <-ctx.Done():
			// 涓婁笅鏂囧彇娑?
			return
		case <-firstChunkCh:
			// 棣栧寘宸插埌杈?
			return
		}
	}()

	// 绛夊緟璇锋眰瀹屾垚鎴栬秴鏃?
	select {
	case <-ctx.Done():
		if ctx.Err() == context.DeadlineExceeded {
			// 妫€鏌ユ槸鍚﹀凡缁忛攣瀹?
			locked, _ := c.commitGuard.IsLocked(ctx, streamCtx)
			if !locked {
				// 鏈攣瀹氾紝璇存槑棣栧寘鏈埌杈撅紝杩斿洖棣栧寘瓒呮椂閿欒
				_ = c.commitGuard.EndStream(ctx, streamCtx, false)
				return nil, fmt.Errorf("棣栧寘瓒呮椂锛?绉掞級锛屽彲浠ュ垏鎹rovider")
			}
			// 宸查攣瀹氾紝璇存槑棣栧寘宸插埌杈撅紝杩斿洖鏅€氳秴鏃堕敊璇?
			_ = c.commitGuard.EndStream(ctx, streamCtx, false)
			return nil, fmt.Errorf("璇锋眰瓒呮椂")
		}
		_ = c.commitGuard.EndStream(ctx, streamCtx, false)
		return nil, ctx.Err()
	case r := <-resultCh:
		// 缁撴潫娴佸紡璇锋眰
		_ = c.commitGuard.EndStream(ctx, streamCtx, r.err == nil)
		return r.response, r.err
	}
}

*/

func (c *cascadeController) ExecuteStreamWithFailover(ctx context.Context, operation string, modelID string, strategy SelectionStrategy, executor StreamExecutor) (*CascadeResult, error) {
	operation = model.NormalizeOperation(operation)
	startTime := time.Now()

	resolvedModelID, providers, err := c.buildProvidersForDispatch(ctx, operation, modelID, strategy)
	if err != nil {
		return nil, err
	}

	if len(providers) == 0 {
		return nil, &NoAvailableProviderError{Operation: operation, ModelID: modelID}
	}

	result := &CascadeResult{
		UsedProviders:   make([]uint, 0),
		FailedProviders: make([]FailedAttempt, 0),
	}

	c.mu.RLock()
	maxRetries := c.config.MaxRetries
	retryDelay := c.config.RetryDelay
	enableCircuitBreaker := c.config.EnableCircuitBreaker
	c.mu.RUnlock()

	if maxRetries <= 0 || maxRetries > len(providers) {
		maxRetries = len(providers)
	}

	var lastError error

	for attempt := 0; attempt < maxRetries && attempt < len(providers); attempt++ {
		provider := providers[attempt]
		result.UsedProviders = append(result.UsedProviders, provider.ID)
		result.AttemptCount++

		if enableCircuitBreaker && c.circuitBreaker != nil {
			canExecute, err := c.circuitBreaker.CanExecute(ctx, provider.ID)
			if err != nil {
				lastError = fmt.Errorf("鐔旀柇鍣ㄦ鏌ュけ璐? %w", err)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID: provider.ID,
					Error:      lastError.Error(),
					LatencyMs:  0,
				})
				c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
					RequestID:        dispatchRequestIDFromContext(ctx),
					Operation:        operation,
					RequestedModelID: modelID,
					ResolvedModelID:  resolvedModelID,
					RouteModelID:     provider.ModelID,
					ProviderID:       provider.ID,
					AttemptNo:        attempt + 1,
					Strategy:         string(strategy),
					Stage:            "health_gate",
					Decision:         "rejected",
					Success:          false,
					ErrorType:        "circuit_check_error",
					ErrorMessage:     lastError.Error(),
				})
				continue
			}
			if !canExecute {
				lastError = fmt.Errorf("婧愬ご %d 鐔旀柇鍣ㄦ墦寮€", provider.ID)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID: provider.ID,
					Error:      "鐔旀柇鍣ㄦ墦寮€",
					LatencyMs:  0,
				})
				c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
					RequestID:        dispatchRequestIDFromContext(ctx),
					Operation:        operation,
					RequestedModelID: modelID,
					ResolvedModelID:  resolvedModelID,
					RouteModelID:     provider.ModelID,
					ProviderID:       provider.ID,
					AttemptNo:        attempt + 1,
					Strategy:         string(strategy),
					Stage:            "health_gate",
					Decision:         "rejected",
					Success:          false,
					ErrorType:        "circuit_open",
					ErrorMessage:     "circuit_open",
				})
				continue
			}
		}

		if c.rateLimiter != nil {
			allowed, err := c.rateLimiter.Allow(ctx, provider.ID, operation)
			if err == nil && !allowed {
				lastError = fmt.Errorf("源头 %d 达到限流阈值", provider.ID)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID:   provider.ID,
					ProviderName: c.getProviderName(provider),
					Error:        "rate_limit",
					LatencyMs:    0,
				})
				c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
					RequestID:        dispatchRequestIDFromContext(ctx),
					Operation:        operation,
					RequestedModelID: modelID,
					ResolvedModelID:  resolvedModelID,
					RouteModelID:     provider.ModelID,
					ProviderID:       provider.ID,
					AttemptNo:        attempt + 1,
					Strategy:         string(strategy),
					Stage:            "health_gate",
					Decision:         "rejected",
					Success:          false,
					ErrorType:        "rate_limit",
					ErrorMessage:     "rate_limit",
				})
				continue
			}
		}

		attemptStart := time.Now()
		canFailover, err := c.executeStreamWithGuard(ctx, provider, operation, executor)
		attemptLatency := time.Since(attemptStart).Milliseconds()

		if err != nil {
			lastError = err
			result.FailedProviders = append(result.FailedProviders, FailedAttempt{
				ProviderID:   provider.ID,
				ProviderName: c.getProviderName(provider),
				Error:        err.Error(),
				LatencyMs:    attemptLatency,
			})

			if enableCircuitBreaker && c.circuitBreaker != nil {
				_ = c.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			}
			_ = c.providerRepo.IncrementStats(ctx, provider.ID, false, attemptLatency, 0, 0, decimal.Zero)
			c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
				RequestID:        dispatchRequestIDFromContext(ctx),
				Operation:        operation,
				RequestedModelID: modelID,
				ResolvedModelID:  resolvedModelID,
				RouteModelID:     provider.ModelID,
				ProviderID:       provider.ID,
				AttemptNo:        attempt + 1,
				Strategy:         string(strategy),
				Stage:            "cascade",
				Decision:         "failed",
				Success:          false,
				ErrorType:        "execute_error",
				ErrorMessage:     err.Error(),
				LatencyMs:        attemptLatency,
			})

			if !canFailover {
				result.TotalLatencyMs = time.Since(startTime).Milliseconds()
				result.Error = lastError
				return result, err
			}

			if attempt < maxRetries-1 && retryDelay > 0 {
				time.Sleep(retryDelay)
			}
			continue
		}

		result.Success = true
		result.Provider = provider
		result.TotalLatencyMs = time.Since(startTime).Milliseconds()

		if enableCircuitBreaker && c.circuitBreaker != nil {
			_ = c.circuitBreaker.RecordSuccess(ctx, provider.ID)
		}
		c.recordDispatchAttempt(ctx, &model.DispatchAttempt{
			RequestID:        dispatchRequestIDFromContext(ctx),
			Operation:        operation,
			RequestedModelID: modelID,
			ResolvedModelID:  resolvedModelID,
			RouteModelID:     provider.ModelID,
			ProviderID:       provider.ID,
			AttemptNo:        attempt + 1,
			Strategy:         string(strategy),
			Stage:            "cascade",
			Decision:         "succeeded",
			Success:          true,
			LatencyMs:        attemptLatency,
		})

		return result, nil
	}

	result.TotalLatencyMs = time.Since(startTime).Milliseconds()
	result.Error = lastError
	return result, fmt.Errorf("鎵€鏈夋簮澶村潎澶辫触锛屾渶鍚庨敊璇? %w", lastError)
}

// executeStreamWithGuard 浣跨敤 StreamGuard 鎵ц娴佸紡璇锋眰
// 杩斿洖鍊?canFailover 琛ㄧず锛氬綋鍓嶉敊璇槸鍚﹀厑璁稿垏鎹㈠埌涓嬩竴涓?Provider锛堥鍖呮湭閿佸畾鏃舵墠鍏佽锛?func (c *cascadeController) executeStreamWithGuard(ctx context.Context, provider *model.ModelProvider, executor StreamExecutor) (canFailover bool, err error) {
func (c *cascadeController) executeStreamWithGuard(ctx context.Context, provider *model.ModelProvider, operation string, executor StreamExecutor) (canFailover bool, err error) {
	if executor == nil {
		return false, fmt.Errorf("StreamExecutor 涓嶈兘涓虹┖")
	}

	// 没有 CommitGuard 时，无法锁定语义，默认允许 failover
	if c.commitGuard == nil {
		c.attachSelectedInstance(ctx, provider, operation)
		return true, executor(ctx, provider, func() {})
	}

	c.mu.RLock()
	attemptTimeout := c.config.Timeout
	c.mu.RUnlock()
	if provider.AttemptTimeoutMs > 0 {
		attemptTimeout = time.Duration(provider.AttemptTimeoutMs) * time.Millisecond
	}

	attemptCtx, cancel := context.WithTimeout(ctx, attemptTimeout)
	defer cancel()

	streamCtx, err := c.commitGuard.StartStream(attemptCtx, provider.ID)
	if err != nil {
		return true, fmt.Errorf("鍚姩娴佸紡璇锋眰澶辫触: %w", err)
	}

	firstChunkTimeout := 3 * time.Second
	if provider.StreamFirstChunkTimeoutMs > 0 {
		firstChunkTimeout = time.Duration(provider.StreamFirstChunkTimeoutMs) * time.Millisecond
	}

	firstChunkTimer := time.NewTimer(firstChunkTimeout)
	defer firstChunkTimer.Stop()

	firstChunkOnce := sync.Once{}
	onFirstChunk := func() {
		firstChunkOnce.Do(func() {
			_ = c.commitGuard.OnFirstChunk(attemptCtx, streamCtx)
			if !firstChunkTimer.Stop() {
				select {
				case <-firstChunkTimer.C:
				default:
				}
			}
		})
	}

	errCh := make(chan error, 1)
	go func() {
		c.attachSelectedInstance(attemptCtx, provider, operation)
		errCh <- c.executeStreamViaAdapter(attemptCtx, operation, provider, executor, onFirstChunk)
	}()

	firstChunkTimeoutCh := firstChunkTimer.C
	for {
		select {
		case <-firstChunkTimeoutCh:
			locked, _ := c.commitGuard.IsLocked(attemptCtx, streamCtx)
			if !locked {
				cancel()
				_ = c.commitGuard.EndStream(context.Background(), streamCtx, false)
				return true, fmt.Errorf("首包超时（%dms），可以切换Provider", firstChunkTimeout.Milliseconds())
			}
			firstChunkTimeoutCh = nil

		case err := <-errCh:
			locked, _ := c.commitGuard.IsLocked(context.Background(), streamCtx)
			_ = c.commitGuard.EndStream(context.Background(), streamCtx, err == nil)
			if err != nil {
				return !locked, err
			}
			return false, nil

		case <-attemptCtx.Done():
			locked, _ := c.commitGuard.IsLocked(context.Background(), streamCtx)
			_ = c.commitGuard.EndStream(context.Background(), streamCtx, false)
			if attemptCtx.Err() == context.DeadlineExceeded {
				if !locked {
					return true, fmt.Errorf("璇锋眰瓒呮椂锛堥鍖呭墠锛夛紝鍙互鍒囨崲Provider")
				}
				return false, fmt.Errorf("璇锋眰瓒呮椂")
			}
			return !locked, attemptCtx.Err()
		}
	}
}

func (c *cascadeController) executeStreamViaAdapter(ctx context.Context, operation string, provider *model.ModelProvider, executor StreamExecutor, onFirstChunk func()) error {
	if executor == nil {
		return fmt.Errorf("StreamExecutor 不能为空")
	}

	// 预留 SourceAdapter 的流式扩展点：
	// 当前仍透传至既有 StreamExecutor，后续在这里接入统一流式协议适配。
	_ = operation
	return executor(ctx, provider, onFirstChunk)
}

// attachSelectedInstance 涓?provider 閫夋嫨骞剁粦瀹氬疄渚嬶紝骞剁‘淇濇墽琛岀粨鏉熷悗閲婃斁骞跺彂妲戒綅
func (c *cascadeController) attachSelectedInstance(ctx context.Context, provider *model.ModelProvider, operation string) {
	if c.instanceScheduler == nil {
		return
	}
	if provider == nil {
		return
	}

	// 若上层已指定实例（例如 Job CommitGuard 锁定），则复用该实例，禁止重选导致语义漂移
	if provider.SelectedInstance != nil && provider.SelectedInstance.ID > 0 {
		instance, err := c.instanceScheduler.AcquireInstance(ctx, provider.SelectedInstance.ID, operation)
		if err != nil || instance == nil {
			return
		}

		provider.SelectedInstance = instance
		go func(instanceID uint) {
			<-ctx.Done()
			_ = c.instanceScheduler.ReleaseInstanceSlot(context.Background(), instanceID)
		}(instance.ID)
		return
	}

	selectedInstance, err := c.instanceScheduler.SelectInstance(ctx, provider.ID, operation)
	if err != nil || selectedInstance == nil {
		return
	}

	provider.SelectedInstance = selectedInstance
	go func(instanceID uint) {
		<-ctx.Done()
		_ = c.instanceScheduler.ReleaseInstanceSlot(context.Background(), instanceID)
	}(selectedInstance.ID)
}

// CascadeStats 绾ц仈缁熻
type CascadeStats struct {
	TotalRequests       int64   `json:"total_requests"`
	SuccessRequests     int64   `json:"success_requests"`
	FailedRequests      int64   `json:"failed_requests"`
	AvgAttempts         float64 `json:"avg_attempts"`
	FirstAttemptSuccess int64   `json:"first_attempt_success"`
	RetrySuccess        int64   `json:"retry_success"`
}

// CascadeMetrics 绾ц仈鎸囨爣鏀堕泦鍣?
type CascadeMetrics struct {
	mu                  sync.Mutex
	totalRequests       int64
	successRequests     int64
	failedRequests      int64
	totalAttempts       int64
	firstAttemptSuccess int64
	retrySuccess        int64
}

// NewCascadeMetrics 鍒涘缓绾ц仈鎸囨爣鏀堕泦鍣?
func NewCascadeMetrics() *CascadeMetrics {
	return &CascadeMetrics{}
}

// RecordResult 璁板綍缁撴灉
func (m *CascadeMetrics) RecordResult(result *CascadeResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests++
	m.totalAttempts += int64(result.AttemptCount)

	if result.Success {
		m.successRequests++
		if result.AttemptCount == 1 {
			m.firstAttemptSuccess++
		} else {
			m.retrySuccess++
		}
	} else {
		m.failedRequests++
	}
}

// GetStats 鑾峰彇缁熻
func (m *CascadeMetrics) GetStats() *CascadeStats {
	m.mu.Lock()
	defer m.mu.Unlock()

	avgAttempts := float64(0)
	if m.totalRequests > 0 {
		avgAttempts = float64(m.totalAttempts) / float64(m.totalRequests)
	}

	return &CascadeStats{
		TotalRequests:       m.totalRequests,
		SuccessRequests:     m.successRequests,
		FailedRequests:      m.failedRequests,
		AvgAttempts:         avgAttempts,
		FirstAttemptSuccess: m.firstAttemptSuccess,
		RetrySuccess:        m.retrySuccess,
	}
}

// Reset 閲嶇疆缁熻
func (m *CascadeMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.totalRequests = 0
	m.successRequests = 0
	m.failedRequests = 0
	m.totalAttempts = 0
	m.firstAttemptSuccess = 0
	m.retrySuccess = 0
}
