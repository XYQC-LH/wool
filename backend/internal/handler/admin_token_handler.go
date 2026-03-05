package handler

import (
	"net/http"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminTokenHandler 管理员 Token 处理器
type AdminTokenHandler struct {
	tokenService service.AdminTokenService
}

// NewAdminTokenHandler 创建管理员 Token 处理器
func NewAdminTokenHandler(tokenService service.AdminTokenService) *AdminTokenHandler {
	return &AdminTokenHandler{tokenService: tokenService}
}

func validTokenStatus(status model.TokenStatus) bool {
	switch status {
	case model.TokenStatusActive, model.TokenStatusDisabled, model.TokenStatusExpired:
		return true
	default:
		return false
	}
}

// ListTokens 获取 Token 列表
func (h *AdminTokenHandler) ListTokens(c *gin.Context) {
	var query model.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的查询参数"))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	filters := map[string]interface{}{}

	if rawUserID := strings.TrimSpace(c.Query("user_id")); rawUserID != "" {
		userID, err := uuid.Parse(rawUserID)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 user_id"))
			return
		}
		filters["user_id"] = userID
	}

	if rawStatus := strings.TrimSpace(c.Query("status")); rawStatus != "" {
		status := model.TokenStatus(rawStatus)
		if !validTokenStatus(status) {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 status"))
			return
		}
		filters["status"] = rawStatus
	}

	if keyword := strings.TrimSpace(c.Query("keyword")); keyword != "" {
		filters["keyword"] = keyword
	}

	tokens, pagination, err := h.tokenService.AdminList(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(tokens, pagination))
}

// GetToken 获取 Token 详情
func (h *AdminTokenHandler) GetToken(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	token, err := h.tokenService.AdminGetByID(tokenID)
	if err != nil {
		if err.Error() == "Token 不存在" {
			c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(token))
}

// CreateToken 管理员创建 Token
func (h *AdminTokenHandler) CreateToken(c *gin.Context) {
	var req model.AdminCreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	token, err := h.tokenService.AdminCreate(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"id":           token.ID,
		"key":          token.Key,
		"user_id":      token.UserID,
		"name":         token.Name,
		"status":       token.Status,
		"remain_quota": token.RemainQuota,
		"expires_at":   token.ExpiresAt,
		"created_at":   token.CreatedAt,
	}))
}

// UpdateToken 更新 Token
func (h *AdminTokenHandler) UpdateToken(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	var req model.UpdateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	token, err := h.tokenService.AdminUpdate(tokenID, &req)
	if err != nil {
		if err.Error() == "Token 不存在" {
			c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(token))
}

// DeleteToken 删除 Token
func (h *AdminTokenHandler) DeleteToken(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	if err := h.tokenService.AdminDelete(tokenID); err != nil {
		if err.Error() == "Token 不存在" {
			c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "Token 删除成功"}))
}

// UpdateTokenStatus 更新 Token 状态
func (h *AdminTokenHandler) UpdateTokenStatus(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	var req struct {
		Status model.TokenStatus `json:"status" binding:"required,oneof=active disabled expired"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	token, err := h.tokenService.AdminUpdate(tokenID, &model.UpdateTokenRequest{Status: req.Status})
	if err != nil {
		if err.Error() == "Token 不存在" {
			c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
			return
		}
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(token))
}

// GetTokenUsage 获取 Token 使用统计
func (h *AdminTokenHandler) GetTokenUsage(c *gin.Context) {
	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	stats, err := h.tokenService.AdminGetUsage(tokenID)
	if err != nil {
		if err.Error() == "Token 不存在" {
			c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// RegisterRoutes 注册路由
func (h *AdminTokenHandler) RegisterRoutes(router *gin.RouterGroup) {
	tokens := router.Group("/tokens")
	{
		tokens.GET("", h.ListTokens)
		tokens.POST("", h.CreateToken)
		tokens.GET("/:id", h.GetToken)
		tokens.PUT("/:id", h.UpdateToken)
		tokens.DELETE("/:id", h.DeleteToken)
		tokens.PUT("/:id/status", h.UpdateTokenStatus)
		tokens.GET("/:id/usage", h.GetTokenUsage)
	}
}
