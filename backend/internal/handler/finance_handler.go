package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// FinanceHandler 财务报表 API 处理器
type FinanceHandler struct {
	db *gorm.DB
}

// NewFinanceHandler 创建财务报表处理器
func NewFinanceHandler(db *gorm.DB) *FinanceHandler {
	return &FinanceHandler{db: db}
}

type FinanceOverview struct {
	TotalRevenue float64 `json:"total_revenue"`
	TotalCost    float64 `json:"total_cost"`
	TotalProfit  float64 `json:"total_profit"`
	ProfitMargin float64 `json:"profit_margin"`

	TodayRevenue float64 `json:"today_revenue"`
	TodayCost    float64 `json:"today_cost"`
	TodayProfit  float64 `json:"today_profit"`

	MonthRevenue float64 `json:"month_revenue"`
	MonthCost    float64 `json:"month_cost"`
	MonthProfit  float64 `json:"month_profit"`
}

type RevenueData struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Orders  int64   `json:"orders"`
}

type CostData struct {
	Date     string  `json:"date"`
	Cost     float64 `json:"cost"`
	Requests int64   `json:"requests"`
}

type ProfitData struct {
	Date    string  `json:"date"`
	Revenue float64 `json:"revenue"`
	Cost    float64 `json:"cost"`
	Profit  float64 `json:"profit"`
}

type TopUser struct {
	UserID            string  `json:"user_id"`
	Username          string  `json:"username"`
	Email             string  `json:"email"`
	TotalSpent        float64 `json:"total_spent"`
	TotalRequests     int64   `json:"total_requests"`
	AvgCostPerRequest float64 `json:"avg_cost_per_request"`
}

func parseDateInLocal(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.ParseInLocation("2006-01-02", dateStr, time.Local)
}

func parseDateRangeFromQuery(c *gin.Context) (time.Time, time.Time, error) {
	startStr := c.Query("start_date")
	endStr := c.Query("end_date")

	startDateRaw, err := parseDateInLocal(startStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	endDateRaw, err := parseDateInLocal(endStr)
	if err != nil {
		return time.Time{}, time.Time{}, err
	}

	now := time.Now()

	var startDate, endDate time.Time
	switch {
	case startDateRaw.IsZero() && endDateRaw.IsZero():
		endDate = now
		startDate = dayStart(endDate.AddDate(0, 0, -30))
	case startDateRaw.IsZero() && !endDateRaw.IsZero():
		startDate = dayStart(endDateRaw.AddDate(0, 0, -30))
		endDate = endDateRaw.Add(24*time.Hour - time.Second)
	case !startDateRaw.IsZero() && endDateRaw.IsZero():
		startDate = dayStart(startDateRaw)
		endDate = now
	default:
		startDate = dayStart(startDateRaw)
		endDate = endDateRaw.Add(24*time.Hour - time.Second)
	}

	if startDate.After(endDate) {
		return time.Time{}, time.Time{}, errors.New("invalid date range")
	}

	return startDate, endDate, nil
}

func dayStart(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}

func buildDayKeys(startDate, endDate time.Time) []string {
	startDay := dayStart(startDate)
	endDay := dayStart(endDate)
	keys := make([]string, 0, int(endDay.Sub(startDay).Hours()/24)+1)
	for d := startDay; !d.After(endDay); d = d.AddDate(0, 0, 1) {
		keys = append(keys, d.Format("2006-01-02"))
	}
	return keys
}

func (h *FinanceHandler) sumOrderRevenue(startDate, endDate time.Time) (float64, error) {
	var row struct {
		Total float64 `gorm:"column:total"`
	}

	err := h.db.Table("orders").
		Select("COALESCE(SUM(amount), 0)::float8 as total").
		Where("status = ? AND paid_at IS NOT NULL AND paid_at >= ? AND paid_at <= ?", model.OrderStatusPaid, startDate, endDate).
		Scan(&row).Error
	return row.Total, err
}

func (h *FinanceHandler) sumUpstreamCost(startDate, endDate time.Time) (float64, error) {
	var row struct {
		Total float64 `gorm:"column:total"`
	}

	err := h.db.Table("logs").
		Select("COALESCE(SUM(upstream_cost), 0)::float8 as total").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Scan(&row).Error
	return row.Total, err
}

// GetOverview 获取财务概览
// GET /api/admin/finance/overview
func (h *FinanceHandler) GetOverview(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	totalRevenue, err := h.sumOrderRevenue(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询收入失败"))
		return
	}

	totalCost, err := h.sumUpstreamCost(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询成本失败"))
		return
	}

	totalProfit := totalRevenue - totalCost
	profitMargin := float64(0)
	if totalRevenue > 0 {
		profitMargin = totalProfit / totalRevenue * 100
	}

	now := time.Now()
	todayStart := dayStart(now)
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())

	todayRevenue, _ := h.sumOrderRevenue(todayStart, now)
	todayCost, _ := h.sumUpstreamCost(todayStart, now)
	monthRevenue, _ := h.sumOrderRevenue(monthStart, now)
	monthCost, _ := h.sumUpstreamCost(monthStart, now)

	overview := &FinanceOverview{
		TotalRevenue: totalRevenue,
		TotalCost:    totalCost,
		TotalProfit:  totalProfit,
		ProfitMargin: profitMargin,

		TodayRevenue: todayRevenue,
		TodayCost:    todayCost,
		TodayProfit:  todayRevenue - todayCost,

		MonthRevenue: monthRevenue,
		MonthCost:    monthCost,
		MonthProfit:  monthRevenue - monthCost,
	}

	c.JSON(http.StatusOK, model.SuccessResponse(overview))
}

