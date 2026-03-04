package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"

	"nexus-api/internal/cache"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// LogEntry 日志条目
type LogEntry struct {
	RequestID    string        `json:"request_id"`
	Timestamp    time.Time     `json:"timestamp"`
	Method       string        `json:"method"`
	Path         string        `json:"path"`
	Query        string        `json:"query,omitempty"`
	ClientIP     string        `json:"client_ip"`
	UserAgent    string        `json:"user_agent"`
	UserID       string        `json:"user_id,omitempty"`
	StatusCode   int           `json:"status_code"`
	Latency      time.Duration `json:"latency"`
	LatencyMs    int64         `json:"latency_ms"`
	RequestSize  int           `json:"request_size"`
	ResponseSize int           `json:"response_size"`
	Error        string        `json:"error,omitempty"`
}

// responseWriter 自定义响应写入器，用于捕获响应大小
type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
	size int
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	n, err := w.ResponseWriter.Write(b)
	w.size += n
	return n, err
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	n, err := w.ResponseWriter.WriteString(s)
	w.size += n
	return n, err
}

// LoggerMiddleware 日志中间件
func LoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求 ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 记录开始时间
		startTime := time.Now()

		// 获取请求体大小
		var requestSize int
		if c.Request.Body != nil {
			bodyBytes, _ := io.ReadAll(c.Request.Body)
			requestSize = len(bodyBytes)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		// 包装响应写入器
		blw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(startTime)

		// 获取用户 ID
		var userID string
		if uid, exists := c.Get(ContextKeyUserID); exists {
			userID = uid.(uuid.UUID).String()
		}

		// 获取错误信息
		var errorMsg string
		if len(c.Errors) > 0 {
			errorMsg = c.Errors.String()
		}

		// 创建日志条目
		entry := &LogEntry{
			RequestID:    requestID,
			Timestamp:    startTime,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			Query:        c.Request.URL.RawQuery,
			ClientIP:     c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			UserID:       userID,
			StatusCode:   c.Writer.Status(),
			Latency:      latency,
			LatencyMs:    latency.Milliseconds(),
			RequestSize:  requestSize,
			ResponseSize: blw.size,
			Error:        errorMsg,
		}

		// 异步记录日志
		go logAsync(entry)

		// 控制台输出
		logToConsole(entry)
	}
}

// logAsync 异步记录日志到 Redis
func logAsync(entry *LogEntry) {
	// 将日志推送到 Redis 列表
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal log entry: %v", err)
		return
	}

	err = cache.RPush("logs:requests", string(data))
	if err != nil {
		log.Printf("Failed to push log to Redis: %v", err)
	}
}

// logToConsole 输出日志到控制台
func logToConsole(entry *LogEntry) {
	statusColor := getStatusColor(entry.StatusCode)
	methodColor := getMethodColor(entry.Method)

	log.Printf("[%s] %s%s\033[0m %s%d\033[0m %s %dms %s",
		entry.RequestID[:8],
		methodColor, entry.Method,
		statusColor, entry.StatusCode,
		entry.Path,
		entry.LatencyMs,
		entry.ClientIP,
	)
}

// getStatusColor 获取状态码颜色
func getStatusColor(code int) string {
	switch {
	case code >= 200 && code < 300:
		return "\033[32m" // 绿色
	case code >= 300 && code < 400:
		return "\033[36m" // 青色
	case code >= 400 && code < 500:
		return "\033[33m" // 黄色
	default:
		return "\033[31m" // 红色
	}
}

// getMethodColor 获取方法颜色
func getMethodColor(method string) string {
	switch method {
	case "GET":
		return "\033[34m" // 蓝色
	case "POST":
		return "\033[32m" // 绿色
	case "PUT":
		return "\033[33m" // 黄色
	case "DELETE":
		return "\033[31m" // 红色
	case "PATCH":
		return "\033[35m" // 紫色
	default:
		return "\033[0m" // 默认
	}
}

