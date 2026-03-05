package service

import (
	"context"
	"errors"
	"fmt"
	"math"
	"time"

	"nexus-api/internal/observability"

	"gorm.io/gorm"
)

const (
	defaultPerformanceWindow      = 1 * time.Hour
	defaultSlowThresholdMs  int64 = 1000
	defaultPerformanceTopN        = 10

	defaultCapacityLookbackHours = 24 * 7
	defaultCapacityForecastHours = 24
)

// ObservabilityService 可观测性服务
type ObservabilityService interface {
	AnalyzePerformance(ctx context.Context, query *PerformanceAnalysisQuery) (*PerformanceAnalysisResult, error)
	ForecastCapacity(ctx context.Context, query *CapacityForecastQuery) (*CapacityForecastResult, error)
}

// PerformanceAnalysisQuery 性能分析参数
type PerformanceAnalysisQuery struct {
	Window          time.Duration
	SlowThresholdMs int64
	TopN            int
}

// PerformanceAnalysisSummary 性能摘要
type PerformanceAnalysisSummary struct {
	TotalRequests int64   `json:"total_requests"`
	SlowRequests  int64   `json:"slow_requests"`
	SlowRate      float64 `json:"slow_rate"`
	AvgLatencyMs  float64 `json:"avg_latency_ms"`
	P95LatencyMs  float64 `json:"p95_latency_ms"`
	P99LatencyMs  float64 `json:"p99_latency_ms"`
}

// PerformanceSlowSample 慢请求样本
type PerformanceSlowSample struct {
	Timestamp  time.Time `json:"timestamp"`
	TraceID    string    `json:"trace_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
}

// PerformanceBottleneck 性能瓶颈项
type PerformanceBottleneck struct {
	Path         string  `json:"path"`
	RequestCount int64   `json:"request_count"`
	SlowCount    int64   `json:"slow_count"`
	SlowRate     float64 `json:"slow_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	MaxLatencyMs int64   `json:"max_latency_ms"`
}

// PerformanceAnalysisResult 性能分析结果
type PerformanceAnalysisResult struct {
	GeneratedAt      time.Time                 `json:"generated_at"`
	WindowSeconds    int                       `json:"window_seconds"`
	SlowThresholdMs  int64                     `json:"slow_threshold_ms"`
	Summary          PerformanceAnalysisSummary `json:"summary"`
	SlowSamples      []PerformanceSlowSample   `json:"slow_samples"`
	Bottlenecks      []PerformanceBottleneck   `json:"bottlenecks"`
	Recommendations  []string                  `json:"recommendations"`
}

// CapacityForecastQuery 容量预测参数
type CapacityForecastQuery struct {
	LookbackHours int
	ForecastHours int
}

// CapacityLoadTrend 负载趋势点
type CapacityLoadTrend struct {
	Time         time.Time `json:"time"`
	RequestCount int64     `json:"request_count"`
	TotalTokens  int64     `json:"total_tokens"`
	AvgLatencyMs float64   `json:"avg_latency_ms"`
}

// CapacityForecastPoint 容量预测点
type CapacityForecastPoint struct {
	Time                   time.Time `json:"time"`
	PredictedRequests      float64   `json:"predicted_requests"`
	PredictedTokens        float64   `json:"predicted_tokens"`
	RequestsConfidenceLow  float64   `json:"requests_confidence_low"`
	RequestsConfidenceHigh float64   `json:"requests_confidence_high"`
}

// CapacityResourcePrediction 资源使用预测
type CapacityResourcePrediction struct {
	CurrentHourlyRequests             float64 `json:"current_hourly_requests"`
	HistoricalPeakHourlyRequests      float64 `json:"historical_peak_hourly_requests"`
	PredictedPeakHourlyRequests       float64 `json:"predicted_peak_hourly_requests"`
	PredictedAverageHourlyRequests    float64 `json:"predicted_average_hourly_requests"`
	PredictedPeakHourlyTokens         float64 `json:"predicted_peak_hourly_tokens"`
	EstimatedCapacityUtilization      float64 `json:"estimated_capacity_utilization"`
	RecommendedHourlyRequestCapacity  float64 `json:"recommended_hourly_request_capacity"`
}

