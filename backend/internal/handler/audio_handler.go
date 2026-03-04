package handler

import (
	"io"
	"net/http"
	"os"
	"path/filepath"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
)

// AudioHandler 音频能力 API 处理器
type AudioHandler struct {
	audioService service.AudioService
}

func NewAudioHandler(audioService service.AudioService) *AudioHandler {
	return &AudioHandler{audioService: audioService}
}

// Transcriptions 音频转写
// @Summary 音频转写
// @Description 创建音频转写请求，兼容 OpenAI 风格接口（multipart/form-data）
// @Tags Audio
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param file formData file true "音频文件"
// @Param model formData string true "模型 ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/audio/transcriptions [post]
func (h *AudioHandler) Transcriptions(c *gin.Context) {
	h.handleMultipartAudio(c, model.OperationAudioTranscriptions)
}

// Translations 音频翻译
// @Summary 音频翻译
// @Description 创建音频翻译请求，兼容 OpenAI 风格接口（multipart/form-data）
// @Tags Audio
// @Accept multipart/form-data
// @Produce json
// @Param Authorization header string true "Bearer Token"
// @Param file formData file true "音频文件"
// @Param model formData string true "模型 ID"
// @Success 200 {object} map[string]interface{}
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/audio/translations [post]
func (h *AudioHandler) Translations(c *gin.Context) {
	h.handleMultipartAudio(c, model.OperationAudioTranslations)
}

func (h *AudioHandler) handleMultipartAudio(c *gin.Context, operation string) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"未授权的请求",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缺少必需参数: file",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	modelID := c.PostForm("model")
	if modelID == "" {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"缺少必需参数: model",
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

	// 模型白名单校验（AllowedModels 为空表示不限制）
	if !token.IsModelAllowed(modelID) {
		c.JSON(http.StatusForbidden, model.NewOpenAIError(
			"当前 API Key 无权访问该模型",
			model.OpenAIErrorTypePermission,
			nil,
		))
		return
	}

	form, _ := c.MultipartForm()
	fields := map[string][]string{}
	if form != nil {
		for k, vals := range form.Value {
			copied := make([]string, 0, len(vals))
			copied = append(copied, vals...)
			fields[k] = copied
		}
	}

	src, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewOpenAIError(
			"读取文件失败",
			model.OpenAIErrorTypeServer,
			nil,
		))
		return
	}
	defer src.Close()

	suffix := filepath.Ext(fileHeader.Filename)
	tmp, err := os.CreateTemp("", "nexus-audio-*"+suffix)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.NewOpenAIError(
			"创建临时文件失败",
			model.OpenAIErrorTypeServer,
			nil,
		))
		return
	}
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmp.Name())
	}()

	if _, err := io.Copy(tmp, src); err != nil {
		c.JSON(http.StatusInternalServerError, model.NewOpenAIError(
			"保存文件失败",
			model.OpenAIErrorTypeServer,
			nil,
		))
		return
	}

	input := &service.AudioMultipartInput{
		Model:    modelID,
		FilePath: tmp.Name(),
		FileName: fileHeader.Filename,
		Fields:   fields,
	}

	var resp *service.AudioProxyResponse
	switch operation {
	case model.OperationAudioTranslations:
		resp, err = h.audioService.Translate(c.Request.Context(), input, token)
	default:
		resp, err = h.audioService.Transcribe(c.Request.Context(), input, token)
	}

	if err != nil {
		WriteOpenAIError(c, err)
		return
	}

	contentType := resp.ContentType
	if contentType == "" {
		contentType = "application/json"
	}
	c.Data(http.StatusOK, contentType, resp.Body)
}

// Speech 语音合成
// @Summary 语音合成
// @Description 创建语音合成请求，兼容 OpenAI 风格接口（JSON）
// @Tags Audio
// @Accept json
// @Produce application/octet-stream
// @Param Authorization header string true "Bearer Token"
// @Param request body model.AudioSpeechRequest true "请求参数"
// @Success 200 {string} string
// @Failure 400 {object} model.OpenAIError
// @Failure 401 {object} model.OpenAIError
// @Failure 500 {object} model.OpenAIError
// @Router /v1/audio/speech [post]
func (h *AudioHandler) Speech(c *gin.Context) {
	token, ok := middleware.GetCurrentToken(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.NewOpenAIError(
			"未授权的请求",
			model.OpenAIErrorTypeAuthentication,
			nil,
		))
		return
	}

	var req model.AudioSpeechRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.NewOpenAIError(
			"无效的请求参数: "+err.Error(),
			model.OpenAIErrorTypeInvalidRequest,
			nil,
		))
		return
	}

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

	err := h.audioService.Speech(c.Request.Context(), &req, token, c.Writer)
	if err != nil {
		if !c.Writer.Written() {
			WriteOpenAIError(c, err)
		}
		return
	}
}
