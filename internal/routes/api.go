package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/config"
	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/services"
)

// RegisterAPIRoutes 注册API路由
func RegisterAPIRoutes(
	router *gin.Engine,
	accountHandler *handlers.AccountHandler,
	taskHandler *handlers.TaskHandler,
	proxyHandler *handlers.ProxyHandler,
	moduleHandler *handlers.ModuleHandler,
	authService *services.AuthService,
	config *config.Config,
) {
	// 注册各模块路由
	SetupTaskRoutes(router, taskHandler)
	SetupProxyRoutes(router, proxyHandler)

	// API路由组（需要认证）
	api := router.Group("/api/v1")
	api.Use(middleware.JWTAuthMiddleware())

	// 用户资料管理
	{
		authHandler := handlers.NewAuthHandler(authService)
		api.GET("/auth/profile", authHandler.GetProfile)
		api.PUT("/auth/profile", authHandler.UpdateProfile)
		api.POST("/auth/logout", authHandler.Logout)
	}

	// 账号管理路由
	accounts := api.Group("/accounts")
	{
		accounts.POST("", accountHandler.CreateAccount)                          // 创建账号
		accounts.GET("", accountHandler.GetAccounts)                             // 获取账号列表
		accounts.GET("/:id", accountHandler.GetAccount)                          // 获取账号详情
		accounts.PUT("/:id", accountHandler.UpdateAccount)                       // 更新账号
		accounts.DELETE("/:id", accountHandler.DeleteAccount)                    // 删除账号
		accounts.GET("/:id/health", accountHandler.CheckAccountHealth)           // 检查健康度
		accounts.GET("/:id/availability", accountHandler.GetAccountAvailability) // 获取可用性
		accounts.POST("/:id/bind-proxy", accountHandler.BindProxy)               // 绑定代理
	}

	// 模块功能路由（五大核心模块）
	modules := api.Group("/modules")
	{
		modules.POST("/check", moduleHandler.AccountCheck)     // 账号检查模块
		modules.POST("/private", moduleHandler.PrivateMessage) // 私信模块
		modules.POST("/broadcast", moduleHandler.Broadcast)    // 群发模块
		modules.POST("/verify", moduleHandler.VerifyCode)      // 验证码接收模块
		modules.POST("/groupchat", moduleHandler.GroupChat)    // AI炒群模块
	}

	// 统计和监控路由
	stats := api.Group("/stats")
	{
		stats.GET("/overview", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "统计概览接口待实现"})
		})
		stats.GET("/accounts", func(c *gin.Context) {
			c.JSON(200, gin.H{"message": "账号统计接口待实现"})
		})
		stats.GET("/tasks", taskHandler.GetTaskStats)     // 任务统计
		stats.GET("/proxies", proxyHandler.GetProxyStats) // 代理统计
	}
}
