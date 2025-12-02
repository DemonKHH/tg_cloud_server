package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/services"
)

// SetupProxyRoutes 设置代理相关路由
func SetupProxyRoutes(router *gin.Engine, proxyHandler *handlers.ProxyHandler, authService *services.AuthService) {
	// 代理管理API路由组
	proxyGroup := router.Group("/api/v1/proxies")
	proxyGroup.Use(middleware.JWTAuthMiddleware(authService))
	{
		// 代理基本操作
		proxyGroup.POST("", proxyHandler.CreateProxy)            // 创建代理
		proxyGroup.POST("/batch", proxyHandler.BatchCreateProxy) // 批量创建代理
		proxyGroup.GET("", proxyHandler.GetProxies)              // 获取代理列表
		proxyGroup.GET("/:id", proxyHandler.GetProxy)            // 获取代理详情
		proxyGroup.POST("/:id/update", proxyHandler.UpdateProxy) // 更新代理
		proxyGroup.POST("/:id/delete", proxyHandler.DeleteProxy) // 删除代理

		// 代理测试
		proxyGroup.POST("/:id/test", proxyHandler.TestProxy) // 测试代理

		// 代理统计
		proxyGroup.GET("/stats", proxyHandler.GetProxyStats) // 获取代理统计

		// 批量操作
		proxyGroup.POST("/batch/delete", proxyHandler.BatchDeleteProxy) // 批量删除代理
		proxyGroup.POST("/batch/test", proxyHandler.BatchTestProxy)     // 批量测试代理
	}
}
