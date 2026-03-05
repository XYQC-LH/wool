package handler

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/model"
	"nexus-api/internal/service"
	"nexus-api/internal/service/scheduler"

	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
)

// ModelProviderHandler 妯″瀷婧愬ご澶勭悊鍣?
type ModelProviderHandler struct {
	providerService service.ModelProviderService
	costCalculator  scheduler.CostCalculator
}

// NewModelProviderHandler 鍒涘缓妯″瀷婧愬ご澶勭悊鍣?
func NewModelProviderHandler(
	providerService service.ModelProviderService,
	costCalculator scheduler.CostCalculator,
) *ModelProviderHandler {
	return &ModelProviderHandler{
		providerService: providerService,
		costCalculator:  costCalculator,
	}
}

// CreateProviderRequest 鍒涘缓婧愬ご璇锋眰
type CreateProviderRequest struct {
	Operation                 string  `json:"operation,omitempty"`
	ModelID                   string  `json:"model_id" binding:"required"`
	ChannelID                 uint    `json:"channel_id" binding:"required"`
	UpstreamModelName         string  `json:"upstream_model_name" binding:"required,min=1,max=100"`
	ActualCostPer1kInput      float64 `json:"actual_cost_per_1k_input" binding:"required"`
	ActualCostPer1kOutput     float64 `json:"actual_cost_per_1k_output" binding:"required"`
	IsCostPriority            *bool   `json:"is_cost_priority,omitempty"`
	Priority                  int     `json:"priority"`
	Weight                    int     `json:"weight"`
	ConnectTimeoutMs          int     `json:"connect_timeout_ms"`
	AttemptTimeoutMs          int     `json:"attempt_timeout_ms"`
	StreamFirstChunkTimeoutMs int     `json:"stream_first_chunk_timeout_ms"`
	FailureThreshold          int     `json:"failure_threshold"`
	RecoveryTimeoutSeconds    int     `json:"recovery_timeout_seconds"`
	Status                    string  `json:"status,omitempty" binding:"omitempty,oneof=active disabled cooling"`
}

// UpdateProviderRequest 鏇存柊婧愬ご璇锋眰
type UpdateProviderRequest struct {
	ActualCostPer1kInput      *float64 `json:"actual_cost_per_1k_input,omitempty"`
	ActualCostPer1kOutput     *float64 `json:"actual_cost_per_1k_output,omitempty"`
	UpstreamModelName         *string  `json:"upstream_model_name,omitempty" binding:"omitempty,min=1,max=100"`
	IsCostPriority            *bool    `json:"is_cost_priority,omitempty"`
	Priority                  *int     `json:"priority,omitempty"`
	Weight                    *int     `json:"weight,omitempty"`
	ConnectTimeoutMs          *int     `json:"connect_timeout_ms,omitempty"`
	AttemptTimeoutMs          *int     `json:"attempt_timeout_ms,omitempty"`
	StreamFirstChunkTimeoutMs *int     `json:"stream_first_chunk_timeout_ms,omitempty"`
	FailureThreshold          *int     `json:"failure_threshold,omitempty"`
	RecoveryTimeoutSeconds    *int     `json:"recovery_timeout_seconds,omitempty"`
	Status                    *string  `json:"status,omitempty" binding:"omitempty,oneof=active disabled cooling"`
}

// CircuitActionRequest 鐔旀柇鎿嶄綔璇锋眰
type CircuitActionRequest struct {
	Action   string `json:"action" binding:"required,oneof=open close"` // open 鎴?close
	Duration int    `json:"duration,omitempty"`                         // 鐔旀柇鏃堕暱锛堢锛夛紝浠?open 鏃舵湁鏁?
	Reason   string `json:"reason,omitempty"`                           // 鍘熷洜
}

// BatchStatusRequest 鎵归噺鐘舵€佹洿鏂拌姹?
type BatchStatusRequest struct {
	IDs    []uint `json:"ids" binding:"required"`
	Status string `json:"status" binding:"required,oneof=active disabled cooling"`
}

