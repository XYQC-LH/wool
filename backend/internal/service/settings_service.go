package service

import (
	"errors"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
)

// SettingsService 系统设置服务（按 section 存储 JSON 配置）
type SettingsService interface {
	GetSection(section string) (map[string]interface{}, error)
	UpdateSection(section string, patch map[string]interface{}, updatedBy uuid.UUID) error
}

type settingsService struct {
	repo repository.SystemSettingRepository
}

func NewSettingsService(repo repository.SystemSettingRepository) SettingsService {
	return &settingsService{repo: repo}
}

var validSettingSections = map[string]struct{}{
	"general":      {},
	"security":     {},
	"notification": {},
	"system":       {},
}

func (s *settingsService) GetSection(section string) (map[string]interface{}, error) {
	section = strings.TrimSpace(section)
	if !isValidSettingSection(section) {
		return nil, errors.New("无效的设置部分")
	}

	setting, err := s.repo.GetBySection(section)
	if err != nil {
		return nil, err
	}
	if setting == nil || setting.Data == nil {
		return map[string]interface{}{}, nil
	}

	data := cloneMap(setting.Data)
	maskSensitiveFields(section, data)
	return map[string]interface{}(data), nil
}

func (s *settingsService) UpdateSection(section string, patch map[string]interface{}, updatedBy uuid.UUID) error {
	section = strings.TrimSpace(section)
	if !isValidSettingSection(section) {
		return errors.New("无效的设置部分")
	}

	existing, err := s.repo.GetBySection(section)
	if err != nil {
		return err
	}

	merged := model.JSON{}
	if existing != nil && existing.Data != nil {
		merged = cloneMap(existing.Data)
	}

	if patch != nil {
		applyPatch(section, merged, patch)
	}

	uid := updatedBy
	setting := &model.SystemSetting{
		Section:   section,
		Data:      merged,
		UpdatedBy: &uid,
	}

	return s.repo.Upsert(setting)
}

func isValidSettingSection(section string) bool {
	_, ok := validSettingSections[section]
	return ok
}

func cloneMap(input model.JSON) model.JSON {
	if input == nil {
		return model.JSON{}
	}
	out := make(model.JSON, len(input))
	for k, v := range input {
		out[k] = v
	}
	return out
}

func maskSensitiveFields(section string, data model.JSON) {
	switch section {
	case "security":
		maskIfPresent(data, "jwt_secret")
	case "notification":
		maskIfPresent(data, "email_smtp_password")
	}
}

func maskIfPresent(data model.JSON, key string) {
	value, ok := data[key]
	if !ok || value == nil {
		return
	}
	str, ok := value.(string)
	if ok && strings.TrimSpace(str) == "" {
		return
	}
	data[key] = "********"
}

func applyPatch(section string, target model.JSON, patch map[string]interface{}) {
	for k, v := range patch {
		if v == nil {
			continue
		}

		if isSensitiveKey(section, k) {
			if shouldIgnoreSensitiveValue(v) {
				continue
			}
		}

		target[k] = v
	}
}

func isSensitiveKey(section, key string) bool {
	switch section {
	case "security":
		return key == "jwt_secret"
	case "notification":
		return key == "email_smtp_password"
	default:
		return false
	}
}

func shouldIgnoreSensitiveValue(value interface{}) bool {
	str, ok := value.(string)
	if !ok {
		return true
	}
	trimmed := strings.TrimSpace(str)
	if trimmed == "" {
		return true
	}
	// 前端 masked 占位（历史：•••••••• 或 ********）
	if isMaskedPlaceholder(trimmed) {
		return true
	}
	return false
}

func isMaskedPlaceholder(value string) bool {
	if value == "" {
		return true
	}
	allMask := true
	for _, r := range value {
		if r != '*' && r != '•' {
			allMask = false
			break
		}
	}
	return allMask
}
