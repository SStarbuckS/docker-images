package handlers

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"navdesk/models"
	"navdesk/storage"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BookmarksHandler 书签处理器
type BookmarksHandler struct {
	storage *storage.Storage
}

// NewBookmarksHandler 创建书签处理器
func NewBookmarksHandler(storage *storage.Storage) *BookmarksHandler {
	return &BookmarksHandler{
		storage: storage,
	}
}

// GetBookmarks 获取所有书签
func (h *BookmarksHandler) GetBookmarks(c *gin.Context) {
	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取书签失败",
		})
		return
	}

	// 按排序值排序，相同排序值按创建时间排序
	sort.Slice(bookmarks, func(i, j int) bool {
		if bookmarks[i].Sort == bookmarks[j].Sort {
			return bookmarks[i].CreatedAt.Before(bookmarks[j].CreatedAt)
		}
		return bookmarks[i].Sort < bookmarks[j].Sort
	})

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    bookmarks,
	})
}

// GetBookmarksByCategory 根据分类获取书签
func (h *BookmarksHandler) GetBookmarksByCategory(c *gin.Context) {
	categoryId := c.Param("categoryId")

	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取书签失败",
		})
		return
	}

	var filteredBookmarks []models.Bookmark
	if categoryId == "all" {
		filteredBookmarks = bookmarks
	} else {
		for _, bookmark := range bookmarks {
			if bookmark.Category == categoryId {
				filteredBookmarks = append(filteredBookmarks, bookmark)
			}
		}
	}

	// 按排序值排序，相同排序值按创建时间排序
	sort.Slice(filteredBookmarks, func(i, j int) bool {
		if filteredBookmarks[i].Sort == filteredBookmarks[j].Sort {
			return filteredBookmarks[i].CreatedAt.Before(filteredBookmarks[j].CreatedAt)
		}
		return filteredBookmarks[i].Sort < filteredBookmarks[j].Sort
	})

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    filteredBookmarks,
	})
}

// GetBookmark 获取单个书签
func (h *BookmarksHandler) GetBookmark(c *gin.Context) {
	id := c.Param("id")

	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取书签失败",
		})
		return
	}

	for _, bookmark := range bookmarks {
		if bookmark.ID == id {
			c.JSON(http.StatusOK, models.APIResponse{
				Success: true,
				Data:    bookmark,
			})
			return
		}
	}

	c.JSON(http.StatusNotFound, models.APIResponse{
		Success: false,
		Message: "书签不存在",
	})
}