// CapacityForecastResult 容量预测结果
type CapacityForecastResult struct {
	GeneratedAt      time.Time                 `json:"generated_at"`
	LookbackHours    int                       `json:"lookback_hours"`
	ForecastHours    int                       `json:"forecast_hours"`
	Trend            []CapacityLoadTrend       `json:"trend"`
	Forecast         []CapacityForecastPoint   `json:"forecast"`
	ResourceUsage    CapacityResourcePrediction `json:"resource_usage"`
	Recommendations  []string                  `json:"recommendations"`
}

type observabilityService struct {
	db *gorm.DB
}

type hourlyLoadRow struct {
	Bucket       time.Time `gorm:"column:bucket"`
	RequestCount int64     `gorm:"column:request_count"`
	TotalTokens  int64     `gorm:"column:total_tokens"`
	AvgLatencyMs float64   `gorm:"column:avg_latency_ms"`
}

// NewObservabilityService 创建可观测性服务
func NewObservabilityService(db *gorm.DB) ObservabilityService {
	return &observabilityService{db: db}
}

func (s *observabilityService) AnalyzePerformance(_ context.Context, query *PerformanceAnalysisQuery) (*PerformanceAnalysisResult, error) {
	normalizedQuery := normalizePerformanceQuery(query)
	httpAnalysis := observability.AnalyzeHTTPPerformance(normalizedQuery.Window, normalizedQuery.SlowThresholdMs, normalizedQuery.TopN)
	if httpAnalysis == nil {
		httpAnalysis = &observability.HTTPPerformanceAnalysis{
			WindowSeconds:   int(normalizedQuery.Window.Seconds()),
			SlowThresholdMs: normalizedQuery.SlowThresholdMs,
		}
	}

	result := &PerformanceAnalysisResult{
		GeneratedAt:     time.Now(),
		WindowSeconds:   httpAnalysis.WindowSeconds,
		SlowThresholdMs: httpAnalysis.SlowThresholdMs,
		Summary: PerformanceAnalysisSummary{
			TotalRequests: httpAnalysis.TotalRequests,
			SlowRequests:  httpAnalysis.SlowRequests,
			SlowRate:      roundFloat(httpAnalysis.SlowRate, 2),
			AvgLatencyMs:  roundFloat(httpAnalysis.AvgLatencyMs, 2),
			P95LatencyMs:  roundFloat(httpAnalysis.P95LatencyMs, 2),
			P99LatencyMs:  roundFloat(httpAnalysis.P99LatencyMs, 2),
		},
		SlowSamples:     make([]PerformanceSlowSample, 0, len(httpAnalysis.SlowSamples)),
		Bottlenecks:     make([]PerformanceBottleneck, 0, len(httpAnalysis.Bottlenecks)),
		Recommendations: make([]string, 0),
	}

	for _, sample := range httpAnalysis.SlowSamples {
		result.SlowSamples = append(result.SlowSamples, PerformanceSlowSample{
			Timestamp:  sample.Timestamp,
			TraceID:    sample.TraceID,
			Method:     sample.Method,
			Path:       sample.Path,
			StatusCode: sample.StatusCode,
			LatencyMs:  sample.LatencyMs,
			Error:      sample.Error,
		})
	}
	for _, bottleneck := range httpAnalysis.Bottlenecks {
		result.Bottlenecks = append(result.Bottlenecks, PerformanceBottleneck{
			Path:         bottleneck.Path,
			RequestCount: bottleneck.RequestCount,
			SlowCount:    bottleneck.SlowCount,
			SlowRate:     roundFloat(bottleneck.SlowRate, 2),
			AvgLatencyMs: roundFloat(bottleneck.AvgLatencyMs, 2),
			P95LatencyMs: roundFloat(bottleneck.P95LatencyMs, 2),
			MaxLatencyMs: bottleneck.MaxLatencyMs,
		})
	}

	result.Recommendations = buildPerformanceRecommendations(result)
	return result, nil
}

