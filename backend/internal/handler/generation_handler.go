package handler

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GenerationHandler 生成相关 API 处理器
type GenerationHandler struct {
	generationService service.GenerationService
}

// NewGenerationHandler 创建生成处理器
func NewGenerationHandler(generationService service.GenerationService) *GenerationHandler {
	return &GenerationHandler{generationService: generationService}
}

// ImageGenerations 鍥剧墖鐢熸垚
// @Summary 鍥剧墖鐢熸垚
// @Description 鍒涘缓鍥剧墖鐢熸垚璇锋眰锛屽吋瀹?OpenAI 椋庢牸鎺ュ彛
// @Tags Generation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body model.ImageGenerationRequest true "璇锋眰鍙傛暟"
// @Success 200 {object} model.ImageGenerationResponse
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/images/generations [post]
func (h *GenerationHandler) ImageGenerations(c *gin.Context) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"鏈巿鏉冪殑璇锋眰",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	var req model.ImageGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"鏃犳晥鐨勮姹傚弬鏁? "+err.Error(),
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缂哄皯蹇呴渶鍙傛暟: model",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 妯″瀷鐧藉悕鍗曟牎楠岋紙AllowedModels 涓虹┖琛ㄧず涓嶉檺鍒讹級
	if !token.IsModelAllowed(req.Model) {
		c.JSON(http.StatusForbidden, model.NewOpenAIError(
			"当前 API Key 无权访问该模型",
			model.OpenAIErrorTypePermission,
			nil,
		))
		return
	}

	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缂哄皯蹇呴渶鍙傛暟: prompt",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	if req.N <= 0 {
		req.N = 1
	}
	// 当前网关图片生成仅支持单张输出，避免 n>1 导致计费与返回不一致。
	if req.N != 1 {
		req.N = 1
	}
	if req.AspectRatio == "" {
		req.AspectRatio = "1:1"
	}
	if req.Resolution == "" {
		req.Resolution = "1K"
	}
	// 目前仅返回 url，暂不支持 b64_json。
	req.ResponseFormat = "url"

	idempotencyKey := c.GetHeader("Idempotency-Key")

	prefer := strings.ToLower(strings.TrimSpace(c.GetHeader("Prefer")))
	asyncQuery := strings.ToLower(strings.TrimSpace(c.Query("async")))
	asyncRequested := strings.Contains(prefer, "respond-async") || asyncQuery == "1" || asyncQuery == "true"

	var (
		resp *model.ImageGenerationResponse
		err  error
	)
	if asyncRequested {
		resp, err = h.generationService.CreateImageTask(c.Request.Context(), &req, token, idempotencyKey)
	} else {
		resp, err = h.generationService.GenerateImage(c.Request.Context(), &req, token, idempotencyKey)
	}
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// VideoGenerations 瑙嗛鐢熸垚
// @Summary 瑙嗛鐢熸垚
// @Description 鍒涘缓瑙嗛鐢熸垚璇锋眰
// @Tags Generation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param request body model.VideoGenerationRequest true "璇锋眰鍙傛暟"
// @Success 200 {object} model.VideoGenerationResponse
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/videos/generations [post]
func (h *GenerationHandler) VideoGenerations(c *gin.Context) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"鏈巿鏉冪殑璇锋眰",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	var req model.VideoGenerationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"鏃犳晥鐨勮姹傚弬鏁? "+err.Error(),
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	if req.Model == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缂哄皯蹇呴渶鍙傛暟: model",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 妯″瀷鐧藉悕鍗曟牎楠岋紙AllowedModels 涓虹┖琛ㄧず涓嶉檺鍒讹級
	if !token.IsModelAllowed(req.Model) {
		c.JSON(http.StatusForbidden, model.NewOpenAIError(
			"当前 API Key 无权访问该模型",
			model.OpenAIErrorTypePermission,
			nil,
		))
		return
	}

	if req.Prompt == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缂哄皯蹇呴渶鍙傛暟: prompt",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	if req.AspectRatio == "" {
		req.AspectRatio = "9:16"
	}
	if req.Duration <= 0 {
		req.Duration = 10
	}
	if req.Size == "" {
		req.Size = "small"
	}

	idempotencyKey := c.GetHeader("Idempotency-Key")
	resp, err := h.generationService.GenerateVideo(c.Request.Context(), &req, token, idempotencyKey)
	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// GetTaskStatus 鑾峰彇浠诲姟鐘舵€?// @Summary 鑾峰彇浠诲姟鐘舵€?// @Description 鏌ヨ鍥剧墖/瑙嗛鐢熸垚浠诲姟鐘舵€?// @Tags Generation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param id path string true "浠诲姟ID"
// @Success 200 {object} model.GenerationTaskResponse
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 403 {object} model.OpenAIError
// @Failure 404 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/generations/{id} [get]
func (h *GenerationHandler) GetTaskStatus(c *gin.Context) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"鏈巿鏉冪殑璇锋眰",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	taskID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"鏃犳晥鐨勪换鍔D",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	resp, err := h.generationService.GetTaskStatus(c.Request.Context(), taskID, token.UserID)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrGenerationTaskNotFound):
			c.JSON(http.StatusNotFound, model.NewOpenAIError(
				err.Error(),
				model.OpenAIErrorTypeNotFound,
				nil,
			))
			return
		case errors.Is(err, service.ErrGenerationTaskForbidden):
			c.JSON(http.StatusForbidden, model.NewOpenAIError(
				err.Error(),
				model.OpenAIErrorTypePermission,
				nil,
			))
			return
		default:
			c.JSON(http.StatusInternalServerError, model.NewOpenAIError(
				err.Error(),
				model.OpenAIErrorTypeServer,
				nil,
			))
			return
		}
	}

	c.JSON(http.StatusOK, resp)
}

// ListUserTasks 鑾峰彇鐢ㄦ埛浠诲姟鍒楄〃
// @Summary 鑾峰彇鐢ㄦ埛浠诲姟鍒楄〃
// @Description 鍒嗛〉鏌ヨ鐢ㄦ埛鍥剧墖/瑙嗛鐢熸垚浠诲姟
// @Tags Generation
// @Accept json
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param type query string false "浠诲姟绫诲瀷(image/video)"
// @Param page query int false "椤电爜" default(1)
// @Param page_size query int false "姣忛〉鏁伴噺" default(20)
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/generations [get]
func (h *GenerationHandler) ListUserTasks(c *gin.Context) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"鏈巿鏉冪殑璇锋眰",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	taskType := model.GenerationType(c.Query("type"))
	if taskType != "" && taskType != model.GenerationTypeImage && taskType != model.GenerationTypeVideo {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"无效的任务类型",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	page := parsePositiveInt(c.Query("page"), 1)
	pageSize := parsePositiveInt(c.Query("page_size"), 20)
	if pageSize > 100 {
		pageSize = 100
	}

	tasks, total, err := h.generationService.ListUserTasks(c.Request.Context(), token.UserID, taskType, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewOpenAIError(
			err.Error(),
			model.OpenAIErrorTypeServer,
			nil,
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":      tasks,
		"total":     total,
		"page":      page,
		"page_size": pageSize,
	})
}

func parsePositiveInt(s string, defaultValue int) int {
	if s == "" {
		return defaultValue
	}
	v, err := strconv.Atoi(s)
	if err != nil || v <= 0 {
		return defaultValue
	}
	return v
}
