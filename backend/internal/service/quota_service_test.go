package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type fakeQuotaPolicyRepo struct {
	items map[uuid.UUID]*model.QuotaPolicy
}

func newFakeQuotaPolicyRepo() *fakeQuotaPolicyRepo {
	return &fakeQuotaPolicyRepo{
		items: map[uuid.UUID]*model.QuotaPolicy{},
	}
}

func (r *fakeQuotaPolicyRepo) clone(policy *model.QuotaPolicy) *model.QuotaPolicy {
	if policy == nil {
		return nil
	}
	copied := *policy
	return &copied
}

func (r *fakeQuotaPolicyRepo) Create(ctx context.Context, policy *model.QuotaPolicy) error {
	if policy.ID == uuid.Nil {
		policy.ID = uuid.New()
	}
	r.items[policy.ID] = r.clone(policy)
	return nil
}

func (r *fakeQuotaPolicyRepo) Update(ctx context.Context, policy *model.QuotaPolicy) error {
	r.items[policy.ID] = r.clone(policy)
	return nil
}

func (r *fakeQuotaPolicyRepo) Delete(ctx context.Context, id uuid.UUID) error {
	delete(r.items, id)
	return nil
}

func (r *fakeQuotaPolicyRepo) GetByID(ctx context.Context, id uuid.UUID) (*model.QuotaPolicy, error) {
	return r.clone(r.items[id]), nil
}

func (r *fakeQuotaPolicyRepo) GetByTenantID(ctx context.Context, tenantID string) (*model.QuotaPolicy, error) {
	for _, item := range r.items {
		if strings.TrimSpace(item.TenantID) == strings.TrimSpace(tenantID) {
			return r.clone(item), nil
		}
	}
	return nil, nil
}

