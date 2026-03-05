package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type mockAdminTokenService struct {
	adminCreateFn  func(req *model.AdminCreateTokenRequest) (*model.Token, error)
	adminGetByIDFn func(id uuid.UUID) (*model.AdminTokenResponse, error)
	adminListFn    func(page, pageSize int, filters map[string]interface{}) ([]*model.AdminTokenResponse, *model.Pagination, error)
	adminUpdateFn  func(id uuid.UUID, req *model.UpdateTokenRequest) (*model.AdminTokenResponse, error)
	adminDeleteFn  func(id uuid.UUID) error
	adminUsageFn   func(id uuid.UUID) (*model.TokenUsageStats, error)
}

func (m *mockAdminTokenService) AdminCreate(req *model.AdminCreateTokenRequest) (*model.Token, error) {
	if m.adminCreateFn != nil {
		return m.adminCreateFn(req)
	}
	return nil, nil
}

func (m *mockAdminTokenService) AdminGetByID(id uuid.UUID) (*model.AdminTokenResponse, error) {
	if m.adminGetByIDFn != nil {
		return m.adminGetByIDFn(id)
	}
	return nil, nil
}

func (m *mockAdminTokenService) AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AdminTokenResponse, *model.Pagination, error) {
	if m.adminListFn != nil {
		return m.adminListFn(page, pageSize, filters)
	}
	return nil, nil, nil
}

func (m *mockAdminTokenService) AdminUpdate(id uuid.UUID, req *model.UpdateTokenRequest) (*model.AdminTokenResponse, error) {
	if m.adminUpdateFn != nil {
		return m.adminUpdateFn(id, req)
	}
	return nil, nil
}

func (m *mockAdminTokenService) AdminDelete(id uuid.UUID) error {
	if m.adminDeleteFn != nil {
		return m.adminDeleteFn(id)
	}
	return nil
}

func (m *mockAdminTokenService) AdminGetUsage(id uuid.UUID) (*model.TokenUsageStats, error) {
	if m.adminUsageFn != nil {
		return m.adminUsageFn(id)
	}
	return nil, nil
}

func setupAdminTokenTestRouter(svc *mockAdminTokenService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	adminGroup := r.Group("/api/admin")
	NewAdminTokenHandler(svc).RegisterRoutes(adminGroup)
	return r
}

func TestAdminTokenHandlerListTokensSuccess(t *testing.T) {
	userID := uuid.New()
	called := false

	router := setupAdminTokenTestRouter(&mockAdminTokenService{
		adminListFn: func(page, pageSize int, filters map[string]interface{}) ([]*model.AdminTokenResponse, *model.Pagination, error) {
			called = true
			if page != 1 || pageSize != 20 {
				t.Fatalf("unexpected pagination: page=%d pageSize=%d", page, pageSize)
			}
			if got, ok := filters["status"]; !ok || got != "active" {
				t.Fatalf("unexpected status filter: %v", filters["status"])
			}
			if got, ok := filters["keyword"]; !ok || got != "demo" {
				t.Fatalf("unexpected keyword filter: %v", filters["keyword"])
			}
			if got, ok := filters["user_id"]; !ok || got != userID {
				t.Fatalf("unexpected user_id filter: %v", filters["user_id"])
			}

			return []*model.AdminTokenResponse{
					{
						ID:     uuid.New(),
						UserID: userID,
						Name:   "demo-token",
						Status: model.TokenStatusActive,
						Usage: &model.TokenUsageStats{
							RequestCount: 10,
							TotalTokens:  1234,
							TotalCost:    decimal.NewFromFloat(2.5),
						},
					},
				},
				model.NewPagination(1, 20, 1),
				nil
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/tokens?status=active&keyword=demo&user_id="+userID.String(), nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}
	if !called {
		t.Fatalf("expected adminList to be called")
	}
}

func TestAdminTokenHandlerListTokensInvalidUserID(t *testing.T) {
	router := setupAdminTokenTestRouter(&mockAdminTokenService{})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/tokens?user_id=invalid-user-id", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d, body=%s", w.Code, w.Body.String())
	}
}

func TestAdminTokenHandlerCreateTokenSuccess(t *testing.T) {
	userID := uuid.New()
	now := time.Now()

	router := setupAdminTokenTestRouter(&mockAdminTokenService{
		adminCreateFn: func(req *model.AdminCreateTokenRequest) (*model.Token, error) {
			if req.UserID != userID {
				t.Fatalf("unexpected user id: %v", req.UserID)
			}
			if req.Name != "team-token" {
				t.Fatalf("unexpected token name: %s", req.Name)
			}

			return &model.Token{
				ID:        uuid.New(),
				Key:       "sk-admin-created-key",
				UserID:    userID,
				Name:      req.Name,
				Status:    model.TokenStatusActive,
				CreatedAt: now,
			}, nil
		},
	})

	body := map[string]interface{}{
		"user_id": userID.String(),
		"name":    "team-token",
	}
	raw, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/admin/tokens", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d, body=%s", w.Code, w.Body.String())
	}

	var resp model.Response
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}
	data, ok := resp.Data.(map[string]interface{})
	if !ok {
		t.Fatalf("unexpected response data type: %T", resp.Data)
	}
	if gotKey, ok := data["key"].(string); !ok || gotKey != "sk-admin-created-key" {
		t.Fatalf("unexpected key in response: %v", data["key"])
	}
}

func TestAdminTokenHandlerGetTokenUsageNotFound(t *testing.T) {
	router := setupAdminTokenTestRouter(&mockAdminTokenService{
		adminUsageFn: func(id uuid.UUID) (*model.TokenUsageStats, error) {
			return nil, errors.New("Token 不存在")
		},
	})

	req := httptest.NewRequest(http.MethodGet, "/api/admin/tokens/"+uuid.New().String()+"/usage", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("expected status 404, got %d, body=%s", w.Code, w.Body.String())
	}
}
