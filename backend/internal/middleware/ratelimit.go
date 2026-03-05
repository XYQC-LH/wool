package middleware

import (
	"fmt"
	"net/http"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/config"
	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RateLimitConfigLocal 限流配置
type RateLimitConfigLocal struct {
	// 每分钟请求数限制
	RequestsPerMinute int64
	// 每分钟 Token 数限制
	TokensPerMinute int64
	// 每天请求数限制
	RequestsPerDay int64
	// 每天 Token 数限制
	TokensPerDay int64
}

// DefaultRateLimitConfig 默认限流配置
var DefaultRateLimitConfig = RateLimitConfigLocal{
	RequestsPerMinute: 60,
	TokensPerMinute:   100000,
	RequestsPerDay:    10000,
	TokensPerDay:      1000000,
}

// IPRateLimitMiddleware IP 限流中间件
func IPRateLimitMiddleware(cfg *config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := fmt.Sprintf("%s%s", cache.KeyIPRateLimit, clientIP)

		// 使用滑动窗口限流
		allowed, err := cache.SlidingWindowRateLimit(key, int64(cfg.IPRequestsPerMinute), time.Minute)
		if err != nil {
			// Redis 错误时放行，但记录日志
			c.Next()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
				"请求过于频繁，请稍后再试",
				model.OpenAIErrorTypeRateLimit,
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// UserRateLimitMiddleware 用户限流中间件
func UserRateLimitMiddleware(cfg *config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户 ID
		userID, exists := c.Get(ContextKeyUserID)
		if !exists {
			c.Next()
			return
		}

		uid := userID.(uuid.UUID)
		key := fmt.Sprintf("%s%s", cache.KeyUserRateLimit, uid.String())

		// 使用滑动窗口限流
		allowed, err := cache.SlidingWindowRateLimit(key, int64(cfg.UserRequestsPerMinute), time.Minute)
		if err != nil {
			// Redis 错误时放行
			c.Next()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
				"请求过于频繁，请稍后再试",
				model.OpenAIErrorTypeRateLimit,
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// TokenBucketRateLimitMiddleware 令牌桶限流中间件
func TokenBucketRateLimitMiddleware(capacity, rate int64) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取用户 ID
		userID, exists := c.Get(ContextKeyUserID)
		if !exists {
			c.Next()
			return
		}

		uid := userID.(uuid.UUID)
		key := fmt.Sprintf("bucket:user:%s", uid.String())

		bucket := cache.NewTokenBucket(key, capacity, rate, time.Hour)
		allowed, err := bucket.Allow(1)
		if err != nil {
			// Redis 错误时放行
			c.Next()
			return
		}

		if !allowed {
			c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
				"请求过于频繁，请稍后再试",
				model.OpenAIErrorTypeRateLimit,
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GatewayRateLimitMiddleware Gateway API 限流中间件
func GatewayRateLimitMiddleware(cfg config.RateLimitConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 先进行 IP 限流
		clientIP := c.ClientIP()
		ipKey := fmt.Sprintf("%s%s:gateway", cache.KeyIPRateLimit, clientIP)

		ipAllowed, err := cache.SlidingWindowRateLimit(ipKey, int64(cfg.IPRequestsPerMinute), time.Minute)
		if err == nil && !ipAllowed {
			c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
				"IP 请求过于频繁，请稍后再试",
				model.OpenAIErrorTypeRateLimit,
				nil,
			))
			c.Abort()
			return
		}

		// 获取 Token 信息进行用户级限流
		token, exists := c.Get(ContextKeyToken)
		if !exists {
			c.Next()
			return
		}

		tokenInfo := token.(*model.Token)
		userKey := fmt.Sprintf("%s%s:gateway", cache.KeyUserRateLimit, tokenInfo.UserID.String())

		userAllowed, err := cache.SlidingWindowRateLimit(userKey, int64(cfg.UserRequestsPerMinute), time.Minute)
		if err == nil && !userAllowed {
			c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
				"用户请求过于频繁，请稍后再试",
				model.OpenAIErrorTypeRateLimit,
				nil,
			))
			c.Abort()
			return
		}

		// Token 级别限流（如果 Token 有单独的限流配置）
		if tokenInfo.RateLimit != nil && *tokenInfo.RateLimit > 0 {
			tokenKey := fmt.Sprintf("rate:token:%s", tokenInfo.Key)
			tokenAllowed, err := cache.SlidingWindowRateLimit(tokenKey, int64(*tokenInfo.RateLimit), time.Minute)
			if err == nil && !tokenAllowed {
				c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
					"API Key 请求过于频繁，请稍后再试",
					model.OpenAIErrorTypeRateLimit,
					nil,
				))
				c.Abort()
				return
			}
		}

		tenantID, ok := GetCurrentTenantID(c)
		if !ok {
			tenantID = tokenInfo.EffectiveTenantID()
		}
		if tenantID != "" {
			if cfg.TenantRequestsPerMinute > 0 {
				tenantKey := fmt.Sprintf("rate:tenant:%s:gateway", tenantID)
				tenantAllowed, err := cache.SlidingWindowRateLimit(tenantKey, int64(cfg.TenantRequestsPerMinute), time.Minute)
				if err == nil && !tenantAllowed {
					c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
						"租户请求过于频繁，请稍后再试",
						model.OpenAIErrorTypeRateLimit,
						nil,
					))
					c.Abort()
					return
				}
			}

			if cfg.TenantRequestsPerDay > 0 {
				today := time.Now().Format("2006-01-02")
				tenantDailyKey := fmt.Sprintf("quota:tenant:daily:%s:%s", tenantID, today)
				tenantDailyCount, err := cache.Incr(tenantDailyKey)
				if err == nil {
					if tenantDailyCount == 1 {
						_ = cache.Expire(tenantDailyKey, 49*time.Hour)
					}
					if tenantDailyCount > int64(cfg.TenantRequestsPerDay) {
						c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
							"租户已达到每日请求配额上限",
							model.OpenAIErrorTypeRateLimit,
							nil,
						))
						c.Abort()
						return
					}
				}
			}
		}

		c.Next()
	}
}