func buildOpenAIV1URL(baseURL string, endpoint string) (string, error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", fmt.Errorf("base_url 不能为空")
	}

	endpoint = strings.TrimSpace(endpoint)
	if endpoint == "" {
		return "", fmt.Errorf("endpoint 不能为空")
	}
	if !strings.HasPrefix(endpoint, "/") {
		endpoint = "/" + endpoint
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	if u.Scheme == "" || u.Host == "" {
		return "", fmt.Errorf("无效 base_url: %s", baseURL)
	}

	basePath := strings.TrimRight(u.Path, "/")
	v1Prefix := "/v1"
	if basePath == "/v1" || strings.HasSuffix(basePath, "/v1") {
		v1Prefix = ""
	}
	u.Path = basePath + v1Prefix + endpoint
	return u.String(), nil
}

type ProviderTestResult struct {
	OK         bool   `json:"ok"`
	StatusCode int    `json:"status_code"`
	LatencyMs  int64  `json:"latency_ms"`
	Message    string `json:"message,omitempty"`
}

type BatchHealthCheckRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

type BatchProviderTestResult struct {
	ProviderID uint `json:"provider_id"`
	ProviderTestResult
}

// Create 鍒涘缓妯″瀷婧愬ご
// @Summary 鍒涘缓妯″瀷婧愬ご
// @Description 鍒涘缓鏂扮殑妯″瀷婧愬ご閰嶇疆
// @Tags 妯″瀷婧愬ご
// @Accept json
// @Produce json
// @Param request body CreateProviderRequest true "鍒涘缓璇锋眰"
// @Success 200 {object} model.ModelProviderResponse
// @Router /api/admin/providers [post]
func (h *ModelProviderHandler) Create(c *gin.Context) {
	var req CreateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	// 杞崲涓烘湇鍔″眰璇锋眰
	serviceReq := &service.CreateProviderRequest{
		Operation:                 req.Operation,
		ModelID:                   req.ModelID,
		ChannelID:                 req.ChannelID,
		UpstreamModelName:         req.UpstreamModelName,
		ActualCostPer1kInput:      decimal.NewFromFloat(req.ActualCostPer1kInput),
		ActualCostPer1kOutput:     decimal.NewFromFloat(req.ActualCostPer1kOutput),
		IsCostPriority:            req.IsCostPriority,
		Priority:                  req.Priority,
		Weight:                    req.Weight,
		ConnectTimeoutMs:          req.ConnectTimeoutMs,
		AttemptTimeoutMs:          req.AttemptTimeoutMs,
		StreamFirstChunkTimeoutMs: req.StreamFirstChunkTimeoutMs,
		FailureThreshold:          req.FailureThreshold,
		RecoveryTimeoutSeconds:    req.RecoveryTimeoutSeconds,
		Status:                    model.ProviderStatus(req.Status),
	}

	provider, err := h.providerService.Create(c.Request.Context(), serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(provider.ToResponse()))
}

// Update 鏇存柊妯″瀷婧愬ご
// @Summary 鏇存柊妯″瀷婧愬ご
// @Description 鏇存柊妯″瀷婧愬ご閰嶇疆
// @Tags 妯″瀷婧愬ご
// @Accept json
// @Produce json
// @Param id path int true "婧愬ごID"
// @Param request body UpdateProviderRequest true "鏇存柊璇锋眰"
// @Success 200 {object} model.ModelProviderResponse
// @Router /api/admin/providers/{id} [put]
func (h *ModelProviderHandler) Update(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	var req UpdateProviderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	// 杞崲涓烘湇鍔″眰璇锋眰
	serviceReq := &service.UpdateProviderRequest{}
	if req.ActualCostPer1kInput != nil {
		cost := decimal.NewFromFloat(*req.ActualCostPer1kInput)
		serviceReq.ActualCostPer1kInput = &cost
	}
	if req.ActualCostPer1kOutput != nil {
		cost := decimal.NewFromFloat(*req.ActualCostPer1kOutput)
		serviceReq.ActualCostPer1kOutput = &cost
	}
	if req.UpstreamModelName != nil {
		serviceReq.UpstreamModelName = req.UpstreamModelName
	}
	if req.IsCostPriority != nil {
		serviceReq.IsCostPriority = req.IsCostPriority
	}
	if req.Priority != nil {
		serviceReq.Priority = req.Priority
	}
	if req.Weight != nil {
		serviceReq.Weight = req.Weight
	}
	if req.ConnectTimeoutMs != nil {
		serviceReq.ConnectTimeoutMs = req.ConnectTimeoutMs
	}
	if req.AttemptTimeoutMs != nil {
		serviceReq.AttemptTimeoutMs = req.AttemptTimeoutMs
	}
	if req.StreamFirstChunkTimeoutMs != nil {
		serviceReq.StreamFirstChunkTimeoutMs = req.StreamFirstChunkTimeoutMs
	}
	if req.FailureThreshold != nil {
		serviceReq.FailureThreshold = req.FailureThreshold
	}
	if req.RecoveryTimeoutSeconds != nil {
		serviceReq.RecoveryTimeoutSeconds = req.RecoveryTimeoutSeconds
	}
	if req.Status != nil {
		status := model.ProviderStatus(*req.Status)
		serviceReq.Status = &status
	}

	provider, err := h.providerService.Update(c.Request.Context(), uint(id), serviceReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(provider.ToResponse()))
}

