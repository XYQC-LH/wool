package service

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/shopspring/decimal"
)

var (
	ErrQuotaPolicyNotFound      = errors.New("配额策略不存在")
	ErrQuotaPolicyConflict      = errors.New("配额策略冲突")
	ErrInvalidQuotaPolicyInput  = errors.New("无效的配额策略参数")
	ErrInvalidQuotaPolicyStatus = errors.New("无效的配额策略状态")
)

type quotaAlertSink interface {
	CreateAlert(alertType model.AlertType, severity model.AlertSeverity, title, message string, metadata model.JSON) error
}

type quotaUsageReader interface {
	GetDailyUsage(ctx context.Context, tenantID string, date time.Time) (*TenantQuotaUsage, error)
}

// QuotaService 租户配额管理服务
type QuotaService interface {
	Create(ctx context.Context, input *CreateQuotaPolicyInput) (*model.QuotaPolicy, error)
	Update(ctx context.Context, id string, input *UpdateQuotaPolicyInput) (*model.QuotaPolicy, error)
	Delete(ctx context.Context, id string) error
	GetByID(ctx context.Context, id string) (*model.QuotaPolicy, error)
	List(ctx context.Context, keyword string, status model.QuotaPolicyStatus, page, pageSize int) ([]*model.QuotaPolicy, int64, error)
	GetStats(ctx context.Context, date time.Time) (*QuotaStats, error)
	GetMonitoring(ctx context.Context, date time.Time, tenantID string) ([]*QuotaMonitoringItem, error)
	CheckAlerts(ctx context.Context, date time.Time) (*QuotaAlertCheckResult, error)
}

type quotaService struct {
	repo       repository.QuotaPolicyRepository
	usage      quotaUsageReader
	alertSink  quotaAlertSink
}

type redisQuotaUsageReader struct{}

// CreateQuotaPolicyInput 创建配额策略请求
type CreateQuotaPolicyInput struct {
	TenantID              string
	Name                  string
	Description           string
	DailyRequestLimit     int64
	DailyCostLimit        decimal.Decimal
	AlertThresholdPercent int
	Status                model.QuotaPolicyStatus
}

// UpdateQuotaPolicyInput 更新配额策略请求
type UpdateQuotaPolicyInput struct {
	TenantID              *string
	Name                  *string
	Description           *string
	DailyRequestLimit     *int64
	DailyCostLimit        *decimal.Decimal
	AlertThresholdPercent *int
	Status                *model.QuotaPolicyStatus
}

// TenantQuotaUsage 租户某日配额使用情况
type TenantQuotaUsage struct {
	TenantID        string          `json:"tenant_id"`
	Date            string          `json:"date"`
	RequestCount    int64           `json:"request_count"`
	BilledRequests  int64           `json:"billed_requests"`
	TotalCost       decimal.Decimal `json:"total_cost"`
	TotalTokensUsed int64           `json:"total_tokens_used"`
}

// QuotaMonitoringItem 配额监控视图
type QuotaMonitoringItem struct {
	TenantID              string          `json:"tenant_id"`
	Name                  string          `json:"name"`
	Date                  string          `json:"date"`
	PolicyStatus          string          `json:"policy_status"`
	AlertThresholdPercent int             `json:"alert_threshold_percent"`
	DailyRequestLimit     int64           `json:"daily_request_limit"`
	RequestCount          int64           `json:"request_count"`
	BilledRequests        int64           `json:"billed_requests"`
	RequestUtilization    float64         `json:"request_utilization"`
	DailyCostLimit        decimal.Decimal `json:"daily_cost_limit"`
	TotalCost             decimal.Decimal `json:"total_cost"`
	TotalTokensUsed       int64           `json:"total_tokens_used"`
	CostUtilization       float64         `json:"cost_utilization"`
	MaxUtilization        float64         `json:"max_utilization"`
	Status                string          `json:"status"` // normal / warning / exceeded / disabled / unlimited
}