// CreateBookmark 新增书签
func (h *BookmarksHandler) CreateBookmark(c *gin.Context) {
	var req models.CreateBookmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "书签名称、网址和分类不能为空",
		})
		return
	}

	// 验证URL格式
	if _, err := url.Parse(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "网址格式不正确",
		})
		return
	}

	// 验证分类是否存在
	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	if req.Category != "all" {
		categoryExists := false
		for _, category := range categories {
			if category.ID == req.Category {
				categoryExists = true
				break
			}
		}
		if !categoryExists {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "指定的分类不存在",
			})
			return
		}
	}

	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取书签失败",
		})
		return
	}

	// 检查同一分类下名称是否重复
	for _, bookmark := range bookmarks {
		if bookmark.Category == req.Category && bookmark.Name == req.Name {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "该分类下已存在相同名称的书签",
			})
			return
		}
	}

	// 生成唯一ID
	id := "bookmark_" + fmt.Sprintf("%d", time.Now().UnixNano()) + "_" + strings.ReplaceAll(uuid.New().String()[:8], "-", "")

	// 处理图标设置逻辑
	icon := req.Icon
	if icon == "" {
		// 用户留空：使用书签网址拼接 "/favicon.ico"
		parsedURL, err := url.Parse(req.URL)
		if err == nil && parsedURL.Host != "" {
			icon = parsedURL.Scheme + "://" + parsedURL.Host + "/favicon.ico"
		} else {
			icon = "/favicon.ico" // 如果URL解析失败，使用默认图标
		}
	} else if strings.ToLower(strings.TrimSpace(icon)) == "local" {
		// 用户输入 "local"：使用默认图标
		icon = "/favicon.ico"
	}
	// 其他情况：保持用户输入的图标URL或上传的本地图标路径

	sort := req.Sort
	if sort == 0 {
		// 计算该分类下的书签数量
		categoryBookmarksCount := 0
		for _, bookmark := range bookmarks {
			if bookmark.Category == req.Category {
				categoryBookmarksCount++
			}
		}
		sort = categoryBookmarksCount + 1
	}

	newBookmark := models.Bookmark{
		ID:          id,
		Name:        req.Name,
		URL:         req.URL,
		Description: req.Description,
		Icon:        icon,
		Category:    req.Category,
		Tags:        req.Tags,
		Sort:        sort,
		CreatedAt:   time.Now(),
	}

	if newBookmark.Tags == nil {
		newBookmark.Tags = []string{}
	}

	bookmarks = append(bookmarks, newBookmark)

	if err := h.storage.SaveBookmarks(bookmarks); err != nil {
		log.Printf("书签创建失败: %s - 保存数据失败", req.Name)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "保存书签失败",
		})
		return
	}

	session := sessions.Default(c)
	username := session.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}

	log.Printf("书签创建成功: %s (%s) - 用户: %s", newBookmark.Name, newBookmark.Category, usernameStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "书签创建成功",
		Data:    newBookmark,
	})
}

// 移动图标到新分类目录
func (h *BookmarksHandler) moveIconToNewCategory(oldIconPath, oldCategoryId, newCategoryId string, categories []models.Category) string {
	// 解析旧图标路径
	if !strings.HasPrefix(oldIconPath, "/uploads/") {
		log.Printf("图标路径格式不正确: %s", oldIconPath)
		return oldIconPath
	}

	parts := strings.Split(strings.TrimPrefix(oldIconPath, "/uploads/"), "/")
	if len(parts) < 2 {
		log.Printf("图标路径格式不正确: %s", oldIconPath)
		return oldIconPath
	}

	oldDir := parts[0]
	filename := parts[1]

	// 获取新分类的上传目录
	newUploadDir := "common"
	for _, category := range categories {
		if category.ID == newCategoryId {
			newUploadDir = category.UploadDir
			break
		}
	}

	// 构建文件路径
	oldFilePath := filepath.Join(h.storage.GetDataPath(), "uploads", oldDir, filename)
	newDirPath := filepath.Join(h.storage.GetUploadsPath(), newUploadDir)
	newFilePath := filepath.Join(newDirPath, filename)

	// 检查旧文件是否存在
	if _, err := os.Stat(oldFilePath); os.IsNotExist(err) {
		log.Printf("旧图标文件不存在: %s", oldFilePath)
		return fmt.Sprintf("/uploads/%s/%s", newUploadDir, filename)
	}

	// 创建新目录（如果不存在）
	if _, err := os.Stat(newDirPath); os.IsNotExist(err) {
		os.MkdirAll(newDirPath, 0755)
	}

	// 移动文件
	if err := os.Rename(oldFilePath, newFilePath); err != nil {
		log.Printf("移动图标文件失败: %v", err)
		return oldIconPath
	}

	log.Printf("图标文件移动成功: %s -> %s", oldFilePath, newFilePath)
	return fmt.Sprintf("/uploads/%s/%s", newUploadDir, filename)
}

