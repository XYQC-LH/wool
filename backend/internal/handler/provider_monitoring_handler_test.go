package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

type mockProviderEntityReader struct {
	getByIDFn func(ctx context.Context, id uint) (*model.ModelProvider, error)
}

func (m *mockProviderEntityReader) GetByID(ctx context.Context, id uint) (*model.ModelProvider, error) {
	if m.getByIDFn != nil {
		return m.getByIDFn(ctx, id)
	}
	return nil, nil
}

type mockProviderMetricsReader struct {
	getAggregatedMetricsFn func(ctx context.Context, providerID uint, startTime, endTime time.Time) (*model.AggregatedMetrics, error)
	getTimeSeriesFn        func(ctx context.Context, providerID uint, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.TimeSeriesMetric, error)
	getCircuitEventsFn     func(ctx context.Context, providerID uint, startTime, endTime time.Time) ([]*model.CircuitEventRecord, error)
	getTrafficDistFn       func(ctx context.Context, modelID string, startTime, endTime time.Time) ([]*model.ProviderTrafficDistribution, error)
}

func (m *mockProviderMetricsReader) GetAggregatedMetrics(ctx context.Context, providerID uint, startTime, endTime time.Time) (*model.AggregatedMetrics, error) {
	if m.getAggregatedMetricsFn != nil {
		return m.getAggregatedMetricsFn(ctx, providerID, startTime, endTime)
	}
	return nil, nil
}

func (m *mockProviderMetricsReader) GetTimeSeries(ctx context.Context, providerID uint, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.TimeSeriesMetric, error) {
	if m.getTimeSeriesFn != nil {
		return m.getTimeSeriesFn(ctx, providerID, granularity, startTime, endTime)
	}
	return nil, nil
}

func (m *mockProviderMetricsReader) GetCircuitEvents(ctx context.Context, providerID uint, startTime, endTime time.Time) ([]*model.CircuitEventRecord, error) {
	if m.getCircuitEventsFn != nil {
		return m.getCircuitEventsFn(ctx, providerID, startTime, endTime)
	}
	return nil, nil
}

func (m *mockProviderMetricsReader) GetTrafficDistribution(ctx context.Context, modelID string, startTime, endTime time.Time) ([]*model.ProviderTrafficDistribution, error) {
	if m.getTrafficDistFn != nil {
		return m.getTrafficDistFn(ctx, modelID, startTime, endTime)
	}
	return nil, nil
}

func setupProviderMonitoringRouter(providerRepo providerEntityReader, metricsRepo providerMetricsReader) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminGroup := r.Group("/api/admin")
	NewProviderMonitoringHandler(providerRepo, metricsRepo).RegisterRoutes(adminGroup)
	return r
}

func TestProviderMonitoringHandlerGetProviderStatsSuccess(t *testing.T) {
	router := setupProviderMonitoringRouter(
		&mockProviderEntityReader{
			getByIDFn: func(ctx context.Context, id uint) (*model.ModelProvider, error) {
				return &model.ModelProvider{ID: id}, nil
			},
		},
		&mockProviderMetricsReader{
			getAggregatedMetricsFn: func(ctx context.Context, providerID uint, startTime, endTime time.Time) (*model.AggregatedMetrics, error) {
				return &model.AggregatedMetrics{
					TotalRequests: 120,
					TotalSuccess:  100,
					TotalFailure:  20,
					SuccessRate:   83.33,
					TotalCost:     decimal.NewFromFloat(12.5),
				}, nil
			},
		},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/providers/1/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestProviderMonitoringHandlerGetProviderStatsInvalidID(t *testing.T) {
	router := setupProviderMonitoringRouter(
		&mockProviderEntityReader{},
		&mockProviderMetricsReader{},
	)

	req := httptest.NewRequest(http.MethodGet, "/api/admin/providers/invalid/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}
