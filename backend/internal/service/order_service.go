package service

import (
	"errors"
	"fmt"
	"time"

	"nexus-api/internal/cache"
	"nexus-api/internal/database"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// OrderService 订单服务接口
type OrderService interface {
	// 用户端接口
	Create(userID uuid.UUID, req *model.CreateOrderRequest) (*model.CreateOrderResponse, error)
	GetByID(userID uuid.UUID, orderID uuid.UUID) (*model.OrderResponse, error)
	GetByOrderNo(userID uuid.UUID, orderNo string) (*model.OrderResponse, error)
	List(userID uuid.UUID, page, pageSize int, status string) ([]*model.OrderResponse, *model.Pagination, error)
	GetTotalRecharge(userID uuid.UUID) (decimal.Decimal, error)
	Cancel(userID uuid.UUID, orderID uuid.UUID) error
	PayByOrderNo(userID uuid.UUID, orderNo string) error

	// 管理员接口
	AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AdminOrderResponse, *model.Pagination, error)
	AdminGetByID(orderID uuid.UUID) (*model.AdminOrderResponse, error)
	UpdateStatus(orderID uuid.UUID, status model.OrderStatus) error

	// 支付回调
	HandlePaymentCallback(orderNo string, paymentInfo map[string]interface{}) error
}

// orderService 订单服务实现
type orderService struct {
	orderRepo repository.OrderRepository
	userRepo  repository.UserRepository
}

// NewOrderService 创建订单服务
func NewOrderService(orderRepo repository.OrderRepository, userRepo repository.UserRepository) OrderService {
	return &orderService{
		orderRepo: orderRepo,
		userRepo:  userRepo,
	}
}

// generateOrderNo 生成订单号
func generateOrderNo() string {
	return fmt.Sprintf("ORD%s%d", time.Now().Format("20060102150405"), time.Now().UnixNano()%10000)
}

// Create 创建订单
func (s *orderService) Create(userID uuid.UUID, req *model.CreateOrderRequest) (*model.CreateOrderResponse, error) {
	// 验证金额
	if req.Amount <= 0 {
		return nil, errors.New("充值金额必须大于0")
	}

	// 设置默认货币
	currency := req.Currency
	if currency == "" {
		currency = "USD"
	}

	// 创建订单
	order := &model.Order{
		UserID:        userID,
		OrderNo:       generateOrderNo(),
		Amount:        decimal.NewFromFloat(req.Amount),
		Currency:      currency,
		PaymentMethod: req.PaymentMethod,
		Status:        model.OrderStatusPending,
	}

	if err := s.orderRepo.Create(order); err != nil {
		return nil, fmt.Errorf("创建订单失败: %w", err)
	}

	// 生成支付链接（这里需要根据实际支付方式生成）
	paymentURL := s.generatePaymentURL(order)

	return &model.CreateOrderResponse{
		OrderID:    order.ID,
		OrderNo:    order.OrderNo,
		Amount:     order.Amount,
		Currency:   order.Currency,
		PaymentURL: paymentURL,
		ExpiresAt:  time.Now().Add(30 * time.Minute), // 30分钟过期
		CreatedAt:  order.CreatedAt,
	}, nil
}

// generatePaymentURL 生成支付链接
func (s *orderService) generatePaymentURL(order *model.Order) string {
	// 当前返回前端支付页面路由；真实支付请接入支付渠道（如 Stripe Webhook）并回调确认。
	switch order.PaymentMethod {
	case model.PaymentMethodStripe:
		return fmt.Sprintf("/pay/stripe?order_no=%s", order.OrderNo)
	case model.PaymentMethodCrypto:
		return fmt.Sprintf("/pay/crypto?order_no=%s", order.OrderNo)
	default:
		return fmt.Sprintf("/pay?order_no=%s", order.OrderNo)
	}
}

// GetByID 获取订单详情
func (s *orderService) GetByID(userID uuid.UUID, orderID uuid.UUID) (*model.OrderResponse, error) {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}
	if order == nil {
		return nil, errors.New("订单不存在")
	}

	// 验证订单所有权
	if order.UserID != userID {
		return nil, errors.New("无权访问此订单")
	}

	return order.ToResponse(), nil
}

// GetByOrderNo 根据订单号获取订单详情
func (s *orderService) GetByOrderNo(userID uuid.UUID, orderNo string) (*model.OrderResponse, error) {
	if orderNo == "" {
		return nil, errors.New("订单号不能为空")
	}

	order, err := s.orderRepo.GetByOrderNo(orderNo)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}
	if order == nil {
		return nil, errors.New("订单不存在")
	}

	// 验证订单所有权
	if order.UserID != userID {
		return nil, errors.New("无权访问此订单")
	}

	return order.ToResponse(), nil
}

// List 获取用户订单列表
func (s *orderService) List(userID uuid.UUID, page, pageSize int, status string) ([]*model.OrderResponse, *model.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	filters := make(map[string]interface{})
	filters["user_id"] = userID
	if status != "" {
		filters["status"] = status
	}

	orders, total, err := s.orderRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("获取订单列表失败: %w", err)
	}

	// 转换为响应格式
	responses := make([]*model.OrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = order.ToResponse()
	}

	pagination := model.NewPagination(page, pageSize, total)

	return responses, pagination, nil
}

