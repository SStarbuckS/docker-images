package storage

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"

	"navdesk/models"
)

const (
	dataDir        = "./data"
	usersFile      = "users.json"
	categoriesFile = "categories.json"
	bookmarksFile  = "bookmarks.json"
	settingsFile   = "settings.json"
	uploadsDir     = "uploads"
)

// Storage 存储接口
type Storage struct {
	dataPath string
}

// NewStorage 创建存储实例
func NewStorage() *Storage {
	return &Storage{
		dataPath: dataDir,
	}
}

// 确保目录存在（保留此函数，因为上传功能仍需要）
func (s *Storage) ensureDirectoryExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return os.MkdirAll(dir, 0755)
	}
	return nil
}

// GetUsers 获取用户数据
func (s *Storage) GetUsers() (map[string]models.User, error) {
	usersPath := filepath.Join(s.dataPath, usersFile)
	data, err := ioutil.ReadFile(usersPath)
	if err != nil {
		return nil, err
	}

	var usersData map[string]interface{}
	if err := json.Unmarshal(data, &usersData); err != nil {
		return nil, err
	}

	users := make(map[string]models.User)
	for key, value := range usersData {
		if key == "secretKey" {
			continue // 跳过secretKey字段
		}
		userBytes, _ := json.Marshal(value)
		var user models.User
		if err := json.Unmarshal(userBytes, &user); err == nil {
			users[key] = user
		}
	}

	return users, nil
}

// GetSecretKey 获取会话密钥
func (s *Storage) GetSecretKey() (string, error) {
	usersPath := filepath.Join(s.dataPath, usersFile)
	data, err := ioutil.ReadFile(usersPath)
	if err != nil {
		return "", err
	}

	var usersData map[string]interface{}
	if err := json.Unmarshal(data, &usersData); err != nil {
		return "", err
	}

	if secretKey, exists := usersData["secretKey"]; exists {
		if keyStr, ok := secretKey.(string); ok {
			return keyStr, nil
		}
	}

	return "", nil // 如果没有找到secretKey，返回空字符串
}

// GetCategories 获取分类数据
func (s *Storage) GetCategories() ([]models.Category, error) {
	categoriesPath := filepath.Join(s.dataPath, categoriesFile)
	data, err := ioutil.ReadFile(categoriesPath)
	if err != nil {
		return nil, err
	}

	var categories []models.Category
	if err := json.Unmarshal(data, &categories); err != nil {
		return nil, err
	}

	return categories, nil
}

// SaveCategories 保存分类数据
func (s *Storage) SaveCategories(categories []models.Category) error {
	categoriesPath := filepath.Join(s.dataPath, categoriesFile)
	data, err := json.MarshalIndent(categories, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(categoriesPath, data, 0644)
}

// GetBookmarks 获取书签数据
func (s *Storage) GetBookmarks() ([]models.Bookmark, error) {
	bookmarksPath := filepath.Join(s.dataPath, bookmarksFile)
	data, err := ioutil.ReadFile(bookmarksPath)
	if err != nil {
		return nil, err
	}

	var bookmarks []models.Bookmark
	if err := json.Unmarshal(data, &bookmarks); err != nil {
		return nil, err
	}

	return bookmarks, nil
}

// SaveBookmarks 保存书签数据
func (s *Storage) SaveBookmarks(bookmarks []models.Bookmark) error {
	bookmarksPath := filepath.Join(s.dataPath, bookmarksFile)
	data, err := json.MarshalIndent(bookmarks, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(bookmarksPath, data, 0644)
}

// GetSettings 获取设置数据
func (s *Storage) GetSettings() (models.Settings, error) {
	settingsPath := filepath.Join(s.dataPath, settingsFile)
	data, err := ioutil.ReadFile(settingsPath)
	if err != nil {
		// 如果文件不存在，返回默认设置
		return models.Settings{
			SiteTitle:    "极简网站导航",
			CardWidth:    180,
			CardHeight:   80,
			IconWidth:    50,
			IconHeight:   50,
			SidebarWidth: 300,
			Theme:        "auto",
			UpdatedAt:    time.Now(),
		}, nil
	}

	var settings models.Settings
	if err := json.Unmarshal(data, &settings); err != nil {
		return models.Settings{}, err
	}

	return settings, nil
}

// SaveSettings 保存设置数据
func (s *Storage) SaveSettings(settings models.Settings) error {
	settingsPath := filepath.Join(s.dataPath, settingsFile)
	settings.UpdatedAt = time.Now()
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return err
	}

	return ioutil.WriteFile(settingsPath, data, 0644)
}

// GetDataPath 获取数据目录路径
func (s *Storage) GetDataPath() string {
	return s.dataPath
}

// GetUploadsPath 获取上传目录路径
func (s *Storage) GetUploadsPath() string {
	return filepath.Join(s.dataPath, uploadsDir)
}
