package scheduler

import (
	"context"
	"fmt"
	"strings"

	"nexus-api/internal/model"
)

const openAICompatibleAdapterName = "openai-compatible"

type openAICompatibleAdapter struct{}

// NewOpenAICompatibleAdapter 创建 OpenAI 兼容适配器（骨架实现）
func NewOpenAICompatibleAdapter() SourceAdapter {
	return &openAICompatibleAdapter{}
}

func (a *openAICompatibleAdapter) Name() string {
	return openAICompatibleAdapterName
}

func (a *openAICompatibleAdapter) Match(operation string, provider *model.ModelProvider) bool {
	if provider == nil || provider.Channel == nil {
		return false
	}
	if strings.TrimSpace(provider.Channel.BaseURL) == "" {
		return false
	}
	if model.NormalizeOperation(operation) == "" {
		return false
	}
	return ResolveProviderProtocol(provider) == SourceProtocolOpenAICompatible
}

func (a *openAICompatibleAdapter) Execute(ctx context.Context, request *AdapterRequest) (interface{}, error) {
	if request == nil || request.Provider == nil {
		return nil, fmt.Errorf("adapter request 无效")
	}
	if request.Executor == nil {
		return nil, fmt.Errorf("adapter executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, ResolveProviderTransport(request.Provider))
	return request.Executor(execCtx, request.Provider)
}

func (a *openAICompatibleAdapter) ExecuteStream(ctx context.Context, request *AdapterStreamRequest) error {
	if request == nil || request.Provider == nil {
		return fmt.Errorf("adapter stream request 无效")
	}
	if request.Executor == nil {
		return fmt.Errorf("adapter stream executor 不能为空")
	}

	execCtx := WithSourceTransport(ctx, ResolveProviderTransport(request.Provider))
	return request.Executor(execCtx, request.Provider, request.OnFirstChunk)
}
