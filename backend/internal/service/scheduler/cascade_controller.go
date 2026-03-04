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

// cascadeController 绾ц仈鎺у埗鍣ㄥ疄鐜?
type cascadeController struct {
	selector          ProviderSelector
	circuitBreaker    CircuitBreaker
	instanceScheduler InstanceScheduler
	instanceRepo      repository.ProviderInstanceRepository
	commitGuard       CommitGuard     // ⭐ CommitGuard：提交点门控（现阶段复用 StreamGuard 首包锁定）
	modelAggregator   ModelAggregator // 猸?鏂板锛氭ā鍨嬭仛鍚堝櫒
	capabilityRepo    repository.ModelCapabilityRepository
	rateLimiter       ProviderRateLimiter
	providerRepo      repository.ModelProviderRepository
	metricsRepo       repository.ProviderMetricsRepository
	config            *CascadeConfig
	mu                sync.RWMutex
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
	rateLimiter ProviderRateLimiter,
	config *CascadeConfig,
) CascadeController {
	if config == nil {
		config = DefaultCascadeConfig()
	}
	return &cascadeController{
		selector:          selector,
		circuitBreaker:    circuitBreaker,
		instanceScheduler: instanceScheduler,
		instanceRepo:      instanceRepo,
		commitGuard:       commitGuard,     // ⭐ CommitGuard
		modelAggregator:   modelAggregator, // 猸?鏂板
		capabilityRepo:    capabilityRepo,
		rateLimiter:       rateLimiter,
		providerRepo:      providerRepo,
		metricsRepo:       metricsRepo,
		config:            config,
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

	// 猸?姝ラ2: ModelAggregator - 瑙ｆ瀽妯″瀷鍒悕骞惰幏鍙?ProviderGroup 鍒楄〃
	resolvedModelID, err := c.modelAggregator.ResolveModelAlias(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("瑙ｆ瀽妯″瀷鍒悕澶辫触: %w", err)
	}

	// 鑾峰彇鎺掑簭鍚庣殑婧愬ご鍒楄〃
	if err := c.ensureModelCapability(ctx, resolvedModelID, operation); err != nil {
		return nil, err
	}

	providers, err := c.selector.SelectProviders(ctx, operation, resolvedModelID, strategy)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇婧愬ご鍒楄〃澶辫触: %w", err)
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
			return nil, fmt.Errorf("熔断器检查失败: %w", err)
		}
		if !canExecute {
			return nil, fmt.Errorf("源头 %d 熔断器打开", provider.ID)
		}
	}

	if c.rateLimiter != nil {
		allowed, err := c.rateLimiter.Allow(ctx, provider.ID, operation)
		if err == nil && !allowed {
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
		return result, err
	}

	result.Success = true
	result.Provider = provider
	result.Response = response

	if enableCircuitBreaker && c.circuitBreaker != nil {
		_ = c.circuitBreaker.RecordSuccess(ctx, provider.ID)
	}

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
		response, err := executor(timeoutCtx, provider)
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

// getProviderName 鑾峰彇婧愬ご鍚嶇О
func (c *cascadeController) getProviderName(provider *model.ModelProvider) string {
	if provider.Channel != nil {
		return provider.Channel.Name
	}
	return fmt.Sprintf("Provider-%d", provider.ID)
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

	// ExecuteStreamWithFailover 鎵ц娴佸紡璇锋眰骞舵敮鎸侀鍖呰秴鏃?failover
	// ? 棣栧寘瀹氫箟锛氱敱 StreamExecutor 鍦ㄢ€滈鏉?data: 鍐欏叆骞?Flush鈥濆悗瑙﹀彂 onFirstChunk 鍥炶皟
	// 鏍规嵁鏋舵瀯璁捐锛?// - 棣栧寘鍒拌揪鍓嶏細鍏佽 failover 鍒颁笅涓€涓簮澶?// - 棣栧寘鍒拌揪鍚庯細閿佸畾褰撳墠婧愬ご锛屼笉鍐嶅垏鎹?func (c *cascadeController) ExecuteStreamWithFailover(ctx context.Context, modelID string, strategy SelectionStrategy, executor StreamExecutor) (*CascadeResult, error) {
	startTime := time.Now()

	resolvedModelID, err := c.modelAggregator.ResolveModelAlias(ctx, modelID)
	if err != nil {
		return nil, fmt.Errorf("瑙ｆ瀽妯″瀷鍒悕澶辫触: %w", err)
	}

	providers, err := c.selector.SelectProviders(ctx, operation, resolvedModelID, strategy)
	if err != nil {
		return nil, fmt.Errorf("鑾峰彇婧愬ご鍒楄〃澶辫触: %w", err)
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

		if enableCircuitBreaker {
			canExecute, err := c.circuitBreaker.CanExecute(ctx, provider.ID)
			if err != nil {
				lastError = fmt.Errorf("鐔旀柇鍣ㄦ鏌ュけ璐? %w", err)
				result.FailedProviders = append(result.FailedProviders, FailedAttempt{
					ProviderID: provider.ID,
					Error:      lastError.Error(),
					LatencyMs:  0,
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
				continue
			}
		}

		attemptStart := time.Now()
		canFailover, err := c.executeStreamWithGuard(ctx, provider, executor)
		attemptLatency := time.Since(attemptStart).Milliseconds()

		if err != nil {
			lastError = err
			result.FailedProviders = append(result.FailedProviders, FailedAttempt{
				ProviderID:   provider.ID,
				ProviderName: c.getProviderName(provider),
				Error:        err.Error(),
				LatencyMs:    attemptLatency,
			})

			if enableCircuitBreaker {
				_ = c.circuitBreaker.RecordFailure(ctx, provider.ID, err)
			}
			_ = c.providerRepo.IncrementStats(ctx, provider.ID, false, attemptLatency, 0, 0, decimal.Zero)

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

		if enableCircuitBreaker {
			_ = c.circuitBreaker.RecordSuccess(ctx, provider.ID)
		}

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
		errCh <- executor(attemptCtx, provider, onFirstChunk)
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
