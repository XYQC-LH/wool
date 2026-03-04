package handler

import (
	"net/http"
	"time"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UserHandler 用户处理器
type UserHandler struct {
	userService         service.UserService
	tokenService        service.TokenService
	orderService        service.OrderService
	logService          service.LogService
	modelService        service.ModelService
	announcementService service.AnnouncementService
}

// NewUserHandler 创建用户处理器
func NewUserHandler(userService service.UserService, tokenService service.TokenService, orderService service.OrderService, logService service.LogService, modelService service.ModelService, announcementService service.AnnouncementService) *UserHandler {
	return &UserHandler{
		userService:         userService,
		tokenService:        tokenService,
		orderService:        orderService,
		logService:          logService,
		modelService:        modelService,
		announcementService: announcementService,
	}
}

// Register 用户注册
// @Summary 用户注册
// @Description 创建新用户账户
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body model.RegisterRequest true "注册信息"
// @Success 200 {object} model.Response{data=model.UserResponse}
// @Failure 400 {object} model.Response
// @Router /api/user/auth/register [post]
func (h *UserHandler) Register(c *gin.Context) {
	var req model.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	user, err := h.userService.Register(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user.ToResponse()))
}

// Login 用户登录
// @Summary 用户登录
// @Description 用户登录获取 JWT Token
// @Tags 用户认证
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "登录信息"
// @Success 200 {object} model.Response{data=model.LoginResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/user/auth/login [post]
func (h *UserHandler) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	resp, err := h.userService.Login(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(resp))
}

// GetProfile 获取用户资料
// @Summary 获取用户资料
// @Description 获取当前登录用户的资料
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response{data=model.UserResponse}
// @Failure 401 {object} model.Response
// @Router /api/user/profile [get]
func (h *UserHandler) GetProfile(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	profile, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(profile))
}

// UpdateProfile 更新用户资料
// @Summary 更新用户资料
// @Description 更新当前登录用户的资料
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.UpdateProfileRequest true "更新信息"
// @Success 200 {object} model.Response{data=model.UserResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/user/profile [put]
func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var req model.UpdateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	profile, err := h.userService.UpdateProfile(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(profile))
}

// ChangePassword 修改密码
// @Summary 修改密码
// @Description 修改当前登录用户的密码
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.ChangePasswordRequest true "密码信息"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/user/password [put]
func (h *UserHandler) ChangePassword(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var req model.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	if err := h.userService.ChangePassword(userID, &req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "密码修改成功"}))
}

// UpdateNotifications 更新通知设置
// @Summary 更新通知设置
// @Description 更新当前登录用户的通知偏好设置
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.UpdateNotificationsRequest true "通知设置"
// @Success 200 {object} model.Response{data=model.UserResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/user/notifications [put]
func (h *UserHandler) UpdateNotifications(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var req model.UpdateNotificationsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	user, err := h.userService.UpdateNotifications(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user))
}

// GetDashboard 获取仪表盘数据
// @Summary 获取仪表盘数据
// @Description 获取用户仪表盘统计数据
// @Tags 用户
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response{data=model.UserDashboard}
// @Failure 401 {object} model.Response
// @Router /api/user/dashboard [get]
func (h *UserHandler) GetDashboard(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	dashboard, err := h.userService.GetDashboard(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(dashboard))
}

// ==================== Token 管理 ====================

// CreateToken 创建 API Token
// @Summary 创建 API Token
// @Description 创建新的 API Token
// @Tags Token
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateTokenRequest true "Token 信息"
// @Success 200 {object} model.Response{data=model.TokenResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/user/tokens [post]
func (h *UserHandler) CreateToken(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var req model.CreateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	token, err := h.tokenService.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	// 返回完整的 Token（只在创建时返回一次）
	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"id":           token.ID,
		"key":          token.Key, // 完整的 API Key
		"name":         token.Name,
		"status":       token.Status,
		"remain_quota": token.RemainQuota,
		"expires_at":   token.ExpiresAt,
		"created_at":   token.CreatedAt,
	}))
}

// ListTokens 获取 Token 列表
// @Summary 获取 Token 列表
// @Description 获取当前用户的所有 API Token
// @Tags Token
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} model.PaginatedResponse{data=[]model.TokenResponse}
// @Failure 401 {object} model.Response
// @Router /api/user/tokens [get]
func (h *UserHandler) ListTokens(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var query model.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的查询参数"))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	tokens, pagination, err := h.tokenService.List(userID, query.Page, query.PageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(tokens, pagination))
}

// GetToken 获取 Token 详情
// @Summary 获取 Token 详情
// @Description 获取指定 Token 的详情
// @Tags Token
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID"
// @Success 200 {object} model.Response{data=model.TokenResponse}
// @Failure 401 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/tokens/{id} [get]
func (h *UserHandler) GetToken(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	token, err := h.tokenService.GetByID(userID, tokenID)
	if err != nil {
		if err.Error() == "无权访问此 Token" {
			c.JSON(http.StatusForbidden, model.ErrorResponse(model.ErrCodeForbidden, err.Error()))
			return
		}
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(token))
}

