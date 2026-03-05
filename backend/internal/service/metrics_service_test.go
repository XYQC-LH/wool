package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"nexus-api/internal/model"
)

type fakeMetricsStore struct {
	queryMetricsFn         func(ctx context.Context, query *MetricsQuery) ([]*metricsQueryRow, int64, error)
	queryRealtimeSummaryFn func(ctx context.Context, startTime, endTime time.Time) (*realtimeSummaryRow, error)
	queryProviderHealthFn  func(ctx context.Context) (*providerHealthRow, error)
	countActiveAlertsFn    func(ctx context.Context) (int64, error)
}

func (f *fakeMetricsStore) QueryMetrics(ctx context.Context, query *MetricsQuery) ([]*metricsQueryRow, int64, error) {
	if f.queryMetricsFn != nil {
		return f.queryMetricsFn(ctx, query)
	}
	return []*metricsQueryRow{}, 0, nil
}

func (f *fakeMetricsStore) QueryRealtimeSummary(ctx context.Context, startTime, endTime time.Time) (*realtimeSummaryRow, error) {
	if f.queryRealtimeSummaryFn != nil {
		return f.queryRealtimeSummaryFn(ctx, startTime, endTime)
	}
	return &realtimeSummaryRow{}, nil
}

func (f *fakeMetricsStore) QueryProviderHealth(ctx context.Context) (*providerHealthRow, error) {
	if f.queryProviderHealthFn != nil {
		return f.queryProviderHealthFn(ctx)
	}
	return &providerHealthRow{}, nil
}

func (f *fakeMetricsStore) CountActiveAlerts(ctx context.Context) (int64, error) {
	if f.countActiveAlertsFn != nil {
		return f.countActiveAlertsFn(ctx)
	}
	return 0, nil
}

func TestMetricsServiceQueryConvertsSuccessRate(t *testing.T) {
	store := &fakeMetricsStore{
		queryMetricsFn: func(ctx context.Context, query *MetricsQuery) ([]*metricsQueryRow, int64, error) {
			return []*metricsQueryRow{
				{
					ProviderID:   1,
					ModelID:      "gpt-4o-mini",
					ModelName:    "GPT-4o Mini",
					RequestCount: 10,
					SuccessCount: 8,
					FailureCount: 2,
					InputTokens:  300,
					OutputTokens: 700,
					ActualCost:   1.2,
					Revenue:      2.4,
					Profit:       1.2,
				},
			}, 1, nil
		},
	}

	svc := newMetricsServiceWithStore(store)
	items, pagination, err := svc.Query(context.Background(), &MetricsQuery{
		Granularity: model.MetricGranularityHour,
		Page:        1,
		PageSize:    20,
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     time.Now(),
	})
	if err != nil {
		t.Fatalf("Query returned error: %v", err)
	}
	if pagination == nil || pagination.Total != 1 {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].SuccessRate < 79.9 || items[0].SuccessRate > 80.1 {
		t.Fatalf("unexpected success rate: %.2f", items[0].SuccessRate)
	}
	if items[0].TotalTokens != 1000 {
		t.Fatalf("unexpected total tokens: %d", items[0].TotalTokens)
	}
}

func TestMetricsServiceQueryRejectsInvalidGranularity(t *testing.T) {
	svc := newMetricsServiceWithStore(&fakeMetricsStore{})
	_, _, err := svc.Query(context.Background(), &MetricsQuery{
		Granularity: "week",
		StartTime:   time.Now().Add(-time.Hour),
		EndTime:     time.Now(),
	})
	if err == nil {
		t.Fatalf("expected invalid granularity error")
	}
	if !errors.Is(err, ErrInvalidMetricsGranularity) {
		t.Fatalf("expected ErrInvalidMetricsGranularity, got %v", err)
	}
}

func TestMetricsServiceGetRealtimeAggregatesData(t *testing.T) {
	store := &fakeMetricsStore{
		queryRealtimeSummaryFn: func(ctx context.Context, startTime, endTime time.Time) (*realtimeSummaryRow, error) {
			return &realtimeSummaryRow{
				RequestCount: 20,
				SuccessCount: 18,
				FailureCount: 2,
				AvgLatencyMs: 120,
				TotalTokens:  5000,
				TotalRevenue: 10,
				TotalCost:    4,
			}, nil
		},
		queryProviderHealthFn: func(ctx context.Context) (*providerHealthRow, error) {
			return &providerHealthRow{
				Total:       12,
				Active:      9,
				CircuitOpen: 1,
			}, nil
		},
		countActiveAlertsFn: func(ctx context.Context) (int64, error) {
			return 3, nil
		},
	}

	svc := newMetricsServiceWithStore(store)
	result, err := svc.GetRealtime(context.Background(), 10*time.Minute)
	if err != nil {
		t.Fatalf("GetRealtime returned error: %v", err)
	}
	if result.RequestCount != 20 {
		t.Fatalf("unexpected request count: %d", result.RequestCount)
	}
	if result.TotalProfit != 6 {
		t.Fatalf("unexpected total profit: %.2f", result.TotalProfit)
	}
	if result.ActiveAlerts != 3 {
		t.Fatalf("unexpected active alerts: %d", result.ActiveAlerts)
	}
	if result.Providers.Active != 9 {
		t.Fatalf("unexpected active providers: %d", result.Providers.Active)
	}
}
