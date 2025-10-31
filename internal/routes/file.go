package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/services"
)

// SetupFileRoutes 设置文件路由
func SetupFileRoutes(
	router *gin.RouterGroup,
	fileHandler *handlers.FileHandler,
	authService *services.AuthService,
) {
	// 文件管理路由组
	fileGroup := router.Group("/files")
	fileGroup.Use(middleware.JWTAuthMiddleware(authService))

	// 基本文件操作
	fileGroup.POST("/upload", fileHandler.UploadFile)        // 上传文件
	fileGroup.POST("/upload-url", fileHandler.UploadFromURL) // 从URL上传文件
	fileGroup.GET("/", fileHandler.GetFiles)                 // 获取文件列表
	fileGroup.GET("/:id", fileHandler.GetFile)               // 获取文件信息
	fileGroup.DELETE("/:id", fileHandler.DeleteFile)         // 删除文件

	// 文件访问
	fileGroup.GET("/:id/download", fileHandler.DownloadFile) // 下载文件
	fileGroup.GET("/:id/preview", fileHandler.PreviewFile)   // 预览文件
	fileGroup.GET("/:id/url", fileHandler.GetFileURL)        // 获取文件URL

	// 批量操作（需要高级用户权限）
	fileGroup.POST("/batch-upload", middleware.RequirePermission("advanced_features"), fileHandler.BatchUpload)   // 批量上传
	fileGroup.DELETE("/batch-delete", middleware.RequirePermission("advanced_features"), fileHandler.BatchDelete) // 批量删除
}
