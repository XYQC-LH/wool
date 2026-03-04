package repository

import (
	"errors"
	"time"

	"nexus-api/internal/model"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// OrderRepository Order 仓库接口
type OrderRepository interface {
	Create(order *model.Order) error
	GetByID(id uuid.UUID) (*model.Order, error)
	GetByOrderNo(orderNo string) (*model.Order, error)
	Update(order *model.Order) error
	ListByUserID(userID uuid.UUID, page, pageSize int) ([]*model.Order, int64, error)
	List(page, pageSize int, filters map[string]interface{}) ([]*model.Order, int64, error)
	UpdateStatus(id uuid.UUID, status model.OrderStatus) error
	MarkAsPaid(id uuid.UUID) error
	GetTotalPaidAmountByUserID(userID uuid.UUID) (decimal.Decimal, error)
	GetTotalRevenue(startDate, endDate time.Time) (decimal.Decimal, error)
	CountByStatus(status model.OrderStatus) (int64, error)
}

// orderRepository Order 仓库实现
type orderRepository struct {
	db *gorm.DB
}

// NewOrderRepository 创建 Order 仓库
func NewOrderRepository(db *gorm.DB) OrderRepository {
	return &orderRepository{db: db}
}

// Create 创建订单
func (r *orderRepository) Create(order *model.Order) error {
	return r.db.Create(order).Error
}

// GetByID 根据 ID 获取订单
func (r *orderRepository) GetByID(id uuid.UUID) (*model.Order, error) {
	var order model.Order
	if err := r.db.Preload("User").Where("id = ?", id).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// GetByOrderNo 根据订单号获取订单
func (r *orderRepository) GetByOrderNo(orderNo string) (*model.Order, error) {
	var order model.Order
	if err := r.db.Preload("User").Where("order_no = ?", orderNo).First(&order).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &order, nil
}

// Update 更新订单
func (r *orderRepository) Update(order *model.Order) error {
	return r.db.Save(order).Error
}

// ListByUserID 获取用户的订单列表
func (r *orderRepository) ListByUserID(userID uuid.UUID, page, pageSize int) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	query := r.db.Model(&model.Order{}).Where("user_id = ?", userID)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// List 获取订单列表（管理员）
func (r *orderRepository) List(page, pageSize int, filters map[string]interface{}) ([]*model.Order, int64, error) {
	var orders []*model.Order
	var total int64

	query := r.db.Model(&model.Order{})

	// 应用过滤条件
	if userID, ok := filters["user_id"]; ok {
		query = query.Where("user_id = ?", userID)
	}
	if status, ok := filters["status"]; ok {
		query = query.Where("status = ?", status)
	}
	if paymentMethod, ok := filters["payment_method"]; ok {
		query = query.Where("payment_method = ?", paymentMethod)
	}
	if startDate, ok := filters["start_date"]; ok {
		query = query.Where("created_at >= ?", startDate)
	}
	if endDate, ok := filters["end_date"]; ok {
		query = query.Where("created_at <= ?", endDate)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("User").Offset(offset).Limit(pageSize).Order("created_at DESC").Find(&orders).Error; err != nil {
		return nil, 0, err
	}

	return orders, total, nil
}

// UpdateStatus 更新订单状态
func (r *orderRepository) UpdateStatus(id uuid.UUID, status model.OrderStatus) error {
	return r.db.Model(&model.Order{}).Where("id = ?", id).Update("status", status).Error
}

// MarkAsPaid 标记为已支付
func (r *orderRepository) MarkAsPaid(id uuid.UUID) error {
	now := time.Now()
	return r.db.Model(&model.Order{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":  model.OrderStatusPaid,
			"paid_at": now,
		}).Error
}

// GetTotalPaidAmountByUserID 获取用户总充值金额（已支付）
func (r *orderRepository) GetTotalPaidAmountByUserID(userID uuid.UUID) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}

	err := r.db.Model(&model.Order{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("user_id = ? AND status = ?", userID, model.OrderStatusPaid).
		Scan(&result).Error

	return result.Total, err
}

// GetTotalRevenue 获取总收入
func (r *orderRepository) GetTotalRevenue(startDate, endDate time.Time) (decimal.Decimal, error) {
	var result struct {
		Total decimal.Decimal
	}

	err := r.db.Model(&model.Order{}).
		Select("COALESCE(SUM(amount), 0) as total").
		Where("status = ? AND paid_at >= ? AND paid_at <= ?", model.OrderStatusPaid, startDate, endDate).
		Scan(&result).Error

	return result.Total, err
}

// CountByStatus 按状态统计订单数
func (r *orderRepository) CountByStatus(status model.OrderStatus) (int64, error) {
	var count int64
	err := r.db.Model(&model.Order{}).Where("status = ?", status).Count(&count).Error
	return count, err
}
