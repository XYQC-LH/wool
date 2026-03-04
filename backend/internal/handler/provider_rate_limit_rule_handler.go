package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/gin-gonic/gin"
)

// ProviderRateLimitRuleHandler 多模态限流规则处理器
type ProviderRateLimitRuleHandler struct {
	repo repository.ProviderRateLimitRuleRepository
}

func NewProviderRateLimitRuleHandler(repo repository.ProviderRateLimitRuleRepository) *ProviderRateLimitRuleHandler {
	return &ProviderRateLimitRuleHandler{repo: repo}
}

type CreateProviderRateLimitRuleRequest struct {
	Scope         string `json:"scope,omitempty"`
	ProviderID    uint   `json:"provider_id" binding:"required"`
	InstanceID    *uint  `json:"instance_id,omitempty"`
	Operation     string `json:"operation" binding:"required"`
	Unit          string `json:"unit" binding:"required"`
	Limit         int64  `json:"limit" binding:"min=0"`
	WindowSeconds int    `json:"window_seconds" binding:"min=1"`
	Enabled       *bool  `json:"enabled,omitempty"`
}

type UpdateProviderRateLimitRuleRequest struct {
	Scope         *string `json:"scope,omitempty"`
	InstanceID    *uint   `json:"instance_id,omitempty"`
	Operation     *string `json:"operation,omitempty"`
	Unit          *string `json:"unit,omitempty"`
	Limit         *int64  `json:"limit,omitempty"`
	WindowSeconds *int    `json:"window_seconds,omitempty"`
	Enabled       *bool   `json:"enabled,omitempty"`
}

// Create 创建限流规则
// @Summary 创建限流规则
// @Tags 多模态限流
// @Accept json
// @Produce json
// @Param request body CreateProviderRateLimitRuleRequest true "创建请求"
// @Success 200 {object} model.ProviderRateLimitRule
// @Router /api/admin/provider-rate-limit-rules [post]
func (h *ProviderRateLimitRuleHandler) Create(c *gin.Context) {
	var req CreateProviderRateLimitRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	opRaw := strings.TrimSpace(req.Operation)
	if opRaw == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "operation 不能为空"))
		return
	}
	op := model.NormalizeOperation(opRaw)
	unit := strings.TrimSpace(req.Unit)
	if unit == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "unit 不能为空"))
		return
	}
	unit = strings.ToLower(unit)

	scope := model.NormalizeRateLimitScope(req.Scope)
	var instanceID uint
	if scope == model.RateLimitScopeInstance {
		if req.InstanceID == nil || *req.InstanceID == 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "instance_id 不能为空"))
			return
		}
		instanceID = *req.InstanceID
	}

	window := req.WindowSeconds
	if window <= 0 {
		window = 60
	}

	existing, err := h.repo.GetByScopeOperationUnitWindow(c.Request.Context(), scope, req.ProviderID, instanceID, op, unit, window)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, model.ErrorResponse(model.ErrCodeConflict, "该限流规则已存在"))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	entity := &model.ProviderRateLimitRule{
		Scope:         scope,
		ProviderID:    req.ProviderID,
		InstanceID:    req.InstanceID,
		Operation:     op,
		Unit:          unit,
		Limit:         req.Limit,
		WindowSeconds: window,
		Enabled:       enabled,
	}
	if scope == model.RateLimitScopeProvider {
		entity.InstanceID = nil
	}

	if err := h.repo.Create(c.Request.Context(), entity); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	created, err := h.repo.GetByID(c.Request.Context(), entity.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, model.SuccessResponse(created))
}

