package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/services"
)

// SetupTaskRoutes 设置任务相关路由
func SetupTaskRoutes(router *gin.Engine, taskHandler *handlers.TaskHandler, authService *services.AuthService) {
	// 任务管理API路由组
	taskGroup := router.Group("/api/v1/tasks")
	taskGroup.Use(middleware.JWTAuthMiddleware(authService))
	{
		// 任务基本操作
		taskGroup.POST("", taskHandler.CreateTask)            // 创建任务
		taskGroup.GET("", taskHandler.GetTasks)               // 获取任务列表
		taskGroup.GET("/:id", taskHandler.GetTask)            // 获取任务详情
		taskGroup.POST("/:id/update", taskHandler.UpdateTask) // 更新任务
		taskGroup.POST("/:id/cancel", taskHandler.CancelTask) // 取消任务

		// 任务操作
		taskGroup.POST("/:id/retry", taskHandler.RetryTask) // 重试任务
		taskGroup.GET("/:id/logs", taskHandler.GetTaskLogs) // 获取任务日志

		// 批量操作（需要高级用户权限）
		taskGroup.POST("/batch/cancel", middleware.RequirePermission("advanced_features"), taskHandler.BatchCancel) // 批量取消任务

		// 统计与监控
		taskGroup.GET("/stats", taskHandler.GetTaskStats)                                 // 获取任务统计
		taskGroup.POST("/cleanup", middleware.RequirePremium(), taskHandler.CleanupTasks) // 清理已完成任务（需要高级用户）
	}

	// 账号队列信息路由
	queueGroup := router.Group("/api/v1/accounts/:id/queue")
	queueGroup.Use(middleware.JWTAuthMiddleware(authService))
	{
		queueGroup.GET("", taskHandler.GetQueueInfo) // 获取队列信息
	}
}
