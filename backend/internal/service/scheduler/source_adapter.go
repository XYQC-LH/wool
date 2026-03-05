package scheduler

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"

	"nexus-api/internal/model"
)

// AdapterRequest 调度链路传递给 SourceAdapter 的请求上下文
type AdapterRequest struct {
	Operation string
	Provider  *model.ModelProvider
	Executor  RequestExecutor
}

// AdapterStreamRequest 调度链路传递给 SourceAdapter 的流式请求上下文
type AdapterStreamRequest struct {
	Operation    string
	Provider     *model.ModelProvider
	Executor     StreamExecutor
	OnFirstChunk func()
}

// SourceAdapter Source Layer 适配器接口
type SourceAdapter interface {
	Name() string
	Match(operation string, provider *model.ModelProvider) bool
	Execute(ctx context.Context, request *AdapterRequest) (interface{}, error)
	ExecuteStream(ctx context.Context, request *AdapterStreamRequest) error
}

// SourceAdapterRegistry 适配器注册表
type SourceAdapterRegistry interface {
	Register(adapter SourceAdapter) error
	Resolve(operation string, provider *model.ModelProvider) SourceAdapter
	List() []string
}

type sourceAdapterRegistry struct {
	mu       sync.RWMutex
	adapters []SourceAdapter
}

// NewSourceAdapterRegistry 创建适配器注册表
func NewSourceAdapterRegistry(adapters ...SourceAdapter) SourceAdapterRegistry {
	registry := &sourceAdapterRegistry{
		adapters: make([]SourceAdapter, 0, len(adapters)),
	}

	for _, adapter := range adapters {
		if err := registry.Register(adapter); err != nil {
			continue
		}
	}
	return registry
}

func (r *sourceAdapterRegistry) Register(adapter SourceAdapter) error {
	if r == nil {
		return fmt.Errorf("source adapter registry 未初始化")
	}
	if adapter == nil {
		return fmt.Errorf("source adapter 不能为空")
	}
	name := strings.TrimSpace(adapter.Name())
	if name == "" {
		return fmt.Errorf("source adapter 名称不能为空")
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for _, current := range r.adapters {
		if current == nil {
			continue
		}
		if strings.EqualFold(strings.TrimSpace(current.Name()), name) {
			return fmt.Errorf("source adapter 重复注册: %s", name)
		}
	}

	r.adapters = append(r.adapters, adapter)
	return nil
}

func (r *sourceAdapterRegistry) Resolve(operation string, provider *model.ModelProvider) SourceAdapter {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	adapters := make([]SourceAdapter, len(r.adapters))
	copy(adapters, r.adapters)
	r.mu.RUnlock()
	if len(adapters) == 0 {
		return nil
	}

	for _, adapter := range adapters {
		if adapter == nil {
			continue
		}
		if adapter.Match(operation, provider) {
			return adapter
		}
	}
	return nil
}

func (r *sourceAdapterRegistry) List() []string {
	if r == nil {
		return nil
	}

	r.mu.RLock()
	defer r.mu.RUnlock()
	if len(r.adapters) == 0 {
		return nil
	}

	names := make([]string, 0, len(r.adapters))
	for _, adapter := range r.adapters {
		if adapter == nil {
			continue
		}
		name := strings.TrimSpace(adapter.Name())
		if name == "" {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