func (r *fakeQuotaPolicyRepo) List(ctx context.Context, keyword string, status model.QuotaPolicyStatus, page, pageSize int) ([]*model.QuotaPolicy, int64, error) {
	list := make([]*model.QuotaPolicy, 0)
	for _, item := range r.items {
		if keyword != "" && !strings.Contains(item.TenantID, keyword) && !strings.Contains(item.Name, keyword) {
			continue
		}
		if status.IsValid() && item.Status != status {
			continue
		}
		list = append(list, r.clone(item))
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if offset >= len(list) {
		return []*model.QuotaPolicy{}, int64(len(list)), nil
	}
	end := offset + pageSize
	if end > len(list) {
		end = len(list)
	}
	return list[offset:end], int64(len(list)), nil
}

func (r *fakeQuotaPolicyRepo) ListActive(ctx context.Context) ([]*model.QuotaPolicy, error) {
	list := make([]*model.QuotaPolicy, 0)
	for _, item := range r.items {
		if item.Status == model.QuotaPolicyStatusActive {
			list = append(list, r.clone(item))
		}
	}
	return list, nil
}

type fakeQuotaUsageReader struct {
	usage map[string]*TenantQuotaUsage
}

func (r *fakeQuotaUsageReader) GetDailyUsage(ctx context.Context, tenantID string, date time.Time) (*TenantQuotaUsage, error) {
	key := tenantID + ":" + date.Format("2006-01-02")
	if value, ok := r.usage[key]; ok {
		copied := *value
		return &copied, nil
	}
	return &TenantQuotaUsage{
		TenantID: tenantID,
		Date:     date.Format("2006-01-02"),
	}, nil
}

type fakeQuotaAlertSink struct {
	alerts []model.AlertType
}

func (s *fakeQuotaAlertSink) CreateAlert(alertType model.AlertType, severity model.AlertSeverity, title, message string, metadata model.JSON) error {
	s.alerts = append(s.alerts, alertType)
	return nil
}

func TestQuotaServiceCreateConflict(t *testing.T) {
	repo := newFakeQuotaPolicyRepo()
	err := repo.Create(context.Background(), &model.QuotaPolicy{
		TenantID:              "tenant-a",
		Name:                  "tenant-a",
		DailyRequestLimit:     100,
		DailyCostLimit:        decimal.NewFromInt(10),
		AlertThresholdPercent: 80,
		Status:                model.QuotaPolicyStatusActive,
	})
	if err != nil {
		t.Fatalf("seed policy failed: %v", err)
	}

	svc := newQuotaServiceWithDeps(repo, &fakeQuotaUsageReader{}, nil)
	_, err = svc.Create(context.Background(), &CreateQuotaPolicyInput{
		TenantID:          "tenant-a",
		DailyRequestLimit: 10,
		DailyCostLimit:    decimal.NewFromInt(1),
	})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if !errors.Is(err, ErrQuotaPolicyConflict) {
		t.Fatalf("expected ErrQuotaPolicyConflict, got %v", err)
	}
}

func TestQuotaServiceGetMonitoringWarning(t *testing.T) {
	repo := newFakeQuotaPolicyRepo()
	policy := &model.QuotaPolicy{
		TenantID:              "tenant-a",
		Name:                  "tenant-a",
		DailyRequestLimit:     100,
		DailyCostLimit:        decimal.NewFromInt(10),
		AlertThresholdPercent: 80,
		Status:                model.QuotaPolicyStatusActive,
	}
	_ = repo.Create(context.Background(), policy)

	usageReader := &fakeQuotaUsageReader{
		usage: map[string]*TenantQuotaUsage{
			"tenant-a:2026-03-05": {
				TenantID:     "tenant-a",
				Date:         "2026-03-05",
				RequestCount: 90,
				TotalCost:    decimal.NewFromFloat(9.2),
			},
		},
	}

	svc := newQuotaServiceWithDeps(repo, usageReader, nil)
	items, err := svc.GetMonitoring(context.Background(), time.Date(2026, 3, 5, 10, 0, 0, 0, time.Local), "")
	if err != nil {
		t.Fatalf("GetMonitoring returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Status != "warning" {
		t.Fatalf("expected status=warning, got %s", items[0].Status)
	}
	if items[0].RequestUtilization < 89.9 {
		t.Fatalf("unexpected request utilization: %.2f", items[0].RequestUtilization)
	}
}

func TestQuotaServiceCheckAlertsCreatesCriticalAlerts(t *testing.T) {
	repo := newFakeQuotaPolicyRepo()
	policy := &model.QuotaPolicy{
		TenantID:              "tenant-a",
		Name:                  "tenant-a",
		DailyRequestLimit:     100,
		DailyCostLimit:        decimal.NewFromInt(10),
		AlertThresholdPercent: 80,
		Status:                model.QuotaPolicyStatusActive,
	}
	_ = repo.Create(context.Background(), policy)

	usageReader := &fakeQuotaUsageReader{
		usage: map[string]*TenantQuotaUsage{
			"tenant-a:2026-03-05": {
				TenantID:     "tenant-a",
				Date:         "2026-03-05",
				RequestCount: 120,
				TotalCost:    decimal.NewFromFloat(12.5),
			},
		},
	}
	alertSink := &fakeQuotaAlertSink{}
	svc := newQuotaServiceWithDeps(repo, usageReader, alertSink)

	result, err := svc.CheckAlerts(context.Background(), time.Date(2026, 3, 5, 8, 0, 0, 0, time.Local))
	if err != nil {
		t.Fatalf("CheckAlerts returned error: %v", err)
	}
	if result.CreatedAlerts != 2 {
		t.Fatalf("expected created alerts = 2, got %d", result.CreatedAlerts)
	}
	if result.CriticalAlerts != 2 {
		t.Fatalf("expected critical alerts = 2, got %d", result.CriticalAlerts)
	}
	if len(alertSink.alerts) != 2 {
		t.Fatalf("expected 2 alert records, got %d", len(alertSink.alerts))
	}
}