// QuotaStats 配额统计
type QuotaStats struct {
	Date                     string  `json:"date"`
	TotalPolicies            int64   `json:"total_policies"`
	ActivePolicies           int64   `json:"active_policies"`
	DisabledPolicies         int64   `json:"disabled_policies"`
	NearLimitTenants         int64   `json:"near_limit_tenants"`
	ExceededTenants          int64   `json:"exceeded_tenants"`
	AverageRequestUtilization float64 `json:"average_request_utilization"`
	AverageCostUtilization   float64 `json:"average_cost_utilization"`
}

// QuotaAlertCheckResult 配额告警检查结果
type QuotaAlertCheckResult struct {
	Date            string `json:"date"`
	CheckedPolicies int    `json:"checked_policies"`
	CreatedAlerts   int    `json:"created_alerts"`
	WarningAlerts   int    `json:"warning_alerts"`
	CriticalAlerts  int    `json:"critical_alerts"`
}

func NewQuotaService(repo repository.QuotaPolicyRepository, alertSink quotaAlertSink) QuotaService {
	return newQuotaServiceWithDeps(repo, newRedisQuotaUsageReader(), alertSink)
}

func newQuotaServiceWithDeps(repo repository.QuotaPolicyRepository, usage quotaUsageReader, alertSink quotaAlertSink) *quotaService {
	if usage == nil {
		usage = newRedisQuotaUsageReader()
	}
	return &quotaService{
		repo:      repo,
		usage:     usage,
		alertSink: alertSink,
	}
}

func newRedisQuotaUsageReader() quotaUsageReader {
	return &redisQuotaUsageReader{}
}

func (r *redisQuotaUsageReader) GetDailyUsage(ctx context.Context, tenantID string, date time.Time) (*TenantQuotaUsage, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id 不能为空", ErrInvalidQuotaPolicyInput)
	}

	normalizedDate := normalizeQuotaDate(date)
	day := normalizedDate.Format("2006-01-02")
	usage := &TenantQuotaUsage{
		TenantID: tenantID,
		Date:     day,
	}

	client := cache.GetClient()
	if client == nil {
		return usage, nil
	}

	readInt64 := func(key string) (int64, error) {
		value, err := client.Get(cache.GetContext(), key).Int64()
		if err == nil {
			return value, nil
		}
		if errors.Is(err, redis.Nil) {
			return 0, nil
		}
		return 0, err
	}

	requestKey := tenantDailyRequestQuotaKey(tenantID, normalizedDate)
	requestCount, err := readInt64(requestKey)
	if err != nil {
		return nil, err
	}
	usage.RequestCount = requestCount

	billedRequestKey := tenantDailyRequestsKey(tenantID, normalizedDate)
	billedRequests, err := readInt64(billedRequestKey)
	if err != nil {
		return nil, err
	}
	usage.BilledRequests = billedRequests

	tokenKey := tenantDailyTokensKey(tenantID, normalizedDate)
	totalTokens, err := readInt64(tokenKey)
	if err != nil {
		return nil, err
	}
	usage.TotalTokensUsed = totalTokens

	costKey := tenantDailyCostKey(tenantID, normalizedDate)
	costMicro, err := readInt64(costKey)
	if err != nil {
		return nil, err
	}
	usage.TotalCost = microToDecimal(costMicro)

	return usage, nil
}

func (s *quotaService) Create(ctx context.Context, input *CreateQuotaPolicyInput) (*model.QuotaPolicy, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}

	policy, err := buildQuotaPolicyFromCreateInput(input)
	if err != nil {
		return nil, err
	}

	existing, err := s.repo.GetByTenantID(ctx, policy.TenantID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, fmt.Errorf("%w: tenant_id 已存在", ErrQuotaPolicyConflict)
	}

	if err := s.repo.Create(ctx, policy); err != nil {
		return nil, err
	}
	return policy, nil
}

