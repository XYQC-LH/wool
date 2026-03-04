package scheduler

import "fmt"

// ProviderRateLimitedError 表示源头组层面触发了限流（如按 operation 的 RPM/TPM/SPM 等规则）。
//
// 该错误属于“可重试/可 failover”的上游限制类错误；对外建议映射为 HTTP 429（OpenAI: rate_limit_error）。
type ProviderRateLimitedError struct {
	ProviderID uint
	Operation  string
}

func (e *ProviderRateLimitedError) Error() string {
	if e == nil {
		return "rate limited"
	}
	if e.Operation != "" {
		return fmt.Sprintf("源头 %d 达到限流阈值（%s）", e.ProviderID, e.Operation)
	}
	return fmt.Sprintf("源头 %d 达到限流阈值", e.ProviderID)
}

