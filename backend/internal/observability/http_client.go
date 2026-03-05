package observability

import (
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	registerHTTPClientMetricsOnce sync.Once

	outboundRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nexus_outbound_requests_total",
			Help: "出站请求总数",
		},
		[]string{"method", "target", "status"},
	)

	outboundRequestDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nexus_outbound_request_duration_seconds",
			Help:    "出站请求耗时（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "target"},
	)
)

func init() {
	registerHTTPClientMetrics()
}

func registerHTTPClientMetrics() {
	registerHTTPClientMetricsOnce.Do(func() {
		prometheus.MustRegister(outboundRequestsTotal, outboundRequestDurationSeconds)
	})
}

// NewHTTPClient 创建带 trace 传播能力的 HTTP 客户端
func NewHTTPClient(timeout time.Duration) *http.Client {
	client := &http.Client{
		Timeout:   timeout,
		Transport: NewTraceRoundTripper(nil),
	}
	return client
}

// NewTraceRoundTripper 创建 trace RoundTripper
func NewTraceRoundTripper(base http.RoundTripper) http.RoundTripper {
	if base == nil {
		base = http.DefaultTransport
	}
	return &traceRoundTripper{base: base}
}

type traceRoundTripper struct {
	base http.RoundTripper
}

func (transport *traceRoundTripper) RoundTrip(request *http.Request) (*http.Response, error) {
	if transport == nil || transport.base == nil {
		transport = &traceRoundTripper{base: http.DefaultTransport}
	}
	if request == nil {
		return transport.base.RoundTrip(request)
	}

	target := strings.TrimSpace(request.URL.Host)
	if target == "" {
		target = "unknown"
	}
	method := strings.ToUpper(strings.TrimSpace(request.Method))
	if method == "" {
		method = http.MethodGet
	}

	spanContext, span := StartSpan(request.Context(), "http.client", SpanKindClient, map[string]string{
		"method": method,
		"target": target,
		"path":   request.URL.Path,
	})
	clonedRequest := request.Clone(spanContext)
	if clonedRequest.Header == nil {
		clonedRequest.Header = make(http.Header)
	}
	InjectTraceHeaders(clonedRequest.Header, span.TraceID, span.SpanID, span.ParentSpanID)

	startTime := time.Now()
	response, err := transport.base.RoundTrip(clonedRequest)
	duration := time.Since(startTime)

	status := "error"
	if response != nil {
		status = strconv.Itoa(response.StatusCode)
	}
	outboundRequestsTotal.WithLabelValues(method, target, status).Inc()
	outboundRequestDurationSeconds.WithLabelValues(method, target).Observe(duration.Seconds())

	span.End(err)
	return response, err
}
