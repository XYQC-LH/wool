package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type mockQuotaService struct {
	createFn     func(ctx context.Context, input *service.CreateQuotaPolicyInput) (*model.QuotaPolicy, error)
	listFn       func(ctx context.Context, keyword string, status model.QuotaPolicyStatus, page, pageSize int) ([]*model.QuotaPolicy, int64, error)
	getStatsFn   func(ctx context.Context, date time.Time) (*service.QuotaStats, error)
	getPolicyFn  func(ctx context.Context, id string) (*model.QuotaPolicy, error)
	updateFn     func(ctx context.Context, id string, input *service.UpdateQuotaPolicyInput) (*model.QuotaPolicy, error)
	deleteFn     func(ctx context.Context, id string) error
	monitorFn    func(ctx context.Context, date time.Time, tenantID string) ([]*service.QuotaMonitoringItem, error)
	checkAlertFn func(ctx context.Context, date time.Time) (*service.QuotaAlertCheckResult, error)
}

func (m *mockQuotaService) Create(ctx context.Context, input *service.CreateQuotaPolicyInput) (*model.QuotaPolicy, error) {
	if m.createFn != nil {
		return m.createFn(ctx, input)
	}
	return nil, nil
}

func (m *mockQuotaService) Update(ctx context.Context, id string, input *service.UpdateQuotaPolicyInput) (*model.QuotaPolicy, error) {
	if m.updateFn != nil {
		return m.updateFn(ctx, id, input)
	}
	return nil, nil
}

func (m *mockQuotaService) Delete(ctx context.Context, id string) error {
	if m.deleteFn != nil {
		return m.deleteFn(ctx, id)
	}
	return nil
}

func (m *mockQuotaService) GetByID(ctx context.Context, id string) (*model.QuotaPolicy, error) {
	if m.getPolicyFn != nil {
		return m.getPolicyFn(ctx, id)
	}
	return nil, nil
}

func (m *mockQuotaService) List(ctx context.Context, keyword string, status model.QuotaPolicyStatus, page, pageSize int) ([]*model.QuotaPolicy, int64, error) {
	if m.listFn != nil {
		return m.listFn(ctx, keyword, status, page, pageSize)
	}
	return []*model.QuotaPolicy{}, 0, nil
}

func (m *mockQuotaService) GetStats(ctx context.Context, date time.Time) (*service.QuotaStats, error) {
	if m.getStatsFn != nil {
		return m.getStatsFn(ctx, date)
	}
	return &service.QuotaStats{}, nil
}

func (m *mockQuotaService) GetMonitoring(ctx context.Context, date time.Time, tenantID string) ([]*service.QuotaMonitoringItem, error) {
	if m.monitorFn != nil {
		return m.monitorFn(ctx, date, tenantID)
	}
	return []*service.QuotaMonitoringItem{}, nil
}

func (m *mockQuotaService) CheckAlerts(ctx context.Context, date time.Time) (*service.QuotaAlertCheckResult, error) {
	if m.checkAlertFn != nil {
		return m.checkAlertFn(ctx, date)
	}
	return &service.QuotaAlertCheckResult{}, nil
}

func setupAdminQuotaRouter(handler *AdminQuotaHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/admin")
	handler.RegisterRoutes(admin)
	return r
}

func TestAdminQuotaHandlerCreate(t *testing.T) {
	called := false
	router := setupAdminQuotaRouter(NewAdminQuotaHandler(&mockQuotaService{
		createFn: func(ctx context.Context, input *service.CreateQuotaPolicyInput) (*model.QuotaPolicy, error) {
			called = true
			if input.TenantID != "tenant-a" {
				t.Fatalf("unexpected tenant_id: %s", input.TenantID)
			}
			return &model.QuotaPolicy{
				TenantID:              input.TenantID,
				Name:                  "租户A",
				DailyRequestLimit:     100,
				DailyCostLimit:        decimal.NewFromInt(20),
				AlertThresholdPercent: 80,
				Status:                model.QuotaPolicyStatusActive,
			}, nil
		},
	}))

	body := map[string]interface{}{
		"tenant_id":               "tenant-a",
		"name":                    "租户A",
		"daily_request_limit":     100,
		"daily_cost_limit":        20.0,
		"alert_threshold_percent": 80,
		"status":                  "active",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/quotas", bytes.NewReader(raw))
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

func TestAdminQuotaHandlerListInvalidStatus(t *testing.T) {
	router := setupAdminQuotaRouter(NewAdminQuotaHandler(&mockQuotaService{}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/quotas?status=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

