package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/services"
)

// SetupTemplateRoutes 设置模板路由
func SetupTemplateRoutes(
	router *gin.RouterGroup,
	templateHandler *handlers.TemplateHandler,
	authService *services.AuthService,
) {
	// 模板管理路由组
	templateGroup := router.Group("/templates")
	templateGroup.Use(middleware.JWTAuthMiddleware(authService))

	// 基本CRUD操作
	templateGroup.POST("/", templateHandler.CreateTemplate)      // 创建模板
	templateGroup.GET("/", templateHandler.GetTemplates)         // 获取模板列表
	templateGroup.GET("/:id", templateHandler.GetTemplate)       // 获取模板详情
	templateGroup.PUT("/:id", templateHandler.UpdateTemplate)    // 更新模板
	templateGroup.DELETE("/:id", templateHandler.DeleteTemplate) // 删除模板

	// 模板操作
	templateGroup.POST("/render", templateHandler.RenderTemplate)           // 渲染模板
	templateGroup.POST("/validate", templateHandler.ValidateTemplate)       // 验证模板
	templateGroup.POST("/:id/duplicate", templateHandler.DuplicateTemplate) // 复制模板

	// 批量操作（需要高级用户权限）
	templateGroup.POST("/batch", middleware.RequirePermission("advanced_features"), templateHandler.BatchOperation)   // 批量操作
	templateGroup.POST("/import", middleware.RequirePermission("advanced_features"), templateHandler.ImportTemplates) // 导入模板
	templateGroup.GET("/export", middleware.RequirePermission("advanced_features"), templateHandler.ExportTemplates)  // 导出模板

	// 统计分析
	templateGroup.GET("/stats", templateHandler.GetTemplateStats)      // 模板统计
	templateGroup.GET("/popular", templateHandler.GetPopularTemplates) // 热门模板
}
