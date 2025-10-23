package middleware

import (
	"tg_cloud_server/internal/services"

	"github.com/gin-gonic/gin"
)

// JWTAuthMiddleware JWT认证中间件的别名
// 实际实现在auth.go中
func JWTAuthMiddleware(authService *services.AuthService) gin.HandlerFunc {
	return AuthMiddleware(authService)
}
