package scheduler

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"
)

// ErrorClassifier 错误分类器接口
// 判断错误是否为可重试错误，区分临时性错误和永久性错误
type ErrorClassifier interface {
	// IsRetryable 判断错误是否可重试
	IsRetryable(err error) bool
	// IsTemporary 判断错误是否为临时性错误
	IsTemporary(err error) bool
	// ClassifyError 分类错误
	ClassifyError(err error) ErrorType
	// GetRetryDelay 获取重试延迟
	GetRetryDelay(err error, attempt int) time.Duration
	// ShouldCircuitBreak 判断是否应该触发熔断
	ShouldCircuitBreak(err error) bool
}

// ErrorType 错误类型
type ErrorType string

const (
	ErrorTypeTemporary  ErrorType = "temporary"  // 临时性错误（可重试）
	ErrorTypePermanent  ErrorType = "permanent"  // 永久性错误（不可重试）
	ErrorTypeRateLimit  ErrorType = "rate_limit" // 速率限制错误
	ErrorTypeTimeout    ErrorType = "timeout"    // 超时错误
	ErrorTypeNetwork    ErrorType = "network"    // 网络错误
	ErrorTypeAuth       ErrorType = "auth"       // 认证错误
	ErrorTypeValidation ErrorType = "validation" // 验证错误
	ErrorTypeUnknown    ErrorType = "unknown"    // 未知错误
)

// ErrorClassifierConfig 错误分类器配置
type ErrorClassifierConfig struct {
	// 默认重试延迟
	DefaultRetryDelay time.Duration
	// 最大重试延迟
	MaxRetryDelay time.Duration
	// 重试延迟倍数
	RetryDelayMultiplier float64
	// 网络错误重试次数
	NetworkErrorMaxRetries int
	// 超时错误重试次数
	TimeoutErrorMaxRetries int
	// 速率限制错误重试次数
	RateLimitErrorMaxRetries int
	// 是否对速率限制错误触发熔断
	CircuitBreakOnRateLimit bool
}

// DefaultErrorClassifierConfig 默认错误分类器配置
func DefaultErrorClassifierConfig() *ErrorClassifierConfig {
	return &ErrorClassifierConfig{
		DefaultRetryDelay:        100 * time.Millisecond,
		MaxRetryDelay:            5 * time.Second,
		RetryDelayMultiplier:     2.0,
		NetworkErrorMaxRetries:   3,
		TimeoutErrorMaxRetries:   2,
		RateLimitErrorMaxRetries: 1,
		CircuitBreakOnRateLimit:  true,
	}
}

// errorClassifier 错误分类器实现
type errorClassifier struct {
	config *ErrorClassifierConfig
}

// NewErrorClassifier 创建错误分类器
func NewErrorClassifier(config *ErrorClassifierConfig) ErrorClassifier {
	if config == nil {
		config = DefaultErrorClassifierConfig()
	}
	return &errorClassifier{
		config: config,
	}
}

// IsRetryable 判断错误是否可重试
// ⭐ 核心方法：根据错误类型判断是否可以重试
func (ec *errorClassifier) IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	errorType := ec.ClassifyError(err)

	switch errorType {
	case ErrorTypeTemporary, ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeRateLimit:
		return true
	case ErrorTypePermanent, ErrorTypeAuth, ErrorTypeValidation:
		return false
	case ErrorTypeUnknown:
		// 未知错误默认可重试一次
		return true
	default:
		return false
	}
}

// IsTemporary 判断错误是否为临时性错误
func (ec *errorClassifier) IsTemporary(err error) bool {
	if err == nil {
		return false
	}

	errorType := ec.ClassifyError(err)
	return errorType == ErrorTypeTemporary || errorType == ErrorTypeNetwork || errorType == ErrorTypeTimeout
}

