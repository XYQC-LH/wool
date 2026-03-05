package handler

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// AdminQuotaHandler 管理员配额管理处理器
type AdminQuotaHandler struct {
	quotaService service.QuotaService
}

func NewAdminQuotaHandler(quotaService service.QuotaService) *AdminQuotaHandler {
	return &AdminQuotaHandler{quotaService: quotaService}
}

type createQuotaPolicyRequest struct {
	TenantID              string  `json:"tenant_id" binding:"required"`
	Name                  string  `json:"name,omitempty"`
	Description           string  `json:"description,omitempty"`
	DailyRequestLimit     int64   `json:"daily_request_limit" binding:"min=0"`
	DailyCostLimit        *float64 `json:"daily_cost_limit,omitempty" binding:"omitempty,min=0"`
	AlertThresholdPercent int     `json:"alert_threshold_percent,omitempty" binding:"omitempty,min=1,max=100"`
	Status                string  `json:"status,omitempty"`
}

type updateQuotaPolicyRequest struct {
	TenantID              *string  `json:"tenant_id,omitempty"`
	Name                  *string  `json:"name,omitempty"`
	Description           *string  `json:"description,omitempty"`
	DailyRequestLimit     *int64   `json:"daily_request_limit,omitempty" binding:"omitempty,min=0"`
	DailyCostLimit        *float64 `json:"daily_cost_limit,omitempty" binding:"omitempty,min=0"`
	AlertThresholdPercent *int     `json:"alert_threshold_percent,omitempty" binding:"omitempty,min=1,max=100"`
	Status                *string  `json:"status,omitempty"`
}

func (h *AdminQuotaHandler) Create(c *gin.Context) {
	var req createQuotaPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	costLimit := decimal.Zero
	if req.DailyCostLimit != nil {
		costLimit = decimal.NewFromFloat(*req.DailyCostLimit)
	}

	status := model.QuotaPolicyStatus(strings.ToLower(strings.TrimSpace(req.Status)))
	input := &service.CreateQuotaPolicyInput{
		TenantID:              req.TenantID,
		Name:                  req.Name,
		Description:           req.Description,
		DailyRequestLimit:     req.DailyRequestLimit,
		DailyCostLimit:        costLimit,
		AlertThresholdPercent: req.AlertThresholdPercent,
		Status:                status,
	}

	policy, err := h.quotaService.Create(c.Request.Context(), input)
	if err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(policy))
}

func (h *AdminQuotaHandler) Update(c *gin.Context) {
	var req updateQuotaPolicyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	var costLimit *decimal.Decimal
	if req.DailyCostLimit != nil {
		parsed := decimal.NewFromFloat(*req.DailyCostLimit)
		costLimit = &parsed
	}

	var status *model.QuotaPolicyStatus
	if req.Status != nil {
		parsedStatus := model.QuotaPolicyStatus(strings.ToLower(strings.TrimSpace(*req.Status)))
		status = &parsedStatus
	}

	input := &service.UpdateQuotaPolicyInput{
		TenantID:              req.TenantID,
		Name:                  req.Name,
		Description:           req.Description,
		DailyRequestLimit:     req.DailyRequestLimit,
		DailyCostLimit:        costLimit,
		AlertThresholdPercent: req.AlertThresholdPercent,
		Status:                status,
	}

	policy, err := h.quotaService.Update(c.Request.Context(), c.Param("id"), input)
	if err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(policy))
}

func (h *AdminQuotaHandler) Delete(c *gin.Context) {
	if err := h.quotaService.Delete(c.Request.Context(), c.Param("id")); err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"deleted": true}))
}

func (h *AdminQuotaHandler) GetByID(c *gin.Context) {
	policy, err := h.quotaService.GetByID(c.Request.Context(), c.Param("id"))
	if err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(policy))
}

func (h *AdminQuotaHandler) List(c *gin.Context) {
	keyword := strings.TrimSpace(c.Query("keyword"))
	statusRaw := strings.ToLower(strings.TrimSpace(c.Query("status")))
	status := model.QuotaPolicyStatus(statusRaw)
	if statusRaw != "" && !status.IsValid() {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "status 仅支持 active/disabled"))
		return
	}

	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	policies, total, err := h.quotaService.List(c.Request.Context(), keyword, status, page, pageSize)
	if err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	pagination := model.NewPagination(page, pageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(policies, pagination))
}

func (h *AdminQuotaHandler) GetStats(c *gin.Context) {
	date, err := parseQuotaDate(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	stats, err := h.quotaService.GetStats(c.Request.Context(), date)
	if err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

func (h *AdminQuotaHandler) GetMonitoring(c *gin.Context) {
	date, err := parseQuotaDate(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}
	tenantID := strings.TrimSpace(c.Query("tenant_id"))

	items, err := h.quotaService.GetMonitoring(c.Request.Context(), date, tenantID)
	if err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(items))
}

func (h *AdminQuotaHandler) CheckAlerts(c *gin.Context) {
	date, err := parseQuotaDate(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	result, err := h.quotaService.CheckAlerts(c.Request.Context(), date)
	if err != nil {
		handleQuotaServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

func (h *AdminQuotaHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/quotas")
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/stats", h.GetStats)
		group.GET("/monitoring", h.GetMonitoring)
		group.POST("/alerts/check", h.CheckAlerts)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}

func parseQuotaDate(c *gin.Context) (time.Time, error) {
	raw := strings.TrimSpace(c.Query("date"))
	if raw == "" {
		return time.Now(), nil
	}
	parsed, err := parseDateInLocal(raw)
	if err != nil {
		return time.Time{}, errors.New("date 需为 YYYY-MM-DD 格式")
	}
	return parsed, nil
}

func handleQuotaServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrQuotaPolicyNotFound):
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
	case errors.Is(err, service.ErrQuotaPolicyConflict):
		c.JSON(http.StatusConflict, model.ErrorResponse(model.ErrCodeConflict, err.Error()))
	case errors.Is(err, service.ErrInvalidQuotaPolicyInput), errors.Is(err, service.ErrInvalidQuotaPolicyStatus):
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
	}
}

