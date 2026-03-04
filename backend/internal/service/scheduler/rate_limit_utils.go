package scheduler

import (
	"context"
	"strings"
)

type rateLimitUsageKey struct{}

// WithRateLimitUsage 在 context 上附加用量信息，供限流计算使用。
func WithRateLimitUsage(ctx context.Context, usage map[string]int64) context.Context {
	if ctx == nil || usage == nil {
		return ctx
	}
	return context.WithValue(ctx, rateLimitUsageKey{}, usage)
}

func rateLimitUsageFromContext(ctx context.Context) map[string]int64 {
	if ctx == nil {
		return nil
	}
	usage, ok := ctx.Value(rateLimitUsageKey{}).(map[string]int64)
	if !ok {
		return nil
	}
	return usage
}

func resolveRateLimitIncrement(ctx context.Context, unit string) int64 {
	unit = strings.ToLower(strings.TrimSpace(unit))
	if unit == "" {
		return 0
	}

	usage := rateLimitUsageFromContext(ctx)
	if usage != nil {
		if v, ok := usage[unit]; ok && v > 0 {
			return v
		}
	}

	switch unit {
	case "request", "requests", "rpm", "rps":
		return 1
	case "token", "tokens", "tpm":
		return pickUsageValue(usage, "tokens", "token")
	case "second", "seconds", "spm", "video_second", "video_seconds", "audio_second", "audio_seconds":
		return pickUsageValue(usage, "video_second", "video_seconds", "audio_second", "audio_seconds", "seconds", "second")
	case "pixel", "pixels", "ppm":
		return pickUsageValue(usage, "pixels", "pixel")
	case "image", "images":
		return pickUsageValue(usage, "images", "image")
	default:
		return 0
	}
}

func pickUsageValue(usage map[string]int64, keys ...string) int64 {
	if usage == nil {
		return 0
	}
	for _, key := range keys {
		if v, ok := usage[key]; ok && v > 0 {
			return v
		}
	}
	return 0
}
