package service

import (
	"context"
	"strings"
	"testing"

	"nexus-api/internal/model"
)

type fakeModelRouteRepo struct {
	nextID uint
	items  map[uint]*model.ModelRoute
}

func newFakeModelRouteRepo() *fakeModelRouteRepo {
	return &fakeModelRouteRepo{
		nextID: 1,
		items:  map[uint]*model.ModelRoute{},
	}
}

func (r *fakeModelRouteRepo) clone(entity *model.ModelRoute) *model.ModelRoute {
	if entity == nil {
		return nil
	}
	copied := *entity
	if entity.Provider != nil {
		provider := *entity.Provider
		copied.Provider = &provider
	}
	return &copied
}

func (r *fakeModelRouteRepo) Create(ctx context.Context, route *model.ModelRoute) error {
	if route.ID == 0 {
		route.ID = r.nextID
		r.nextID++
	}
	r.items[route.ID] = r.clone(route)
	return nil
}

func (r *fakeModelRouteRepo) Update(ctx context.Context, route *model.ModelRoute) error {
	r.items[route.ID] = r.clone(route)
	return nil
}

func (r *fakeModelRouteRepo) Delete(ctx context.Context, id uint) error {
	delete(r.items, id)
	return nil
}

func (r *fakeModelRouteRepo) GetByID(ctx context.Context, id uint) (*model.ModelRoute, error) {
	return r.clone(r.items[id]), nil
}

func (r *fakeModelRouteRepo) GetByModelAndProvider(ctx context.Context, operation string, modelID string, providerID uint) (*model.ModelRoute, error) {
	op := model.NormalizeOperation(operation)
	targetModel := strings.TrimSpace(modelID)
	for _, item := range r.items {
		if model.NormalizeOperation(item.Operation) == op &&
			strings.TrimSpace(item.ModelID) == targetModel &&
			item.ProviderID == providerID {
			return r.clone(item), nil
		}
	}
	return nil, nil
}

func (r *fakeModelRouteRepo) List(ctx context.Context, operation string, modelID string, providerID uint, isEnabled *bool, page, pageSize int) ([]*model.ModelRoute, int64, error) {
	filtered := make([]*model.ModelRoute, 0)
	op := strings.TrimSpace(operation)
	targetModel := strings.TrimSpace(modelID)
	for _, item := range r.items {
		if op != "" && model.NormalizeOperation(item.Operation) != model.NormalizeOperation(op) {
			continue
		}
		if targetModel != "" && strings.TrimSpace(item.ModelID) != targetModel {
			continue
		}
		if providerID > 0 && item.ProviderID != providerID {
			continue
		}
		if isEnabled != nil && item.IsEnabled != *isEnabled {
			continue
		}
		filtered = append(filtered, r.clone(item))
	}
	return filtered, int64(len(filtered)), nil
}

func (r *fakeModelRouteRepo) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	for _, id := range ids {
		if item, ok := r.items[id]; ok {
			item.IsEnabled = enabled
			r.items[id] = item
		}
	}
	return nil
}

func (r *fakeModelRouteRepo) BatchUpdatePriority(ctx context.Context, updates map[uint]int) error {
	for id, priority := range updates {
		if item, ok := r.items[id]; ok {
			item.Priority = priority
			r.items[id] = item
		}
	}
	return nil
}

func (r *fakeModelRouteRepo) GetStats(ctx context.Context, operation string, modelID string) (*model.ModelRouteStats, error) {
	stats := &model.ModelRouteStats{}
	op := strings.TrimSpace(operation)
	targetModel := strings.TrimSpace(modelID)
	modelSet := map[string]struct{}{}
	providerSet := map[uint]struct{}{}
	for _, item := range r.items {
		if op != "" && model.NormalizeOperation(item.Operation) != model.NormalizeOperation(op) {
			continue
		}
		if targetModel != "" && strings.TrimSpace(item.ModelID) != targetModel {
			continue
		}
		stats.TotalRoutes++
		if item.IsEnabled {
			stats.EnabledRoutes++
		} else {
			stats.DisabledRoutes++
		}
		modelSet[item.ModelID] = struct{}{}
		providerSet[item.ProviderID] = struct{}{}
	}
	stats.DistinctModels = int64(len(modelSet))
	stats.DistinctProviders = int64(len(providerSet))
	return stats, nil
}

