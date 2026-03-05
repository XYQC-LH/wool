package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
)

type mockAdminModelService struct {
	batchUpdateStatusFn func(ids []string, enabled bool) error
	batchDeleteFn       func(ids []string) error
	getAdminStatsFn     func() (*model.ModelAdminStats, error)
}

func (m *mockAdminModelService) ListEnabled() ([]*model.ModelResponse, error) {
	return nil, nil
}

func (m *mockAdminModelService) ListPublic() ([]*model.Model, error) {
	return nil, nil
}

func (m *mockAdminModelService) GetByID(id string) (*model.ModelResponse, error) {
	return nil, nil
}

func (m *mockAdminModelService) AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.ModelResponse, *model.Pagination, error) {
	return nil, nil, nil
}

func (m *mockAdminModelService) AdminGetByID(id string) (*model.ModelResponse, error) {
	return nil, nil
}

func (m *mockAdminModelService) Create(req *model.CreateModelRequest) (*model.ModelResponse, error) {
	return nil, nil
}

func (m *mockAdminModelService) Update(id string, req *model.UpdateModelRequest) (*model.ModelResponse, error) {
	return nil, nil
}

func (m *mockAdminModelService) Delete(id string) error {
	return nil
}

func (m *mockAdminModelService) BatchDelete(ids []string) error {
	if m.batchDeleteFn != nil {
		return m.batchDeleteFn(ids)
	}
	return nil
}

func (m *mockAdminModelService) UpdateStatus(id string, enabled bool) error {
	return nil
}

func (m *mockAdminModelService) BatchUpdateStatus(ids []string, enabled bool) error {
	if m.batchUpdateStatusFn != nil {
		return m.batchUpdateStatusFn(ids, enabled)
	}
	return nil
}

func (m *mockAdminModelService) GetAdminStats() (*model.ModelAdminStats, error) {
	if m.getAdminStatsFn != nil {
		return m.getAdminStatsFn()
	}
	return nil, nil
}

func setupAdminModelRouter(handler *AdminHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/admin")
	admin.POST("/models/batch/status", handler.BatchUpdateModelStatus)
	admin.POST("/models/batch/delete", handler.BatchDeleteModels)
	admin.GET("/models/stats", handler.GetModelStats)
	return r
}

func TestAdminHandlerBatchUpdateModelStatus(t *testing.T) {
	called := false
	adminHandler := &AdminHandler{
		modelService: &mockAdminModelService{
			batchUpdateStatusFn: func(ids []string, enabled bool) error {
				called = true
				if len(ids) != 2 {
					t.Fatalf("unexpected ids length: %d", len(ids))
				}
				if enabled {
					t.Fatalf("expected enabled=false")
				}
				return nil
			},
		},
	}
	router := setupAdminModelRouter(adminHandler)

	body := map[string]interface{}{
		"ids":     []string{"gpt-4o-mini", "text-embedding-3-large"},
		"enabled": false,
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/models/batch/status", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected BatchUpdateStatus to be called")
	}
}

func TestAdminHandlerBatchDeleteModels(t *testing.T) {
	called := false
	adminHandler := &AdminHandler{
		modelService: &mockAdminModelService{
			batchDeleteFn: func(ids []string) error {
				called = true
				if len(ids) != 2 {
					t.Fatalf("unexpected ids length: %d", len(ids))
				}
				return nil
			},
		},
	}
	router := setupAdminModelRouter(adminHandler)

	body := map[string]interface{}{
		"ids": []string{"gpt-4o-mini", "text-embedding-3-large"},
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/models/batch/delete", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected BatchDelete to be called")
	}
}

func TestAdminHandlerGetModelStats(t *testing.T) {
	adminHandler := &AdminHandler{
		modelService: &mockAdminModelService{
			getAdminStatsFn: func() (*model.ModelAdminStats, error) {
				return &model.ModelAdminStats{
					TotalModels:      10,
					EnabledModels:    8,
					DisabledModels:   2,
					ActiveModels:     7,
					DeprecatedModels: 1,
				}, nil
			},
		},
	}
	router := setupAdminModelRouter(adminHandler)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/models/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}