// Delete 鍒犻櫎妯″瀷婧愬ご
// @Summary 鍒犻櫎妯″瀷婧愬ご
// @Description 鍒犻櫎妯″瀷婧愬ご閰嶇疆
// @Tags 妯″瀷婧愬ご
// @Param id path int true "婧愬ごID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/providers/{id} [delete]
func (h *ModelProviderHandler) Delete(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.providerService.Delete(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "删除成功"}))
}

// GetByID 鑾峰彇鍗曚釜妯″瀷婧愬ご
// @Summary 鑾峰彇妯″瀷婧愬ご璇︽儏
// @Description 鏍规嵁ID鑾峰彇妯″瀷婧愬ご璇︽儏
// @Tags 妯″瀷婧愬ご
// @Param id path int true "婧愬ごID"
// @Success 200 {object} model.ModelProviderResponse
// @Router /api/admin/providers/{id} [get]
func (h *ModelProviderHandler) GetByID(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	provider, err := h.providerService.GetByID(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if provider == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "源头不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(provider))
}

// List 鑾峰彇妯″瀷婧愬ご鍒楄〃
// @Summary 鑾峰彇妯″瀷婧愬ご鍒楄〃
// @Description 鍒嗛〉鑾峰彇妯″瀷婧愬ご鍒楄〃
// @Tags 妯″瀷婧愬ご
// @Param model_id query string false "妯″瀷ID"
// @Param channel_id query int false "娓犻亾ID"
// @Param status query string false "鐘舵€?
// @Param circuit_state query string false "鐔旀柇鐘舵€?
// @Param page query int false "椤电爜"
// @Param page_size query int false "姣忛〉鏁伴噺"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/providers [get]
func (h *ModelProviderHandler) List(c *gin.Context) {
	params := &model.ProviderQueryParams{
		Operation:    c.Query("operation"),
		ModelID:      c.Query("model_id"),
		Status:       c.Query("status"),
		CircuitState: model.CircuitState(c.Query("circuit_state")),
	}

	if channelID := c.Query("channel_id"); channelID != "" {
		if id, err := strconv.ParseUint(channelID, 10, 32); err == nil {
			params.ChannelID = uint(id)
		}
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

	providers, total, err := h.providerService.List(c.Request.Context(), params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	pagination := model.NewPagination(params.Page, params.PageSize, total)
	c.JSON(http.StatusOK, model.PaginatedSuccessResponse(providers, pagination))
}

// GetByModelID 鏍规嵁妯″瀷ID鑾峰彇婧愬ご鍒楄〃
// @Summary 鏍规嵁妯″瀷ID鑾峰彇婧愬ご鍒楄〃
// @Description 鑾峰彇鎸囧畾妯″瀷鐨勬墍鏈夋簮澶?
// @Tags 妯″瀷婧愬ご
// @Param model_id path string true "妯″瀷ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/providers/model/{model_id} [get]
func (h *ModelProviderHandler) GetByModelID(c *gin.Context) {
	modelID := c.Param("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "模型ID不能为空"))
		return
	}

	operation := c.Query("operation")
	providers, err := h.providerService.GetByModelID(c.Request.Context(), operation, modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(providers))
}

// GetByChannelID 鏍规嵁娓犻亾ID鑾峰彇婧愬ご鍒楄〃
// @Summary 鏍规嵁娓犻亾ID鑾峰彇婧愬ご鍒楄〃
// @Description 鑾峰彇鎸囧畾娓犻亾鐨勬墍鏈夋簮澶?
// @Tags 妯″瀷婧愬ご
// @Param channel_id path int true "娓犻亾ID"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/providers/channel/{channel_id} [get]
func (h *ModelProviderHandler) GetByChannelID(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("channel_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道ID"))
		return
	}

	providers, err := h.providerService.GetByChannelID(c.Request.Context(), uint(channelID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(providers))
}

// Enable 鍚敤婧愬ご
// @Summary 鍚敤婧愬ご
// @Description 鍚敤鎸囧畾鐨勬ā鍨嬫簮澶?
// @Tags 妯″瀷婧愬ご
// @Param id path int true "婧愬ごID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/providers/{id}/enable [post]
func (h *ModelProviderHandler) Enable(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.providerService.Enable(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "启用成功"}))
}

