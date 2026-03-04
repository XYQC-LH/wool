package service

import (
	"fmt"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// LogService 日志服务接口
type LogService interface {
	// 用户端接口
	List(userID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]*model.LogResponse, *model.Pagination, error)
	GetStats(userID uuid.UUID, startDate, endDate string) (*UserStatsResponse, error)

	// 管理员接口
	AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AdminLogResponse, *model.Pagination, error)
	AdminGetStats(startDate, endDate string) (*AdminStatsResponse, error)
}

// UserStatsResponse 用户统计响应
type UserStatsResponse struct {
	TotalRequests int64                `json:"total_requests"`
	TotalTokens   int64                `json:"total_tokens"`
	TotalCost     decimal.Decimal      `json:"total_cost"`
	AvgLatencyMs  float64              `json:"avg_latency_ms"`
	DailyStats    []*DailyStatResponse `json:"daily_stats"`
	ModelStats    []*ModelStatResponse `json:"model_stats"`
}

// DailyStatResponse 每日统计响应
type DailyStatResponse struct {
	Date     string          `json:"date"`
	Requests int64           `json:"requests"`
	Tokens   int64           `json:"tokens"`
	Cost     decimal.Decimal `json:"cost"`
}

// ModelStatResponse 模型统计响应
type ModelStatResponse struct {
	Model    string          `json:"model"`
	Requests int64           `json:"requests"`
	Tokens   int64           `json:"tokens"`
	Cost     decimal.Decimal `json:"cost"`
}

// AdminStatsResponse 管理员统计响应
type AdminStatsResponse struct {
	Summary      *StatsSummary          `json:"summary"`
	DailyStats   []*DailyStatResponse   `json:"daily_stats"`
	ModelStats   []*ModelStatResponse   `json:"model_stats"`
	ChannelStats []*ChannelStatResponse `json:"channel_stats"`
}

// StatsSummary 统计摘要
type StatsSummary struct {
	TotalRequests   int64           `json:"total_requests"`
	SuccessRequests int64           `json:"success_requests"`
	FailedRequests  int64           `json:"failed_requests"`
	TotalTokens     int64           `json:"total_tokens"`
	TotalCost       decimal.Decimal `json:"total_cost"`
	UpstreamCost    decimal.Decimal `json:"upstream_cost"`
	Profit          decimal.Decimal `json:"profit"`
	AvgLatency      float64         `json:"avg_latency"`
	P95Latency      float64         `json:"p95_latency"`
	P99Latency      float64         `json:"p99_latency"`
}

// ChannelStatResponse 渠道统计响应
type ChannelStatResponse struct {
	ChannelID    uint            `json:"channel_id"`
	ChannelName  string          `json:"channel_name"`
	RequestCount int64           `json:"request_count"`
	SuccessCount int64           `json:"success_count"`
	FailCount    int64           `json:"fail_count"`
	SuccessRate  float64         `json:"success_rate"`
	AvgLatency   float64         `json:"avg_latency"`
	TotalCost    decimal.Decimal `json:"total_cost"`
}

// logService 日志服务实现
type logService struct {
	logRepo repository.LogRepository
}

// NewLogService 创建日志服务
func NewLogService(logRepo repository.LogRepository) LogService {
	return &logService{
		logRepo: logRepo,
	}
}

// parseDate 解析日期字符串
func parseDate(dateStr string) (time.Time, error) {
	if dateStr == "" {
		return time.Time{}, nil
	}
	return time.Parse("2006-01-02", dateStr)
}

// getDefaultDateRange 获取默认日期范围（最近30天）
func getDefaultDateRange() (time.Time, time.Time) {
	endDate := time.Now()
	startDate := endDate.AddDate(0, 0, -30)
	return startDate, endDate
}

// List 获取用户日志列表
func (s *logService) List(userID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]*model.LogResponse, *model.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	logs, total, err := s.logRepo.ListByUserID(userID, page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("获取日志列表失败: %w", err)
	}

	// 转换为响应格式
	responses := make([]*model.LogResponse, len(logs))
	for i, log := range logs {
		responses[i] = log.ToResponse()
	}

	pagination := model.NewPagination(page, pageSize, total)

	return responses, pagination, nil
}