func (s *quotaService) Update(ctx context.Context, id string, input *UpdateQuotaPolicyInput) (*model.QuotaPolicy, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}
	if input == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidQuotaPolicyInput)
	}

	policy, err := s.getPolicyByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if input.TenantID != nil {
		tenantID := strings.TrimSpace(*input.TenantID)
		if tenantID == "" {
			return nil, fmt.Errorf("%w: tenant_id 不能为空", ErrInvalidQuotaPolicyInput)
		}

		existing, err := s.repo.GetByTenantID(ctx, tenantID)
		if err != nil {
			return nil, err
		}
		if existing != nil && existing.ID != policy.ID {
			return nil, fmt.Errorf("%w: tenant_id 已存在", ErrQuotaPolicyConflict)
		}

		policy.TenantID = tenantID
	}

	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return nil, fmt.Errorf("%w: name 不能为空", ErrInvalidQuotaPolicyInput)
		}
		policy.Name = name
	}

	if input.Description != nil {
		policy.Description = strings.TrimSpace(*input.Description)
	}

	if input.DailyRequestLimit != nil {
		if *input.DailyRequestLimit < 0 {
			return nil, fmt.Errorf("%w: daily_request_limit 不能小于0", ErrInvalidQuotaPolicyInput)
		}
		policy.DailyRequestLimit = *input.DailyRequestLimit
	}

	if input.DailyCostLimit != nil {
		if input.DailyCostLimit.LessThan(decimal.Zero) {
			return nil, fmt.Errorf("%w: daily_cost_limit 不能小于0", ErrInvalidQuotaPolicyInput)
		}
		policy.DailyCostLimit = *input.DailyCostLimit
	}

	if input.AlertThresholdPercent != nil {
		threshold := *input.AlertThresholdPercent
		if threshold <= 0 || threshold > 100 {
			return nil, fmt.Errorf("%w: alert_threshold_percent 需在 1-100 之间", ErrInvalidQuotaPolicyInput)
		}
		policy.AlertThresholdPercent = threshold
	}

	if input.Status != nil {
		if !input.Status.IsValid() {
			return nil, fmt.Errorf("%w: status 必须是 active/disabled", ErrInvalidQuotaPolicyStatus)
		}
		policy.Status = *input.Status
	}

	if err := s.repo.Update(ctx, policy); err != nil {
		return nil, err
	}
	return policy, nil
}

func (s *quotaService) Delete(ctx context.Context, id string) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}

	policy, err := s.getPolicyByID(ctx, id)
	if err != nil {
		return err
	}
	return s.repo.Delete(ctx, policy.ID)
}

func (s *quotaService) GetByID(ctx context.Context, id string) (*model.QuotaPolicy, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}
	return s.getPolicyByID(ctx, id)
}

func (s *quotaService) List(ctx context.Context, keyword string, status model.QuotaPolicyStatus, page, pageSize int) ([]*model.QuotaPolicy, int64, error) {
	if s == nil || s.repo == nil {
		return nil, 0, fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}
	if status != "" && !status.IsValid() {
		return nil, 0, fmt.Errorf("%w: status 必须是 active/disabled", ErrInvalidQuotaPolicyStatus)
	}
	return s.repo.List(ctx, keyword, status, page, pageSize)
}

func (s *quotaService) GetStats(ctx context.Context, date time.Time) (*QuotaStats, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}

	policies, err := s.loadAllPolicies(ctx, "")
	if err != nil {
		return nil, err
	}

	normalizedDate := normalizeQuotaDate(date)
	stats := &QuotaStats{
		Date:          normalizedDate.Format("2006-01-02"),
		TotalPolicies: int64(len(policies)),
	}

	activePolicies := make([]*model.QuotaPolicy, 0, len(policies))
	for _, policy := range policies {
		if policy.Status == model.QuotaPolicyStatusActive {
			stats.ActivePolicies++
			activePolicies = append(activePolicies, policy)
		} else {
			stats.DisabledPolicies++
		}
	}

	items, err := s.buildMonitoringItems(ctx, activePolicies, normalizedDate)
	if err != nil {
		return nil, err
	}

	var requestRatioSum float64
	var requestRatioCount int64
	var costRatioSum float64
	var costRatioCount int64

	for _, item := range items {
		if item.Status == "warning" {
			stats.NearLimitTenants++
		}
		if item.Status == "exceeded" {
			stats.ExceededTenants++
		}

		if item.DailyRequestLimit > 0 {
			requestRatioSum += item.RequestUtilization
			requestRatioCount++
		}
		if item.DailyCostLimit.GreaterThan(decimal.Zero) {
			costRatioSum += item.CostUtilization
			costRatioCount++
		}
	}

	if requestRatioCount > 0 {
		stats.AverageRequestUtilization = requestRatioSum / float64(requestRatioCount)
	}
	if costRatioCount > 0 {
		stats.AverageCostUtilization = costRatioSum / float64(costRatioCount)
	}

	return stats, nil
}

