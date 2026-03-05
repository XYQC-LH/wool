package service

import (
	"context"
	"strings"
	"testing"

	"nexus-api/internal/model"
)

type fakeProviderCapabilityRepo struct {
	nextID uint
	items  map[uint]*model.ProviderCapability
}

func newFakeProviderCapabilityRepo() *fakeProviderCapabilityRepo {
	return &fakeProviderCapabilityRepo{
		nextID: 1,
		items:  map[uint]*model.ProviderCapability{},
	}
}

func (r *fakeProviderCapabilityRepo) clone(entity *model.ProviderCapability) *model.ProviderCapability {
	if entity == nil {
		return nil
	}
	copied := *entity
	if entity.Constraints != nil {
		cloned := model.JSON{}
		for key, value := range entity.Constraints {
			cloned[key] = value
		}
		copied.Constraints = cloned
	}
	return &copied
}

func (r *fakeProviderCapabilityRepo) Create(ctx context.Context, capability *model.ProviderCapability) error {
	if capability.ID == 0 {
		capability.ID = r.nextID
		r.nextID++
	}
	r.items[capability.ID] = r.clone(capability)
	return nil
}

func (r *fakeProviderCapabilityRepo) Update(ctx context.Context, capability *model.ProviderCapability) error {
	r.items[capability.ID] = r.clone(capability)
	return nil
}

func (r *fakeProviderCapabilityRepo) Delete(ctx context.Context, id uint) error {
	delete(r.items, id)
	return nil
}

func (r *fakeProviderCapabilityRepo) GetByID(ctx context.Context, id uint) (*model.ProviderCapability, error) {
	return r.clone(r.items[id]), nil
}

func (r *fakeProviderCapabilityRepo) GetByProvider(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error) {
	result := make([]*model.ProviderCapability, 0)
	for _, item := range r.items {
		if item.ProviderID == providerID && item.IsEnabled {
			result = append(result, r.clone(item))
		}
	}
	return result, nil
}

func (r *fakeProviderCapabilityRepo) GetByProviderAll(ctx context.Context, providerID uint) ([]*model.ProviderCapability, error) {
	result := make([]*model.ProviderCapability, 0)
	for _, item := range r.items {
		if item.ProviderID == providerID {
			result = append(result, r.clone(item))
		}
	}
	return result, nil
}

func (r *fakeProviderCapabilityRepo) GetByProviderAndOperation(ctx context.Context, providerID uint, operation string) (*model.ProviderCapability, error) {
	target := model.NormalizeOperation(operation)
	for _, item := range r.items {
		if item.ProviderID == providerID && model.NormalizeOperation(item.Operation) == target {
			return r.clone(item), nil
		}
	}
	return nil, nil
}

