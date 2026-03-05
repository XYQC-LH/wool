package scheduler

import (
	"context"
	"fmt"

	"nexus-api/internal/model"
)

const googleAdapterName = "google-compatible"

type googleAdapter struct{}

// NewGoogleAdapter 创建 Google 兼容适配器（骨架实现）
func NewGoogleAdapter() SourceAdapter {
	return &googleAdapter{}
}

func (a *googleAdapter) Name() string {
	return googleAdapterName
}

func (a *googleAdapter) Match(operation string, provider *model.ModelProvider) bool {
	if provider == nil || provider.Channel == nil {
		return false
	}
	_ = model.NormalizeOperation(operation)
	return ResolveProviderProtocol(provider) == SourceProtocolGoogle
}

func (a *googleAdapter) Execute(ctx context.Context, request *AdapterRequest) (interface{}, error) {
	if request == nil || request.Provider == nil {
		return nil, fmt.Errorf("adapter request 无效")
	}
	if request.Executor == nil {
		return nil, fmt.Errorf("adapter executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, ResolveProviderTransport(request.Provider))
	return request.Executor(execCtx, request.Provider)
}

func (a *googleAdapter) ExecuteStream(ctx context.Context, request *AdapterStreamRequest) error {
	if request == nil || request.Provider == nil {
		return fmt.Errorf("adapter stream request 无效")
	}
	if request.Executor == nil {
		return fmt.Errorf("adapter stream executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, ResolveProviderTransport(request.Provider))
	return request.Executor(execCtx, request.Provider, request.OnFirstChunk)
}

