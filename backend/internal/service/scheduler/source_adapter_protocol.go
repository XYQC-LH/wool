package scheduler

import (
	"context"
	"net/url"
	"strings"

	"nexus-api/internal/model"
)

// SourceProtocol 上游 API 协议类型
type SourceProtocol string

const (
	SourceProtocolOpenAICompatible SourceProtocol = "openai-compatible"
	SourceProtocolAzureOpenAI      SourceProtocol = "azure-openai"
	SourceProtocolAnthropic        SourceProtocol = "anthropic"
	SourceProtocolGoogle           SourceProtocol = "google"
)

// SourceTransport 上游传输协议
type SourceTransport string

const (
	SourceTransportHTTP      SourceTransport = "http"
	SourceTransportWebSocket SourceTransport = "websocket"
	SourceTransportGRPC      SourceTransport = "grpc"
)

type sourceTransportContextKey struct{}

// WithSourceTransport 将上游传输协议注入上下文，供执行器读取
func WithSourceTransport(ctx context.Context, transport SourceTransport) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := normalizeSourceTransport(transport)
	return context.WithValue(ctx, sourceTransportContextKey{}, normalized)
}

// SourceTransportFromContext 从上下文读取上游传输协议
func SourceTransportFromContext(ctx context.Context) SourceTransport {
	if ctx == nil {
		return SourceTransportHTTP
	}
	value := ctx.Value(sourceTransportContextKey{})
	transport, _ := value.(SourceTransport)
	return normalizeSourceTransport(transport)
}

// ResolveProviderProtocol 解析 Provider 协议（优先读取 channel.config，再回退 URL 规则）
func ResolveProviderProtocol(provider *model.ModelProvider) SourceProtocol {
	configProtocol := readProviderConfigString(provider, "source_protocol", "protocol", "upstream_protocol")
	if normalized, ok := normalizeSourceProtocol(configProtocol); ok {
		return normalized
	}

	host := parseProviderHost(provider)
	if strings.Contains(host, "openai.azure.com") {
		return SourceProtocolAzureOpenAI
	}
	if strings.Contains(host, "anthropic.com") {
		return SourceProtocolAnthropic
	}
	if strings.Contains(host, "generativelanguage.googleapis.com") || strings.Contains(host, "aiplatform.googleapis.com") {
		return SourceProtocolGoogle
	}

	return SourceProtocolOpenAICompatible
}

// ResolveProviderTransport 解析 Provider 传输协议（默认 HTTP）
func ResolveProviderTransport(provider *model.ModelProvider) SourceTransport {
	transport := readProviderConfigString(provider, "source_transport", "transport", "upstream_transport")
	return normalizeSourceTransport(SourceTransport(transport))
}

func parseProviderHost(provider *model.ModelProvider) string {
	if provider == nil || provider.Channel == nil {
		return ""
	}
	raw := strings.TrimSpace(provider.Channel.BaseURL)
	if raw == "" {
		return ""
	}

	parsed, err := url.Parse(raw)
	if err == nil && strings.TrimSpace(parsed.Hostname()) != "" {
		return strings.ToLower(strings.TrimSpace(parsed.Hostname()))
	}

	trimmed := strings.TrimPrefix(strings.TrimPrefix(raw, "https://"), "http://")
	if idx := strings.Index(trimmed, "/"); idx >= 0 {
		trimmed = trimmed[:idx]
	}
	return strings.ToLower(strings.TrimSpace(trimmed))
}

func readProviderConfigString(provider *model.ModelProvider, keys ...string) string {
	if provider == nil || provider.Channel == nil || provider.Channel.Config == nil || len(keys) == 0 {
		return ""
	}

	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		raw, ok := provider.Channel.Config[key]
		if !ok || raw == nil {
			continue
		}
		value, ok := raw.(string)
		if !ok {
			continue
		}
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}

	return ""
}

func normalizeSourceProtocol(raw string) (SourceProtocol, bool) {
	candidate := strings.ToLower(strings.TrimSpace(raw))
	switch candidate {
	case "", "default":
		return "", false
	case "openai", "openai-compatible", "openai_compatible":
		return SourceProtocolOpenAICompatible, true
	case "azure", "azure-openai", "azure_openai":
		return SourceProtocolAzureOpenAI, true
	case "anthropic", "claude":
		return SourceProtocolAnthropic, true
	case "google", "gemini", "vertexai", "vertex-ai":
		return SourceProtocolGoogle, true
	default:
		return "", false
	}
}

func normalizeSourceTransport(transport SourceTransport) SourceTransport {
	candidate := strings.ToLower(strings.TrimSpace(string(transport)))
	switch candidate {
	case "ws", "wss", "websocket":
		return SourceTransportWebSocket
	case "grpc", "grpcs":
		return SourceTransportGRPC
	default:
		return SourceTransportHTTP
	}
}

