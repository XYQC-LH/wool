package handler

import (
	"net/http"

	"nexus-api/internal/model"

	"github.com/gin-gonic/gin"
)

type BatchModelStatusRequest struct {
	IDs     []string `json:"ids" binding:"required,min=1"`
	Enabled *bool    `json:"enabled" binding:"required"`
}

type BatchModelDeleteRequest struct {
	IDs []string `json:"ids" binding:"required,min=1"`
}

func (h *AdminHandler) BatchUpdateModelStatus(c *gin.Context) {
	var req BatchModelStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}
	if req.Enabled == nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "enabled 不能为空"))
		return
	}

	if err := h.modelService.BatchUpdateStatus(req.IDs, *req.Enabled); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"updated": len(req.IDs),
	}))
}

func (h *AdminHandler) BatchDeleteModels(c *gin.Context) {
	var req BatchModelDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, "无效的请求参数: "+err.Error()))
		return
	}

	if err := h.modelService.BatchDelete(req.IDs); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(model.ErrCodeInvalidRequest, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"deleted": len(req.IDs),
	}))
}

func (h *AdminHandler) GetModelStats(c *gin.Context) {
	stats, err := h.modelService.GetAdminStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(model.ErrCodeInternalError, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(stats))
}
