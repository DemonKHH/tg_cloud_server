package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/services"
)

// SetupAIRoutes 设置AI服务路由
func SetupAIRoutes(
	router *gin.RouterGroup,
	aiHandler *handlers.AIHandler,
	authService *services.AuthService,
) {
	// AI服务路由组
	aiGroup := router.Group("/ai")
	aiGroup.Use(middleware.JWTAuthMiddleware(authService))

	// 内容生成
	aiGroup.POST("/group-chat", aiHandler.GenerateGroupChatResponse)   // 生成群聊回复
	aiGroup.POST("/private-message", aiHandler.GeneratePrivateMessage) // 生成私信内容

	// 文本分析
	aiGroup.POST("/analyze-sentiment", aiHandler.AnalyzeSentiment)     // 情感分析
	aiGroup.POST("/extract-keywords", aiHandler.ExtractKeywords)       // 关键词提取
	aiGroup.POST("/generate-variations", aiHandler.GenerateVariations) // 生成变体

	// 服务管理
	aiGroup.GET("/config", aiHandler.GetAIConfig)  // 获取AI配置
	aiGroup.POST("/test", aiHandler.TestAIService) // 测试AI服务
}
