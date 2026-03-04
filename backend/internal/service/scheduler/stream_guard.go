package scheduler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// StreamGuard 流式请求保护机制接口
// 实现Stream First-Chunk Failover逻辑
type StreamGuard interface {
	// StartStream 开始流式请求
	StartStream(ctx context.Context, providerID uint) (StreamContext, error)
	// OnFirstChunk 首字节到达
	OnFirstChunk(ctx context.Context, streamCtx StreamContext) error
	// IsLocked 检查流式请求是否锁定
	IsLocked(ctx context.Context, streamCtx StreamContext) (bool, error)
	// EndStream 结束流式请求
	EndStream(ctx context.Context, streamCtx StreamContext, success bool) error
	// GetStreamInfo 获取流式请求信息
	GetStreamInfo(ctx context.Context, streamID string) (*StreamInfo, error)
	// CleanupExpiredStreams 清理过期的流式请求
	CleanupExpiredStreams(ctx context.Context) error
}

// StreamContext 流式请求上下文
type StreamContext struct {
	StreamID     string     `json:"stream_id"`
	ProviderID   uint       `json:"provider_id"`
	StartedAt    time.Time  `json:"started_at"`
	FirstChunkAt *time.Time `json:"first_chunk_at,omitempty"`
	Locked       bool       `json:"locked"`
}

// StreamGuardConfig 流式请求保护配置
type StreamGuardConfig struct {
	// 流式请求超时时间
	StreamTimeout time.Duration
	// 首字节超时时间
	FirstChunkTimeout time.Duration
	// 是否启用首字节锁定
	EnableFirstChunkLock bool
	// 清理间隔
	CleanupInterval time.Duration
}

// DefaultStreamGuardConfig 默认流式请求保护配置
func DefaultStreamGuardConfig() *StreamGuardConfig {
	return &StreamGuardConfig{
		StreamTimeout:        5 * time.Minute,
		FirstChunkTimeout:    3 * time.Second, // ⭐ 修复：根据架构设计，首包超时应为3秒
		EnableFirstChunkLock: true,
		CleanupInterval:      1 * time.Minute,
	}
}

// streamGuard 流式请求保护机制实现
type streamGuard struct {
	stateStore RuntimeStateStore
	config     *StreamGuardConfig
	mu         sync.RWMutex
}

// NewStreamGuard 创建流式请求保护机制
func NewStreamGuard(
	stateStore RuntimeStateStore,
	config *StreamGuardConfig,
) StreamGuard {
	if config == nil {
		config = DefaultStreamGuardConfig()
	}

	guard := &streamGuard{
		stateStore: stateStore,
		config:     config,
	}

	// 启动后台清理协程
	go guard.cleanupLoop()

	return guard
}

// StartStream 开始流式请求
// ⭐ 核心方法：创建流式请求上下文
func (sg *streamGuard) StartStream(ctx context.Context, providerID uint) (StreamContext, error) {
	// 生成唯一的流式请求ID
	streamID := uuid.New().String()

	// 创建流式请求上下文
	streamCtx := StreamContext{
		StreamID:   streamID,
		ProviderID: providerID,
		StartedAt:  time.Now(),
		Locked:     false,
	}

	// 存储到Redis
	err := sg.stateStore.StartStream(ctx, streamID, providerID)
	if err != nil {
		return StreamContext{}, fmt.Errorf("创建流式请求失败: %w", err)
	}

	return streamCtx, nil
}

// OnFirstChunk 首字节到达
// ⭐ 核心方法：首字节到达后锁定Provider
func (sg *streamGuard) OnFirstChunk(ctx context.Context, streamCtx StreamContext) error {
	if !sg.config.EnableFirstChunkLock {
		// 如果未启用首字节锁定，直接返回
		return nil
	}

	// 检查是否已经锁定
	locked, err := sg.stateStore.IsStreamLocked(ctx, streamCtx.StreamID)
	if err != nil {
		return fmt.Errorf("检查流式请求锁定状态失败: %w", err)
	}

	if locked {
		// 已经锁定，无需重复操作
		return nil
	}

	// 锁定流式请求
	err = sg.stateStore.OnFirstChunk(ctx, streamCtx.StreamID)
	if err != nil {
		return fmt.Errorf("锁定流式请求失败: %w", err)
	}

	return nil
}

