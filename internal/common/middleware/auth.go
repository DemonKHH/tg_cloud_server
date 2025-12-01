package middleware

import (
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
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
			response.Unauthorized(c, "缺少认证令牌")
			c.Abort()
			return
		}

		// 检查Bearer前缀
		const bearerPrefix = "Bearer "
		if !strings.HasPrefix(authHeader, bearerPrefix) {
			log.Warn("Invalid authorization header format",
				zap.String("header", authHeader))
			response.Unauthorized(c, "无效的认证令牌格式")
			c.Abort()
			return
		}

		// 提取令牌
		token := authHeader[len(bearerPrefix):]
		if token == "" {
			log.Warn("Empty token")
			response.Unauthorized(c, "认证令牌为空")
			c.Abort()
			return
		}

		// 验证令牌
		userID, err := authService.VerifyToken(token)
		if err != nil {
			log.Warn("Token verification failed",
				zap.Error(err),
				zap.String("token_prefix", token[:min(10, len(token))]))
			response.Unauthorized(c, "无效的认证令牌")
			c.Abort()
			return
		}

		// 获取用户信息（包括角色和权限）
		userProfile, err := authService.GetUserProfile(userID)
		if err != nil {
			log.Warn("Failed to get user profile",
				zap.Uint64("user_id", userID),
				zap.Error(err))
			response.Unauthorized(c, "无法获取用户信息")
			c.Abort()
			return
		}

		// 检查用户状态
		if !userProfile.IsActive {
			log.Warn("User account is inactive",
				zap.Uint64("user_id", userID))
			response.Forbidden(c, "用户账号已被禁用")
			c.Abort()
			return
		}

		// 检查用户是否过期
		if userProfile.IsExpired {
			log.Warn("User account is expired",
				zap.Uint64("user_id", userID),
				zap.Time("expires_at", *userProfile.ExpiresAt))
			response.Forbidden(c, "用户账号已过期，请联系管理员续费")
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
