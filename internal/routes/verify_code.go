package routes

import (
	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/services"
)

// SetupVerifyCodeRoutes 设置验证码相关路由
func SetupVerifyCodeRoutes(
	router *gin.Engine,
	verifyCodeHandler *handlers.VerifyCodeHandler,
	authService *services.AuthService,
) {
	// 验证码API路由组
	verifyGroup := router.Group("/api/v1/verify-code")
	{
		// 需要认证的接口
		authenticatedGroup := verifyGroup.Group("")
		authenticatedGroup.Use(middleware.JWTAuthMiddleware(authService))
		{
			// 生成验证码访问链接
			authenticatedGroup.POST("/generate", verifyCodeHandler.GenerateCode)
			authenticatedGroup.POST("/batch/generate", verifyCodeHandler.BatchGenerateCode)

			// 获取访问码信息 (调试用)
			authenticatedGroup.GET("/:code/info", verifyCodeHandler.GetCodeInfo)
		}

		// 公开接口 (不需要认证)
		// 通过访问码获取验证码
		verifyGroup.GET("/:code", verifyCodeHandler.GetVerifyCode)
	}
}
