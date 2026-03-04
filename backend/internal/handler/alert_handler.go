package handler

import (
	"net/http"
	"strconv"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AlertHandler 告警处理器
type AlertHandler struct {
	alertService service.AlertService
}

// NewAlertHandler 创建告警处理器
func NewAlertHandler(alertService service.AlertService) *AlertHandler {
	return &AlertHandler{
		alertService: alertService,
	}
}

// ListAlerts 获取告警列表
// @Summary 获取告警列表
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param type query string false "告警类型"
// @Param severity query string false "严重级别"
// @Param status query string false "状态"
// @Success 200 {object} model.Response{data=[]model.AlertResponse}
// @Router /api/admin/alerts [get]
func (h *AlertHandler) ListAlerts(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := make(map[string]interface{})
	if alertType := c.Query("type"); alertType != "" {
		filters["type"] = alertType
	}
	if severity := c.Query("severity"); severity != "" {
		filters["severity"] = severity
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}

	alerts, pagination, err := h.alertService.List(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(alerts, pagination))
}

// GetAlert 获取告警详情
// @Summary 获取告警详情
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path string true "告警ID"
// @Success 200 {object} model.Response{data=model.AlertResponse}
// @Router /api/admin/alerts/{id} [get]
func (h *AlertHandler) GetAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的告警ID"))
		return
	}

	alert, err := h.alertService.GetByID(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(alert))
}

// ResolveAlert 解决告警
// @Summary 解决告警
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path string true "告警ID"
// @Success 200 {object} model.Response
// @Router /api/admin/alerts/{id}/resolve [put]
func (h *AlertHandler) ResolveAlert(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的告警ID"))
		return
	}

	// 获取当前用户ID
	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	if err := h.alertService.Resolve(id, userID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(nil))
}

// GetAlertStats 获取告警统计
// @Summary 获取告警统计
// @Tags 告警管理
// @Accept json
// @Produce json
// @Success 200 {object} model.Response{data=model.AlertStats}
// @Router /api/admin/alerts/stats [get]
func (h *AlertHandler) GetAlertStats(c *gin.Context) {
	stats, err := h.alertService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// GetActiveAlerts 获取活跃告警
// @Summary 获取活跃告警
// @Tags 告警管理
// @Accept json
// @Produce json
// @Success 200 {object} model.Response{data=[]model.AlertResponse}
// @Router /api/admin/alerts/active [get]
func (h *AlertHandler) GetActiveAlerts(c *gin.Context) {
	alerts, err := h.alertService.GetActiveAlerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(alerts))
}
