package service

import (
	"testing"

	"nexus-api/internal/model"

	"github.com/google/uuid"
)

type fakeSystemSettingRepo struct {
	stored     map[string]*model.SystemSetting
	lastUpsert *model.SystemSetting
}

func (r *fakeSystemSettingRepo) GetBySection(section string) (*model.SystemSetting, error) {
	if r.stored == nil {
		return nil, nil
	}
	setting, ok := r.stored[section]
	if !ok {
		return nil, nil
	}
	return setting, nil
}

func (r *fakeSystemSettingRepo) Upsert(setting *model.SystemSetting) error {
	r.lastUpsert = setting
	if r.stored == nil {
		r.stored = map[string]*model.SystemSetting{}
	}
	r.stored[setting.Section] = setting
	return nil
}

func TestSettingsService_GetSection_ReturnsEmptyWhenMissing(t *testing.T) {
	repo := &fakeSystemSettingRepo{}
	svc := NewSettingsService(repo)

	got, err := svc.GetSection("general")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got == nil {
		t.Fatalf("expected non-nil map")
	}
	if len(got) != 0 {
		t.Fatalf("expected empty map, got %v", got)
	}
}

func TestSettingsService_GetSection_MasksSensitiveFields(t *testing.T) {
	repo := &fakeSystemSettingRepo{
		stored: map[string]*model.SystemSetting{
			"security": {
				Section: "security",
				Data: model.JSON{
					"jwt_secret":      "real-secret",
					"session_timeout": 7200,
				},
			},
		},
	}
	svc := NewSettingsService(repo)

	got, err := svc.GetSection("security")
	if err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if got["jwt_secret"] != "********" {
		t.Fatalf("expected jwt_secret to be masked, got %v", got["jwt_secret"])
	}
	if got["session_timeout"] != 7200 {
		t.Fatalf("expected session_timeout to be preserved, got %v", got["session_timeout"])
	}
}

func TestSettingsService_UpdateSection_MergesAndIgnoresMaskedSensitive(t *testing.T) {
	repo := &fakeSystemSettingRepo{
		stored: map[string]*model.SystemSetting{
			"security": {
				Section: "security",
				Data: model.JSON{
					"jwt_secret":         "real-secret",
					"max_login_attempts": 3,
				},
			},
		},
	}
	svc := NewSettingsService(repo)

	adminID := uuid.New()
	patch := map[string]interface{}{
		"jwt_secret":         "•••••••••••••••••",
		"max_login_attempts": 5,
	}

	if err := svc.UpdateSection("security", patch, adminID); err != nil {
		t.Fatalf("expected nil error, got %v", err)
	}
	if repo.lastUpsert == nil {
		t.Fatalf("expected Upsert to be called")
	}
	if repo.lastUpsert.Section != "security" {
		t.Fatalf("expected section security, got %v", repo.lastUpsert.Section)
	}
	if repo.lastUpsert.UpdatedBy == nil || *repo.lastUpsert.UpdatedBy != adminID {
		t.Fatalf("expected UpdatedBy to be set to %v, got %v", adminID, repo.lastUpsert.UpdatedBy)
	}

	if repo.lastUpsert.Data["jwt_secret"] != "real-secret" {
		t.Fatalf("expected jwt_secret to remain unchanged, got %v", repo.lastUpsert.Data["jwt_secret"])
	}
	if repo.lastUpsert.Data["max_login_attempts"] != 5 {
		t.Fatalf("expected max_login_attempts to be updated to 5, got %v", repo.lastUpsert.Data["max_login_attempts"])
	}
}

func TestSettingsService_UpdateSection_RejectsInvalidSection(t *testing.T) {
	repo := &fakeSystemSettingRepo{}
	svc := NewSettingsService(repo)

	if err := svc.UpdateSection("bad", map[string]interface{}{"a": 1}, uuid.New()); err == nil {
		t.Fatalf("expected error for invalid section")
	}
}
