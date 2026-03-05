package observability

import (
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	defaultHTTPHistorySize        = 20000
	defaultSlowRequestThresholdMs = int64(1000)
)

var (
	registerPerformanceMetricsOnce sync.Once

	httpSlowRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nexus_http_slow_requests_total",
			Help: "慢请求总数",
		},
		[]string{"method", "path", "status"},
	)

	httpPerformanceRecorder = newHTTPHistoryRecorder(defaultHTTPHistorySize)
)

func init() {
	registerPerformanceMetrics()
}

func registerPerformanceMetrics() {
	registerPerformanceMetricsOnce.Do(func() {
		prometheus.MustRegister(httpSlowRequestsTotal)
	})
}

// HTTPRequestRecord 请求记录
type HTTPRequestRecord struct {
	Timestamp   time.Time `json:"timestamp"`
	TraceID     string    `json:"trace_id"`
	Method      string    `json:"method"`
	Path        string    `json:"path"`
	StatusCode  int       `json:"status_code"`
	LatencyMs   int64     `json:"latency_ms"`
	RequestSize int       `json:"request_size,omitempty"`
	Error       string    `json:"error,omitempty"`
}

// HTTPSlowRequest 慢请求样本
type HTTPSlowRequest struct {
	Timestamp  time.Time `json:"timestamp"`
	TraceID    string    `json:"trace_id"`
	Method     string    `json:"method"`
	Path       string    `json:"path"`
	StatusCode int       `json:"status_code"`
	LatencyMs  int64     `json:"latency_ms"`
	Error      string    `json:"error,omitempty"`
}

// HTTPBottleneck 路径瓶颈
type HTTPBottleneck struct {
	Path         string  `json:"path"`
	RequestCount int64   `json:"request_count"`
	SlowCount    int64   `json:"slow_count"`
	SlowRate     float64 `json:"slow_rate"`
	AvgLatencyMs float64 `json:"avg_latency_ms"`
	P95LatencyMs float64 `json:"p95_latency_ms"`
	MaxLatencyMs int64   `json:"max_latency_ms"`
}

// HTTPPerformanceAnalysis HTTP 性能分析
type HTTPPerformanceAnalysis struct {
	WindowSeconds   int               `json:"window_seconds"`
	SlowThresholdMs int64             `json:"slow_threshold_ms"`
	TotalRequests   int64             `json:"total_requests"`
	SlowRequests    int64             `json:"slow_requests"`
	SlowRate        float64           `json:"slow_rate"`
	AvgLatencyMs    float64           `json:"avg_latency_ms"`
	P95LatencyMs    float64           `json:"p95_latency_ms"`
	P99LatencyMs    float64           `json:"p99_latency_ms"`
	SlowSamples     []HTTPSlowRequest `json:"slow_samples"`
	Bottlenecks     []HTTPBottleneck  `json:"bottlenecks"`
}

type httpHistoryRecorder struct {
	mutex      sync.RWMutex
	maxRecords int
	records    []HTTPRequestRecord
}

func newHTTPHistoryRecorder(maxRecords int) *httpHistoryRecorder {
	if maxRecords <= 0 {
		maxRecords = defaultHTTPHistorySize
	}
	return &httpHistoryRecorder{
		maxRecords: maxRecords,
		records:    make([]HTTPRequestRecord, 0, maxRecords),
	}
}

// RecordHTTPRequest 记录 HTTP 请求
func RecordHTTPRequest(record HTTPRequestRecord) {
	if record.Timestamp.IsZero() {
		record.Timestamp = time.Now()
	}
	record.Method = strings.ToUpper(strings.TrimSpace(record.Method))
	record.Path = strings.TrimSpace(record.Path)
	if record.Path == "" {
		record.Path = "unknown"
	}
	record.TraceID = strings.ToLower(strings.TrimSpace(record.TraceID))
	record.Error = strings.TrimSpace(record.Error)

	httpPerformanceRecorder.append(record)

	if record.LatencyMs >= defaultSlowRequestThresholdMs {
		httpSlowRequestsTotal.WithLabelValues(
			record.Method,
			record.Path,
			strconv.Itoa(record.StatusCode),
		).Inc()
	}
}

func (r *httpHistoryRecorder) append(record HTTPRequestRecord) {
	if r == nil {
		return
	}

	r.mutex.Lock()
	defer r.mutex.Unlock()

	r.records = append(r.records, record)
	if len(r.records) <= r.maxRecords {
		return
	}

	overflow := len(r.records) - r.maxRecords
	trimmed := make([]HTTPRequestRecord, r.maxRecords)
	copy(trimmed, r.records[overflow:])
	r.records = trimmed
}

func (r *httpHistoryRecorder) snapshot(window time.Duration) []HTTPRequestRecord {
	if r == nil {
		return nil
	}

	cutoff := time.Now().Add(-window)

	r.mutex.RLock()
	defer r.mutex.RUnlock()

	filtered := make([]HTTPRequestRecord, 0, len(r.records))
	for _, record := range r.records {
		if window > 0 && record.Timestamp.Before(cutoff) {
			continue
		}
		filtered = append(filtered, record)
	}
	return filtered
}

