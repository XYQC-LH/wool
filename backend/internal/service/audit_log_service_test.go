package service

import (
	"context"
	"testing"

	"nexus-api/internal/model"

	"github.com/google/uuid"
)

type fakeAuditLogRepo struct {
	createFn func(log *model.AuditLog) error
	listFn   func(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLog, int64, error)
}

func (f *fakeAuditLogRepo) Create(log *model.AuditLog) error {
	if f.createFn != nil {
		return f.createFn(log)
	}
	return nil
}

func (f *fakeAuditLogRepo) List(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLog, int64, error) {
	if f.listFn != nil {
		return f.listFn(page, pageSize, filters)
	}
	return []*model.AuditLog{}, 0, nil
}

func TestAuditLogServiceRecord(t *testing.T) {
	var captured *model.AuditLog
	svc := NewAuditLogService(&fakeAuditLogRepo{
		createFn: func(log *model.AuditLog) error {
			captured = log
			return nil
		},
	})

	actorID := uuid.New()
	err := svc.Record(context.Background(), &CreateAuditLogInput{
		ActorID:    &actorID,
		ActorRole:  "admin",
		Action:     "PUT /api/admin/channels/:id/status",
		Resource:   "channels",
		Method:     "PUT",
		Path:       "/api/admin/channels/:id/status",
		StatusCode: 200,
		Success:    true,
	})
	if err != nil {
		t.Fatalf("Record returned error: %v", err)
	}
	if captured == nil {
		t.Fatalf("expected captured audit log")
	}
	if captured.Action == "" {
		t.Fatalf("expected non-empty action")
	}
}

func TestAuditLogServiceList(t *testing.T) {
	svc := NewAuditLogService(&fakeAuditLogRepo{
		listFn: func(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLog, int64, error) {
			return []*model.AuditLog{
				{
					Action: "GET /api/admin/logs",
					Method: "GET",
				},
			}, 1, nil
		},
	})

	list, pagination, err := svc.List(1, 20, map[string]interface{}{})
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 item, got %d", len(list))
	}
	if pagination == nil || pagination.Total != 1 {
		t.Fatalf("unexpected pagination: %+v", pagination)
	}
}