func (r *fakeProviderCapabilityRepo) List(ctx context.Context, providerID uint, operation string, isEnabled *bool, page, pageSize int) ([]*model.ProviderCapability, int64, error) {
	filtered := make([]*model.ProviderCapability, 0)
	targetOperation := model.NormalizeOperation(operation)
	for _, item := range r.items {
		if providerID > 0 && item.ProviderID != providerID {
			continue
		}
		if strings.TrimSpace(operation) != "" && model.NormalizeOperation(item.Operation) != targetOperation {
			continue
		}
		if isEnabled != nil && item.IsEnabled != *isEnabled {
			continue
		}
		filtered = append(filtered, r.clone(item))
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	offset := (page - 1) * pageSize
	if offset >= len(filtered) {
		return []*model.ProviderCapability{}, int64(len(filtered)), nil
	}
	end := offset + pageSize
	if end > len(filtered) {
		end = len(filtered)
	}
	return filtered[offset:end], int64(len(filtered)), nil
}

func (r *fakeProviderCapabilityRepo) BatchUpdateEnabled(ctx context.Context, ids []uint, enabled bool) error {
	for _, id := range ids {
		if item, ok := r.items[id]; ok {
			item.IsEnabled = enabled
			r.items[id] = item
		}
	}
	return nil
}

func (r *fakeProviderCapabilityRepo) GetSummary(ctx context.Context, providerID uint) (*model.ProviderCapabilitySummary, error) {
	summary := &model.ProviderCapabilitySummary{
		ProviderID:         providerID,
		OperationBreakdown: map[string]int{},
	}

	for _, item := range r.items {
		if providerID > 0 && item.ProviderID != providerID {
			continue
		}
		summary.Total++
		if item.IsEnabled {
			summary.Enabled++
		} else {
			summary.Disabled++
		}
		op := model.NormalizeOperation(item.Operation)
		summary.OperationBreakdown[op]++
	}
	return summary, nil
}

type fakeProviderReader struct {
	providers map[uint]*model.ModelProvider
}

func (r *fakeProviderReader) GetByID(ctx context.Context, id uint) (*model.ModelProvider, error) {
	if r.providers == nil {
		return nil, nil
	}
	return r.providers[id], nil
}

func TestProviderCapabilityServiceCreateDefaultValues(t *testing.T) {
	repo := newFakeProviderCapabilityRepo()
	reader := &fakeProviderReader{
		providers: map[uint]*model.ModelProvider{
			1: {ID: 1},
		},
	}
	svc := NewProviderCapabilityService(repo, reader)

	entity, err := svc.Create(context.Background(), &CreateProviderCapabilityRequest{
		ProviderID: 1,
		Operation:  "images.generations",
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if entity == nil {
		t.Fatalf("expected non-nil entity")
	}
	if !entity.IsEnabled {
		t.Fatalf("expected default is_enabled=true")
	}
	if entity.Operation != "images.generations" {
		t.Fatalf("unexpected operation: %s", entity.Operation)
	}
	if entity.Constraints == nil {
		t.Fatalf("expected default constraints map")
	}
}

func TestProviderCapabilityServiceUpdateConflict(t *testing.T) {
	repo := newFakeProviderCapabilityRepo()
	_ = repo.Create(context.Background(), &model.ProviderCapability{
		ProviderID: 1,
		Operation:  "images.generations",
		IsEnabled:  true,
	})
	_ = repo.Create(context.Background(), &model.ProviderCapability{
		ProviderID: 1,
		Operation:  "videos.generations",
		IsEnabled:  true,
	})

	reader := &fakeProviderReader{
		providers: map[uint]*model.ModelProvider{
			1: {ID: 1},
		},
	}
	svc := NewProviderCapabilityService(repo, reader)

	operation := "videos.generations"
	_, err := svc.Update(context.Background(), 1, &UpdateProviderCapabilityRequest{
		Operation: &operation,
	})
	if err == nil {
		t.Fatalf("expected conflict error")
	}
	if !strings.Contains(err.Error(), "已存在") {
		t.Fatalf("expected conflict message, got %v", err)
	}
}

func TestProviderCapabilityServiceValidateConstraints(t *testing.T) {
	repo := newFakeProviderCapabilityRepo()
	_ = repo.Create(context.Background(), &model.ProviderCapability{
		ProviderID: 1,
		Operation:  "images.generations",
		Constraints: model.JSON{
			"resolution": []interface{}{"1k", "2k"},
			"steps": map[string]interface{}{
				"min": 1,
				"max": 30,
			},
		},
		IsEnabled: true,
	})
	reader := &fakeProviderReader{
		providers: map[uint]*model.ModelProvider{
			1: {ID: 1},
		},
	}
	svc := NewProviderCapabilityService(repo, reader)

	okResult, err := svc.ValidateConstraints(context.Background(), &ValidateProviderConstraintsRequest{
		ProviderID: 1,
		Operation:  "images.generations",
		Constraints: map[string]interface{}{
			"resolution": "2k",
			"steps":      10,
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if okResult == nil || !okResult.Matched {
		t.Fatalf("expected matched result, got %+v", okResult)
	}

	failedResult, err := svc.ValidateConstraints(context.Background(), &ValidateProviderConstraintsRequest{
		ProviderID: 1,
		Operation:  "images.generations",
		Constraints: map[string]interface{}{
			"resolution": "4k",
		},
	})
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if failedResult == nil || failedResult.Matched {
		t.Fatalf("expected unmatched result, got %+v", failedResult)
	}
	if failedResult.Reason == "" {
		t.Fatalf("expected mismatch reason")
	}
}