// UpdateBookmark 更新书签
func (h *BookmarksHandler) UpdateBookmark(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateBookmarkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "书签名称、网址和分类不能为空",
		})
		return
	}

	// 验证URL格式
	if _, err := url.Parse(req.URL); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "网址格式不正确",
		})
		return
	}

	// 验证分类是否存在
	categories, err := h.storage.GetCategories()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取分类失败",
		})
		return
	}

	if req.Category != "all" {
		categoryExists := false
		for _, category := range categories {
			if category.ID == req.Category {
				categoryExists = true
				break
			}
		}
		if !categoryExists {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "指定的分类不存在",
			})
			return
		}
	}

	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取书签失败",
		})
		return
	}

	// 查找要更新的书签
	bookmarkIndex := -1
	for i, bookmark := range bookmarks {
		if bookmark.ID == id {
			bookmarkIndex = i
			break
		}
	}

	if bookmarkIndex == -1 {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "书签不存在",
		})
		return
	}

	// 检查同一分类下名称是否与其他书签重复
	for i, bookmark := range bookmarks {
		if i != bookmarkIndex && bookmark.Category == req.Category && bookmark.Name == req.Name {
			c.JSON(http.StatusBadRequest, models.APIResponse{
				Success: false,
				Message: "该分类下已存在相同名称的书签",
			})
			return
		}
	}

	oldBookmark := bookmarks[bookmarkIndex]
	oldCategory := oldBookmark.Category
	oldIcon := oldBookmark.Icon

	newIconPath := req.Icon

	// 处理图标变化的情况
	if oldIcon != req.Icon {
		// 图标发生了变化
		if oldIcon != "" && strings.HasPrefix(oldIcon, "/uploads/") {
			if req.Icon != "" && strings.HasPrefix(req.Icon, "/uploads/") {
				// 新图标也是本地文件，检查是否需要移动
				if oldCategory != req.Category {
					// 分类也变了，移动文件
					newIconPath = h.moveIconToNewCategory(oldIcon, oldCategory, req.Category, categories)
				} else {
					// 分类没变，但图标变了
					log.Printf("图标已更新，旧图标已在上传时删除: %s -> %s", oldIcon, req.Icon)
				}
			} else {
				// 新图标不是本地文件（可能是网络图标），删除旧的本地图标
				oldIconPath := filepath.Join(h.storage.GetDataPath(), strings.TrimPrefix(oldIcon, "/"))
				log.Printf("准备删除旧图标文件: %s -> %s", oldIcon, oldIconPath)
				if _, err := os.Stat(oldIconPath); err == nil {
					os.Remove(oldIconPath)
					log.Printf("旧图标文件删除成功: %s", oldIconPath)
				} else {
					log.Printf("旧图标文件不存在: %s", oldIconPath)
				}
			}
		}
	} else {
		// 图标没变，只检查分类变化
		if oldCategory != req.Category && oldIcon != "" && strings.HasPrefix(oldIcon, "/uploads/") {
			newIconPath = h.moveIconToNewCategory(oldIcon, oldCategory, req.Category, categories)
		}
	}

	// 处理图标设置逻辑
	icon := newIconPath
	if icon == "" {
		// 用户留空：使用书签网址拼接 "/favicon.ico"
		parsedURL, err := url.Parse(req.URL)
		if err == nil && parsedURL.Host != "" {
			icon = parsedURL.Scheme + "://" + parsedURL.Host + "/favicon.ico"
		} else {
			icon = "/favicon.ico" // 如果URL解析失败，使用默认图标
		}
	} else if strings.ToLower(strings.TrimSpace(icon)) == "local" {
		// 用户输入 "local"：使用默认图标
		icon = "/favicon.ico"
	}
	// 其他情况：保持用户输入的图标URL或上传的本地图标路径

	// 更新书签信息
	bookmarks[bookmarkIndex].Name = req.Name
	bookmarks[bookmarkIndex].URL = req.URL
	bookmarks[bookmarkIndex].Description = req.Description
	bookmarks[bookmarkIndex].Icon = icon
	bookmarks[bookmarkIndex].Category = req.Category
	bookmarks[bookmarkIndex].Tags = req.Tags
	bookmarks[bookmarkIndex].Sort = req.Sort
	bookmarks[bookmarkIndex].UpdatedAt = time.Now()

	if bookmarks[bookmarkIndex].Tags == nil {
		bookmarks[bookmarkIndex].Tags = []string{}
	}

	if err := h.storage.SaveBookmarks(bookmarks); err != nil {
		log.Printf("书签更新失败: %s - 保存数据失败", req.Name)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "保存书签失败",
		})
		return
	}

	session := sessions.Default(c)
	username := session.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}

	log.Printf("书签更新成功: %s (%s → %s) - 用户: %s", bookmarks[bookmarkIndex].Name, oldCategory, req.Category, usernameStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "书签更新成功",
		Data:    bookmarks[bookmarkIndex],
	})
}

