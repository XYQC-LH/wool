package service

import (
	"errors"
	"time"

	"nexus-api/internal/auth"
	"nexus-api/internal/config"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"golang.org/x/crypto/bcrypt"
)

// UserService 用户服务接口
type UserService interface {
	Register(req *model.RegisterRequest) (*model.User, error)
	Login(req *model.LoginRequest) (*model.LoginResponse, error)
	AdminLogin(username, password string) (*model.User, string, error)
	GetProfile(userID uuid.UUID) (*model.UserResponse, error)
	UpdateProfile(userID uuid.UUID, req *model.UpdateProfileRequest) (*model.UserResponse, error)
	UpdateNotifications(userID uuid.UUID, req *model.UpdateNotificationsRequest) (*model.UserResponse, error)
	ChangePassword(userID uuid.UUID, req *model.ChangePasswordRequest) error
	GetDashboard(userID uuid.UUID) (*model.UserDashboard, error)
	List(page, pageSize int, filters map[string]interface{}) ([]*model.AdminUserResponse, *model.Pagination, error)
	GetByID(id uuid.UUID) (*model.AdminUserResponse, error)
	Create(req *model.AdminCreateUserRequest) (*model.User, error)
	Update(id uuid.UUID, req *model.AdminUpdateUserRequest) (*model.AdminUserResponse, error)
	Delete(id uuid.UUID) error
	UpdateBalance(id uuid.UUID, amount decimal.Decimal) error
	GetStats() (*UserStats, error)
}

// userService 用户服务实现
type userService struct {
	userRepo repository.UserRepository
	logRepo  repository.LogRepository
	cfg      *config.Config
}

// NewUserService 创建用户服务
func NewUserService(userRepo repository.UserRepository, logRepo repository.LogRepository, cfg *config.Config) UserService {
	return &userService{
		userRepo: userRepo,
		logRepo:  logRepo,
		cfg:      cfg,
	}
}

// UserStats 用户统计
type UserStats struct {
	TotalUsers    int64 `json:"total_users"`
	ActiveUsers   int64 `json:"active_users"`
	NewUsersToday int64 `json:"new_users_today"`
}

// Register 用户注册
func (s *userService) Register(req *model.RegisterRequest) (*model.User, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUser, err = s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 创建用户
	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         model.UserRoleUser,
		Status:       model.UserStatusActive,
		Balance:      decimal.Zero,

		EmailNotifications: true,
		UsageAlerts:        true,
		BillingAlerts:      true,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// Login 用户登录
func (s *userService) Login(req *model.LoginRequest) (*model.LoginResponse, error) {
	// 查找用户
	user, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 检查用户状态
	if user.Status != model.UserStatusActive {
		return nil, errors.New("账户已被禁用")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)); err != nil {
		return nil, errors.New("用户名或密码错误")
	}

	// 生成 JWT
	token, expiresAt, err := auth.GenerateJWT(user, s.cfg)
	if err != nil {
		return nil, err
	}

	// 更新最后登录时间
	_ = s.userRepo.UpdateLastLogin(user.ID)

	return &model.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      user.ToResponse(),
	}, nil
}

// AdminLogin 管理员登录
func (s *userService) AdminLogin(username, password string) (*model.User, string, error) {
	// 查找用户
	user, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, "", err
	}
	if user == nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	// 检查是否是管理员
	if !user.IsAdmin() {
		return nil, "", errors.New("无管理员权限")
	}

	// 检查用户状态
	if user.Status != model.UserStatusActive {
		return nil, "", errors.New("账户已被禁用")
	}

	// 验证密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", errors.New("用户名或密码错误")
	}

	// 生成 JWT
	token, _, err := auth.GenerateJWT(user, s.cfg)
	if err != nil {
		return nil, "", err
	}

	// 更新最后登录时间
	_ = s.userRepo.UpdateLastLogin(user.ID)

	return user, token, nil
}

// GetProfile 获取用户资料
func (s *userService) GetProfile(userID uuid.UUID) (*model.UserResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	return user.ToResponse(), nil
}