// IsLocked 检查流式请求是否锁定
// ⭐ 核心方法：判断是否可以切换Provider
func (sg *streamGuard) IsLocked(ctx context.Context, streamCtx StreamContext) (bool, error) {
	if !sg.config.EnableFirstChunkLock {
		// 如果未启用首字节锁定，始终返回false（可以切换）
		return false, nil
	}

	// 检查流式请求是否锁定
	locked, err := sg.stateStore.IsStreamLocked(ctx, streamCtx.StreamID)
	if err != nil {
		return false, fmt.Errorf("检查流式请求锁定状态失败: %w", err)
	}

	return locked, nil
}

// EndStream 结束流式请求
func (sg *streamGuard) EndStream(ctx context.Context, streamCtx StreamContext, success bool) error {
	// 从Redis删除流式请求信息
	err := sg.stateStore.EndStream(ctx, streamCtx.StreamID, success)
	if err != nil {
		return fmt.Errorf("结束流式请求失败: %w", err)
	}

	return nil
}

// GetStreamInfo 获取流式请求信息
func (sg *streamGuard) GetStreamInfo(ctx context.Context, streamID string) (*StreamInfo, error) {
	return sg.stateStore.GetStreamInfo(ctx, streamID)
}

// CleanupExpiredStreams 清理过期的流式请求
func (sg *streamGuard) CleanupExpiredStreams(ctx context.Context) error {
	// 这个方法需要扫描所有流式请求，比较耗时
	// 实际实现中可以使用Redis的SCAN命令
	// 这里简化实现，依赖Redis的TTL自动过期
	return nil
}

// cleanupLoop 后台清理循环
func (sg *streamGuard) cleanupLoop() {
	ticker := time.NewTicker(sg.config.CleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		ctx := context.Background()
		_ = sg.CleanupExpiredStreams(ctx)
	}
}

