package handlers

import (
	"log"
	"net/http"
	"time"

	"navdesk/models"
	"navdesk/storage"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// AuthHandler 认证处理器
type AuthHandler struct {
	storage *storage.Storage
}

// NewAuthHandler 创建认证处理器
func NewAuthHandler(storage *storage.Storage) *AuthHandler {
	return &AuthHandler{
		storage: storage,
	}
}

// Login 用户登录
func (h *AuthHandler) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.APIResponse{
			Success: false,
			Message: "用户名和密码不能为空",
		})
		return
	}

	users, err := h.storage.GetUsers()
	if err != nil {
		log.Printf("Error reading users: %v", err)
		c.JSON(http.StatusInternalServerError, models.APIResponse{
			Success: false,
			Message: "登录失败",
		})
		return
	}

	// 查找用户
	var foundUser *models.User
	for _, user := range users {
		if user.Username == req.Username && user.Password == req.Password {
			foundUser = &user
			break
		}
	}

	if foundUser == nil {
		log.Printf("登录失败: 用户名 %s - 账号或密码错误", req.Username)
		c.JSON(http.StatusUnauthorized, models.APIResponse{
			Success: false,
			Message: "账号或密码错误",
		})
		return
	}

	// 创建会话
	session := sessions.Default(c)
	session.Set("username", foundUser.Username)
	session.Set("role", foundUser.Role)
	session.Set("loginTime", time.Now().Format(time.RFC3339))
	if err := session.Save(); err != nil {
		log.Printf("Session保存失败: %v", err)
	} else {
		log.Printf("Session保存成功: username=%s, role=%s", foundUser.Username, foundUser.Role)
	}

	log.Printf("用户登录成功: %s (%s)", foundUser.Username, foundUser.Role)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "登录成功",
		Data: map[string]interface{}{
			"username": foundUser.Username,
			"role":     foundUser.Role,
		},
	})
}

// Logout 用户登出
func (h *AuthHandler) Logout(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	var usernameStr string = "未知用户"
	if username != nil {
		if name, ok := username.(string); ok {
			usernameStr = name
		}
	}

	session.Delete("username")
	session.Delete("role")
	session.Delete("loginTime")
	session.Save()

	log.Printf("用户登出成功: %s", usernameStr)

	c.JSON(http.StatusOK, models.APIResponse{
		Success: true,
		Message: "登出成功",
	})
}

// Status 检查登录状态
func (h *AuthHandler) Status(c *gin.Context) {
	session := sessions.Default(c)
	username := session.Get("username")

	if username != nil {
		role := session.Get("role")
		loginTime := session.Get("loginTime")

		c.JSON(http.StatusOK, map[string]interface{}{
			"success":    true,
			"isLoggedIn": true,
			"user": map[string]interface{}{
				"username":  username,
				"role":      role,
				"loginTime": loginTime,
			},
		})
		return
	}

	c.JSON(http.StatusOK, map[string]interface{}{
		"success":    true,
		"isLoggedIn": false,
	})
}
