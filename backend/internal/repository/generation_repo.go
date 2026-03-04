package repository

import (
	"context"
	"time"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GenerationRepository 生成任务仓库接口
type GenerationRepository interface {
	Create(task *model.GenerationTask) error
	GetByID(id uuid.UUID) (*model.GenerationTask, error)
	GetByUserID(userID uuid.UUID, taskType model.GenerationType, page, pageSize int) ([]model.GenerationTask, int64, error)
	Update(task *model.GenerationTask) error
	UpdateStatus(id uuid.UUID, status model.GenerationStatus, progress float64, resultURL *string, errorMsg *string) error
	Delete(id uuid.UUID) error
	GetPendingTasks() ([]model.GenerationTask, error)
	ClaimPendingTasks(ctx context.Context, limit int) ([]*model.GenerationTask, error)
	RequeueStaleProcessingTasks(ctx context.Context, staleBefore time.Time) (int64, error)
}

// generationRepository 生成任务仓库实现
type generationRepository struct {
	db *gorm.DB
}

// NewGenerationRepository 创建生成任务仓库
func NewGenerationRepository(db *gorm.DB) GenerationRepository {
	return &generationRepository{db: db}
}

// Create 创建生成任务
func (r *generationRepository) Create(task *model.GenerationTask) error {
	return r.db.Create(task).Error
}

// GetByID 根据 ID 获取生成任务
func (r *generationRepository) GetByID(id uuid.UUID) (*model.GenerationTask, error) {
	var task model.GenerationTask
	err := r.db.Where("id = ?", id).First(&task).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// GetByUserID 根据用户 ID 获取生成任务列表
func (r *generationRepository) GetByUserID(userID uuid.UUID, taskType model.GenerationType, page, pageSize int) ([]model.GenerationTask, int64, error) {
	var tasks []model.GenerationTask
	var total int64

	query := r.db.Model(&model.GenerationTask{}).Where("user_id = ?", userID)
	if taskType != "" {
		query = query.Where("type = ?", taskType)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&tasks).Error; err != nil {
		return nil, 0, err
	}

	return tasks, total, nil
}

// Update 更新生成任务
func (r *generationRepository) Update(task *model.GenerationTask) error {
	return r.db.Save(task).Error
}

// UpdateStatus 更新任务状态
func (r *generationRepository) UpdateStatus(id uuid.UUID, status model.GenerationStatus, progress float64, resultURL *string, errorMsg *string) error {
	updates := map[string]interface{}{
		"status":   status,
		"progress": progress,
	}
	if resultURL != nil {
		updates["result_url"] = *resultURL
	}
	if errorMsg != nil {
		updates["error_message"] = *errorMsg
	}
	if status == model.GenerationStatusCompleted || status == model.GenerationStatusFailed {
		updates["completed_at"] = gorm.Expr("NOW()")
	}
	return r.db.Model(&model.GenerationTask{}).Where("id = ?", id).Updates(updates).Error
}

// Delete 删除生成任务
func (r *generationRepository) Delete(id uuid.UUID) error {
	return r.db.Delete(&model.GenerationTask{}, "id = ?", id).Error
}

// GetPendingTasks 获取待处理的任务
func (r *generationRepository) GetPendingTasks() ([]model.GenerationTask, error) {
	var tasks []model.GenerationTask
	err := r.db.Where("status IN ?", []model.GenerationStatus{
		model.GenerationStatusPending,
		model.GenerationStatusProcessing,
	}).Order("created_at ASC").Find(&tasks).Error
	return tasks, err
}

func (r *generationRepository) ClaimPendingTasks(ctx context.Context, limit int) ([]*model.GenerationTask, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if limit <= 0 {
		limit = 10
	}
	if limit > 100 {
		limit = 100
	}

	tx := r.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if tx.Error != nil {
			_ = tx.Rollback().Error
		}
	}()

	var tasks []*model.GenerationTask
	if err := tx.
		Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("status = ?", model.GenerationStatusPending).
		Order("created_at ASC").
		Limit(limit).
		Find(&tasks).Error; err != nil {
		_ = tx.Rollback().Error
		return nil, err
	}

	if len(tasks) == 0 {
		_ = tx.Rollback().Error
		return []*model.GenerationTask{}, nil
	}

	ids := make([]uuid.UUID, 0, len(tasks))
	for _, task := range tasks {
		if task != nil {
			ids = append(ids, task.ID)
		}
	}

	now := time.Now()
	if err := tx.Model(&model.GenerationTask{}).
		Where("id IN ?", ids).
		Updates(map[string]interface{}{
			"status":     model.GenerationStatusProcessing,
			"progress":   0,
			"updated_at": now,
		}).Error; err != nil {
		_ = tx.Rollback().Error
		return nil, err
	}

	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	for _, task := range tasks {
		if task == nil {
			continue
		}
		task.Status = model.GenerationStatusProcessing
		task.Progress = 0
		task.UpdatedAt = now
	}

	return tasks, nil
}

func (r *generationRepository) RequeueStaleProcessingTasks(ctx context.Context, staleBefore time.Time) (int64, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	result := r.db.WithContext(ctx).
		Model(&model.GenerationTask{}).
		Where("status = ? AND updated_at < ?", model.GenerationStatusProcessing, staleBefore).
		Updates(map[string]interface{}{
			"status":   model.GenerationStatusPending,
			"progress": 0,
		})
	return result.RowsAffected, result.Error
}
