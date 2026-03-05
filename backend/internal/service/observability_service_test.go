package service

import (
	"context"
	"testing"
	"time"
)

func TestObservabilityServiceAnalyzePerformanceNoData(t *testing.T) {
	svc := NewObservabilityService(nil)

	result, err := svc.AnalyzePerformance(context.Background(), &PerformanceAnalysisQuery{
		Window:          5 * time.Minute,
		SlowThresholdMs: 1000,
		TopN:            5,
	})
	if err != nil {
		t.Fatalf("AnalyzePerformance returned error: %v", err)
	}
	if result == nil {
		t.Fatalf("expected non-nil result")
	}
}

func TestObservabilityServiceForecastCapacityWithoutDB(t *testing.T) {
	svc := NewObservabilityService(nil)

	_, err := svc.ForecastCapacity(context.Background(), &CapacityForecastQuery{
		LookbackHours: 24,
		ForecastHours: 24,
	})
	if err == nil {
		t.Fatalf("expected error when db is nil")
	}
}