// Disable 绂佺敤婧愬ご
// @Summary 绂佺敤婧愬ご
// @Description 绂佺敤鎸囧畾鐨勬ā鍨嬫簮澶?
// @Tags 妯″瀷婧愬ご
// @Param id path int true "婧愬ごID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/providers/{id}/disable [post]
func (h *ModelProviderHandler) Disable(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.providerService.Disable(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "禁用成功"}))
}

// BatchUpdateStatus 鎵归噺鏇存柊鐘舵€?
// @Summary 鎵归噺鏇存柊婧愬ご鐘舵€?
// @Description 鎵归噺鏇存柊澶氫釜婧愬ご鐨勭姸鎬?
// @Tags 妯″瀷婧愬ご
// @Accept json
// @Produce json
// @Param request body BatchStatusRequest true "鎵归噺鏇存柊璇锋眰"
// @Success 200 {object} map[string]bool
// @Router /api/admin/providers/batch/status [post]
func (h *ModelProviderHandler) BatchUpdateStatus(c *gin.Context) {
	var req BatchStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	if err := h.providerService.BatchUpdateStatus(c.Request.Context(), req.IDs, model.ProviderStatus(req.Status)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "批量更新成功"}))
}

// CircuitAction 鐔旀柇鎿嶄綔
// @Summary 鐔旀柇鎿嶄綔
// @Description 鎵嬪姩鎵撳紑鎴栧叧闂啍鏂櫒
// @Tags 妯″瀷婧愬ご
// @Accept json
// @Produce json
// @Param id path int true "婧愬ごID"
// @Param request body CircuitActionRequest true "鐔旀柇鎿嶄綔璇锋眰"
// @Success 200 {object} map[string]bool
// @Router /api/admin/providers/{id}/circuit [post]
func (h *ModelProviderHandler) CircuitAction(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	var req CircuitActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	switch req.Action {
	case "open":
		duration := time.Duration(req.Duration) * time.Second
		if duration <= 0 {
			duration = 30 * time.Second // 榛樿30绉?
		}
		if err := h.providerService.OpenCircuit(c.Request.Context(), uint(id), duration, req.Reason); err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "熔断器已打开"}))

	case "close":
		if err := h.providerService.CloseCircuit(c.Request.Context(), uint(id)); err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
			return
		}
		c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "熔断器已关闭"}))

	default:
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的操作"))
	}
}

// ResetCircuit 重置源头组熔断状态
// @Summary 重置源头组熔断状态
// @Description 手动关闭熔断器并重置连续失败计数/半开计数
// @Tags 模型源头
// @Param id path int true "源头ID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/providers/{id}/reset-circuit [post]
func (h *ModelProviderHandler) ResetCircuit(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	if err := h.providerService.CloseCircuit(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "熔断器已重置"}))
}

