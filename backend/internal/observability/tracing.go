package observability

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

const (
	HeaderTraceID      = "X-Trace-ID"
	HeaderSpanID       = "X-Span-ID"
	HeaderParentSpanID = "X-Parent-Span-ID"
	HeaderTraceParent  = "traceparent"

	SpanKindServer = "server"
	SpanKindClient = "client"
)

var (
	traceIDPattern = regexp.MustCompile("^[0-9a-f]{32}$")
	spanIDPattern  = regexp.MustCompile("^[0-9a-f]{16}$")
)

type traceContextKey string

const (
	contextKeyTraceID traceContextKey = "trace_id"
	contextKeySpanID  traceContextKey = "span_id"
)

var (
	registerTracingMetricsOnce sync.Once

	traceSpansTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "nexus_trace_spans_total",
			Help: "Span 总数",
		},
		[]string{"kind", "name", "status"},
	)

	traceSpanDurationSeconds = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "nexus_trace_span_duration_seconds",
			Help:    "Span 耗时（秒）",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"kind", "name", "status"},
	)

	traceActiveSpans = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "nexus_trace_active_spans",
			Help: "活跃 Span 数量",
		},
		[]string{"kind"},
	)
)

func init() {
	registerTracingMetrics()
}

func registerTracingMetrics() {
	registerTracingMetricsOnce.Do(func() {
		prometheus.MustRegister(traceSpansTotal, traceSpanDurationSeconds, traceActiveSpans)
	})
}

// Span 轻量级 span 记录
type Span struct {
	Name         string
	Kind         string
	TraceID      string
	SpanID       string
	ParentSpanID string
	StartTime    time.Time
	attributes   map[string]string
}

// StartSpan 基于上下文创建 span
func StartSpan(ctx context.Context, name string, kind string, attributes map[string]string) (context.Context, *Span) {
	if ctx == nil {
		ctx = context.Background()
	}

	traceID := TraceIDFromContext(ctx)
	if traceID == "" {
		traceID = NewTraceID()
	}

	parentSpanID := SpanIDFromContext(ctx)
	spanID := NewSpanID()

	if strings.TrimSpace(kind) == "" {
		kind = SpanKindServer
	}
	if strings.TrimSpace(name) == "" {
		name = "span"
	}

	span := &Span{
		Name:         name,
		Kind:         kind,
		TraceID:      traceID,
		SpanID:       spanID,
		ParentSpanID: parentSpanID,
		StartTime:    time.Now(),
		attributes:   cloneAttributes(attributes),
	}

	traceActiveSpans.WithLabelValues(span.Kind).Inc()

	nextContext := ContextWithTrace(ctx, traceID, spanID)
	return nextContext, span
}

// End 结束 span 并记录指标
func (s *Span) End(err error) {
	if s == nil {
		return
	}

	status := "ok"
	errMessage := ""
	if err != nil {
		status = "error"
		errMessage = strings.TrimSpace(err.Error())
	}

	duration := time.Since(s.StartTime)
	traceSpansTotal.WithLabelValues(s.Kind, s.Name, status).Inc()
	traceSpanDurationSeconds.WithLabelValues(s.Kind, s.Name, status).Observe(duration.Seconds())
	traceActiveSpans.WithLabelValues(s.Kind).Dec()

	log.Printf(
		"[Trace] trace=%s span=%s parent=%s kind=%s name=%s duration_ms=%d status=%s error=%s attrs=%s",
		s.TraceID,
		s.SpanID,
		s.ParentSpanID,
		s.Kind,
		s.Name,
		duration.Milliseconds(),
		status,
		errMessage,
		formatAttributes(s.attributes),
	)
}

// ContextWithTrace 写入 trace/span 到上下文
func ContextWithTrace(ctx context.Context, traceID string, spanID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	if strings.TrimSpace(traceID) != "" {
		ctx = context.WithValue(ctx, contextKeyTraceID, strings.ToLower(strings.TrimSpace(traceID)))
	}
	if strings.TrimSpace(spanID) != "" {
		ctx = context.WithValue(ctx, contextKeySpanID, strings.ToLower(strings.TrimSpace(spanID)))
	}
	return ctx
}

// TraceIDFromContext 从上下文读取 traceID
func TraceIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, ok := ctx.Value(contextKeyTraceID).(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}

