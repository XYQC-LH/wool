package service

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/config"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

const (
	tenantBillingScale = int64(1_000_000)
	tenantBillingTTL   = 49 * time.Hour
)

// TenantBillingEvent 表示一次租户级计费落账事件。
type TenantBillingEvent struct {
	TenantID         string
	UserID           uuid.UUID
	TokenID          uuid.UUID
	Operation        string
	Model            string
	PromptTokens     int
	CompletionTokens int
	Cost             decimal.Decimal
	OccurredAt       time.Time
}

// TenantBillingHook 定义租户计费闭环钩子。
// - CheckQuota: 请求执行前预算预检
// - OnBilled: 计费完成后写入租户日维度聚合数据
type TenantBillingHook interface {
	CheckQuota(ctx context.Context, tenantID string, estimatedCost decimal.Decimal) error
	OnBilled(ctx context.Context, event *TenantBillingEvent) error
}

type noopTenantBillingHook struct{}

func (h *noopTenantBillingHook) CheckQuota(ctx context.Context, tenantID string, estimatedCost decimal.Decimal) error {
	return nil
}

func (h *noopTenantBillingHook) OnBilled(ctx context.Context, event *TenantBillingEvent) error {
	return nil
}

type redisTenantBillingHook struct {
	dailyCostLimit decimal.Decimal
}

// NewTenantBillingHook 创建租户计费钩子。
func NewTenantBillingHook(rateLimitCfg config.RateLimitConfig) TenantBillingHook {
	if cache.GetClient() == nil {
		return &noopTenantBillingHook{}
	}

	limit := decimal.Zero
	if rateLimitCfg.TenantDailyCostLimit > 0 {
		limit = decimal.NewFromFloat(rateLimitCfg.TenantDailyCostLimit)
	}

	return &redisTenantBillingHook{
		dailyCostLimit: limit,
	}
}

func (h *redisTenantBillingHook) CheckQuota(ctx context.Context, tenantID string, estimatedCost decimal.Decimal) error {
	if h == nil || cache.GetClient() == nil {
		return nil
	}
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" || estimatedCost.LessThanOrEqual(decimal.Zero) || h.dailyCostLimit.LessThanOrEqual(decimal.Zero) {
		return nil
	}

	now := time.Now()
	costKey := tenantDailyCostKey(tenantID, now)
	currentMicro, err := cache.GetClient().Get(cache.GetContext(), costKey).Int64()
	if err != nil && !isRedisNil(err) {
		return nil
	}

	neededMicro := decimalToMicro(estimatedCost)
	limitMicro := decimalToMicro(h.dailyCostLimit)
	if currentMicro+neededMicro <= limitMicro {
		return nil
	}

	return &TenantQuotaExceededError{
		TenantID:   tenantID,
		Needed:     estimatedCost,
		Current:    microToDecimal(currentMicro),
		DailyLimit: h.dailyCostLimit,
	}
}

func (h *redisTenantBillingHook) OnBilled(ctx context.Context, event *TenantBillingEvent) error {
	if h == nil || event == nil || cache.GetClient() == nil {
		return nil
	}
	tenantID := strings.TrimSpace(event.TenantID)
	if tenantID == "" {
		return nil
	}

	occurredAt := event.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = time.Now()
	}

	requestKey := tenantDailyRequestsKey(tenantID, occurredAt)
	tokenKey := tenantDailyTokensKey(tenantID, occurredAt)
	costKey := tenantDailyCostKey(tenantID, occurredAt)

	if requestCount, err := cache.IncrBy(requestKey, 1); err == nil && requestCount == 1 {
		_ = cache.Expire(requestKey, tenantBillingTTL)
	}

	totalTokens := int64(event.PromptTokens + event.CompletionTokens)
	if totalTokens > 0 {
		if tokenCount, err := cache.IncrBy(tokenKey, totalTokens); err == nil && tokenCount == totalTokens {
			_ = cache.Expire(tokenKey, tenantBillingTTL)
		}
	}

	costMicro := decimalToMicro(event.Cost)
	if costMicro > 0 {
		if costSum, err := cache.IncrBy(costKey, costMicro); err == nil && costSum == costMicro {
			_ = cache.Expire(costKey, tenantBillingTTL)
		}
	}

	return nil
}

func tenantDailyRequestsKey(tenantID string, ts time.Time) string {
	return fmt.Sprintf("billing:tenant:daily:req:%s:%s", tenantID, ts.Format("2006-01-02"))
}

func tenantDailyTokensKey(tenantID string, ts time.Time) string {
	return fmt.Sprintf("billing:tenant:daily:tokens:%s:%s", tenantID, ts.Format("2006-01-02"))
}

func tenantDailyCostKey(tenantID string, ts time.Time) string {
	return fmt.Sprintf("billing:tenant:daily:cost:%s:%s", tenantID, ts.Format("2006-01-02"))
}

func decimalToMicro(value decimal.Decimal) int64 {
	if value.LessThanOrEqual(decimal.Zero) {
		return 0
	}
	return value.Mul(decimal.NewFromInt(tenantBillingScale)).Ceil().IntPart()
}

func microToDecimal(value int64) decimal.Decimal {
	if value <= 0 {
		return decimal.Zero
	}
	return decimal.NewFromInt(value).Div(decimal.NewFromInt(tenantBillingScale))
}

func isRedisNil(err error) bool {
	return err == redis.Nil
}