// GetCircuitInfo 鑾峰彇鐔旀柇鍣ㄤ俊鎭?
// @Summary 鑾峰彇鐔旀柇鍣ㄤ俊鎭?
// @Description 鑾峰彇鎸囧畾婧愬ご鐨勭啍鏂櫒鐘舵€佷俊鎭?
// @Tags 妯″瀷婧愬ご
// @Param id path int true "婧愬ごID"
// @Success 200 {object} scheduler.CircuitInfo
// @Router /api/admin/providers/{id}/circuit [get]
func (h *ModelProviderHandler) GetCircuitInfo(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	info, err := h.providerService.GetCircuitInfo(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if info == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "源头不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(info))
}

// GetProviderHealth 鑾峰彇婧愬ご鍋ュ悍璇︽儏
// @Summary 鑾峰彇婧愬ご鍋ュ悍璇︽儏
// @Description 鑾峰彇鎸囧畾婧愬ご鐨勫仴搴风姸鎬佽鎯?
// @Tags 妯″瀷婧愬ご
// @Param id path int true "婧愬ごID"
// @Success 200 {object} scheduler.ProviderHealth
// @Router /api/admin/providers/{id}/health [get]
func (h *ModelProviderHandler) GetProviderHealth(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的ID"))
		return
	}

	health, err := h.providerService.GetProviderHealth(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if health == nil {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "源头不存在"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(health))
}

// GetModelHealth 鑾峰彇妯″瀷鍋ュ悍姒傝
// @Summary 鑾峰彇妯″瀷鍋ュ悍姒傝
// @Description 鑾峰彇鎸囧畾妯″瀷鐨勫仴搴风姸鎬佹瑙?
// @Tags 妯″瀷婧愬ご
// @Param model_id path string true "妯″瀷ID"
// @Success 200 {object} scheduler.ModelHealth
// @Router /api/admin/providers/model/{model_id}/health [get]
func (h *ModelProviderHandler) GetModelHealth(c *gin.Context) {
	modelID := c.Param("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "模型ID不能为空"))
		return
	}

	operation := c.Query("operation")
	health, err := h.providerService.GetModelHealth(c.Request.Context(), operation, modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(health))
}

// GetHealthSummary 鑾峰彇鍋ュ悍鎽樿
// @Summary 鑾峰彇鍋ュ悍鎽樿
// @Description 鑾峰彇鎵€鏈夋簮澶寸殑鍋ュ悍鐘舵€佹憳瑕?
// @Tags 妯″瀷婧愬ご
// @Success 200 {object} scheduler.HealthSummary
// @Router /api/admin/providers/health/summary [get]
func (h *ModelProviderHandler) GetHealthSummary(c *gin.Context) {
	summary, err := h.providerService.GetHealthSummary(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(summary))
}

// SyncFromChannelModels 浠庢笭閬撴ā鍨嬪悓姝?
// @Summary 浠庢笭閬撴ā鍨嬪悓姝ユ簮澶?
// @Description 浠庢寚瀹氭笭閬撶殑妯″瀷閰嶇疆鍚屾鍒涘缓婧愬ご
// @Tags 妯″瀷婧愬ご
// @Param channel_id path int true "娓犻亾ID"
// @Success 200 {object} map[string]bool
// @Router /api/admin/providers/sync/channel/{channel_id} [post]
func (h *ModelProviderHandler) SyncFromChannelModels(c *gin.Context) {
	channelID, err := strconv.ParseUint(c.Param("channel_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的渠道ID"))
		return
	}

	if err := h.providerService.SyncFromChannelModels(c.Request.Context(), uint(channelID)); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"message": "同步成功"}))
}

// GetCostAnalysis 鑾峰彇鎴愭湰鍒嗘瀽
// @Summary 鑾峰彇鎴愭湰鍒嗘瀽
// @Description 鑾峰彇鎸囧畾妯″瀷鐨勬垚鏈垎鏋愭姤鍛?
// @Tags 妯″瀷婧愬ご
// @Param model_id path string true "妯″瀷ID"
// @Param start_date query string false "寮€濮嬫棩鏈?(YYYY-MM-DD)"
// @Param end_date query string false "缁撴潫鏃ユ湡 (YYYY-MM-DD)"
// @Success 200 {object} scheduler.CostAnalysis
// @Router /api/admin/providers/cost/analysis/{model_id} [get]
func (h *ModelProviderHandler) GetCostAnalysis(c *gin.Context) {
	modelID := c.Param("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "模型ID不能为空"))
		return
	}

	// 瑙ｆ瀽鏃堕棿鑼冨洿
	startDateStr := c.Query("start_date")
	endDateStr := c.Query("end_date")

	var startTime, endTime time.Time
	var err error

	if startDateStr != "" {
		startTime, err = time.Parse("2006-01-02", startDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的开始日期格式"))
			return
		}
	} else {
		// 榛樿鏈€杩?澶?
		startTime = time.Now().AddDate(0, 0, -7)
	}

	if endDateStr != "" {
		endTime, err = time.Parse("2006-01-02", endDateStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的结束日期格式"))
			return
		}
	} else {
		// 榛樿鍒扮幇鍦?
		endTime = time.Now()
	}

	analysis, err := h.costCalculator.AnalyzeCosts(modelID, startTime, endTime)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(analysis))
}

