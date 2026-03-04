package handler

import (
	"net/http"
	"strconv"
	"time"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AdminHandler 管理员处理器
type AdminHandler struct {
	userService         service.UserService
	channelService      service.ChannelService
	orderService        service.OrderService
	logService          service.LogService
	modelService        service.ModelService
	alertService        service.AlertService
	settingsService     service.SettingsService
	resourceAccountRepo repository.ResourceAccountRepository
	announcementRepo    repository.AnnouncementRepository
}

func stringMapToJSON(input map[string]string) model.JSON {
	if input == nil {
		return nil
	}
	out := make(model.JSON, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

// NewAdminHandler 创建管理员处理器
func NewAdminHandler(
	userService service.UserService,
	channelService service.ChannelService,
	orderService service.OrderService,
	logService service.LogService,
	modelService service.ModelService,
	resourceAccountRepo repository.ResourceAccountRepository,
	announcementRepo repository.AnnouncementRepository,
	settingsService service.SettingsService,
	alertService service.AlertService,
) *AdminHandler {
	return &AdminHandler{
		userService:         userService,
		channelService:      channelService,
		orderService:        orderService,
		logService:          logService,
		modelService:        modelService,
		alertService:        alertService,
		settingsService:     settingsService,
		resourceAccountRepo: resourceAccountRepo,
		announcementRepo:    announcementRepo,
	}
}

// AdminLogin 管理员登录
// @Summary 管理员登录
// @Description 管理员账户登录
// @Tags 管理员-认证
// @Accept json
// @Produce json
// @Param request body model.AdminLoginRequest true "登录信息"
// @Success 200 {object} model.Response{data=model.AdminLoginResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/admin/login [post]
func (h *AdminHandler) AdminLogin(c *gin.Context) {
	var req model.AdminLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	// 使用 userService 进行管理员登录验证
	user, token, err := h.userService.AdminLogin(req.Username, req.Password)
	if err != nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(model.AdminLoginResponse{
		Token: token,
		User:  user.ToAdminResponse(),
	}))
}

// ==================== 用户管理 ====================

// ListUsers 获取用户列表
// @Summary 获取用户列表
// @Description 获取所有用户列表（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "状态筛选"
// @Param role query string false "角色筛选"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} model.PaginatedResponse{data=[]model.AdminUserResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/users [get]
func (h *AdminHandler) ListUsers(c *gin.Context) {
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

	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if role := c.Query("role"); role != "" {
		filters["role"] = role
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filters["keyword"] = keyword
	}

	users, pagination, err := h.userService.List(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(users, pagination))
}

// GetUser 获取用户详情
// @Summary 获取用户详情
// @Description 获取指定用户的详情（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Success 200 {object} model.Response{data=model.AdminUserResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/users/{id} [get]
func (h *AdminHandler) GetUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的用户 ID"))
		return
	}

	user, err := h.userService.GetByID(userID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user))
}

// GetUserStats 获取用户使用统计
// @Summary 获取用户使用统计
// @Description 获取指定用户的使用统计（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} model.Response{data=service.UserStatsResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/users/{id}/stats [get]
func (h *AdminHandler) GetUserStats(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的用户 ID"))
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

// CreateUser 创建用户
// @Summary 创建用户
// @Description 创建新用户（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.AdminCreateUserRequest true "用户信息"
// @Success 200 {object} model.Response{data=model.AdminUserResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/users [post]
func (h *AdminHandler) CreateUser(c *gin.Context) {
	var req model.AdminCreateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	user, err := h.userService.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user.ToAdminResponse()))
}

// UpdateUser 更新用户
// @Summary 更新用户
// @Description 更新指定用户的信息（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Param request body model.AdminUpdateUserRequest true "更新信息"
// @Success 200 {object} model.Response{data=model.AdminUserResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/users/{id} [put]
func (h *AdminHandler) UpdateUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的用户 ID"))
		return
	}

	var req model.AdminUpdateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	user, err := h.userService.Update(userID, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user))
}

// DeleteUser 删除用户
// @Summary 删除用户
// @Description 删除指定用户（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/users/{id} [delete]
func (h *AdminHandler) DeleteUser(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的用户 ID"))
		return
	}

	if err := h.userService.Delete(userID); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "用户删除成功"}))
}

