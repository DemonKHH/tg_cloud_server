package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/services"
)

// AuthMiddleware JWT认证中间件
func AuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	log := logger.Get().Named("auth_middleware")

	return func(c *gin.Context) {
		// 获取Authorization头
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			log.Warn("Missing authorization header", 
				zap.String("path", c.Request.URL.Path))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "缺少认证令牌",
			})
			c.Abort()
			return
		}

		// 检查Bearer前缀
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			log.Warn("Invalid authorization header format", 
				zap.String("header", authHeader))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "无效的认证令牌格式",
			})
			c.Abort()
			return
		}

		// 提取令牌
		token := authHeader[len(bearerPrefix):]
		if token == "" {
			log.Warn("Empty token")
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "认证令牌为空",
			})
			c.Abort()
			return
		}

		// 验证令牌
		userID, err := authService.VerifyToken(token)
		if err != nil {
			log.Warn("Token verification failed", 
				zap.Error(err),
				zap.String("token_prefix", token[:min(10, len(token))]))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "无效的认证令牌",
			})
			c.Abort()
			return
		}

		// 将用户ID存储到上下文
		c.Set("user_id", userID)

		// 继续处理请求
		c.Next()
	}
}

// min 返回两个整数中的较小值
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