func (s *observabilityService) ForecastCapacity(ctx context.Context, query *CapacityForecastQuery) (*CapacityForecastResult, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("observability service 未初始化")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	normalizedQuery := normalizeCapacityQuery(query)
	lookbackHours := normalizedQuery.LookbackHours
	forecastHours := normalizedQuery.ForecastHours

	endTime := time.Now().Truncate(time.Hour)
	startTime := endTime.Add(-time.Duration(lookbackHours-1) * time.Hour)

	rows, err := s.queryHourlyLoadRows(ctx, startTime, endTime.Add(time.Hour))
	if err != nil {
		return nil, err
	}

	trend := fillHourlyTrend(startTime, endTime, rows)
	if len(trend) == 0 {
		return &CapacityForecastResult{
			GeneratedAt:     time.Now(),
			LookbackHours:   lookbackHours,
			ForecastHours:   forecastHours,
			Trend:           []CapacityLoadTrend{},
			Forecast:        []CapacityForecastPoint{},
			ResourceUsage:   CapacityResourcePrediction{},
			Recommendations: []string{"暂无足够数据，请先积累请求数据后再进行容量预测"},
		}, nil
	}

	requestSeries := make([]float64, 0, len(trend))
	tokenSeries := make([]float64, 0, len(trend))
	for _, point := range trend {
		requestSeries = append(requestSeries, float64(point.RequestCount))
		tokenSeries = append(tokenSeries, float64(point.TotalTokens))
	}

	requestSlope, requestIntercept := linearRegression(requestSeries)
	tokenSlope, tokenIntercept := linearRegression(tokenSeries)
	requestResidualStdDev := regressionResidualStdDev(requestSeries, requestSlope, requestIntercept)

	forecast := make([]CapacityForecastPoint, 0, forecastHours)
	forecastStart := endTime.Add(1 * time.Hour)
	for index := 0; index < forecastHours; index++ {
		seriesIndex := float64(len(requestSeries) + index)
		predictedRequests := math.Max(0, requestIntercept+requestSlope*seriesIndex)
		predictedTokens := math.Max(0, tokenIntercept+tokenSlope*seriesIndex)

		margin := 1.96 * requestResidualStdDev
		requestLow := math.Max(0, predictedRequests-margin)
		requestHigh := predictedRequests + margin

		forecast = append(forecast, CapacityForecastPoint{
			Time:                   forecastStart.Add(time.Duration(index) * time.Hour),
			PredictedRequests:      roundFloat(predictedRequests, 2),
			PredictedTokens:        roundFloat(predictedTokens, 2),
			RequestsConfidenceLow:  roundFloat(requestLow, 2),
			RequestsConfidenceHigh: roundFloat(requestHigh, 2),
		})
	}

	resourcePrediction := buildCapacityResourcePrediction(trend, forecast)
	recommendations := buildCapacityRecommendations(resourcePrediction, requestSlope)

	return &CapacityForecastResult{
		GeneratedAt:     time.Now(),
		LookbackHours:   lookbackHours,
		ForecastHours:   forecastHours,
		Trend:           trend,
		Forecast:        forecast,
		ResourceUsage:   resourcePrediction,
		Recommendations: recommendations,
	}, nil
}

