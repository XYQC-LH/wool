package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockMetricsService struct {
	queryFn       func(ctx context.Context, query *service.MetricsQuery) ([]*service.MetricsQueryItem, *model.Pagination, error)
	getRealtimeFn func(ctx context.Context, window time.Duration) (*service.RealtimeMetrics, error)
}

func (m *mockMetricsService) Query(ctx context.Context, query *service.MetricsQuery) ([]*service.MetricsQueryItem, *model.Pagination, error) {
	if m.queryFn != nil {
		return m.queryFn(ctx, query)
	}
	return []*service.MetricsQueryItem{}, model.NewPagination(1, 20, 0), nil
}

func (m *mockMetricsService) GetRealtime(ctx context.Context, window time.Duration) (*service.RealtimeMetrics, error) {
	if m.getRealtimeFn != nil {
		return m.getRealtimeFn(ctx, window)
	}
	return &service.RealtimeMetrics{}, nil
}

type mockAlertService struct {
	listFn    func(page, pageSize int, filters map[string]interface{}) ([]*model.AlertResponse, *model.Pagination, error)
	resolveFn func(id uuid.UUID, resolvedBy uuid.UUID) error
	statsFn   func() (*model.AlertStats, error)
}

type mockObservabilityService struct {
	analyzePerformanceFn func(ctx context.Context, query *service.PerformanceAnalysisQuery) (*service.PerformanceAnalysisResult, error)
	forecastCapacityFn   func(ctx context.Context, query *service.CapacityForecastQuery) (*service.CapacityForecastResult, error)
}

func (m *mockObservabilityService) AnalyzePerformance(ctx context.Context, query *service.PerformanceAnalysisQuery) (*service.PerformanceAnalysisResult, error) {
	if m.analyzePerformanceFn != nil {
		return m.analyzePerformanceFn(ctx, query)
	}
	return &service.PerformanceAnalysisResult{}, nil
}

func (m *mockObservabilityService) ForecastCapacity(ctx context.Context, query *service.CapacityForecastQuery) (*service.CapacityForecastResult, error) {
	if m.forecastCapacityFn != nil {
		return m.forecastCapacityFn(ctx, query)
	}
	return &service.CapacityForecastResult{}, nil
}

func (m *mockAlertService) CreateAlert(alertType model.AlertType, severity model.AlertSeverity, title, message string, metadata model.JSON) error {
	return nil
}

func (m *mockAlertService) List(page, pageSize int, filters map[string]interface{}) ([]*model.AlertResponse, *model.Pagination, error) {
	if m.listFn != nil {
		return m.listFn(page, pageSize, filters)
	}
	return []*model.AlertResponse{}, model.NewPagination(page, pageSize, 0), nil
}

func (m *mockAlertService) GetByID(id uuid.UUID) (*model.AlertResponse, error) {
	return nil, nil
}

func (m *mockAlertService) Resolve(id uuid.UUID, resolvedBy uuid.UUID) error {
	if m.resolveFn != nil {
		return m.resolveFn(id, resolvedBy)
	}
	return nil
}

func (m *mockAlertService) GetStats() (*model.AlertStats, error) {
	if m.statsFn != nil {
		return m.statsFn()
	}
	return &model.AlertStats{}, nil
}

func (m *mockAlertService) GetActiveAlerts() ([]*model.AlertResponse, error) {
	return []*model.AlertResponse{}, nil
}

func (m *mockAlertService) CheckChannelHealth(channelID uint, errorRate float64, latency int) error {
	return nil
}

func (m *mockAlertService) CheckUserBalance(userID uuid.UUID, balance float64) error {
	return nil
}

func setupAdminMetricsRouter(handler *AdminMetricsHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/admin")
	{
		admin.Use(func(c *gin.Context) {
			c.Set("user_id", uuid.New())
			c.Next()
		})
	}
	handler.RegisterRoutes(admin)
	return r
}

func TestAdminMetricsHandlerQuerySuccess(t *testing.T) {
	router := setupAdminMetricsRouter(NewAdminMetricsHandler(
		&mockMetricsService{
			queryFn: func(ctx context.Context, query *service.MetricsQuery) ([]*service.MetricsQueryItem, *model.Pagination, error) {
				return []*service.MetricsQueryItem{
					{ProviderID: 1, ModelID: "gpt-4o-mini"},
				}, model.NewPagination(1, 20, 1), nil
			},
		},
		&mockAlertService{},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics/query?granularity=minute", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminMetricsHandlerRealtimeSuccess(t *testing.T) {
	router := setupAdminMetricsRouter(NewAdminMetricsHandler(
		&mockMetricsService{
			getRealtimeFn: func(ctx context.Context, window time.Duration) (*service.RealtimeMetrics, error) {
				return &service.RealtimeMetrics{
					RequestCount: 10,
				}, nil
			},
		},
		&mockAlertService{},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics/realtime?window_seconds=60", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminMetricsHandlerListAlertsSuccess(t *testing.T) {
	router := setupAdminMetricsRouter(NewAdminMetricsHandler(
		&mockMetricsService{},
		&mockAlertService{
			listFn: func(page, pageSize int, filters map[string]interface{}) ([]*model.AlertResponse, *model.Pagination, error) {
				return []*model.AlertResponse{
					{Title: "高延迟告警"},
				}, model.NewPagination(page, pageSize, 1), nil
			},
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics/alerts?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminMetricsHandlerResolveAlertInvalidID(t *testing.T) {
	router := setupAdminMetricsRouter(NewAdminMetricsHandler(&mockMetricsService{}, &mockAlertService{}))

	req := httptest.NewRequest(http.MethodPut, "/api/admin/metrics/alerts/invalid/resolve", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminMetricsHandlerPerformanceSuccess(t *testing.T) {
	router := setupAdminMetricsRouter(NewAdminMetricsHandler(
		&mockMetricsService{},
		&mockAlertService{},
		&mockObservabilityService{
			analyzePerformanceFn: func(ctx context.Context, query *service.PerformanceAnalysisQuery) (*service.PerformanceAnalysisResult, error) {
				return &service.PerformanceAnalysisResult{
					Summary: service.PerformanceAnalysisSummary{
						TotalRequests: 10,
						SlowRequests:  1,
					},
				}, nil
			},
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics/performance?window_seconds=600", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminMetricsHandlerCapacityForecastSuccess(t *testing.T) {
	router := setupAdminMetricsRouter(NewAdminMetricsHandler(
		&mockMetricsService{},
		&mockAlertService{},
		&mockObservabilityService{
			forecastCapacityFn: func(ctx context.Context, query *service.CapacityForecastQuery) (*service.CapacityForecastResult, error) {
				return &service.CapacityForecastResult{
					LookbackHours: 24,
					ForecastHours: 24,
				}, nil
			},
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/metrics/capacity/forecast?lookback_hours=24&forecast_hours=24", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}