func (s *quotaService) GetMonitoring(ctx context.Context, date time.Time, tenantID string) ([]*QuotaMonitoringItem, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}

	normalizedDate := normalizeQuotaDate(date)
	policies, err := s.loadPoliciesForMonitoring(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	items, err := s.buildMonitoringItems(ctx, policies, normalizedDate)
	if err != nil {
		return nil, err
	}

	sort.Slice(items, func(i, j int) bool {
		if items[i].MaxUtilization == items[j].MaxUtilization {
			return items[i].TenantID < items[j].TenantID
		}
		return items[i].MaxUtilization > items[j].MaxUtilization
	})

	return items, nil
}

func (s *quotaService) CheckAlerts(ctx context.Context, date time.Time) (*QuotaAlertCheckResult, error) {
	if s == nil || s.repo == nil {
		return nil, fmt.Errorf("%w: 配额服务未初始化", ErrInvalidQuotaPolicyInput)
	}

	normalizedDate := normalizeQuotaDate(date)
	result := &QuotaAlertCheckResult{
		Date: normalizedDate.Format("2006-01-02"),
	}

	activePolicies, err := s.repo.ListActive(ctx)
	if err != nil {
		return nil, err
	}
	result.CheckedPolicies = len(activePolicies)
	if len(activePolicies) == 0 || s.alertSink == nil {
		return result, nil
	}

	items, err := s.buildMonitoringItems(ctx, activePolicies, normalizedDate)
	if err != nil {
		return nil, err
	}

	for _, item := range items {
		created, critical, err := s.maybeCreateQuotaAlert(item, "request")
		if err != nil {
			return nil, err
		}
		if created {
			result.CreatedAlerts++
			if critical {
				result.CriticalAlerts++
			} else {
				result.WarningAlerts++
			}
		}

		created, critical, err = s.maybeCreateQuotaAlert(item, "cost")
		if err != nil {
			return nil, err
		}
		if created {
			result.CreatedAlerts++
			if critical {
				result.CriticalAlerts++
			} else {
				result.WarningAlerts++
			}
		}
	}

	return result, nil
}

func (s *quotaService) maybeCreateQuotaAlert(item *QuotaMonitoringItem, dimension string) (bool, bool, error) {
	if s.alertSink == nil || item == nil {
		return false, false, nil
	}
	if strings.TrimSpace(dimension) != "request" && strings.TrimSpace(dimension) != "cost" {
		return false, false, nil
	}

	threshold := float64(item.AlertThresholdPercent)
	var utilization float64
	var used, limit string

	if dimension == "request" {
		if item.DailyRequestLimit <= 0 {
			return false, false, nil
		}
		utilization = item.RequestUtilization
		used = fmt.Sprintf("%d", item.RequestCount)
		limit = fmt.Sprintf("%d", item.DailyRequestLimit)
	} else {
		if !item.DailyCostLimit.GreaterThan(decimal.Zero) {
			return false, false, nil
		}
		utilization = item.CostUtilization
		used = item.TotalCost.StringFixed(6)
		limit = item.DailyCostLimit.StringFixed(6)
	}

	if utilization < threshold {
		return false, false, nil
	}

	isCritical := utilization >= 100
	if !s.allowQuotaAlert(item.TenantID, item.Date, dimension, isCritical) {
		return false, isCritical, nil
	}

	alertType := model.AlertTypeQuotaWarning
	severity := model.AlertSeverityWarning
	levelDesc := "接近阈值"
	if isCritical {
		alertType = model.AlertTypeQuotaExceeded
		severity = model.AlertSeverityCritical
		levelDesc = "已超限"
	}

	title := fmt.Sprintf("租户 %s %s配额%s", item.TenantID, dimensionLabel(dimension), levelDesc)
	message := fmt.Sprintf(
		"租户 %s 当日%s配额使用率 %.2f%%（已用 %s / 限额 %s，阈值 %d%%）",
		item.TenantID,
		dimensionLabel(dimension),
		utilization,
		used,
		limit,
		item.AlertThresholdPercent,
	)

	metadata := model.JSON{
		"tenant_id":      item.TenantID,
		"date":           item.Date,
		"dimension":      dimension,
		"utilization":    utilization,
		"used":           used,
		"limit":          limit,
		"threshold":      item.AlertThresholdPercent,
		"policy_status":  item.PolicyStatus,
		"monitor_status": item.Status,
	}

	if err := s.alertSink.CreateAlert(alertType, severity, title, message, metadata); err != nil {
		return false, isCritical, err
	}

	return true, isCritical, nil
}

