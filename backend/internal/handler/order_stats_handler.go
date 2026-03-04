package handler

import (
	"net/http"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// OrderStatsHandler 订单统计 API 处理器
type OrderStatsHandler struct {
	db *gorm.DB
}

// NewOrderStatsHandler 创建订单统计处理器
func NewOrderStatsHandler(db *gorm.DB) *OrderStatsHandler {
	return &OrderStatsHandler{db: db}
}

// OrderStats 订单统计数据
type OrderStats struct {
	TotalOrders   int64   `json:"total_orders"`
	PaidOrders    int64   `json:"paid_orders"`
	PendingOrders int64   `json:"pending_orders"`
	FailedOrders  int64   `json:"failed_orders"`
	TotalAmount   float64 `json:"total_amount"`
	PaidAmount    float64 `json:"paid_amount"`
}

// GetStats 获取订单统计
// GET /api/admin/orders/stats
func (h *OrderStatsHandler) GetStats(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	rangeQuery := h.db.Table("orders").Where("created_at >= ? AND created_at <= ?", startDate, endDate)

	var totalOrders int64
	if err := rangeQuery.Count(&totalOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询订单统计失败"))
		return
	}

	var paidOrders int64
	if err := h.db.Table("orders").
		Where("status = ? AND created_at >= ? AND created_at <= ?", model.OrderStatusPaid, startDate, endDate).
		Count(&paidOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询订单统计失败"))
		return
	}

	var pendingOrders int64
	if err := h.db.Table("orders").
		Where("status = ? AND created_at >= ? AND created_at <= ?", model.OrderStatusPending, startDate, endDate).
		Count(&pendingOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询订单统计失败"))
		return
	}

	var failedOrders int64
	if err := h.db.Table("orders").
		Where("status = ? AND created_at >= ? AND created_at <= ?", model.OrderStatusFailed, startDate, endDate).
		Count(&failedOrders).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询订单统计失败"))
		return
	}

	var totalAmountRow struct {
		Total float64 `gorm:"column:total"`
	}
	if err := h.db.Table("orders").
		Select("COALESCE(SUM(amount), 0)::float8 as total").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Scan(&totalAmountRow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询订单统计失败"))
		return
	}

	var paidAmountRow struct {
		Total float64 `gorm:"column:total"`
	}
	if err := h.db.Table("orders").
		Select("COALESCE(SUM(amount), 0)::float8 as total").
		Where("status = ? AND created_at >= ? AND created_at <= ?", model.OrderStatusPaid, startDate, endDate).
		Scan(&paidAmountRow).Error; err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询订单统计失败"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(&OrderStats{
		TotalOrders:   totalOrders,
		PaidOrders:    paidOrders,
		PendingOrders: pendingOrders,
		FailedOrders:  failedOrders,
		TotalAmount:   totalAmountRow.Total,
		PaidAmount:    paidAmountRow.Total,
	}))
}

// RegisterOrderStatsRoutes 注册订单统计路由
func RegisterOrderStatsRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewOrderStatsHandler(db)

	orders := r.Group("/orders")
	{
		orders.GET("/stats", h.GetStats)
	}
}