// UpdateProfile 更新用户资料
func (s *userService) UpdateProfile(userID uuid.UUID, req *model.UpdateProfileRequest) (*model.UserResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 更新邮箱
	if req.Email != nil && *req.Email != user.Email {
		// 检查邮箱是否已被使用
		existingUser, err := s.userRepo.GetByEmail(*req.Email)
		if err != nil {
			return nil, err
		}
		if existingUser != nil && existingUser.ID != userID {
			return nil, errors.New("邮箱已被使用")
		}
		user.Email = *req.Email
	}

	// 更新用户名
	if req.Username != nil && *req.Username != user.Username {
		existingUser, err := s.userRepo.GetByUsername(*req.Username)
		if err != nil {
			return nil, err
		}
		if existingUser != nil && existingUser.ID != userID {
			return nil, errors.New("用户名已存在")
		}
		user.Username = *req.Username
	}

	// 更新头像
	if req.AvatarURL != nil {
		user.AvatarURL = req.AvatarURL
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

// UpdateNotifications 更新通知设置
func (s *userService) UpdateNotifications(userID uuid.UUID, req *model.UpdateNotificationsRequest) (*model.UserResponse, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	if req.EmailNotifications != nil {
		user.EmailNotifications = *req.EmailNotifications
	}
	if req.UsageAlerts != nil {
		user.UsageAlerts = *req.UsageAlerts
	}
	if req.BillingAlerts != nil {
		user.BillingAlerts = *req.BillingAlerts
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user.ToResponse(), nil
}

// ChangePassword 修改密码
func (s *userService) ChangePassword(userID uuid.UUID, req *model.ChangePasswordRequest) error {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	// 验证旧密码
	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.OldPassword)); err != nil {
		return errors.New("旧密码错误")
	}

	// 加密新密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.PasswordHash = string(hashedPassword)
	return s.userRepo.Update(user)
}

// GetDashboard 获取用户仪表盘数据
func (s *userService) GetDashboard(userID uuid.UUID) (*model.UserDashboard, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 获取今日统计
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	todayStats, err := s.logRepo.GetUserStats(userID, startOfDay, endOfDay)
	if err != nil {
		return nil, err
	}

	// 获取本月统计
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthStats, err := s.logRepo.GetUserStats(userID, startOfMonth, endOfDay)
	if err != nil {
		return nil, err
	}

	return &model.UserDashboard{
		Balance:       user.Balance,
		TodayRequests: todayStats.TotalRequests,
		TodayTokens:   todayStats.TotalTokens,
		TodayCost:     todayStats.TotalCost,
		MonthRequests: monthStats.TotalRequests,
		MonthTokens:   monthStats.TotalTokens,
		MonthCost:     monthStats.TotalCost,
	}, nil
}

// List 获取用户列表（管理员）
func (s *userService) List(page, pageSize int, filters map[string]interface{}) ([]*model.AdminUserResponse, *model.Pagination, error) {
	users, total, err := s.userRepo.List(page, pageSize, filters)
	if err != nil {
		return nil, nil, err
	}

	var responses []*model.AdminUserResponse
	for _, user := range users {
		responses = append(responses, user.ToAdminResponse())
	}

	pagination := model.NewPagination(page, pageSize, total)
	return responses, pagination, nil
}

// GetByID 根据 ID 获取用户（管理员）
func (s *userService) GetByID(id uuid.UUID) (*model.AdminUserResponse, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	return user.ToAdminResponse(), nil
}

// Create 创建用户（管理员）
func (s *userService) Create(req *model.AdminCreateUserRequest) (*model.User, error) {
	// 检查用户名是否已存在
	existingUser, err := s.userRepo.GetByUsername(req.Username)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("用户名已存在")
	}

	// 检查邮箱是否已存在
	existingUser, err = s.userRepo.GetByEmail(req.Email)
	if err != nil {
		return nil, err
	}
	if existingUser != nil {
		return nil, errors.New("邮箱已被注册")
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	// 设置默认值
	role := model.UserRoleUser
	if req.Role != "" {
		role = req.Role
	}

	status := model.UserStatusActive
	if req.Status != "" {
		status = req.Status
	}

	balance := decimal.Zero
	if req.Balance != nil {
		balance = decimal.NewFromFloat(*req.Balance)
	}

	// 创建用户
	user := &model.User{
		Username:     req.Username,
		Email:        req.Email,
		PasswordHash: string(hashedPassword),
		Role:         role,
		Status:       status,
		Balance:      balance,
	}

	if err := s.userRepo.Create(user); err != nil {
		return nil, err
	}

	return user, nil
}

// Update 更新用户（管理员）
func (s *userService) Update(id uuid.UUID, req *model.AdminUpdateUserRequest) (*model.AdminUserResponse, error) {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 更新邮箱
	if req.Email != nil && *req.Email != user.Email {
		existingUser, err := s.userRepo.GetByEmail(*req.Email)
		if err != nil {
			return nil, err
		}
		if existingUser != nil && existingUser.ID != id {
			return nil, errors.New("邮箱已被使用")
		}
		user.Email = *req.Email
	}

	// 更新密码
	if req.Password != nil {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(*req.Password), bcrypt.DefaultCost)
		if err != nil {
			return nil, err
		}
		user.PasswordHash = string(hashedPassword)
	}

	// 更新角色
	if req.Role != "" {
		user.Role = req.Role
	}

	// 更新状态
	if req.Status != "" {
		user.Status = req.Status
	}

	// 更新余额
	if req.Balance != nil {
		user.Balance = decimal.NewFromFloat(*req.Balance)
	}

	if err := s.userRepo.Update(user); err != nil {
		return nil, err
	}

	return user.ToAdminResponse(), nil
}

// Delete 删除用户（管理员）
func (s *userService) Delete(id uuid.UUID) error {
	user, err := s.userRepo.GetByID(id)
	if err != nil {
		return err
	}
	if user == nil {
		return errors.New("用户不存在")
	}

	return s.userRepo.Delete(id)
}

// UpdateBalance 更新余额
func (s *userService) UpdateBalance(id uuid.UUID, amount decimal.Decimal) error {
	return s.userRepo.UpdateBalance(id, amount)
}

// GetStats 获取用户统计
func (s *userService) GetStats() (*UserStats, error) {
	totalUsers, err := s.userRepo.CountByStatus(model.UserStatusActive)
	if err != nil {
		return nil, err
	}

	// 获取最近24小时活跃用户
	activeUsers, err := s.userRepo.GetActiveUsersCount(time.Now().Add(-24 * time.Hour))
	if err != nil {
		return nil, err
	}

	return &UserStats{
		TotalUsers:  totalUsers,
		ActiveUsers: activeUsers,
	}, nil
}
