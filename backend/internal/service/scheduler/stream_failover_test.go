package scheduler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"nexus-api/internal/model"
)

type fakeStreamGuard struct {
	mu     sync.Mutex
	nextID int
	locked map[string]bool
}

func newFakeStreamGuard() *fakeStreamGuard {
	return &fakeStreamGuard{
		locked: make(map[string]bool),
	}
}

func (g *fakeStreamGuard) StartStream(ctx context.Context, providerID uint) (StreamContext, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	g.nextID++
	streamID := fmt.Sprintf("test-stream-%d", g.nextID)
	g.locked[streamID] = false
	return StreamContext{
		StreamID:   streamID,
		ProviderID: providerID,
		StartedAt:  time.Now(),
		Locked:     false,
	}, nil
}

func (g *fakeStreamGuard) OnFirstChunk(ctx context.Context, streamCtx StreamContext) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	if _, ok := g.locked[streamCtx.StreamID]; ok {
		g.locked[streamCtx.StreamID] = true
	}
	return nil
}

func (g *fakeStreamGuard) IsLocked(ctx context.Context, streamCtx StreamContext) (bool, error) {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.locked[streamCtx.StreamID], nil
}

func (g *fakeStreamGuard) EndStream(ctx context.Context, streamCtx StreamContext, success bool) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	delete(g.locked, streamCtx.StreamID)
	return nil
}

func (g *fakeStreamGuard) GetStreamInfo(ctx context.Context, streamID string) (*StreamInfo, error) {
	return nil, nil
}

func (g *fakeStreamGuard) CleanupExpiredStreams(ctx context.Context) error {
	return nil
}

func (g *fakeStreamGuard) GetJobCommit(ctx context.Context, jobID string) (*JobCommitInfo, error) {
	return nil, nil
}

func (g *fakeStreamGuard) EnsureJobCommit(ctx context.Context, jobID string, providerID uint, instanceID uint, ttl time.Duration) (*JobCommitInfo, error) {
	return nil, nil
}

func (g *fakeStreamGuard) DeleteJobCommit(ctx context.Context, jobID string) error {
	return nil
}

func TestExecuteStreamWithGuard_PreFirstChunkError_AllowsFailover(t *testing.T) {
	guard := newFakeStreamGuard()
	controller := &cascadeController{
		commitGuard: guard,
		config: &CascadeConfig{
			Timeout: 1 * time.Second,
		},
	}

	provider := &model.ModelProvider{
		ID:                        1,
		StreamFirstChunkTimeoutMs: 50,
		AttemptTimeoutMs:          500,
	}

	executor := func(ctx context.Context, p *model.ModelProvider, onFirstChunk func()) error {
		return errors.New("upstream failed")
	}

	canFailover, err := controller.executeStreamWithGuard(context.Background(), provider, executor)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !canFailover {
		t.Fatalf("expected canFailover=true")
	}
}

func TestExecuteStreamWithGuard_PostFirstChunkError_DisallowsFailover(t *testing.T) {
	guard := newFakeStreamGuard()
	controller := &cascadeController{
		commitGuard: guard,
		config: &CascadeConfig{
			Timeout: 1 * time.Second,
		},
	}

	provider := &model.ModelProvider{
		ID:                        1,
		StreamFirstChunkTimeoutMs: 50,
		AttemptTimeoutMs:          500,
	}

	executor := func(ctx context.Context, p *model.ModelProvider, onFirstChunk func()) error {
		onFirstChunk()
		return errors.New("failed after first chunk")
	}

	canFailover, err := controller.executeStreamWithGuard(context.Background(), provider, executor)
	if err == nil {
		t.Fatalf("expected error")
	}
	if canFailover {
		t.Fatalf("expected canFailover=false")
	}
}

func TestExecuteStreamWithGuard_FirstChunkTimeout_AllowsFailover(t *testing.T) {
	guard := newFakeStreamGuard()
	controller := &cascadeController{
		commitGuard: guard,
		config: &CascadeConfig{
			Timeout: 1 * time.Second,
		},
	}

	provider := &model.ModelProvider{
		ID:                        1,
		StreamFirstChunkTimeoutMs: 30,
		AttemptTimeoutMs:          500,
	}

	executor := func(ctx context.Context, p *model.ModelProvider, onFirstChunk func()) error {
		<-ctx.Done()
		return ctx.Err()
	}

	canFailover, err := controller.executeStreamWithGuard(context.Background(), provider, executor)
	if err == nil {
		t.Fatalf("expected error")
	}
	if !strings.Contains(err.Error(), "首包超时") {
		t.Fatalf("expected first-chunk timeout error, got: %v", err)
	}
	if !canFailover {
		t.Fatalf("expected canFailover=true")
	}
}

func TestExecuteStreamWithGuard_Success_ReturnsNil(t *testing.T) {
	guard := newFakeStreamGuard()
	controller := &cascadeController{
		commitGuard: guard,
		config: &CascadeConfig{
			Timeout: 1 * time.Second,
		},
	}

	provider := &model.ModelProvider{
		ID:                        1,
		StreamFirstChunkTimeoutMs: 50,
		AttemptTimeoutMs:          500,
	}

	executor := func(ctx context.Context, p *model.ModelProvider, onFirstChunk func()) error {
		onFirstChunk()
		return nil
	}

	canFailover, err := controller.executeStreamWithGuard(context.Background(), provider, executor)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
	if canFailover {
		t.Fatalf("expected canFailover=false")
	}
}