// UpdateToken 更新 Token
// @Summary 更新 Token
// @Description 更新指定 Token 的信息
// @Tags Token
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID"
// @Param request body model.UpdateTokenRequest true "更新信息"
// @Success 200 {object} model.Response{data=model.TokenResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/tokens/{id} [put]
func (h *UserHandler) UpdateToken(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	var req model.UpdateTokenRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	token, err := h.tokenService.Update(userID, tokenID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(token))
}

// DeleteToken 删除 Token
// @Summary 删除 Token
// @Description 删除指定的 Token
// @Tags Token
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID"
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/tokens/{id} [delete]
func (h *UserHandler) DeleteToken(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	if err := h.tokenService.Delete(userID, tokenID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "Token 删除成功"}))
}

// ==================== 订单管理 ====================

// CreateOrder 创建订单
// @Summary 创建充值订单
// @Description 创建新的充值订单
// @Tags 订单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateOrderRequest true "订单信息"
// @Success 200 {object} model.Response{data=model.CreateOrderResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/user/orders [post]
func (h *UserHandler) CreateOrder(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var req model.CreateOrderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	order, err := h.orderService.Create(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(order))
}

// ListOrders 获取订单列表
// @Summary 获取订单列表
// @Description 获取当前用户的所有订单
// @Tags 订单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "状态筛选"
// @Success 200 {object} model.PaginatedResponse{data=[]model.OrderResponse}
// @Failure 401 {object} model.Response
// @Router /api/user/orders [get]
func (h *UserHandler) ListOrders(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var query model.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的查询参数"))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	status := c.Query("status")

	orders, pagination, err := h.orderService.List(userID, query.Page, query.PageSize, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(orders, pagination))
}

// GetOrder 获取订单详情
// @Summary 获取订单详情
// @Description 获取指定订单的详情
// @Tags 订单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "订单 ID"
// @Success 200 {object} model.Response{data=model.OrderResponse}
// @Failure 401 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/orders/{id} [get]
func (h *UserHandler) GetOrder(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的订单 ID"))
		return
	}

	order, err := h.orderService.GetByID(userID, orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(order))
}

// GetOrderByOrderNo 获取订单详情（按订单号）
// @Summary 获取订单详情（按订单号）
// @Description 获取指定订单号的订单详情
// @Tags 订单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param order_no path string true "订单号"
// @Success 200 {object} model.Response{data=model.OrderResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/orders/by-no/{order_no} [get]
func (h *UserHandler) GetOrderByOrderNo(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	orderNo := c.Param("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "缺少订单号"))
		return
	}

	order, err := h.orderService.GetByOrderNo(userID, orderNo)
	if err != nil {
		switch err.Error() {
		case "订单不存在":
			c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		case "无权访问此订单":
			c.JSON(http.StatusForbidden, model.ErrorResponse(model.ErrCodeForbidden, err.Error()))
		default:
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(order))
}

// CancelOrder 取消订单
// @Summary 取消订单
// @Description 取消指定的订单
// @Tags 订单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "订单 ID"
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/orders/{id}/cancel [post]
func (h *UserHandler) CancelOrder(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的订单 ID"))
		return
	}

	if err := h.orderService.Cancel(userID, orderID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "订单取消成功"}))
}

// PayOrderByOrderNo 支付订单（测试/手动确认）
// @Summary 支付订单（按订单号）
// @Description 手动确认支付（用于测试/开发环境）。生产环境应接入真实支付网关回调。
// @Tags 订单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param order_no path string true "订单号"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/orders/by-no/{order_no}/pay [post]
func (h *UserHandler) PayOrderByOrderNo(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	// ⚠️ 安全保护：手动支付确认仅允许在非 release 环境使用
	// 生产环境应接入真实支付渠道（如 Stripe Webhook）来回调并确认订单支付。
	if gin.Mode() == gin.ReleaseMode {
		c.JSON(http.StatusForbidden, model.ErrorResponse(model.ErrCodeForbidden, "支付确认接口仅限开发/测试环境"))
		return
	}

	orderNo := c.Param("order_no")
	if orderNo == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "缺少订单号"))
		return
	}

	if err := h.orderService.PayByOrderNo(userID, orderNo); err != nil {
		switch err.Error() {
		case "订单不存在":
			c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		case "无权操作此订单":
			c.JSON(http.StatusForbidden, model.ErrorResponse(model.ErrCodeForbidden, err.Error()))
		default:
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		}
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "支付成功"}))
}

// ==================== 日志管理 ====================

// ListLogs 获取日志列表
// @Summary 获取调用日志列表
// @Description 获取当前用户的 API 调用日志
// @Tags 日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param model query string false "模型筛选"
// @Param status query string false "状态筛选"
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} model.PaginatedResponse{data=[]model.LogResponse}
// @Failure 401 {object} model.Response
// @Router /api/user/logs [get]
func (h *UserHandler) ListLogs(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var query model.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的查询参数"))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	// 构建过滤条件
	filters := make(map[string]interface{})
	if modelName := c.Query("model"); modelName != "" {
		filters["model"] = modelName
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if startDate := c.Query("start_date"); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filters["end_date"] = endDate
	}

	logs, pagination, err := h.logService.List(userID, query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(logs, pagination))
}

