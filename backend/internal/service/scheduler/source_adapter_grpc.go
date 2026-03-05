package scheduler

import (
	"context"
	"fmt"

	"nexus-api/internal/model"
)

const grpcTransportAdapterName = "grpc-transport"

type grpcTransportAdapter struct{}

// NewGRPCTransportAdapter 创建 gRPC 传输适配器
func NewGRPCTransportAdapter() SourceAdapter {
	return &grpcTransportAdapter{}
}

func (a *grpcTransportAdapter) Name() string {
	return grpcTransportAdapterName
}

func (a *grpcTransportAdapter) Match(operation string, provider *model.ModelProvider) bool {
	if provider == nil || provider.Channel == nil {
		return false
	}
	_ = model.NormalizeOperation(operation)
	return ResolveProviderTransport(provider) == SourceTransportGRPC
}

func (a *grpcTransportAdapter) Execute(ctx context.Context, request *AdapterRequest) (interface{}, error) {
	if request == nil || request.Provider == nil {
		return nil, fmt.Errorf("adapter request 无效")
	}
	if request.Executor == nil {
		return nil, fmt.Errorf("adapter executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, SourceTransportGRPC)
	return request.Executor(execCtx, request.Provider)
}

func (a *grpcTransportAdapter) ExecuteStream(ctx context.Context, request *AdapterStreamRequest) error {
	if request == nil || request.Provider == nil {
		return fmt.Errorf("adapter stream request 无效")
	}
	if request.Executor == nil {
		return fmt.Errorf("adapter stream executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, SourceTransportGRPC)
	return request.Executor(execCtx, request.Provider, request.OnFirstChunk)
}

