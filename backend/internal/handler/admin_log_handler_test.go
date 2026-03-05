package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockAdminLogService struct {
	adminListFn     func(page, pageSize int, filters map[string]interface{}) ([]*model.AdminLogResponse, *model.Pagination, error)
	adminGetStatsFn func(startDate, endDate string) (*service.AdminStatsResponse, error)
}

func (m *mockAdminLogService) List(userID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]*model.LogResponse, *model.Pagination, error) {
	return []*model.LogResponse{}, model.NewPagination(page, pageSize, 0), nil
}

func (m *mockAdminLogService) GetStats(userID uuid.UUID, startDate, endDate string) (*service.UserStatsResponse, error) {
	return &service.UserStatsResponse{}, nil
}

func (m *mockAdminLogService) AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AdminLogResponse, *model.Pagination, error) {
	if m.adminListFn != nil {
		return m.adminListFn(page, pageSize, filters)
	}
	return []*model.AdminLogResponse{}, model.NewPagination(page, pageSize, 0), nil
}

func (m *mockAdminLogService) AdminGetStats(startDate, endDate string) (*service.AdminStatsResponse, error) {
	if m.adminGetStatsFn != nil {
		return m.adminGetStatsFn(startDate, endDate)
	}
	return &service.AdminStatsResponse{
		Summary: &service.StatsSummary{
			TotalCost: decimal.Zero,
		},
	}, nil
}

type mockAuditLogService struct {
	listFn func(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLogResponse, *model.Pagination, error)
}

func (m *mockAuditLogService) Record(ctx context.Context, input *service.CreateAuditLogInput) error {
	return nil
}

func (m *mockAuditLogService) List(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLogResponse, *model.Pagination, error) {
	if m.listFn != nil {
		return m.listFn(page, pageSize, filters)
	}
	return []*model.AuditLogResponse{}, model.NewPagination(page, pageSize, 0), nil
}

func setupAdminLogRouter(handler *AdminLogHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	admin := r.Group("/api/admin")
	handler.RegisterRoutes(admin)
	return r
}

func TestAdminLogHandlerListLogsSuccess(t *testing.T) {
	router := setupAdminLogRouter(NewAdminLogHandler(
		&mockAdminLogService{
			adminListFn: func(page, pageSize int, filters map[string]interface{}) ([]*model.AdminLogResponse, *model.Pagination, error) {
				return []*model.AdminLogResponse{
					{Model: "gpt-4o-mini"},
				}, model.NewPagination(page, pageSize, 1), nil
			},
		},
		&mockAuditLogService{},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs?page=1&page_size=20", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminLogHandlerGetStatsSuccess(t *testing.T) {
	router := setupAdminLogRouter(NewAdminLogHandler(
		&mockAdminLogService{
			adminGetStatsFn: func(startDate, endDate string) (*service.AdminStatsResponse, error) {
				return &service.AdminStatsResponse{
					Summary: &service.StatsSummary{
						TotalRequests: 10,
						TotalTokens:   1000,
						TotalCost:     decimal.NewFromFloat(2.5),
					},
				}, nil
			},
		},
		&mockAuditLogService{},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminLogHandlerListAuditLogsSuccess(t *testing.T) {
	router := setupAdminLogRouter(NewAdminLogHandler(
		&mockAdminLogService{},
		&mockAuditLogService{
			listFn: func(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLogResponse, *model.Pagination, error) {
				return []*model.AuditLogResponse{
					{Action: "PUT /api/admin/channels/:id/status"},
				}, model.NewPagination(page, pageSize, 1), nil
			},
		},
	))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs/audit?page=1&page_size=20&success=true", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminLogHandlerListAuditLogsInvalidSuccess(t *testing.T) {
	router := setupAdminLogRouter(NewAdminLogHandler(&mockAdminLogService{}, &mockAuditLogService{}))

	req := httptest.NewRequest(http.MethodGet, "/api/admin/logs/audit?success=maybe", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}