// GetCostBreakdown 鑾峰彇鎴愭湰鏄庣粏
// @Summary 鑾峰彇鎴愭湰鏄庣粏
// @Description 鑾峰彇鎸囧畾妯″瀷鐨勬垚鏈槑缁?
// @Tags 妯″瀷婧愬ご
// @Param model_id path string true "妯″瀷ID"
// @Param prompt_tokens query int false "Prompt tokens"
// @Param completion_tokens query int false "Completion tokens"
// @Success 200 {object} scheduler.CostBreakdown
// @Router /api/admin/providers/cost/breakdown/{model_id} [get]
func (h *ModelProviderHandler) GetCostBreakdown(c *gin.Context) {
	modelID := c.Param("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "模型ID不能为空"))
		return
	}

	promptTokens := 1000 // 榛樿鍊?
	if pt := c.Query("prompt_tokens"); pt != "" {
		if p, err := strconv.Atoi(pt); err == nil {
			promptTokens = p
		}
	}

	completionTokens := 500 // 榛樿鍊?
	if ct := c.Query("completion_tokens"); ct != "" {
		if c, err := strconv.Atoi(ct); err == nil {
			completionTokens = c
		}
	}

	breakdown, err := h.costCalculator.GetCostBreakdown(modelID, promptTokens, completionTokens)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(breakdown))
}

// GetCostOptimizationSuggestions 鑾峰彇鎴愭湰浼樺寲寤鸿
// @Summary 鑾峰彇鎴愭湰浼樺寲寤鸿
// @Description 鑾峰彇鎸囧畾妯″瀷鐨勬垚鏈紭鍖栧缓璁?
// @Tags 妯″瀷婧愬ご
// @Param model_id path string true "妯″瀷ID"
// @Success 200 {object} []scheduler.CostOptimizationSuggestion
// @Router /api/admin/providers/cost/optimization/{model_id} [get]
func (h *ModelProviderHandler) GetCostOptimizationSuggestions(c *gin.Context) {
	modelID := c.Param("model_id")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "模型ID不能为空"))
		return
	}

	suggestions, err := h.costCalculator.GetCostOptimizationSuggestions(modelID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(suggestions))
}