func (s *quotaService) allowQuotaAlert(tenantID, day, dimension string, isCritical bool) bool {
	if cache.GetClient() == nil {
		return true
	}

	level := "warning"
	if isCritical {
		level = "critical"
	}
	key := fmt.Sprintf("alert:quota:%s:%s:%s:%s", tenantID, day, dimension, level)
	ok, err := cache.SetNX(key, map[string]string{
		"tenant_id": tenantID,
		"date":      day,
	}, 26*time.Hour)
	if err != nil {
		return true
	}
	return ok
}

func (s *quotaService) loadPoliciesForMonitoring(ctx context.Context, tenantID string) ([]*model.QuotaPolicy, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return s.loadAllPolicies(ctx, "")
	}

	policy, err := s.repo.GetByTenantID(ctx, tenantID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return []*model.QuotaPolicy{}, nil
	}
	return []*model.QuotaPolicy{policy}, nil
}

func (s *quotaService) buildMonitoringItems(ctx context.Context, policies []*model.QuotaPolicy, date time.Time) ([]*QuotaMonitoringItem, error) {
	normalizedDate := normalizeQuotaDate(date)
	items := make([]*QuotaMonitoringItem, 0, len(policies))

	for _, policy := range policies {
		usage, err := s.usage.GetDailyUsage(ctx, policy.TenantID, normalizedDate)
		if err != nil {
			return nil, err
		}
		if usage == nil {
			usage = &TenantQuotaUsage{
				TenantID: policy.TenantID,
				Date:     normalizedDate.Format("2006-01-02"),
			}
		}
		items = append(items, buildQuotaMonitoringItem(policy, usage))
	}

	return items, nil
}

func buildQuotaMonitoringItem(policy *model.QuotaPolicy, usage *TenantQuotaUsage) *QuotaMonitoringItem {
	requestUtilization := computeInt64QuotaUsagePercent(usage.RequestCount, policy.DailyRequestLimit)
	costUtilization := computeDecimalQuotaUsagePercent(usage.TotalCost, policy.DailyCostLimit)

	maxUtilization := requestUtilization
	if costUtilization > maxUtilization {
		maxUtilization = costUtilization
	}

	monitorStatus := "normal"
	if policy.Status != model.QuotaPolicyStatusActive {
		monitorStatus = "disabled"
	} else if policy.DailyRequestLimit <= 0 && !policy.DailyCostLimit.GreaterThan(decimal.Zero) {
		monitorStatus = "unlimited"
	} else if maxUtilization >= 100 {
		monitorStatus = "exceeded"
	} else if maxUtilization >= float64(policy.EffectiveThresholdPercent()) {
		monitorStatus = "warning"
	}

	return &QuotaMonitoringItem{
		TenantID:              policy.TenantID,
		Name:                  policy.Name,
		Date:                  usage.Date,
		PolicyStatus:          string(policy.Status),
		AlertThresholdPercent: policy.EffectiveThresholdPercent(),
		DailyRequestLimit:     policy.DailyRequestLimit,
		RequestCount:          usage.RequestCount,
		BilledRequests:        usage.BilledRequests,
		RequestUtilization:    requestUtilization,
		DailyCostLimit:        policy.DailyCostLimit,
		TotalCost:             usage.TotalCost,
		TotalTokensUsed:       usage.TotalTokensUsed,
		CostUtilization:       costUtilization,
		MaxUtilization:        maxUtilization,
		Status:                monitorStatus,
	}
}

