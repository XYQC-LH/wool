package service

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

var ErrInvalidBillingGroupBy = errors.New("无效的 group_by")

// BillingService 管理员计费服务
type BillingService interface {
	GetStatistics(ctx context.Context, startDate, endDate time.Time) (*BillingStatistics, error)
	GetUsage(ctx context.Context, query *BillingUsageQuery) ([]*BillingUsageItem, error)
	GetCostAnalysis(ctx context.Context, query *BillingCostAnalysisQuery) (*BillingCostAnalysis, error)
}

// BillingStatistics 计费统计摘要
type BillingStatistics struct {
	StartDate            time.Time `json:"start_date"`
	EndDate              time.Time `json:"end_date"`
	TotalRequests        int64     `json:"total_requests"`
	TotalTokens          int64     `json:"total_tokens"`
	TotalRevenue         float64   `json:"total_revenue"`
	TotalCost            float64   `json:"total_cost"`
	TotalProfit          float64   `json:"total_profit"`
	GrossMargin          float64   `json:"gross_margin"`
	ActiveUsers          int64     `json:"active_users"`
	AvgRevenuePerRequest float64   `json:"avg_revenue_per_request"`
	AvgCostPerRequest    float64   `json:"avg_cost_per_request"`
}

// BillingUsageQuery 使用量统计查询
type BillingUsageQuery struct {
	StartDate time.Time
	EndDate   time.Time
	GroupBy   string
	Limit     int
}

// BillingUsageItem 使用量统计项
type BillingUsageItem struct {
	Dimension    string  `json:"dimension"`
	RequestCount int64   `json:"request_count"`
	TotalTokens  int64   `json:"total_tokens"`
	Revenue      float64 `json:"revenue"`
	Cost         float64 `json:"cost"`
	Profit       float64 `json:"profit"`
	Margin       float64 `json:"margin"`
}

// BillingCostAnalysisQuery 成本分析查询
type BillingCostAnalysisQuery struct {
	StartDate time.Time
	EndDate   time.Time
	GroupBy   string
	Limit     int
}

// BillingCostBreakdown 成本分析明细
type BillingCostBreakdown struct {
	Dimension             string  `json:"dimension"`
	RequestCount          int64   `json:"request_count"`
	TotalTokens           int64   `json:"total_tokens"`
	Revenue               float64 `json:"revenue"`
	Cost                  float64 `json:"cost"`
	Profit                float64 `json:"profit"`
	Margin                float64 `json:"margin"`
	AvgRevenuePerRequest  float64 `json:"avg_revenue_per_request"`
	AvgCostPerRequest     float64 `json:"avg_cost_per_request"`
	AvgProfitPerRequest   float64 `json:"avg_profit_per_request"`
}

// BillingCostAnalysis 成本分析响应
type BillingCostAnalysis struct {
	Summary   *BillingStatistics     `json:"summary"`
	GroupBy   string                 `json:"group_by"`
	Breakdown []*BillingCostBreakdown `json:"breakdown"`
}

type billingStore interface {
	QueryStatistics(ctx context.Context, startDate, endDate time.Time) (*billingSummaryRow, error)
	QueryUsage(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error)
	QueryCostBreakdown(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error)
}

type billingService struct {
	store billingStore
}

type gormBillingStore struct {
	db *gorm.DB
}

type billingSummaryRow struct {
	TotalRequests int64   `gorm:"column:total_requests"`
	TotalTokens   int64   `gorm:"column:total_tokens"`
	TotalRevenue  float64 `gorm:"column:total_revenue"`
	TotalCost     float64 `gorm:"column:total_cost"`
	ActiveUsers   int64   `gorm:"column:active_users"`
}

type billingUsageRow struct {
	Dimension    string  `gorm:"column:dimension"`
	RequestCount int64   `gorm:"column:request_count"`
	TotalTokens  int64   `gorm:"column:total_tokens"`
	TotalRevenue float64 `gorm:"column:total_revenue"`
	TotalCost    float64 `gorm:"column:total_cost"`
}

