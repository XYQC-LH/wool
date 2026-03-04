package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"gorm.io/gorm"
)

// UserRole 用户角色
type UserRole string

const (
	RoleUser       UserRole = "user"
	RoleAdmin      UserRole = "admin"
	RoleSuperAdmin UserRole = "super_admin"
	// 别名，用于兼容
	UserRoleUser  UserRole = "user"
	UserRoleAdmin UserRole = "admin"
)

// UserStatus 用户状态
type UserStatus string

const (
	UserStatusActive   UserStatus = "active"
	UserStatusDisabled UserStatus = "disabled"
	UserStatusBanned   UserStatus = "banned"
)

// User 用户模型
type User struct {
	ID           uuid.UUID       `gorm:"type:uuid;primary_key;default:gen_random_uuid()" json:"id"`
	Username     string          `gorm:"type:varchar(50);uniqueIndex;not null" json:"username"`
	Email        string          `gorm:"type:varchar(255);uniqueIndex;not null" json:"email"`
	PasswordHash string          `gorm:"type:varchar(255);not null" json:"-"`
	Balance      decimal.Decimal `gorm:"type:decimal(12,4);default:0" json:"balance"`
	Role         UserRole        `gorm:"type:varchar(20);default:'user'" json:"role"`
	Status       UserStatus      `gorm:"type:varchar(20);default:'active'" json:"status"`
	AvatarURL    *string         `gorm:"type:varchar(500)" json:"avatar_url,omitempty"`

	EmailNotifications bool `gorm:"not null;default:true" json:"email_notifications"`
	UsageAlerts        bool `gorm:"not null;default:true" json:"usage_alerts"`
	BillingAlerts      bool `gorm:"not null;default:true" json:"billing_alerts"`

	LastLoginAt *time.Time `json:"last_login_at,omitempty"`
	CreatedAt   time.Time  `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime" json:"updated_at"`

	// 关联
	Tokens []Token `gorm:"foreignKey:UserID" json:"tokens,omitempty"`
	Orders []Order `gorm:"foreignKey:UserID" json:"orders,omitempty"`
}

// TableName 表名
func (User) TableName() string {
	return "users"
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	if u.ID == uuid.Nil {
		u.ID = uuid.New()
	}
	return nil
}

// IsAdmin 是否是管理员
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin || u.Role == RoleSuperAdmin
}

// UserResponse 用户响应结构
type UserResponse struct {
	ID        uuid.UUID       `json:"id"`
	Username  string          `json:"username"`
	Email     string          `json:"email"`
	Balance   decimal.Decimal `json:"balance"`
	Role      UserRole        `json:"role"`
	Status    UserStatus      `json:"status"`
	AvatarURL *string         `json:"avatar_url,omitempty"`

	EmailNotifications bool `json:"email_notifications"`
	UsageAlerts        bool `json:"usage_alerts"`
	BillingAlerts      bool `json:"billing_alerts"`

	CreatedAt time.Time `json:"created_at"`
}

// ToResponse 转换为响应结构
func (u *User) ToResponse() *UserResponse {
	return &UserResponse{
		ID:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Balance:   u.Balance,
		Role:      u.Role,
		Status:    u.Status,
		AvatarURL: u.AvatarURL,

		EmailNotifications: u.EmailNotifications,
		UsageAlerts:        u.UsageAlerts,
		BillingAlerts:      u.BillingAlerts,

		CreatedAt: u.CreatedAt,
	}
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8,max=100"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	Token     string        `json:"token"`
	ExpiresAt time.Time     `json:"expires_at"`
	User      *UserResponse `json:"user"`
}

// UpdateProfileRequest 更新用户信息请求
type UpdateProfileRequest struct {
	Email     *string `json:"email,omitempty" binding:"omitempty,email"`
	Username  *string `json:"username,omitempty" binding:"omitempty,min=3,max=50"`
	AvatarURL *string `json:"avatar_url,omitempty" binding:"omitempty,url"`
}

// UpdateNotificationsRequest 更新通知设置请求
type UpdateNotificationsRequest struct {
	EmailNotifications *bool `json:"email_notifications,omitempty"`
	UsageAlerts        *bool `json:"usage_alerts,omitempty"`
	BillingAlerts      *bool `json:"billing_alerts,omitempty"`
}

// ChangePasswordRequest 修改密码请求
type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=8,max=100"`
}

// AuthResponse 认证响应
type AuthResponse struct {
	User      *UserResponse `json:"user"`
	Token     string        `json:"token"`
	ExpiresAt time.Time     `json:"expires_at"`
}

// UserDashboard 用户仪表盘数据
type UserDashboard struct {
	Balance       decimal.Decimal `json:"balance"`
	TodayRequests int64           `json:"today_requests"`
	TodayTokens   int64           `json:"today_tokens"`
	TodayCost     decimal.Decimal `json:"today_cost"`
	MonthRequests int64           `json:"month_requests"`
	MonthTokens   int64           `json:"month_tokens"`
	MonthCost     decimal.Decimal `json:"month_cost"`
}

// AdminUserResponse 管理员用户响应
type AdminUserResponse struct {
	ID          uuid.UUID       `json:"id"`
	Username    string          `json:"username"`
	Email       string          `json:"email"`
	Balance     decimal.Decimal `json:"balance"`
	Role        UserRole        `json:"role"`
	Status      UserStatus      `json:"status"`
	AvatarURL   *string         `json:"avatar_url,omitempty"`
	LastLoginAt *time.Time      `json:"last_login_at,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

// ToAdminResponse 转换为管理员响应结构
func (u *User) ToAdminResponse() *AdminUserResponse {
	return &AdminUserResponse{
		ID:          u.ID,
		Username:    u.Username,
		Email:       u.Email,
		Balance:     u.Balance,
		Role:        u.Role,
		Status:      u.Status,
		AvatarURL:   u.AvatarURL,
		LastLoginAt: u.LastLoginAt,
		CreatedAt:   u.CreatedAt,
		UpdatedAt:   u.UpdatedAt,
	}
}

// AdminCreateUserRequest 管理员创建用户请求
type AdminCreateUserRequest struct {
	Username string     `json:"username" binding:"required,min=3,max=50"`
	Email    string     `json:"email" binding:"required,email"`
	Password string     `json:"password" binding:"required,min=8,max=100"`
	Role     UserRole   `json:"role,omitempty"`
	Status   UserStatus `json:"status,omitempty"`
	Balance  *float64   `json:"balance,omitempty"`
}

// AdminUpdateUserRequest 管理员更新用户请求
type AdminUpdateUserRequest struct {
	Email    *string    `json:"email,omitempty" binding:"omitempty,email"`
	Password *string    `json:"password,omitempty" binding:"omitempty,min=8,max=100"`
	Role     UserRole   `json:"role,omitempty"`
	Status   UserStatus `json:"status,omitempty"`
	Balance  *float64   `json:"balance,omitempty"`
}

// AdminLoginRequest 管理员登录请求
type AdminLoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// AdminLoginResponse 管理员登录响应
type AdminLoginResponse struct {
	Token string             `json:"token"`
	User  *AdminUserResponse `json:"user"`
}