func (r *fakeModelRouteRepo) GetByRouteKey(ctx context.Context, operation string, modelID string) ([]*model.ModelRoute, error) {
	filtered := make([]*model.ModelRoute, 0)
	op := model.NormalizeOperation(operation)
	targetModel := strings.TrimSpace(modelID)
	for _, item := range r.items {
		if !item.IsEnabled {
			continue
		}
		if model.NormalizeOperation(item.Operation) == op && strings.TrimSpace(item.ModelID) == targetModel {
			filtered = append(filtered, r.clone(item))
		}
	}
	return filtered, nil
}

type fakeModelRouteModelReader struct {
	models map[string]*model.Model
}

func (r *fakeModelRouteModelReader) GetByID(id string) (*model.Model, error) {
	if r.models == nil {
		return nil, nil
	}
	return r.models[id], nil
}

type fakeModelRouteProviderReader struct {
	providers map[uint]*model.ModelProvider
}

func (r *fakeModelRouteProviderReader) GetByID(ctx context.Context, id uint) (*model.ModelProvider, error) {
	if r.providers == nil {
		return nil, nil
	}
	return r.providers[id], nil
}

func TestModelRouteServiceCreateSuccess(t *testing.T) {
	repo := newFakeModelRouteRepo()
	svc := NewModelRouteService(
		repo,
		&fakeModelRouteModelReader{models: map[string]*model.Model{
			"gpt-4o-mini": {ID: "gpt-4o-mini"},
		}},
		&fakeModelRouteProviderReader{providers: map[uint]*model.ModelProvider{
			10: {ID: 10, Operation: model.OperationChatCompletions, ModelID: "gpt-4o-mini"},
		}},
	)

	created, err := svc.Create(context.Background(), &CreateModelRouteRequest{
		ModelID:    "gpt-4o-mini",
		ProviderID: 10,
		Priority:   5,
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if created == nil {
		t.Fatalf("expected non-nil route")
	}
	if created.RouteKey == "" {
		t.Fatalf("expected route_key to be generated")
	}
	if created.Operation != model.OperationChatCompletions {
		t.Fatalf("unexpected operation: %s", created.Operation)
	}
	if created.Priority != 5 {
		t.Fatalf("unexpected priority: %d", created.Priority)
	}
}

func TestModelRouteServiceCreateProviderMismatch(t *testing.T) {
	repo := newFakeModelRouteRepo()
	svc := NewModelRouteService(
		repo,
		&fakeModelRouteModelReader{models: map[string]*model.Model{
			"gpt-4o-mini": {ID: "gpt-4o-mini"},
		}},
		&fakeModelRouteProviderReader{providers: map[uint]*model.ModelProvider{
			10: {ID: 10, Operation: model.OperationEmbeddings, ModelID: "text-embedding-3-large"},
		}},
	)

	_, err := svc.Create(context.Background(), &CreateModelRouteRequest{
		Operation:  model.OperationChatCompletions,
		ModelID:    "gpt-4o-mini",
		ProviderID: 10,
	})
	if err == nil {
		t.Fatalf("expected mismatch error")
	}
	if !strings.Contains(err.Error(), "不匹配") {
		t.Fatalf("expected mismatch message, got %v", err)
	}
}

func TestModelRouteServiceBatchUpdatePriority(t *testing.T) {
	repo := newFakeModelRouteRepo()
	_ = repo.Create(context.Background(), &model.ModelRoute{
		Operation:  model.OperationChatCompletions,
		ModelID:    "gpt-4o-mini",
		ProviderID: 10,
		Priority:   100,
		IsEnabled:  true,
	})
	_ = repo.Create(context.Background(), &model.ModelRoute{
		Operation:  model.OperationChatCompletions,
		ModelID:    "gpt-4o-mini",
		ProviderID: 11,
		Priority:   200,
		IsEnabled:  true,
	})

	svc := NewModelRouteService(
		repo,
		&fakeModelRouteModelReader{models: map[string]*model.Model{
			"gpt-4o-mini": {ID: "gpt-4o-mini"},
		}},
		&fakeModelRouteProviderReader{providers: map[uint]*model.ModelProvider{
			10: {ID: 10, Operation: model.OperationChatCompletions, ModelID: "gpt-4o-mini"},
			11: {ID: 11, Operation: model.OperationChatCompletions, ModelID: "gpt-4o-mini"},
		}},
	)

	err := svc.BatchUpdatePriority(context.Background(), []ModelRoutePriorityItem{
		{ID: 1, Priority: 1},
		{ID: 2, Priority: 2},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}

	item1, _ := repo.GetByID(context.Background(), 1)
	item2, _ := repo.GetByID(context.Background(), 2)
	if item1.Priority != 1 || item2.Priority != 2 {
		t.Fatalf("unexpected priorities: %d, %d", item1.Priority, item2.Priority)
	}
}
