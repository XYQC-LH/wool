package repository

import (
	"time"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// LogRepository Log 仓库接口
type LogRepository interface {
	Create(log *model.Log) error
	GetByID(id uuid.UUID) (*model.Log, error)
	ListByUserID(userID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]*model.Log, int64, error)
	ListByTokenID(tokenID uuid.UUID, page, pageSize int) ([]*model.Log, int64, error)
	List(page, pageSize int, filters map[string]interface{}) ([]*model.Log, int64, error)
	GetUserStats(userID uuid.UUID, startDate, endDate time.Time) (*UserLogStats, error)
	GetTokenUsageStatsByTokenIDs(userID uuid.UUID, tokenIDs []uuid.UUID) ([]*model.TokenUsageStats, error)
	GetUserDailyStats(userID uuid.UUID, startDate, endDate time.Time) ([]*model.UsageStats, error)
	GetUserModelStats(userID uuid.UUID, startDate, endDate time.Time) ([]*model.ModelUsageStats, error)
	GetDailyStats(startDate, endDate time.Time) ([]*model.UsageStats, error)
	GetModelStats(startDate, endDate time.Time) ([]*model.ModelUsageStats, error)
	GetChannelStats(startDate, endDate time.Time) ([]*model.ChannelStats, error)
	GetTotalStats(startDate, endDate time.Time) (*TotalLogStats, error)
}

// logRepository Log 仓库实现
type logRepository struct {
	db *gorm.DB
}

// NewLogRepository 创建 Log 仓库
func NewLogRepository(db *gorm.DB) LogRepository {
	return &logRepository{db: db}
}

// UserLogStats 用户日志统计
type UserLogStats struct {
	TotalRequests    int64           `json:"total_requests"`
	PromptTokens     int64           `json:"prompt_tokens"`
	CompletionTokens int64           `json:"completion_tokens"`
	TotalTokens      int64           `json:"total_tokens"`
	TotalCost        decimal.Decimal `json:"total_cost"`
	AvgLatencyMs     float64         `json:"avg_latency_ms"`
}

// TotalLogStats 总体日志统计
type TotalLogStats struct {
	TotalRequests    int64           `json:"total_requests"`
	SuccessRequests  int64           `json:"success_requests"`
	FailedRequests   int64           `json:"failed_requests"`
	PromptTokens     int64           `json:"prompt_tokens"`
	CompletionTokens int64           `json:"completion_tokens"`
	TotalTokens      int64           `json:"total_tokens"`
	TotalCost        decimal.Decimal `json:"total_cost"`
	UpstreamCost     decimal.Decimal `json:"upstream_cost"`
	Profit           decimal.Decimal `json:"profit"`
	AvgLatency       float64         `json:"avg_latency"`
	P95Latency       float64         `json:"p95_latency"`
	P99Latency       float64         `json:"p99_latency"`
}

// Create 创建日志
func (r *logRepository) Create(log *model.Log) error {
	return r.db.Create(log).Error
}

// GetByID 根据 ID 获取日志
func (r *logRepository) GetByID(id uuid.UUID) (*model.Log, error) {
	var log model.Log
	if err := r.db.Where("id = ?", id).First(&log).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, err
	}
	return &log, nil
}

