//go:build integration
// +build integration

package scheduler

import (
	"context"
	"testing"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// Mock repositories for testing
type MockModelProviderRepo struct {
	mock.Mock
}

func (m *MockModelProviderRepo) GetByID(ctx context.Context, id uint) (*model.ModelProvider, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ModelProvider), args.Error(1)
}

func (m *MockModelProviderRepo) GetByModelID(ctx context.Context, modelID string) ([]*model.ModelProvider, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ModelProvider), args.Error(1)
}

func (m *MockModelProviderRepo) GetAvailableProviders(ctx context.Context, modelID string) ([]*model.ModelProvider, error) {
	args := m.Called(ctx, modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ModelProvider), args.Error(1)
}

func (m *MockModelProviderRepo) Create(ctx context.Context, provider *model.ModelProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockModelProviderRepo) Update(ctx context.Context, provider *model.ModelProvider) error {
	args := m.Called(ctx, provider)
	return args.Error(0)
}

func (m *MockModelProviderRepo) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockModelProviderRepo) List(ctx context.Context, page, pageSize int) ([]*model.ModelProvider, int, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*model.ModelProvider), args.Int(1), args.Error(2)
}

func (m *MockModelProviderRepo) IncrementStats(ctx context.Context, providerID uint, success bool, latencyMs int64, promptTokens, completionTokens int, cost decimal.Decimal) error {
	args := m.Called(ctx, providerID, success, latencyMs, promptTokens, completionTokens, cost)
	return args.Error(0)
}

func (m *MockModelProviderRepo) GetMetrics(ctx context.Context, providerID uint, startTime, endTime time.Time) (*model.ProviderMetrics, error) {
	args := m.Called(ctx, providerID, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ProviderMetrics), args.Error(1)
}

type MockModelRepo struct {
	mock.Mock
}

func (m *MockModelRepo) GetByID(ctx context.Context, id string) (*model.Model, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.Model), args.Error(1)
}

func (m *MockModelRepo) GetPricing(modelID string) (*model.ModelPricing, error) {
	args := m.Called(modelID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*model.ModelPricing), args.Error(1)
}

func (m *MockModelRepo) Create(ctx context.Context, model *model.Model) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockModelRepo) Update(ctx context.Context, model *model.Model) error {
	args := m.Called(ctx, model)
	return args.Error(0)
}

func (m *MockModelRepo) Delete(ctx context.Context, id string) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockModelRepo) List(ctx context.Context, page, pageSize int) ([]*model.Model, int, error) {
	args := m.Called(ctx, page, pageSize)
	if args.Get(0) == nil {
		return nil, 0, args.Error(2)
	}
	return args.Get(0).([]*model.Model), args.Int(1), args.Error(2)
}

type MockProviderMetricsRepo struct {
	mock.Mock
}

func (m *MockProviderMetricsRepo) GetByProvider(ctx context.Context, providerID uint, granularity string, startTime, endTime time.Time) ([]*model.ProviderMetrics, error) {
	args := m.Called(ctx, providerID, granularity, startTime, endTime)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*model.ProviderMetrics), args.Error(1)
}

func (m *MockProviderMetricsRepo) Create(ctx context.Context, metrics *model.ProviderMetrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockProviderMetricsRepo) Update(ctx context.Context, metrics *model.ProviderMetrics) error {
	args := m.Called(ctx, metrics)
	return args.Error(0)
}

func (m *MockProviderMetricsRepo) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

