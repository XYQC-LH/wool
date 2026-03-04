package handler

import (
	"net/http"
	"strconv"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"
	"nexus-api/internal/service/scheduler"

	"github.com/gin-gonic/gin"
)

// ProviderInstanceHandler 源头实例处理器
type ProviderInstanceHandler struct {
	instanceRepo repository.ProviderInstanceRepository
	stateStore   scheduler.RuntimeStateStore
}

// NewProviderInstanceHandler 创建源头实例处理器
func NewProviderInstanceHandler(instanceRepo repository.ProviderInstanceRepository, stateStore scheduler.RuntimeStateStore) *ProviderInstanceHandler {
	return &ProviderInstanceHandler{
		instanceRepo: instanceRepo,
		stateStore:   stateStore,
	}
}

// CreateInstanceRequest 创建实例请求
type CreateInstanceRequest struct {
	Name              string               `json:"name" binding:"required,min=1,max=100"`
	InstanceType      model.InstanceType   `json:"instance_type" binding:"required"`
	ResourceAccountID *uint                `json:"resource_account_id,omitempty"`
	Weight            int                  `json:"weight"`
	MaxConcurrency    int                  `json:"max_concurrency"`
	RPMLimit          int                  `json:"rpm_limit"`
	TPMLimit          int                  `json:"tpm_limit"`
	Status            model.InstanceStatus `json:"status,omitempty"`
}

// UpdateInstanceRequest 更新实例请求
type UpdateInstanceRequest struct {
	Name              *string               `json:"name,omitempty"`
	InstanceType      *model.InstanceType   `json:"instance_type,omitempty"`
	ResourceAccountID *uint                 `json:"resource_account_id,omitempty"`
	Weight            *int                  `json:"weight,omitempty"`
	MaxConcurrency    *int                  `json:"max_concurrency,omitempty"`
	RPMLimit          *int                  `json:"rpm_limit,omitempty"`
	TPMLimit          *int                  `json:"tpm_limit,omitempty"`
	Status            *model.InstanceStatus `json:"status,omitempty"`
}

// BatchInstanceStatusRequest 批量状态更新请求
type BatchInstanceStatusRequest struct {
	IDs    []uint               `json:"ids" binding:"required"`
	Status model.InstanceStatus `json:"status" binding:"required"`
}

// Create 创建源头实例
// @Summary 创建源头实例
// @Description 在指定源头下创建新的实例
// @Tags 源头实例
// @Accept json
// @Produce json
// @Param id path int true "源头ID"
// @Param request body CreateInstanceRequest true "创建请求"
// @Success 200 {object} model.ProviderInstanceResponse
// @Router /api/admin/providers/{id}/instances [post]
func (h *ProviderInstanceHandler) Create(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	var req CreateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	instance := &model.ProviderInstance{
		ProviderID:        uint(providerID),
		Name:              req.Name,
		InstanceType:      req.InstanceType,
		ResourceAccountID: req.ResourceAccountID,
		Weight:            req.Weight,
		MaxConcurrency:    req.MaxConcurrency,
		RPMLimit:          req.RPMLimit,
		TPMLimit:          req.TPMLimit,
		Status:            req.Status,
	}

	// 设置默认值
	if instance.Weight <= 0 {
		instance.Weight = 1
	}
	if instance.Status == "" {
		instance.Status = model.InstanceStatusActive
	}

	if err := h.instanceRepo.Create(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 重新获取完整数据（包含关联）
	instance, err = h.instanceRepo.GetByID(c.Request.Context(), instance.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(instance.ToResponse()))
}

// Update 更新源头实例
// @Summary 更新源头实例
// @Description 更新源头实例配置
// @Tags 源头实例
// @Accept json
// @Produce json
// @Param id path int true "实例ID"
// @Param request body UpdateInstanceRequest true "更新请求"
// @Success 200 {object} model.ProviderInstanceResponse
// @Router /api/admin/provider-instances/{id} [put]
func (h *ProviderInstanceHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if instance == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "实例不存在"))
		return
	}

	var req UpdateInstanceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	// 更新字段
	if req.Name != nil {
		instance.Name = *req.Name
	}
	if req.InstanceType != nil {
		instance.InstanceType = *req.InstanceType
	}
	if req.ResourceAccountID != nil {
		instance.ResourceAccountID = req.ResourceAccountID
	}
	if req.Weight != nil {
		instance.Weight = *req.Weight
	}
	if req.MaxConcurrency != nil {
		instance.MaxConcurrency = *req.MaxConcurrency
	}
	if req.RPMLimit != nil {
		instance.RPMLimit = *req.RPMLimit
	}
	if req.TPMLimit != nil {
		instance.TPMLimit = *req.TPMLimit
	}
	if req.Status != nil {
		instance.Status = *req.Status
	}

	if err := h.instanceRepo.Update(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 重新获取完整数据
	instance, err = h.instanceRepo.GetByID(c.Request.Context(), instance.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(instance.ToResponse()))
}

// Delete 删除源头实例
// @Summary 删除源头实例
// @Description 删除源头实例
// @Tags 源头实例
// @Param id path int true "实例ID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/provider-instances/{id} [delete]
func (h *ProviderInstanceHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.instanceRepo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "删除成功"}))
}

// GetByID 获取单个源头实例
// @Summary 获取源头实例详情
// @Description 根据ID获取源头实例详情
// @Tags 源头实例
// @Param id path int true "实例ID"
// @Success 200 {object} model.ProviderInstanceResponse
// @Router /api/admin/provider-instances/{id} [get]
func (h *ProviderInstanceHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if instance == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "实例不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(instance.ToResponse()))
}

