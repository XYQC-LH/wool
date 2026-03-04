package model

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// JSON 自定义 JSON 类型，用于 GORM 的 JSONB 字段
type JSON map[string]interface{}

// Value 实现 driver.Valuer 接口
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// JSONArray 自定义 JSON 数组类型
type JSONArray []interface{}

// Value 实现 driver.Valuer 接口
func (j JSONArray) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan 实现 sql.Scanner 接口
func (j *JSONArray) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}

// StringArray 字符串数组类型
type StringArray []string

// Value 实现 driver.Valuer 接口
func (s StringArray) Value() (driver.Value, error) {
	if s == nil {
		return nil, nil
	}
	return json.Marshal(s)
}

// Scan 实现 sql.Scanner 接口
func (s *StringArray) Scan(value interface{}) error {
	if value == nil {
		*s = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, s)
}

// Response 通用 API 响应结构（兼容前端 code/message/data 格式）
type Response struct {
	Code    int         `json:"code"`    // 0 表示成功，非 0 表示错误
	Message string      `json:"message"` // 响应消息
	Data    interface{} `json:"data,omitempty"`
	// 保留旧字段以向后兼容
	Success bool       `json:"success,omitempty"`
	Error   *ErrorInfo `json:"error,omitempty"`
}

// ErrorInfo 错误信息结构（保留用于向后兼容）
type ErrorInfo struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// PaginatedData 分页数据结构（前端期望的格式）
type PaginatedData struct {
	List       interface{} `json:"list"`
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
}

// PaginatedResponse 分页响应结构（使用标准 Response 格式）
type PaginatedResponse = Response

// Pagination 分页信息（内部使用）
type Pagination struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// NewPagination 创建分页信息
func NewPagination(page, pageSize int, total int64) *Pagination {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	return &Pagination{
		Page:       page,
		PageSize:   pageSize,
		Total:      total,
		TotalPages: totalPages,
	}
}

// SuccessResponse 成功响应
func SuccessResponse(data interface{}) *Response {
	return &Response{
		Code:    0,
		Message: "success",
		Data:    data,
		Success: true, // 向后兼容
	}
}

// SuccessResponseWithMessage 成功响应（带自定义消息）
func SuccessResponseWithMessage(message string, data interface{}) *Response {
	return &Response{
		Code:    0,
		Message: message,
		Data:    data,
		Success: true,
	}
}

// ErrorResponse 错误响应
func ErrorResponse(code, message string) *Response {
	// 将字符串错误码转换为数字
	var codeNum int
	switch code {
	case ErrCodeInvalidRequest:
		codeNum = 400
	case ErrCodeUnauthorized:
		codeNum = 401
	case ErrCodeForbidden:
		codeNum = 403
	case ErrCodeNotFound:
		codeNum = 404
	case ErrCodeConflict:
		codeNum = 409
	case ErrCodeInternalError:
		codeNum = 500
	case ErrCodeRateLimited:
		codeNum = 429
	case ErrCodeInsufficientFund:
		codeNum = 402
	case ErrCodeQuotaExceeded:
		codeNum = 429
	case ErrCodeChannelError:
		codeNum = 502
	case ErrCodeModelNotFound:
		codeNum = 404
	default:
		codeNum = 500
	}

	return &Response{
		Code:    codeNum,
		Message: message,
		Success: false,
		Error: &ErrorInfo{
			Code:    code,
			Message: message,
		},
	}
}

// ErrorResponseWithCode 错误响应（带自定义数字错误码）
func ErrorResponseWithCode(codeNum int, message string) *Response {
	return &Response{
		Code:    codeNum,
		Message: message,
		Success: false,
		Error: &ErrorInfo{
			Code:    fmt.Sprintf("%d", codeNum),
			Message: message,
		},
	}
}

// PaginatedSuccessResponse 分页成功响应（返回前端期望的格式）
func PaginatedSuccessResponse(data interface{}, pagination *Pagination) *Response {
	paginatedData := &PaginatedData{
		List:       data,
		Total:      pagination.Total,
		Page:       pagination.Page,
		PageSize:   pagination.PageSize,
		TotalPages: pagination.TotalPages,
	}
	return &Response{
		Code:    0,
		Message: "success",
		Data:    paginatedData,
		Success: true,
	}
}

// PaginationQuery 分页查询参数
type PaginationQuery struct {
	Page     int    `form:"page" binding:"omitempty,min=1"`
	PageSize int    `form:"page_size" binding:"omitempty,min=1,max=100"`
	SortBy   string `form:"sort_by" binding:"omitempty"`
	SortDesc bool   `form:"sort_desc" binding:"omitempty"`
}

// GetOffset 获取偏移量
func (p *PaginationQuery) GetOffset() int {
	if p.Page <= 0 {
		p.Page = 1
	}
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	return (p.Page - 1) * p.PageSize
}

// GetLimit 获取限制数量
func (p *PaginationQuery) GetLimit() int {
	if p.PageSize <= 0 {
		p.PageSize = 20
	}
	if p.PageSize > 100 {
		p.PageSize = 100
	}
	return p.PageSize
}

// DateRange 日期范围查询
type DateRange struct {
	StartDate *time.Time `form:"start_date" binding:"omitempty"`
	EndDate   *time.Time `form:"end_date" binding:"omitempty"`
}

