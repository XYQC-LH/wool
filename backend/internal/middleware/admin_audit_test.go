package middleware

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type mockAuditService struct {
	lastInput *service.CreateAuditLogInput
}

func (m *mockAuditService) Record(ctx context.Context, input *service.CreateAuditLogInput) error {
	m.lastInput = input
	return nil
}

func (m *mockAuditService) List(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLogResponse, *model.Pagination, error) {
	return []*model.AuditLogResponse{}, model.NewPagination(page, pageSize, 0), nil
}

func TestAdminAuditMiddlewareWritesAuditLog(t *testing.T) {
	gin.SetMode(gin.TestMode)

	mockSvc := &mockAuditService{}
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set(ContextKeyUserID, uuid.New())
		c.Set(ContextKeyRole, model.RoleAdmin)
		c.Next()
	})
	r.Use(AdminAuditMiddleware(mockSvc))
	r.GET("/api/admin/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", w.Code)
	}
	if mockSvc.lastInput == nil {
		t.Fatalf("expected audit log recorded")
	}
	if mockSvc.lastInput.Method != http.MethodGet {
		t.Fatalf("expected method GET, got %s", mockSvc.lastInput.Method)
	}
	if mockSvc.lastInput.Path == "" {
		t.Fatalf("expected non-empty path")
	}
}
