package scheduler

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"

	"github.com/redis/go-redis/v9"
)

// RuntimeStateStore 运行时状态存储接口
// 使用Redis存储熔断器状态、健康指标、实例限流状态等运行时数据
type RuntimeStateStore interface {
	// ==================== 熔断器状态 ====================

	// GetCircuitState 获取熔断器状态
	GetCircuitState(ctx context.Context, providerID uint) (model.CircuitState, error)
	// SetCircuitState 设置熔断器状态
	SetCircuitState(ctx context.Context, providerID uint, state model.CircuitState, ttl time.Duration) error
	// IncrementFailureCount 增加失败计数
	IncrementFailureCount(ctx context.Context, providerID uint) (int64, error)
	// ResetFailureCount 重置失败计数
	ResetFailureCount(ctx context.Context, providerID uint) error
	// GetFailureCount 获取失败计数
	GetFailureCount(ctx context.Context, providerID uint) (int64, error)

	// ==================== 健康指标 ====================

	// GetHealthMetrics 获取健康指标
	GetHealthMetrics(ctx context.Context, providerID uint) (*HealthMetrics, error)
	// UpdateHealthMetrics 更新健康指标
	UpdateHealthMetrics(ctx context.Context, providerID uint, metrics *HealthMetrics) error
	// IncrementHealthMetric 增加健康指标
	IncrementHealthMetric(ctx context.Context, providerID uint, success bool, latencyMs int64) error

	// ==================== 实例限流状态 ====================

	// AcquireInstanceSlot 获取实例槽位（并发控制）
	// ⭐ 修复：添加 maxConcurrency 参数，从实例配置读取并发限制
	AcquireInstanceSlot(ctx context.Context, instanceID uint, maxConcurrency int64) (bool, error)
	// ReleaseInstanceSlot 释放实例槽位
	ReleaseInstanceSlot(ctx context.Context, instanceID uint) error
	// GetInstanceConcurrency 获取实例当前并发数
	GetInstanceConcurrency(ctx context.Context, instanceID uint) (int64, error)
	// CheckInstanceRateLimit 检查实例速率限制
	CheckInstanceRateLimit(ctx context.Context, instanceID uint, limitType string, limit int64) (bool, error)
	// ConsumeInstanceRateLimit 按窗口消耗实例速率额度（固定窗口）
	ConsumeInstanceRateLimit(ctx context.Context, instanceID uint, limitType string, limit int64, window time.Duration, increment int64) (bool, error)
	// IncrementInstanceRate 增加实例速率计数
	IncrementInstanceRate(ctx context.Context, instanceID uint, limitType string) error

	// ==================== 实例熔断器状态 ====================

	// GetInstanceCircuitState 获取实例熔断器状态
	GetInstanceCircuitState(ctx context.Context, instanceID uint) (model.CircuitState, error)
	// SetInstanceCircuitState 设置实例熔断器状态
	SetInstanceCircuitState(ctx context.Context, instanceID uint, state model.CircuitState, ttl time.Duration) error
	// IncrementInstanceFailureCount 增加实例失败计数（连续失败）
	IncrementInstanceFailureCount(ctx context.Context, instanceID uint) (int64, error)
	// ResetInstanceFailureCount 重置实例失败计数（连续失败）
	ResetInstanceFailureCount(ctx context.Context, instanceID uint) error
	// GetInstanceFailureCount 获取实例失败计数（连续失败）
	GetInstanceFailureCount(ctx context.Context, instanceID uint) (int64, error)

	// ==================== 半开状态计数器 ====================

	// GetHalfOpenCounter 获取半开状态计数器
	GetHalfOpenCounter(ctx context.Context, providerID uint) (*HalfOpenCounter, error)
	// SetHalfOpenCounter 设置半开状态计数器
	SetHalfOpenCounter(ctx context.Context, providerID uint, counter *HalfOpenCounter, ttl time.Duration) error
	// DeleteHalfOpenCounter 删除半开状态计数器
	DeleteHalfOpenCounter(ctx context.Context, providerID uint) error

	// ==================== 流式请求状态 ====================

	// StartStream 开始流式请求
	StartStream(ctx context.Context, streamID string, providerID uint) error
	// OnFirstChunk 首字节到达
	OnFirstChunk(ctx context.Context, streamID string) error
	// IsStreamLocked 检查流式请求是否锁定
	IsStreamLocked(ctx context.Context, streamID string) (bool, error)
	// EndStream 结束流式请求
	EndStream(ctx context.Context, streamID string, success bool) error
	// GetStreamInfo 获取流式请求信息
	GetStreamInfo(ctx context.Context, streamID string) (*StreamInfo, error)

	// ==================== Job 提交点（Commit Point） ====================

	// GetJobCommit 获取 Job 提交锁
	GetJobCommit(ctx context.Context, jobID string) (*JobCommitInfo, error)
	// EnsureJobCommit 确保 Job 提交锁存在（幂等：已存在则返回既有值）
	EnsureJobCommit(ctx context.Context, jobID string, providerID uint, instanceID uint, ttl time.Duration) (*JobCommitInfo, error)
	// DeleteJobCommit 删除 Job 提交锁
	DeleteJobCommit(ctx context.Context, jobID string) error
}