// ==================== 渠道管理 ====================

// ListChannels 获取渠道列表
// @Summary 获取渠道列表
// @Description 获取所有渠道列表（管理员）
// @Tags 管理员-渠道管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "状态筛选"
// @Param type query string false "类型筛选"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} model.PaginatedResponse{data=[]model.ChannelResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/channels [get]
func (h *AdminHandler) ListChannels(c *gin.Context) {
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

	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if channelType := c.Query("type"); channelType != "" {
		filters["type"] = channelType
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filters["keyword"] = keyword
	}

	channels, pagination, err := h.channelService.List(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(channels, pagination))
}

// GetChannel 获取渠道详情
// @Summary 获取渠道详情
// @Description 获取指定渠道的详情（管理员）
// @Tags 管理员-渠道管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "渠道 ID"
// @Success 200 {object} model.Response{data=model.ChannelResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/channels/{id} [get]
func (h *AdminHandler) GetChannel(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道 ID"))
		return
	}

	channel, err := h.channelService.GetByID(uint(channelID))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(channel))
}

// CreateChannel 创建渠道
// @Summary 创建渠道
// @Description 创建新渠道（管理员）
// @Tags 管理员-渠道管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateChannelRequest true "渠道信息"
// @Success 200 {object} model.Response{data=model.ChannelResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/channels [post]
func (h *AdminHandler) CreateChannel(c *gin.Context) {
	var req model.CreateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	channel, err := h.channelService.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(channel.ToResponse()))
}

// UpdateChannel 更新渠道
// @Summary 更新渠道
// @Description 更新指定渠道的信息（管理员）
// @Tags 管理员-渠道管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "渠道 ID"
// @Param request body model.UpdateChannelRequest true "更新信息"
// @Success 200 {object} model.Response{data=model.ChannelResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/channels/{id} [put]
func (h *AdminHandler) UpdateChannel(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道 ID"))
		return
	}

	var req model.UpdateChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	channel, err := h.channelService.Update(uint(channelID), &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(channel))
}

// UpdateChannelStatus 更新渠道状态
// @Summary 更新渠道状态
// @Description 更新指定渠道的状态（管理员）
// @Tags 管理员-渠道管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "渠道 ID"
// @Param request body object true "状态信息"
// @Success 200 {object} model.Response{data=model.ChannelResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/channels/{id}/status [put]
func (h *AdminHandler) UpdateChannelStatus(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道 ID"))
		return
	}

	var req struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	status := model.ChannelStatus(req.Status)
	switch status {
	case model.ChannelStatusHealthy, model.ChannelStatusDegraded, model.ChannelStatusDown, model.ChannelStatusDisabled:
	default:
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道状态"))
		return
	}

	updated, err := h.channelService.Update(uint(channelID), &model.UpdateChannelRequest{Status: status})
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(updated))
}

// DeleteChannel 删除渠道
// @Summary 删除渠道
// @Description 删除指定渠道（管理员）
// @Tags 管理员-渠道管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "渠道 ID"
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/channels/{id} [delete]
func (h *AdminHandler) DeleteChannel(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道 ID"))
		return
	}

	if err := h.channelService.Delete(uint(channelID)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "渠道删除成功"}))
}

// TestChannel 测试渠道
// @Summary 测试渠道
// @Description 测试指定渠道的连通性（管理员）
// @Tags 管理员-渠道管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "渠道 ID"
// @Success 200 {object} model.Response{data=service.ChannelTestResult}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/channels/{id}/test [post]
func (h *AdminHandler) TestChannel(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道 ID"))
		return
	}

	result, err := h.channelService.TestChannel(uint(channelID))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

