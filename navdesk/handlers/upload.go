package handlers

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"navdesk/models"
	"navdesk/storage"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// UploadHandler 上传处理器
type UploadHandler struct {
	storage *storage.Storage
}

// NewUploadHandler 创建上传处理器
func NewUploadHandler(storage *storage.Storage) *UploadHandler {
	return &UploadHandler{
		storage: storage,
	}
}

// 检查文件类型
func (h *UploadHandler) isValidImageFile(filename string) bool {
	allowedExtensions := []string{".jpeg", ".jpg", ".png", ".gif", ".svg", ".ico", ".webp"}
	ext := strings.ToLower(filepath.Ext(filename))

	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

// UploadIcon 图标上传接口
func (h *UploadHandler) UploadIcon(c *gin.Context) {
	// 解析multipart表单
	err := c.Request.ParseMultipartForm(2 << 20) // 2MB
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "解析表单失败",
		})
		return
	}

	file, header, err := c.Request.FormFile("icon")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "请选择要上传的文件",
		})
		return
	}
	defer file.Close()

	// 检查文件大小
	if header.Size > 2*1024*1024 { // 2MB
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "文件大小不能超过 2MB",
		})
		return
	}

	// 检查文件类型
	if !h.isValidImageFile(header.Filename) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "只允许上传图片文件 (jpeg, jpg, png, gif, svg, ico, webp)",
		})
		return
	}

	categoryId := c.PostForm("category")
	if categoryId == "" {
		categoryId = "common"
	}
	oldIcon := c.PostForm("oldIcon")

	// 获取分类信息
	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	uploadDir := "common"
	for _, category := range categories {
		if category.ID == categoryId {
			uploadDir = category.UploadDir
			break
		}
	}

	// 如果有旧图标，先删除旧图标文件
	if oldIcon != "" && strings.HasPrefix(oldIcon, "/uploads/") {
		oldIconPath := filepath.Join(h.storage.GetDataPath(), strings.TrimPrefix(oldIcon, "/"))
		if _, err := os.Stat(oldIconPath); err == nil {
			os.Remove(oldIconPath)
			log.Printf("旧图标文件删除成功: %s", oldIcon)
		}
	}

	// 生成新的文件名
	timestamp := time.Now().UnixNano()
	randomStr := strings.ReplaceAll(uuid.New().String()[:8], "-", "")
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("icon_%d_%s%s", timestamp, randomStr, ext)

	// 创建目标目录
	targetDir := filepath.Join(h.storage.GetUploadsPath(), uploadDir)
	if _, err := os.Stat(targetDir); os.IsNotExist(err) {
		os.MkdirAll(targetDir, 0755)
	}

	// 保存文件
	targetFilePath := filepath.Join(targetDir, filename)
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "文件保存失败",
		})
		return
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, file)
	if err != nil {
		os.Remove(targetFilePath) // 清理失败的文件
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "文件处理失败",
		})
		return
	}

	fileUrl := fmt.Sprintf("/uploads/%s/%s", uploadDir, filename)

	log.Printf("图标上传成功: %s → %s (%dKB) - 分类: %s%s",
		header.Filename,
		fileUrl,
		header.Size/1024,
		categoryId,
		func() string {
			if oldIcon != "" {
				return " | 已删除旧图标"
			}
			return ""
		}())

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "文件上传成功",
		"url":     fileUrl,
		"data": models.UploadResponse{
			URL:          fileUrl,
			Filename:     filename,
			OriginalName: header.Filename,
			Size:         header.Size,
		},
	})
}

// UploadFavicon 网站图标上传接口
func (h *UploadHandler) UploadFavicon(c *gin.Context) {
	// 解析multipart表单
	err := c.Request.ParseMultipartForm(2 << 20) // 2MB
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "解析表单失败",
		})
		return
	}

	file, header, err := c.Request.FormFile("favicon")
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "请选择要上传的文件",
		})
		return
	}
	defer file.Close()

	// 检查文件大小
	if header.Size > 2*1024*1024 { // 2MB
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "文件大小不能超过 2MB",
		})
		return
	}

	// 检查文件类型
	if !h.isValidImageFile(header.Filename) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "只允许上传图片文件 (jpeg, jpg, png, gif, svg, ico, webp)",
		})
		return
	}

	// 确保favicon目录存在
	faviconDir := filepath.Join(h.storage.GetUploadsPath(), "favicon")
	if _, err := os.Stat(faviconDir); os.IsNotExist(err) {
		os.MkdirAll(faviconDir, 0755)
	}

	// 保存为favicon.ico
	targetFilePath := filepath.Join(faviconDir, "favicon.ico")
	targetFile, err := os.Create(targetFilePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "文件保存失败",
		})
		return
	}
	defer targetFile.Close()

	_, err = io.Copy(targetFile, file)
	if err != nil {
		os.Remove(targetFilePath) // 清理失败的文件
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "文件处理失败",
		})
		return
	}

	log.Printf("网站图标更新成功: %s (%dKB)",
		header.Filename,
		header.Size/1024)

	c.JSON(http.StatusOK, map[string]interface{}{
		"success": true,
		"message": "网站图标更新成功",
		"url":     "/favicon.ico",
		"data": models.UploadResponse{
			URL:          "/favicon.ico",
			Filename:     "favicon.ico",
			OriginalName: header.Filename,
			Size:         header.Size,
		},
	})
}
