package handler

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// DispatchHandler 调度监控 API 处理器
type DispatchHandler struct {
	db *gorm.DB
}

// NewDispatchHandler 创建调度监控处理器
func NewDispatchHandler(db *gorm.DB) *DispatchHandler {
	return &DispatchHandler{db: db}
}

// DispatchStats 调度统计数据
type DispatchStats struct {
	TotalRequests        int64              `json:"total_requests"`
	SuccessRate          float64            `json:"success_rate"`
	AvgLatencyMs         int                `json:"avg_latency_ms"`
	ActiveProviders      int                `json:"active_providers"`
	CircuitOpenProviders int                `json:"circuit_open_providers"`
	TotalProviders       int                `json:"total_providers"`
	Providers            []ProviderStatItem `json:"providers"`
}

// ProviderStatItem 单个源头的统计数据
type ProviderStatItem struct {
	ID                uint    `json:"id"`
	Name              string  `json:"name"`
	ChannelName       string  `json:"channel_name"`
	CostPer1K         float64 `json:"cost_per_1k"`
	Status            string  `json:"status"`
	CircuitState      string  `json:"circuit_state"`
	SuccessRate       float64 `json:"success_rate"`
	AvgLatencyMs      int     `json:"avg_latency_ms"`
	RequestCount      int64   `json:"request_count"`
	TrafficPercentage float64 `json:"traffic_percentage"`
	HealthScore       float64 `json:"health_score"`
}

// MetricPoint 指标数据点
type MetricPoint struct {
	Time         string                        `json:"time"`
	RequestCount int64                         `json:"request_count"`
	SuccessRate  float64                       `json:"success_rate"`
	AvgLatencyMs int                           `json:"avg_latency_ms"`
	Providers    map[string]ProviderMetricData `json:"providers,omitempty"`
}

// MetricRow 指标查询行（provider_metrics 联表查询结果）
type MetricRow struct {
	MetricTime   time.Time
	RequestCount int64
	SuccessCount int64
	AvgLatencyMs float64
	ProviderID   uint
	ModelName    string
}

// ProviderMetricData 源头指标数据
type ProviderMetricData struct {
	Requests    int64   `json:"requests"`
	SuccessRate float64 `json:"success_rate"`
}

// CircuitEvent 熔断事件
type CircuitEvent struct {
	ID           uint      `json:"id"`
	ProviderID   uint      `json:"provider_id"`
	ProviderName string    `json:"provider_name"`
	ChannelName  string    `json:"channel_name"`
	EventType    string    `json:"event_type"` // open, close, half_open
	Reason       string    `json:"reason"`
	Duration     int       `json:"duration,omitempty"` // 熔断时长（秒）
	CreatedAt    time.Time `json:"created_at"`
}

