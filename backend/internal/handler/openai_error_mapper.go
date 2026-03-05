package handler

import (
	"errors"
	"net/http"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/service"
	"nexus-api/internal/service/scheduler"

	"github.com/gin-gonic/gin"
)

func writeOpenAIError(c *gin.Context, status int, message string, errType string, code *string) {
	if c == nil {
		return
	}
	c.JSON(status, model.NewOpenAIError(message, errType, code))
}

// WriteOpenAIError 将内部错误映射为 OpenAI 兼容错误体与合适的 HTTP 状态码。
//
// 适用范围：/v1/* 兼容接口（gateway/generation/audio 等），不要用于 /api/admin/*。
func WriteOpenAIError(c *gin.Context, err error) {
	if err == nil {
		writeOpenAIError(c, http.StatusInternalServerError, "unknown error", model.OpenAIErrorTypeServer, nil)
		return
	}

	// 1) 明确的类型错误优先
	var insufficientFunds *service.InsufficientFundsError
	if errors.As(err, &insufficientFunds) {
		code := "insufficient_funds"
		writeOpenAIError(c, http.StatusPaymentRequired, insufficientFunds.Error(), model.OpenAIErrorTypeInvalidRequest, &code)
		return
	}

	var quotaExceeded *service.QuotaExceededError
	if errors.As(err, &quotaExceeded) {
		code := "quota_exceeded"
		writeOpenAIError(c, http.StatusTooManyRequests, quotaExceeded.Error(), model.OpenAIErrorTypeRateLimit, &code)
		return
	}

	var tenantQuotaExceeded *service.TenantQuotaExceededError
	if errors.As(err, &tenantQuotaExceeded) {
		code := "tenant_quota_exceeded"
		writeOpenAIError(c, http.StatusTooManyRequests, tenantQuotaExceeded.Error(), model.OpenAIErrorTypeRateLimit, &code)
		return
	}

	var upstreamAuth *service.UpstreamAuthError
	if errors.As(err, &upstreamAuth) {
		code := "upstream_auth_failed"
		writeOpenAIError(c, http.StatusBadGateway, upstreamAuth.Error(), model.OpenAIErrorTypeServer, &code)
		return
	}

	var upstreamRateLimited *service.UpstreamRateLimitedError
	if errors.As(err, &upstreamRateLimited) {
		code := "upstream_rate_limited"
		writeOpenAIError(c, http.StatusTooManyRequests, upstreamRateLimited.Error(), model.OpenAIErrorTypeRateLimit, &code)
		return
	}

	var upstreamInsufficientFunds *service.UpstreamInsufficientFundsError
	if errors.As(err, &upstreamInsufficientFunds) {
		code := "upstream_insufficient_funds"
		writeOpenAIError(c, http.StatusBadGateway, upstreamInsufficientFunds.Error(), model.OpenAIErrorTypeServer, &code)
		return
	}

	var notSupported *scheduler.ModelOperationNotSupportedError
	if errors.As(err, &notSupported) {
		code := "operation_not_supported"
		writeOpenAIError(c, http.StatusBadRequest, "模型不支持该能力: "+notSupported.ModelID, model.OpenAIErrorTypeInvalidRequest, &code)
		return
	}

	var noProviders *scheduler.NoAvailableProviderError
	if errors.As(err, &noProviders) {
		code := "no_available_provider"
		writeOpenAIError(c, http.StatusBadRequest, "未配置可用源头: "+noProviders.ModelID, model.OpenAIErrorTypeInvalidRequest, &code)
		return
	}

	var providerLimited *scheduler.ProviderRateLimitedError
	if errors.As(err, &providerLimited) {
		code := "provider_rate_limited"
		writeOpenAIError(c, http.StatusTooManyRequests, err.Error(), model.OpenAIErrorTypeRateLimit, &code)
		return
	}

	if errors.Is(err, service.ErrIdempotencyConflict) {
		code := "idempotency_key_conflict"
		writeOpenAIError(c, http.StatusConflict, err.Error(), model.OpenAIErrorTypeInvalidRequest, &code)
		return
	}

	// 2) 兼容旧实现：基于消息文本归类（逐步收敛到强类型错误）
	msg := err.Error()

	if strings.Contains(msg, "限流") || strings.Contains(msg, "RPM") || strings.Contains(msg, "TPM") {
		code := "rate_limited"
		writeOpenAIError(c, http.StatusTooManyRequests, msg, model.OpenAIErrorTypeRateLimit, &code)
		return
	}

	// 3) 兜底：服务端错误
	writeOpenAIError(c, http.StatusInternalServerError, msg, model.OpenAIErrorTypeServer, nil)
}
