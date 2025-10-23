package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/handlers"
)

// RegisterAuthRoutes 注册认证相关路由
func RegisterAuthRoutes(router *gin.Engine, authHandler *handlers.AuthHandler) {
	// 认证路由组（无需认证）
	auth := router.Group("/api/v1/auth")
	{
		auth.POST("/register", authHandler.Register)     // 用户注册
		auth.POST("/login", authHandler.Login)           // 用户登录
		auth.POST("/refresh", authHandler.RefreshToken)  // 刷新令牌
	}
}