func NewBillingService(db *gorm.DB) BillingService {
	return newBillingServiceWithStore(newGormBillingStore(db))
}

func newBillingServiceWithStore(store billingStore) *billingService {
	return &billingService{store: store}
}

func newGormBillingStore(db *gorm.DB) billingStore {
	return &gormBillingStore{db: db}
}

func (s *billingService) GetStatistics(ctx context.Context, startDate, endDate time.Time) (*BillingStatistics, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("billing service 未初始化")
	}

	startDate, endDate, err := normalizeBillingWindow(startDate, endDate)
	if err != nil {
		return nil, err
	}

	row, err := s.store.QueryStatistics(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}
	if row == nil {
		row = &billingSummaryRow{}
	}

	totalProfit := row.TotalRevenue - row.TotalCost
	stats := &BillingStatistics{
		StartDate:     startDate,
		EndDate:       endDate,
		TotalRequests: row.TotalRequests,
		TotalTokens:   row.TotalTokens,
		TotalRevenue:  row.TotalRevenue,
		TotalCost:     row.TotalCost,
		TotalProfit:   totalProfit,
		GrossMargin:   calculateMargin(totalProfit, row.TotalRevenue),
		ActiveUsers:   row.ActiveUsers,
	}

	if row.TotalRequests > 0 {
		requestCount := float64(row.TotalRequests)
		stats.AvgRevenuePerRequest = row.TotalRevenue / requestCount
		stats.AvgCostPerRequest = row.TotalCost / requestCount
	}

	return stats, nil
}

func (s *billingService) GetUsage(ctx context.Context, query *BillingUsageQuery) ([]*BillingUsageItem, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("billing service 未初始化")
	}
	if query == nil {
		query = &BillingUsageQuery{}
	}

	startDate, endDate, err := normalizeBillingWindow(query.StartDate, query.EndDate)
	if err != nil {
		return nil, err
	}
	groupBy, err := normalizeUsageGroupBy(query.GroupBy)
	if err != nil {
		return nil, err
	}
	limit := normalizeBillingLimit(query.Limit)

	rows, err := s.store.QueryUsage(ctx, startDate, endDate, groupBy, limit)
	if err != nil {
		return nil, err
	}

	result := make([]*BillingUsageItem, 0, len(rows))
	for _, row := range rows {
		profit := row.TotalRevenue - row.TotalCost
		result = append(result, &BillingUsageItem{
			Dimension:    row.Dimension,
			RequestCount: row.RequestCount,
			TotalTokens:  row.TotalTokens,
			Revenue:      row.TotalRevenue,
			Cost:         row.TotalCost,
			Profit:       profit,
			Margin:       calculateMargin(profit, row.TotalRevenue),
		})
	}

	return result, nil
}

func (s *billingService) GetCostAnalysis(ctx context.Context, query *BillingCostAnalysisQuery) (*BillingCostAnalysis, error) {
	if s == nil || s.store == nil {
		return nil, errors.New("billing service 未初始化")
	}
	if query == nil {
		query = &BillingCostAnalysisQuery{}
	}

	startDate, endDate, err := normalizeBillingWindow(query.StartDate, query.EndDate)
	if err != nil {
		return nil, err
	}
	groupBy, err := normalizeCostAnalysisGroupBy(query.GroupBy)
	if err != nil {
		return nil, err
	}
	limit := normalizeBillingLimit(query.Limit)

	summary, err := s.GetStatistics(ctx, startDate, endDate)
	if err != nil {
		return nil, err
	}

	rows, err := s.store.QueryCostBreakdown(ctx, startDate, endDate, groupBy, limit)
	if err != nil {
		return nil, err
	}

	breakdown := make([]*BillingCostBreakdown, 0, len(rows))
	for _, row := range rows {
		profit := row.TotalRevenue - row.TotalCost
		item := &BillingCostBreakdown{
			Dimension:    row.Dimension,
			RequestCount: row.RequestCount,
			TotalTokens:  row.TotalTokens,
			Revenue:      row.TotalRevenue,
			Cost:         row.TotalCost,
			Profit:       profit,
			Margin:       calculateMargin(profit, row.TotalRevenue),
		}
		if row.RequestCount > 0 {
			requestCount := float64(row.RequestCount)
			item.AvgRevenuePerRequest = row.TotalRevenue / requestCount
			item.AvgCostPerRequest = row.TotalCost / requestCount
			item.AvgProfitPerRequest = profit / requestCount
		}
		breakdown = append(breakdown, item)
	}

	return &BillingCostAnalysis{
		Summary:   summary,
		GroupBy:   groupBy,
		Breakdown: breakdown,
	}, nil
}

