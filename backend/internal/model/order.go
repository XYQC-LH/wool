package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// OrderStatus 订单状态
type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "pending"
	OrderStatusPaid      OrderStatus = "paid"
	OrderStatusFailed    OrderStatus = "failed"
	OrderStatusRefunded  OrderStatus = "refunded"
	OrderStatusCancelled OrderStatus = "cancelled"
)

// PaymentMethod 支付方式
type PaymentMethod string

const (
	PaymentMethodStripe PaymentMethod = "stripe"
	PaymentMethodCrypto PaymentMethod = "crypto"
)

// Order 订单模型
type Order struct {
	ID            uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	UserID        uuid.UUID       `gorm:"type:uuid;not null;index" json:"user_id"`
	OrderNo       string          `gorm:"type:varchar(50);uniqueIndex;not null" json:"order_no"`
	Amount        decimal.Decimal `gorm:"type:decimal(12,2);not null" json:"amount"`
	Currency      string          `gorm:"type:varchar(10);default:'USD'" json:"currency"`
	PaymentMethod PaymentMethod   `gorm:"type:varchar(50);not null" json:"payment_method"`
	Status        OrderStatus     `gorm:"type:varchar(20);default:'pending'" json:"status"`
	PaymentInfo   JSON            `gorm:"type:jsonb;default:'{}'" json:"payment_info,omitempty"`
	PaidAt        *time.Time      `json:"paid_at,omitempty"`
	CreatedAt     time.Time       `gorm:"autoCreateTime" json:"created_at"`

	// 关联
	User *User `gorm:"foreignKey:UserID" json:"user,omitempty"`
}

// TableName 表名
func (Order) TableName() string {
	return "orders"
}

// BeforeCreate 创建前钩子
func (o *Order) BeforeCreate(tx *gorm.DB) error {
	if o.ID == uuid.Nil {
		o.ID = uuid.New()
	}
	return nil
}

// IsPaid 是否已支付
func (o *Order) IsPaid() bool {
	return o.Status == OrderStatusPaid
}

// OrderResponse 订单响应结构
type OrderResponse struct {
	ID            uuid.UUID       `json:"id"`
	OrderNo       string          `json:"order_no"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	PaymentMethod PaymentMethod   `json:"payment_method"`
	Status        OrderStatus     `json:"status"`
	PaidAt        *time.Time      `json:"paid_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ToResponse 转换为响应结构
func (o *Order) ToResponse() *OrderResponse {
	return &OrderResponse{
		ID:            o.ID,
		OrderNo:       o.OrderNo,
		Amount:        o.Amount,
		Currency:      o.Currency,
		PaymentMethod: o.PaymentMethod,
		Status:        o.Status,
		PaidAt:        o.PaidAt,
		CreatedAt:     o.CreatedAt,
	}
}

// CreateOrderResponse 创建订单响应
type CreateOrderResponse struct {
	OrderID    uuid.UUID       `json:"order_id"`
	OrderNo    string          `json:"order_no"`
	Amount     decimal.Decimal `json:"amount"`
	Currency   string          `json:"currency"`
	PaymentURL string          `json:"payment_url"`
	ExpiresAt  time.Time       `json:"expires_at"`
	CreatedAt  time.Time       `json:"created_at"`
}

// CreateOrderRequest 创建订单请求
type CreateOrderRequest struct {
	Amount        float64       `json:"amount" binding:"required,min=1"`
	Currency      string        `json:"currency" binding:"omitempty,oneof=USD CNY EUR"`
	PaymentMethod PaymentMethod `json:"payment_method" binding:"required,oneof=stripe crypto"`
}

// AdminOrderResponse 管理员订单响应
type AdminOrderResponse struct {
	ID            uuid.UUID       `json:"id"`
	UserID        uuid.UUID       `json:"user_id"`
	Username      string          `json:"username,omitempty"`
	Email         string          `json:"email,omitempty"`
	OrderNo       string          `json:"order_no"`
	Amount        decimal.Decimal `json:"amount"`
	Currency      string          `json:"currency"`
	PaymentMethod PaymentMethod   `json:"payment_method"`
	Status        OrderStatus     `json:"status"`
	PaidAt        *time.Time      `json:"paid_at,omitempty"`
	CreatedAt     time.Time       `json:"created_at"`
}

// ToAdminResponse 转换为管理员响应结构
func (o *Order) ToAdminResponse() *AdminOrderResponse {
	resp := &AdminOrderResponse{
		ID:            o.ID,
		UserID:        o.UserID,
		OrderNo:       o.OrderNo,
		Amount:        o.Amount,
		Currency:      o.Currency,
		PaymentMethod: o.PaymentMethod,
		Status:        o.Status,
		PaidAt:        o.PaidAt,
		CreatedAt:     o.CreatedAt,
	}

	if o.User != nil {
		resp.Username = o.User.Username
		resp.Email = o.User.Email
	}

	return resp
}