// GetDashboard 获取管理员仪表盘数据
// @Summary 获取管理员仪表盘数据
// @Description 获取系统概览统计数据（管理员）
// @Tags 管理员-仪表盘
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response{data=model.DashboardStats}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/dashboard [get]
func (h *AdminHandler) GetDashboard(c *gin.Context) {
	// 获取用户统计
	userStats, err := h.userService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 获取渠道统计
	channelStats, err := h.channelService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	dashboard := &model.DashboardStats{
		TotalUsers:        userStats.TotalUsers,
		ActiveUsers:       userStats.ActiveUsers,
		HealthyChannels:   channelStats.HealthyChannels,
		UnhealthyChannels: channelStats.UnhealthyChannels,
	}

	c.JSON(http.StatusOK, model.SuccessResponse(dashboard))
}

// ==================== 模型管理 ====================

// ListModels 获取模型列表
// @Summary 获取模型列表
// @Description 获取所有模型列表（管理员）
// @Tags 管理员-模型管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param enabled query bool false "是否启用"
// @Param type query string false "类型筛选"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} model.PaginatedResponse{data=[]model.ModelResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/models [get]
func (h *AdminHandler) ListModels(c *gin.Context) {
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

	filters := make(map[string]interface{})
	if enabled := c.Query("enabled"); enabled != "" {
		filters["enabled"] = enabled == "true"
	}
	if modelType := c.Query("type"); modelType != "" {
		filters["type"] = modelType
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filters["keyword"] = keyword
	}

	models, pagination, err := h.modelService.AdminList(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(models, pagination))
}

// GetModel 获取模型详情
// @Summary 获取模型详情
// @Description 获取指定模型的详情（管理员）
// @Tags 管理员-模型管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "模型 ID"
// @Success 200 {object} model.Response{data=model.ModelResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/models/{id} [get]
func (h *AdminHandler) GetModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的模型 ID"))
		return
	}

	modelResp, err := h.modelService.AdminGetByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(modelResp))
}

// CreateModel 创建模型
// @Summary 创建模型
// @Description 创建新模型（管理员）
// @Tags 管理员-模型管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateModelRequest true "模型信息"
// @Success 200 {object} model.Response{data=model.ModelResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/models [post]
func (h *AdminHandler) CreateModel(c *gin.Context) {
	var req model.CreateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	createdModel, err := h.modelService.Create(&req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(createdModel))
}

// UpdateModel 更新模型
// @Summary 更新模型
// @Description 更新指定模型的信息（管理员）
// @Tags 管理员-模型管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "模型 ID"
// @Param request body model.UpdateModelRequest true "更新信息"
// @Success 200 {object} model.Response{data=model.ModelResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/models/{id} [put]
func (h *AdminHandler) UpdateModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的模型 ID"))
		return
	}

	var req model.UpdateModelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	updatedModel, err := h.modelService.Update(id, &req)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(updatedModel))
}

// DeleteModel 删除模型
// @Summary 删除模型
// @Description 删除指定模型（管理员）
// @Tags 管理员-模型管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "模型 ID"
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/models/{id} [delete]
func (h *AdminHandler) DeleteModel(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的模型 ID"))
		return
	}

	if err := h.modelService.Delete(id); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "模型删除成功"}))
}

// UpdateModelStatus 更新模型状态
// @Summary 更新模型状态
// @Description 更新指定模型的启用状态（管理员）
// @Tags 管理员-模型管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "模型 ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/models/{id}/status [put]
func (h *AdminHandler) UpdateModelStatus(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的模型 ID"))
		return
	}

	var req struct {
		Enabled bool `json:"enabled" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数"))
		return
	}

	if err := h.modelService.UpdateStatus(id, req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "模型状态更新成功"}))
}

// ==================== 用户状态和余额管理 ====================

// UpdateUserStatus 更新用户状态
// @Summary 更新用户状态
// @Description 更新指定用户的状态（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/users/{id}/status [put]
func (h *AdminHandler) UpdateUserStatus(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的用户 ID"))
		return
	}

	var req struct {
		Status model.UserStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数"))
		return
	}

	updateReq := &model.AdminUpdateUserRequest{
		Status: req.Status,
	}

	user, err := h.userService.Update(userID, updateReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user))
}

// UpdateUserBalance 更新用户余额
// @Summary 更新用户余额
// @Description 更新指定用户的余额（管理员）
// @Tags 管理员-用户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "用户 ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/users/{id}/balance [put]
func (h *AdminHandler) UpdateUserBalance(c *gin.Context) {
	userID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的用户 ID"))
		return
	}

	var req struct {
		Balance *float64 `json:"balance" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数"))
		return
	}

	updateReq := &model.AdminUpdateUserRequest{
		Balance: req.Balance,
	}

	user, err := h.userService.Update(userID, updateReq)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(user))
}

