package scheduler

import (
	"context"
	"sort"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/model"
)

type providerTrafficProfile struct {
	lane       string
	experiment string
	weight     int
	canary     bool
	tags       map[string]struct{}
}

func (c *cascadeController) applyTrafficGovernance(ctx context.Context, operation string, modelID string, providers []*model.ModelProvider) []*model.ModelProvider {
	if len(providers) <= 1 {
		result := make([]*model.ModelProvider, len(providers))
		copy(result, providers)
		return result
	}

	hints := TrafficHintsFromContext(ctx)
	result := make([]*model.ModelProvider, len(providers))
	copy(result, providers)

	result = c.applyTrafficTagFilter(hints, result)
	result = c.applyCanaryRouting(hints, operation, modelID, result)
	result = c.applyExperimentSplit(hints, operation, modelID, result)
	result = c.applySessionAffinityPriority(ctx, hints, operation, modelID, result)

	return dedupeProviderOrder(result)
}

func (c *cascadeController) applyTrafficTagFilter(hints TrafficHints, providers []*model.ModelProvider) []*model.ModelProvider {
	tag := strings.TrimSpace(strings.ToLower(hints.TrafficTag))
	if tag == "" || len(providers) <= 1 {
		return providers
	}

	matched := make([]*model.ModelProvider, 0, len(providers))
	others := make([]*model.ModelProvider, 0, len(providers))
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		profile := resolveProviderTrafficProfile(provider)
		if _, ok := profile.tags[tag]; ok || profile.lane == tag || profile.experiment == tag {
			matched = append(matched, provider)
			continue
		}
		others = append(others, provider)
	}

	if len(matched) == 0 {
		return providers
	}
	return append(matched, others...)
}

func (c *cascadeController) applyCanaryRouting(hints TrafficHints, operation string, modelID string, providers []*model.ModelProvider) []*model.ModelProvider {
	if len(providers) <= 1 {
		return providers
	}

	canaryProviders := make([]*model.ModelProvider, 0, len(providers))
	stableProviders := make([]*model.ModelProvider, 0, len(providers))
	canaryPercent := 0

	for _, provider := range providers {
		if provider == nil {
			continue
		}
		profile := resolveProviderTrafficProfile(provider)
		if profile.canary {
			canaryProviders = append(canaryProviders, provider)
			if p := readProviderCanaryPercent(provider); p > canaryPercent {
				canaryPercent = p
			}
			continue
		}
		stableProviders = append(stableProviders, provider)
	}

	if len(canaryProviders) == 0 {
		return providers
	}

	if hints.ForceCanary {
		return append(canaryProviders, stableProviders...)
	}

	if canaryPercent <= 0 {
		return append(stableProviders, canaryProviders...)
	}

	seed := hints.RoutingSeed(operation, modelID)
	bucket := DeterministicBucket(seed, 100)
	if bucket < canaryPercent {
		return append(canaryProviders, stableProviders...)
	}
	if len(stableProviders) == 0 {
		return canaryProviders
	}
	return append(stableProviders, canaryProviders...)
}

func (c *cascadeController) applyExperimentSplit(hints TrafficHints, operation string, modelID string, providers []*model.ModelProvider) []*model.ModelProvider {
	if len(providers) <= 1 {
		return providers
	}

	experiment := strings.TrimSpace(strings.ToLower(hints.ExperimentID))
	if experiment != "" {
		matched := make([]*model.ModelProvider, 0, len(providers))
		others := make([]*model.ModelProvider, 0, len(providers))
		for _, provider := range providers {
			if provider == nil {
				continue
			}
			profile := resolveProviderTrafficProfile(provider)
			if profile.experiment == experiment || profile.lane == experiment {
				matched = append(matched, provider)
				continue
			}
			others = append(others, provider)
		}
		if len(matched) > 0 {
			return append(matched, others...)
		}
	}

	type weightedProvider struct {
		provider *model.ModelProvider
		weight   int
	}

	weighted := make([]weightedProvider, 0, len(providers))
	totalWeight := 0
	for _, provider := range providers {
		if provider == nil {
			continue
		}
		profile := resolveProviderTrafficProfile(provider)
		weight := profile.weight
		if weight <= 0 {
			weight = 1
		}
		totalWeight += weight
		weighted = append(weighted, weightedProvider{
			provider: provider,
			weight:   weight,
		})
	}
	if len(weighted) <= 1 || totalWeight <= 0 {
		return providers
	}

	seed := hints.RoutingSeed(operation, modelID)
	pick := DeterministicBucket(seed, totalWeight)
	selectedIndex := 0
	cumulative := 0
	for i, item := range weighted {
		cumulative += item.weight
		if pick < cumulative {
			selectedIndex = i
			break
		}
	}

	selectedProvider := weighted[selectedIndex].provider
	ordered := make([]*model.ModelProvider, 0, len(weighted))
	if selectedProvider != nil {
		ordered = append(ordered, selectedProvider)
	}
	for _, item := range weighted {
		if item.provider == nil || (selectedProvider != nil && item.provider.ID == selectedProvider.ID) {
			continue
		}
		ordered = append(ordered, item.provider)
	}
	return ordered
}

