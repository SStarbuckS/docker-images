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
	allowedExtensions := []string{".jpeg", ".jpg", ".png", ".gif", ".ico", ".webp"}
	ext := strings.ToLower(filepath.Ext(filename))

	for _, allowedExt := range allowedExtensions {
		if ext == allowedExt {
			return true
		}
	}
	return false
}

/**
 * 原子保存上传文件
 */
func saveUploadedFile(targetFilePath string, source io.Reader) error {
	tempFile, err := os.CreateTemp(filepath.Dir(targetFilePath), "."+filepath.Base(targetFilePath)+".tmp-*")
	if err != nil {
		return err
	}
	tempFilePath := tempFile.Name()
	defer tempFile.Close()
	defer os.Remove(tempFilePath)

	if err := tempFile.Chmod(0644); err != nil {
		return err
	}
	if _, err := io.Copy(tempFile, source); err != nil {
		return err
	}
	if err := tempFile.Sync(); err != nil {
		return err
	}
	if err := tempFile.Close(); err != nil {
		return err
	}

	return os.Rename(tempFilePath, targetFilePath)
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
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
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
			Message: "只允许上传图片文件 (jpeg, jpg, png, gif, ico, webp)",
		})
		return
	}

	h.storage.LockMutation()
	defer h.storage.UnlockMutation()

	categoryId := c.PostForm("category")
	if categoryId == "" {
		categoryId = "common"
	}

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

	// 生成新的文件名
	timestamp := time.Now().UnixNano()
	randomStr := strings.ReplaceAll(uuid.New().String()[:8], "-", "")
	ext := filepath.Ext(header.Filename)
	filename := fmt.Sprintf("icon_%d_%s%s", timestamp, randomStr, ext)

	// 创建目标目录
	targetDir, err := resolveUploadDirectory(h.storage.GetUploadsPath(), uploadDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "上传目录配置不合法",
		})
		return
	}
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "创建上传目录失败",
		})
		return
	}

	// 保存文件
	targetFilePath, err := resolvePathWithinBase(targetDir, filename)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "文件路径不合法",
		})
		return
	}
	if err := saveUploadedFile(targetFilePath, file); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "文件保存失败",
		})
		return
	}

	fileUrl := fmt.Sprintf("/uploads/%s/%s", uploadDir, filename)

	log.Printf("图标上传成功: %s → %s (%dKB) - 分类: %s",
		header.Filename,
		fileUrl,
		header.Size/1024,
		categoryId)

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
	if c.Request.MultipartForm != nil {
		defer c.Request.MultipartForm.RemoveAll()
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
			Message: "只允许上传图片文件 (jpeg, jpg, png, gif, ico, webp)",
		})
		return
	}

	h.storage.LockMutation()
	defer h.storage.UnlockMutation()

	// 确保favicon目录存在
	faviconDir, err := resolveUploadDirectory(h.storage.GetUploadsPath(), "favicon")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "网站图标目录不合法",
		})
		return
	}
	if err := os.MkdirAll(faviconDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "创建网站图标目录失败",
		})
		return
	}

	// 保存为favicon.ico
	targetFilePath, err := resolvePathWithinBase(faviconDir, "favicon.ico")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "网站图标路径不合法",
		})
		return
	}
	if err := saveUploadedFile(targetFilePath, file); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "文件保存失败",
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