// TestCascadeControllerIntegration 测试级联控制器的完整流程
func TestCascadeControllerIntegration(t *testing.T) {
	ctx := context.Background()

	// 创建 mock repositories
	mockProviderRepo := new(MockModelProviderRepo)
	mockMetricsRepo := new(MockProviderMetricsRepo)

	// 创建测试数据
	providers := []*model.ModelProvider{
		{
			ID:                     1,
			ModelID:                "gpt-4",
			ChannelID:              1,
			ActualCostPer1kInput:   decimal.NewFromFloat(0.03),
			ActualCostPer1kOutput:  decimal.NewFromFloat(0.06),
			Priority:               100,
			Weight:                 100,
			Status:                 "active",
			CircuitState:           "closed",
			FailureThreshold:       5,
			RecoveryTimeoutSeconds: 30,
			HealthScore:            95.0,
			Channel: &model.Channel{
				ID:   1,
				Name: "OpenAI",
			},
		},
		{
			ID:                     2,
			ModelID:                "gpt-4",
			ChannelID:              2,
			ActualCostPer1kInput:   decimal.NewFromFloat(0.02),
			ActualCostPer1kOutput:  decimal.NewFromFloat(0.04),
			Priority:               90,
			Weight:                 100,
			Status:                 "active",
			CircuitState:           "closed",
			FailureThreshold:       5,
			RecoveryTimeoutSeconds: 30,
			HealthScore:            90.0,
			Channel: &model.Channel{
				ID:   2,
				Name: "Azure",
			},
		},
	}

	// 设置 mock 期望
	mockProviderRepo.On("GetAvailableProviders", ctx, "gpt-4").Return(providers, nil)
	mockProviderRepo.On("IncrementStats", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 创建调度器组件
	stateStore := NewRuntimeStateStore(nil) // 使用内存存储
	circuitBreaker := NewCircuitBreaker(stateStore, DefaultCircuitBreakerConfig())
	healthTracker := NewHealthTracker(stateStore, DefaultHealthTrackerConfig())
	providerSelector := NewProviderSelector(mockProviderRepo, healthTracker, DefaultProviderSelectorConfig())
	instanceScheduler := NewInstanceScheduler(nil, nil, stateStore, DefaultInstanceSchedulerConfig())
	streamGuard := NewStreamGuard(stateStore, DefaultStreamGuardConfig())
	modelAggregator := NewModelAggregator(mockProviderRepo, nil)

	cascadeController := NewCascadeController(
		providerSelector,
		circuitBreaker,
		mockProviderRepo,
		mockMetricsRepo,
		instanceScheduler,
		streamGuard,
		modelAggregator,
		DefaultCascadeConfig(),
	)

	// 测试1: 成功的请求
	t.Run("SuccessfulRequest", func(t *testing.T) {
		executor := func(ctx context.Context, provider *model.ModelProvider) (interface{}, error) {
			return "success", nil
		}

		result, err := cascadeController.ExecuteWithStrategy(ctx, "gpt-4", StrategyCostFirst, executor)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Provider)
		assert.Equal(t, "success", result.Response)
		assert.Equal(t, 1, result.AttemptCount)
	})

	// 测试2: 第一个源头失败，切换到第二个源头
	t.Run("FailoverToSecondProvider", func(t *testing.T) {
		attemptCount := 0
		executor := func(ctx context.Context, provider *model.ModelProvider) (interface{}, error) {
			attemptCount++
			if attemptCount == 1 {
				return nil, assert.AnError
			}
			return "success", nil
		}

		result, err := cascadeController.ExecuteWithStrategy(ctx, "gpt-4", StrategyCostFirst, executor)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Provider)
		assert.Equal(t, "success", result.Response)
		assert.Equal(t, 2, result.AttemptCount)
		assert.Len(t, result.FailedProviders, 1)
	})

	// 测试3: 所有源头都失败
	t.Run("AllProvidersFailed", func(t *testing.T) {
		executor := func(ctx context.Context, provider *model.ModelProvider) (interface{}, error) {
			return nil, assert.AnError
		}

		result, err := cascadeController.ExecuteWithStrategy(ctx, "gpt-4", StrategyCostFirst, executor)

		assert.Error(t, err)
		assert.False(t, result.Success)
		assert.Nil(t, result.Response)
		assert.Equal(t, 3, result.AttemptCount) // maxRetries = 3
		assert.Len(t, result.FailedProviders, 3)
	})
}