// HealthMetrics 健康指标
type HealthMetrics struct {
	TotalRequests   int64      `json:"total_requests"`
	SuccessRequests int64      `json:"success_requests"`
	FailedRequests  int64      `json:"failed_requests"`
	TotalLatency    int64      `json:"total_latency_ms"`
	LastSuccessAt   *time.Time `json:"last_success_at,omitempty"`
	LastFailureAt   *time.Time `json:"last_failure_at,omitempty"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

// HalfOpenCounter 半开状态计数器
type HalfOpenCounter struct {
	Requests  int `json:"requests"`
	Successes int `json:"successes"`
}

// StreamInfo 流式请求信息
type StreamInfo struct {
	StreamID     string     `json:"stream_id"`
	ProviderID   uint       `json:"provider_id"`
	StartedAt    time.Time  `json:"started_at"`
	FirstChunkAt *time.Time `json:"first_chunk_at,omitempty"`
	Locked       bool       `json:"locked"`
}

// JobCommitInfo Job 提交锁信息
// 用于“异步/长耗时任务”的 Routing Commit 语义：一旦进入提交点，禁止 failover，避免重复下单。
type JobCommitInfo struct {
	JobID      string    `json:"job_id"`
	ProviderID uint      `json:"provider_id"`
	InstanceID uint      `json:"instance_id"`
	CreatedAt  time.Time `json:"created_at"`
}

// runtimeStateStore 运行时状态存储实现
type runtimeStateStore struct {
	redisClient interface{}
}

// NewRuntimeStateStore 创建运行时状态存储
func NewRuntimeStateStore() RuntimeStateStore {
	return &runtimeStateStore{
		redisClient: cache.GetClient(),
	}
}

// ==================== 熔断器状态 ====================

const (
	keyCircuitState         = "circuit:state:%d"            // circuit:state:{provider_id} -> CircuitState
	keyFailureCount         = "circuit:failure:%d"          // circuit:failure:{provider_id} -> int
	keyCircuitOpenUntil     = "circuit:open_until:%d"       // circuit:open_until:{provider_id} -> int64(open_until_unix_ms)
	keyHealthMetrics        = "health:metrics:%d"           // health:metrics:{provider_id} -> HealthMetrics
	keyInstanceSlot         = "instance:slot:%d"            // instance:slot:{instance_id} -> int (concurrent count)
	keyInstanceRate         = "instance:rate:%d:%s"         // instance:rate:{instance_id}:{type} -> int
	keyInstanceRateWindow   = "instance:rate:%d:%s:%d"      // instance:rate:{instance_id}:{type}:{window_seconds} -> int
	keyHalfOpenCounter      = "circuit:halfopen:%d"         // circuit:halfopen:{provider_id} -> HalfOpenCounter
	keyStreamInfo           = "stream:info:%s"              // stream:info:{stream_id} -> StreamInfo
	keyInstanceCircuitState = "instance:circuit:state:%d"   // instance:circuit:state:{instance_id} -> CircuitState
	keyInstanceFailureCount = "instance:circuit:failure:%d" // instance:circuit:failure:{instance_id} -> int
	keyJobCommit            = "job:commit:%s"               // job:commit:{job_id} -> JobCommitInfo
)

// GetCircuitState 获取熔断器状态
func (s *runtimeStateStore) GetCircuitState(ctx context.Context, providerID uint) (model.CircuitState, error) {
	var state model.CircuitState
	key := fmt.Sprintf(keyCircuitState, providerID)

	err := cache.Get(key, &state)
	if err != nil {
		// Redis中没有，返回默认关闭状态
		return model.CircuitStateClosed, nil
	}

	// OPEN 使用 open_until key 驱动从 OPEN -> HALF_OPEN 的状态迁移：
	// - OPEN 状态不依赖 state key 的 TTL（避免 TTL 到期后直接“回到 CLOSED”）
	// - open_until key 通过 TTL 自动清理；消失即表示到达半开窗口
	if state == model.CircuitStateOpen {
		openUntilKey := fmt.Sprintf(keyCircuitOpenUntil, providerID)
		exists, existsErr := cache.Exists(openUntilKey)
		if existsErr == nil && !exists {
			// 自动转为半开状态（允许少量探测请求）
			_ = cache.Set(key, model.CircuitStateHalfOpen, 0)
			return model.CircuitStateHalfOpen, nil
		}
	}

	return state, nil
}

// SetCircuitState 设置熔断器状态
func (s *runtimeStateStore) SetCircuitState(ctx context.Context, providerID uint, state model.CircuitState, ttl time.Duration) error {
	key := fmt.Sprintf(keyCircuitState, providerID)

	// 约定：OPEN 的 ttl 表示“恢复窗口”，由 open_until key 的 TTL 驱动从 OPEN -> HALF_OPEN。
	// state key 本身不使用 ttl（避免到期后无法区分“正常关闭”和“OPEN 过期”两类状态）。
	if err := cache.Set(key, state, 0); err != nil {
		return err
	}

	openUntilKey := fmt.Sprintf(keyCircuitOpenUntil, providerID)
	if state != model.CircuitStateOpen {
		_ = cache.Delete(openUntilKey)
		return nil
	}

	if ttl <= 0 {
		// 兜底：给一个较短窗口，避免永远停留在 OPEN
		ttl = 30 * time.Second
	}
	openUntilMs := time.Now().Add(ttl).UnixMilli()
	if err := cache.Set(openUntilKey, openUntilMs, ttl); err != nil {
		return err
	}
	return nil
}

// IncrementFailureCount 增加失败计数
func (s *runtimeStateStore) IncrementFailureCount(ctx context.Context, providerID uint) (int64, error) {
	key := fmt.Sprintf(keyFailureCount, providerID)
	return cache.Incr(key)
}

// ResetFailureCount 重置失败计数
func (s *runtimeStateStore) ResetFailureCount(ctx context.Context, providerID uint) error {
	key := fmt.Sprintf(keyFailureCount, providerID)
	return cache.Delete(key)
}

// GetFailureCount 获取失败计数
func (s *runtimeStateStore) GetFailureCount(ctx context.Context, providerID uint) (int64, error) {
	key := fmt.Sprintf(keyFailureCount, providerID)

	countStr, err := cache.GetClient().Get(ctx, key).Result()
	if err != nil {
		return 0, nil
	}

	var count int64
	_, err = fmt.Sscanf(countStr, "%d", &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ==================== 健康指标 ====================

// GetHealthMetrics 获取健康指标
func (s *runtimeStateStore) GetHealthMetrics(ctx context.Context, providerID uint) (*HealthMetrics, error) {
	var metrics HealthMetrics
	key := fmt.Sprintf(keyHealthMetrics, providerID)

	err := cache.Get(key, &metrics)
	if err != nil {
		// Redis中没有，返回空指标
		return &HealthMetrics{
			UpdatedAt: time.Now(),
		}, nil
	}

	return &metrics, nil
}

// UpdateHealthMetrics 更新健康指标
func (s *runtimeStateStore) UpdateHealthMetrics(ctx context.Context, providerID uint, metrics *HealthMetrics) error {
	key := fmt.Sprintf(keyHealthMetrics, providerID)
	metrics.UpdatedAt = time.Now()

	// 设置较长的过期时间（1小时）
	return cache.Set(key, metrics, time.Hour)
}

// IncrementHealthMetric 增加健康指标
func (s *runtimeStateStore) IncrementHealthMetric(ctx context.Context, providerID uint, success bool, latencyMs int64) error {
	key := fmt.Sprintf(keyHealthMetrics, providerID)

	// 使用Lua脚本原子性更新
	script := `
		local key = KEYS[1]
		local success = ARGV[1]
		local latency = tonumber(ARGV[2])
		local now = ARGV[3]
		
		local data = redis.call("get", key)
		local metrics
		
		if data == false then
			metrics = {
				total_requests = 0,
				success_requests = 0,
				failed_requests = 0,
				total_latency = 0,
				updated_at = now
			}
		else
			metrics = cjson.decode(data)
		end
		
		metrics.total_requests = metrics.total_requests + 1
		metrics.total_latency = metrics.total_latency + latency
		metrics.updated_at = now
		
		if success == "1" then
			metrics.success_requests = metrics.success_requests + 1
			metrics.last_success_at = now
		else
			metrics.failed_requests = metrics.failed_requests + 1
			metrics.last_failure_at = now
		end
		
		redis.call("set", key, cjson.encode(metrics))
		redis.call("expire", key, 3600)
		
		return "OK"
	`

	successFlag := "0"
	if success {
		successFlag = "1"
	}

	now := time.Now().Format(time.RFC3339)

	_, err := cache.GetClient().Eval(ctx, script, []string{key}, successFlag, latencyMs, now).Result()
	return err
}

// ==================== 实例限流状态 ====================

// AcquireInstanceSlot 获取实例槽位（并发控制）
// ⭐ 修复：从参数接收 maxConcurrency，而不是硬编码为 10
// 根据架构设计：并发限制应从 ProviderInstance.MaxConcurrency 配置读取
func (s *runtimeStateStore) AcquireInstanceSlot(ctx context.Context, instanceID uint, maxConcurrency int64) (bool, error) {
	key := fmt.Sprintf(keyInstanceSlot, instanceID)

	// 使用Lua脚本原子性检查和增加
	script := `
		local key = KEYS[1]
		local max_concurrency = tonumber(ARGV[1])
		
		local current = tonumber(redis.call("get", key))
		if current == nil then
			current = 0
		end
		
		if current >= max_concurrency then
			return 0
		end
		
		redis.call("incr", key)
		redis.call("expire", key, 300)
		return 1
	`

	// ⭐ 修复：使用传入的 maxConcurrency 参数，而不是硬编码
	result, err := cache.GetClient().Eval(ctx, script, []string{key}, maxConcurrency).Int64()
	if err != nil {
		return false, err
	}

	return result == 1, nil
}

// ReleaseInstanceSlot 释放实例槽位
func (s *runtimeStateStore) ReleaseInstanceSlot(ctx context.Context, instanceID uint) error {
	key := fmt.Sprintf(keyInstanceSlot, instanceID)

	// 使用Lua脚本原子性减少
	script := `
		local key = KEYS[1]
		
		local current = tonumber(redis.call("get", key))
		if current == nil or current <= 0 then
			redis.call("del", key)
			return 0
		end
		
		redis.call("decr", key)
		return current - 1
	`

	_, err := cache.GetClient().Eval(ctx, script, []string{key}).Result()
	return err
}

// GetInstanceConcurrency 获取实例当前并发数
func (s *runtimeStateStore) GetInstanceConcurrency(ctx context.Context, instanceID uint) (int64, error) {
	key := fmt.Sprintf(keyInstanceSlot, instanceID)

	countStr, err := cache.GetClient().Get(ctx, key).Result()
	if err != nil {
		return 0, nil
	}

	var count int64
	_, err = fmt.Sscanf(countStr, "%d", &count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// CheckInstanceRateLimit 检查实例速率限制
func (s *runtimeStateStore) CheckInstanceRateLimit(ctx context.Context, instanceID uint, limitType string, limit int64) (bool, error) {
	key := fmt.Sprintf(keyInstanceRate, instanceID, limitType)

	// 使用滑动窗口限流
	return cache.SlidingWindowRateLimit(key, limit, time.Minute)
}

// ConsumeInstanceRateLimit 按窗口消耗实例速率额度（固定窗口）
func (s *runtimeStateStore) ConsumeInstanceRateLimit(ctx context.Context, instanceID uint, limitType string, limit int64, window time.Duration, increment int64) (bool, error) {
	_ = ctx
	if limit <= 0 {
		return true, nil
	}
	if window <= 0 {
		window = time.Minute
	}
	if increment <= 0 {
		increment = 1
	}

	limitType = strings.ToLower(strings.TrimSpace(limitType))
	key := fmt.Sprintf(keyInstanceRateWindow, instanceID, limitType, int64(window.Seconds()))
	return cache.FixedWindowRateLimit(key, limit, window, increment)
}

// IncrementInstanceRate 增加实例速率计数
func (s *runtimeStateStore) IncrementInstanceRate(ctx context.Context, instanceID uint, limitType string) error {
	key := fmt.Sprintf(keyInstanceRate, instanceID, limitType)

	// 添加到有序集合
	now := float64(time.Now().UnixMilli())
	return cache.GetClient().ZAdd(ctx, key, redis.Z{Score: now, Member: now}).Err()
}

// ==================== 半开状态计数器 ====================

// GetHalfOpenCounter 获取半开状态计数器
func (s *runtimeStateStore) GetHalfOpenCounter(ctx context.Context, providerID uint) (*HalfOpenCounter, error) {
	var counter HalfOpenCounter
	key := fmt.Sprintf(keyHalfOpenCounter, providerID)

	err := cache.Get(key, &counter)
	if err != nil {
		// Redis中没有，返回空计数器
		return &HalfOpenCounter{}, nil
	}

	return &counter, nil
}

// SetHalfOpenCounter 设置半开状态计数器
func (s *runtimeStateStore) SetHalfOpenCounter(ctx context.Context, providerID uint, counter *HalfOpenCounter, ttl time.Duration) error {
	key := fmt.Sprintf(keyHalfOpenCounter, providerID)
	return cache.Set(key, counter, ttl)
}

// DeleteHalfOpenCounter 删除半开状态计数器
func (s *runtimeStateStore) DeleteHalfOpenCounter(ctx context.Context, providerID uint) error {
	key := fmt.Sprintf(keyHalfOpenCounter, providerID)
	return cache.Delete(key)
}

// ==================== 流式请求状态 ====================

// StartStream 开始流式请求
func (s *runtimeStateStore) StartStream(ctx context.Context, streamID string, providerID uint) error {
	key := fmt.Sprintf(keyStreamInfo, streamID)

	info := &StreamInfo{
		StreamID:   streamID,
		ProviderID: providerID,
		StartedAt:  time.Now(),
		Locked:     false,
	}

	// 设置5分钟过期时间
	return cache.Set(key, info, 5*time.Minute)
}

// OnFirstChunk 首字节到达
func (s *runtimeStateStore) OnFirstChunk(ctx context.Context, streamID string) error {
	key := fmt.Sprintf(keyStreamInfo, streamID)

	// 使用Lua脚本原子性更新
	script := `
		local key = KEYS[1]
		local now = ARGV[1]
		
		local data = redis.call("get", key)
		if data == false then
			return nil
		end
		
		local info = cjson.decode(data)
		info.first_chunk_at = now
		info.locked = true
		
		redis.call("set", key, cjson.encode(info))
		return "OK"
	`

	now := time.Now().Format(time.RFC3339)

	_, err := cache.GetClient().Eval(ctx, script, []string{key}, now).Result()
	return err
}

// IsStreamLocked 检查流式请求是否锁定
func (s *runtimeStateStore) IsStreamLocked(ctx context.Context, streamID string) (bool, error) {
	key := fmt.Sprintf(keyStreamInfo, streamID)

	var info StreamInfo
	err := cache.Get(key, &info)
	if err != nil {
		// Redis中没有，未锁定
		return false, nil
	}

	return info.Locked, nil
}

// EndStream 结束流式请求
func (s *runtimeStateStore) EndStream(ctx context.Context, streamID string, success bool) error {
	key := fmt.Sprintf(keyStreamInfo, streamID)

	// 删除流式请求信息
	return cache.Delete(key)
}

// GetStreamInfo 获取流式请求信息
func (s *runtimeStateStore) GetStreamInfo(ctx context.Context, streamID string) (*StreamInfo, error) {
	var info StreamInfo
	key := fmt.Sprintf(keyStreamInfo, streamID)

	err := cache.Get(key, &info)
	if err != nil {
		return nil, err
	}

	return &info, nil
}

// ==================== 实例熔断器状态 ====================

func (s *runtimeStateStore) GetInstanceCircuitState(ctx context.Context, instanceID uint) (model.CircuitState, error) {
	var state model.CircuitState
	key := fmt.Sprintf(keyInstanceCircuitState, instanceID)
	if err := cache.Get(key, &state); err != nil {
		return model.CircuitStateClosed, nil
	}
	return state, nil
}

func (s *runtimeStateStore) SetInstanceCircuitState(ctx context.Context, instanceID uint, state model.CircuitState, ttl time.Duration) error {
	key := fmt.Sprintf(keyInstanceCircuitState, instanceID)
	return cache.Set(key, state, ttl)
}

func (s *runtimeStateStore) IncrementInstanceFailureCount(ctx context.Context, instanceID uint) (int64, error) {
	key := fmt.Sprintf(keyInstanceFailureCount, instanceID)
	return cache.Incr(key)
}

func (s *runtimeStateStore) ResetInstanceFailureCount(ctx context.Context, instanceID uint) error {
	key := fmt.Sprintf(keyInstanceFailureCount, instanceID)
	return cache.Delete(key)
}

func (s *runtimeStateStore) GetInstanceFailureCount(ctx context.Context, instanceID uint) (int64, error) {
	key := fmt.Sprintf(keyInstanceFailureCount, instanceID)
	countStr, err := cache.GetClient().Get(ctx, key).Result()
	if err != nil {
		return 0, nil
	}
	var count int64
	if _, err := fmt.Sscanf(countStr, "%d", &count); err != nil {
		return 0, err
	}
	return count, nil
}

// ==================== Job 提交点（Commit Point） ====================

func (s *runtimeStateStore) GetJobCommit(ctx context.Context, jobID string) (*JobCommitInfo, error) {
	_ = ctx
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, nil
	}

	key := fmt.Sprintf(keyJobCommit, jobID)
	var info JobCommitInfo
	if err := cache.Get(key, &info); err != nil {
		return nil, nil
	}
	if info.JobID == "" {
		info.JobID = jobID
	}
	return &info, nil
}

func (s *runtimeStateStore) EnsureJobCommit(ctx context.Context, jobID string, providerID uint, instanceID uint, ttl time.Duration) (*JobCommitInfo, error) {
	_ = ctx
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil, fmt.Errorf("jobID 不能为空")
	}

	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	existing, _ := s.GetJobCommit(ctx, jobID)
	if existing != nil {
		if existing.ProviderID != providerID || existing.InstanceID != instanceID {
			return nil, fmt.Errorf("job commit 冲突: job=%s existing(provider=%d,instance=%d) new(provider=%d,instance=%d)",
				jobID, existing.ProviderID, existing.InstanceID, providerID, instanceID)
		}
		return existing, nil
	}

	info := &JobCommitInfo{
		JobID:      jobID,
		ProviderID: providerID,
		InstanceID: instanceID,
		CreatedAt:  time.Now(),
	}

	key := fmt.Sprintf(keyJobCommit, jobID)
	created, err := cache.SetNX(key, info, ttl)
	if err != nil {
		return nil, err
	}
	if !created {
		loaded, _ := s.GetJobCommit(ctx, jobID)
		if loaded != nil {
			if loaded.ProviderID != providerID || loaded.InstanceID != instanceID {
				return nil, fmt.Errorf("job commit 冲突: job=%s existing(provider=%d,instance=%d) new(provider=%d,instance=%d)",
					jobID, loaded.ProviderID, loaded.InstanceID, providerID, instanceID)
			}
			return loaded, nil
		}
	}
	return info, nil
}

func (s *runtimeStateStore) DeleteJobCommit(ctx context.Context, jobID string) error {
	_ = ctx
	jobID = strings.TrimSpace(jobID)
	if jobID == "" {
		return nil
	}
	key := fmt.Sprintf(keyJobCommit, jobID)
	return cache.Delete(key)
}
