package scheduler

import (
	"context"
	"fmt"
	"strings"

	"nexus-api/internal/model"
)

// SourceAdapterPluginOptions 插件化 SourceAdapter 构建参数
type SourceAdapterPluginOptions struct {
	Name          string
	Match         func(operation string, provider *model.ModelProvider) bool
	Execute       func(ctx context.Context, request *AdapterRequest) (interface{}, error)
	ExecuteStream func(ctx context.Context, request *AdapterStreamRequest) error
}

type pluginSourceAdapter struct {
	name          string
	match         func(operation string, provider *model.ModelProvider) bool
	execute       func(ctx context.Context, request *AdapterRequest) (interface{}, error)
	executeStream func(ctx context.Context, request *AdapterStreamRequest) error
}

// NewPluginSourceAdapter 创建插件化 SourceAdapter
func NewPluginSourceAdapter(options SourceAdapterPluginOptions) (SourceAdapter, error) {
	name := strings.TrimSpace(options.Name)
	if name == "" {
		return nil, fmt.Errorf("plugin source adapter name 不能为空")
	}

	adapter := &pluginSourceAdapter{
		name:  name,
		match: options.Match,
	}

	if options.Execute != nil {
		adapter.execute = options.Execute
	} else {
		adapter.execute = func(ctx context.Context, request *AdapterRequest) (interface{}, error) {
			if request == nil || request.Provider == nil {
				return nil, fmt.Errorf("adapter request 无效")
			}
			if request.Executor == nil {
				return nil, fmt.Errorf("adapter executor 不能为空")
			}
			return request.Executor(ctx, request.Provider)
		}
	}

	if options.ExecuteStream != nil {
		adapter.executeStream = options.ExecuteStream
	} else {
		adapter.executeStream = func(ctx context.Context, request *AdapterStreamRequest) error {
			if request == nil || request.Provider == nil {
				return fmt.Errorf("adapter stream request 无效")
			}
			if request.Executor == nil {
				return fmt.Errorf("adapter stream executor 不能为空")
			}
			return request.Executor(ctx, request.Provider, request.OnFirstChunk)
		}
	}

	return adapter, nil
}

func (a *pluginSourceAdapter) Name() string {
	if a == nil {
		return ""
	}
	return a.name
}

func (a *pluginSourceAdapter) Match(operation string, provider *model.ModelProvider) bool {
	if a == nil {
		return false
	}
	if a.match == nil {
		return true
	}
	return a.match(operation, provider)
}

func (a *pluginSourceAdapter) Execute(ctx context.Context, request *AdapterRequest) (interface{}, error) {
	if a == nil || a.execute == nil {
		return nil, fmt.Errorf("adapter execute 未实现")
	}
	return a.execute(ctx, request)
}

func (a *pluginSourceAdapter) ExecuteStream(ctx context.Context, request *AdapterStreamRequest) error {
	if a == nil || a.executeStream == nil {
		return fmt.Errorf("adapter stream execute 未实现")
	}
	return a.executeStream(ctx, request)
}