func (c *cascadeController) applySessionAffinityPriority(ctx context.Context, hints TrafficHints, operation string, modelID string, providers []*model.ModelProvider) []*model.ModelProvider {
	if c == nil || c.stateStore == nil || len(providers) <= 1 {
		return providers
	}

	sessionID := strings.TrimSpace(hints.SessionID)
	if sessionID == "" {
		return providers
	}

	key := buildSessionAffinityKey(sessionID, operation, modelID)
	affinity, err := c.stateStore.GetSessionAffinity(ctx, key)
	if err != nil || affinity == nil || affinity.ProviderID == 0 {
		return providers
	}

	selectedIndex := -1
	for index, provider := range providers {
		if provider == nil {
			continue
		}
		if provider.ID == affinity.ProviderID {
			selectedIndex = index
			break
		}
	}
	if selectedIndex <= 0 {
		return providers
	}

	ordered := make([]*model.ModelProvider, 0, len(providers))
	selected := providers[selectedIndex]
	if selected != nil && affinity.InstanceID > 0 {
		selected.SelectedInstance = &model.ProviderInstance{
			ID:         affinity.InstanceID,
			ProviderID: selected.ID,
		}
	}
	ordered = append(ordered, selected)
	for i, provider := range providers {
		if i == selectedIndex {
			continue
		}
		ordered = append(ordered, provider)
	}
	return ordered
}

func (c *cascadeController) persistSessionAffinity(ctx context.Context, operation string, modelID string, provider *model.ModelProvider) {
	if c == nil || c.stateStore == nil || provider == nil {
		return
	}

	hints := TrafficHintsFromContext(ctx)
	sessionID := strings.TrimSpace(hints.SessionID)
	if sessionID == "" {
		return
	}

	ttl := 30 * time.Minute
	if provider.Channel != nil {
		if sec := readProviderSessionAffinityTTL(provider); sec > 0 {
			ttl = time.Duration(sec) * time.Second
		}
	}

	affinity := &SessionAffinity{
		Operation: strings.TrimSpace(operation),
		ModelID:   strings.TrimSpace(modelID),
		ProviderID: provider.ID,
	}
	if provider.SelectedInstance != nil && provider.SelectedInstance.ID > 0 {
		affinity.InstanceID = provider.SelectedInstance.ID
	}

	key := buildSessionAffinityKey(sessionID, operation, modelID)
	_ = c.stateStore.SetSessionAffinity(ctx, key, affinity, ttl)
}

func buildSessionAffinityKey(sessionID string, operation string, modelID string) string {
	return strings.TrimSpace(sessionID) + "|" + strings.TrimSpace(operation) + "|" + strings.TrimSpace(modelID)
}

func dedupeProviderOrder(providers []*model.ModelProvider) []*model.ModelProvider {
	if len(providers) <= 1 {
		return providers
	}

	deduped := make([]*model.ModelProvider, 0, len(providers))
	seen := make(map[uint]struct{}, len(providers))
	for _, provider := range providers {
		if provider == nil || provider.ID == 0 {
			continue
		}
		if _, ok := seen[provider.ID]; ok {
			continue
		}
		seen[provider.ID] = struct{}{}
		deduped = append(deduped, provider)
	}
	return deduped
}