// GetRevenue 获取收入趋势
// GET /api/admin/finance/revenue
func (h *FinanceHandler) GetRevenue(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	groupBy := c.DefaultQuery("group_by", "day")
	var truncExpr, dateFormat string
	switch groupBy {
	case "day":
		truncExpr = "DATE_TRUNC('day', paid_at)"
		dateFormat = "YYYY-MM-DD"
	case "week":
		truncExpr = "DATE_TRUNC('week', paid_at)"
		dateFormat = "YYYY-MM-DD"
	case "month":
		truncExpr = "DATE_TRUNC('month', paid_at)"
		dateFormat = "YYYY-MM"
	default:
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的聚合粒度"))
		return
	}

	type revenueRow struct {
		Date    string  `gorm:"column:date"`
		Revenue float64 `gorm:"column:revenue"`
		Orders  int64   `gorm:"column:orders"`
	}

	var rows []revenueRow
	err = h.db.Table("orders").
		Select("TO_CHAR("+truncExpr+", ?) as date, COALESCE(SUM(amount), 0)::float8 as revenue, COUNT(*) as orders", dateFormat).
		Where("status = ? AND paid_at IS NOT NULL AND paid_at >= ? AND paid_at <= ?", model.OrderStatusPaid, startDate, endDate).
		Group(truncExpr).
		Order(truncExpr + " ASC").
		Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询收入趋势失败"))
		return
	}

	if groupBy != "day" {
		data := make([]*RevenueData, 0, len(rows))
		for _, r := range rows {
			data = append(data, &RevenueData{Date: r.Date, Revenue: r.Revenue, Orders: r.Orders})
		}
		c.JSON(http.StatusOK, model.SuccessResponse(data))
		return
	}

	m := make(map[string]*RevenueData, len(rows))
	for _, r := range rows {
		m[r.Date] = &RevenueData{Date: r.Date, Revenue: r.Revenue, Orders: r.Orders}
	}

	keys := buildDayKeys(startDate, endDate)
	data := make([]*RevenueData, 0, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			data = append(data, v)
			continue
		}
		data = append(data, &RevenueData{Date: k, Revenue: 0, Orders: 0})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(data))
}

// GetCost 获取成本趋势
// GET /api/admin/finance/cost
func (h *FinanceHandler) GetCost(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	truncExpr := "DATE_TRUNC('day', created_at)"

	type costRow struct {
		Date     string  `gorm:"column:date"`
		Cost     float64 `gorm:"column:cost"`
		Requests int64   `gorm:"column:requests"`
	}

	var rows []costRow
	err = h.db.Table("logs").
		Select("TO_CHAR("+truncExpr+", 'YYYY-MM-DD') as date, COALESCE(SUM(upstream_cost), 0)::float8 as cost, COUNT(*) as requests").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group(truncExpr).
		Order(truncExpr + " ASC").
		Scan(&rows).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询成本趋势失败"))
		return
	}

	m := make(map[string]*CostData, len(rows))
	for _, r := range rows {
		m[r.Date] = &CostData{Date: r.Date, Cost: r.Cost, Requests: r.Requests}
	}

	keys := buildDayKeys(startDate, endDate)
	data := make([]*CostData, 0, len(keys))
	for _, k := range keys {
		if v, ok := m[k]; ok {
			data = append(data, v)
			continue
		}
		data = append(data, &CostData{Date: k, Cost: 0, Requests: 0})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(data))
}

// GetProfit 获取利润趋势
// GET /api/admin/finance/profit
func (h *FinanceHandler) GetProfit(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	revenue, err := h.getRevenueSeriesDay(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询收入趋势失败"))
		return
	}

	cost, err := h.getCostSeriesDay(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询成本趋势失败"))
		return
	}

	keys := buildDayKeys(startDate, endDate)
	data := make([]*ProfitData, 0, len(keys))
	for _, k := range keys {
		r := revenue[k]
		costVal := cost[k]
		data = append(data, &ProfitData{
			Date:    k,
			Revenue: r,
			Cost:    costVal,
			Profit:  r - costVal,
		})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(data))
}

