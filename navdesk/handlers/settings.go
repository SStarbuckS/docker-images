package handlers

import (
	"net/http"
	"strings"

	"navdesk/models"
	"navdesk/storage"

	"github.com/gin-gonic/gin"
)

// SettingsHandler 设置处理器
type SettingsHandler struct {
	storage *storage.Storage
}

// NewSettingsHandler 创建设置处理器
func NewSettingsHandler(storage *storage.Storage) *SettingsHandler {
	return &SettingsHandler{
		storage: storage,
	}
}

// GetSettings 获取设置
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	settings, err := h.storage.GetSettings()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取设置失败",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    settings,
	})
}

// UpdateSettings 更新设置
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var req models.UpdateSettingsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "参数不完整",
		})
		return
	}

	// 验证网站标题
	if strings.TrimSpace(req.SiteTitle) == "" || len(req.SiteTitle) > 50 {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "网站标题不能为空且长度不能超过50个字符",
		})
		return
	}

	// 验证数值范围
	if req.CardWidth < 120 || req.CardWidth > 300 ||
		req.CardHeight < 60 || req.CardHeight > 150 ||
		req.IconWidth < 24 || req.IconWidth > 80 ||
		req.IconHeight < 24 || req.IconHeight > 80 ||
		req.SidebarWidth < 50 || req.SidebarWidth > 600 {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "参数超出有效范围",
		})
		return
	}

	// 验证主题
	if req.Theme != "auto" && req.Theme != "light" && req.Theme != "dark" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "无效的主题设置",
		})
		return
	}

	settings := models.Settings{
		SiteTitle:    strings.TrimSpace(req.SiteTitle),
		CardWidth:    req.CardWidth,
		CardHeight:   req.CardHeight,
		IconWidth:    req.IconWidth,
		IconHeight:   req.IconHeight,
		SidebarWidth: req.SidebarWidth,
		Theme:        req.Theme,
	}

	if err := h.storage.SaveSettings(settings); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "保存设置失败",
		})
		return
	}

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "设置保存成功",
		Data:    settings,
	})
}
