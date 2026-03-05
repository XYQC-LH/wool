package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

type ProviderCapabilityHandler struct {
	service service.ProviderCapabilityService
}

func NewProviderCapabilityHandler(svc service.ProviderCapabilityService) *ProviderCapabilityHandler {
	return &ProviderCapabilityHandler{service: svc}
}

type CreateProviderCapabilityRequest struct {
	ProviderID  uint       `json:"provider_id" binding:"required"`
	Operation   string     `json:"operation" binding:"required"`
	Constraints model.JSON `json:"constraints,omitempty"`
	IsEnabled   *bool      `json:"is_enabled,omitempty"`
}

type UpdateProviderCapabilityRequest struct {
	Operation   *string     `json:"operation,omitempty"`
	Constraints *model.JSON `json:"constraints,omitempty"`
	IsEnabled   *bool       `json:"is_enabled,omitempty"`
}

type BatchProviderCapabilityEnabledRequest struct {
	IDs       []uint `json:"ids" binding:"required,min=1"`
	IsEnabled bool   `json:"is_enabled"`
}

type ValidateProviderCapabilityRequest struct {
	ProviderID  uint                   `json:"provider_id" binding:"required"`
	Operation   string                 `json:"operation" binding:"required"`
	Constraints map[string]interface{} `json:"constraints" binding:"required"`
}

func (h *ProviderCapabilityHandler) Create(c *gin.Context) {
	var req CreateProviderCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	entity, err := h.service.Create(c.Request.Context(), &service.CreateProviderCapabilityRequest{
		ProviderID:  req.ProviderID,
		Operation:   req.Operation,
		Constraints: req.Constraints,
		IsEnabled:   req.IsEnabled,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		errCode := model.ErrCodeInternalError
		if strings.Contains(err.Error(), "已存在") {
			statusCode = http.StatusConflict
			errCode = model.ErrCodeConflict
		}
		if strings.Contains(err.Error(), "源头不存在") {
			statusCode = http.StatusNotFound
			errCode = model.ErrCodeNotFound
		}
		if strings.Contains(err.Error(), "不能为空") {
			statusCode = http.StatusBadRequest
			errCode = model.ErrCodeInvalidRequest
		}
		c.JSON(statusCode, model.ErrorResponse(errCode, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(entity))
}

func (h *ProviderCapabilityHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	var req UpdateProviderCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	entity, err := h.service.Update(c.Request.Context(), uint(id), &service.UpdateProviderCapabilityRequest{
		Operation:   req.Operation,
		Constraints: req.Constraints,
		IsEnabled:   req.IsEnabled,
	})
	if err != nil {
		statusCode := http.StatusInternalServerError
		errCode := model.ErrCodeInternalError
		if strings.Contains(err.Error(), "不存在") {
			statusCode = http.StatusNotFound
			errCode = model.ErrCodeNotFound
		}
		if strings.Contains(err.Error(), "已存在") {
			statusCode = http.StatusConflict
			errCode = model.ErrCodeConflict
		}
		if strings.Contains(err.Error(), "不能为空") {
			statusCode = http.StatusBadRequest
			errCode = model.ErrCodeInvalidRequest
		}
		c.JSON(statusCode, model.ErrorResponse(errCode, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(entity))
}

func (h *ProviderCapabilityHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.service.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"deleted": true}))
}

func (h *ProviderCapabilityHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	entity, err := h.service.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if entity == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "记录不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(entity))
}

func (h *ProviderCapabilityHandler) List(c *gin.Context) {
	providerID := uint(parsePositiveInt(c.Query("provider_id"), 0))
	operation := strings.TrimSpace(c.Query("operation"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	var isEnabled *bool
	if raw := strings.TrimSpace(c.Query("is_enabled")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "is_enabled 必须是布尔值"))
			return
		}
		isEnabled = &parsed
	}

	list, total, err := h.service.List(c.Request.Context(), &service.ProviderCapabilityQueryParams{
		ProviderID: providerID,
		Operation:  operation,
		IsEnabled:  isEnabled,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	pagination := model.NewPagination(page, pageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(list, pagination))
}

func (h *ProviderCapabilityHandler) ListByProvider(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	list, err := h.service.ListByProvider(c.Request.Context(), uint(providerID))
	if err != nil {
		statusCode := http.StatusInternalServerError
		errCode := model.ErrCodeInternalError
		if strings.Contains(err.Error(), "不存在") {
			statusCode = http.StatusNotFound
			errCode = model.ErrCodeNotFound
		}
		if strings.Contains(err.Error(), "不能为空") {
			statusCode = http.StatusBadRequest
			errCode = model.ErrCodeInvalidRequest
		}
		c.JSON(statusCode, model.ErrorResponse(errCode, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(list))
}

func (h *ProviderCapabilityHandler) BatchUpdateEnabled(c *gin.Context) {
	var req BatchProviderCapabilityEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	if err := h.service.BatchUpdateEnabled(c.Request.Context(), req.IDs, req.IsEnabled); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"updated": true}))
}

func (h *ProviderCapabilityHandler) ValidateConstraints(c *gin.Context) {
	var req ValidateProviderCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	result, err := h.service.ValidateConstraints(c.Request.Context(), &service.ValidateProviderConstraintsRequest{
		ProviderID:  req.ProviderID,
		Operation:   req.Operation,
		Constraints: req.Constraints,
	})
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

func (h *ProviderCapabilityHandler) GetSummary(c *gin.Context) {
	providerID := uint(parsePositiveInt(c.Query("provider_id"), 0))
	summary, err := h.service.GetSummary(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(summary))
}

func (h *ProviderCapabilityHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/provider-capabilities")
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/summary", h.GetSummary)
		group.POST("/batch/enabled", h.BatchUpdateEnabled)
		group.POST("/validate", h.ValidateConstraints)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}

	providers := router.Group("/providers")
	{
		providers.GET("/:id/capabilities", h.ListByProvider)
	}
}
