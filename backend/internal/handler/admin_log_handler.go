package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminLogHandler 管理员日志管理处理器
type AdminLogHandler struct {
	logService      service.LogService
	auditLogService service.AuditLogService
}

func NewAdminLogHandler(logService service.LogService, auditLogService service.AuditLogService) *AdminLogHandler {
	return &AdminLogHandler{
		logService:      logService,
		auditLogService: auditLogService,
	}
}

// ListLogs 获取请求日志列表
func (h *AdminLogHandler) ListLogs(c *gin.Context) {
	var query model.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的查询参数"))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	filters := make(map[string]interface{})
	if userID := strings.TrimSpace(c.Query("user_id")); userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			filters["user_id"] = uid
		}
	}
	if channelID := strings.TrimSpace(c.Query("channel_id")); channelID != "" {
		if cid, err := strconv.ParseUint(channelID, 10, 32); err == nil {
			filters["channel_id"] = uint(cid)
		}
	}
	if modelName := strings.TrimSpace(c.Query("model")); modelName != "" {
		filters["model"] = modelName
	}
	if status := strings.TrimSpace(c.Query("status")); status != "" {
		filters["status"] = status
	}
	if startDate := strings.TrimSpace(c.Query("start_date")); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := strings.TrimSpace(c.Query("end_date")); endDate != "" {
		filters["end_date"] = endDate
	}

	logs, pagination, err := h.logService.AdminList(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(logs, pagination))
}

// GetLogStats 获取日志统计
func (h *AdminLogHandler) GetLogStats(c *gin.Context) {
	startDate := strings.TrimSpace(c.Query("start_date"))
	endDate := strings.TrimSpace(c.Query("end_date"))

	stats, err := h.logService.AdminGetStats(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// ListAuditLogs 获取审计日志
func (h *AdminLogHandler) ListAuditLogs(c *gin.Context) {
	var query model.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的查询参数"))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	filters := make(map[string]interface{})
	if actorID := strings.TrimSpace(c.Query("actor_id")); actorID != "" {
		if parsed, err := uuid.Parse(actorID); err == nil {
			filters["actor_id"] = parsed
		}
	}
	if action := strings.TrimSpace(c.Query("action")); action != "" {
		filters["action"] = action
	}
	if resource := strings.TrimSpace(c.Query("resource")); resource != "" {
		filters["resource"] = resource
	}
	if method := strings.TrimSpace(c.Query("method")); method != "" {
		filters["method"] = strings.ToUpper(method)
	}
	if statusCode := strings.TrimSpace(c.Query("status_code")); statusCode != "" {
		parsed, err := strconv.Atoi(statusCode)
		if err == nil && parsed > 0 {
			filters["status_code"] = parsed
		}
	}
	if successRaw := strings.TrimSpace(c.Query("success")); successRaw != "" {
		parsed, err := strconv.ParseBool(successRaw)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "success 仅支持 true/false"))
			return
		}
		filters["success"] = parsed
	}

	startTime, endTime, err := parseDateRangeFromQuery(c)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的日期范围"))
		return
	}
	filters["start_time"] = startTime
	filters["end_time"] = endTime

	logs, pagination, err := h.auditLogService.List(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(logs, pagination))
}

func (h *AdminLogHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/logs")
	{
		group.GET("", h.ListLogs)
		group.GET("/stats", h.GetLogStats)
		group.GET("/audit", h.ListAuditLogs)
	}
}