// TestStreamGuardIntegration 测试流式请求保护机制
func TestStreamGuardIntegration(t *testing.T) {
	ctx := context.Background()

	// 创建 mock repositories
	mockProviderRepo := new(MockModelProviderRepo)
	mockMetricsRepo := new(MockProviderMetricsRepo)

	// 创建测试数据
	providers := []*model.ModelProvider{
		{
			ID:                     1,
			ModelID:                "gpt-4",
			ChannelID:              1,
			ActualCostPer1kInput:   decimal.NewFromFloat(0.03),
			ActualCostPer1kOutput:  decimal.NewFromFloat(0.06),
			Priority:               100,
			Weight:                 100,
			Status:                 "active",
			CircuitState:           "closed",
			FailureThreshold:       5,
			RecoveryTimeoutSeconds: 30,
			HealthScore:            95.0,
			Channel: &model.Channel{
				ID:   1,
				Name: "OpenAI",
			},
		},
	}

	// 设置 mock 期望
	mockProviderRepo.On("GetAvailableProviders", ctx, "gpt-4").Return(providers, nil)
	mockProviderRepo.On("IncrementStats", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 创建调度器组件
	stateStore := NewRuntimeStateStore(nil)
	circuitBreaker := NewCircuitBreaker(stateStore, DefaultCircuitBreakerConfig())
	healthTracker := NewHealthTracker(stateStore, DefaultHealthTrackerConfig())
	providerSelector := NewProviderSelector(mockProviderRepo, healthTracker, DefaultProviderSelectorConfig())
	instanceScheduler := NewInstanceScheduler(nil, nil, stateStore, DefaultInstanceSchedulerConfig())
	streamGuard := NewStreamGuard(stateStore, DefaultStreamGuardConfig())
	modelAggregator := NewModelAggregator(mockProviderRepo, nil)

	cascadeController := NewCascadeController(
		providerSelector,
		circuitBreaker,
		mockProviderRepo,
		mockMetricsRepo,
		instanceScheduler,
		streamGuard,
		modelAggregator,
		DefaultCascadeConfig(),
	)

	// 测试1: 流式请求成功
	t.Run("SuccessfulStreamRequest", func(t *testing.T) {
		executor := func(ctx context.Context, provider *model.ModelProvider) (interface{}, error) {
			return "stream success", nil
		}

		result, err := cascadeController.ExecuteStreamWithFailover(ctx, "gpt-4", StrategyCostFirst, executor)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Provider)
		assert.Equal(t, "stream success", result.Response)
	})

	// 测试2: 流式请求首包超时，允许 failover
	t.Run("StreamFirstChunkTimeoutFailover", func(t *testing.T) {
		attemptCount := 0
		executor := func(ctx context.Context, provider *model.ModelProvider) (interface{}, error) {
			attemptCount++
			if attemptCount == 1 {
				// 模拟首包超时
				time.Sleep(4 * time.Second)
				return nil, assert.AnError
			}
			return "stream success", nil
		}

		result, err := cascadeController.ExecuteStreamWithFailover(ctx, "gpt-4", StrategyCostFirst, executor)

		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, "stream success", result.Response)
	})
}