// List 获取源头实例列表
// @Summary 获取源头实例列表
// @Description 分页获取指定源头的实例列表
// @Tags 源头实例
// @Param id path int true "源头ID"
// @Param status query string false "状态"
// @Param instance_type query string false "实例类型"
// @Param page query int false "页码"
// @Param page_size query int false "每页数量"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/providers/{id}/instances [get]
func (h *ProviderInstanceHandler) List(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	params := &model.ProviderInstanceQueryParams{
		ProviderID:   uint(providerID),
		Status:       model.InstanceStatus(c.Query("status")),
		InstanceType: model.InstanceType(c.Query("instance_type")),
	}

	if page := c.Query("page"); page != "" {
		if p, err := strconv.Atoi(page); err == nil {
			params.Page = p
		}
	}
	if params.Page <= 0 {
		params.Page = 1
	}

	if pageSize := c.Query("page_size"); pageSize != "" {
		if ps, err := strconv.Atoi(pageSize); err == nil {
			params.PageSize = ps
		}
	}
	if params.PageSize <= 0 {
		params.PageSize = 20
	}

	instances, total, err := h.instanceRepo.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	// 转换为响应格式
	responses := make([]*model.ProviderInstanceResponse, len(instances))
	for i, instance := range instances {
		responses[i] = instance.ToResponse()
	}

	pagination := model.NewPagination(params.Page, params.PageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(responses, pagination))
}

// Enable 启用实例
// @Summary 启用实例
// @Description 启用指定的源头实例
// @Tags 源头实例
// @Param id path int true "实例ID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/provider-instances/{id}/enable [post]
func (h *ProviderInstanceHandler) Enable(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if instance == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "实例不存在"))
		return
	}

	instance.Status = model.InstanceStatusActive
	if err := h.instanceRepo.Update(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "启用成功"}))
}

// Disable 禁用实例
// @Summary 禁用实例
// @Description 禁用指定的源头实例
// @Tags 源头实例
// @Param id path int true "实例ID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/provider-instances/{id}/disable [post]
func (h *ProviderInstanceHandler) Disable(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	instance, err := h.instanceRepo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if instance == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "实例不存在"))
		return
	}

	instance.Status = model.InstanceStatusDisabled
	if err := h.instanceRepo.Update(c.Request.Context(), instance); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "禁用成功"}))
}

// BatchUpdateStatus 批量更新状态
// @Summary 批量更新实例状态
// @Description 批量更新多个实例的状态
// @Tags 源头实例
// @Accept json
// @Produce json
// @Param request body BatchInstanceStatusRequest true "批量更新请求"
// @Success 200 {object} map[string]bool
// @Router /api/admin/provider-instances/batch/status [post]
func (h *ProviderInstanceHandler) BatchUpdateStatus(c *gin.Context) {
	var req BatchInstanceStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	if err := h.instanceRepo.BatchUpdateStatus(c.Request.Context(), req.IDs, req.Status); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "批量更新成功"}))
}

// ResetStats 重置统计数据
// @Summary 重置实例统计数据
// @Description 重置指定实例的统计数据
// @Tags 源头实例
// @Param id path int true "实例ID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/provider-instances/{id}/reset-stats [post]
func (h *ProviderInstanceHandler) ResetStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.instanceRepo.ResetStats(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "统计数据已重置"}))
}

// GetStats 获取实例统计
// @Summary 获取实例统计
// @Description 获取指定源头的实例统计数据
// @Tags 源头实例
// @Param id path int true "源头ID"
// @Success 200 {object} model.ProviderInstanceStats
// @Router /api/admin/providers/{id}/instances/stats [get]
func (h *ProviderInstanceHandler) GetStats(c *gin.Context) {
	providerID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	stats, err := h.instanceRepo.GetStats(c.Request.Context(), uint(providerID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}

// ResetCircuit 重置实例熔断状态
// @Summary 重置实例熔断状态
// @Description 将实例熔断状态置为 closed，并清空连续失败计数
// @Tags 源头实例
// @Param id path int true "实例ID"
// @Success 200 {object} model.Response
// @Router /api/admin/provider-instances/{id}/reset-circuit [post]
func (h *ProviderInstanceHandler) ResetCircuit(c *gin.Context) {
	if h.stateStore == nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, "运行时状态存储未初始化"))
		return
	}

	instanceID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的实例ID"))
		return
	}

	_ = h.stateStore.SetInstanceCircuitState(c.Request.Context(), uint(instanceID), model.CircuitStateClosed, 30*time.Second)
	_ = h.stateStore.ResetInstanceFailureCount(c.Request.Context(), uint(instanceID))

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"reset": true}))
}

// RegisterRoutes 注册路由
func (h *ProviderInstanceHandler) RegisterRoutes(router *gin.RouterGroup) {
	// 源头下的实例路由
	providers := router.Group("/providers")
	{
		// 注意：参数名需与 ModelProviderHandler 中的 "/providers/:id" 保持一致，否则 Gin 会因通配符冲突而 panic
		providers.POST("/:id/instances", h.Create)
		providers.GET("/:id/instances", h.List)
		providers.GET("/:id/instances/stats", h.GetStats)
	}

	// 实例独立路由
	instances := router.Group("/provider-instances")
	{
		instances.GET("/:id", h.GetByID)
		instances.PUT("/:id", h.Update)
		instances.DELETE("/:id", h.Delete)
		instances.POST("/:id/enable", h.Enable)
		instances.POST("/:id/disable", h.Disable)
		instances.POST("/:id/reset-stats", h.ResetStats)
		instances.POST("/:id/reset-circuit", h.ResetCircuit)
		instances.POST("/batch/status", h.BatchUpdateStatus)
	}
}
