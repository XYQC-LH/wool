package service

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/shopspring/decimal"
)

// InsufficientFundsError 表示用户余额不足（业务错误，通常映射为 HTTP 402）。
type InsufficientFundsError struct {
	Needed  decimal.Decimal
	Balance decimal.Decimal
}

func (e *InsufficientFundsError) Error() string {
	if e == nil {
		return "用户余额不足"
	}
	if e.Needed.GreaterThan(decimal.Zero) && e.Balance.GreaterThanOrEqual(decimal.Zero) {
		return fmt.Sprintf("用户余额不足，需要 %.6f，当前余额 %.6f", e.Needed.InexactFloat64(), e.Balance.InexactFloat64())
	}
	return "用户余额不足"
}

// QuotaExceededError 表示 Token 配额不足/超限（业务错误，通常映射为 HTTP 429）。
type QuotaExceededError struct {
	Needed    decimal.Decimal
	Remaining decimal.Decimal
}

func (e *QuotaExceededError) Error() string {
	if e == nil {
		return "Token配额不足"
	}
	if e.Needed.GreaterThan(decimal.Zero) && e.Remaining.GreaterThanOrEqual(decimal.Zero) {
		return fmt.Sprintf("Token配额不足，需要 %.6f，剩余 %.6f", e.Needed.InexactFloat64(), e.Remaining.InexactFloat64())
	}
	return "Token配额不足"
}

// UpstreamAuthError 表示“上游鉴权失败”，通常是渠道 APIKey/凭证失效导致。
// 这属于网关侧的配置错误，对下游建议映射为 502（Bad Gateway）。
type UpstreamAuthError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamAuthError) UpstreamStatusCode() int {
	if e == nil {
		return 0
	}
	return e.StatusCode
}

func (e *UpstreamAuthError) Error() string {
	if e == nil {
		return "上游鉴权失败"
	}

	msg := fmt.Sprintf("上游鉴权失败: %d", e.StatusCode)
	if e.Body != "" {
		msg = msg + " - " + e.Body
	}
	return msg
}

func IsUpstreamAuthStatus(code int) bool {
	return code == http.StatusUnauthorized || code == http.StatusForbidden
}

// UpstreamRateLimitedError 表示上游返回 429/限流类错误。
// 这属于“可 failover”的上游限制类错误；对外建议映射为 HTTP 429（OpenAI: rate_limit_error）。
type UpstreamRateLimitedError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamRateLimitedError) UpstreamStatusCode() int {
	if e == nil {
		return 0
	}
	return e.StatusCode
}

func (e *UpstreamRateLimitedError) Error() string {
	if e == nil {
		return "上游限流"
	}

	msg := fmt.Sprintf("上游限流: %d", e.StatusCode)
	if strings.TrimSpace(e.Body) != "" {
		msg = msg + " - " + strings.TrimSpace(e.Body)
	}
	return msg
}

func IsUpstreamRateLimitedStatus(code int) bool {
	return code == http.StatusTooManyRequests
}

// UpstreamInsufficientFundsError 表示上游账户/渠道余额不足（例如 402）。
// 这属于网关侧的配置/资源问题；对下游建议映射为 502（Bad Gateway）。
type UpstreamInsufficientFundsError struct {
	StatusCode int
	Body       string
}

func (e *UpstreamInsufficientFundsError) UpstreamStatusCode() int {
	if e == nil {
		return 0
	}
	return e.StatusCode
}

func (e *UpstreamInsufficientFundsError) Error() string {
	if e == nil {
		return "上游余额不足"
	}

	msg := fmt.Sprintf("上游余额不足: %d", e.StatusCode)
	if strings.TrimSpace(e.Body) != "" {
		msg = msg + " - " + strings.TrimSpace(e.Body)
	}
	return msg
}

func IsUpstreamInsufficientFundsStatus(code int) bool {
	return code == http.StatusPaymentRequired
}

// NewUpstreamHTTPError 将上游 HTTP 非 200 状态码统一收敛为强类型错误（或兜底错误）。
// 目的：减少基于字符串的误判，并统一 /v1/* 的错误映射行为。
func NewUpstreamHTTPError(statusCode int, body string) error {
	body = strings.TrimSpace(body)

	if IsUpstreamAuthStatus(statusCode) {
		return &UpstreamAuthError{StatusCode: statusCode, Body: body}
	}
	if IsUpstreamRateLimitedStatus(statusCode) {
		return &UpstreamRateLimitedError{StatusCode: statusCode, Body: body}
	}
	if IsUpstreamInsufficientFundsStatus(statusCode) {
		return &UpstreamInsufficientFundsError{StatusCode: statusCode, Body: body}
	}

	if body == "" {
		return fmt.Errorf("上游返回错误: %d", statusCode)
	}
	return fmt.Errorf("上游返回错误: %d - %s", statusCode, body)
}
