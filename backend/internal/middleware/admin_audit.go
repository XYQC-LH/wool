package middleware

import (
	"bytes"
	"context"
	"io"
	"strings"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

// AdminAuditMiddleware 管理员审计日志中间件
func AdminAuditMiddleware(auditService service.AuditLogService) gin.HandlerFunc {
	return func(c *gin.Context) {
		if auditService == nil {
			c.Next()
			return
		}

		body := ""
		if shouldCaptureAuditBody(c) && c.Request.Body != nil {
			raw, err := io.ReadAll(c.Request.Body)
			if err == nil {
				c.Request.Body = io.NopCloser(bytes.NewBuffer(raw))
				body = truncateAuditText(string(raw), 4096)
			}
		}

		startAt := time.Now()
		c.Next()

		userID, hasUser := GetCurrentUserID(c)
		if !hasUser {
			return
		}
		roleValue, _ := c.Get(ContextKeyRole)
		role := ""
		if roleValue != nil {
			role = strings.TrimSpace(toString(roleValue))
		}

		routePath := strings.TrimSpace(c.FullPath())
		if routePath == "" {
			routePath = strings.TrimSpace(c.Request.URL.Path)
		}

		method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
		statusCode := c.Writer.Status()
		success := statusCode < 400
		errorMessage := strings.TrimSpace(c.Errors.String())

		action := method + " " + routePath
		resource := inferAuditResource(routePath)

		actorID := userID
		metadata := model.JSON{
			"latency_ms":  time.Since(startAt).Milliseconds(),
			"request_id":  c.GetString("request_id"),
			"raw_path":    c.Request.URL.Path,
			"status_code": statusCode,
		}

		_ = auditService.Record(context.Background(), &service.CreateAuditLogInput{
			ActorID:     &actorID,
			ActorRole:   role,
			Action:      action,
			Resource:    resource,
			Method:      method,
			Path:        routePath,
			StatusCode:  statusCode,
			Success:     success,
			RequestIP:   c.ClientIP(),
			UserAgent:   truncateAuditText(c.Request.UserAgent(), 1024),
			QueryParams: truncateAuditText(c.Request.URL.RawQuery, 2048),
			RequestBody: body,
			ErrorMsg:    truncateAuditText(errorMessage, 2048),
			Metadata:    metadata,
		})
	}
}

func inferAuditResource(path string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "unknown"
	}

	segments := strings.Split(strings.Trim(trimmed, "/"), "/")
	if len(segments) >= 3 && segments[0] == "api" && segments[1] == "admin" {
		return segments[2]
	}
	if len(segments) > 0 {
		return segments[len(segments)-1]
	}

	return "unknown"
}

func toString(v interface{}) string {
	switch typed := v.(type) {
	case string:
		return typed
	case model.UserRole:
		return string(typed)
	default:
		return ""
	}
}

func truncateAuditText(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	raw := strings.TrimSpace(text)
	if raw == "" {
		return ""
	}
	runes := []rune(raw)
	if len(runes) <= limit {
		return raw
	}
	return string(runes[:limit]) + "...(truncated)"
}

func shouldCaptureAuditBody(c *gin.Context) bool {
	if c == nil || c.Request == nil {
		return false
	}

	method := strings.ToUpper(strings.TrimSpace(c.Request.Method))
	switch method {
	case "POST", "PUT", "PATCH", "DELETE":
	default:
		return false
	}

	contentType := strings.ToLower(strings.TrimSpace(c.ContentType()))
	if strings.Contains(contentType, "multipart/form-data") {
		return false
	}
	if !(strings.Contains(contentType, "application/json") || strings.Contains(contentType, "application/x-www-form-urlencoded")) {
		return false
	}

	contentLength := c.Request.ContentLength
	if contentLength < 0 {
		return false
	}
	return contentLength <= 1<<20
}
