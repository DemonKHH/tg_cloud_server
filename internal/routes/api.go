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
	statsHandler *handlers.StatsHandler,
	settingsHandler *handlers.SettingsHandler,
	aiHandler *handlers.AIHandler,
	authService *services.AuthService,
	config *config.Config,
) {
	// 注册各模块路由
	SetupTaskRoutes(router, taskHandler, authService)
	SetupProxyRoutes(router, proxyHandler, authService)

	// API路由组（需要认证）
	api := router.Group("/api/v1")

	// 添加日志中间件
	api.Use(middleware.APILoggerMiddleware())
	api.Use(middleware.TaskLoggerMiddleware())

	// 如果需要详细日志（包含请求响应体），可以启用这个中间件
	// api.Use(middleware.DetailedAPILoggerMiddleware())

	api.Use(middleware.JWTAuthMiddleware(authService))

	// 用户资料管理
	{
		authHandler := handlers.NewAuthHandler(authService)
		api.GET("/auth/profile", authHandler.GetProfile)
		api.POST("/auth/profile", authHandler.UpdateProfile)
		api.POST("/auth/logout", authHandler.Logout)
	}

	// 账号管理路由
	accounts := api.Group("/accounts")
	{
		accounts.POST("", accountHandler.CreateAccount)                          // 创建账号
		accounts.GET("", accountHandler.GetAccounts)                             // 获取账号列表
		accounts.GET("/:id", accountHandler.GetAccount)                          // 获取账号详情
		accounts.POST("/:id/update", accountHandler.UpdateAccount)               // 更新账号
		accounts.POST("/:id/delete", accountHandler.DeleteAccount)               // 删除账号
		accounts.GET("/:id/health", accountHandler.CheckAccountHealth)           // 检查健康度
		accounts.GET("/:id/availability", accountHandler.GetAccountAvailability) // 获取可用性
		accounts.POST("/:id/bind-proxy", accountHandler.BindProxy)               // 绑定代理
		accounts.POST("/upload", accountHandler.UploadAccountFiles)              // 上传并解析账号文件
		accounts.POST("/export", accountHandler.ExportAccounts)                  // 导出账号

		// 批量操作
		accounts.POST("/batch/bind-proxy", accountHandler.BatchBindProxy)  // 批量绑定/解绑代理
		accounts.POST("/batch/set-2fa", accountHandler.BatchSet2FA)        // 批量设置2FA
		accounts.POST("/batch/update-2fa", accountHandler.BatchUpdate2FA)  // 批量修改2FA
		accounts.POST("/batch/delete", accountHandler.BatchDeleteAccounts) // 批量删除账号
	}

	// 模块功能路由（五大核心模块）- 需要基础权限
	modules := api.Group("/modules")
	modules.Use(middleware.RequirePermission("basic_features"))
	{
		modules.POST("/check", moduleHandler.AccountCheck)     // 账号检查模块
		modules.POST("/private", moduleHandler.PrivateMessage) // 私信模块
		modules.POST("/broadcast", moduleHandler.Broadcast)    // 群发模块
		modules.POST("/verify", moduleHandler.VerifyCode)      // 验证码接收模块
		modules.POST("/groupchat", moduleHandler.GroupChat)    // AI炒群模块
	}

	// AI服务路由
	SetupAIRoutes(api, aiHandler, authService)

	// 统计和监控路由（需要标准用户权限）
	stats := api.Group("/stats")
	stats.Use(middleware.RequirePermission("basic_features"))
	{
		stats.GET("/overview", statsHandler.GetOverview)       // 系统统计概览
		stats.GET("/accounts", statsHandler.GetAccountStats)   // 账号统计详情
		stats.GET("/dashboard", statsHandler.GetUserDashboard) // 用户仪表盘
		stats.GET("/tasks", taskHandler.GetTaskStats)          // 任务统计
		stats.GET("/proxies", proxyHandler.GetProxyStats)      // 代理统计
	}

	// 设置路由
	settings := api.Group("/settings")
	{
		settings.GET("/risk", settingsHandler.GetRiskSettings)    // 获取风控配置
		settings.PUT("/risk", settingsHandler.UpdateRiskSettings) // 更新风控配置
	}
}
