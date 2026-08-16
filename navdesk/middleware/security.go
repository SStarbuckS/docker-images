package middleware

import "github.com/gin-gonic/gin"

/**
 * 添加浏览器安全响应头
 */
func SecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Next()
	}
}
