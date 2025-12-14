package middleware

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
)

// UserRateLimit 基于用户的限流中间件
func UserRateLimit(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	log := logger.Get().Named("user_rate_limit")

	return func(c *gin.Context) {
		// 获取用户ID
		userIDInterface, exists := c.Get("user_id")
		if !exists {
			// 如果没有用户ID，使用IP限流
			clientIP := c.ClientIP()
			key := fmt.Sprintf("rate_limit:ip:%s", clientIP)
			handleRateLimit(c, redisClient, key, limit, window, log)
			return
		}

		userID, ok := userIDInterface.(uint64)
		if !ok {
			log.Warn("Invalid user_id type in context")
			c.Next()
			return
		}

		// 构建基于用户的Redis键
		key := fmt.Sprintf("rate_limit:user:%d", userID)

		handleRateLimit(c, redisClient, key, limit, window, log)
	}

}

// handleRateLimit 处理限流逻辑
func handleRateLimit(c *gin.Context, redisClient *redis.Client, key string, limit int, window time.Duration, log *zap.Logger) {
	ctx := context.Background()

	// 检查当前请求数
	current, err := redisClient.Get(ctx, key).Int()
	if err != nil && err != redis.Nil {
		log.Error("Failed to get rate limit from Redis",
			zap.String("key", key),
			zap.Error(err))
		// Redis出错时允许请求继续
		c.Next()
		return
	}

	// 检查是否超过限制
	if current >= limit {
		log.Warn("Rate limit exceeded",
			zap.String("key", key),
			zap.Int("current", current),
			zap.Int("limit", limit))

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", "0")
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))

		response.TooManyRequests(c, "请求过于频繁，请稍后重试", fmt.Sprintf("retry_after: %d", int(window.Seconds())))
		c.Abort()
		return
	}

	// 增加计数器
	pipe := redisClient.Pipeline()
	pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err = pipe.Exec(ctx)

	if err != nil {
		log.Error("Failed to update rate limit in Redis",
			zap.String("key", key),
			zap.Error(err))
	}

	// 设置响应头
	remaining := limit - current - 1
	if remaining < 0 {
		remaining = 0
	}

	c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
	c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
	c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))

	c.Next()
}

// APIEndpointRateLimit 接口级别的限流中间件（不同接口可以有不同的限流策略）
func APIEndpointRateLimit(redisClient *redis.Client, endpointLimitMap map[string]EndpointLimit) gin.HandlerFunc {
	log := logger.Get().Named("endpoint_rate_limit")

	return func(c *gin.Context) {
		path := c.FullPath()
		method := c.Request.Method
		endpointKey := fmt.Sprintf("%s:%s", method, path)

		// 查找匹配的限流配置
		var limitConfig *EndpointLimit
		for key, config := range endpointLimitMap {
			if key == endpointKey || matchesPattern(key, endpointKey) {
				limitConfig = &config
				break
			}
		}

		// 如果没有找到匹配的配置，跳过限流
		if limitConfig == nil {
			c.Next()
			return
		}

		// 构建限流键（基于用户或IP）
		var rateLimitKey string
		if userIDInterface, exists := c.Get("user_id"); exists {
			if userID, ok := userIDInterface.(uint64); ok {
				rateLimitKey = fmt.Sprintf("rate_limit:endpoint:%s:user:%d", endpointKey, userID)
			}
		}

		if rateLimitKey == "" {
			// 如果没有用户ID，使用IP
			clientIP := c.ClientIP()
			rateLimitKey = fmt.Sprintf("rate_limit:endpoint:%s:ip:%s", endpointKey, clientIP)
		}

		handleRateLimit(c, redisClient, rateLimitKey, limitConfig.Limit, limitConfig.Window, log)
	}
}

// EndpointLimit 接口限流配置
type EndpointLimit struct {
	Limit  int           // 请求限制数
	Window time.Duration // 时间窗口
}

// matchesPattern 简单的模式匹配（支持通配符）
func matchesPattern(pattern, target string) bool {
	// 简单实现，可以根据需要扩展
	return pattern == target
}
