package middleware

import (
	"time"

	"tg_cloud_server/internal/common/response"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// Logger 日志中间件
func Logger(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		// 处理请求
		c.Next()

		// 计算处理时间
		latency := time.Since(start)

		// 获取客户端IP
		clientIP := c.ClientIP()

		// 获取请求方法
		method := c.Request.Method

		// 获取状态码
		statusCode := c.Writer.Status()

		// 构建完整路径
		if raw != "" {
			path = path + "?" + raw
		}

		// 根据状态码选择日志级别
		fields := []zap.Field{
			zap.Int("status", statusCode),
			zap.String("method", method),
			zap.String("path", path),
			zap.String("ip", clientIP),
			zap.Duration("latency", latency),
			zap.String("user_agent", c.Request.UserAgent()),
		}

		// 添加用户ID（如果存在）
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		// 添加错误信息（如果有）
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// 根据状态码记录不同级别的日志
		switch {
		case statusCode >= 400 && statusCode < 500:
			logger.Warn("Client error", fields...)
		case statusCode >= 500:
			logger.Error("Server error", fields...)
		case statusCode >= 300 && statusCode < 400:
			logger.Info("Redirect", fields...)
		default:
			logger.Info("Request completed", fields...)
		}
	}
}

// Recovery 恢复中间件
func Recovery(logger *zap.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				// 记录panic信息
				logger.Error("Panic recovered",
					zap.Any("error", err),
					zap.String("method", c.Request.Method),
					zap.String("path", c.Request.URL.Path),
					zap.String("ip", c.ClientIP()),
					zap.String("user_agent", c.Request.UserAgent()),
				)

				// 返回500错误
				// 返回500错误
				response.InternalError(c, "服务器内部错误")
				c.Abort()
			}
		}()

		c.Next()
	}
}