// AnalyzeHTTPPerformance 分析指定窗口的 HTTP 性能
func AnalyzeHTTPPerformance(window time.Duration, slowThresholdMs int64, topN int) *HTTPPerformanceAnalysis {
	if window <= 0 {
		window = 1 * time.Hour
	}
	if slowThresholdMs <= 0 {
		slowThresholdMs = defaultSlowRequestThresholdMs
	}
	if topN <= 0 {
		topN = 10
	}

	records := httpPerformanceRecorder.snapshot(window)
	result := &HTTPPerformanceAnalysis{
		WindowSeconds:   int(window.Seconds()),
		SlowThresholdMs: slowThresholdMs,
		SlowSamples:     make([]HTTPSlowRequest, 0),
		Bottlenecks:     make([]HTTPBottleneck, 0),
	}
	if len(records) == 0 {
		return result
	}

	latencySeries := make([]float64, 0, len(records))
	slowSamples := make([]HTTPSlowRequest, 0)
	pathStats := make(map[string][]HTTPRequestRecord, len(records))

	var totalLatency float64
	for _, record := range records {
		latency := float64(record.LatencyMs)
		latencySeries = append(latencySeries, latency)
		totalLatency += latency

		pathStats[record.Path] = append(pathStats[record.Path], record)
		if record.LatencyMs >= slowThresholdMs {
			slowSamples = append(slowSamples, HTTPSlowRequest{
				Timestamp:  record.Timestamp,
				TraceID:    record.TraceID,
				Method:     record.Method,
				Path:       record.Path,
				StatusCode: record.StatusCode,
				LatencyMs:  record.LatencyMs,
				Error:      record.Error,
			})
		}
	}

	sort.Float64s(latencySeries)
	result.TotalRequests = int64(len(records))
	result.SlowRequests = int64(len(slowSamples))
	result.AvgLatencyMs = totalLatency / float64(len(records))
	result.P95LatencyMs = percentile(latencySeries, 0.95)
	result.P99LatencyMs = percentile(latencySeries, 0.99)
	if result.TotalRequests > 0 {
		result.SlowRate = float64(result.SlowRequests) / float64(result.TotalRequests) * 100
	}

	sort.Slice(slowSamples, func(leftIndex, rightIndex int) bool {
		if slowSamples[leftIndex].LatencyMs == slowSamples[rightIndex].LatencyMs {
			return slowSamples[leftIndex].Timestamp.After(slowSamples[rightIndex].Timestamp)
		}
		return slowSamples[leftIndex].LatencyMs > slowSamples[rightIndex].LatencyMs
	})
	if len(slowSamples) > topN {
		slowSamples = slowSamples[:topN]
	}
	result.SlowSamples = slowSamples

	bottlenecks := make([]HTTPBottleneck, 0, len(pathStats))
	for path, grouped := range pathStats {
		if len(grouped) == 0 {
			continue
		}

		pathLatencySeries := make([]float64, 0, len(grouped))
		var pathTotalLatency float64
		var pathSlowCount int64
		var pathMaxLatency int64
		for _, item := range grouped {
			pathLatency := float64(item.LatencyMs)
			pathLatencySeries = append(pathLatencySeries, pathLatency)
			pathTotalLatency += pathLatency
			if item.LatencyMs >= slowThresholdMs {
				pathSlowCount++
			}
			if item.LatencyMs > pathMaxLatency {
				pathMaxLatency = item.LatencyMs
			}
		}

		sort.Float64s(pathLatencySeries)
		pathRequestCount := int64(len(grouped))
		pathSlowRate := float64(0)
		if pathRequestCount > 0 {
			pathSlowRate = float64(pathSlowCount) / float64(pathRequestCount) * 100
		}

		bottlenecks = append(bottlenecks, HTTPBottleneck{
			Path:         path,
			RequestCount: pathRequestCount,
			SlowCount:    pathSlowCount,
			SlowRate:     pathSlowRate,
			AvgLatencyMs: pathTotalLatency / float64(len(grouped)),
			P95LatencyMs: percentile(pathLatencySeries, 0.95),
			MaxLatencyMs: pathMaxLatency,
		})
	}

	sort.Slice(bottlenecks, func(leftIndex, rightIndex int) bool {
		left := bottlenecks[leftIndex]
		right := bottlenecks[rightIndex]
		if left.P95LatencyMs == right.P95LatencyMs {
			if left.SlowRate == right.SlowRate {
				return left.RequestCount > right.RequestCount
			}
			return left.SlowRate > right.SlowRate
		}
		return left.P95LatencyMs > right.P95LatencyMs
	})
	if len(bottlenecks) > topN {
		bottlenecks = bottlenecks[:topN]
	}
	result.Bottlenecks = bottlenecks

	return result
}

func percentile(sortedValues []float64, quantile float64) float64 {
	if len(sortedValues) == 0 {
		return 0
	}
	if quantile <= 0 {
		return sortedValues[0]
	}
	if quantile >= 1 {
		return sortedValues[len(sortedValues)-1]
	}

	index := quantile * float64(len(sortedValues)-1)
	lowerIndex := int(index)
	upperIndex := lowerIndex + 1
	if upperIndex >= len(sortedValues) {
		return sortedValues[lowerIndex]
	}
	fraction := index - float64(lowerIndex)
	return sortedValues[lowerIndex]*(1-fraction) + sortedValues[upperIndex]*fraction
}
