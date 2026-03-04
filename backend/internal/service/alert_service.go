package service

import (
	"fmt"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
)

// AlertService 告警服务接口
type AlertService interface {
	CreateAlert(alertType model.AlertType, severity model.AlertSeverity, title, message string, metadata model.JSON) error
	List(page, pageSize int, filters map[string]interface{}) ([]*model.AlertResponse, *model.Pagination, error)
	GetByID(id uuid.UUID) (*model.AlertResponse, error)
	Resolve(id uuid.UUID, resolvedBy uuid.UUID) error
	GetStats() (*model.AlertStats, error)
	GetActiveAlerts() ([]*model.AlertResponse, error)
	CheckChannelHealth(channelID uint, errorRate float64, latency int) error
	CheckUserBalance(userID uuid.UUID, balance float64) error
}

// alertService 告警服务实现
type alertService struct {
	alertRepo repository.AlertRepository
}

// NewAlertService 创建告警服务
func NewAlertService(alertRepo repository.AlertRepository) AlertService {
	return &alertService{
		alertRepo: alertRepo,
	}
}

// CreateAlert 创建告警
func (s *alertService) CreateAlert(alertType model.AlertType, severity model.AlertSeverity, title, message string, metadata model.JSON) error {
	alert := &model.Alert{
		Type:     alertType,
		Severity: severity,
		Status:   model.AlertStatusActive,
		Title:    title,
		Message:  message,
		Metadata: metadata,
	}
	return s.alertRepo.Create(alert)
}

// List 获取告警列表
func (s *alertService) List(page, pageSize int, filters map[string]interface{}) ([]*model.AlertResponse, *model.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	alerts, total, err := s.alertRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("获取告警列表失败: %w", err)
	}

	responses := make([]*model.AlertResponse, len(alerts))
	for i, alert := range alerts {
		responses[i] = alert.ToResponse()
	}

	pagination := model.NewPagination(page, pageSize, total)
	return responses, pagination, nil
}

// GetByID 根据 ID 获取告警
func (s *alertService) GetByID(id uuid.UUID) (*model.AlertResponse, error) {
	alert, err := s.alertRepo.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("获取告警失败: %w", err)
	}
	if alert == nil {
		return nil, fmt.Errorf("告警不存在")
	}
	return alert.ToResponse(), nil
}

// Resolve 解决告警
func (s *alertService) Resolve(id uuid.UUID, resolvedBy uuid.UUID) error {
	return s.alertRepo.Resolve(id, resolvedBy)
}

// GetStats 获取告警统计
func (s *alertService) GetStats() (*model.AlertStats, error) {
	return s.alertRepo.GetStats()
}

// GetActiveAlerts 获取活跃告警
func (s *alertService) GetActiveAlerts() ([]*model.AlertResponse, error) {
	alerts, err := s.alertRepo.GetActiveAlerts()
	if err != nil {
		return nil, fmt.Errorf("获取活跃告警失败: %w", err)
	}

	responses := make([]*model.AlertResponse, len(alerts))
	for i, alert := range alerts {
		responses[i] = alert.ToResponse()
	}
	return responses, nil
}

// CheckChannelHealth 检查渠道健康状态并创建告警
func (s *alertService) CheckChannelHealth(channelID uint, errorRate float64, latency int) error {
	// 检查错误率
	if errorRate > 0.5 { // 错误率超过50%
		metadata := model.JSON{
			"channel_id": channelID,
			"error_rate": errorRate,
			"latency":    latency,
		}
		return s.CreateAlert(
			model.AlertTypeChannelHighError,
			model.AlertSeverityCritical,
			fmt.Sprintf("渠道 %d 错误率过高", channelID),
			fmt.Sprintf("渠道错误率达到 %.2f%%，超过阈值 50%%", errorRate*100),
			metadata,
		)
	}

	// 检查延迟
	if latency > 5000 { // 延迟超过5秒
		metadata := model.JSON{
			"channel_id": channelID,
			"latency":    latency,
		}
		return s.CreateAlert(
			model.AlertTypeHighLatency,
			model.AlertSeverityWarning,
			fmt.Sprintf("渠道 %d 延迟过高", channelID),
			fmt.Sprintf("渠道延迟达到 %dms，超过阈值 5000ms", latency),
			metadata,
		)
	}

	return nil
}

// CheckUserBalance 检查用户余额并创建告警
func (s *alertService) CheckUserBalance(userID uuid.UUID, balance float64) error {
	if balance < 10.0 { // 余额低于10元
		metadata := model.JSON{
			"user_id": userID,
			"balance": balance,
		}
		return s.CreateAlert(
			model.AlertTypeLowBalance,
			model.AlertSeverityWarning,
			fmt.Sprintf("用户余额不足"),
			fmt.Sprintf("用户 %s 余额仅剩 %.2f 元", userID.String(), balance),
			metadata,
		)
	}
	return nil
}
