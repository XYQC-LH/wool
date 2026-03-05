package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

type ModelRouteHandler struct {
	modelRouteService service.ModelRouteService
}

func NewModelRouteHandler(modelRouteService service.ModelRouteService) *ModelRouteHandler {
	return &ModelRouteHandler{modelRouteService: modelRouteService}
}

type CreateModelRouteRequest struct {
	Operation   string `json:"operation,omitempty"`
	ModelID     string `json:"model_id" binding:"required"`
	ProviderID  uint   `json:"provider_id" binding:"required"`
	Priority    int    `json:"priority"`
	IsEnabled   *bool  `json:"is_enabled,omitempty"`
	Description string `json:"description,omitempty"`
}

type UpdateModelRouteRequest struct {
	Operation   *string `json:"operation,omitempty"`
	ModelID     *string `json:"model_id,omitempty"`
	ProviderID  *uint   `json:"provider_id,omitempty"`
	Priority    *int    `json:"priority,omitempty"`
	IsEnabled   *bool   `json:"is_enabled,omitempty"`
	Description *string `json:"description,omitempty"`
}

type BatchRouteEnabledRequest struct {
	IDs       []uint `json:"ids" binding:"required,min=1"`
	IsEnabled *bool  `json:"is_enabled" binding:"required"`
}

type BatchRoutePriorityItem struct {
	ID       uint `json:"id" binding:"required"`
	Priority int  `json:"priority" binding:"required"`
}

type BatchRoutePriorityRequest struct {
	Items []BatchRoutePriorityItem `json:"items" binding:"required,min=1"`
}

func (h *ModelRouteHandler) Create(c *gin.Context) {
	var req CreateModelRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	entity, err := h.modelRouteService.Create(c.Request.Context(), &service.CreateModelRouteRequest{
		Operation:   req.Operation,
		ModelID:     req.ModelID,
		ProviderID:  req.ProviderID,
		Priority:    req.Priority,
		IsEnabled:   req.IsEnabled,
		Description: req.Description,
	})
	if err != nil {
		handleModelRouteError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(entity))
}

func (h *ModelRouteHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	var req UpdateModelRouteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	entity, err := h.modelRouteService.Update(c.Request.Context(), uint(id), &service.UpdateModelRouteRequest{
		Operation:   req.Operation,
		ModelID:     req.ModelID,
		ProviderID:  req.ProviderID,
		Priority:    req.Priority,
		IsEnabled:   req.IsEnabled,
		Description: req.Description,
	})
	if err != nil {
		handleModelRouteError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(entity))
}

func (h *ModelRouteHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.modelRouteService.Delete(c.Request.Context(), uint(id)); err != nil {
		handleModelRouteError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"deleted": true}))
}

func (h *ModelRouteHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	entity, err := h.modelRouteService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		handleModelRouteError(c, err)
		return
	}
	if entity == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "路由映射不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(entity))
}

func (h *ModelRouteHandler) List(c *gin.Context) {
	operation := strings.TrimSpace(c.Query("operation"))
	modelID := strings.TrimSpace(c.Query("model_id"))

	var providerID uint
	if raw := strings.TrimSpace(c.Query("provider_id")); raw != "" {
		parsed, err := strconv.ParseUint(raw, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 provider_id"))
			return
		}
		providerID = uint(parsed)
	}

	var isEnabled *bool
	if raw := strings.TrimSpace(c.Query("is_enabled")); raw != "" {
		parsed, err := strconv.ParseBool(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 is_enabled"))
			return
		}
		isEnabled = &parsed
	}

	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	list, total, err := h.modelRouteService.List(c.Request.Context(), &service.ModelRouteQueryParams{
		Operation:  operation,
		ModelID:    modelID,
		ProviderID: providerID,
		IsEnabled:  isEnabled,
		Page:       page,
		PageSize:   pageSize,
	})
	if err != nil {
		handleModelRouteError(c, err)
		return
	}

	pagination := model.NewPagination(page, pageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(list, pagination))
}

func (h *ModelRouteHandler) BatchUpdateEnabled(c *gin.Context) {
	var req BatchRouteEnabledRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}
	if req.IsEnabled == nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "is_enabled 不能为空"))
		return
	}

	if err := h.modelRouteService.BatchUpdateEnabled(c.Request.Context(), req.IDs, *req.IsEnabled); err != nil {
		handleModelRouteError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"updated": len(req.IDs)}))
}

func (h *ModelRouteHandler) BatchUpdatePriority(c *gin.Context) {
	var req BatchRoutePriorityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	items := make([]service.ModelRoutePriorityItem, 0, len(req.Items))
	for _, item := range req.Items {
		items = append(items, service.ModelRoutePriorityItem{
			ID:       item.ID,
			Priority: item.Priority,
		})
	}
	if err := h.modelRouteService.BatchUpdatePriority(c.Request.Context(), items); err != nil {
		handleModelRouteError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"updated": len(items)}))
}

func (h *ModelRouteHandler) GetStats(c *gin.Context) {
	stats, err := h.modelRouteService.GetStats(
		c.Request.Context(),
		strings.TrimSpace(c.Query("operation")),
		strings.TrimSpace(c.Query("model_id")),
	)
	if err != nil {
		handleModelRouteError(c, err)
		return
	}
	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

func (h *ModelRouteHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/model-routes")
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/stats", h.GetStats)
		group.POST("/batch/enabled", h.BatchUpdateEnabled)
		group.POST("/batch/priority", h.BatchUpdatePriority)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}

func handleModelRouteError(c *gin.Context, err error) {
	status := http.StatusInternalServerError
	errorCode := model.ErrCodeInternalError
	message := err.Error()

	switch {
	case strings.Contains(message, "不存在"):
		status = http.StatusNotFound
		errorCode = model.ErrCodeNotFound
	case strings.Contains(message, "已存在"):
		status = http.StatusConflict
		errorCode = model.ErrCodeConflict
	case strings.Contains(message, "不能为空"),
		strings.Contains(message, "不匹配"),
		strings.Contains(message, "必须"):
		status = http.StatusBadRequest
		errorCode = model.ErrCodeInvalidRequest
	}

	c.JSON(status, model.ErrorResponse(errorCode, message))
}