// TestCircuitBreakerIntegration 测试熔断器集成
func TestCircuitBreakerIntegration(t *testing.T) {
	ctx := context.Background()

	// 创建 mock repositories
	mockProviderRepo := new(MockModelProviderRepo)
	mockMetricsRepo := new(MockProviderMetricsRepo)

	// 创建测试数据
	providers := []*model.ModelProvider{
		{
			ID:                     1,
			ModelID:                "gpt-4",
			ChannelID:              1,
			ActualCostPer1kInput:   decimal.NewFromFloat(0.03),
			ActualCostPer1kOutput:  decimal.NewFromFloat(0.06),
			Priority:               100,
			Weight:                 100,
			Status:                 "active",
			CircuitState:           "closed",
			FailureThreshold:       3,
			RecoveryTimeoutSeconds: 30,
			HealthScore:            95.0,
			Channel: &model.Channel{
				ID:   1,
				Name: "OpenAI",
			},
		},
	}

	// 设置 mock 期望
	mockProviderRepo.On("GetAvailableProviders", ctx, "gpt-4").Return(providers, nil)
	mockProviderRepo.On("IncrementStats", mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return(nil)

	// 创建调度器组件
	stateStore := NewRuntimeStateStore(nil)
	circuitBreaker := NewCircuitBreaker(stateStore, DefaultCircuitBreakerConfig())
	healthTracker := NewHealthTracker(stateStore, DefaultHealthTrackerConfig())
	providerSelector := NewProviderSelector(mockProviderRepo, healthTracker, DefaultProviderSelectorConfig())
	instanceScheduler := NewInstanceScheduler(nil, nil, stateStore, DefaultInstanceSchedulerConfig())
	streamGuard := NewStreamGuard(stateStore, DefaultStreamGuardConfig())
	modelAggregator := NewModelAggregator(mockProviderRepo, nil)

	cascadeController := NewCascadeController(
		providerSelector,
		circuitBreaker,
		mockProviderRepo,
		mockMetricsRepo,
		instanceScheduler,
		streamGuard,
		modelAggregator,
		DefaultCascadeConfig(),
	)

	// 测试1: 触发熔断器
	t.Run("TriggerCircuitBreaker", func(t *testing.T) {
		executor := func(ctx context.Context, provider *model.ModelProvider) (interface{}, error) {
			return nil, assert.AnError
		}

		// 连续失败3次，触发熔断
		for i := 0; i < 3; i++ {
			_, _ = cascadeController.ExecuteWithStrategy(ctx, "gpt-4", StrategyCostFirst, executor)
		}

		// 检查熔断器状态
		canExecute, err := circuitBreaker.CanExecute(ctx, 1)
		assert.NoError(t, err)
		assert.False(t, canExecute, "熔断器应该打开")
	})

	// 测试2: 熔断器恢复
	t.Run("CircuitBreakerRecovery", func(t *testing.T) {
		// 等待恢复超时
		time.Sleep(35 * time.Second)

		// 检查熔断器状态
		canExecute, err := circuitBreaker.CanExecute(ctx, 1)
		assert.NoError(t, err)
		assert.True(t, canExecute, "熔断器应该恢复")
	})
}

// TestModelAggregatorIntegration 测试模型聚合器集成
func TestModelAggregatorIntegration(t *testing.T) {
	ctx := context.Background()

	// 创建 mock repositories
	mockProviderRepo := new(MockModelProviderRepo)

	// 创建测试数据
	providers := []*model.ModelProvider{
		{
			ID:                     1,
			ModelID:                "gpt-4",
			ChannelID:              1,
			ActualCostPer1kInput:   decimal.NewFromFloat(0.03),
			ActualCostPer1kOutput:  decimal.NewFromFloat(0.06),
			Priority:               100,
			Weight:                 100,
			Status:                 "active",
			CircuitState:           "closed",
			FailureThreshold:       5,
			RecoveryTimeoutSeconds: 30,
			HealthScore:            95.0,
			Channel: &model.Channel{
				ID:   1,
				Name: "OpenAI",
			},
		},
	}

	// 设置 mock 期望
	mockProviderRepo.On("GetAvailableProviders", ctx, "gpt-4").Return(providers, nil)

	// 创建模型聚合器
	modelAggregator := NewModelAggregator(mockProviderRepo, nil)

	// 测试1: 解析模型别名
	t.Run("ResolveModelAlias", func(t *testing.T) {
		resolvedModelID, err := modelAggregator.ResolveModelAlias(ctx, "gpt-4")
		assert.NoError(t, err)
		assert.Equal(t, "gpt-4", resolvedModelID)
	})

	// 测试2: 获取可用的源头组
	t.Run("GetAvailableProviderGroups", func(t *testing.T) {
		providers, err := modelAggregator.GetAvailableProviderGroups(ctx, "gpt-4")
		assert.NoError(t, err)
		assert.Len(t, providers, 1)
		assert.Equal(t, uint(1), providers[0].ID)
	})
}

// TestHealthTrackerIntegration 测试健康追踪器集成
func TestHealthTrackerIntegration(t *testing.T) {
	ctx := context.Background()

	// 创建健康追踪器
	stateStore := NewRuntimeStateStore(nil)
	healthTracker := NewHealthTracker(stateStore, DefaultHealthTrackerConfig())

	// 测试1: 记录成功请求
	t.Run("RecordSuccess", func(t *testing.T) {
		err := healthTracker.RecordSuccess(ctx, 1, 100, 50, 100)
		assert.NoError(t, err)

		healthScore, err := healthTracker.GetHealthScore(ctx, 1)
		assert.NoError(t, err)
		assert.Greater(t, healthScore, 90.0, "健康分应该很高")
	})

	// 测试2: 记录失败请求
	t.Run("RecordFailure", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			err := healthTracker.RecordFailure(ctx, 1, 500, 0, 0, assert.AnError)
			assert.NoError(t, err)
		}

		healthScore, err := healthTracker.GetHealthScore(ctx, 1)
		assert.NoError(t, err)
		assert.Less(t, healthScore, 80.0, "健康分应该降低")
	})
}

