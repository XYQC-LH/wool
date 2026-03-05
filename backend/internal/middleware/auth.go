package middleware

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"nexus-api/internal/auth"
	"nexus-api/internal/cache"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// 上下文键
const (
	ContextKeyUser    = "user"
	ContextKeyUserID  = "user_id"
	ContextKeyTenantID = "tenant_id"
	ContextKeyToken   = "token"
	ContextKeyTokenID = "token_id"
	ContextKeyRole    = "role"
)

// JWTAuthMiddleware JWT 认证中间件（用于用户/管理员 API）
func JWTAuthMiddleware(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "缺少认证令牌"))
			c.Abort()
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "无效的认证格式"))
			c.Abort()
			return
		}

		tokenString := parts[1]

		// 解析 JWT
		claims := &auth.JWTClaims{}
		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, errors.New("无效的签名方法")
			}
			return []byte(secret), nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "无效或过期的令牌"))
			c.Abort()
			return
		}

		// 设置上下文
		c.Set(ContextKeyUserID, claims.UserID)
		c.Set(ContextKeyRole, claims.Role)

		c.Next()
	}
}

// AdminOnlyMiddleware 管理员权限中间件
func AdminOnlyMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// 检查是否是管理员
		role, exists := c.Get(ContextKeyRole)
		if !exists {
			c.JSON(http.StatusForbidden, model.ErrorResponse(model.ErrCodeForbidden, "权限不足"))
			c.Abort()
			return
		}

		userRole := role.(model.UserRole)
		if userRole != model.RoleAdmin && userRole != model.RoleSuperAdmin {
			c.JSON(http.StatusForbidden, model.ErrorResponse(model.ErrCodeForbidden, "需要管理员权限"))
			c.Abort()
			return
		}

		c.Next()
	}
}

// GatewayAuthMiddleware Gateway API 认证中间件（使用 API Key）
func GatewayAuthMiddleware(tokenService service.TokenService) gin.HandlerFunc {
	return func(c *gin.Context) {
		// 获取 Authorization 头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
				"缺少 API Key",
				model.OpenAIErrorTypeAuthentication,
				nil,
			))
			c.Abort()
			return
		}

		// 解析 Bearer Token
		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
				"无效的认证格式，请使用 Bearer Token",
				model.OpenAIErrorTypeAuthentication,
				nil,
			))
			c.Abort()
			return
		}

		apiKey := parts[1]

		// 验证 API Key 格式
		if !strings.HasPrefix(apiKey, "sk-") {
			c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
				"无效的 API Key 格式",
				model.OpenAIErrorTypeAuthentication,
				nil,
			))
			c.Abort()
			return
		}

		// 先从缓存查找（使用带用户信息的缓存key）
		var tokenInfo *model.Token
		cacheKey := cache.KeyTokenPrefix + apiKey + ":with_user"
		err := cache.Get(cacheKey, &tokenInfo)

		if err != nil {
			// 缓存未命中，从服务层查询（使用GetByKeyWithUser获取用户信息）
			tokenInfo, err = tokenService.ValidateToken(apiKey)
			if err != nil || tokenInfo == nil {
				c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
					err.Error(),
					model.OpenAIErrorTypeAuthentication,
					nil,
				))
				c.Abort()
				return
			}

			// 缓存 Token 信息（5分钟）
			_ = cache.Set(cacheKey, tokenInfo, 5*time.Minute)
		}

		// 检查 Token 状态
		if tokenInfo.Status != model.TokenStatusActive {
			c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
				"API Key 已被禁用",
				model.OpenAIErrorTypeAuthentication,
				nil,
			))
			c.Abort()
			return
		}

		// 检查是否过期
		if tokenInfo.IsExpired() {
			c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
				"API Key 已过期",
				model.OpenAIErrorTypeAuthentication,
				nil,
			))
			c.Abort()
			return
		}

		// IP 白名单校验（AllowedIPs 为空表示不限制）
		clientIP := c.ClientIP()
		if !tokenInfo.IsIPAllowed(clientIP) {
			c.JSON(http.StatusForbidden, model.NewOpenAIError(
				"当前 IP 未在该 API Key 的白名单中",
				model.OpenAIErrorTypePermission,
				nil,
			))
			c.Abort()
			return
		}

		// 租户隔离校验：请求租户必须与 Token 绑定租户一致
		requestTenantID := strings.TrimSpace(c.GetHeader("X-Tenant-ID"))
		effectiveTenantID := tokenInfo.EffectiveTenantID()
		if effectiveTenantID == "" {
			c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
				"租户信息缺失",
				model.OpenAIErrorTypeAuthentication,
				nil,
			))
			c.Abort()
			return
		}
		if requestTenantID != "" && requestTenantID != effectiveTenantID {
			c.JSON(http.StatusForbidden, model.NewOpenAIError(
				"租户隔离校验失败",
				model.OpenAIErrorTypePermission,
				nil,
			))
			c.Abort()
			return
		}

		// 设置上下文
		c.Set(ContextKeyToken, tokenInfo)
		c.Set(ContextKeyTokenID, tokenInfo.ID)
		c.Set(ContextKeyUserID, tokenInfo.UserID)
		c.Set(ContextKeyTenantID, effectiveTenantID)
		c.Header("X-Tenant-ID", effectiveTenantID)

		// 更新最后使用时间（使用 Redis 进行节流：每个 Token 每分钟最多写一次 DB）
		lastUsedKey := "token:last_used:" + tokenInfo.ID.String()
		if ok, err := cache.SetNX(lastUsedKey, 1, time.Minute); err != nil || ok {
			go func(tokenID uuid.UUID) {
				_ = tokenService.UpdateLastUsed(tokenID)
			}(tokenInfo.ID)
		}

		c.Next()
	}
}

// GetCurrentUser 从上下文获取当前用户
func GetCurrentUser(c *gin.Context) (*model.User, bool) {
	user, exists := c.Get(ContextKeyUser)
	if !exists {
		return nil, false
	}
	return user.(*model.User), true
}

// GetCurrentUserID 从上下文获取当前用户 ID
func GetCurrentUserID(c *gin.Context) (uuid.UUID, bool) {
	userID, exists := c.Get(ContextKeyUserID)
	if !exists {
		return uuid.Nil, false
	}
	return userID.(uuid.UUID), true
}

// GetCurrentTenantID 从上下文获取当前租户 ID
func GetCurrentTenantID(c *gin.Context) (string, bool) {
	tenantID, exists := c.Get(ContextKeyTenantID)
	if !exists {
		return "", false
	}
	value, ok := tenantID.(string)
	if !ok {
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", false
	}
	return value, true
}

// GetCurrentToken 从上下文获取当前 Token
func GetCurrentToken(c *gin.Context) (*model.Token, bool) {
	token, exists := c.Get(ContextKeyToken)
	if !exists {
		return nil, false
	}
	return token.(*model.Token), true
}

// IsAdmin 检查当前用户是否是管理员
func IsAdmin(c *gin.Context) bool {
	role, exists := c.Get(ContextKeyRole)
	if !exists {
		return false
	}
	return role.(model.UserRole) == model.UserRoleAdmin
}
