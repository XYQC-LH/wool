package scheduler

import (
	"context"
	"fmt"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
)

// RouteResolver 路由解析器
// 负责将请求 RouteKey(operation + model_id) 解析为候选模型列表
type RouteResolver interface {
	ResolveRouteProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error)
	ResolveRouteModels(ctx context.Context, operation string, modelID string) ([]string, error)
}

type routeResolver struct {
	routeRepo repository.ModelRouteRepository
}

// NewRouteResolver 创建路由解析器
func NewRouteResolver(routeRepo repository.ModelRouteRepository) RouteResolver {
	return &routeResolver{
		routeRepo: routeRepo,
	}
}

func (r *routeResolver) ResolveRouteModels(ctx context.Context, operation string, modelID string) ([]string, error) {
	modelID = strings.TrimSpace(modelID)
	providers, err := r.ResolveRouteProviders(ctx, operation, modelID)
	if err != nil {
		return nil, err
	}
	if len(providers) > 0 {
		ordered := make([]string, 0, len(providers)+1)
		seen := make(map[string]struct{}, len(providers)+1)
		for _, provider := range providers {
			if provider == nil {
				continue
			}
			target := strings.TrimSpace(provider.ModelID)
			if target == "" {
				continue
			}
			if _, exists := seen[target]; exists {
				continue
			}
			seen[target] = struct{}{}
			ordered = append(ordered, target)
		}
		if _, exists := seen[modelID]; !exists {
			ordered = append(ordered, modelID)
		}
		if len(ordered) > 0 {
			return ordered, nil
		}
	}

	operation = model.NormalizeOperation(operation)
	if modelID == "" {
		return nil, fmt.Errorf("model_id 不能为空")
	}

	if r == nil || r.routeRepo == nil {
		return []string{modelID}, nil
	}

	routes, err := r.routeRepo.GetByRouteKey(ctx, operation, modelID)
	if err != nil {
		return nil, fmt.Errorf("加载路由配置失败(operation=%s, model=%s): %w", operation, modelID, err)
	}
	if len(routes) == 0 {
		return []string{modelID}, nil
	}

	ordered := make([]string, 0, len(routes)+1)
	seen := make(map[string]struct{}, len(routes)+1)

	for _, route := range routes {
		if route == nil || !route.IsEnabled {
			continue
		}
		target := strings.TrimSpace(route.ModelID)
		if target == "" {
			continue
		}
		if _, exists := seen[target]; exists {
			continue
		}
		seen[target] = struct{}{}
		ordered = append(ordered, target)
	}

	// 兜底：确保请求模型始终在候选链中
	if _, exists := seen[modelID]; !exists {
		ordered = append(ordered, modelID)
	}

	if len(ordered) == 0 {
		return []string{modelID}, nil
	}
	return ordered, nil
}

func (r *routeResolver) ResolveRouteProviders(ctx context.Context, operation string, modelID string) ([]*model.ModelProvider, error) {
	operation = model.NormalizeOperation(operation)
	modelID = strings.TrimSpace(modelID)
	if modelID == "" {
		return nil, fmt.Errorf("model_id 不能为空")
	}
	if r == nil || r.routeRepo == nil {
		return nil, nil
	}

	routes, err := r.routeRepo.GetByRouteKey(ctx, operation, modelID)
	if err != nil {
		return nil, fmt.Errorf("加载路由配置失败(operation=%s, model=%s): %w", operation, modelID, err)
	}
	if len(routes) == 0 {
		return nil, nil
	}

	providers := make([]*model.ModelProvider, 0, len(routes))
	seen := make(map[uint]struct{}, len(routes))
	for _, route := range routes {
		if route == nil || !route.IsEnabled || route.Provider == nil {
			continue
		}

		provider := route.Provider
		if provider.ID == 0 {
			continue
		}
		if model.NormalizeOperation(provider.Operation) != operation {
			continue
		}
		if strings.TrimSpace(provider.ModelID) != modelID {
			continue
		}
		if _, exists := seen[provider.ID]; exists {
			continue
		}
		seen[provider.ID] = struct{}{}
		providers = append(providers, provider)
	}

	return providers, nil
}