// TestCostCalculatorIntegration 测试成本计算器集成
func TestCostCalculatorIntegration(t *testing.T) {
	// 创建 mock repositories
	mockModelRepo := new(MockModelRepo)
	mockProviderRepo := new(MockModelProviderRepo)

	// 创建测试数据
	pricing := &model.ModelPricing{
		InputPrice:  decimal.NewFromFloat(0.03),
		OutputPrice: decimal.NewFromFloat(0.06),
		PriceUnit:   1000,
	}

	provider := &model.ModelProvider{
		ID:                    1,
		ModelID:               "gpt-4",
		ActualCostPer1kInput:  decimal.NewFromFloat(0.03),
		ActualCostPer1kOutput: decimal.NewFromFloat(0.06),
	}

	// 设置 mock 期望
	mockModelRepo.On("GetPricing", "gpt-4").Return(pricing, nil)

	// 创建成本计算器
	costCalculator := NewCostCalculator(mockModelRepo, mockProviderRepo, DefaultCostCalculatorConfig())

	// 测试1: 计算请求成本
	t.Run("CalculateCost", func(t *testing.T) {
		cost, err := costCalculator.CalculateCost("gpt-4", 1000, 500)
		assert.NoError(t, err)
		expectedCost := decimal.NewFromFloat(0.03).Add(decimal.NewFromFloat(0.03))
		assert.True(t, cost.Equal(expectedCost), "成本计算错误")
	})

	// 测试2: 计算源头成本
	t.Run("CalculateProviderCost", func(t *testing.T) {
		cost := costCalculator.CalculateProviderCost(provider, 1000, 500)
		expectedCost := decimal.NewFromFloat(0.03).Add(decimal.NewFromFloat(0.03))
		assert.True(t, cost.Equal(expectedCost), "源头成本计算错误")
	})

	// 测试3: 估算请求成本
	t.Run("EstimateCost", func(t *testing.T) {
		req := &ChatCompletionRequest{
			Model: "gpt-4",
			Messages: []map[string]interface{}{
				{"role": "user", "content": "Hello, how are you?"},
			},
			MaxTokens: 500,
		}

		cost, err := costCalculator.EstimateCost(req)
		assert.NoError(t, err)
		assert.Greater(t, cost.Sign(), 0, "估算成本应该大于0")
	})
}