// GetStats 获取调度统计数据
// GET /api/admin/dispatch/stats
func (h *DispatchHandler) GetStats(c *gin.Context) {
	modelID := c.Query("model_id")
	operation := c.Query("operation")
	period := c.DefaultQuery("period", "24h")

	// 计算时间范围
	var startTime time.Time
	switch period {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
	case "6h":
		startTime = time.Now().Add(-6 * time.Hour)
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
	case "30d":
		startTime = time.Now().Add(-30 * 24 * time.Hour)
	default:
		startTime = time.Now().Add(-24 * time.Hour)
	}

	endTime := time.Now()

	// 构建查询：以日志为准，按时间范围聚合源头统计
	query := h.db.Table("model_providers").
		Select(`
			model_providers.id,
			models.name as model_name,
			channels.name as channel_name,
			model_providers.actual_cost_per_1k_input,
			model_providers.status,
			model_providers.circuit_state,
			model_providers.health_score,
			COALESCE(COUNT(logs.id), 0) as request_count,
			COALESCE(SUM(CASE WHEN logs.status = 'success' THEN 1 ELSE 0 END), 0) as success_count,
			COALESCE(AVG(logs.duration), 0) as avg_latency_ms
		`).
		Joins("LEFT JOIN models ON models.id = model_providers.model_id").
		Joins("LEFT JOIN channels ON channels.id = model_providers.channel_id").
		Joins(
			"LEFT JOIN logs ON logs.model = model_providers.model_id AND logs.channel_id = model_providers.channel_id AND logs.created_at >= ? AND logs.created_at <= ?",
			startTime,
			endTime,
		)

	if modelID != "" {
		query = query.Where("model_providers.model_id = ?", modelID)
	}
	if operation != "" {
		query = query.Where("model_providers.operation = ?", operation)
	}

	type ProviderRow struct {
		ID                   uint
		ModelName            string
		ChannelName          string
		ActualCostPer1KInput float64
		Status               string
		CircuitState         string
		HealthScore          float64
		RequestCount         int64
		SuccessCount         int64
		AvgLatencyMs         float64
	}

	var providers []ProviderRow
	if err := query.Group(`
			model_providers.id,
			models.name,
			channels.name,
			model_providers.actual_cost_per_1k_input,
			model_providers.status,
			model_providers.circuit_state,
			model_providers.health_score
		`).Find(&providers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询源头数据失败: " + err.Error(),
		})
		return
	}

	// 计算统计数据
	var totalRequests int64
	var totalSuccess int64
	var totalLatency int64
	var activeCount int
	var circuitOpenCount int
	var providerItems []ProviderStatItem

	for _, p := range providers {
		totalRequests += p.RequestCount
		totalSuccess += p.SuccessCount
		if p.RequestCount > 0 {
			totalLatency += int64(p.AvgLatencyMs) * p.RequestCount
		}

		if p.Status == "active" && p.CircuitState == "closed" {
			activeCount++
		}
		if p.CircuitState == "open" {
			circuitOpenCount++
		}

		successRate := float64(0)
		if p.RequestCount > 0 {
			successRate = float64(p.SuccessCount) / float64(p.RequestCount) * 100
		}

		providerItems = append(providerItems, ProviderStatItem{
			ID:           p.ID,
			Name:         p.ModelName,
			ChannelName:  p.ChannelName,
			CostPer1K:    p.ActualCostPer1KInput,
			Status:       p.Status,
			CircuitState: p.CircuitState,
			SuccessRate:  successRate,
			AvgLatencyMs: int(p.AvgLatencyMs),
			RequestCount: p.RequestCount,
			HealthScore:  p.HealthScore,
		})
	}

	// 计算流量百分比
	for i := range providerItems {
		if totalRequests > 0 {
			providerItems[i].TrafficPercentage = float64(providerItems[i].RequestCount) / float64(totalRequests) * 100
		}
	}

	// 计算总体成功率和平均延迟
	overallSuccessRate := float64(0)
	avgLatency := 0
	if totalRequests > 0 {
		overallSuccessRate = float64(totalSuccess) / float64(totalRequests) * 100
		avgLatency = int(totalLatency / totalRequests)
	}

	stats := DispatchStats{
		TotalRequests:        totalRequests,
		SuccessRate:          overallSuccessRate,
		AvgLatencyMs:         avgLatency,
		ActiveProviders:      activeCount,
		CircuitOpenProviders: circuitOpenCount,
		TotalProviders:       len(providers),
		Providers:            providerItems,
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    stats,
	})
}

// GetMetrics 获取调度指标（用于图表）
// GET /api/admin/dispatch/metrics
func (h *DispatchHandler) GetMetrics(c *gin.Context) {
	modelID := c.Query("model_id")
	period := c.DefaultQuery("period", "24h")
	granularity := c.DefaultQuery("granularity", "hour")

	// 计算时间范围和数据点数量
	var startTime time.Time
	var points int
	switch period {
	case "1h":
		startTime = time.Now().Add(-1 * time.Hour)
		points = 12 // 每5分钟一个点
	case "6h":
		startTime = time.Now().Add(-6 * time.Hour)
		points = 12 // 每30分钟一个点
	case "24h":
		startTime = time.Now().Add(-24 * time.Hour)
		points = 24 // 每小时一个点
	case "7d":
		startTime = time.Now().Add(-7 * 24 * time.Hour)
		points = 7 // 每天一个点
	default:
		startTime = time.Now().Add(-24 * time.Hour)
		points = 24
	}

	truncUnit := "hour"
	switch granularity {
	case "minute", "hour", "day":
		truncUnit = granularity
	}

	selectSQL := fmt.Sprintf(`
		DATE_TRUNC('%s', logs.created_at) as metric_time,
		COUNT(*) as request_count,
		SUM(CASE WHEN logs.status = 'success' THEN 1 ELSE 0 END) as success_count,
		COALESCE(AVG(logs.duration), 0) as avg_latency_ms
	`, truncUnit)

	query := h.db.Table("logs").
		Select(selectSQL).
		Where("logs.created_at >= ? AND logs.created_at <= ?", startTime, time.Now())

	if modelID != "" {
		query = query.Where("logs.model = ?", modelID)
	}

	var metrics []MetricRow
	groupExpr := fmt.Sprintf("DATE_TRUNC('%s', logs.created_at)", truncUnit)
	if err := query.Group(groupExpr).Order("metric_time ASC").Scan(&metrics).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"success": false,
			"error":   "查询调度指标失败: " + err.Error(),
		})
		return
	}

	// 如果没有数据，返回空列表（避免误导性的模拟数据）
	if len(metrics) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"metrics": []MetricPoint{},
			},
		})
		return
	}

	metricPoints := aggregateMetrics(metrics, points, startTime, period)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"metrics": metricPoints,
		},
	})
}

