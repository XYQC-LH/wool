package handler

import (
	"net/http"
	"strconv"
	"strings"

	"nexus-api/internal/model"
	"nexus-api/internal/repository"

	"github.com/gin-gonic/gin"
)

// ModelCapabilityHandler 模型能力配置处理器
type ModelCapabilityHandler struct {
	repo repository.ModelCapabilityRepository
}

func NewModelCapabilityHandler(repo repository.ModelCapabilityRepository) *ModelCapabilityHandler {
	return &ModelCapabilityHandler{repo: repo}
}

type CreateModelCapabilityRequest struct {
	ModelID   string `json:"model_id" binding:"required"`
	Operation string `json:"operation" binding:"required"`
	Enabled   *bool  `json:"enabled,omitempty"`
}

type UpdateModelCapabilityRequest struct {
	Enabled *bool `json:"enabled,omitempty"`
}

// Create 创建模型能力
// @Summary 创建模型能力
// @Tags 模型能力
// @Accept json
// @Produce json
// @Param request body CreateModelCapabilityRequest true "创建请求"
// @Success 200 {object} model.ModelCapability
// @Router /api/admin/model-capabilities [post]
func (h *ModelCapabilityHandler) Create(c *gin.Context) {
	var req CreateModelCapabilityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	req.ModelID = strings.TrimSpace(req.ModelID)
	if req.ModelID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "model_id 不能为空"))
		return
	}
	op := model.NormalizeOperation(req.Operation)
	if strings.TrimSpace(op) == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "operation 不能为空"))
		return
	}

	existing, err := h.repo.GetByModelAndOperation(c.Request.Context(), req.ModelID, op)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if existing != nil {
		c.JSON(http.StatusConflict, model.ErrorResponse(model.ErrCodeConflict, "该模型能力已存在"))
		return
	}

	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}

	entity := &model.ModelCapability{
		ModelID:   req.ModelID,
		Operation: op,
		Enabled:   enabled,
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

// Update 更新模型能力（仅更新 enabled）
// @Summary 更新模型能力
// @Tags 模型能力
// @Accept json
// @Produce json
// @Param id path int true "能力ID"
// @Param request body UpdateModelCapabilityRequest true "更新请求"
// @Success 200 {object} model.ModelCapability
// @Router /api/admin/model-capabilities/{id} [put]
func (h *ModelCapabilityHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	var req UpdateModelCapabilityRequest
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

	if req.Enabled != nil {
		entity.Enabled = *req.Enabled
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

// GetByID 获取模型能力详情
// @Summary 获取模型能力详情
// @Tags 模型能力
// @Produce json
// @Param id path int true "能力ID"
// @Success 200 {object} model.ModelCapability
// @Router /api/admin/model-capabilities/{id} [get]
func (h *ModelCapabilityHandler) GetByID(c *gin.Context) {
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

// List 查询模型能力列表
// @Summary 查询模型能力列表
// @Tags 模型能力
// @Produce json
// @Param model_id query string false "模型ID"
// @Param operation query string false "operation"
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(20)
// @Success 200 {object} model.Response
// @Router /api/admin/model-capabilities [get]
func (h *ModelCapabilityHandler) List(c *gin.Context) {
	modelID := strings.TrimSpace(c.Query("model_id"))
	operation := strings.TrimSpace(c.Query("operation"))
	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)

	list, total, err := h.repo.List(c.Request.Context(), modelID, operation, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	pagination := model.NewPagination(page, pageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(list, pagination))
}

// Delete 删除模型能力
// @Summary 删除模型能力
// @Tags 模型能力
// @Param id path int true "能力ID"
// @Success 200 {object} model.Response
// @Router /api/admin/model-capabilities/{id} [delete]
func (h *ModelCapabilityHandler) Delete(c *gin.Context) {
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

func (h *ModelCapabilityHandler) RegisterRoutes(router *gin.RouterGroup) {
	group := router.Group("/model-capabilities")
	{
		group.GET("", h.List)
		group.POST("", h.Create)
		group.GET("/:id", h.GetByID)
		group.PUT("/:id", h.Update)
		group.DELETE("/:id", h.Delete)
	}
}
