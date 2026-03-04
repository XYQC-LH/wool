package scheduler

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

// ProviderRateLimiter 源头组限流器（按 provider_rate_limit_rules 生效）。
//
// 当前实现（KISS）：
// - 仅内置接入 unit=request（每次调度尝试消耗 1）
// - 其他 unit（image/video_second/audio_second...）交由业务侧按需扩展
type ProviderRateLimiter interface {
	Allow(ctx context.Context, providerID uint, operation string) (bool, error)
}

type ProviderRateLimiterConfig struct {
	CacheTTL time.Duration
}

func DefaultProviderRateLimiterConfig() *ProviderRateLimiterConfig {
	return &ProviderRateLimiterConfig{
		CacheTTL: 10 * time.Second,
	}
}

type providerRateLimiter struct {
	repo repository.ProviderRateLimitRuleRepository
	cfg  *ProviderRateLimiterConfig

	mu    sync.RWMutex
	cache map[string]providerRateLimiterCacheEntry
}

type providerRateLimiterCacheEntry struct {
	rules    []*model.ProviderRateLimitRule
	expireAt time.Time
}

func NewProviderRateLimiter(repo repository.ProviderRateLimitRuleRepository, cfg *ProviderRateLimiterConfig) ProviderRateLimiter {
	if cfg == nil {
		cfg = DefaultProviderRateLimiterConfig()
	}
	return &providerRateLimiter{
		repo:  repo,
		cfg:   cfg,
		cache: make(map[string]providerRateLimiterCacheEntry),
	}
}

func (l *providerRateLimiter) Allow(ctx context.Context, providerID uint, operation string) (bool, error) {
	if l == nil || l.repo == nil || providerID == 0 {
		return true, nil
	}
	if ctx == nil {
		ctx = context.Background()
	}

	operation = model.NormalizeOperation(operation)
	rules, err := l.getRules(ctx, providerID, operation)
	if err != nil {
		return true, err
	}

	for _, rule := range rules {
		if rule == nil || !rule.Enabled {
			continue
		}
		unit := strings.ToLower(strings.TrimSpace(rule.Unit))
		increment := resolveRateLimitIncrement(ctx, unit)
		if increment <= 0 {
			continue
		}
		if rule.Limit <= 0 || rule.WindowSeconds <= 0 {
			continue
		}

		window := time.Duration(rule.WindowSeconds) * time.Second
		key := fmt.Sprintf("provider:rate:%d:%s:%s:%d", providerID, operation, unit, rule.WindowSeconds)
		allowed, err := cache.FixedWindowRateLimit(key, rule.Limit, window, increment)
		if err != nil {
			return true, err
		}
		if !allowed {
			return false, nil
		}
	}

	return true, nil
}

func (l *providerRateLimiter) cacheKey(providerID uint, operation string) string {
	return fmt.Sprintf("%d:%s", providerID, model.NormalizeOperation(operation))
}

func (l *providerRateLimiter) getRules(ctx context.Context, providerID uint, operation string) ([]*model.ProviderRateLimitRule, error) {
	cacheKey := l.cacheKey(providerID, operation)

	l.mu.RLock()
	cached, ok := l.cache[cacheKey]
	l.mu.RUnlock()

	if ok && l.cfg.CacheTTL > 0 && time.Now().Before(cached.expireAt) {
		return cached.rules, nil
	}

	rules, err := l.repo.ListEnabledByProviderOperation(ctx, providerID, operation)
	if err != nil {
		return nil, err
	}

	expireAt := time.Now()
	if l.cfg.CacheTTL > 0 {
		expireAt = expireAt.Add(l.cfg.CacheTTL)
	}

	l.mu.Lock()
	l.cache[cacheKey] = providerRateLimiterCacheEntry{
		rules:    rules,
		expireAt: expireAt,
	}
	l.mu.Unlock()

	return rules, nil
}