// ListByUserID 获取用户的日志列表
func (r *logRepository) ListByUserID(userID uuid.UUID, page, pageSize int, filters map[string]interface{}) ([]*model.Log, int64, error) {
	var logs []*model.Log
	var total int64

	query := r.db.Model(&model.Log{}).Where("user_id = ?", userID)

	// 应用过滤条件
	if model, ok := filters["model"]; ok && model != "" {
		query = query.Where("model = ?", model)
	}
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if startDate, ok := filters["start_date"]; ok {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := filters["end_date"]; ok {
		query = query.Where("created_at <= ?", endDate)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// ListByTokenID 获取 Token 的日志列表
func (r *logRepository) ListByTokenID(tokenID uuid.UUID, page, pageSize int) ([]*model.Log, int64, error) {
	var logs []*model.Log
	var total int64

	query := r.db.Model(&model.Log{}).Where("token_id = ?", tokenID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// List 获取日志列表（管理员）
func (r *logRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.Log, int64, error) {
	var logs []*model.Log
	var total int64

	query := r.db.Model(&model.Log{})

	// 应用过滤条件
	if userID, ok := filters["user_id"]; ok {
		query = query.Where("user_id = ?", userID)
	}
	if channelID, ok := filters["channel_id"]; ok {
		query = query.Where("channel_id = ?", channelID)
	}
	if modelName, ok := filters["model"]; ok && modelName != "" {
		query = query.Where("model = ?", modelName)
	}
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if startDate, ok := filters["start_date"]; ok {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := filters["end_date"]; ok {
		query = query.Where("created_at <= ?", endDate)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("User").Preload("Channel").
		Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

// GetUserStats 获取用户统计
func (r *logRepository) GetUserStats(userID uuid.UUID, startDate, endDate time.Time) (*UserLogStats, error) {
	var stats UserLogStats

	err := r.db.Model(&model.Log{}).
		Select("COUNT(*) as total_requests, "+
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, "+
			"COALESCE(SUM(completion_tokens), 0) as completion_tokens, "+
			"COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(total_cost), 0) as total_cost, "+
			"COALESCE(AVG(duration), 0) as avg_latency_ms").
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startDate, endDate).
		Scan(&stats).Error

	return &stats, err
}

// GetTokenUsageStatsByTokenIDs 获取指定 Token 列表的用量统计（按用户隔离）
func (r *logRepository) GetTokenUsageStatsByTokenIDs(userID uuid.UUID, tokenIDs []uuid.UUID) ([]*model.TokenUsageStats, error) {
	if len(tokenIDs) == 0 {
		return []*model.TokenUsageStats{}, nil
	}

	var stats []*model.TokenUsageStats
	err := r.db.Model(&model.Log{}).
		Select("token_id, "+
			"COUNT(*) as request_count, "+
			"COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(total_cost), 0) as total_cost").
		Where("user_id = ? AND token_id IN ?", userID, tokenIDs).
		Group("token_id").
		Scan(&stats).Error

	return stats, err
}

// GetUserDailyStats 获取用户每日统计
func (r *logRepository) GetUserDailyStats(userID uuid.UUID, startDate, endDate time.Time) ([]*model.UsageStats, error) {
	var stats []*model.UsageStats

	err := r.db.Model(&model.Log{}).
		Select("DATE(created_at) as date, "+
			"COUNT(*) as request_count, "+
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, "+
			"COALESCE(SUM(completion_tokens), 0) as completion_tokens, "+
			"COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(total_cost), 0) as cost").
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&stats).Error

	return stats, err
}

// GetUserModelStats 获取用户模型统计
func (r *logRepository) GetUserModelStats(userID uuid.UUID, startDate, endDate time.Time) ([]*model.ModelUsageStats, error) {
	var stats []*model.ModelUsageStats

	err := r.db.Model(&model.Log{}).
		Select("model, "+
			"COUNT(*) as request_count, "+
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, "+
			"COALESCE(SUM(completion_tokens), 0) as completion_tokens, "+
			"COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(total_cost), 0) as cost").
		Where("user_id = ? AND created_at >= ? AND created_at <= ?", userID, startDate, endDate).
		Group("model").
		Order("request_count DESC").
		Scan(&stats).Error

	return stats, err
}

// GetDailyStats 获取每日统计
func (r *logRepository) GetDailyStats(startDate, endDate time.Time) ([]*model.UsageStats, error) {
	var stats []*model.UsageStats

	err := r.db.Model(&model.Log{}).
		Select("DATE(created_at) as date, "+
			"COUNT(*) as request_count, "+
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, "+
			"COALESCE(SUM(completion_tokens), 0) as completion_tokens, "+
			"COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(total_cost), 0) as cost").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("DATE(created_at)").
		Order("date ASC").
		Scan(&stats).Error

	return stats, err
}

// GetModelStats 获取模型统计
func (r *logRepository) GetModelStats(startDate, endDate time.Time) ([]*model.ModelUsageStats, error) {
	var stats []*model.ModelUsageStats

	err := r.db.Model(&model.Log{}).
		Select("model, "+
			"COUNT(*) as request_count, "+
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, "+
			"COALESCE(SUM(completion_tokens), 0) as completion_tokens, "+
			"COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(total_cost), 0) as cost").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("model").
		Order("request_count DESC").
		Scan(&stats).Error

	return stats, err
}

// GetChannelStats 获取渠道统计
func (r *logRepository) GetChannelStats(startDate, endDate time.Time) ([]*model.ChannelStats, error) {
	var stats []*model.ChannelStats

	err := r.db.Model(&model.Log{}).
		Select("channel_id, "+
			"COUNT(*) as request_count, "+
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_count, "+
			"SUM(CASE WHEN status IN ('error','failed') THEN 1 ELSE 0 END) as fail_count, "+
			"COALESCE(AVG(duration), 0) as avg_latency, "+
			"COALESCE(SUM(upstream_cost), 0) as total_cost").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Group("channel_id").
		Order("request_count DESC").
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	// 计算成功率
	for _, s := range stats {
		if s.RequestCount > 0 {
			s.SuccessRate = float64(s.SuccessCount) / float64(s.RequestCount) * 100
		}
	}

	return stats, nil
}

// GetTotalStats 获取总体统计
func (r *logRepository) GetTotalStats(startDate, endDate time.Time) (*TotalLogStats, error) {
	var stats TotalLogStats

	err := r.db.Model(&model.Log{}).
		Select("COUNT(*) as total_requests, "+
			"SUM(CASE WHEN status = 'success' THEN 1 ELSE 0 END) as success_requests, "+
			"SUM(CASE WHEN status IN ('error','failed') THEN 1 ELSE 0 END) as failed_requests, "+
			"COALESCE(SUM(prompt_tokens), 0) as prompt_tokens, "+
			"COALESCE(SUM(completion_tokens), 0) as completion_tokens, "+
			"COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens, "+
			"COALESCE(SUM(total_cost), 0) as total_cost, "+
			"COALESCE(SUM(upstream_cost), 0) as upstream_cost, "+
			"COALESCE(AVG(duration), 0) as avg_latency, "+
			"COALESCE(PERCENTILE_CONT(0.95) WITHIN GROUP (ORDER BY duration), 0) as p95_latency, "+
			"COALESCE(PERCENTILE_CONT(0.99) WITHIN GROUP (ORDER BY duration), 0) as p99_latency").
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Scan(&stats).Error

	if err != nil {
		return nil, err
	}

	// 计算利润
	stats.Profit = stats.TotalCost.Sub(stats.UpstreamCost)

	return &stats, nil
}
