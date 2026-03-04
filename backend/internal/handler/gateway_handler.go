package handler

import (
	"net/http"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

// GatewayHandler Gateway 处理器
type GatewayHandler struct {
	gatewayService service.GatewayService
}

// NewGatewayHandler 创建 Gateway 处理器
func NewGatewayHandler(gatewayService service.GatewayService) *GatewayHandler {
	return &GatewayHandler{
		gatewayService: gatewayService,
	}
}

// ChatCompletions 聊天完成
// @Summary 聊天完成
// @Description 创建聊天完成请求，兼容 OpenAI API
// @Tags Gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body service.ChatCompletionRequest true "请求参数"
// @Success 200 {object} service.ChatCompletionResponse
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/chat/completions [post]
func (h *GatewayHandler) ChatCompletions(c *gin.Context) {
	// 获取 Token 信息
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"未授权的请求",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	// 解析请求
	var req service.ChatCompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"无效的请求参数: "+err.Error(),
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 验证模型
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缺少必需参数: model",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 模型白名单校验（AllowedModels 为空表示不限制）
	if !token.IsModelAllowed(req.Model) {
		c.JSON(http.StatusForbidden, model.NewOpenAIError(
			"当前 API Key 无权访问该模型",
			model.OpenAIErrorTypePermission,
			nil,
		))
		return
	}

	// 验证消息
	if len(req.Messages) == 0 {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缺少必需参数: messages",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 处理流式请求
	if req.Stream {
		h.handleStreamRequest(c, &req, token)
		return
	}

	// 处理普通请求
	resp, err := h.gatewayService.HandleChatCompletion(&req, token)
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// handleStreamRequest 处理流式请求
func (h *GatewayHandler) handleStreamRequest(c *gin.Context, req *service.ChatCompletionRequest, token *model.Token) {
	// 设置响应头
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("Transfer-Encoding", "chunked")

	// 处理流式请求
	err := h.gatewayService.HandleChatCompletionStream(req, token, c.Writer)
	if err != nil {
		// 如果还没有开始写入响应，返回错误
		if !c.Writer.Written() {
			WriteOpenAIError(c, err)
		}
		return
	}
}

// Completions 文本完成
// @Summary 文本完成
// @Description 创建文本完成请求，兼容 OpenAI API
// @Tags Gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body service.CompletionRequest true "请求参数"
// @Success 200 {object} service.CompletionResponse
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/completions [post]
func (h *GatewayHandler) Completions(c *gin.Context) {
	// 获取 Token 信息
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"未授权的请求",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	// 解析请求
	var req service.CompletionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"无效的请求参数: "+err.Error(),
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 验证模型
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缺少必需参数: model",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 模型白名单校验（AllowedModels 为空表示不限制）
	if !token.IsModelAllowed(req.Model) {
		c.JSON(http.StatusForbidden, model.NewOpenAIError(
			"当前 API Key 无权访问该模型",
			model.OpenAIErrorTypePermission,
			nil,
		))
		return
	}

	// 处理请求
	resp, err := h.gatewayService.HandleCompletion(&req, token)
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// Embeddings 嵌入
// @Summary 嵌入
// @Description 创建嵌入请求，兼容 OpenAI API
// @Tags Gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body service.EmbeddingRequest true "请求参数"
// @Success 200 {object} service.EmbeddingResponse
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/embeddings [post]
func (h *GatewayHandler) Embeddings(c *gin.Context) {
	// 获取 Token 信息
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"未授权的请求",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	// 解析请求
	var req service.EmbeddingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"无效的请求参数: "+err.Error(),
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 验证模型
	if req.Model == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缺少必需参数: model",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 模型白名单校验（AllowedModels 为空表示不限制）
	if !token.IsModelAllowed(req.Model) {
		c.JSON(http.StatusForbidden, model.NewOpenAIError(
			"当前 API Key 无权访问该模型",
			model.OpenAIErrorTypePermission,
			nil,
		))
		return
	}

	// 处理请求
	resp, err := h.gatewayService.HandleEmbedding(&req, token)
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// ListModels 列出模型
// @Summary 列出模型
// @Description 获取可用模型列表，兼容 OpenAI API
// @Tags Gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Success 200 {object} service.ModelsResponse
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/models [get]
func (h *GatewayHandler) ListModels(c *gin.Context) {
	resp, err := h.gatewayService.ListModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewOpenAIError(
			err.Error(),
			model.OpenAIErrorTypeServer,
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetModel 获取模型详情
// @Summary 获取模型详情
// @Description 获取指定模型的详情，兼容 OpenAI API
// @Tags Gateway
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param model path string true "模型 ID"
// @Success 200 {object} service.ModelData
// @Failure 401 {object} model.OpenAIError
// @Failure 404 {object} model.OpenAIError
// @Router /v1/models/{model} [get]
func (h *GatewayHandler) GetModel(c *gin.Context) {
	modelID := c.Param("model")

	resp, err := h.gatewayService.ListModels()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewOpenAIError(
			err.Error(),
			model.OpenAIErrorTypeServer,
			nil,
		))
		return
	}

	// 查找指定模型
	for _, m := range resp.Data {
		if m.ID == modelID {
			c.JSON(http.StatusOK, m)
			return
		}
	}

	c.JSON(http.StatusNotFound, model.NewOpenAIError(
		"模型不存在: "+modelID,
		model.OpenAIErrorTypeNotFound,
		nil,
	))
}
