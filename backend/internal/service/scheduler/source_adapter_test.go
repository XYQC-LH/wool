package scheduler

import (
	"context"
	"testing"

	"nexus-api/internal/model"

	"github.com/stretchr/testify/require"
)

type mockSourceAdapter struct {
	name    string
	matched bool
}

func (a *mockSourceAdapter) Name() string {
	return a.name
}

func (a *mockSourceAdapter) Match(_ string, _ *model.ModelProvider) bool {
	return a.matched
}

func (a *mockSourceAdapter) Execute(_ context.Context, _ *AdapterRequest) (interface{}, error) {
	return "ok", nil
}

func (a *mockSourceAdapter) ExecuteStream(_ context.Context, _ *AdapterStreamRequest) error {
	return nil
}

func TestResolveProviderProtocol(t *testing.T) {
	t.Run("config 优先", func(t *testing.T) {
		provider := &model.ModelProvider{
			Channel: &model.Channel{
				BaseURL: "https://api.openai.com",
				Config:  model.JSON{"protocol": "anthropic"},
			},
		}
		require.Equal(t, SourceProtocolAnthropic, ResolveProviderProtocol(provider))
	})

	t.Run("azure host 识别", func(t *testing.T) {
		provider := &model.ModelProvider{
			Channel: &model.Channel{
				BaseURL: "https://foo.openai.azure.com",
			},
		}
		require.Equal(t, SourceProtocolAzureOpenAI, ResolveProviderProtocol(provider))
	})

	t.Run("google host 识别", func(t *testing.T) {
		provider := &model.ModelProvider{
			Channel: &model.Channel{
				BaseURL: "https://generativelanguage.googleapis.com",
			},
		}
		require.Equal(t, SourceProtocolGoogle, ResolveProviderProtocol(provider))
	})

	t.Run("默认 openai", func(t *testing.T) {
		provider := &model.ModelProvider{
			Channel: &model.Channel{
				BaseURL: "https://api.openai.com",
			},
		}
		require.Equal(t, SourceProtocolOpenAICompatible, ResolveProviderProtocol(provider))
	})
}

func TestResolveProviderTransport(t *testing.T) {
	provider := &model.ModelProvider{
		Channel: &model.Channel{
			BaseURL: "https://api.openai.com",
			Config:  model.JSON{"transport": "grpc"},
		},
	}
	require.Equal(t, SourceTransportGRPC, ResolveProviderTransport(provider))

	provider.Channel.Config["transport"] = "websocket"
	require.Equal(t, SourceTransportWebSocket, ResolveProviderTransport(provider))

	provider.Channel.Config["transport"] = "http"
	require.Equal(t, SourceTransportHTTP, ResolveProviderTransport(provider))
}

func TestSourceAdapterRegistry_RegisterResolveList(t *testing.T) {
	registry := NewSourceAdapterRegistry().(*sourceAdapterRegistry)

	err := registry.Register(&mockSourceAdapter{name: "b-adapter", matched: false})
	require.NoError(t, err)
	err = registry.Register(&mockSourceAdapter{name: "a-adapter", matched: true})
	require.NoError(t, err)

	resolved := registry.Resolve(model.OperationChatCompletions, nil)
	require.NotNil(t, resolved)
	require.Equal(t, "a-adapter", resolved.Name())

	names := registry.List()
	require.Equal(t, []string{"a-adapter", "b-adapter"}, names)
}

func TestSourceAdapterRegistry_RegisterDuplicate(t *testing.T) {
	registry := NewSourceAdapterRegistry().(*sourceAdapterRegistry)
	require.NoError(t, registry.Register(&mockSourceAdapter{name: "dup-adapter", matched: true}))
	require.Error(t, registry.Register(&mockSourceAdapter{name: "dup-adapter", matched: true}))
}

func TestBuiltInAdapters_RespectTransportAndProtocol(t *testing.T) {
	registry := NewSourceAdapterRegistry(
		NewWebSocketTransportAdapter(),
		NewGRPCTransportAdapter(),
		NewAzureOpenAIAdapter(),
		NewAnthropicAdapter(),
		NewGoogleAdapter(),
		NewOpenAICompatibleAdapter(),
	)

	provider := &model.ModelProvider{
		Channel: &model.Channel{
			BaseURL: "https://api.openai.com",
			Config:  model.JSON{"transport": "websocket"},
		},
	}
	require.Equal(t, webSocketTransportAdapterName, registry.Resolve(model.OperationChatCompletions, provider).Name())

	provider.Channel.Config = model.JSON{"transport": "grpc"}
	require.Equal(t, grpcTransportAdapterName, registry.Resolve(model.OperationChatCompletions, provider).Name())

	provider.Channel.Config = model.JSON{"protocol": "azure-openai"}
	require.Equal(t, azureOpenAIAdapterName, registry.Resolve(model.OperationChatCompletions, provider).Name())

	provider.Channel.Config = model.JSON{"protocol": "anthropic"}
	require.Equal(t, anthropicAdapterName, registry.Resolve(model.OperationChatCompletions, provider).Name())

	provider.Channel.Config = model.JSON{"protocol": "google"}
	require.Equal(t, googleAdapterName, registry.Resolve(model.OperationChatCompletions, provider).Name())
}

func TestOpenAIAdapter_ExecuteStreamInjectsTransport(t *testing.T) {
	adapter := NewOpenAICompatibleAdapter()
	provider := &model.ModelProvider{
		Channel: &model.Channel{
			BaseURL: "https://api.openai.com",
			Config:  model.JSON{"transport": "grpc"},
		},
	}

	var actualTransport SourceTransport
	err := adapter.ExecuteStream(context.Background(), &AdapterStreamRequest{
		Operation: model.OperationChatCompletions,
		Provider:  provider,
		Executor: func(ctx context.Context, _ *model.ModelProvider, _ func()) error {
			actualTransport = SourceTransportFromContext(ctx)
			return nil
		},
	})
	require.NoError(t, err)
	require.Equal(t, SourceTransportGRPC, actualTransport)
}

func TestPluginSourceAdapter_Defaults(t *testing.T) {
	adapter, err := NewPluginSourceAdapter(SourceAdapterPluginOptions{
		Name: "custom-plugin",
	})
	require.NoError(t, err)
	require.NotNil(t, adapter)
	require.Equal(t, "custom-plugin", adapter.Name())
	require.True(t, adapter.Match(model.OperationChatCompletions, nil))

	_, err = adapter.Execute(context.Background(), &AdapterRequest{
		Provider: &model.ModelProvider{ID: 1},
		Executor: func(_ context.Context, _ *model.ModelProvider) (interface{}, error) {
			return "ok", nil
		},
	})
	require.NoError(t, err)
}