func normalizeBillingWindow(startDate, endDate time.Time) (time.Time, time.Time, error) {
	if startDate.IsZero() && endDate.IsZero() {
		endDate = time.Now()
		startDate = endDate.AddDate(0, 0, -30)
	}
	if startDate.IsZero() {
		startDate = endDate.AddDate(0, 0, -30)
	}
	if endDate.IsZero() {
		endDate = time.Now()
	}
	if !startDate.Before(endDate) && !startDate.Equal(endDate) {
		return time.Time{}, time.Time{}, errors.New("start_date 不能晚于 end_date")
	}
	return startDate, endDate, nil
}

func normalizeBillingLimit(limit int) int {
	if limit <= 0 {
		return 20
	}
	if limit > 200 {
		return 200
	}
	return limit
}

func normalizeUsageGroupBy(raw string) (string, error) {
	groupBy := strings.ToLower(strings.TrimSpace(raw))
	if groupBy == "" {
		groupBy = "day"
	}

	switch groupBy {
	case "day", "model", "user", "tenant":
		return groupBy, nil
	default:
		return "", fmt.Errorf("%w: usage 仅支持 day/model/user/tenant", ErrInvalidBillingGroupBy)
	}
}

func normalizeCostAnalysisGroupBy(raw string) (string, error) {
	groupBy := strings.ToLower(strings.TrimSpace(raw))
	if groupBy == "" {
		groupBy = "model"
	}

	switch groupBy {
	case "model", "channel", "user", "tenant":
		return groupBy, nil
	default:
		return "", fmt.Errorf("%w: cost-analysis 仅支持 model/channel/user/tenant", ErrInvalidBillingGroupBy)
	}
}

func calculateMargin(profit, revenue float64) float64 {
	if revenue <= 0 {
		return 0
	}
	return profit / revenue * 100
}

func (s *gormBillingStore) QueryStatistics(ctx context.Context, startDate, endDate time.Time) (*billingSummaryRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db 未初始化")
	}

	var row billingSummaryRow
	err := s.db.WithContext(ctx).
		Table("logs").
		Select(`
			COUNT(*) as total_requests,
			COALESCE(SUM(prompt_tokens + completion_tokens), 0) as total_tokens,
			COALESCE(SUM(total_cost), 0)::float8 as total_revenue,
			COALESCE(SUM(upstream_cost), 0)::float8 as total_cost,
			COUNT(DISTINCT user_id) as active_users
		`).
		Where("created_at >= ? AND created_at <= ?", startDate, endDate).
		Scan(&row).Error
	if err != nil {
		return nil, err
	}
	return &row, nil
}

func (s *gormBillingStore) QueryUsage(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db 未初始化")
	}

	dimensionExpr, groupExpr, orderExpr, joins, err := buildUsageGroupParts(groupBy)
	if err != nil {
		return nil, err
	}

	query := s.db.WithContext(ctx).Table("logs")
	for _, join := range joins {
		query = query.Joins(join)
	}

	var rows []*billingUsageRow
	err = query.
		Select(`
			` + dimensionExpr + ` as dimension,
			COUNT(*) as request_count,
			COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) as total_tokens,
			COALESCE(SUM(logs.total_cost), 0)::float8 as total_revenue,
			COALESCE(SUM(logs.upstream_cost), 0)::float8 as total_cost
		`).
		Where("logs.created_at >= ? AND logs.created_at <= ?", startDate, endDate).
		Group(groupExpr).
		Order(orderExpr).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}
	return rows, nil
}

