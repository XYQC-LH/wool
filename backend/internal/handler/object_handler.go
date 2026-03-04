package handler

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"nexus-api/internal/storage"

	"github.com/gin-gonic/gin"
)

// ObjectHandler 本地对象存储访问（签名 URL）
type ObjectHandler struct {
	localDir   string
	signSecret []byte
}

func NewObjectHandler(localDir string, signSecret string) *ObjectHandler {
	return &ObjectHandler{
		localDir:   strings.TrimSpace(localDir),
		signSecret: []byte(strings.TrimSpace(signSecret)),
	}
}

// GetObject 访问本地对象存储（需要签名参数 exp/sig）
// @Summary 访问对象存储资源（本地模式）
// @Tags Assets
// @Param key path string true "Object Key"
// @Param exp query int true "过期时间（Unix 秒）"
// @Param sig query string true "签名"
// @Success 200
// @Router /objects/{key} [get]
func (h *ObjectHandler) GetObject(c *gin.Context) {
	if h.localDir == "" || len(h.signSecret) == 0 {
		c.Status(http.StatusNotFound)
		return
	}

	objectKey := strings.TrimPrefix(c.Param("key"), "/")
	if strings.TrimSpace(objectKey) == "" {
		c.Status(http.StatusNotFound)
		return
	}

	expStr := c.Query("exp")
	sig := c.Query("sig")
	if expStr == "" || sig == "" {
		c.Status(http.StatusForbidden)
		return
	}

	exp, err := strconv.ParseInt(expStr, 10, 64)
	if err != nil || exp <= 0 {
		c.Status(http.StatusForbidden)
		return
	}
	if time.Now().Unix() > exp {
		c.Status(http.StatusForbidden)
		return
	}

	if !storage.VerifyLocalObjectSignature(objectKey, exp, sig, h.signSecret) {
		c.Status(http.StatusForbidden)
		return
	}

	fullPath, err := storage.ResolveLocalObjectPath(h.localDir, objectKey)
	if err != nil {
		c.Status(http.StatusNotFound)
		return
	}

	c.File(fullPath)
}
