package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// ProviderPricingRuleHandler 多模态计费规则处理器
type ProviderPricingRuleHandler struct {
	repo repository.ProviderPricingRuleRepository
}

func NewProviderPricingRuleHandler(repo repository.ProviderPricingRuleRepository) *ProviderPricingRuleHandler {
	return &ProviderPricingRuleHandler{repo: repo}
}

type CreateProviderPricingRuleRequest struct {
	ProviderID   uint       `json:"provider_id" binding:"required"`
	Operation    string     `json:"operation" binding:"required"`
	Unit         string     `json:"unit" binding:"required"`
	CostPerUnit  float64    `json:"cost_per_unit" binding:"min=0"`
	PricePerUnit float64    `json:"price_per_unit" binding:"min=0"`
	Meta         model.JSON `json:"meta,omitempty"`
	Enabled      *bool      `json:"enabled,omitempty"`
}

type UpdateProviderPricingRuleRequest struct {
	Operation    *string     `json:"operation,omitempty"`
	Unit         *string     `json:"unit,omitempty"`
	CostPerUnit  *float64    `json:"cost_per_unit,omitempty"`
	PricePerUnit *float64    `json:"price_per_unit,omitempty"`
	Meta         *model.JSON `json:"meta,omitempty"`
	Enabled      *bool       `json:"enabled,omitempty"`
}

// Create 创建计费规则
// @Summary 创建计费规则
// @Tags 多模态计费
// @Accept json
// @Produce json
// @Param request body CreateProviderPricingRuleRequest true "创建请求"
// @Success 200 {object} model.ProviderPricingRule
// @Router /api/admin/provider-pricing-rules [post]
func (h *ProviderPricingRuleHandler) Create(c *gin.Context) {
	var req CreateProviderPricingRuleRequest
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

	existing, err := h.repo.GetByProviderOperationUnit(c.Request.Context(), req.ProviderID, op, unit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, model.ErrorResponse(model.ErrCodeConflict, "该计费规则已存在"))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	entity := &model.ProviderPricingRule{
		ProviderID:   req.ProviderID,
		Operation:    op,
		Unit:         unit,
		CostPerUnit:  decimal.NewFromFloat(req.CostPerUnit),
		PricePerUnit: decimal.NewFromFloat(req.PricePerUnit),
		Meta:         req.Meta,
		Enabled:      enabled,
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

// Update 更新计费规则
// @Summary 更新计费规则
// @Tags 多模态计费
// @Accept json
// @Produce json
// @Param id path int true "规则ID"
// @Param request body UpdateProviderPricingRuleRequest true "更新请求"
// @Success 200 {object} model.ProviderPricingRule
// @Router /api/admin/provider-pricing-rules/{id} [put]
func (h *ProviderPricingRuleHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	var req UpdateProviderPricingRuleRequest
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
	if req.Unit != nil {
		unit := strings.TrimSpace(*req.Unit)
		if unit == "" {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "unit 不能为空"))
			return
		}
		entity.Unit = strings.ToLower(unit)
	}
	if req.CostPerUnit != nil {
		if *req.CostPerUnit < 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "cost_per_unit 不能为负数"))
			return
		}
		entity.CostPerUnit = decimal.NewFromFloat(*req.CostPerUnit)
	}
	if req.PricePerUnit != nil {
		if *req.PricePerUnit < 0 {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "price_per_unit 不能为负数"))
			return
		}
		entity.PricePerUnit = decimal.NewFromFloat(*req.PricePerUnit)
	}
	if req.Meta != nil {
		entity.Meta = *req.Meta
	}
	if req.Enabled != nil {
		entity.Enabled = *req.Enabled
	}

	// 唯一性冲突检查（provider_id + operation + unit）
	conflict, err := h.repo.GetByProviderOperationUnit(c.Request.Context(), entity.ProviderID, entity.Operation, entity.Unit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if conflict != nil && conflict.ID != entity.ID {
		c.JSON(http.StatusConflict, model.ErrorResponse(model.ErrCodeConflict, "该计费规则已存在"))
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

// GetByID 获取计费规则详情
// @Summary 获取计费规则详情
// @Tags 多模态计费
// @Produce json
// @Param id path int true "规则ID"
// @Success 200 {object} model.ProviderPricingRule
// @Router /api/admin/provider-pricing-rules/{id} [get]
func (h *ProviderPricingRuleHandler) GetByID(c *gin.Context) {
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

// List 查询计费规则列表
// @Summary 查询计费规则列表
// @Tags 多模态计费
// @Produce json
// @Param provider_id query int false "源头组ID"
// @Param operation query string false "operation"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} model.Response
// @Router /api/admin/provider-pricing-rules [get]
func (h *ProviderPricingRuleHandler) List(c *gin.Context) {
	providerID := uint(parsePositiveInt(c.Query("provider_id"), 0))
	operation := strings.TrimSpace(c.Query("operation"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	list, total, err := h.repo.List(c.Request.Context(), providerID, operation, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	pagination := model.NewPagination(page, pageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(list, pagination))
}

// Delete 删除计费规则
// @Summary 删除计费规则
// @Tags 多模态计费
// @Param id path int true "规则ID"
// @Success 200 {object} model.Response
// @Router /api/admin/provider-pricing-rules/{id} [delete]
func (h *ProviderPricingRuleHandler) Delete(c *gin.Context) {
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

func (h *ProviderPricingRuleHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/provider-pricing-rules")
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}
