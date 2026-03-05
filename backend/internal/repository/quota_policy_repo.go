package repository

import (
	"context"
	"errors"
	"strings"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// QuotaPolicyRepository 租户配额策略仓库
type QuotaPolicyRepository interface {
	Create(ctx context.Context, policy *model.QuotaPolicy) error
	Update(ctx context.Context, policy *model.QuotaPolicy) error
	Delete(ctx context.Context, id uuid.UUID) error
	GetByID(ctx context.Context, id uuid.UUID) (*model.QuotaPolicy, error)
	GetByTenantID(ctx context.Context, tenantID string) (*model.QuotaPolicy, error)
	List(ctx context.Context, keyword string, status model.QuotaPolicyStatus, page, pageSize int) ([]*model.QuotaPolicy, int64, error)
	ListActive(ctx context.Context) ([]*model.QuotaPolicy, error)
}

type quotaPolicyRepository struct {
	db *gorm.DB
}

func NewQuotaPolicyRepository(db *gorm.DB) QuotaPolicyRepository {
	return &quotaPolicyRepository{db: db}
}

func (r *quotaPolicyRepository) Create(ctx context.Context, policy *model.QuotaPolicy) error {
	return r.db.WithContext(ctx).Create(policy).Error
}

func (r *quotaPolicyRepository) Update(ctx context.Context, policy *model.QuotaPolicy) error {
	return r.db.WithContext(ctx).Save(policy).Error
}

func (r *quotaPolicyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.db.WithContext(ctx).Delete(&model.QuotaPolicy{}, "id = ?", id).Error
}

func (r *quotaPolicyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.QuotaPolicy, error) {
	var policy model.QuotaPolicy
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&policy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (r *quotaPolicyRepository) GetByTenantID(ctx context.Context, tenantID string) (*model.QuotaPolicy, error) {
	tenantID = strings.TrimSpace(tenantID)
	if tenantID == "" {
		return nil, nil
	}

	var policy model.QuotaPolicy
	err := r.db.WithContext(ctx).Where("tenant_id = ?", tenantID).First(&policy).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &policy, nil
}

func (r *quotaPolicyRepository) List(ctx context.Context, keyword string, status model.QuotaPolicyStatus, page, pageSize int) ([]*model.QuotaPolicy, int64, error) {
	var list []*model.QuotaPolicy
	var total int64

	query := r.db.WithContext(ctx).Model(&model.QuotaPolicy{})

	keyword = strings.TrimSpace(keyword)
	if keyword != "" {
		likeKeyword := "%" + keyword + "%"
		query = query.Where("tenant_id LIKE ? OR name LIKE ?", likeKeyword, likeKeyword)
	}
	if status.IsValid() {
		query = query.Where("status = ?", status)
	}

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}
	offset := (page - 1) * pageSize

	err := query.
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&list).Error
	if err != nil {
		return nil, 0, err
	}

	return list, total, nil
}

func (r *quotaPolicyRepository) ListActive(ctx context.Context) ([]*model.QuotaPolicy, error) {
	var list []*model.QuotaPolicy
	err := r.db.WithContext(ctx).
		Where("status = ?", model.QuotaPolicyStatusActive).
		Order("created_at DESC").
		Find(&list).Error
	if err != nil {
		return nil, err
	}
	return list, nil
}

