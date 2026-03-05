package handler

import (
	"strings"

	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

type gatewayControlHeaders struct {
	sessionID      string
	trafficTag     string
	experimentID   string
	idempotencyKey string
	forceCanary    bool
	enableCache    bool
	bypassCache    bool
	disableDedup   bool
}

func applyGatewayControlHeadersToChat(c *gin.Context, req *service.ChatCompletionRequest) {
	if c == nil || req == nil {
		return
	}
	headers := readGatewayControlHeaders(c)
	req.SessionID = firstNonEmpty(headers.sessionID, strings.TrimSpace(req.User))
	req.TrafficTag = headers.trafficTag
	req.ExperimentID = headers.experimentID
	req.IdempotencyKey = headers.idempotencyKey
	req.ForceCanary = headers.forceCanary
	req.EnableCache = headers.enableCache
	req.BypassCache = headers.bypassCache
	req.DisableDedup = headers.disableDedup
}

func applyGatewayControlHeadersToCompletion(c *gin.Context, req *service.CompletionRequest) {
	if c == nil || req == nil {
		return
	}
	headers := readGatewayControlHeaders(c)
	req.SessionID = firstNonEmpty(headers.sessionID, strings.TrimSpace(req.User))
	req.TrafficTag = headers.trafficTag
	req.ExperimentID = headers.experimentID
	req.IdempotencyKey = headers.idempotencyKey
	req.ForceCanary = headers.forceCanary
	req.EnableCache = headers.enableCache
	req.BypassCache = headers.bypassCache
	req.DisableDedup = headers.disableDedup
}

func applyGatewayControlHeadersToEmbedding(c *gin.Context, req *service.EmbeddingRequest) {
	if c == nil || req == nil {
		return
	}
	headers := readGatewayControlHeaders(c)
	req.SessionID = firstNonEmpty(headers.sessionID, strings.TrimSpace(req.User))
	req.TrafficTag = headers.trafficTag
	req.ExperimentID = headers.experimentID
	req.IdempotencyKey = headers.idempotencyKey
	req.ForceCanary = headers.forceCanary
	req.EnableCache = headers.enableCache
	req.BypassCache = headers.bypassCache
	req.DisableDedup = headers.disableDedup
}

func readGatewayControlHeaders(c *gin.Context) gatewayControlHeaders {
	if c == nil {
		return gatewayControlHeaders{}
	}

	cacheControl := strings.ToLower(strings.TrimSpace(c.GetHeader("Cache-Control")))
	bypassByCacheControl := strings.Contains(cacheControl, "no-cache") || strings.Contains(cacheControl, "no-store")

	return gatewayControlHeaders{
		sessionID:      strings.TrimSpace(c.GetHeader("X-Session-ID")),
		trafficTag:     strings.ToLower(strings.TrimSpace(c.GetHeader("X-Traffic-Tag"))),
		experimentID:   strings.ToLower(strings.TrimSpace(c.GetHeader("X-Experiment-ID"))),
		idempotencyKey: strings.TrimSpace(c.GetHeader("Idempotency-Key")),
		forceCanary:    parseBoolHeader(c.GetHeader("X-Canary")),
		enableCache:    parseBoolHeader(c.GetHeader("X-Cache-Enabled")),
		bypassCache:    bypassByCacheControl || parseBoolHeader(c.GetHeader("X-Bypass-Cache")),
		disableDedup:   parseBoolHeader(c.GetHeader("X-Disable-Dedup")),
	}
}

func parseBoolHeader(raw string) bool {
	raw = strings.TrimSpace(strings.ToLower(raw))
	switch raw {
	case "1", "true", "yes", "on", "y":
		return true
	default:
		return false
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}
