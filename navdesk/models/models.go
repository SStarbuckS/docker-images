package models

import (
	"time"
)

// User 用户模型
type User struct {
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"createdAt"`
}

// Category 分类模型
type Category struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Icon      string    `json:"icon"`
	UploadDir string    `json:"uploadDir"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt,omitempty"`
}

// Bookmark 书签模型
type Bookmark struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Category    string    `json:"category"`
	Tags        []string  `json:"tags"`
	Sort        int       `json:"sort"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt,omitempty"`
}

// Settings 设置模型
type Settings struct {
	SiteTitle    string    `json:"siteTitle"`
	CardWidth    int       `json:"cardWidth"`
	CardHeight   int       `json:"cardHeight"`
	IconWidth    int       `json:"iconWidth"`
	IconHeight   int       `json:"iconHeight"`
	SidebarWidth int       `json:"sidebarWidth"`
	Theme        string    `json:"theme"`
	UpdatedAt    time.Time `json:"updatedAt"`
}

// APIResponse 通用API响应
type APIResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
}

// DataResponse 数据接口响应
type DataResponse struct {
	Categories []Category `json:"categories"`
	Bookmarks  []Bookmark `json:"bookmarks"`
	Settings   Settings   `json:"settings"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// UserSession 用户会话信息
type UserSession struct {
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	LoginTime time.Time `json:"loginTime"`
}

// CreateCategoryRequest 创建分类请求
type CreateCategoryRequest struct {
	Name      string `json:"name" binding:"required"`
	Icon      string `json:"icon" binding:"required"`
	UploadDir string `json:"uploadDir" binding:"required"`
	Sort      int    `json:"sort"`
}

// UpdateCategoryRequest 更新分类请求
type UpdateCategoryRequest struct {
	Name      string `json:"name" binding:"required"`
	Icon      string `json:"icon" binding:"required"`
	UploadDir string `json:"uploadDir" binding:"required"`
	Sort      int    `json:"sort"`
}

// CreateBookmarkRequest 创建书签请求
type CreateBookmarkRequest struct {
	Name        string   `json:"name" binding:"required"`
	URL         string   `json:"url" binding:"required"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Category    string   `json:"category" binding:"required"`
	Tags        []string `json:"tags"`
	Sort        int      `json:"sort"`
}

// UpdateBookmarkRequest 更新书签请求
type UpdateBookmarkRequest struct {
	Name        string   `json:"name" binding:"required"`
	URL         string   `json:"url" binding:"required"`
	Description string   `json:"description"`
	Icon        string   `json:"icon"`
	Category    string   `json:"category" binding:"required"`
	Tags        []string `json:"tags"`
	Sort        int      `json:"sort"`
}

// UpdateSettingsRequest 更新设置请求
type UpdateSettingsRequest struct {
	SiteTitle    string `json:"siteTitle" binding:"required"`
	CardWidth    int    `json:"cardWidth" binding:"required"`
	CardHeight   int    `json:"cardHeight" binding:"required"`
	IconWidth    int    `json:"iconWidth" binding:"required"`
	IconHeight   int    `json:"iconHeight" binding:"required"`
	SidebarWidth int    `json:"sidebarWidth" binding:"required"`
	Theme        string `json:"theme" binding:"required"`
}

// UploadResponse 上传响应
type UploadResponse struct {
	URL          string `json:"url"`
	Filename     string `json:"filename"`
	OriginalName string `json:"originalname"`
	Size         int64  `json:"size"`
}
