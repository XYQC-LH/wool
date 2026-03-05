package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

type mockBillingService struct {
	getStatisticsFn   func(ctx context.Context, startDate, endDate time.Time) (*service.BillingStatistics, error)
	getUsageFn        func(ctx context.Context, query *service.BillingUsageQuery) ([]*service.BillingUsageItem, error)
	getCostAnalysisFn func(ctx context.Context, query *service.BillingCostAnalysisQuery) (*service.BillingCostAnalysis, error)
}

func (m *mockBillingService) GetStatistics(ctx context.Context, startDate, endDate time.Time) (*service.BillingStatistics, error) {
	if m.getStatisticsFn != nil {
		return m.getStatisticsFn(ctx, startDate, endDate)
	}
	return &service.BillingStatistics{}, nil
}

func (m *mockBillingService) GetUsage(ctx context.Context, query *service.BillingUsageQuery) ([]*service.BillingUsageItem, error) {
	if m.getUsageFn != nil {
		return m.getUsageFn(ctx, query)
	}
	return []*service.BillingUsageItem{}, nil
}

func (m *mockBillingService) GetCostAnalysis(ctx context.Context, query *service.BillingCostAnalysisQuery) (*service.BillingCostAnalysis, error) {
	if m.getCostAnalysisFn != nil {
		return m.getCostAnalysisFn(ctx, query)
	}
	return &service.BillingCostAnalysis{}, nil
}

func setupAdminBillingRouter(handler *AdminBillingHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/admin")
	handler.RegisterRoutes(admin)
	return r
}

func TestAdminBillingHandlerGetStatistics(t *testing.T) {
	router := setupAdminBillingRouter(NewAdminBillingHandler(&mockBillingService{
		getStatisticsFn: func(ctx context.Context, startDate, endDate time.Time) (*service.BillingStatistics, error) {
			return &service.BillingStatistics{
				TotalRequests: 10,
				TotalRevenue:  20,
				TotalCost:     8,
				TotalProfit:   12,
			}, nil
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/billing/statistics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminBillingHandlerGetUsageInvalidGroup(t *testing.T) {
	router := setupAdminBillingRouter(NewAdminBillingHandler(&mockBillingService{
		getUsageFn: func(ctx context.Context, query *service.BillingUsageQuery) ([]*service.BillingUsageItem, error) {
			return nil, service.ErrInvalidBillingGroupBy
		},
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/billing/usage?group_by=invalid", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

