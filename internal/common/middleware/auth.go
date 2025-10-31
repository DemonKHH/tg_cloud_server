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

		// 获取用户信息（包括角色和权限）
		userProfile, err := authService.GetUserProfile(userID)
		if err != nil {
			log.Warn("Failed to get user profile",
				zap.Uint64("user_id", userID),
				zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{
				"error":   "unauthorized",
				"message": "无法获取用户信息",
			})
			c.Abort()
			return
		}

		// 检查用户状态
		if !userProfile.IsActive {
			log.Warn("User account is inactive",
				zap.Uint64("user_id", userID))
			c.JSON(http.StatusForbidden, gin.H{
				"error":   "forbidden",
				"message": "用户账号已被禁用",
			})
			c.Abort()
			return
		}

		// 将用户信息存储到上下文
		c.Set("user_id", userID)
		c.Set("user_role", userProfile.Role)
		c.Set("user_profile", userProfile)

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