// DailyQuotaMiddleware 每日配额中间件
func DailyQuotaMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get(ContextKeyUserID)
		if !exists {
			c.Next()
			return
		}

		uid := userID.(uuid.UUID)
		today := time.Now().Format("2006-01-02")
		key := fmt.Sprintf("quota:daily:%s:%s", uid.String(), today)

		// 获取今日请求数
		count, err := cache.Incr(key)
		if err != nil {
			c.Next()
			return
		}

		// 设置过期时间（第一次设置）
		if count == 1 {
			_ = cache.Expire(key, 25*time.Hour) // 多留1小时缓冲
		}

		// 检查是否超过每日限额
		if count > DefaultRateLimitConfig.RequestsPerDay {
			c.JSON(http.StatusTooManyRequests, model.NewOpenAIError(
				"已达到每日请求限额",
				model.OpenAIErrorTypeRateLimit,
				nil,
			))
			c.Abort()
			return
		}

		c.Next()
	}
}

// RateLimitInfo 限流信息
type RateLimitInfo struct {
	Limit     int64 `json:"limit"`
	Remaining int64 `json:"remaining"`
	Reset     int64 `json:"reset"` // Unix timestamp
}

// SetRateLimitHeaders 设置限流响应头
func SetRateLimitHeaders(c *gin.Context, info *RateLimitInfo) {
	c.Header("X-RateLimit-Limit", fmt.Sprintf("%d", info.Limit))
	c.Header("X-RateLimit-Remaining", fmt.Sprintf("%d", info.Remaining))
	c.Header("X-RateLimit-Reset", fmt.Sprintf("%d", info.Reset))
}

// GetRateLimitInfo 获取限流信息
func GetRateLimitInfo(userID uuid.UUID, limit int64) (*RateLimitInfo, error) {
	key := fmt.Sprintf("%s%s", cache.KeyUserRateLimit, userID.String())

	// 获取当前窗口内的请求数
	count, err := cache.ZCard(key)
	if err != nil {
		return nil, err
	}

	remaining := limit - count
	if remaining < 0 {
		remaining = 0
	}

	// 计算重置时间（下一分钟）
	now := time.Now()
	reset := now.Add(time.Minute).Truncate(time.Minute).Unix()

	return &RateLimitInfo{
		Limit:     limit,
		Remaining: remaining,
		Reset:     reset,
	}, nil
}
