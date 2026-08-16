package handlers

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"navdesk/models"
	"navdesk/storage"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// CategoriesHandler 分类处理器
type CategoriesHandler struct {
	storage *storage.Storage
}

// NewCategoriesHandler 创建分类处理器
func NewCategoriesHandler(storage *storage.Storage) *CategoriesHandler {
	return &CategoriesHandler{
		storage: storage,
	}
}

// GetCategories 获取所有分类
func (h *CategoriesHandler) GetCategories(c *gin.Context) {
	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	// 按排序值排序
	sort.Slice(categories, func(i, j int) bool {
		return categories[i].Sort < categories[j].Sort
	})

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    categories,
	})
}

// GetCategory 获取单个分类
func (h *CategoriesHandler) GetCategory(c *gin.Context) {
	id := c.Param("id")

	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	for _, category := range categories {
		if category.ID == id {
			c.JSON(http.StatusOK, models.APIResponse{
				Success: true,
				Data:    category,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, models.APIResponse{
		Success: false,
		Message: "分类不存在",
	})
}

// CreateCategory 新增分类
func (h *CategoriesHandler) CreateCategory(c *gin.Context) {
	var req models.CreateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "分类名称、图标和上传目录不能为空",
		})
		return
	}
	if !isValidUploadDir(req.UploadDir) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "上传目录只能包含字母、数字、下划线和连字符",
		})
		return
	}
	h.storage.LockMutation()
	defer h.storage.UnlockMutation()

	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	// 检查名称是否重复
	for _, category := range categories {
		if category.Name == req.Name {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "分类名称已存在",
			})
			return
		}
	}

	// 检查上传目录是否重复
	for _, category := range categories {
		if category.UploadDir == req.UploadDir {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "上传目录已存在",
			})
			return
		}
	}

	// 创建上传目录
	uploadPath, err := resolveUploadDirectory(h.storage.GetUploadsPath(), req.UploadDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "上传目录不合法",
		})
		return
	}
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "创建上传目录失败",
		})
		return
	}

	// 生成唯一ID
	id := "cat_" + fmt.Sprintf("%d", time.Now().UnixNano()) + "_" + strings.ReplaceAll(uuid.New().String()[:8], "-", "")

	newCategory := models.Category{
		ID:        id,
		Name:      req.Name,
		Icon:      req.Icon,
		UploadDir: req.UploadDir,
		Sort:      req.Sort,
		CreatedAt: time.Now(),
	}

	if newCategory.Sort == 0 {
		newCategory.Sort = len(categories)
	}

	categories = append(categories, newCategory)

	if err := h.storage.SaveCategories(categories); err != nil {
		log.Printf("分类创建失败: %s - 保存数据失败", req.Name)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "保存分类失败",
		})
		return
	}

	session := sessions.Default(c)
	username := session.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}

	log.Printf("分类创建成功: %s (图标目录: %s) - 用户: %s", newCategory.Name, req.UploadDir, usernameStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "分类创建成功",
		Data:    newCategory,
	})
}

// UpdateCategory 更新分类
func (h *CategoriesHandler) UpdateCategory(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateCategoryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "分类名称、图标和上传目录不能为空",
		})
		return
	}
	if !isValidUploadDir(req.UploadDir) {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "上传目录只能包含字母、数字、下划线和连字符",
		})
		return
	}
	h.storage.LockMutation()
	defer h.storage.UnlockMutation()

	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	// 查找要更新的分类
	categoryIndex := -1
	for i, category := range categories {
		if category.ID == id {
			categoryIndex = i
			break
		}
	}

	if categoryIndex == -1 {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "分类不存在",
		})
		return
	}

	// 检查名称是否与其他分类重复
	for i, category := range categories {
		if i != categoryIndex && category.Name == req.Name {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "分类名称已存在",
			})
			return
		}
	}

	// 检查上传目录是否与其他分类重复
	for i, category := range categories {
		if i != categoryIndex && category.UploadDir == req.UploadDir {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "上传目录已存在",
			})
			return
		}
	}

	// 创建新的上传目录（如果不存在）
	uploadPath, err := resolveUploadDirectory(h.storage.GetUploadsPath(), req.UploadDir)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "上传目录不合法",
		})
		return
	}
	if err := os.MkdirAll(uploadPath, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "创建上传目录失败",
		})
		return
	}

	// 更新分类信息
	oldIcon := categories[categoryIndex].Icon
	categories[categoryIndex].Name = req.Name
	categories[categoryIndex].Icon = req.Icon
	categories[categoryIndex].UploadDir = req.UploadDir
	categories[categoryIndex].Sort = req.Sort
	categories[categoryIndex].UpdatedAt = time.Now()

	if err := h.storage.SaveCategories(categories); err != nil {
		log.Printf("分类更新失败: %s - 保存数据失败", req.Name)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "保存分类失败",
		})
		return
	}

	// 数据保存成功后清理旧图标
	if oldIcon != req.Icon && strings.HasPrefix(oldIcon, "/uploads/") {
		oldIconPath, pathErr := resolveUploadedFilePath(h.storage.GetUploadsPath(), oldIcon)
		if pathErr != nil {
			log.Printf("跳过不合法的旧分类图标路径: %s", oldIcon)
		} else if err := os.Remove(oldIconPath); err == nil {
			log.Printf("旧分类图标文件删除成功: %s", oldIconPath)
		} else if !os.IsNotExist(err) {
			log.Printf("旧分类图标文件删除失败: %v", err)
		}
	}

	session := sessions.Default(c)
	username := session.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}

	log.Printf("分类更新成功: %s (图标目录: %s) - 用户: %s", categories[categoryIndex].Name, req.UploadDir, usernameStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "分类更新成功",
		Data:    categories[categoryIndex],
	})
}