// ClassifyError 分类错误
// ⭐ 核心方法：根据错误消息和类型进行分类
func (ec *errorClassifier) ClassifyError(err error) ErrorType {
	if err == nil {
		return ErrorTypeUnknown
	}

	// 1) 强类型/结构化错误优先（减少基于字符串的误判）
	{
		var providerLimited *ProviderRateLimitedError
		if errors.As(err, &providerLimited) {
			return ErrorTypeRateLimit
		}

		var notSupported *ModelOperationNotSupportedError
		if errors.As(err, &notSupported) {
			return ErrorTypeValidation
		}

		var noProviders *NoAvailableProviderError
		if errors.As(err, &noProviders) {
			return ErrorTypePermanent
		}

		// 上游 HTTP 错误：由 service 层强类型错误提供 StatusCode（避免依赖 err.Error() 里是否包含 "401"/"429" 等文本）。
		var upstream interface{ UpstreamStatusCode() int }
		if errors.As(err, &upstream) {
			switch upstream.UpstreamStatusCode() {
			case http.StatusTooManyRequests:
				return ErrorTypeRateLimit
			case http.StatusUnauthorized, http.StatusForbidden:
				return ErrorTypeAuth
			case http.StatusPaymentRequired:
				// 上游余额不足/资源耗尽：对网关而言属于配置/资源问题，通常不应重试同一 provider
				return ErrorTypePermanent
			}
		}
	}

	errMsg := strings.ToLower(err.Error())

	// 检查网络错误
	if ec.isNetworkError(err) {
		return ErrorTypeNetwork
	}

	// 检查超时错误
	if ec.isTimeoutError(err) {
		return ErrorTypeTimeout
	}

	// 检查速率限制错误
	if ec.isRateLimitError(errMsg) {
		return ErrorTypeRateLimit
	}

	// 检查认证错误
	if ec.isAuthError(errMsg) {
		return ErrorTypeAuth
	}

	// 检查验证错误
	if ec.isValidationError(errMsg) {
		return ErrorTypeValidation
	}

	// 检查临时性错误
	if ec.isTemporaryError(errMsg) {
		return ErrorTypeTemporary
	}

	// 检查永久性错误
	if ec.isPermanentError(errMsg) {
		return ErrorTypePermanent
	}

	// 默认为未知错误
	return ErrorTypeUnknown
}

