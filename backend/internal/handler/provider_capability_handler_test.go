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

type mockProviderCapabilityService struct {
	createFn              func(ctx context.Context, req *service.CreateProviderCapabilityRequest) (*model.ProviderCapability, error)
	updateFn              func(ctx context.Context, id uint, req *service.UpdateProviderCapabilityRequest) (*model.ProviderCapability, error)
	deleteFn              func(ctx context.Context, id uint) error
	getByIDFn             func(ctx context.Context, id uint) (*model.ProviderCapability, error)
	listFn                func(ctx context.Context, params *service.ProviderCapabilityQueryParams) ([]*model.ProviderCapability, int64, error)
	listByProviderFn      func(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error)
	batchUpdateEnabledFn  func(ctx context.Context, ids []uint, enabled bool) error
	getSummaryFn          func(ctx context.Context, providerID uint) (*model.ProviderCapabilitySummary, error)
	validateConstraintsFn func(ctx context.Context, req *service.ValidateProviderConstraintsRequest) (*model.ProviderCapabilityValidationResult, error)
}

func (m *mockProviderCapabilityService) Create(ctx context.Context, req *service.CreateProviderCapabilityRequest) (*model.ProviderCapability, error) {
	if m.createFn != nil {
		return m.createFn(ctx, req)
	}
	return nil, nil
}

func (m *mockProviderCapabilityService) Update(ctx context.Context, id uint, req *service.UpdateProviderCapabilityRequest) (*model.ProviderCapability, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, req)
	}
	return nil, nil
}

func (m *mockProviderCapabilityService) Delete(ctx context.Context, id uint) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockProviderCapabilityService) GetByID(ctx context.Context, id uint) (*model.ProviderCapability, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

func (m *mockProviderCapabilityService) List(ctx context.Context, params *service.ProviderCapabilityQueryParams) ([]*model.ProviderCapability, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, params)
	}
	return nil, 0, nil
}

func (m *mockProviderCapabilityService) ListByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error) {
	if m.listByProviderFn != nil {
		return m.listByProviderFn(ctx, providerID)
	}
	return nil, nil
}

func (m *mockProviderCapabilityService) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	if m.batchUpdateEnabledFn != nil {
		return m.batchUpdateEnabledFn(ctx, ids, enabled)
	}
	return nil
}

func (m *mockProviderCapabilityService) GetSummary(ctx context.Context, providerID uint) (*model.ProviderCapabilitySummary, error) {
	if m.getSummaryFn != nil {
		return m.getSummaryFn(ctx, providerID)
	}
	return nil, nil
}

func (m *mockProviderCapabilityService) ValidateConstraints(ctx context.Context, req *service.ValidateProviderConstraintsRequest) (*model.ProviderCapabilityValidationResult, error) {
	if m.validateConstraintsFn != nil {
		return m.validateConstraintsFn(ctx, req)
	}
	return nil, nil
}

func setupProviderCapabilityTestRouter(svc service.ProviderCapabilityService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminGroup := r.Group("/api/admin")
	NewProviderCapabilityHandler(svc).RegisterRoutes(adminGroup)
	return r
}

func TestProviderCapabilityHandlerCreateSuccess(t *testing.T) {
	called := false
	router := setupProviderCapabilityTestRouter(&mockProviderCapabilityService{
		createFn: func(ctx context.Context, req *service.CreateProviderCapabilityRequest) (*model.ProviderCapability, error) {
			called = true
			if req.ProviderID != 1 {
				t.Fatalf("unexpected provider id: %d", req.ProviderID)
			}
			if req.Operation != "image.generations" {
				t.Fatalf("unexpected operation: %s", req.Operation)
			}
			if req.Constraints["resolution"] == nil {
				t.Fatalf("expected resolution constraint")
			}
			return &model.ProviderCapability{
				ID:          10,
				ProviderID:  req.ProviderID,
				Operation:   req.Operation,
				Constraints: req.Constraints,
				IsEnabled:   true,
			}, nil
		},
	})

	body := map[string]interface{}{
		"provider_id": 1,
		"operation":   "image.generations",
		"constraints": map[string]interface{}{
			"resolution": []interface{}{"1k", "2k"},
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/provider-capabilities", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected Create to be called")
	}
}

func TestProviderCapabilityHandlerListWithFilters(t *testing.T) {
	called := false
	router := setupProviderCapabilityTestRouter(&mockProviderCapabilityService{
		listFn: func(ctx context.Context, params *service.ProviderCapabilityQueryParams) ([]*model.ProviderCapability, int64, error) {
			called = true
			if params.ProviderID != 12 {
				t.Fatalf("unexpected provider id: %d", params.ProviderID)
			}
			if params.Operation != "audio.transcriptions" {
				t.Fatalf("unexpected operation: %s", params.Operation)
			}
			if params.IsEnabled == nil || !*params.IsEnabled {
				t.Fatalf("expected is_enabled=true")
			}
			if params.Page != 2 || params.PageSize != 10 {
				t.Fatalf("unexpected pagination: page=%d page_size=%d", params.Page, params.PageSize)
			}
			return []*model.ProviderCapability{
				{ID: 1, ProviderID: 12, Operation: "audio.transcriptions", IsEnabled: true},
			}, 1, nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/provider-capabilities?provider_id=12&operation=audio.transcriptions&is_enabled=true&page=2&page_size=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected List to be called")
	}
}

func TestProviderCapabilityHandlerValidateConstraints(t *testing.T) {
	router := setupProviderCapabilityTestRouter(&mockProviderCapabilityService{
		validateConstraintsFn: func(ctx context.Context, req *service.ValidateProviderConstraintsRequest) (*model.ProviderCapabilityValidationResult, error) {
			if req.ProviderID != 2 {
				t.Fatalf("unexpected provider id: %d", req.ProviderID)
			}
			if req.Operation != "image.generations" {
				t.Fatalf("unexpected operation: %s", req.Operation)
			}
			return &model.ProviderCapabilityValidationResult{
				ProviderID: req.ProviderID,
				Operation:  req.Operation,
				Matched:    false,
				Reason:     "resolution:not_in_allowed_values",
			}, nil
		},
	})

	body := map[string]interface{}{
		"provider_id": 2,
		"operation":   "image.generations",
		"constraints": map[string]interface{}{
			"resolution": "4k",
		},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/provider-capabilities/validate", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}
