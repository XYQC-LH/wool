package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminMetricsHandler 管理员监控管理处理器
type AdminMetricsHandler struct {
	metricsService service.MetricsService
	alertService   service.AlertService
}

func NewAdminMetricsHandler(metricsService service.MetricsService, alertService service.AlertService) *AdminMetricsHandler {
	return &AdminMetricsHandler{
		metricsService: metricsService,
		alertService:   alertService,
	}
}

// Query 查询 Metrics 明细
func (h *AdminMetricsHandler) Query(c *gin.Context) {
	startTime, endTime, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	granularity := model.MetricGranularity(strings.TrimSpace(c.DefaultQuery("granularity", string(model.MetricGranularityMinute))))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	var providerID *uint
	if raw := strings.TrimSpace(c.Query("provider_id")); raw != "" {
		parsed, parseErr := strconv.ParseUint(raw, 10, 32)
		if parseErr != nil || parsed == 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "provider_id 无效"))
			return
		}
		normalized := uint(parsed)
		providerID = &normalized
	}

	result, pagination, err := h.metricsService.Query(c.Request.Context(), &service.MetricsQuery{
		ProviderID:  providerID,
		ModelID:     strings.TrimSpace(c.Query("model_id")),
		Granularity: granularity,
		StartTime:   startTime,
		EndTime:     endTime,
		Page:        page,
		PageSize:    pageSize,
	})
	if err != nil {
		if errors.Is(err, service.ErrInvalidMetricsGranularity) || strings.Contains(err.Error(), "start_time") {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(result, pagination))
}

// Realtime 获取实时监控指标
func (h *AdminMetricsHandler) Realtime(c *gin.Context) {
	windowSeconds := parsePositiveInt(c.Query("window_seconds"), 300)
	if windowSeconds > 86400 {
		windowSeconds = 86400
	}

	realtime, err := h.metricsService.GetRealtime(c.Request.Context(), time.Duration(windowSeconds)*time.Second)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	cpuPercent, cpuErr := getSystemCPUPercent()
	if cpuErr != nil {
		cpuPercent = 0
	}
	memoryPercent, memErr := getSystemMemoryPercent()
	if memErr != nil {
		memoryPercent = 0
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"metrics": realtime,
		"system": gin.H{
			"cpu_percent":        cpuPercent,
			"memory_percent":     memoryPercent,
			"redis_connections":  getRedisConnectedClients(),
			"db_connections":     getDBOpenConnections(),
		},
	}))
}

// ListAlerts 获取监控告警列表
func (h *AdminMetricsHandler) ListAlerts(c *gin.Context) {
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	filters := make(map[string]interface{})
	if alertType := strings.TrimSpace(c.Query("type")); alertType != "" {
		filters["type"] = alertType
	}
	if severity := strings.TrimSpace(c.Query("severity")); severity != "" {
		filters["severity"] = severity
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filters["status"] = status
	}

	alerts, pagination, err := h.alertService.List(page, pageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(alerts, pagination))
}

// ResolveAlert 处理告警
func (h *AdminMetricsHandler) ResolveAlert(c *gin.Context) {
	alertID, err := uuid.Parse(strings.TrimSpace(c.Param("id")))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的告警ID"))
		return
	}

	userID, exists := middleware.GetCurrentUserID(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	if err := h.alertService.Resolve(alertID, userID); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"resolved": true}))
}

// AlertStats 获取告警统计
func (h *AdminMetricsHandler) AlertStats(c *gin.Context) {
	stats, err := h.alertService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

func (h *AdminMetricsHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/metrics")
	{
		group.GET("/query", h.Query)
		group.GET("/realtime", h.Realtime)
		group.GET("/alerts", h.ListAlerts)
		group.PUT("/alerts/:id/resolve", h.ResolveAlert)
		group.GET("/alerts/stats", h.AlertStats)
	}
}
