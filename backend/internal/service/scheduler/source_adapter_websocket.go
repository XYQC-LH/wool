package scheduler

import (
	"context"
	"fmt"

	"nexus-api/internal/model"
)

const webSocketTransportAdapterName = "websocket-transport"

type webSocketTransportAdapter struct{}

// NewWebSocketTransportAdapter 创建 WebSocket 传输适配器
func NewWebSocketTransportAdapter() SourceAdapter {
	return &webSocketTransportAdapter{}
}

func (a *webSocketTransportAdapter) Name() string {
	return webSocketTransportAdapterName
}

func (a *webSocketTransportAdapter) Match(operation string, provider *model.ModelProvider) bool {
	if provider == nil || provider.Channel == nil {
		return false
	}
	_ = model.NormalizeOperation(operation)
	return ResolveProviderTransport(provider) == SourceTransportWebSocket
}

func (a *webSocketTransportAdapter) Execute(ctx context.Context, request *AdapterRequest) (interface{}, error) {
	if request == nil || request.Provider == nil {
		return nil, fmt.Errorf("adapter request 无效")
	}
	if request.Executor == nil {
		return nil, fmt.Errorf("adapter executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, SourceTransportWebSocket)
	return request.Executor(execCtx, request.Provider)
}

func (a *webSocketTransportAdapter) ExecuteStream(ctx context.Context, request *AdapterStreamRequest) error {
	if request == nil || request.Provider == nil {
		return fmt.Errorf("adapter stream request 无效")
	}
	if request.Executor == nil {
		return fmt.Errorf("adapter stream executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, SourceTransportWebSocket)
	return request.Executor(execCtx, request.Provider, request.OnFirstChunk)
}

