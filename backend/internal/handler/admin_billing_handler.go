package handler

import (
	"errors"
	"net/http"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminBillingHandler 管理员计费管理处理器
type AdminBillingHandler struct {
	billingService service.BillingService
}

func NewAdminBillingHandler(billingService service.BillingService) *AdminBillingHandler {
	return &AdminBillingHandler{billingService: billingService}
}

// GetStatistics 获取计费统计
// GET /api/admin/billing/statistics
func (h *AdminBillingHandler) GetStatistics(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	stats, err := h.billingService.GetStatistics(c.Request.Context(), startDate, endDate)
	if err != nil {
		handleBillingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// GetUsage 获取使用量统计
// GET /api/admin/billing/usage
func (h *AdminBillingHandler) GetUsage(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	groupBy := strings.TrimSpace(c.Query("group_by"))
	limit := parsePositiveInt(c.Query("limit"), 20)

	data, err := h.billingService.GetUsage(c.Request.Context(), &service.BillingUsageQuery{
		StartDate: startDate,
		EndDate:   endDate,
		GroupBy:   groupBy,
		Limit:     limit,
	})
	if err != nil {
		handleBillingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(data))
}

// GetCostAnalysis 获取成本分析
// GET /api/admin/billing/cost-analysis
func (h *AdminBillingHandler) GetCostAnalysis(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	groupBy := strings.TrimSpace(c.Query("group_by"))
	limit := parsePositiveInt(c.Query("limit"), 20)

	data, err := h.billingService.GetCostAnalysis(c.Request.Context(), &service.BillingCostAnalysisQuery{
		StartDate: startDate,
		EndDate:   endDate,
		GroupBy:   groupBy,
		Limit:     limit,
	})
	if err != nil {
		handleBillingServiceError(c, err)
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(data))
}

func (h *AdminBillingHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/billing")
	{
		group.GET("/statistics", h.GetStatistics)
		group.GET("/usage", h.GetUsage)
		group.GET("/cost-analysis", h.GetCostAnalysis)
	}
}

func handleBillingServiceError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, service.ErrInvalidBillingGroupBy):
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
	default:
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
	}
}

