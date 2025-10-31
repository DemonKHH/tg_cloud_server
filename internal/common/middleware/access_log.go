package middleware

import (
	"context"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
)

// AccessLogMiddleware 接口访问日志和统计中间件
func AccessLogMiddleware(redisClient *redis.Client) gin.HandlerFunc {
	log := logger.Get().Named("access_log")

	return func(c *gin.Context) {
		start := time.Now()

		// 处理请求
		c.Next()

		// 计算处理时间
		duration := time.Since(start)

		// 获取请求信息
		method := c.Request.Method
		path := c.FullPath()
		statusCode := c.Writer.Status()

		// 获取用户信息
		var userID uint64
		if userIDInterface, exists := c.Get("user_id"); exists {
			if id, ok := userIDInterface.(uint64); ok {
				userID = id
			}
		}

		// 记录访问日志
		log.Info("API access",
			zap.String("method", method),
			zap.String("path", path),
			zap.Int("status", statusCode),
			zap.Duration("duration", duration),
			zap.Uint64("user_id", userID),
			zap.String("ip", c.ClientIP()),
			zap.String("user_agent", c.Request.UserAgent()),
		)

		// 统计接口调用（异步，不阻塞请求）
		if redisClient != nil {
			go recordAPIStatistics(redisClient, method, path, statusCode, duration, userID)
		}
	}
}

// recordAPIStatistics 记录API统计信息到Redis
func recordAPIStatistics(redisClient *redis.Client, method, path string, statusCode int, duration time.Duration, userID uint64) {
	ctx := context.Background()
	now := time.Now()

	// 统计键
	statsKey := fmt.Sprintf("api:stats:%s:%s", method, path)
	hourlyKey := fmt.Sprintf("api:stats:hourly:%s:%s:%s", method, path, now.Format("2006-01-02-15"))
	dailyKey := fmt.Sprintf("api:stats:daily:%s:%s:%s", method, path, now.Format("2006-01-02"))

	// 使用Pipeline批量操作
	pipe := redisClient.Pipeline()

	// 总调用次数
	pipe.Incr(ctx, statsKey)

	// 每小时统计
	pipe.Incr(ctx, hourlyKey)
	pipe.Expire(ctx, hourlyKey, 2*time.Hour)

	// 每天统计
	pipe.Incr(ctx, dailyKey)
	pipe.Expire(ctx, dailyKey, 48*time.Hour)

	// 状态码统计
	if statusCode >= 200 && statusCode < 300 {
		pipe.Incr(ctx, fmt.Sprintf("%s:success", statsKey))
	} else if statusCode >= 400 {
		pipe.Incr(ctx, fmt.Sprintf("%s:errors", statsKey))
	}

	// 响应时间统计（使用ZSet存储）
	responseTimeKey := fmt.Sprintf("%s:response_time", statsKey)
	pipe.ZAdd(ctx, responseTimeKey, &redis.Z{
		Score:  float64(duration.Milliseconds()),
		Member: now.Unix(),
	})
	pipe.ZRemRangeByRank(ctx, responseTimeKey, 0, -1001) // 只保留最近1000条
	pipe.Expire(ctx, responseTimeKey, 24*time.Hour)

	// 用户调用统计（如果已登录）
	if userID > 0 {
		userStatsKey := fmt.Sprintf("api:stats:user:%d:%s:%s", userID, method, path)
		pipe.Incr(ctx, userStatsKey)
		pipe.Expire(ctx, userStatsKey, 7*24*time.Hour) // 保留7天
	}

	// 执行Pipeline
	_, err := pipe.Exec(ctx)
	if err != nil {
		logger.Get().Named("access_log").Warn("Failed to record API statistics",
			zap.Error(err),
			zap.String("path", path))
	}
}

// GetAPIStats 获取API统计信息（用于监控端点）
func GetAPIStats(redisClient *redis.Client, method, path string) (map[string]interface{}, error) {
	if redisClient == nil {
		return nil, fmt.Errorf("redis client not available")
	}

	ctx := context.Background()
	statsKey := fmt.Sprintf("api:stats:%s:%s", method, path)

	stats := make(map[string]interface{})

	// 总调用次数
	total, err := redisClient.Get(ctx, statsKey).Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	stats["total_calls"] = total

	// 成功次数
	success, err := redisClient.Get(ctx, fmt.Sprintf("%s:success", statsKey)).Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	stats["success_calls"] = success

	// 错误次数
	errors, err := redisClient.Get(ctx, fmt.Sprintf("%s:errors", statsKey)).Int64()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	stats["error_calls"] = errors

	// 平均响应时间（从ZSet计算）
	responseTimeKey := fmt.Sprintf("%s:response_time", statsKey)
	members, err := redisClient.ZRange(ctx, responseTimeKey, 0, -1).Result()
	if err == nil && len(members) > 0 {
		scores, err := redisClient.ZRangeWithScores(ctx, responseTimeKey, 0, -1).Result()
		if err == nil {
			var sum float64
			for _, score := range scores {
				sum += score.Score
			}
			stats["avg_response_time_ms"] = sum / float64(len(scores))
		}
	}

	return stats, nil
}