// TestProvider 测试源头连通性
// @Summary 测试源头连通性
// @Description 对指定源头发起最小探测请求（GET /v1/models），用于验证 base_url 与 api_key 是否可用
// @Tags 模型源头
// @Param id path int true "源头ID"
// @Success 200 {object} ProviderTestResult
// @Router /api/admin/providers/{id}/test [post]
func (h *ModelProviderHandler) TestProvider(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的源头ID"))
		return
	}

	result, statusCode, err := h.executeProviderHealthCheck(c.Request.Context(), uint(id))
	if err != nil {
		errCode := model.ErrCodeInternalError
		if statusCode == http.StatusBadRequest {
			errCode = model.ErrCodeInvalidRequest
		}
		if statusCode == http.StatusNotFound {
			errCode = model.ErrCodeNotFound
		}
		c.JSON(statusCode, model.ErrorResponse(errCode, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

// HealthCheck 源头健康检查（与 TestProvider 语义一致）
// @Summary 源头健康检查
// @Description 对指定源头发起最小探测请求（GET /v1/models）
// @Tags 模型源头
// @Param id path int true "源头ID"
// @Success 200 {object} ProviderTestResult
// @Router /api/admin/providers/{id}/health-check [get]
func (h *ModelProviderHandler) HealthCheck(c *gin.Context) {
	h.TestProvider(c)
}

// BatchHealthCheck 批量健康检查
// @Summary 批量健康检查
// @Description 批量探测多个源头连通性
// @Tags 模型源头
// @Accept json
// @Produce json
// @Param request body BatchHealthCheckRequest true "批量检查请求"
// @Success 200 {object} map[string]interface{}
// @Router /api/admin/providers/batch/health-check [post]
func (h *ModelProviderHandler) BatchHealthCheck(c *gin.Context) {
	var req BatchHealthCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	results := make([]BatchProviderTestResult, 0, len(req.IDs))
	for _, providerID := range req.IDs {
		item := BatchProviderTestResult{ProviderID: providerID}

		result, _, err := h.executeProviderHealthCheck(c.Request.Context(), providerID)
		if err != nil {
			item.ProviderTestResult = ProviderTestResult{
				OK:         false,
				StatusCode: 0,
				LatencyMs:  0,
				Message:    err.Error(),
			}
		} else {
			item.ProviderTestResult = *result
		}

		results = append(results, item)
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"results": results}))
}

func (h *ModelProviderHandler) executeProviderHealthCheck(ctx context.Context, providerID uint) (*ProviderTestResult, int, error) {
	entity, err := h.providerService.GetEntityByID(ctx, providerID)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if entity == nil || entity.Channel == nil {
		return nil, http.StatusNotFound, fmt.Errorf("源头不存在或缺少渠道信息")
	}
	return h.executeProviderEntityHealthCheck(ctx, entity)
}

func (h *ModelProviderHandler) executeProviderEntityHealthCheck(ctx context.Context, entity *model.ModelProvider) (*ProviderTestResult, int, error) {
	testURL, err := buildOpenAIV1URL(entity.Channel.BaseURL, "/models")
	if err != nil {
		return nil, http.StatusBadRequest, err
	}

	timeoutMs := entity.AttemptTimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = 5000
	}
	requestCtx, cancel := context.WithTimeout(ctx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	start := time.Now()
	req, err := http.NewRequestWithContext(requestCtx, http.MethodGet, testURL, nil)
	if err != nil {
		return nil, http.StatusInternalServerError, err
	}
	if strings.TrimSpace(entity.Channel.APIKey) != "" {
		req.Header.Set("Authorization", "Bearer "+strings.TrimSpace(entity.Channel.APIKey))
	}

	resp, err := http.DefaultClient.Do(req)
	latencyMs := time.Since(start).Milliseconds()
	if err != nil {
		return &ProviderTestResult{
			OK:         false,
			StatusCode: 0,
			LatencyMs:  latencyMs,
			Message:    err.Error(),
		}, http.StatusOK, nil
	}
	defer resp.Body.Close()

	ok := resp.StatusCode >= 200 && resp.StatusCode < 300
	msg := resp.Status
	if !ok {
		msg = "上游返回非 2xx: " + resp.Status
	}

	return &ProviderTestResult{
		OK:         ok,
		StatusCode: resp.StatusCode,
		LatencyMs:  latencyMs,
		Message:    msg,
	}, http.StatusOK, nil
}

// RegisterRoutes 娉ㄥ唽璺敱
func (h *ModelProviderHandler) RegisterRoutes(router *gin.RouterGroup) {
	providers := router.Group("/providers")
	{
		// 鍩虹 CRUD
		providers.POST("", h.Create)
		providers.GET("", h.List)
		providers.GET("/:id", h.GetByID)
		providers.PUT("/:id", h.Update)
		providers.DELETE("/:id", h.Delete)

		// 连通性测试
		providers.POST("/:id/test", h.TestProvider)
		providers.GET("/:id/health-check", h.HealthCheck)
		providers.POST("/batch/health-check", h.BatchHealthCheck)

		// 鐘舵€佺鐞?
		providers.POST("/:id/enable", h.Enable)
		providers.POST("/:id/disable", h.Disable)
		providers.POST("/batch/status", h.BatchUpdateStatus)

		// 鐔旀柇绠＄悊
		providers.GET("/:id/circuit", h.GetCircuitInfo)
		providers.POST("/:id/circuit", h.CircuitAction)
		providers.POST("/:id/reset-circuit", h.ResetCircuit)

		// 鍋ュ悍绠＄悊
		providers.GET("/:id/health", h.GetProviderHealth)
		providers.GET("/health/summary", h.GetHealthSummary)

		// 鎸夋ā鍨?娓犻亾鏌ヨ
		providers.GET("/model/:model_id", h.GetByModelID)
		providers.GET("/model/:model_id/health", h.GetModelHealth)
		providers.GET("/channel/:channel_id", h.GetByChannelID)

		// 鍚屾
		providers.POST("/sync/channel/:channel_id", h.SyncFromChannelModels)

		// 鎴愭湰鍒嗘瀽
		providers.GET("/cost/analysis/:model_id", h.GetCostAnalysis)
		providers.GET("/cost/breakdown/:model_id", h.GetCostBreakdown)
		providers.GET("/cost/optimization/:model_id", h.GetCostOptimizationSuggestions)
	}
}