/*
// ==================== 流式请求执行器 ====================

// StreamExecutor 流式请求执行器
type StreamExecutor func(ctx context.Context, providerID uint, writer interface{}) error

// StreamExecutionResult 流式请求执行结果
type StreamExecutionResult struct {
	Success        bool   `json:"success"`
	ProviderID     uint    `json:"provider_id"`
	StreamID       string  `json:"stream_id"`
	Error          error   `json:"error,omitempty"`
	AttemptCount   int     `json:"attempt_count"`
	TotalLatencyMs int64   `json:"total_latency_ms"`
	FirstChunkMs   int64   `json:"first_chunk_ms,omitempty"`
}

// ExecuteStreamWithFailover 执行流式请求（带故障转移）
// ⭐ 核心方法：实现Stream First-Chunk Failover逻辑
func (sg *streamGuard) ExecuteStreamWithFailover(
	ctx context.Context,
	providerID uint,
	executor StreamExecutor,
	writer interface{},
) (*StreamExecutionResult, error) {
	startTime := time.Now()

	result := &StreamExecutionResult{
		ProviderID:   providerID,
		AttemptCount: 0,
	}

	// 开始流式请求
	streamCtx, err := sg.StartStream(ctx, providerID)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.StreamID = streamCtx.StreamID
	result.AttemptCount++

	// 执行流式请求
	err = executor(ctx, providerID, writer)

	// 记录首字节时间
	firstChunkTime := time.Since(startTime).Milliseconds()
	result.FirstChunkMs = firstChunkTime

	if err != nil {
		// 请求失败
		result.Error = err

		// 检查是否可以切换Provider
		locked, checkErr := sg.IsLocked(ctx, streamCtx)
		if checkErr != nil {
			// 检查失败，直接返回错误
			_ = sg.EndStream(ctx, streamCtx, false)
			return result, err
		}

		if locked {
			// 已锁定，不能切换Provider
			_ = sg.EndStream(ctx, streamCtx, false)
			return result, fmt.Errorf("流式请求已锁定，无法切换Provider: %w", err)
		}

		// 未锁定，可以切换Provider
		// 结束当前流式请求
		_ = sg.EndStream(ctx, streamCtx, false)

		// 返回错误，由上层决定是否重试其他Provider
		return result, fmt.Errorf("流式请求失败（首字节前），可以切换Provider: %w", err)
	}

	// 请求成功
	result.Success = true
	result.TotalLatencyMs = time.Since(startTime).Milliseconds()

	// 结束流式请求
	_ = sg.EndStream(ctx, streamCtx, true)

	return result, nil
}

// ExecuteStreamWithFirstChunkDetection 执行流式请求（带首字节检测）
// ⭐ 核心方法：检测首字节并锁定
func (sg *streamGuard) ExecuteStreamWithFirstChunkDetection(
	ctx context.Context,
	providerID uint,
	executor StreamExecutor,
	writer interface{},
) (*StreamExecutionResult, error) {
	startTime := time.Now()

	result := &StreamExecutionResult{
		ProviderID:   providerID,
		AttemptCount: 0,
	}

	// 开始流式请求
	streamCtx, err := sg.StartStream(ctx, providerID)
	if err != nil {
		result.Error = err
		return result, err
	}

	result.StreamID = streamCtx.StreamID
	result.AttemptCount++

	// 创建首字节检测的执行器
	wrappedExecutor := func(ctx context.Context, providerID uint, writer interface{}) error {
		// 执行原始执行器
		err := executor(ctx, providerID, writer)

		// 无论成功或失败，都记录首字节
		_ = sg.OnFirstChunk(ctx, streamCtx)

		return err
	}

	// 执行流式请求
	err = wrappedExecutor(ctx, providerID, writer)

	// 记录首字节时间
	firstChunkTime := time.Since(startTime).Milliseconds()
	result.FirstChunkMs = firstChunkTime

	if err != nil {
		// 请求失败
		result.Error = err

		// 检查是否可以切换Provider
		locked, checkErr := sg.IsLocked(ctx, streamCtx)
		if checkErr != nil {
			// 检查失败，直接返回错误
			_ = sg.EndStream(ctx, streamCtx, false)
			return result, err
		}

		if locked {
			// 已锁定，不能切换Provider
			_ = sg.EndStream(ctx, streamCtx, false)
			return result, fmt.Errorf("流式请求已锁定，无法切换Provider: %w", err)
		}

		// 未锁定，可以切换Provider
		// 结束当前流式请求
		_ = sg.EndStream(ctx, streamCtx, false)

		// 返回错误，由上层决定是否重试其他Provider
		return result, fmt.Errorf("流式请求失败（首字节前），可以切换Provider: %w", err)
	}

	// 请求成功
	result.Success = true
	result.TotalLatencyMs = time.Since(startTime).Milliseconds()

	// 结束流式请求
	_ = sg.EndStream(ctx, streamCtx, true)

	return result, nil
}

*/

// ==================== 流式请求监控 ====================

// StreamMonitor 流式请求监控
type StreamMonitor struct {
	guard         StreamGuard
	activeStreams map[string]*StreamContext
	mu            sync.RWMutex
}

// NewStreamMonitor 创建流式请求监控
func NewStreamMonitor(guard StreamGuard) *StreamMonitor {
	return &StreamMonitor{
		guard:         guard,
		activeStreams: make(map[string]*StreamContext),
	}
}

// RegisterStream 注册流式请求
func (sm *StreamMonitor) RegisterStream(streamCtx StreamContext) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	sm.activeStreams[streamCtx.StreamID] = &streamCtx
}

// UnregisterStream 注销流式请求
func (sm *StreamMonitor) UnregisterStream(streamID string) {
	sm.mu.Lock()
	defer sm.mu.Unlock()

	delete(sm.activeStreams, streamID)
}

// GetActiveStreams 获取活跃的流式请求
func (sm *StreamMonitor) GetActiveStreams() []StreamContext {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	streams := make([]StreamContext, 0, len(sm.activeStreams))
	for _, ctx := range sm.activeStreams {
		streams = append(streams, *ctx)
	}

	return streams
}

// GetActiveStreamCount 获取活跃流式请求数量
func (sm *StreamMonitor) GetActiveStreamCount() int {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	return len(sm.activeStreams)
}

// GetStreamByProviderID 根据ProviderID获取流式请求
func (sm *StreamMonitor) GetStreamByProviderID(providerID uint) []StreamContext {
	sm.mu.RLock()
	defer sm.mu.RUnlock()

	streams := make([]StreamContext, 0)
	for _, ctx := range sm.activeStreams {
		if ctx.ProviderID == providerID {
			streams = append(streams, *ctx)
		}
	}

	return streams
}