// GetLogStats 获取日志统计
// @Summary 获取使用统计
// @Description 获取当前用户的 API 使用统计数据
// @Tags 日志
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} model.Response{data=service.UserStatsResponse}
// @Failure 401 {object} model.Response
// @Router /api/user/logs/stats [get]
func (h *UserHandler) GetLogStats(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	stats, err := h.logService.GetStats(userID, startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// ==================== 账单管理 ====================

// GetBillingOverview 获取账单概览
// @Summary 获取账单概览
// @Description 获取当前用户的账单概览数据
// @Tags 账单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response{data=model.BillingOverview}
// @Failure 401 {object} model.Response
// @Router /api/user/billing/overview [get]
func (h *UserHandler) GetBillingOverview(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	// 获取用户信息
	user, err := h.userService.GetProfile(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 获取今日统计
	now := time.Now()
	startOfDay := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	todayStats, err := h.logService.GetStats(userID, startOfDay.Format("2006-01-02"), endOfDay.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 获取本月统计
	startOfMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthStats, err := h.logService.GetStats(userID, startOfMonth.Format("2006-01-02"), endOfDay.Format("2006-01-02"))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 获取订单统计
	totalRecharge, err := h.orderService.GetTotalRecharge(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	overview := &model.BillingOverview{
		Balance:       user.Balance,
		TodayCost:     todayStats.TotalCost,
		MonthCost:     monthStats.TotalCost,
		TotalRecharge: totalRecharge,
		TodayRequests: todayStats.TotalRequests,
		MonthRequests: monthStats.TotalRequests,
	}

	c.JSON(http.StatusOK, model.SuccessResponse(overview))
}

// GetConsumptionDetails 获取消费明细
// @Summary 获取消费明细
// @Description 获取当前用户的消费明细列表
// @Tags 账单
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param start_date query string false "开始日期 (YYYY-MM-DD)"
// @Param end_date query string false "结束日期 (YYYY-MM-DD)"
// @Success 200 {object} model.PaginatedResponse{data=[]model.ConsumptionDetail}
// @Failure 401 {object} model.Response
// @Router /api/user/billing/consumption [get]
func (h *UserHandler) GetConsumptionDetails(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	var query model.PaginationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的查询参数"))
		return
	}

	if query.Page <= 0 {
		query.Page = 1
	}
	if query.PageSize <= 0 {
		query.PageSize = 20
	}

	// 构建过滤条件
	filters := make(map[string]interface{})
	if startDate := c.Query("start_date"); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filters["end_date"] = endDate
	}

	logs, pagination, err := h.logService.List(userID, query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 转换为消费明细
	details := make([]*model.ConsumptionDetail, len(logs))
	for i, log := range logs {
		details[i] = &model.ConsumptionDetail{
			ID:               log.ID,
			Model:            log.Model,
			PromptTokens:     int64(log.PromptTokens),
			CompletionTokens: int64(log.CompletionTokens),
			TotalTokens:      int64(log.TotalTokens),
			Cost:             log.TotalCost,
			Status:           string(log.Status),
			CreatedAt:        log.CreatedAt,
		}
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(details, pagination))
}

// ==================== Token 状态管理 ====================

// UpdateTokenStatus 更新Token状态
// @Summary 更新Token状态
// @Description 更新指定Token的状态
// @Tags Token
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Token ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/user/tokens/{id}/status [put]
func (h *UserHandler) UpdateTokenStatus(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	tokenID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的 Token ID"))
		return
	}

	var req struct {
		Status model.TokenStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数"))
		return
	}

	updateReq := &model.UpdateTokenRequest{
		Status: req.Status,
	}

	token, err := h.tokenService.Update(userID, tokenID, updateReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(token))
}

// ==================== 公开 API ====================

// GetPublicModels 获取公开的模型列表
// @Summary 获取模型列表
// @Description 获取所有可用的 AI 模型（公开访问）
// @Tags 公开
// @Produce json
// @Success 200 {object} model.Response{data=[]model.Model}
// @Router /api/public/models [get]
func (h *UserHandler) GetPublicModels(c *gin.Context) {
	// 获取所有激活的模型
	models, err := h.modelService.ListPublic()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(models))
}

// GetPublicAnnouncements 获取公开的公告列表
// @Summary 获取公告列表
// @Description 获取所有已发布的公告（公开访问）
// @Tags 公开
// @Produce json
// @Success 200 {object} model.Response{data=[]model.Announcement}
// @Router /api/public/announcements [get]
func (h *UserHandler) GetPublicAnnouncements(c *gin.Context) {
	// 获取所有已发布的公告
	announcements, err := h.announcementService.ListPublic()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(announcements))
}