func computeInt64QuotaUsagePercent(used, limit int64) float64 {
	if limit <= 0 {
		return 0
	}
	return float64(used) / float64(limit) * 100
}

func computeDecimalQuotaUsagePercent(used, limit decimal.Decimal) float64 {
	if !limit.GreaterThan(decimal.Zero) {
		return 0
	}
	return used.Div(limit).Mul(decimal.NewFromInt(100)).InexactFloat64()
}

func buildQuotaPolicyFromCreateInput(input *CreateQuotaPolicyInput) (*model.QuotaPolicy, error) {
	if input == nil {
		return nil, fmt.Errorf("%w: 请求体不能为空", ErrInvalidQuotaPolicyInput)
	}

	tenantID := strings.TrimSpace(input.TenantID)
	if tenantID == "" {
		return nil, fmt.Errorf("%w: tenant_id 不能为空", ErrInvalidQuotaPolicyInput)
	}

	name := strings.TrimSpace(input.Name)
	if name == "" {
		name = tenantID
	}

	if input.DailyRequestLimit < 0 {
		return nil, fmt.Errorf("%w: daily_request_limit 不能小于0", ErrInvalidQuotaPolicyInput)
	}
	if input.DailyCostLimit.LessThan(decimal.Zero) {
		return nil, fmt.Errorf("%w: daily_cost_limit 不能小于0", ErrInvalidQuotaPolicyInput)
	}

	threshold := input.AlertThresholdPercent
	if threshold <= 0 {
		threshold = 80
	}
	if threshold > 100 {
		return nil, fmt.Errorf("%w: alert_threshold_percent 需在 1-100 之间", ErrInvalidQuotaPolicyInput)
	}

	status := input.Status
	if status == "" {
		status = model.QuotaPolicyStatusActive
	}
	if !status.IsValid() {
		return nil, fmt.Errorf("%w: status 必须是 active/disabled", ErrInvalidQuotaPolicyStatus)
	}

	return &model.QuotaPolicy{
		TenantID:              tenantID,
		Name:                  name,
		Description:           strings.TrimSpace(input.Description),
		DailyRequestLimit:     input.DailyRequestLimit,
		DailyCostLimit:        input.DailyCostLimit,
		AlertThresholdPercent: threshold,
		Status:                status,
	}, nil
}

func (s *quotaService) getPolicyByID(ctx context.Context, id string) (*model.QuotaPolicy, error) {
	parsedID, err := parseUUID(id)
	if err != nil {
		return nil, fmt.Errorf("%w: id 格式错误", ErrInvalidQuotaPolicyInput)
	}
	policy, err := s.repo.GetByID(ctx, parsedID)
	if err != nil {
		return nil, err
	}
	if policy == nil {
		return nil, ErrQuotaPolicyNotFound
	}
	return policy, nil
}

func (s *quotaService) loadAllPolicies(ctx context.Context, status model.QuotaPolicyStatus) ([]*model.QuotaPolicy, error) {
	const pageSize = 200
	page := 1
	result := make([]*model.QuotaPolicy, 0)

	for {
		items, total, err := s.repo.List(ctx, "", status, page, pageSize)
		if err != nil {
			return nil, err
		}
		result = append(result, items...)
		if int64(len(result)) >= total || len(items) == 0 {
			break
		}
		page++
	}

	return result, nil
}

func normalizeQuotaDate(date time.Time) time.Time {
	if date.IsZero() {
		date = time.Now()
	}
	return time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
}

func tenantDailyRequestQuotaKey(tenantID string, ts time.Time) string {
	return fmt.Sprintf("quota:tenant:daily:%s:%s", tenantID, ts.Format("2006-01-02"))
}

func dimensionLabel(dimension string) string {
	switch strings.TrimSpace(dimension) {
	case "request":
		return "请求"
	case "cost":
		return "成本"
	default:
		return "未知"
	}
}

func parseUUID(raw string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimSpace(raw))
}
