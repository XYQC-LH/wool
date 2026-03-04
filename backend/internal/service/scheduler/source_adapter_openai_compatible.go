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
	return true
}

func (a *openAICompatibleAdapter) Execute(ctx context.Context, request *AdapterRequest) (interface{}, error) {
	if request == nil || request.Provider == nil {
		return nil, fmt.Errorf("adapter request 无效")
	}
	if request.Executor == nil {
		return nil, fmt.Errorf("adapter executor 不能为空")
	}

	// 骨架阶段：先透传到既有执行器，后续在此处替换为统一上游协议调用。
	return request.Executor(ctx, request.Provider)
}