// SpanIDFromContext 从上下文读取 spanID
func SpanIDFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	value, ok := ctx.Value(contextKeySpanID).(string)
	if !ok {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(value))
}

// ExtractOrCreateTrace 从请求头提取 trace 信息，不存在则创建
func ExtractOrCreateTrace(header http.Header, fallbackTraceID string) (string, string) {
	if header == nil {
		return normalizeTraceIDOrNew(fallbackTraceID), ""
	}

	traceID := normalizeTraceID(header.Get(HeaderTraceID))
	spanID := normalizeSpanID(header.Get(HeaderSpanID))

	if traceID == "" {
		if parentTraceID, parentSpanID := parseTraceParent(header.Get(HeaderTraceParent)); parentTraceID != "" {
			traceID = parentTraceID
			if spanID == "" {
				spanID = parentSpanID
			}
		}
	}

	if traceID == "" {
		traceID = normalizeTraceIDOrNew(fallbackTraceID)
	}

	return traceID, spanID
}

// InjectTraceHeaders 注入 trace/span 头
func InjectTraceHeaders(headers http.Header, traceID string, spanID string, parentSpanID string) {
	if headers == nil {
		return
	}

	normalizedTraceID := normalizeTraceID(traceID)
	if normalizedTraceID == "" {
		normalizedTraceID = NewTraceID()
	}

	normalizedSpanID := normalizeSpanID(spanID)
	if normalizedSpanID == "" {
		normalizedSpanID = NewSpanID()
	}

	headers.Set(HeaderTraceID, normalizedTraceID)
	headers.Set(HeaderSpanID, normalizedSpanID)

	normalizedParentSpanID := normalizeSpanID(parentSpanID)
	if normalizedParentSpanID != "" {
		headers.Set(HeaderParentSpanID, normalizedParentSpanID)
	} else {
		headers.Del(HeaderParentSpanID)
	}

	headers.Set(HeaderTraceParent, buildTraceParent(normalizedTraceID, normalizedSpanID))
}

// NewTraceID 生成 32 位十六进制 traceID
func NewTraceID() string {
	return randomHex(16)
}

// NewSpanID 生成 16 位十六进制 spanID
func NewSpanID() string {
	return randomHex(8)
}

func normalizeTraceIDOrNew(raw string) string {
	normalized := normalizeTraceID(raw)
	if normalized != "" {
		return normalized
	}
	return NewTraceID()
}

func normalizeTraceID(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", "")
	if traceIDPattern.MatchString(value) {
		return value
	}
	return ""
}

func normalizeSpanID(raw string) string {
	value := strings.ToLower(strings.TrimSpace(raw))
	value = strings.ReplaceAll(value, "-", "")
	if spanIDPattern.MatchString(value) {
		return value
	}
	return ""
}

func parseTraceParent(raw string) (string, string) {
	value := strings.TrimSpace(raw)
	parts := strings.Split(value, "-")
	if len(parts) < 4 {
		return "", ""
	}

	traceID := normalizeTraceID(parts[1])
	spanID := normalizeSpanID(parts[2])
	if traceID == "" || spanID == "" {
		return "", ""
	}
	return traceID, spanID
}

func buildTraceParent(traceID string, spanID string) string {
	return fmt.Sprintf("00-%s-%s-01", traceID, spanID)
}

func randomHex(byteLength int) string {
	if byteLength <= 0 {
		return ""
	}
	buffer := make([]byte, byteLength)
	if _, err := rand.Read(buffer); err != nil {
		return strings.Repeat("0", byteLength*2)
	}
	return hex.EncodeToString(buffer)
}

func cloneAttributes(attributes map[string]string) map[string]string {
	if len(attributes) == 0 {
		return nil
	}
	cloned := make(map[string]string, len(attributes))
	for key, value := range attributes {
		trimmedKey := strings.TrimSpace(key)
		if trimmedKey == "" {
			continue
		}
		cloned[trimmedKey] = strings.TrimSpace(value)
	}
	return cloned
}

func formatAttributes(attributes map[string]string) string {
	if len(attributes) == 0 {
		return "{}"
	}

	parts := make([]string, 0, len(attributes))
	for key, value := range attributes {
		parts = append(parts, fmt.Sprintf("%s=%s", key, value))
	}
	return "{" + strings.Join(parts, ",") + "}"
}