// DeleteBookmark 删除书签
func (h *BookmarksHandler) DeleteBookmark(c *gin.Context) {
	id := c.Param("id")

	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "获取书签失败",
		})
		return
	}

	// 查找要删除的书签
	bookmarkIndex := -1
	for i, bookmark := range bookmarks {
		if bookmark.ID == id {
			bookmarkIndex = i
			break
		}
	}

	if bookmarkIndex == -1 {
		c.JSON(http.StatusNotFound, models.APIResponse{
			Success: false,
			Message: "书签不存在",
		})
		return
	}

	bookmarkToDelete := bookmarks[bookmarkIndex]

	// 如果书签有本地图标文件，先删除图标文件
	if bookmarkToDelete.Icon != "" && strings.HasPrefix(bookmarkToDelete.Icon, "/uploads/") {
		iconPath := filepath.Join(h.storage.GetDataPath(), strings.TrimPrefix(bookmarkToDelete.Icon, "/"))
		log.Printf("准备删除图标文件: %s -> %s", bookmarkToDelete.Icon, iconPath)

		if _, err := os.Stat(iconPath); err == nil {
			os.Remove(iconPath)
			log.Printf("图标文件删除成功: %s", iconPath)
		} else {
			log.Printf("图标文件不存在: %s", iconPath)
		}
	}

	// 删除书签记录
	newBookmarkslist := make([]models.Bookmark, 0, len(bookmarks)-1)
	for i, bookmark := range bookmarks {
		if i != bookmarkIndex {
			newBookmarkslist = append(newBookmarkslist, bookmark)
		}
	}

	if err := h.storage.SaveBookmarks(newBookmarkslist); err != nil {
		log.Printf("书签删除失败: %s - 保存数据失败", bookmarkToDelete.Name)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "删除书签失败",
		})
		return
	}

	session := sessions.Default(c)
	username := session.Get("username")
	usernameStr := "unknown"
	if username != nil {
		usernameStr = username.(string)
	}

	log.Printf("书签删除成功: %s (%s) - 用户: %s", bookmarkToDelete.Name, bookmarkToDelete.Category, usernameStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "书签删除成功",
	})
}

// SearchBookmarksH 搜索书签
func (h *BookmarksHandler) SearchBookmarksH(c *gin.Context) {
	keyword := c.Param("keyword")
	keyword = strings.ToLower(keyword)

	bookmarks, err := h.storage.GetBookmarks()
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "搜索书签失败",
		})
		return
	}

	var filteredBookmarks []models.Bookmark
	for _, bookmark := range bookmarks {
		nameMatch := strings.Contains(strings.ToLower(bookmark.Name), keyword)
		descMatch := strings.Contains(strings.ToLower(bookmark.Description), keyword)

		tagsMatch := false
		for _, tag := range bookmark.Tags {
			if strings.Contains(strings.ToLower(tag), keyword) {
				tagsMatch = true
				break
			}
		}

		if nameMatch || descMatch || tagsMatch {
			filteredBookmarks = append(filteredBookmarks, bookmark)
		}
	}

	// 按排序值排序
	sort.Slice(filteredBookmarks, func(i, j int) bool {
		if filteredBookmarks[i].Sort == filteredBookmarks[j].Sort {
			return filteredBookmarks[i].CreatedAt.Before(filteredBookmarks[j].CreatedAt)
		}
		return filteredBookmarks[i].Sort < filteredBookmarks[j].Sort
	})

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Data:    filteredBookmarks,
	})
}
