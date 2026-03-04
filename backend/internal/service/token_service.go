package service

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// TokenService Token 服务接口
type TokenService interface {
	Create(userID uuid.UUID, req *model.CreateTokenRequest) (*model.Token, error)
	GetByID(userID uuid.UUID, id uuid.UUID) (*model.TokenResponse, error)
	GetByKey(key string) (*model.Token, error)
	List(userID uuid.UUID, page, pageSize int) ([]*model.TokenResponse, *model.Pagination, error)
	Update(userID uuid.UUID, id uuid.UUID, req *model.UpdateTokenRequest) (*model.TokenResponse, error)
	Delete(userID uuid.UUID, id uuid.UUID) error
	DeductQuota(id uuid.UUID, amount decimal.Decimal) error
	UpdateLastUsed(id uuid.UUID) error
	ValidateToken(key string) (*model.Token, error)
}

// tokenService Token 服务实现
type tokenService struct {
	tokenRepo repository.TokenRepository
	userRepo  repository.UserRepository
	logRepo   repository.LogRepository
}

// NewTokenService 创建 Token 服务
func NewTokenService(tokenRepo repository.TokenRepository, userRepo repository.UserRepository, logRepo repository.LogRepository) TokenService {
	return &tokenService{
		tokenRepo: tokenRepo,
		userRepo:  userRepo,
		logRepo:   logRepo,
	}
}

// generateAPIKey 生成 API Key
func generateAPIKey() (string, error) {
	bytes := make([]byte, 24)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return "sk-" + hex.EncodeToString(bytes), nil
}

// Create 创建 Token
func (s *tokenService) Create(userID uuid.UUID, req *model.CreateTokenRequest) (*model.Token, error) {
	// 检查用户是否存在
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, errors.New("用户不存在")
	}

	// 检查 Token 数量限制
	count, err := s.tokenRepo.CountByUserID(userID)
	if err != nil {
		return nil, err
	}
	if count >= 10 {
		return nil, errors.New("Token 数量已达上限")
	}

	// 生成 API Key
	key, err := generateAPIKey()
	if err != nil {
		return nil, err
	}

	// 创建 Token
	token := &model.Token{
		Key:    key,
		Name:   req.Name,
		UserID: userID,
		Status: model.TokenStatusActive,
	}

	// 设置配额
	if req.Quota != nil {
		quota := decimal.NewFromFloat(*req.Quota)
		token.RemainQuota = &quota
	}

	// 设置过期时间
	if req.ExpiresAt != nil {
		token.ExpiresAt = req.ExpiresAt
	}

	// 设置限流
	if req.RateLimit != nil {
		token.RateLimit = req.RateLimit
	}

	// 设置模型白名单
	if len(req.AllowedModels) > 0 {
		token.AllowedModels = req.AllowedModels
	}

	// 设置 IP 白名单
	if len(req.AllowedIPs) > 0 {
		token.AllowedIPs = req.AllowedIPs
	}

	if err := s.tokenRepo.Create(token); err != nil {
		return nil, err
	}

	return token, nil
}

// GetByID 根据 ID 获取 Token
func (s *tokenService) GetByID(userID uuid.UUID, id uuid.UUID) (*model.TokenResponse, error) {
	token, err := s.tokenRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.New("Token 不存在")
	}
	if token.UserID != userID {
		return nil, errors.New("无权访问此 Token")
	}

	return token.ToResponse(), nil
}

// GetByKey 根据 Key 获取 Token
func (s *tokenService) GetByKey(key string) (*model.Token, error) {
	return s.tokenRepo.GetByKey(key)
}