// ==================== 日志管理 ====================

// ListLogs 获取日志列表
// @Summary 获取日志列表
// @Description 获取所有请求日志列表（管理员）
// @Tags 管理员-日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param user_id query string false "用户 ID"
// @Param channel_id query int false "渠道 ID"
// @Param model query string false "模型名称"
// @Param status query string false "状态"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} model.PaginatedResponse{data=[]model.AdminLogResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/logs [get]
func (h *AdminHandler) ListLogs(c *gin.Context) {
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

	filters := make(map[string]interface{})
	if userID := c.Query("user_id"); userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			filters["user_id"] = uid
		}
	}
	if channelID := c.Query("channel_id"); channelID != "" {
		if cid, err := strconv.ParseUint(channelID, 10, 32); err == nil {
			filters["channel_id"] = uint(cid)
		}
	}
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

	logs, pagination, err := h.logService.AdminList(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(logs, pagination))
}

// GetLogStats 获取日志统计
// @Summary 获取日志统计
// @Description 获取日志统计数据（管理员）
// @Tags 管理员-日志管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} model.Response{data=service.AdminStatsResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/logs/stats [get]
func (h *AdminHandler) GetLogStats(c *gin.Context) {
	startDate := c.Query("start_date")
	endDate := c.Query("end_date")

	stats, err := h.logService.AdminGetStats(startDate, endDate)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// ==================== 订单管理 ====================

// ListOrders 获取订单列表
// @Summary 获取订单列表
// @Description 获取所有订单列表（管理员）
// @Tags 管理员-订单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "状态筛选"
// @Param payment_method query string false "支付方式筛选"
// @Param start_date query string false "开始日期"
// @Param end_date query string false "结束日期"
// @Success 200 {object} model.PaginatedResponse{data=[]model.AdminOrderResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/orders [get]
func (h *AdminHandler) ListOrders(c *gin.Context) {
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

	filters := make(map[string]interface{})
	if userID := c.Query("user_id"); userID != "" {
		if uid, err := uuid.Parse(userID); err == nil {
			filters["user_id"] = uid
		}
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if paymentMethod := c.Query("payment_method"); paymentMethod != "" {
		filters["payment_method"] = paymentMethod
	}
	if startDate := c.Query("start_date"); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filters["end_date"] = endDate
	}

	orders, pagination, err := h.orderService.AdminList(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(orders, pagination))
}

// GetOrder 获取订单详情
// @Summary 获取订单详情
// @Description 获取指定订单的详情（管理员）
// @Tags 管理员-订单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "订单 ID"
// @Success 200 {object} model.Response{data=model.AdminOrderResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/orders/{id} [get]
func (h *AdminHandler) GetOrder(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的订单 ID"))
		return
	}

	order, err := h.orderService.AdminGetByID(orderID)
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(order))
}

