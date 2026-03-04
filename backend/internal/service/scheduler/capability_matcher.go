package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

// CapabilityReject 记录能力过滤淘汰原因
type CapabilityReject struct {
	ProviderID uint   `json:"provider_id"`
	Reason     string `json:"reason"`
}

// CapabilityMatcher 源头能力匹配器
// 负责 Constraints 阶段的硬过滤：不满足约束直接剔除
type CapabilityMatcher interface {
	MatchProviders(ctx context.Context, operation string, providers []*model.ModelProvider) ([]*model.ModelProvider, []CapabilityReject, error)
}

type capabilityMatcher struct {
	capabilityRepo repository.ProviderCapabilityRepository
}

// NewCapabilityMatcher 创建能力匹配器
func NewCapabilityMatcher(capabilityRepo repository.ProviderCapabilityRepository) CapabilityMatcher {
	return &capabilityMatcher{
		capabilityRepo: capabilityRepo,
	}
}

type capabilityConstraintsContextKey struct{}

// WithCapabilityConstraints 将请求约束注入上下文
func WithCapabilityConstraints(ctx context.Context, constraints map[string]interface{}) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if len(constraints) == 0 {
		return ctx
	}

	cloned := make(map[string]interface{}, len(constraints))
	for key, value := range constraints {
		cloned[key] = value
	}
	return context.WithValue(ctx, capabilityConstraintsContextKey{}, cloned)
}

// CapabilityConstraintsFromContext 从上下文读取请求约束
func CapabilityConstraintsFromContext(ctx context.Context) map[string]interface{} {
	if ctx == nil {
		return nil
	}
	value := ctx.Value(capabilityConstraintsContextKey{})
	if value == nil {
		return nil
	}
	constraints, ok := value.(map[string]interface{})
	if !ok || len(constraints) == 0 {
		return nil
	}
	cloned := make(map[string]interface{}, len(constraints))
	for key, item := range constraints {
		cloned[key] = item
	}
	return cloned
}

func (m *capabilityMatcher) MatchProviders(ctx context.Context, operation string, providers []*model.ModelProvider) ([]*model.ModelProvider, []CapabilityReject, error) {
	if len(providers) == 0 {
		return nil, nil, nil
	}

	constraints := CapabilityConstraintsFromContext(ctx)
	if len(constraints) == 0 {
		result := make([]*model.ModelProvider, 0, len(providers))
		for _, provider := range providers {
			if provider != nil {
				result = append(result, provider)
			}
		}
		return result, nil, nil
	}

	operation = model.NormalizeOperation(operation)
	result := make([]*model.ModelProvider, 0, len(providers))
	rejected := make([]CapabilityReject, 0)

	for _, provider := range providers {
		if provider == nil {
			continue
		}

		if m.capabilityRepo == nil {
			result = append(result, provider)
			continue
		}

		caps, err := m.capabilityRepo.GetByProvider(ctx, provider.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("加载源头能力失败(provider=%d): %w", provider.ID, err)
		}

		matched, reason := matchProviderConstraints(operation, constraints, caps)
		if matched {
			result = append(result, provider)
			continue
		}

		rejected = append(rejected, CapabilityReject{
			ProviderID: provider.ID,
			Reason:     reason,
		})
	}

	return result, rejected, nil
}

func matchProviderConstraints(operation string, requirements map[string]interface{}, caps []*model.ProviderCapability) (bool, string) {
	var target *model.ProviderCapability
	for _, cap := range caps {
		if cap == nil || !cap.IsEnabled {
			continue
		}
		if model.NormalizeOperation(cap.Operation) == operation {
			target = cap
			break
		}
	}

	if target == nil {
		return false, "capability_not_found"
	}

	for key, requiredValue := range requirements {
		ruleValue, exists := target.Constraints[key]
		if !exists {
			return false, "constraint_missing:" + key
		}

		matched, reason := matchConstraintValue(requiredValue, ruleValue)
		if !matched {
			return false, fmt.Sprintf("%s:%s", key, reason)
		}
	}
	return true, ""
}

func matchConstraintValue(required interface{}, rule interface{}) (bool, string) {
	switch typedRule := rule.(type) {
	case []interface{}:
		if containsValue(typedRule, required) {
			return true, ""
		}
		return false, "not_in_allowed_values"
	case map[string]interface{}:
		return matchRuleObject(required, typedRule)
	default:
		if valuesEqual(required, typedRule) {
			return true, ""
		}
		return false, "not_equal"
	}
}

func matchRuleObject(required interface{}, rule map[string]interface{}) (bool, string) {
	if expected, ok := rule["eq"]; ok {
		if !valuesEqual(required, expected) {
			return false, "eq_mismatch"
		}
	}

	if allowed, ok := rule["in"]; ok {
		allowedList, ok := toSlice(allowed)
		if !ok || !containsValue(allowedList, required) {
			return false, "in_mismatch"
		}
	}

	if oneOf, ok := rule["one_of"]; ok {
		allowedList, ok := toSlice(oneOf)
		if !ok || !containsValue(allowedList, required) {
			return false, "one_of_mismatch"
		}
	}

	if minValue, ok := firstExisting(rule, "min", "gte"); ok {
		requiredNum, reqOK := toFloat64(required)
		minNum, minOK := toFloat64(minValue)
		if !reqOK || !minOK || requiredNum < minNum {
			return false, "min_mismatch"
		}
	}

	if maxValue, ok := firstExisting(rule, "max", "lte"); ok {
		requiredNum, reqOK := toFloat64(required)
		maxNum, maxOK := toFloat64(maxValue)
		if !reqOK || !maxOK || requiredNum > maxNum {
			return false, "max_mismatch"
		}
	}

	return true, ""
}

func firstExisting(rule map[string]interface{}, keys ...string) (interface{}, bool) {
	for _, key := range keys {
		if value, ok := rule[key]; ok {
			return value, true
		}
	}
	return nil, false
}

func toSlice(value interface{}) ([]interface{}, bool) {
	switch typed := value.(type) {
	case []interface{}:
		return typed, true
	case []string:
		result := make([]interface{}, 0, len(typed))
		for _, item := range typed {
			result = append(result, item)
		}
		return result, true
	default:
		return nil, false
	}
}

func containsValue(list []interface{}, required interface{}) bool {
	for _, item := range list {
		if valuesEqual(item, required) {
			return true
		}
	}
	return false
}

func valuesEqual(left interface{}, right interface{}) bool {
	leftNum, leftIsNum := toFloat64(left)
	rightNum, rightIsNum := toFloat64(right)
	if leftIsNum && rightIsNum {
		return leftNum == rightNum
	}

	leftText := normalizeText(left)
	rightText := normalizeText(right)
	return leftText == rightText
}

func toFloat64(value interface{}) (float64, bool) {
	switch typed := value.(type) {
	case int:
		return float64(typed), true
	case int8:
		return float64(typed), true
	case int16:
		return float64(typed), true
	case int32:
		return float64(typed), true
	case int64:
		return float64(typed), true
	case uint:
		return float64(typed), true
	case uint8:
		return float64(typed), true
	case uint16:
		return float64(typed), true
	case uint32:
		return float64(typed), true
	case uint64:
		return float64(typed), true
	case float32:
		return float64(typed), true
	case float64:
		return typed, true
	case string:
		parsed, err := strconv.ParseFloat(strings.TrimSpace(typed), 64)
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func normalizeText(value interface{}) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(strings.ToLower(typed))
	default:
		return strings.TrimSpace(strings.ToLower(fmt.Sprintf("%v", typed)))
	}
}
