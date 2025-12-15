package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"tg_cloud_server/internal/common/logger"
)

// APILoggerMiddleware API日志中间件
func APILoggerMiddleware() gin.HandlerFunc {
	return gin.LoggerWithFormatter(func(param gin.LogFormatterParams) string {
		// 使用专门的API日志记录器
		fields := []zap.Field{
			zap.String("method", param.Method),
			zap.String("path", param.Path),
			zap.String("protocol", param.Request.Proto),
			zap.Int("status_code", param.StatusCode),
			zap.Duration("latency", param.Latency),
			zap.String("client_ip", param.ClientIP),
			zap.String("user_agent", param.Request.UserAgent()),
		}

		// 添加错误信息（如果有）
		if param.ErrorMessage != "" {
			fields = append(fields, zap.String("error", param.ErrorMessage))
		}

		// // 根据状态码确定日志级别
		// var level zapcore.Level
		// if param.StatusCode >= 500 {
		// 	level = zapcore.ErrorLevel
		// } else if param.StatusCode >= 400 {
		// 	level = zapcore.WarnLevel
		// } else {
		// 	level = zapcore.InfoLevel
		// }

		// logger.LogAPI(level, "API Request", fields...)
		return "" // 返回空字符串，因为我们使用了自定义日志记录
	})
}

// DetailedAPILoggerMiddleware 详细的API日志中间件（包含请求和响应体）
func DetailedAPILoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// 读取请求体
		var requestBody []byte
		if c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))
		}

		// 创建响应体捕获器
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		// 处理请求
		c.Next()

		// 计算处理时间
		latency := time.Since(start)

		// 准备日志字段
		fields := []zap.Field{
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.String("protocol", c.Request.Proto),
			zap.Int("status_code", c.Writer.Status()),
			zap.Duration("latency", latency),
			zap.String("client_ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
			zap.Int("request_size", len(requestBody)),
			zap.Int("response_size", blw.body.Len()),
		}

		// 添加请求体（如果是JSON且不太大）
		if len(requestBody) > 0 && len(requestBody) < 1024 && isJSON(requestBody) {
			fields = append(fields, zap.String("request_body", string(requestBody)))
		}

		// 添加响应体（如果是JSON且不太大）
		responseBody := blw.body.String()
		if len(responseBody) < 1024 && isJSON([]byte(responseBody)) {
			fields = append(fields, zap.String("response_body", responseBody))
		}

		// 添加用户ID（如果存在）
		if userID, exists := c.Get("user_id"); exists {
			fields = append(fields, zap.Any("user_id", userID))
		}

		// 添加错误信息
		if len(c.Errors) > 0 {
			fields = append(fields, zap.String("errors", c.Errors.String()))
		}

		// 根据状态码和处理时间确定日志级别
		var level zapcore.Level
		var message string
		if c.Writer.Status() >= 500 {
			level = zapcore.ErrorLevel
			message = "API Error"
		} else if c.Writer.Status() >= 400 {
			level = zapcore.WarnLevel
			message = "API Warning"
		} else if latency > 5*time.Second {
			level = zapcore.WarnLevel
			message = "API Slow Response"
		} else {
			level = zapcore.InfoLevel
			message = "API Request"
		}

		logger.LogAPI(level, message, fields...)
	}
}

// bodyLogWriter 响应体捕获器
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// isJSON 检查数据是否为JSON格式
func isJSON(data []byte) bool {
	var js json.RawMessage
	return json.Unmarshal(data, &js) == nil
}

// TaskLoggerMiddleware 任务相关的日志中间件
func TaskLoggerMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// start := time.Now()

		// 处理请求
		c.Next()

		// // 如果是任务相关的API，记录到任务日志
		// if isTaskRelatedPath(c.Request.URL.Path) {
		// 	latency := time.Since(start)

		// 	fields := []zap.Field{
		// 		zap.String("method", c.Request.Method),
		// 		zap.String("path", c.Request.URL.Path),
		// 		zap.Int("status_code", c.Writer.Status()),
		// 		zap.Duration("latency", latency),
		// 		zap.String("client_ip", c.ClientIP()),
		// 	}

		// 	// 添加用户ID
		// 	if userID, exists := c.Get("user_id"); exists {
		// 		fields = append(fields, zap.Any("user_id", userID))
		// 	}

		// 	// 添加任务相关参数
		// 	if taskID := c.Param("id"); taskID != "" {
		// 		fields = append(fields, zap.String("task_id", taskID))
		// 	}
		// 	if accountID := c.Query("account_id"); accountID != "" {
		// 		fields = append(fields, zap.String("account_id", accountID))
		// 	}

		// 	var level zapcore.Level
		// 	if c.Writer.Status() >= 400 {
		// 		level = zapcore.ErrorLevel
		// 	} else {
		// 		level = zapcore.InfoLevel
		// 	}

		// 	logger.LogTask(level, "Task API Request", fields...)
		// }
	}
}

// isTaskRelatedPath 检查是否为任务相关的路径
func isTaskRelatedPath(path string) bool {
	taskPaths := []string{
		"/api/v1/tasks",
		"/api/v1/accounts",
		"/api/v1/modules",
	}

	for _, taskPath := range taskPaths {
		if len(path) >= len(taskPath) && path[:len(taskPath)] == taskPath {
			return true
		}
	}

	return false
}