func (s *gormBillingStore) QueryCostBreakdown(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error) {
	if s == nil || s.db == nil {
		return nil, errors.New("db 未初始化")
	}

	dimensionExpr, groupExpr, orderExpr, joins, err := buildCostGroupParts(groupBy)
	if err != nil {
		return nil, err
	}

	query := s.db.WithContext(ctx).Table("logs")
	for _, join := range joins {
		query = query.Joins(join)
	}

	var rows []*billingUsageRow
	err = query.
		Select(`
			` + dimensionExpr + ` as dimension,
			COUNT(*) as request_count,
			COALESCE(SUM(logs.prompt_tokens + logs.completion_tokens), 0) as total_tokens,
			COALESCE(SUM(logs.total_cost), 0)::float8 as total_revenue,
			COALESCE(SUM(logs.upstream_cost), 0)::float8 as total_cost
		`).
		Where("logs.created_at >= ? AND logs.created_at <= ?", startDate, endDate).
		Group(groupExpr).
		Order(orderExpr).
		Limit(limit).
		Scan(&rows).Error
	if err != nil {
		return nil, err
	}

	return rows, nil
}

func buildUsageGroupParts(groupBy string) (dimensionExpr, groupExpr, orderExpr string, joins []string, err error) {
	switch groupBy {
	case "day":
		return "TO_CHAR(DATE_TRUNC('day', logs.created_at), 'YYYY-MM-DD')", "DATE_TRUNC('day', logs.created_at)", "DATE_TRUNC('day', logs.created_at) ASC", nil, nil
	case "model":
		return "logs.model", "logs.model", "total_revenue DESC", nil, nil
	case "user":
		return "COALESCE(NULLIF(users.username, ''), logs.user_id::text)", "COALESCE(NULLIF(users.username, ''), logs.user_id::text)", "total_revenue DESC", []string{
			"LEFT JOIN users ON users.id = logs.user_id",
		}, nil
	case "tenant":
		return "COALESCE(NULLIF(tokens.tenant_id, ''), logs.user_id::text)", "COALESCE(NULLIF(tokens.tenant_id, ''), logs.user_id::text)", "total_revenue DESC", []string{
			"LEFT JOIN tokens ON tokens.id = logs.token_id",
		}, nil
	default:
		return "", "", "", nil, fmt.Errorf("%w: usage 仅支持 day/model/user/tenant", ErrInvalidBillingGroupBy)
	}
}

func buildCostGroupParts(groupBy string) (dimensionExpr, groupExpr, orderExpr string, joins []string, err error) {
	switch groupBy {
	case "model":
		return "logs.model", "logs.model", "total_cost DESC", nil, nil
	case "channel":
		return "COALESCE(NULLIF(channels.name, ''), CONCAT('channel#', logs.channel_id::text))", "COALESCE(NULLIF(channels.name, ''), CONCAT('channel#', logs.channel_id::text))", "total_cost DESC", []string{
			"LEFT JOIN channels ON channels.id = logs.channel_id",
		}, nil
	case "user":
		return "COALESCE(NULLIF(users.username, ''), logs.user_id::text)", "COALESCE(NULLIF(users.username, ''), logs.user_id::text)", "total_cost DESC", []string{
			"LEFT JOIN users ON users.id = logs.user_id",
		}, nil
	case "tenant":
		return "COALESCE(NULLIF(tokens.tenant_id, ''), logs.user_id::text)", "COALESCE(NULLIF(tokens.tenant_id, ''), logs.user_id::text)", "total_cost DESC", []string{
			"LEFT JOIN tokens ON tokens.id = logs.token_id",
		}, nil
	default:
		return "", "", "", nil, fmt.Errorf("%w: cost-analysis 仅支持 model/channel/user/tenant", ErrInvalidBillingGroupBy)
	}
}