func (s *observabilityService) queryHourlyLoadRows(ctx context.Context, startTime time.Time, endTime time.Time) ([]hourlyLoadRow, error) {
	rows := make([]hourlyLoadRow, 0)
	err := s.db.WithContext(ctx).
		Table("logs").
		Select(`
			DATE_TRUNC('hour', created_at) AS bucket,
			COUNT(*) AS request_count,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) AS total_tokens,
			COALESCE(AVG(duration), 0) AS avg_latency_ms
		`).
		Where("created_at >= ? AND created_at < ?", startTime, endTime).
		Group("DATE_TRUNC('hour', created_at)").
		Order("bucket ASC").
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func normalizePerformanceQuery(query *PerformanceAnalysisQuery) *PerformanceAnalysisQuery {
	if query == nil {
		query = &PerformanceAnalysisQuery{}
	}
	if query.Window <= 0 {
		query.Window = defaultPerformanceWindow
	}
	if query.Window > 7*24*time.Hour {
		query.Window = 7 * 24 * time.Hour
	}
	if query.SlowThresholdMs <= 0 {
		query.SlowThresholdMs = defaultSlowThresholdMs
	}
	if query.SlowThresholdMs > 120000 {
		query.SlowThresholdMs = 120000
	}
	if query.TopN <= 0 {
		query.TopN = defaultPerformanceTopN
	}
	if query.TopN > 100 {
		query.TopN = 100
	}
	return query
}

func normalizeCapacityQuery(query *CapacityForecastQuery) *CapacityForecastQuery {
	if query == nil {
		query = &CapacityForecastQuery{}
	}
	if query.LookbackHours <= 0 {
		query.LookbackHours = defaultCapacityLookbackHours
	}
	if query.LookbackHours > 24*30 {
		query.LookbackHours = 24 * 30
	}
	if query.ForecastHours <= 0 {
		query.ForecastHours = defaultCapacityForecastHours
	}
	if query.ForecastHours > 24*7 {
		query.ForecastHours = 24 * 7
	}
	return query
}

func fillHourlyTrend(startTime time.Time, endTime time.Time, rows []hourlyLoadRow) []CapacityLoadTrend {
	if endTime.Before(startTime) {
		return []CapacityLoadTrend{}
	}

	rowMap := make(map[int64]hourlyLoadRow, len(rows))
	for _, row := range rows {
		key := row.Bucket.Unix()
		rowMap[key] = row
	}

	points := make([]CapacityLoadTrend, 0)
	for cursor := startTime; !cursor.After(endTime); cursor = cursor.Add(time.Hour) {
		key := cursor.Unix()
		row, ok := rowMap[key]
		if !ok {
			points = append(points, CapacityLoadTrend{
				Time:         cursor,
				RequestCount: 0,
				TotalTokens:  0,
				AvgLatencyMs: 0,
			})
			continue
		}

		points = append(points, CapacityLoadTrend{
			Time:         cursor,
			RequestCount: row.RequestCount,
			TotalTokens:  row.TotalTokens,
			AvgLatencyMs: roundFloat(row.AvgLatencyMs, 2),
		})
	}
	return points
}

func linearRegression(series []float64) (float64, float64) {
	n := len(series)
	if n == 0 {
		return 0, 0
	}
	if n == 1 {
		return 0, series[0]
	}

	var sumX float64
	var sumY float64
	var sumXY float64
	var sumXX float64

	for index, value := range series {
		x := float64(index)
		sumX += x
		sumY += value
		sumXY += x * value
		sumXX += x * x
	}

	denominator := float64(n)*sumXX - sumX*sumX
	if denominator == 0 {
		return 0, sumY / float64(n)
	}

	slope := (float64(n)*sumXY - sumX*sumY) / denominator
	intercept := (sumY - slope*sumX) / float64(n)
	return slope, intercept
}

func regressionResidualStdDev(series []float64, slope float64, intercept float64) float64 {
	n := len(series)
	if n <= 2 {
		return 0
	}

	var sumSquaredError float64
	for index, value := range series {
		x := float64(index)
		predicted := intercept + slope*x
		residual := value - predicted
		sumSquaredError += residual * residual
	}

	variance := sumSquaredError / float64(n-2)
	if variance <= 0 {
		return 0
	}
	return math.Sqrt(variance)
}

func buildCapacityResourcePrediction(trend []CapacityLoadTrend, forecast []CapacityForecastPoint) CapacityResourcePrediction {
	currentHourlyRequests := averageRecentRequests(trend, 6)
	historicalPeakRequests := maxTrendRequests(trend)
	predictedPeakRequests := maxForecastRequests(forecast)
	predictedAverageRequests := averageForecastRequests(forecast)
	predictedPeakTokens := maxForecastTokens(forecast)

	baselineCapacity := historicalPeakRequests * 1.3
	if baselineCapacity < 1 {
		baselineCapacity = math.Max(predictedPeakRequests*1.2, 1)
	}

	utilization := float64(0)
	if baselineCapacity > 0 {
		utilization = predictedPeakRequests / baselineCapacity * 100
	}
	recommendedCapacity := predictedPeakRequests * 1.2
	if recommendedCapacity < 1 {
		recommendedCapacity = 1
	}

	return CapacityResourcePrediction{
		CurrentHourlyRequests:            roundFloat(currentHourlyRequests, 2),
		HistoricalPeakHourlyRequests:     roundFloat(historicalPeakRequests, 2),
		PredictedPeakHourlyRequests:      roundFloat(predictedPeakRequests, 2),
		PredictedAverageHourlyRequests:   roundFloat(predictedAverageRequests, 2),
		PredictedPeakHourlyTokens:        roundFloat(predictedPeakTokens, 2),
		EstimatedCapacityUtilization:     roundFloat(utilization, 2),
		RecommendedHourlyRequestCapacity: roundFloat(recommendedCapacity, 2),
	}
}

func buildPerformanceRecommendations(result *PerformanceAnalysisResult) []string {
	if result == nil {
		return []string{"暂无分析结果"}
	}

	recommendations := make([]string, 0)
	if result.Summary.TotalRequests == 0 {
		return []string{"当前窗口内没有请求流量，建议扩大时间窗口后重试"}
	}

	if result.Summary.SlowRate >= 10 {
		recommendations = append(recommendations, "慢请求占比超过 10%，建议优先治理高延迟路径并排查上游依赖")
	} else if result.Summary.SlowRate >= 3 {
		recommendations = append(recommendations, "慢请求占比较高，建议针对慢样本 trace_id 做链路回放排查")
	} else {
		recommendations = append(recommendations, "慢请求占比较低，当前性能处于可控区间")
	}

	if result.Summary.P95LatencyMs >= float64(result.SlowThresholdMs)*1.5 {
		recommendations = append(recommendations, "P95 延迟已显著超阈值，建议优先检查数据库与上游网络抖动")
	}

	if len(result.Bottlenecks) > 0 {
		bottleneck := result.Bottlenecks[0]
		recommendations = append(recommendations, fmt.Sprintf("当前瓶颈路径：%s（P95 %.2fms，慢请求率 %.2f%%）", bottleneck.Path, bottleneck.P95LatencyMs, bottleneck.SlowRate))
	}

	return recommendations
}

func buildCapacityRecommendations(resource CapacityResourcePrediction, requestSlope float64) []string {
	recommendations := make([]string, 0)

	if resource.EstimatedCapacityUtilization >= 85 {
		recommendations = append(recommendations, "预测峰值容量利用率超过 85%，建议立即扩容并预留至少 20% 余量")
	} else if resource.EstimatedCapacityUtilization >= 65 {
		recommendations = append(recommendations, "预测峰值容量利用率接近上限，建议提前做弹性扩容预案")
	} else {
		recommendations = append(recommendations, "预测容量余量充足，可维持当前资源配置并持续观察")
	}

	dailyGrowth := requestSlope * 24
	if dailyGrowth > 0.5 {
		recommendations = append(recommendations, fmt.Sprintf("请求量呈上升趋势，预计日均每小时请求增加 %.2f，建议提升自动扩缩容灵敏度", dailyGrowth))
	} else if dailyGrowth < -0.5 {
		recommendations = append(recommendations, "请求量呈下降趋势，可评估在低峰时段回收冗余资源")
	} else {
		recommendations = append(recommendations, "请求趋势整体平稳，建议保持当前容量策略")
	}

	if resource.PredictedPeakHourlyRequests > 0 && resource.PredictedPeakHourlyTokens > 0 {
		recommendations = append(recommendations, "建议同步监控请求峰值与 Token 峰值，避免仅按 QPS 规划导致 Token 维度过载")
	}

	return recommendations
}

func averageRecentRequests(trend []CapacityLoadTrend, sampleSize int) float64 {
	if len(trend) == 0 {
		return 0
	}
	if sampleSize <= 0 {
		sampleSize = 1
	}
	if sampleSize > len(trend) {
		sampleSize = len(trend)
	}

	start := len(trend) - sampleSize
	var total float64
	for _, item := range trend[start:] {
		total += float64(item.RequestCount)
	}
	return total / float64(sampleSize)
}

func maxTrendRequests(trend []CapacityLoadTrend) float64 {
	maxValue := float64(0)
	for _, item := range trend {
		if float64(item.RequestCount) > maxValue {
			maxValue = float64(item.RequestCount)
		}
	}
	return maxValue
}

func maxForecastRequests(forecast []CapacityForecastPoint) float64 {
	maxValue := float64(0)
	for _, item := range forecast {
		if item.PredictedRequests > maxValue {
			maxValue = item.PredictedRequests
		}
	}
	return maxValue
}

func maxForecastTokens(forecast []CapacityForecastPoint) float64 {
	maxValue := float64(0)
	for _, item := range forecast {
		if item.PredictedTokens > maxValue {
			maxValue = item.PredictedTokens
		}
	}
	return maxValue
}

func averageForecastRequests(forecast []CapacityForecastPoint) float64 {
	if len(forecast) == 0 {
		return 0
	}
	var total float64
	for _, item := range forecast {
		total += item.PredictedRequests
	}
	return total / float64(len(forecast))
}

func roundFloat(value float64, precision int) float64 {
	if precision < 0 {
		precision = 0
	}
	if math.IsNaN(value) || math.IsInf(value, 0) {
		return 0
	}
	factor := math.Pow(10, float64(precision))
	return math.Round(value*factor) / factor
}