// OpenAI 兼容的错误响应格式
type OpenAIError struct {
	Error OpenAIErrorDetail `json:"error"`
}

// OpenAIErrorDetail OpenAI 错误详情
type OpenAIErrorDetail struct {
	Message string  `json:"message"`
	Type    string  `json:"type"`
	Param   *string `json:"param,omitempty"`
	Code    *string `json:"code,omitempty"`
}

// NewOpenAIError 创建 OpenAI 格式的错误
func NewOpenAIError(message, errType string, code *string) *OpenAIError {
	return &OpenAIError{
		Error: OpenAIErrorDetail{
			Message: message,
			Type:    errType,
			Code:    code,
		},
	}
}

// OpenAI 错误类型常量
const (
	OpenAIErrorTypeInvalidRequest     = "invalid_request_error"
	OpenAIErrorTypeAuthentication     = "authentication_error"
	OpenAIErrorTypePermission         = "permission_error"
	OpenAIErrorTypeNotFound           = "not_found_error"
	OpenAIErrorTypeRateLimit          = "rate_limit_error"
	OpenAIErrorTypeServer             = "server_error"
	OpenAIErrorTypeServiceUnavailable = "service_unavailable"
)

// 业务错误码常量
const (
	ErrCodeInvalidRequest   = "INVALID_REQUEST"
	ErrCodeUnauthorized     = "UNAUTHORIZED"
	ErrCodeForbidden        = "FORBIDDEN"
	ErrCodeNotFound         = "NOT_FOUND"
	ErrCodeConflict         = "CONFLICT"
	ErrCodeInternalError    = "INTERNAL_ERROR"
	ErrCodeRateLimited      = "RATE_LIMITED"
	ErrCodeInsufficientFund = "INSUFFICIENT_FUND"
	ErrCodeQuotaExceeded    = "QUOTA_EXCEEDED"
	ErrCodeChannelError     = "CHANNEL_ERROR"
	ErrCodeModelNotFound    = "MODEL_NOT_FOUND"
)

// DashboardStats 仪表盘统计数据
type DashboardStats struct {
	TotalUsers        int64   `json:"total_users"`
	ActiveUsers       int64   `json:"active_users"`
	TotalRequests     int64   `json:"total_requests"`
	TotalTokens       int64   `json:"total_tokens"`
	TotalRevenue      float64 `json:"total_revenue"`
	TotalCost         float64 `json:"total_cost"`
	Profit            float64 `json:"profit"`
	HealthyChannels   int64   `json:"healthy_channels"`
	UnhealthyChannels int64   `json:"unhealthy_channels"`
}

// UsageStats 使用统计
type UsageStats struct {
	Date             string  `json:"date"`
	RequestCount     int64   `json:"request_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

// ModelUsageStats 模型使用统计
type ModelUsageStats struct {
	Model            string  `json:"model"`
	RequestCount     int64   `json:"request_count"`
	PromptTokens     int64   `json:"prompt_tokens"`
	CompletionTokens int64   `json:"completion_tokens"`
	TotalTokens      int64   `json:"total_tokens"`
	Cost             float64 `json:"cost"`
}

// ChannelStats 渠道统计
type ChannelStats struct {
	ChannelID    uint    `json:"channel_id"`
	ChannelName  string  `json:"channel_name"`
	RequestCount int64   `json:"request_count"`
	SuccessCount int64   `json:"success_count"`
	FailCount    int64   `json:"fail_count"`
	SuccessRate  float64 `json:"success_rate"`
	AvgLatency   float64 `json:"avg_latency"`
	TotalCost    float64 `json:"total_cost"`
}

// BillingOverview 账单概览
type BillingOverview struct {
	Balance       decimal.Decimal `json:"balance"`
	TodayCost     decimal.Decimal `json:"today_cost"`
	MonthCost     decimal.Decimal `json:"month_cost"`
	TotalRecharge decimal.Decimal `json:"total_recharge"`
	TodayRequests int64           `json:"today_requests"`
	MonthRequests int64           `json:"month_requests"`
}

// ConsumptionDetail 消费明细
type ConsumptionDetail struct {
	ID               uuid.UUID       `json:"id"`
	Model            string          `json:"model"`
	PromptTokens     int64           `json:"prompt_tokens"`
	CompletionTokens int64           `json:"completion_tokens"`
	TotalTokens      int64           `json:"total_tokens"`
	Cost             decimal.Decimal `json:"cost"`
	Status           string          `json:"status"`
	CreatedAt        time.Time       `json:"created_at"`
}

// SystemMonitor 系统监控数据
type SystemMonitor struct {
	CPUPercent       float64 `json:"cpu_percent"`
	MemoryPercent    float64 `json:"memory_percent"`
	RedisConnections int     `json:"redis_connections"`
	DBConnections    int     `json:"db_connections"`
}

// DashboardAlert 仪表板告警摘要（用于管理端 dashboard 展示）
type DashboardAlert struct {
	ID        string `json:"id"`
	Message   string `json:"message"`
	Level     string `json:"level"` // info, warning, error, critical
	CreatedAt string `json:"created_at"`
}
