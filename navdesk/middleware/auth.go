package middleware

import (
	"net/http"
	"time"

	"navdesk/models"

	"github.com/gin-contrib/sessions"
	"github.com/gin-gonic/gin"
)

// RequireAuth 需要登录认证的中间件
func RequireAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := sessions.Default(c)
		username := session.Get("username")

		if username == nil {
			c.JSON(http.StatusUnauthorized, models.APIResponse{
				Success: false,
				Message: "需要登录",
			})
			c.Abort()
			return
		}

		c.Next()
	}
}

// GetCurrentUser 获取当前登录用户信息
func GetCurrentUser(c *gin.Context) *models.UserSession {
	session := sessions.Default(c)
	username := session.Get("username")

	if username == nil {
		return nil
	}

	role := session.Get("role")
	loginTimeStr := session.Get("loginTime")

	userSession := &models.UserSession{
		Username: username.(string),
		Role:     role.(string),
	}

	if loginTimeStr != nil {
		if timeStr, ok := loginTimeStr.(string); ok {
			if loginTime, err := time.Parse(time.RFC3339, timeStr); err == nil {
				userSession.LoginTime = loginTime
			}
		}
	}

	return userSession
}