// List 获取 Token 列表
func (s *tokenService) List(userID uuid.UUID, page, pageSize int) ([]*model.TokenResponse, *model.Pagination, error) {
	tokens, total, err := s.tokenRepo.ListByUserID(userID, page, pageSize)
	if err != nil {
		return nil, nil, err
	}

	// 用量统计（当前页 Token 批量聚合，避免 N+1）
	tokenIDs := make([]uuid.UUID, 0, len(tokens))
	for _, t := range tokens {
		tokenIDs = append(tokenIDs, t.ID)
	}

	usageByTokenID := make(map[uuid.UUID]*model.TokenUsageStats, len(tokens))
	if len(tokenIDs) > 0 {
		usageStats, usageErr := s.logRepo.GetTokenUsageStatsByTokenIDs(userID, tokenIDs)
		if usageErr != nil {
			return nil, nil, usageErr
		}
		for _, stat := range usageStats {
			usageByTokenID[stat.TokenID] = stat
		}
	}

	var responses []*model.TokenResponse
	for _, token := range tokens {
		resp := token.ToResponse()
		if usage, ok := usageByTokenID[token.ID]; ok {
			resp.Usage = usage
		}
		responses = append(responses, resp)
	}

	pagination := model.NewPagination(page, pageSize, total)
	return responses, pagination, nil
}

// Update 更新 Token
func (s *tokenService) Update(userID uuid.UUID, id uuid.UUID, req *model.UpdateTokenRequest) (*model.TokenResponse, error) {
	token, err := s.tokenRepo.GetByID(id)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.New("Token 不存在")
	}

	// 检查所有权
	if token.UserID != userID {
		return nil, errors.New("无权操作此 Token")
	}

	// 更新名称
	if req.Name != nil {
		token.Name = *req.Name
	}

	// 更新状态
	if req.Status != "" {
		token.Status = req.Status
	}

	// 更新配额
	if req.Quota != nil {
		quota := decimal.NewFromFloat(*req.Quota)
		token.RemainQuota = &quota
	}

	// 更新过期时间
	if req.ExpiresAt != nil {
		token.ExpiresAt = req.ExpiresAt
	}

	// 更新限流
	if req.RateLimit != nil {
		token.RateLimit = req.RateLimit
	}

	// 更新模型白名单（允许设置为空数组以清空限制）
	if req.AllowedModels != nil {
		token.AllowedModels = *req.AllowedModels
	}

	// 更新 IP 白名单（允许设置为空数组以清空限制）
	if req.AllowedIPs != nil {
		token.AllowedIPs = *req.AllowedIPs
	}

	if err := s.tokenRepo.Update(token); err != nil {
		return nil, err
	}

	return token.ToResponse(), nil
}

// Delete 删除 Token
func (s *tokenService) Delete(userID uuid.UUID, id uuid.UUID) error {
	token, err := s.tokenRepo.GetByID(id)
	if err != nil {
		return err
	}
	if token == nil {
		return errors.New("Token 不存在")
	}

	// 检查所有权
	if token.UserID != userID {
		return errors.New("无权操作此 Token")
	}

	return s.tokenRepo.Delete(id)
}

// DeductQuota 扣除配额
func (s *tokenService) DeductQuota(id uuid.UUID, amount decimal.Decimal) error {
	err := s.tokenRepo.DeductQuota(id, amount)
	if err == nil {
		return nil
	}
	if errors.Is(err, repository.ErrInsufficientQuota) {
		// 这里无法无成本拿到实时 remaining（且多处调用方本来就忽略 deduct 错误），
		// 但仍返回强类型错误，避免上层再做字符串判断。
		return &QuotaExceededError{Needed: amount, Remaining: decimal.Zero}
	}
	return err
}

// UpdateLastUsed 更新最后使用时间
func (s *tokenService) UpdateLastUsed(id uuid.UUID) error {
	return s.tokenRepo.UpdateLastUsed(id)
}

// ValidateToken 验证 Token
func (s *tokenService) ValidateToken(key string) (*model.Token, error) {
	token, err := s.tokenRepo.GetByKeyWithUser(key)
	if err != nil {
		return nil, err
	}
	if token == nil {
		return nil, errors.New("无效的 API Key")
	}

	// 检查状态
	if token.Status != model.TokenStatusActive {
		return nil, errors.New("API Key 已被禁用")
	}

	// 检查是否过期
	if token.ExpiresAt != nil && time.Now().After(*token.ExpiresAt) {
		return nil, errors.New("API Key 已过期")
	}

	// 检查配额
	if token.RemainQuota != nil && token.RemainQuota.LessThanOrEqual(decimal.Zero) {
		return nil, errors.New("API Key 配额已用尽")
	}

	// 检查用户状态
	if token.User != nil && token.User.Status != model.UserStatusActive {
		return nil, errors.New("用户账户已被禁用")
	}

	return token, nil
}
