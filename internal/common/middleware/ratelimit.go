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

// RateLimit 限流中间件
func RateLimit(redisClient *redis.Client) gin.HandlerFunc {
	log := logger.Get().Named("rate_limit")

	return func(c *gin.Context) {
		// 获取客户端IP
		clientIP := c.ClientIP()

		// 构建Redis键
		key := fmt.Sprintf("rate_limit:%s", clientIP)

		// 检查当前请求数
		ctx := context.Background()
		current, err := redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			log.Error("Failed to get rate limit from Redis",
				zap.String("client_ip", clientIP),
				zap.Error(err))
			// 如果Redis出错，允许请求继续，但记录错误
			c.Next()
			return
		}

		// 设置限制（每分钟100个请求）
		limit := 100
		window := 60 * time.Second

		if current >= limit {
			log.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP),
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
				zap.String("client_ip", clientIP),
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
}

// RateLimitWithCustom 自定义限流中间件
func RateLimitWithCustom(redisClient *redis.Client, limit int, window time.Duration) gin.HandlerFunc {
	log := logger.Get().Named("rate_limit")

	return func(c *gin.Context) {
		clientIP := c.ClientIP()
		key := fmt.Sprintf("rate_limit:%s", clientIP)

		ctx := context.Background()
		current, err := redisClient.Get(ctx, key).Int()
		if err != nil && err != redis.Nil {
			log.Error("Failed to get rate limit from Redis",
				zap.String("client_ip", clientIP),
				zap.Error(err))
			c.Next()
			return
		}

		if current >= limit {
			log.Warn("Rate limit exceeded",
				zap.String("client_ip", clientIP),
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
				zap.String("client_ip", clientIP),
				zap.Error(err))
		}

		remaining := limit - current - 1
		if remaining < 0 {
			remaining = 0
		}

		c.Header("X-RateLimit-Limit", strconv.Itoa(limit))
		c.Header("X-RateLimit-Remaining", strconv.Itoa(remaining))
		c.Header("X-RateLimit-Reset", strconv.FormatInt(time.Now().Add(window).Unix(), 10))

		c.Next()
	}
}