// GetTotalRecharge 获取用户总充值金额（已支付）
func (s *orderService) GetTotalRecharge(userID uuid.UUID) (decimal.Decimal, error) {
	total, err := s.orderRepo.GetTotalPaidAmountByUserID(userID)
	if err != nil {
		return decimal.Zero, fmt.Errorf("获取总充值金额失败: %w", err)
	}
	return total, nil
}

// Cancel 取消订单
func (s *orderService) Cancel(userID uuid.UUID, orderID uuid.UUID) error {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return fmt.Errorf("获取订单失败: %w", err)
	}
	if order == nil {
		return errors.New("订单不存在")
	}

	// 验证订单所有权
	if order.UserID != userID {
		return errors.New("无权操作此订单")
	}

	// 只有待支付的订单可以取消
	if order.Status != model.OrderStatusPending {
		return errors.New("只有待支付的订单可以取消")
	}

	return s.orderRepo.UpdateStatus(orderID, model.OrderStatusCancelled)
}

// PayByOrderNo 支付订单（测试/手动确认）
func (s *orderService) PayByOrderNo(userID uuid.UUID, orderNo string) error {
	if orderNo == "" {
		return errors.New("订单号不能为空")
	}

	var paidUserID uuid.UUID
	err := database.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", orderNo).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("订单不存在")
			}
			return fmt.Errorf("获取订单失败: %w", err)
		}

		paidUserID = order.UserID

		if order.UserID != userID {
			return errors.New("无权操作此订单")
		}

		// 幂等处理：已支付直接返回成功
		if order.Status == model.OrderStatusPaid {
			return nil
		}

		// 只有待支付的订单可以支付
		if order.Status != model.OrderStatusPending {
			return errors.New("只有待支付的订单可以支付")
		}

		now := time.Now()
		if err := tx.Model(&model.Order{}).
			Where("id = ? AND status = ?", order.ID, model.OrderStatusPending).
			Updates(map[string]interface{}{
				"status":  model.OrderStatusPaid,
				"paid_at": now,
			}).Error; err != nil {
			return fmt.Errorf("更新订单状态失败: %w", err)
		}

		if err := tx.Model(&model.User{}).
			Where("id = ?", order.UserID).
			Update("balance", gorm.Expr("balance + ?", order.Amount)).Error; err != nil {
			return fmt.Errorf("更新用户余额失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if paidUserID != uuid.Nil {
		_ = cache.Delete(cache.KeyUserPrefix + paidUserID.String())
	}

	return nil
}

// AdminList 管理员获取订单列表
func (s *orderService) AdminList(page, pageSize int, filters map[string]interface{}) ([]*model.AdminOrderResponse, *model.Pagination, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 {
		pageSize = 20
	}

	orders, total, err := s.orderRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, fmt.Errorf("获取订单列表失败: %w", err)
	}

	// 转换为管理员响应格式
	responses := make([]*model.AdminOrderResponse, len(orders))
	for i, order := range orders {
		responses[i] = order.ToAdminResponse()
	}

	pagination := model.NewPagination(page, pageSize, total)

	return responses, pagination, nil
}

// AdminGetByID 管理员获取订单详情
func (s *orderService) AdminGetByID(orderID uuid.UUID) (*model.AdminOrderResponse, error) {
	order, err := s.orderRepo.GetByID(orderID)
	if err != nil {
		return nil, fmt.Errorf("获取订单失败: %w", err)
	}
	if order == nil {
		return nil, errors.New("订单不存在")
	}

	return order.ToAdminResponse(), nil
}

// UpdateStatus 更新订单状态
func (s *orderService) UpdateStatus(orderID uuid.UUID, status model.OrderStatus) error {
	return s.orderRepo.UpdateStatus(orderID, status)
}

// HandlePaymentCallback 处理支付回调
func (s *orderService) HandlePaymentCallback(orderNo string, paymentInfo map[string]interface{}) error {
	if orderNo == "" {
		return errors.New("订单号不能为空")
	}

	var paidUserID uuid.UUID
	err := database.Transaction(func(tx *gorm.DB) error {
		var order model.Order
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("order_no = ?", orderNo).
			First(&order).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return errors.New("订单不存在")
			}
			return fmt.Errorf("获取订单失败: %w", err)
		}

		paidUserID = order.UserID

		// 幂等处理：已支付直接返回成功
		if order.Status == model.OrderStatusPaid {
			return nil
		}
		if order.Status != model.OrderStatusPending {
			return errors.New("订单状态不正确")
		}

		now := time.Now()
		updates := map[string]interface{}{
			"status":  model.OrderStatusPaid,
			"paid_at": now,
		}
		if paymentInfo != nil {
			updates["payment_info"] = model.JSON(paymentInfo)
		}

		if err := tx.Model(&model.Order{}).
			Where("id = ? AND status = ?", order.ID, model.OrderStatusPending).
			Updates(updates).Error; err != nil {
			return fmt.Errorf("更新订单状态失败: %w", err)
		}

		if err := tx.Model(&model.User{}).
			Where("id = ?", order.UserID).
			Update("balance", gorm.Expr("balance + ?", order.Amount)).Error; err != nil {
			return fmt.Errorf("更新用户余额失败: %w", err)
		}

		return nil
	})
	if err != nil {
		return err
	}

	if paidUserID != uuid.Nil {
		_ = cache.Delete(cache.KeyUserPrefix + paidUserID.String())
	}

	return nil
}
