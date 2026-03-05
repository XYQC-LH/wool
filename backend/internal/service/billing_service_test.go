package service

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeBillingStore struct {
	queryStatisticsFn    func(ctx context.Context, startDate, endDate time.Time) (*billingSummaryRow, error)
	queryUsageFn         func(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error)
	queryCostBreakdownFn func(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error)
}

func (s *fakeBillingStore) QueryStatistics(ctx context.Context, startDate, endDate time.Time) (*billingSummaryRow, error) {
	if s.queryStatisticsFn != nil {
		return s.queryStatisticsFn(ctx, startDate, endDate)
	}
	return &billingSummaryRow{}, nil
}

func (s *fakeBillingStore) QueryUsage(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error) {
	if s.queryUsageFn != nil {
		return s.queryUsageFn(ctx, startDate, endDate, groupBy, limit)
	}
	return []*billingUsageRow{}, nil
}

func (s *fakeBillingStore) QueryCostBreakdown(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error) {
	if s.queryCostBreakdownFn != nil {
		return s.queryCostBreakdownFn(ctx, startDate, endDate, groupBy, limit)
	}
	return []*billingUsageRow{}, nil
}

func TestBillingServiceGetUsageCalculatesProfitAndMargin(t *testing.T) {
	store := &fakeBillingStore{
		queryUsageFn: func(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error) {
			return []*billingUsageRow{
				{
					Dimension:    "gpt-4o-mini",
					RequestCount: 2,
					TotalTokens:  3000,
					TotalRevenue: 12.5,
					TotalCost:    4.5,
				},
			}, nil
		},
	}
	svc := newBillingServiceWithStore(store)

	items, err := svc.GetUsage(context.Background(), &BillingUsageQuery{
		StartDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
		EndDate:   time.Date(2026, 3, 5, 23, 59, 59, 0, time.Local),
		GroupBy:   "model",
		Limit:     10,
	})
	if err != nil {
		t.Fatalf("GetUsage returned error: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	if items[0].Profit != 8.0 {
		t.Fatalf("expected profit=8.0, got %.2f", items[0].Profit)
	}
	if items[0].Margin < 63.9 || items[0].Margin > 64.1 {
		t.Fatalf("unexpected margin: %.2f", items[0].Margin)
	}
}

func TestBillingServiceGetCostAnalysisIncludesSummaryAndBreakdown(t *testing.T) {
	store := &fakeBillingStore{
		queryStatisticsFn: func(ctx context.Context, startDate, endDate time.Time) (*billingSummaryRow, error) {
			return &billingSummaryRow{
				TotalRequests: 5,
				TotalTokens:   5000,
				TotalRevenue:  20,
				TotalCost:     8,
				ActiveUsers:   2,
			}, nil
		},
		queryCostBreakdownFn: func(ctx context.Context, startDate, endDate time.Time, groupBy string, limit int) ([]*billingUsageRow, error) {
			return []*billingUsageRow{
				{
					Dimension:    "tenant-a",
					RequestCount: 5,
					TotalTokens:  5000,
					TotalRevenue: 20,
					TotalCost:    8,
				},
			}, nil
		},
	}
	svc := newBillingServiceWithStore(store)

	result, err := svc.GetCostAnalysis(context.Background(), &BillingCostAnalysisQuery{
		StartDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
		EndDate:   time.Date(2026, 3, 5, 23, 59, 59, 0, time.Local),
		GroupBy:   "tenant",
		Limit:     20,
	})
	if err != nil {
		t.Fatalf("GetCostAnalysis returned error: %v", err)
	}
	if result.Summary == nil {
		t.Fatalf("expected non-nil summary")
	}
	if result.Summary.TotalProfit != 12 {
		t.Fatalf("expected summary profit=12, got %.2f", result.Summary.TotalProfit)
	}
	if len(result.Breakdown) != 1 {
		t.Fatalf("expected 1 breakdown item, got %d", len(result.Breakdown))
	}
	if result.Breakdown[0].AvgProfitPerRequest != 2.4 {
		t.Fatalf("expected avg profit per request=2.4, got %.2f", result.Breakdown[0].AvgProfitPerRequest)
	}
}

func TestBillingServiceRejectsInvalidGroupBy(t *testing.T) {
	svc := newBillingServiceWithStore(&fakeBillingStore{})

	_, err := svc.GetUsage(context.Background(), &BillingUsageQuery{
		StartDate: time.Date(2026, 3, 1, 0, 0, 0, 0, time.Local),
		EndDate:   time.Date(2026, 3, 5, 23, 59, 59, 0, time.Local),
		GroupBy:   "invalid",
		Limit:     20,
	})
	if err == nil {
		t.Fatalf("expected invalid group_by error")
	}
	if !errors.Is(err, ErrInvalidBillingGroupBy) {
		t.Fatalf("expected ErrInvalidBillingGroupBy, got %v", err)
	}
}