// Update 更新限流规则
// @Summary 更新限流规则
// @Tags 多模态限流
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param request body UpdateProviderRateLimitRuleRequest true "更新请求"
// @Success 200 {object} model.ProviderRateLimitRule
// @Router /api/admin/provider-rate-limit-rules/{id} [put]
func (h *ProviderRateLimitRuleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	var req UpdateProviderRateLimitRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	entity, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if entity == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "记录不存在"))
		return
	}

	if req.Operation != nil {
		opRaw := strings.TrimSpace(*req.Operation)
		if opRaw == "" {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "operation 不能为空"))
			return
		}
		entity.Operation = model.NormalizeOperation(opRaw)
	}
	if req.Scope != nil {
		scope := model.NormalizeRateLimitScope(*req.Scope)
		entity.Scope = scope
		if scope == model.RateLimitScopeProvider {
			entity.InstanceID = nil
		}
	}
	if req.InstanceID != nil {
		if entity.Scope == model.RateLimitScopeInstance && *req.InstanceID == 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "instance_id 不能为空"))
			return
		}
		entity.InstanceID = req.InstanceID
	}
	if req.Unit != nil {
		unit := strings.TrimSpace(*req.Unit)
		if unit == "" {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "unit 不能为空"))
			return
		}
		entity.Unit = strings.ToLower(unit)
	}
	if req.Limit != nil {
		if *req.Limit < 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "limit 不能为负数"))
			return
		}
		entity.Limit = *req.Limit
	}
	if req.WindowSeconds != nil {
		if *req.WindowSeconds <= 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "window_seconds 必须大于0"))
			return
		}
		entity.WindowSeconds = *req.WindowSeconds
	}
	if req.Enabled != nil {
		entity.Enabled = *req.Enabled
	}

	if entity.Scope == model.RateLimitScopeInstance {
		if entity.InstanceID == nil || *entity.InstanceID == 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "scope=instance 时 instance_id 不能为空"))
			return
		}
	}

	var instanceID uint
	if entity.InstanceID != nil {
		instanceID = *entity.InstanceID
	}
	conflict, err := h.repo.GetByScopeOperationUnitWindow(c.Request.Context(), entity.Scope, entity.ProviderID, instanceID, entity.Operation, entity.Unit, entity.WindowSeconds)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if conflict != nil && conflict.ID != entity.ID {
		c.JSON(http.StatusConflict, model.ErrorResponse(model.ErrCodeConflict, "该限流规则已存在"))
		return
	}

	if err := h.repo.Update(c.Request.Context(), entity); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	updated, err := h.repo.GetByID(c.Request.Context(), entity.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	c.JSON(http.StatusOK, model.SuccessResponse(updated))
}

// GetByID 获取限流规则详情
// @Summary 获取限流规则详情
// @Tags 多模态限流
// @Produce json
// @Param id path int true "规则ID"
// @Success 200 {object} model.ProviderRateLimitRule
// @Router /api/admin/provider-rate-limit-rules/{id} [get]
func (h *ProviderRateLimitRuleHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	entity, err := h.repo.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if entity == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "记录不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(entity))
}

// List 查询限流规则列表
// @Summary 查询限流规则列表
// @Tags 多模态限流
// @Produce json
// @Param provider_id query int false "源头组ID"
// @Param scope query string false "范围(provider/instance)"
// @Param instance_id query int false "实例ID"
// @Param operation query string false "operation"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} model.Response
// @Router /api/admin/provider-rate-limit-rules [get]
func (h *ProviderRateLimitRuleHandler) List(c *gin.Context) {
	providerID := uint(parsePositiveInt(c.Query("provider_id"), 0))
	instanceID := uint(parsePositiveInt(c.Query("instance_id"), 0))
	scope := strings.TrimSpace(c.Query("scope"))
	operation := strings.TrimSpace(c.Query("operation"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	list, total, err := h.repo.List(c.Request.Context(), scope, providerID, instanceID, operation, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	pagination := model.NewPagination(page, pageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(list, pagination))
}

// Delete 删除限流规则
// @Summary 删除限流规则
// @Tags 多模态限流
// @Param id path int true "规则ID"
// @Success 200 {object} model.Response
// @Router /api/admin/provider-rate-limit-rules/{id} [delete]
func (h *ProviderRateLimitRuleHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.repo.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"deleted": true}))
}

func (h *ProviderRateLimitRuleHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/provider-rate-limit-rules")
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}
