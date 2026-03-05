package service

import (
	"context"
	"fmt"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
)

// AuditLogService 审计日志服务
type AuditLogService interface {
	Record(ctx context.Context, input *CreateAuditLogInput) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLogResponse, *model.Pagination, error)
}

// CreateAuditLogInput 创建审计日志输入
type CreateAuditLogInput struct {
	ActorID     *uuid.UUID
	ActorRole   string
	Action      string
	Resource    string
	Method      string
	Path        string
	StatusCode  int
	Success     bool
	RequestIP   string
	UserAgent   string
	QueryParams string
	RequestBody string
	ErrorMsg    string
	Metadata    model.JSON
}

type auditLogService struct {
	repo repository.AuditLogRepository
}

func NewAuditLogService(repo repository.AuditLogRepository) AuditLogService {
	return &auditLogService{repo: repo}
}

func (s *auditLogService) Record(ctx context.Context, input *CreateAuditLogInput) error {
	if s == nil || s.repo == nil {
		return fmt.Errorf("审计日志服务未初始化")
	}
	if input == nil {
		return nil
	}

	entry := &model.AuditLog{
		ActorID:     input.ActorID,
		ActorRole:   input.ActorRole,
		Action:      input.Action,
		Resource:    input.Resource,
		Method:      input.Method,
		Path:        input.Path,
		StatusCode:  input.StatusCode,
		Success:     input.Success,
		RequestIP:   input.RequestIP,
		UserAgent:   input.UserAgent,
		QueryParams: input.QueryParams,
		RequestBody: input.RequestBody,
		ErrorMsg:    input.ErrorMsg,
		Metadata:    input.Metadata,
	}

	return s.repo.Create(entry)
}

func (s *auditLogService) List(page, pageSize int, filters map[string]interface{}) ([]*model.AuditLogResponse, *model.Pagination, error) {
	if s == nil || s.repo == nil {
		return nil, nil, fmt.Errorf("审计日志服务未初始化")
	}

	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}
	if pageSize > 200 {
		pageSize = 200
	}

	logs, total, err := s.repo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("查询审计日志失败: %w", err)
	}

	responses := make([]*model.AuditLogResponse, 0, len(logs))
	for _, item := range logs {
		responses = append(responses, item.ToResponse())
	}

	return responses, model.NewPagination(page, pageSize, total), nil
}