// GatewayLoggerMiddleware Gateway API 专用日志中间件
func GatewayLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 生成请求 ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// 记录开始时间
		startTime := time.Now()

		// 读取并保存请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 包装响应写入器
		blw := &responseWriter{
			ResponseWriter: c.Writer,
			body:           bytes.NewBufferString(""),
		}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算延迟
		latency := time.Since(startTime)

		// 获取用户和 Token 信息
		var userID, tokenKey string
		if uid, exists := c.Get(ContextKeyUserID); exists {
			userID = uid.(uuid.UUID).String()
		}
		if token, exists := c.Get(ContextKeyToken); exists {
			tokenKey = token.(*Token).Key
		}

		// 创建 Gateway 日志条目
		gatewayLog := &GatewayLogEntry{
			RequestID:    requestID,
			Timestamp:    startTime,
			Method:       c.Request.Method,
			Path:         c.Request.URL.Path,
			ClientIP:     c.ClientIP(),
			UserID:       userID,
			TokenKey:     maskTokenKey(tokenKey),
			StatusCode:   c.Writer.Status(),
			Latency:      latency,
			LatencyMs:    latency.Milliseconds(),
			RequestSize:  len(requestBody),
			ResponseSize: blw.size,
		}

		// 解析请求体获取模型信息
		if len(requestBody) > 0 {
			var reqData map[string]interface{}
			if err := json.Unmarshal(requestBody, &reqData); err == nil {
				if model, ok := reqData["model"].(string); ok {
					gatewayLog.Model = model
				}
				if stream, ok := reqData["stream"].(bool); ok {
					gatewayLog.Stream = stream
				}
			}
		}

		// 从上下文获取 Token 使用信息
		if promptTokens, exists := c.Get("prompt_tokens"); exists {
			gatewayLog.PromptTokens = promptTokens.(int)
		}
		if completionTokens, exists := c.Get("completion_tokens"); exists {
			gatewayLog.CompletionTokens = completionTokens.(int)
		}
		if channelID, exists := c.Get("channel_id"); exists {
			gatewayLog.ChannelID = channelID.(uint)
		}

		// 异步记录 Gateway 日志
		go logGatewayAsync(gatewayLog)

		// 控制台输出
		logGatewayToConsole(gatewayLog)
	}
}

// GatewayLogEntry Gateway 日志条目
type GatewayLogEntry struct {
	RequestID        string        `json:"request_id"`
	Timestamp        time.Time     `json:"timestamp"`
	Method           string        `json:"method"`
	Path             string        `json:"path"`
	ClientIP         string        `json:"client_ip"`
	UserID           string        `json:"user_id,omitempty"`
	TokenKey         string        `json:"token_key,omitempty"`
	Model            string        `json:"model,omitempty"`
	Stream           bool          `json:"stream"`
	ChannelID        uint          `json:"channel_id,omitempty"`
	StatusCode       int           `json:"status_code"`
	Latency          time.Duration `json:"latency"`
	LatencyMs        int64         `json:"latency_ms"`
	RequestSize      int           `json:"request_size"`
	ResponseSize     int           `json:"response_size"`
	PromptTokens     int           `json:"prompt_tokens,omitempty"`
	CompletionTokens int           `json:"completion_tokens,omitempty"`
}

// Token 临时定义，避免循环导入
type Token struct {
	Key string
}

// logGatewayAsync 异步记录 Gateway 日志
func logGatewayAsync(entry *GatewayLogEntry) {
	data, err := json.Marshal(entry)
	if err != nil {
		log.Printf("Failed to marshal gateway log entry: %v", err)
		return
	}

	err = cache.RPush("logs:gateway", string(data))
	if err != nil {
		log.Printf("Failed to push gateway log to Redis: %v", err)
	}
}

// logGatewayToConsole 输出 Gateway 日志到控制台
func logGatewayToConsole(entry *GatewayLogEntry) {
	statusColor := getStatusColor(entry.StatusCode)

	log.Printf("[Gateway] [%s] %s%d\033[0m %s model=%s tokens=%d/%d %dms",
		entry.RequestID[:8],
		statusColor, entry.StatusCode,
		entry.Path,
		entry.Model,
		entry.PromptTokens,
		entry.CompletionTokens,
		entry.LatencyMs,
	)
}

// maskTokenKey 遮蔽 Token Key
func maskTokenKey(key string) string {
	if len(key) <= 8 {
		return "****"
	}
	return key[:4] + "****" + key[len(key)-4:]
}

// RecoveryMiddleware 恢复中间件
func RecoveryMiddleware() gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		if err, ok := recovered.(string); ok {
			log.Printf("[Recovery] Panic recovered: %s", err)
		}

		c.JSON(500, gin.H{
			"success": false,
			"error": gin.H{
				"code":    "INTERNAL_ERROR",
				"message": "服务器内部错误",
			},
		})
	})
}
