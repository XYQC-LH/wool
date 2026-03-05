package scheduler

import (
	"context"
	"hash/fnv"
	"strings"
)

// TrafficHints 调度流量提示信息
// 用于承载流量染色、实验分组、会话保持等请求级上下文。
type TrafficHints struct {
	SessionID      string
	TrafficTag     string
	ExperimentID   string
	IdempotencyKey string
	ForceCanary    bool
}

type trafficHintsContextKey struct{}

// WithTrafficHints 将流量提示注入上下文
func WithTrafficHints(ctx context.Context, hints TrafficHints) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	normalized := hints.normalized()
	return context.WithValue(ctx, trafficHintsContextKey{}, normalized)
}

// TrafficHintsFromContext 从上下文读取流量提示
func TrafficHintsFromContext(ctx context.Context) TrafficHints {
	if ctx == nil {
		return TrafficHints{}
	}
	hints, ok := ctx.Value(trafficHintsContextKey{}).(TrafficHints)
	if !ok {
		return TrafficHints{}
	}
	return hints.normalized()
}

func (h TrafficHints) normalized() TrafficHints {
	h.SessionID = strings.TrimSpace(h.SessionID)
	h.TrafficTag = strings.TrimSpace(strings.ToLower(h.TrafficTag))
	h.ExperimentID = strings.TrimSpace(strings.ToLower(h.ExperimentID))
	h.IdempotencyKey = strings.TrimSpace(h.IdempotencyKey)
	return h
}

// RoutingSeed 生成稳定路由种子（优先使用 session，其次幂等键）
func (h TrafficHints) RoutingSeed(operation string, modelID string) string {
	h = h.normalized()
	operation = strings.TrimSpace(strings.ToLower(operation))
	modelID = strings.TrimSpace(strings.ToLower(modelID))

	switch {
	case h.SessionID != "":
		return h.SessionID + "|" + operation + "|" + modelID
	case h.IdempotencyKey != "":
		return h.IdempotencyKey + "|" + operation + "|" + modelID
	case h.ExperimentID != "":
		return h.ExperimentID + "|" + operation + "|" + modelID
	case h.TrafficTag != "":
		return h.TrafficTag + "|" + operation + "|" + modelID
	default:
		return operation + "|" + modelID
	}
}

// DeterministicBucket 返回稳定桶值（[0, mod)）
func DeterministicBucket(seed string, mod int) int {
	if mod <= 1 {
		return 0
	}
	hasher := fnv.New32a()
	_, _ = hasher.Write([]byte(strings.TrimSpace(seed)))
	value := int(hasher.Sum32() % uint32(mod))
	if value < 0 {
		return -value
	}
	return value
}