func resolveProviderTrafficProfile(provider *model.ModelProvider) providerTrafficProfile {
	profile := providerTrafficProfile{
		lane:       "",
		experiment: "",
		weight:     100,
		canary:     false,
		tags:       make(map[string]struct{}),
	}
	if provider == nil {
		return profile
	}

	if provider.Weight > 0 {
		profile.weight = provider.Weight
	}

	if rawWeight, ok := readProviderChannelConfigInt(provider, "traffic_weight", "traffic_split_weight", "ab_weight"); ok && rawWeight > 0 {
		profile.weight = rawWeight
	}

	profile.lane = strings.ToLower(strings.TrimSpace(readProviderConfigString(provider, "traffic_lane", "traffic_color", "lane")))
	profile.experiment = strings.ToLower(strings.TrimSpace(readProviderConfigString(provider, "traffic_experiment", "ab_experiment", "experiment")))
	if profile.lane == "canary" {
		profile.canary = true
	}
	if rawCanary, ok := readProviderChannelConfigBool(provider, "traffic_canary", "canary"); ok && rawCanary {
		profile.canary = true
	}

	for _, tag := range readProviderTrafficTags(provider) {
		tag = strings.ToLower(strings.TrimSpace(tag))
		if tag == "" {
			continue
		}
		profile.tags[tag] = struct{}{}
	}
	if profile.lane != "" {
		profile.tags[profile.lane] = struct{}{}
	}
	if profile.experiment != "" {
		profile.tags[profile.experiment] = struct{}{}
	}
	return profile
}

func readProviderCanaryPercent(provider *model.ModelProvider) int {
	value, ok := readProviderChannelConfigInt(provider, "traffic_canary_percent", "canary_percent")
	if !ok {
		return 0
	}
	if value < 0 {
		return 0
	}
	if value > 100 {
		return 100
	}
	return value
}

func readProviderSessionAffinityTTL(provider *model.ModelProvider) int {
	value, ok := readProviderChannelConfigInt(provider, "session_affinity_ttl_seconds", "sticky_session_ttl_seconds")
	if !ok || value <= 0 {
		return 0
	}
	return value
}

func readProviderTrafficTags(provider *model.ModelProvider) []string {
	raw, ok := readProviderChannelConfigValue(provider, "traffic_tags", "tags")
	if !ok || raw == nil {
		return nil
	}

	switch value := raw.(type) {
	case string:
		items := strings.Split(value, ",")
		result := make([]string, 0, len(items))
		for _, item := range items {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			result = append(result, item)
		}
		sort.Strings(result)
		return result
	case []string:
		result := make([]string, 0, len(value))
		for _, item := range value {
			item = strings.TrimSpace(item)
			if item == "" {
				continue
			}
			result = append(result, item)
		}
		sort.Strings(result)
		return result
	case []interface{}:
		result := make([]string, 0, len(value))
		for _, item := range value {
			str, ok := item.(string)
			if !ok {
				continue
			}
			str = strings.TrimSpace(str)
			if str == "" {
				continue
			}
			result = append(result, str)
		}
		sort.Strings(result)
		return result
	default:
		return nil
	}
}

func readProviderChannelConfigBool(provider *model.ModelProvider, keys ...string) (bool, bool) {
	raw, ok := readProviderChannelConfigValue(provider, keys...)
	if !ok || raw == nil {
		return false, false
	}

	switch value := raw.(type) {
	case bool:
		return value, true
	case string:
		parsed, err := strconv.ParseBool(strings.TrimSpace(value))
		if err != nil {
			return false, false
		}
		return parsed, true
	default:
		return false, false
	}
}

func readProviderChannelConfigInt(provider *model.ModelProvider, keys ...string) (int, bool) {
	raw, ok := readProviderChannelConfigValue(provider, keys...)
	if !ok || raw == nil {
		return 0, false
	}

	switch value := raw.(type) {
	case int:
		return value, true
	case int32:
		return int(value), true
	case int64:
		return int(value), true
	case float32:
		return int(value), true
	case float64:
		return int(value), true
	case string:
		parsed, err := strconv.Atoi(strings.TrimSpace(value))
		if err != nil {
			return 0, false
		}
		return parsed, true
	default:
		return 0, false
	}
}

func readProviderChannelConfigValue(provider *model.ModelProvider, keys ...string) (interface{}, bool) {
	if provider == nil || provider.Channel == nil || provider.Channel.Config == nil {
		return nil, false
	}
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key == "" {
			continue
		}
		value, ok := provider.Channel.Config[key]
		if !ok || value == nil {
			continue
		}
		return value, true
	}
	return nil, false
}