// isNetworkError 检查是否为网络错误
func (ec *errorClassifier) isNetworkError(err error) bool {
	// 检查是否为net.Error
	var netErr net.Error
	if errors.As(err, &netErr) {
		// 超时错误
		if netErr.Timeout() {
			return true
		}
		// 临时性网络错误
		if netErr.Temporary() {
			return true
		}
	}

	// 检查错误消息
	errMsg := strings.ToLower(err.Error())
	networkKeywords := []string{
		"connection refused",
		"connection reset",
		"connection timed out",
		"network is unreachable",
		"no route to host",
		"host unreachable",
		"network unreachable",
		"broken pipe",
		"connection aborted",
		"eof",
	}

	for _, keyword := range networkKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// isTimeoutError 检查是否为超时错误
func (ec *errorClassifier) isTimeoutError(err error) bool {
	// 检查是否为context.DeadlineExceeded
	if errors.Is(err, context.DeadlineExceeded) {
		return true
	}

	// 检查错误消息
	errMsg := strings.ToLower(err.Error())
	timeoutKeywords := []string{
		"timeout",
		"timed out",
		"deadline exceeded",
		"operation timed out",
		"request timeout",
		"read timeout",
		"write timeout",
	}

	for _, keyword := range timeoutKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// isRateLimitError 检查是否为速率限制错误
func (ec *errorClassifier) isRateLimitError(errMsg string) bool {
	rateLimitKeywords := []string{
		"rate limit",
		"rate_limit",
		"ratelimit",
		"too many requests",
		"too many requests",
		"quota exceeded",
		"quota_exceeded",
		"429",
		"request limit",
		"request_limit",
		"api limit",
		"api_limit",
		"usage limit",
		"usage_limit",
		"throttled",
		"throttle",
	}

	for _, keyword := range rateLimitKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// isAuthError 检查是否为认证错误
func (ec *errorClassifier) isAuthError(errMsg string) bool {
	authKeywords := []string{
		"unauthorized",
		"authentication failed",
		"invalid api key",
		"invalid token",
		"invalid credentials",
		"forbidden",
		"401",
		"403",
		"access denied",
		"permission denied",
	}

	for _, keyword := range authKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// isValidationError 检查是否为验证错误
func (ec *errorClassifier) isValidationError(errMsg string) bool {
	validationKeywords := []string{
		"invalid request",
		"invalid parameter",
		"invalid argument",
		"validation error",
		"bad request",
		"400",
		"malformed",
		"missing required",
	}

	for _, keyword := range validationKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// isTemporaryError 检查是否为临时性错误
func (ec *errorClassifier) isTemporaryError(errMsg string) bool {
	temporaryKeywords := []string{
		"temporary",
		"temporarily",
		"service unavailable",
		"service_unavailable",
		"503",
		"502",
		"504",
		"bad gateway",
		"gateway timeout",
		"try again later",
		"try again later",
	}

	for _, keyword := range temporaryKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// isPermanentError 检查是否为永久性错误
func (ec *errorClassifier) isPermanentError(errMsg string) bool {
	permanentKeywords := []string{
		"not found",
		"not_found",
		"404",
		"invalid model",
		"model not found",
		"model not supported",
		"endpoint not found",
		"resource not found",
		"does not exist",
	}

	for _, keyword := range permanentKeywords {
		if strings.Contains(errMsg, keyword) {
			return true
		}
	}

	return false
}

// GetRetryDelay 获取重试延迟
// 根据错误类型和重试次数计算延迟
func (ec *errorClassifier) GetRetryDelay(err error, attempt int) time.Duration {
	if err == nil {
		return ec.config.DefaultRetryDelay
	}

	errorType := ec.ClassifyError(err)

	switch errorType {
	case ErrorTypeRateLimit:
		// 速率限制错误：使用较长的延迟
		return ec.calculateExponentialDelay(attempt, ec.config.DefaultRetryDelay*5)

	case ErrorTypeTimeout:
		// 超时错误：使用指数退避
		return ec.calculateExponentialDelay(attempt, ec.config.DefaultRetryDelay*2)

	case ErrorTypeNetwork:
		// 网络错误：使用指数退避
		return ec.calculateExponentialDelay(attempt, ec.config.DefaultRetryDelay*3)

	case ErrorTypeTemporary:
		// 临时性错误：使用中等延迟
		return ec.calculateExponentialDelay(attempt, ec.config.DefaultRetryDelay*2)

	default:
		// 其他错误：使用默认延迟
		return ec.calculateExponentialDelay(attempt, ec.config.DefaultRetryDelay)
	}
}

// calculateExponentialDelay 计算指数退避延迟
func (ec *errorClassifier) calculateExponentialDelay(attempt int, baseDelay time.Duration) time.Duration {
	// 计算延迟：baseDelay * (multiplier ^ attempt)
	delay := float64(baseDelay) * ec.pow(ec.config.RetryDelayMultiplier, attempt)

	// 限制最大延迟
	if delay > float64(ec.config.MaxRetryDelay) {
		delay = float64(ec.config.MaxRetryDelay)
	}

	return time.Duration(delay)
}

// pow 计算幂
func (ec *errorClassifier) pow(base float64, exp int) float64 {
	result := 1.0
	for i := 0; i < exp; i++ {
		result *= base
	}
	return result
}

// ShouldCircuitBreak 判断是否应该触发熔断
// ⭐ 核心方法：根据错误类型判断是否应该触发熔断
func (ec *errorClassifier) ShouldCircuitBreak(err error) bool {
	if err == nil {
		return false
	}

	errorType := ec.ClassifyError(err)

	switch errorType {
	case ErrorTypeRateLimit:
		// 速率限制错误：根据配置决定是否触发熔断
		return ec.config.CircuitBreakOnRateLimit

	case ErrorTypeAuth, ErrorTypeValidation:
		// 认证和验证错误：应该触发熔断（配置错误）
		return true

	case ErrorTypePermanent:
		// 永久性错误：应该触发熔断
		return true

	case ErrorTypeNetwork, ErrorTypeTimeout, ErrorTypeTemporary:
		// 网络、超时、临时性错误：不触发熔断（可能恢复）
		return false

	default:
		// 未知错误：不触发熔断
		return false
	}
}

// GetMaxRetries 获取最大重试次数
// 根据错误类型返回最大重试次数
func (ec *errorClassifier) GetMaxRetries(err error) int {
	if err == nil {
		return 1
	}

	errorType := ec.ClassifyError(err)

	switch errorType {
	case ErrorTypeNetwork:
		return ec.config.NetworkErrorMaxRetries

	case ErrorTypeTimeout:
		return ec.config.TimeoutErrorMaxRetries

	case ErrorTypeRateLimit:
		return ec.config.RateLimitErrorMaxRetries

	case ErrorTypeTemporary:
		return 3

	case ErrorTypePermanent, ErrorTypeAuth, ErrorTypeValidation:
		return 0

	default:
		return 1
	}
}

// FormatError 格式化错误信息
func (ec *errorClassifier) FormatError(err error) string {
	if err == nil {
		return "no error"
	}

	errorType := ec.ClassifyError(err)
	return fmt.Sprintf("[%s] %s", errorType, err.Error())
}

// WrapError 包装错误并添加类型信息
func (ec *errorClassifier) WrapError(err error, errorType ErrorType) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s: %w", errorType, err)
}