func (h *FinanceHandler) getRevenueSeriesDay(startDate, endDate time.Time) (map[string]float64, error) {
	truncExpr := "DATE_TRUNC('day', paid_at)"

	type row struct {
		Date    string  `gorm:"column:date"`
		Revenue float64 `gorm:"column:revenue"`
	}

	var rows []row
	err := h.db.Table("orders").
		Select("TO_CHAR("+truncExpr+", 'YYYY-MM-DD') as date, COALESCE(SUM(amount), 0)::float8 as revenue").
		Where("status = ? AND paid_at IS NOT NULL AND paid_at >= ? AND paid_at <= ?", model.OrderStatusPaid, startDate, endDate).
		Group(truncExpr).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	m := make(map[string]float64, len(rows))
	for _, r := range rows {
		m[r.Date] = r.Revenue
	}
	return m, nil
}

func (h *FinanceHandler) getCostSeriesDay(startDate, endDate time.Time) (map[string]float64, error) {
	truncExpr := "DATE_TRUNC('day', created_at)"

	type row struct {
		Date string  `gorm:"column:date"`
		Cost float64 `gorm:"column:cost"`
	}

	var rows []row
	err := h.db.Table("logs").
		Select("TO_CHAR("+truncExpr+", 'YYYY-MM-DD') as date, COALESCE(SUM(upstream_cost), 0)::float8 as cost").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group(truncExpr).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	m := make(map[string]float64, len(rows))
	for _, r := range rows {
		m[r.Date] = r.Cost
	}
	return m, nil
}

// GetTopUsers 获取用户消费排行
// GET /api/admin/finance/top-users
func (h *FinanceHandler) GetTopUsers(c *gin.Context) {
	startDate, endDate, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	type topUserRow struct {
		UserID     uuid.UUID `gorm:"column:user_id"`
		Username   string    `gorm:"column:username"`
		Email      string    `gorm:"column:email"`
		TotalSpent float64   `gorm:"column:total_spent"`
	}

	var topUsers []topUserRow
	err = h.db.Table("orders").
		Select("orders.user_id as user_id, users.username as username, users.email as email, COALESCE(SUM(orders.amount), 0)::float8 as total_spent").
		Joins("JOIN users ON users.id = orders.user_id").
		Where("orders.status = ? AND orders.paid_at IS NOT NULL AND orders.paid_at >= ? AND orders.paid_at <= ?", model.OrderStatusPaid, startDate, endDate).
		Group("orders.user_id, users.username, users.email").
		Order("total_spent DESC").
		Limit(limit).
		Scan(&topUsers).Error
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "查询用户消费排行失败"))
		return
	}

	userIDs := make([]uuid.UUID, 0, len(topUsers))
	for _, u := range topUsers {
		userIDs = append(userIDs, u.UserID)
	}

	type reqRow struct {
		UserID        uuid.UUID `gorm:"column:user_id"`
		TotalRequests int64     `gorm:"column:total_requests"`
	}

	requestCountByUserID := make(map[uuid.UUID]int64, len(userIDs))
	if len(userIDs) > 0 {
		var reqRows []reqRow
		_ = h.db.Table("logs").
			Select("user_id, COUNT(*) as total_requests").
			Where("user_id IN ? AND created_at >= ? AND created_at <= ?", userIDs, startDate, endDate).
			Group("user_id").
			Scan(&reqRows).Error

		for _, r := range reqRows {
			requestCountByUserID[r.UserID] = r.TotalRequests
		}
	}

	data := make([]*TopUser, 0, len(topUsers))
	for _, u := range topUsers {
		totalRequests := requestCountByUserID[u.UserID]
		avg := float64(0)
		if totalRequests > 0 {
			avg = u.TotalSpent / float64(totalRequests)
		}
		data = append(data, &TopUser{
			UserID:            u.UserID.String(),
			Username:          u.Username,
			Email:             u.Email,
			TotalSpent:        u.TotalSpent,
			TotalRequests:     totalRequests,
			AvgCostPerRequest: avg,
		})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(data))
}

// RegisterFinanceRoutes 注册财务报表路由
func RegisterFinanceRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewFinanceHandler(db)

	finance := r.Group("/finance")
	{
		finance.GET("/overview", h.GetOverview)
		finance.GET("/revenue", h.GetRevenue)
		finance.GET("/cost", h.GetCost)
		finance.GET("/profit", h.GetProfit)
		finance.GET("/top-users", h.GetTopUsers)
	}
}