// GetEvents 获取熔断事件记录
// GET /api/admin/dispatch/events
func (h *DispatchHandler) GetEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	providerID := c.Query("provider_id")

	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 20
	}

	offset := (page - 1) * pageSize

	// 查询熔断事件
	query := h.db.Table("circuit_event_records").
		Select(`
			circuit_event_records.id,
			circuit_event_records.provider_id,
			circuit_event_records.event_type,
			circuit_event_records.reason,
			circuit_event_records.created_at,
			models.name as model_name,
			channels.name as channel_name
		`).
		Joins("LEFT JOIN model_providers ON model_providers.id = circuit_event_records.provider_id").
		Joins("LEFT JOIN models ON models.id = model_providers.model_id").
		Joins("LEFT JOIN channels ON channels.id = model_providers.channel_id")

	if providerID != "" {
		query = query.Where("circuit_event_records.provider_id = ?", providerID)
	}

	var total int64
	countQuery := h.db.Table("circuit_event_records")
	if providerID != "" {
		countQuery = countQuery.Where("provider_id = ?", providerID)
	}
	countQuery.Count(&total)

	type EventRow struct {
		ID          uint
		ProviderID  uint
		EventType   string
		Reason      string
		CreatedAt   time.Time
		ModelName   string
		ChannelName string
	}

	var events []EventRow
	if err := query.Order("circuit_event_records.created_at DESC").
		Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		// 如果查询失败，返回空列表
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"data": gin.H{
				"list":        []CircuitEvent{},
				"total":       0,
				"page":        page,
				"page_size":   pageSize,
				"total_pages": 0,
			},
		})
		return
	}

	// 转换为响应格式
	var eventList []CircuitEvent
	for _, e := range events {
		eventList = append(eventList, CircuitEvent{
			ID:           e.ID,
			ProviderID:   e.ProviderID,
			ProviderName: e.ModelName,
			ChannelName:  e.ChannelName,
			EventType:    e.EventType,
			Reason:       e.Reason,
			CreatedAt:    e.CreatedAt,
		})
	}

	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"list":        eventList,
			"total":       total,
			"page":        page,
			"page_size":   pageSize,
			"total_pages": totalPages,
		},
	})
}

// generateMockMetrics 生成模拟指标数据
func generateMockMetrics(points int, startTime time.Time) []MetricPoint {
	metrics := make([]MetricPoint, points)
	interval := time.Since(startTime) / time.Duration(points)

	for i := 0; i < points; i++ {
		t := startTime.Add(interval * time.Duration(i))
		metrics[i] = MetricPoint{
			Time:         t.Format(time.RFC3339),
			RequestCount: int64(100 + i*10),
			SuccessRate:  98.0 + float64(i%5)*0.2,
			AvgLatencyMs: 200 + i*5,
		}
	}

	return metrics
}

// aggregateMetrics 聚合指标数据
func aggregateMetrics(metrics []MetricRow, points int, startTime time.Time, period string) []MetricPoint {
	result := make([]MetricPoint, 0, points)
	interval := time.Since(startTime) / time.Duration(points)

	for i := 0; i < points; i++ {
		bucketStart := startTime.Add(interval * time.Duration(i))
		bucketEnd := bucketStart.Add(interval)

		var totalRequests int64
		var totalSuccess int64
		var totalLatency int64
		var count int

		for _, m := range metrics {
			if !m.MetricTime.Before(bucketStart) && m.MetricTime.Before(bucketEnd) {
				totalRequests += m.RequestCount
				totalSuccess += m.SuccessCount
				totalLatency += int64(m.AvgLatencyMs) * m.RequestCount
				count++
			}
		}

		successRate := float64(0)
		avgLatency := 0
		if totalRequests > 0 {
			successRate = float64(totalSuccess) / float64(totalRequests) * 100
			avgLatency = int(totalLatency / totalRequests)
		}

		result = append(result, MetricPoint{
			Time:         bucketStart.Format(time.RFC3339),
			RequestCount: totalRequests,
			SuccessRate:  successRate,
			AvgLatencyMs: avgLatency,
		})
	}

	return result
}

// RegisterDispatchRoutes 注册调度监控路由
func RegisterDispatchRoutes(r *gin.RouterGroup, db *gorm.DB) {
	h := NewDispatchHandler(db)

	dispatch := r.Group("/dispatch")
	{
		dispatch.GET("/stats", h.GetStats)
		dispatch.GET("/metrics", h.GetMetrics)
		dispatch.GET("/events", h.GetEvents)
	}
}
