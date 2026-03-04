package scheduler

import (
	"context"
	"time"
)

// CommitGuard 提交点门控（Commit Point Guard）。
//
// 现阶段：复用 StreamGuard 的“首包前可切换、首包后锁定”实现，作为流式请求的提交点门控。
// 下一阶段（按核心架构设计.md）：扩展为同步/异步 Job 的提交点语义。
type CommitGuard interface {
	StreamGuard

	// ==================== 异步 Job 提交点 ====================

	// GetJobCommit 获取 Job 的提交锁（不存在返回 nil）
	GetJobCommit(ctx context.Context, jobID string) (*JobCommitInfo, error)
	// EnsureJobCommit 确保 Job 的提交锁存在（幂等）
	EnsureJobCommit(ctx context.Context, jobID string, providerID uint, instanceID uint, ttl time.Duration) (*JobCommitInfo, error)
	// DeleteJobCommit 删除 Job 的提交锁
	DeleteJobCommit(ctx context.Context, jobID string) error
}

type commitGuard struct {
	streamGuard StreamGuard
	stateStore  RuntimeStateStore
}

// NewCommitGuard 创建 CommitGuard。
//
// 当前实现：
// - 流式提交点：复用 StreamGuard 首包锁定
// - Job 提交点：使用 RuntimeStateStore 存储 job commit（provider/instance 锁定）
func NewCommitGuard(stateStore RuntimeStateStore, config *StreamGuardConfig) CommitGuard {
	return &commitGuard{
		streamGuard: NewStreamGuard(stateStore, config),
		stateStore:  stateStore,
	}
}

func (g *commitGuard) StartStream(ctx context.Context, providerID uint) (StreamContext, error) {
	return g.streamGuard.StartStream(ctx, providerID)
}

func (g *commitGuard) OnFirstChunk(ctx context.Context, streamCtx StreamContext) error {
	return g.streamGuard.OnFirstChunk(ctx, streamCtx)
}

func (g *commitGuard) IsLocked(ctx context.Context, streamCtx StreamContext) (bool, error) {
	return g.streamGuard.IsLocked(ctx, streamCtx)
}

func (g *commitGuard) EndStream(ctx context.Context, streamCtx StreamContext, success bool) error {
	return g.streamGuard.EndStream(ctx, streamCtx, success)
}

func (g *commitGuard) GetStreamInfo(ctx context.Context, streamID string) (*StreamInfo, error) {
	return g.streamGuard.GetStreamInfo(ctx, streamID)
}

func (g *commitGuard) CleanupExpiredStreams(ctx context.Context) error {
	return g.streamGuard.CleanupExpiredStreams(ctx)
}

func (g *commitGuard) GetJobCommit(ctx context.Context, jobID string) (*JobCommitInfo, error) {
	if g == nil || g.stateStore == nil {
		return nil, nil
	}
	return g.stateStore.GetJobCommit(ctx, jobID)
}

func (g *commitGuard) EnsureJobCommit(ctx context.Context, jobID string, providerID uint, instanceID uint, ttl time.Duration) (*JobCommitInfo, error) {
	if g == nil || g.stateStore == nil {
		return nil, nil
	}
	return g.stateStore.EnsureJobCommit(ctx, jobID, providerID, instanceID, ttl)
}

func (g *commitGuard) DeleteJobCommit(ctx context.Context, jobID string) error {
	if g == nil || g.stateStore == nil {
		return nil
	}
	return g.stateStore.DeleteJobCommit(ctx, jobID)
}
