package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

type mockModelRouteService struct {
	createFn             func(ctx context.Context, req *service.CreateModelRouteRequest) (*model.ModelRoute, error)
	updateFn             func(ctx context.Context, id uint, req *service.UpdateModelRouteRequest) (*model.ModelRoute, error)
	deleteFn             func(ctx context.Context, id uint) error
	getByIDFn            func(ctx context.Context, id uint) (*model.ModelRoute, error)
	listFn               func(ctx context.Context, params *service.ModelRouteQueryParams) ([]*model.ModelRoute, int64, error)
	batchUpdateEnabledFn func(ctx context.Context, ids []uint, enabled bool) error
	batchUpdatePriorityFn func(ctx context.Context, items []service.ModelRoutePriorityItem) error
	getStatsFn           func(ctx context.Context, operation string, modelID string) (*model.ModelRouteStats, error)
}

func (m *mockModelRouteService) Create(ctx context.Context, req *service.CreateModelRouteRequest) (*model.ModelRoute, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, nil
}

func (m *mockModelRouteService) Update(ctx context.Context, id uint, req *service.UpdateModelRouteRequest) (*model.ModelRoute, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}

func (m *mockModelRouteService) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockModelRouteService) GetByID(ctx context.Context, id uint) (*model.ModelRoute, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockModelRouteService) List(ctx context.Context, params *service.ModelRouteQueryParams) ([]*model.ModelRoute, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, 0, nil
}

func (m *mockModelRouteService) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if m.batchUpdateEnabledFn != nil {
		return m.batchUpdateEnabledFn(ctx, ids, enabled)
	}
	return nil
}

func (m *mockModelRouteService) BatchUpdatePriority(ctx context.Context, items []service.ModelRoutePriorityItem) error {
	if m.batchUpdatePriorityFn != nil {
		return m.batchUpdatePriorityFn(ctx, items)
	}
	return nil
}

func (m *mockModelRouteService) GetStats(ctx context.Context, operation string, modelID string) (*model.ModelRouteStats, error) {
	if m.getStatsFn != nil {
		return m.getStatsFn(ctx, operation, modelID)
	}
	return nil, nil
}

func setupModelRouteRouter(svc service.ModelRouteService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/admin")
	NewModelRouteHandler(svc).RegisterRoutes(admin)
	return r
}

func TestModelRouteHandlerCreateSuccess(t *testing.T) {
	called := false
	router := setupModelRouteRouter(&mockModelRouteService{
		createFn: func(ctx context.Context, req *service.CreateModelRouteRequest) (*model.ModelRoute, error) {
			called = true
			if req.ModelID != "gpt-4o-mini" {
				t.Fatalf("unexpected model id: %s", req.ModelID)
			}
			if req.ProviderID != 10 {
				t.Fatalf("unexpected provider id: %d", req.ProviderID)
			}
			return &model.ModelRoute{
				ID:         1,
				Operation:  model.OperationChatCompletions,
				ModelID:    req.ModelID,
				ProviderID: req.ProviderID,
			}, nil
		},
	})

	body := map[string]interface{}{
		"operation":   model.OperationChatCompletions,
		"model_id":    "gpt-4o-mini",
		"provider_id": 10,
		"priority":    1,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/model-routes", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected service Create to be called")
	}
}

func TestModelRouteHandlerListWithFilters(t *testing.T) {
	called := false
	router := setupModelRouteRouter(&mockModelRouteService{
		listFn: func(ctx context.Context, params *service.ModelRouteQueryParams) ([]*model.ModelRoute, int64, error) {
			called = true
			if params.Operation != model.OperationEmbeddings {
				t.Fatalf("unexpected operation: %s", params.Operation)
			}
			if params.ModelID != "text-embedding-3-large" {
				t.Fatalf("unexpected model id: %s", params.ModelID)
			}
			if params.ProviderID != 8 {
				t.Fatalf("unexpected provider id: %d", params.ProviderID)
			}
			if params.IsEnabled == nil || !*params.IsEnabled {
				t.Fatalf("expected is_enabled=true")
			}
			if params.Page != 2 || params.PageSize != 10 {
				t.Fatalf("unexpected pagination: page=%d pageSize=%d", params.Page, params.PageSize)
			}
			return []*model.ModelRoute{
				{ID: 1, Operation: params.Operation, ModelID: params.ModelID, ProviderID: params.ProviderID, IsEnabled: true},
			}, 1, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/model-routes?operation=embeddings&model_id=text-embedding-3-large&provider_id=8&is_enabled=true&page=2&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected service List to be called")
	}
}

func TestModelRouteHandlerBatchUpdatePriority(t *testing.T) {
	called := false
	router := setupModelRouteRouter(&mockModelRouteService{
		batchUpdatePriorityFn: func(ctx context.Context, items []service.ModelRoutePriorityItem) error {
			called = true
			if len(items) != 2 {
				t.Fatalf("unexpected items length: %d", len(items))
			}
			return nil
		},
	})

	body := map[string]interface{}{
		"items": []map[string]interface{}{
			{"id": 1, "priority": 10},
			{"id": 2, "priority": 20},
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/model-routes/batch/priority", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected BatchUpdatePriority to be called")
	}
}