// DeleteCategory 删除分类
func (h *CategoriesHandler) DeleteCategory(c *gin.Context) {
	id := c.Param("id")

	// 不允许删除"全部"分类
	if id == "all" {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "不能删除\"全部\"分类",
		})
		return
	}

	h.storage.LockMutation()
	defer h.storage.UnlockMutation()

	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	// 查找要删除的分类
	categoryIndex := -1
	var categoryToDelete models.Category
	for i, category := range categories {
		if category.ID == id {
			categoryIndex = i
			categoryToDelete = category
			break
		}
	}

	if categoryIndex == -1 {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "分类不存在",
		})
		return
	}

	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取书签失败",
		})
		return
	}

	var categoryBookmarks []models.Bookmark
	var remainingBookmarks []models.Bookmark
	for _, bookmark := range bookmarks {
		if bookmark.Category == id {
			categoryBookmarks = append(categoryBookmarks, bookmark)
		} else {
			remainingBookmarks = append(remainingBookmarks, bookmark)
		}
	}

	// 删除分类记录
	newCategories := make([]models.Category, 0, len(categories)-1)
	for i, category := range categories {
		if i != categoryIndex {
			newCategories = append(newCategories, category)
		}
	}

	if len(categoryBookmarks) > 0 {
		log.Printf("准备删除分类 %s 及其下的 %d 个书签", id, len(categoryBookmarks))
		if err := h.storage.SaveBookmarks(remainingBookmarks); err != nil {
			c.JSON(http.StatusInternalServerError, models.APIResponse{
				Success: false,
				Message: "删除分类下的书签失败",
			})
			return
		}
	}

	if err := h.storage.SaveCategories(newCategories); err != nil {
		if len(categoryBookmarks) > 0 {
			if rollbackErr := h.storage.SaveBookmarks(bookmarks); rollbackErr != nil {
				log.Printf("分类删除失败且书签数据回滚失败: %v", rollbackErr)
			}
		}
		log.Printf("分类删除失败: %s - 保存数据失败", categoryToDelete.Name)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "删除分类失败",
		})
		return
	}

	// 数据保存成功后清理书签图标
	for _, bookmark := range categoryBookmarks {
		if bookmark.Icon == "" || !strings.HasPrefix(bookmark.Icon, "/uploads/") {
			continue
		}

		iconPath, pathErr := resolveUploadedFilePath(h.storage.GetUploadsPath(), bookmark.Icon)
		if pathErr != nil {
			log.Printf("跳过不合法的书签图标路径: %s", bookmark.Icon)
		} else if err := os.Remove(iconPath); err == nil {
			log.Printf("书签图标文件删除成功: %s", iconPath)
		} else if !os.IsNotExist(err) {
			log.Printf("书签图标文件删除失败: %v", err)
		}
	}

	// 数据保存成功后清理分类上传目录
	if categoryToDelete.UploadDir != "" {
		uploadDirPath, pathErr := resolveUploadDirectory(h.storage.GetUploadsPath(), categoryToDelete.UploadDir)
		if pathErr != nil {
			log.Printf("跳过不合法的分类上传目录: %s", categoryToDelete.UploadDir)
		} else if err := os.RemoveAll(uploadDirPath); err != nil {
			log.Printf("分类上传目录删除失败: %v", err)
		} else {
			log.Printf("分类上传目录删除成功: %s", uploadDirPath)
		}
	}

	session := sessions.Default(c)
	username := session.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}

	log.Printf("分类删除成功: %s - 用户: %s", categoryToDelete.Name, usernameStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "分类删除成功",
	})
}
