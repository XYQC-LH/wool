package repository

import (
	"errors"

	"nexus-api/internal/model"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// SystemSettingRepository 系统设置仓库
type SystemSettingRepository interface {
	GetBySection(section string) (*model.SystemSetting, error)
	Upsert(setting *model.SystemSetting) error
}

type systemSettingRepository struct {
	db *gorm.DB
}

func NewSystemSettingRepository(db *gorm.DB) SystemSettingRepository {
	return &systemSettingRepository{db: db}
}

func (r *systemSettingRepository) GetBySection(section string) (*model.SystemSetting, error) {
	var setting model.SystemSetting
	if err := r.db.Where("section = ?", section).First(&setting).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &setting, nil
}

func (r *systemSettingRepository) Upsert(setting *model.SystemSetting) error {
	if setting == nil {
		return errors.New("setting is nil")
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{{Name: "section"}},
		DoUpdates: clause.AssignmentColumns([]string{
			"data",
			"updated_by",
			"updated_at",
		}),
	}).Create(setting).Error
}
