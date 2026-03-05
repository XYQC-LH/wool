package handler

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
)

type providerEntityReader interface {
	GetByID(ctx context.Context, id uint) (*model.ModelProvider, error)
}

type providerMetricsReader interface {
	GetAggregatedMetrics(ctx context.Context, providerID uint, startTime, endTime time.Time) (*model.AggregatedMetrics, error)
	GetTimeSeries(ctx context.Context, providerID uint, granularity model.MetricGranularity, startTime, endTime time.Time) ([]*model.TimeSeriesMetric, error)
	GetCircuitEvents(ctx context.Context, providerID uint, startTime, endTime time.Time) ([]*model.CircuitEventRecord, error)
	GetTrafficDistribution(ctx context.Context, modelID string, startTime, endTime time.Time) ([]*model.ProviderTrafficDistribution, error)
}

type ProviderMonitoringHandler struct {
	providerRepo providerEntityReader
	metricsRepo  providerMetricsReader
}

func NewProviderMonitoringHandler(providerRepo providerEntityReader, metricsRepo providerMetricsReader) *ProviderMonitoringHandler {
	return &ProviderMonitoringHandler{
		providerRepo: providerRepo,
		metricsRepo:  metricsRepo,
	}
}

func (h *ProviderMonitoringHandler) GetProviderStats(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	if !h.providerExists(c, uint(providerID)) {
		return
	}

	startTime, endTime, err := parseMonitorTimeRange(c, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	stats, err := h.metricsRepo.GetAggregatedMetrics(c.Request.Context(), uint(providerID), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if stats == nil {
		stats = &model.AggregatedMetrics{}
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"provider_id": uint(providerID),
		"start_time":  startTime,
		"end_time":    endTime,
		"stats":       stats,
	}))
}

func (h *ProviderMonitoringHandler) GetProviderMetrics(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	if !h.providerExists(c, uint(providerID)) {
		return
	}

	startTime, endTime, err := parseMonitorTimeRange(c, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	granularity := model.MetricGranularity(strings.TrimSpace(c.DefaultQuery("granularity", string(model.MetricGranularityHour))))
	switch granularity {
	case model.MetricGranularityMinute, model.MetricGranularityHour, model.MetricGranularityDay:
	default:
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "granularity 仅支持 minute/hour/day"))
		return
	}

	series, err := h.metricsRepo.GetTimeSeries(c.Request.Context(), uint(providerID), granularity, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"provider_id":  uint(providerID),
		"granularity":  granularity,
		"start_time":   startTime,
		"end_time":     endTime,
		"time_series":  series,
	}))
}

func (h *ProviderMonitoringHandler) GetProviderCircuitEvents(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	if !h.providerExists(c, uint(providerID)) {
		return
	}

	startTime, endTime, err := parseMonitorTimeRange(c, 7*24*time.Hour)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	events, err := h.metricsRepo.GetCircuitEvents(c.Request.Context(), uint(providerID), startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	resp := make([]*model.CircuitEventRecordResponse, len(events))
	for i, event := range events {
		resp[i] = event.ToResponse()
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"provider_id": uint(providerID),
		"start_time":  startTime,
		"end_time":    endTime,
		"events":      resp,
	}))
}

func (h *ProviderMonitoringHandler) GetModelTrafficDistribution(c *gin.Context) {
	modelID := strings.TrimSpace(c.Param("model_id"))
	if modelID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "model_id 不能为空"))
		return
	}

	startTime, endTime, err := parseMonitorTimeRange(c, 24*time.Hour)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	distribution, err := h.metricsRepo.GetTrafficDistribution(c.Request.Context(), modelID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"model_id":     modelID,
		"start_time":   startTime,
		"end_time":     endTime,
		"distribution": distribution,
	}))
}

func (h *ProviderMonitoringHandler) providerExists(c *gin.Context, providerID uint) bool {
	entity, err := h.providerRepo.GetByID(c.Request.Context(), providerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return false
	}
	if entity == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "源头不存在"))
		return false
	}
	return true
}

func (h *ProviderMonitoringHandler) RegisterRoutes(router *gin.RouterGroup) {
	providers := router.Group("/providers")
	{
		providers.GET("/:id/stats", h.GetProviderStats)
		providers.GET("/:id/metrics", h.GetProviderMetrics)
		providers.GET("/:id/circuit-events", h.GetProviderCircuitEvents)
		providers.GET("/model/:model_id/traffic", h.GetModelTrafficDistribution)
	}
}

func parseMonitorTimeRange(c *gin.Context, defaultWindow time.Duration) (time.Time, time.Time, error) {
	now := time.Now()
	endTime := now
	startTime := now.Add(-defaultWindow)

	if raw := strings.TrimSpace(c.Query("end_time")); raw != "" {
		parsed, err := parseMonitorTime(raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		endTime = parsed
	}

	if raw := strings.TrimSpace(c.Query("start_time")); raw != "" {
		parsed, err := parseMonitorTime(raw)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		startTime = parsed
	} else {
		startTime = endTime.Add(-defaultWindow)
	}

	if !startTime.Before(endTime) {
		return time.Time{}, time.Time{}, fmt.Errorf("start_time 必须早于 end_time")
	}

	return startTime, endTime, nil
}

func parseMonitorTime(raw string) (time.Time, error) {
	if parsed, err := time.Parse(time.RFC3339, raw); err == nil {
		return parsed, nil
	}
	if parsed, err := time.Parse("2006-01-02", raw); err == nil {
		return parsed, nil
	}
	return time.Time{}, fmt.Errorf("时间格式必须为 RFC3339 或 YYYY-MM-DD")
}