// UpdateOrderStatus 更新订单状态
// @Summary 更新订单状态
// @Description 更新指定订单的状态（管理员）
// @Tags 管理员-订单管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "订单 ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/orders/{id}/status [put]
func (h *AdminHandler) UpdateOrderStatus(c *gin.Context) {
	orderID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的订单 ID"))
		return
	}

	var req struct {
		Status model.OrderStatus `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数"))
		return
	}

	if err := h.orderService.UpdateStatus(orderID, req.Status); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "订单状态更新成功"}))
}

// ==================== 公告管理 ====================

// ListAnnouncements 获取公告列表
// @Summary 获取公告列表
// @Description 获取所有公告列表（管理员）
// @Tags 管理员-公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param status query string false "状态筛选"
// @Param type query string false "类型筛选"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} model.PaginatedResponse{data=[]model.AnnouncementResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/announcements [get]
func (h *AdminHandler) ListAnnouncements(c *gin.Context) {
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

	filters := make(map[string]interface{})
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if announcementType := c.Query("type"); announcementType != "" {
		filters["type"] = announcementType
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filters["keyword"] = keyword
	}

	announcements, total, err := h.announcementRepo.List(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	responses := make([]*model.AnnouncementResponse, len(announcements))
	for i, a := range announcements {
		responses[i] = a.ToResponse()
	}

	pagination := model.NewPagination(query.Page, query.PageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(responses, pagination))
}

// GetAnnouncement 获取公告详情
// @Summary 获取公告详情
// @Description 获取指定公告的详情（管理员）
// @Tags 管理员-公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "公告 ID"
// @Success 200 {object} model.Response{data=model.AnnouncementResponse}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/announcements/{id} [get]
func (h *AdminHandler) GetAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的公告 ID"))
		return
	}

	announcement, err := h.announcementRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}
	if announcement == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "公告不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(announcement.ToResponse()))
}

// CreateAnnouncement 创建公告
// @Summary 创建公告
// @Description 创建新公告（管理员）
// @Tags 管理员-公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateAnnouncementRequest true "公告信息"
// @Success 200 {object} model.Response{data=model.AnnouncementResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/announcements [post]
func (h *AdminHandler) CreateAnnouncement(c *gin.Context) {
	var req model.CreateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	announcement := &model.Announcement{
		Title:     req.Title,
		Content:   req.Content,
		Type:      req.Type,
		Status:    req.Status,
		Priority:  req.Priority,
		ExpiresAt: req.ExpiresAt,
	}

	if err := h.announcementRepo.Create(announcement); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(announcement.ToResponse()))
}

// UpdateAnnouncement 更新公告
// @Summary 更新公告
// @Description 更新指定公告的信息（管理员）
// @Tags 管理员-公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "公告 ID"
// @Param request body model.UpdateAnnouncementRequest true "更新信息"
// @Success 200 {object} model.Response{data=model.AnnouncementResponse}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/announcements/{id} [put]
func (h *AdminHandler) UpdateAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的公告 ID"))
		return
	}

	announcement, err := h.announcementRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}
	if announcement == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "公告不存在"))
		return
	}

	var req model.UpdateAnnouncementRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	// 更新字段
	if req.Title != nil {
		announcement.Title = *req.Title
	}
	if req.Content != nil {
		announcement.Content = *req.Content
	}
	if req.Type != "" {
		announcement.Type = req.Type
	}
	if req.Status != "" {
		announcement.Status = req.Status
	}
	if req.Priority != nil {
		announcement.Priority = *req.Priority
	}
	if req.ExpiresAt != nil {
		announcement.ExpiresAt = req.ExpiresAt
	}

	if err := h.announcementRepo.Update(announcement); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(announcement.ToResponse()))
}

// DeleteAnnouncement 删除公告
// @Summary 删除公告
// @Description 删除指定公告（管理员）
// @Tags 管理员-公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "公告 ID"
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/announcements/{id} [delete]
func (h *AdminHandler) DeleteAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的公告 ID"))
		return
	}

	if err := h.announcementRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "公告删除成功"}))
}

// PublishAnnouncement 发布公告
// @Summary 发布公告
// @Description 发布指定公告（管理员）
// @Tags 管理员-公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "公告 ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/announcements/{id}/publish [post]
func (h *AdminHandler) PublishAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的公告 ID"))
		return
	}

	if err := h.announcementRepo.Publish(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "公告发布成功"}))
}

// ArchiveAnnouncement 归档公告
// @Summary 归档公告
// @Description 归档指定公告（管理员）
// @Tags 管理员-公告管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "公告 ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/announcements/{id}/archive [post]
func (h *AdminHandler) ArchiveAnnouncement(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的公告 ID"))
		return
	}

	if err := h.announcementRepo.Archive(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "公告归档成功"}))
}

// ==================== 资源账户管理 ====================

// ListResourceAccounts 获取资源账户列表
// @Summary 获取资源账户列表
// @Description 获取所有资源账户列表（管理员）
// @Tags 管理员-资源账户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Param channel_id query int false "渠道 ID"
// @Param status query string false "状态筛选"
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} model.PaginatedResponse{data=[]model.ResourceAccount}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/resource-accounts [get]
func (h *AdminHandler) ListResourceAccounts(c *gin.Context) {
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

	filters := make(map[string]interface{})
	if channelID := c.Query("channel_id"); channelID != "" {
		if cid, err := strconv.ParseUint(channelID, 10, 32); err == nil {
			filters["channel_id"] = uint(cid)
		}
	}
	if status := c.Query("status"); status != "" {
		filters["status"] = status
	}
	if keyword := c.Query("keyword"); keyword != "" {
		filters["keyword"] = keyword
	}

	accounts, total, err := h.resourceAccountRepo.List(query.Page, query.PageSize, filters)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	pagination := model.NewPagination(query.Page, query.PageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(accounts, pagination))
}

// GetResourceAccount 获取资源账户详情
// @Summary 获取资源账户详情
// @Description 获取指定资源账户的详情（管理员）
// @Tags 管理员-资源账户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源账户 ID"
// @Success 200 {object} model.Response{data=model.ResourceAccount}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/resource-accounts/{id} [get]
func (h *AdminHandler) GetResourceAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的资源账户 ID"))
		return
	}

	account, err := h.resourceAccountRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}
	if account == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "资源账户不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(account))
}

// CreateResourceAccount 创建资源账户
// @Summary 创建资源账户
// @Description 创建新资源账户（管理员）
// @Tags 管理员-资源账户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body model.CreateResourceAccountRequest true "资源账户信息"
// @Success 200 {object} model.Response{data=model.ResourceAccount}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/resource-accounts [post]
func (h *AdminHandler) CreateResourceAccount(c *gin.Context) {
	var req model.CreateResourceAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	account := &model.ResourceAccount{
		ChannelID:    req.ChannelID,
		AccountName:  req.AccountName,
		Credentials:  stringMapToJSON(req.Credentials),
		Status:       req.Status,
		ExpiresAt:    req.ExpiresAt,
		Metadata:     model.JSON(req.Metadata),
		LastActiveAt: nil,
	}

	if account.Status == "" {
		account.Status = model.ResourceAccountStatusActive
	}

	if err := h.resourceAccountRepo.Create(account); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(account))
}

// UpdateResourceAccount 更新资源账户
// @Summary 更新资源账户
// @Description 更新指定资源账户的信息（管理员）
// @Tags 管理员-资源账户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源账户 ID"
// @Param request body model.UpdateResourceAccountRequest true "更新信息"
// @Success 200 {object} model.Response{data=model.ResourceAccount}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/resource-accounts/{id} [put]
func (h *AdminHandler) UpdateResourceAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的资源账户 ID"))
		return
	}

	account, err := h.resourceAccountRepo.GetByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, err.Error()))
		return
	}
	if account == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "资源账户不存在"))
		return
	}

	var req model.UpdateResourceAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	// 更新字段
	if req.AccountName != nil {
		account.AccountName = *req.AccountName
	}
	if req.Credentials != nil {
		account.Credentials = stringMapToJSON(req.Credentials)
	}
	if req.Status != "" {
		account.Status = req.Status
	}
	if req.ExpiresAt != nil {
		account.ExpiresAt = req.ExpiresAt
	}
	if req.Metadata != nil {
		account.Metadata = model.JSON(req.Metadata)
	}

	if err := h.resourceAccountRepo.Update(account); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(account))
}

// DeleteResourceAccount 删除资源账户
// @Summary 删除资源账户
// @Description 删除指定资源账户（管理员）
// @Tags 管理员-资源账户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源账户 ID"
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/resource-accounts/{id} [delete]
func (h *AdminHandler) DeleteResourceAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的资源账户 ID"))
		return
	}

	if err := h.resourceAccountRepo.Delete(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "资源账户删除成功"}))
}

// RefreshResourceAccount 刷新资源账户
// @Summary 刷新资源账户
// @Description 刷新指定资源账户的状态（管理员）
// @Tags 管理员-资源账户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "资源账户 ID"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/admin/resource-accounts/{id}/refresh [post]
func (h *AdminHandler) RefreshResourceAccount(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的资源账户 ID"))
		return
	}

	// 标记为活跃状态
	if err := h.resourceAccountRepo.MarkActive(uint(id)); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "资源账户刷新成功"}))
}

// GetResourceAccountStats 获取资源账户统计
// @Summary 获取资源账户统计
// @Description 获取资源账户统计数据（管理员）
// @Tags 管理员-资源账户管理
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response{data=model.ResourcePoolStats}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/resource-accounts/stats [get]
func (h *AdminHandler) GetResourceAccountStats(c *gin.Context) {
	stats, err := h.resourceAccountRepo.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// ==================== 系统统计 ====================

// GetSystemStats 获取系统统计
// @Summary 获取系统统计
// @Description 获取系统整体统计数据（管理员）
// @Tags 管理员-统计
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/stats [get]
func (h *AdminHandler) GetSystemStats(c *gin.Context) {
	// 获取用户统计
	userStats, err := h.userService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 获取渠道统计
	channelStats, err := h.channelService.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 获取资源账户统计
	resourceStats, err := h.resourceAccountRepo.GetStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	stats := gin.H{
		"users": gin.H{
			"total":  userStats.TotalUsers,
			"active": userStats.ActiveUsers,
		},
		"channels": gin.H{
			"total":     channelStats.TotalChannels,
			"healthy":   channelStats.HealthyChannels,
			"unhealthy": channelStats.UnhealthyChannels,
		},
		"resource_accounts": resourceStats,
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// ==================== 系统监控和设置 ====================

// GetSystemMonitor 获取系统监控数据
// @Summary 获取系统监控数据
// @Description 获取系统资源使用情况
// @Tags 管理员-系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response{data=model.SystemMonitor}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/dashboard/system [get]
func (h *AdminHandler) GetSystemMonitor(c *gin.Context) {
	cpuPercent, err := getSystemCPUPercent()
	if err != nil {
		cpuPercent = 0
	}

	memoryPercent, err := getSystemMemoryPercent()
	if err != nil {
		memoryPercent = 0
	}

	monitor := &model.SystemMonitor{
		CPUPercent:       cpuPercent,
		MemoryPercent:    memoryPercent,
		RedisConnections: getRedisConnectedClients(),
		DBConnections:    getDBOpenConnections(),
	}

	c.JSON(http.StatusOK, model.SuccessResponse(monitor))
}

// GetAlerts 获取异常告警
// @Summary 获取异常告警
// @Description 获取系统异常告警列表
// @Tags 管理员-系统监控
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.Response{data=[]model.DashboardAlert}
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/dashboard/alerts [get]
func (h *AdminHandler) GetAlerts(c *gin.Context) {
	alerts, err := h.alertService.GetActiveAlerts()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	response := make([]*model.DashboardAlert, 0, len(alerts))
	for _, a := range alerts {
		response = append(response, &model.DashboardAlert{
			ID:        a.ID.String(),
			Message:   a.Title,
			Level:     string(a.Severity),
			CreatedAt: a.CreatedAt.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(response))
}

// GetSettings 获取系统设置
// @Summary 获取系统设置
// @Description 获取指定 section 的系统配置
// @Tags 管理员-系统设置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param section path string true "设置部分 (general/security/notification/system)"
// @Success 200 {object} model.Response{data=object}
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/admin/settings/{section} [get]
func (h *AdminHandler) GetSettings(c *gin.Context) {
	section := c.Param("section")

	settings, err := h.settingsService.GetSection(section)
	if err != nil {
		if err.Error() == "无效的设置部分" {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(settings))
}

// SaveSettings 保存系统设置
// @Summary 保存系统设置
// @Description 保存系统配置设置
// @Tags 管理员-系统设置
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param section path string true "设置部分 (general/security/notification/system)"
// @Param request body object true "设置数据"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Failure 403 {object} model.Response
// @Router /api/admin/settings/{section} [put]
func (h *AdminHandler) SaveSettings(c *gin.Context) {
	section := c.Param("section")

	var settings map[string]interface{}
	if err := c.ShouldBindJSON(&settings); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	updatedBy, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "缺少认证信息"))
		return
	}

	if err := h.settingsService.UpdateSection(section, settings, updatedBy); err != nil {
		if err.Error() == "无效的设置部分" {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
			return
		}
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "设置保存成功"}))
}
