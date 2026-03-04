package scheduler

import (
	"context"

	"nexus-api/internal/model"
)

// AdapterRequest 调度链路传递给 SourceAdapter 的请求上下文
type AdapterRequest struct {
	Operation string
	Provider  *model.ModelProvider
	Executor  RequestExecutor
}

// SourceAdapter Source Layer 适配器接口
type SourceAdapter interface {
	Name() string
	Match(operation string, provider *model.ModelProvider) bool
	Execute(ctx context.Context, request *AdapterRequest) (interface{}, error)
}

// SourceAdapterRegistry 适配器注册表
type SourceAdapterRegistry interface {
	Resolve(operation string, provider *model.ModelProvider) SourceAdapter
}

type sourceAdapterRegistry struct {
	adapters []SourceAdapter
}

// NewSourceAdapterRegistry 创建适配器注册表
func NewSourceAdapterRegistry(adapters ...SourceAdapter) SourceAdapterRegistry {
	copied := make([]SourceAdapter, 0, len(adapters))
	for _, adapter := range adapters {
		if adapter != nil {
			copied = append(copied, adapter)
		}
	}
	return &sourceAdapterRegistry{
		adapters: copied,
	}
}

func (r *sourceAdapterRegistry) Resolve(operation string, provider *model.ModelProvider) SourceAdapter {
	if r == nil || len(r.adapters) == 0 {
		return nil
	}

	for _, adapter := range r.adapters {
		if adapter == nil {
			continue
		}
		if adapter.Match(operation, provider) {
			return adapter
		}
	}
	return nil
}