// GetStats 获取用户统计数据
func (s *logService) GetStats(userID uuid.UUID, startDateStr, endDateStr string) (*UserStatsResponse, error) {
	// 解析日期
	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = parseDate(startDateStr)
		if err != nil {
			return nil, fmt.Errorf("无效的开始日期格式: %w", err)
		}
	}

	if endDateStr != "" {
		endDate, err = parseDate(endDateStr)
		if err != nil {
			return nil, fmt.Errorf("无效的结束日期格式: %w", err)
		}
		// 设置为当天结束
		endDate = endDate.Add(24*time.Hour - time.Second)
	}

	// 如果没有指定日期，使用默认范围
	if startDate.IsZero() || endDate.IsZero() {
		startDate, endDate = getDefaultDateRange()
	}

	// 获取用户统计
	userStats, err := s.logRepo.GetUserStats(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取用户统计失败: %w", err)
	}

	// 获取每日统计（按用户过滤）
	dailyStats, err := s.logRepo.GetUserDailyStats(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取每日统计失败: %w", err)
	}

	// 获取模型统计（按用户过滤）
	modelStats, err := s.logRepo.GetUserModelStats(userID, startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取模型统计失败: %w", err)
	}

	// 转换每日统计
	dailyResponses := make([]*DailyStatResponse, len(dailyStats))
	for i, stat := range dailyStats {
		dailyResponses[i] = &DailyStatResponse{
			Date:     stat.Date,
			Requests: stat.RequestCount,
			Tokens:   stat.TotalTokens,
			Cost:     decimal.NewFromFloat(stat.Cost),
		}
	}

	// 转换模型统计
	modelResponses := make([]*ModelStatResponse, len(modelStats))
	for i, stat := range modelStats {
		modelResponses[i] = &ModelStatResponse{
			Model:    stat.Model,
			Requests: stat.RequestCount,
			Tokens:   stat.TotalTokens,
			Cost:     decimal.NewFromFloat(stat.Cost),
		}
	}

	return &UserStatsResponse{
		TotalRequests: userStats.TotalRequests,
		TotalTokens:   userStats.TotalTokens,
		TotalCost:     userStats.TotalCost,
		AvgLatencyMs:  userStats.AvgLatencyMs,
		DailyStats:    dailyResponses,
		ModelStats:    modelResponses,
	}, nil
}

// AdminList 管理员获取日志列表
func (s *logService) AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AdminLogResponse, *model.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	logs, total, err := s.logRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("获取日志列表失败: %w", err)
	}

	// 转换为管理员响应格式
	responses := make([]*model.AdminLogResponse, len(logs))
	for i, log := range logs {
		responses[i] = log.ToAdminResponse()
	}

	pagination := model.NewPagination(page, pageSize, total)

	return responses, pagination, nil
}

// AdminGetStats 管理员获取统计数据
func (s *logService) AdminGetStats(startDateStr, endDateStr string) (*AdminStatsResponse, error) {
	// 解析日期
	var startDate, endDate time.Time
	var err error

	if startDateStr != "" {
		startDate, err = parseDate(startDateStr)
		if err != nil {
			return nil, fmt.Errorf("无效的开始日期格式: %w", err)
		}
	}

	if endDateStr != "" {
		endDate, err = parseDate(endDateStr)
		if err != nil {
			return nil, fmt.Errorf("无效的结束日期格式: %w", err)
		}
		// 设置为当天结束
		endDate = endDate.Add(24*time.Hour - time.Second)
	}

	// 如果没有指定日期，使用默认范围
	if startDate.IsZero() || endDate.IsZero() {
		startDate, endDate = getDefaultDateRange()
	}

	// 获取总体统计
	totalStats, err := s.logRepo.GetTotalStats(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取总体统计失败: %w", err)
	}

	// 获取每日统计
	dailyStats, err := s.logRepo.GetDailyStats(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取每日统计失败: %w", err)
	}

	// 获取模型统计
	modelStats, err := s.logRepo.GetModelStats(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取模型统计失败: %w", err)
	}

	// 获取渠道统计
	channelStats, err := s.logRepo.GetChannelStats(startDate, endDate)
	if err != nil {
		return nil, fmt.Errorf("获取渠道统计失败: %w", err)
	}

	// 转换每日统计
	dailyResponses := make([]*DailyStatResponse, len(dailyStats))
	for i, stat := range dailyStats {
		dailyResponses[i] = &DailyStatResponse{
			Date:     stat.Date,
			Requests: stat.RequestCount,
			Tokens:   stat.TotalTokens,
			Cost:     decimal.NewFromFloat(stat.Cost),
		}
	}

	// 转换模型统计
	modelResponses := make([]*ModelStatResponse, len(modelStats))
	for i, stat := range modelStats {
		modelResponses[i] = &ModelStatResponse{
			Model:    stat.Model,
			Requests: stat.RequestCount,
			Tokens:   stat.TotalTokens,
			Cost:     decimal.NewFromFloat(stat.Cost),
		}
	}

	// 转换渠道统计
	channelResponses := make([]*ChannelStatResponse, len(channelStats))
	for i, stat := range channelStats {
		channelResponses[i] = &ChannelStatResponse{
			ChannelID:    stat.ChannelID,
			RequestCount: stat.RequestCount,
			SuccessCount: stat.SuccessCount,
			FailCount:    stat.FailCount,
			SuccessRate:  stat.SuccessRate,
			AvgLatency:   stat.AvgLatency,
			TotalCost:    decimal.NewFromFloat(stat.TotalCost),
		}
	}

	return &AdminStatsResponse{
		Summary: &StatsSummary{
			TotalRequests:   totalStats.TotalRequests,
			SuccessRequests: totalStats.SuccessRequests,
			FailedRequests:  totalStats.FailedRequests,
			TotalTokens:     totalStats.TotalTokens,
			TotalCost:       totalStats.TotalCost,
			UpstreamCost:    totalStats.UpstreamCost,
			Profit:          totalStats.Profit,
			AvgLatency:      totalStats.AvgLatency,
			P95Latency:      totalStats.P95Latency,
			P99Latency:      totalStats.P99Latency,
		},
		DailyStats:   dailyResponses,
		ModelStats:   modelResponses,
		ChannelStats: channelResponses,
	}, nil
}
