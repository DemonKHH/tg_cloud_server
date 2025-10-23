package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
)

// SetupTaskRoutes 设置任务相关路由
func SetupTaskRoutes(router *gin.Engine, taskHandler *handlers.TaskHandler) {
	// 任务管理API路由组
	taskGroup := router.Group("/api/v1/tasks")
	taskGroup.Use(middleware.JWTAuthMiddleware())
	{
		// 任务基本操作
		taskGroup.POST("", taskHandler.CreateTask)       // 创建任务
		taskGroup.GET("", taskHandler.GetTasks)          // 获取任务列表
		taskGroup.GET("/:id", taskHandler.GetTask)       // 获取任务详情
		taskGroup.PUT("/:id", taskHandler.UpdateTask)    // 更新任务
		taskGroup.DELETE("/:id", taskHandler.CancelTask) // 取消任务

		// 任务操作
		taskGroup.POST("/:id/retry", taskHandler.RetryTask) // 重试任务
		taskGroup.GET("/:id/logs", taskHandler.GetTaskLogs) // 获取任务日志

		// 批量操作
		taskGroup.POST("/batch/cancel", taskHandler.BatchCancel) // 批量取消任务

		// 统计与监控
		taskGroup.GET("/stats", taskHandler.GetTaskStats)    // 获取任务统计
		taskGroup.POST("/cleanup", taskHandler.CleanupTasks) // 清理已完成任务
	}

	// 账号队列信息路由
	queueGroup := router.Group("/api/v1/accounts/:account_id/queue")
	queueGroup.Use(middleware.JWTAuthMiddleware())
	{
		queueGroup.GET("", taskHandler.GetQueueInfo) // 获取队列信息
	}
}
