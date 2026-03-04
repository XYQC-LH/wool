package handler

import (
	"net/http"
	"strings"

	"nexus-api/internal/middleware"
	"nexus-api/internal/model"
	"nexus-api/internal/service"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// AssetHandler 资源处理器
type AssetHandler struct {
	assetService service.AssetService
}

func NewAssetHandler(assetService service.AssetService) *AssetHandler {
	return &AssetHandler{assetService: assetService}
}

type assetUploadResponse struct {
	ID        uuid.UUID          `json:"id"`
	Purpose   model.AssetPurpose `json:"purpose"`
	Kind      model.AssetKind    `json:"kind"`
	MimeType  string             `json:"mime_type"`
	SizeBytes int64              `json:"size_bytes"`
	URL       string             `json:"url"`
}

// UploadSiteMaterial 上传网站素材（管理员）
// @Summary 上传网站素材
// @Tags Assets
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "文件"
// @Success 200 {object} model.Response{data=assetUploadResponse}
// @Router /api/admin/assets/site [post]
func (h *AssetHandler) UploadSiteMaterial(c *gin.Context) {
	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "缺少文件字段: file"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "打开文件失败"))
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	asset, err := h.assetService.UploadSiteMaterial(service.UploadInput{
		Reader:           file,
		OriginalFilename: fileHeader.Filename,
		ContentType:      contentType,
		SizeBytes:        fileHeader.Size,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	url, err := h.assetService.SignGetURL(asset, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	url = absoluteURLFromRequest(c, url)

	c.JSON(http.StatusOK, model.SuccessResponse(&assetUploadResponse{
		ID:        asset.ID,
		Purpose:   asset.Purpose,
		Kind:      asset.Kind,
		MimeType:  asset.MimeType,
		SizeBytes: asset.SizeBytes,
		URL:       url,
	}))
}

// UploadUserUpload 用户上传文件（用户端测试上传）
// @Summary 用户上传文件
// @Tags Assets
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "文件"
// @Success 200 {object} model.Response{data=assetUploadResponse}
// @Router /api/user/assets/uploads [post]
func (h *AssetHandler) UploadUserUpload(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "缺少文件字段: file"))
		return
	}

	file, err := fileHeader.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "打开文件失败"))
		return
	}
	defer file.Close()

	contentType := fileHeader.Header.Get("Content-Type")
	asset, err := h.assetService.UploadUserUpload(userID, service.UploadInput{
		Reader:           file,
		OriginalFilename: fileHeader.Filename,
		ContentType:      contentType,
		SizeBytes:        fileHeader.Size,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	url, err := h.assetService.SignGetURL(asset, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	url = absoluteURLFromRequest(c, url)

	c.JSON(http.StatusOK, model.SuccessResponse(&assetUploadResponse{
		ID:        asset.ID,
		Purpose:   asset.Purpose,
		Kind:      asset.Kind,
		MimeType:  asset.MimeType,
		SizeBytes: asset.SizeBytes,
		URL:       url,
	}))
}

// GetUserAssetURL 获取用户私有资源的签名URL
// @Summary 获取签名URL
// @Tags Assets
// @Produce json
// @Security BearerAuth
// @Param id path string true "Asset ID"
// @Success 200 {object} model.Response{data=map[string]string}
// @Router /api/user/assets/{id}/url [get]
func (h *AssetHandler) GetUserAssetURL(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(model.ErrCodeUnauthorized, "未授权"))
		return
	}

	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的资源ID"))
		return
	}

	asset, err := h.assetService.GetByID(assetID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	if asset == nil || asset.UserID == nil || *asset.UserID != userID {
		c.JSON(http.StatusNotFound, model.ErrorResponse(model.ErrCodeNotFound, "资源不存在"))
		return
	}

	url, err := h.assetService.SignGetURL(asset, 0)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}
	url = absoluteURLFromRequest(c, url)

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{"url": url}))
}

func absoluteURLFromRequest(c *gin.Context, rawURL string) string {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return rawURL
	}
	if strings.HasPrefix(rawURL, "http://") || strings.HasPrefix(rawURL, "https://") {
		return rawURL
	}

	if !strings.HasPrefix(rawURL, "/") {
		rawURL = "/" + rawURL
	}

	scheme := firstForwardedValue(c.GetHeader("X-Forwarded-Proto"))
	if scheme == "" {
		if c.Request.TLS != nil {
			scheme = "https"
		} else {
			scheme = "http"
		}
	}

	host := firstForwardedValue(c.GetHeader("X-Forwarded-Host"))
	if host == "" {
		host = c.Request.Host
	}
	if host == "" {
		return rawURL
	}

	return scheme + "://" + host + rawURL
}

func firstForwardedValue(headerValue string) string {
	headerValue = strings.TrimSpace(headerValue)
	if headerValue == "" {
		return ""
	}
	if idx := strings.IndexByte(headerValue, ','); idx >= 0 {
		headerValue = headerValue[:idx]
	}
	return strings.TrimSpace(headerValue)
}

// RedirectUserAsset 访问用户私有资源（302 跳转到 OSS 签名URL）
// @Summary 访问用户私有资源
// @Tags Assets
// @Security BearerAuth
// @Param id path string true "Asset ID"
// @Success 302
// @Router /api/user/assets/{id} [get]
func (h *AssetHandler) RedirectUserAsset(c *gin.Context) {
	userID, ok := middleware.GetCurrentUserID(c)
	if !ok {
		c.Status(http.StatusUnauthorized)
		return
	}

	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	asset, err := h.assetService.GetByID(assetID)
	if err != nil || asset == nil || asset.UserID == nil || *asset.UserID != userID {
		c.Status(http.StatusNotFound)
		return
	}

	url, err := h.assetService.SignGetURL(asset, 0)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Redirect(http.StatusFound, url)
}

// RedirectSiteAsset 公开访问网站素材（302 跳转到 OSS 签名URL）
// @Summary 访问网站素材
// @Tags Assets
// @Param id path string true "Asset ID"
// @Success 302
// @Router /assets/{id} [get]
func (h *AssetHandler) RedirectSiteAsset(c *gin.Context) {
	assetID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	asset, err := h.assetService.GetByID(assetID)
	if err != nil || asset == nil || !asset.IsPublic() {
		c.Status(http.StatusNotFound)
		return
	}

	url, err := h.assetService.SignGetURL(asset, 0)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.Redirect(http.StatusFound, url)
}
