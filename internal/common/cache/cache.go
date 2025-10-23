package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
)

// Cache 缓存接口
type Cache interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string, dest interface{}) error
	Del(ctx context.Context, keys ...string) error
	Exists(ctx context.Context, key string) (bool, error)
	Expire(ctx context.Context, key string, expiration time.Duration) error
	Keys(ctx context.Context, pattern string) ([]string, error)
	FlushDB(ctx context.Context) error
}

// RedisCache Redis缓存实现
type RedisCache struct {
	client *redis.Client
	logger *zap.Logger
}

// NewRedisCache 创建Redis缓存实例
func NewRedisCache(client *redis.Client) Cache {
	return &RedisCache{
		client: client,
		logger: logger.Get().Named("cache"),
	}
}

// Set 设置缓存
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		c.logger.Error("Failed to marshal cache value",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	err = c.client.Set(ctx, key, data, expiration).Err()
	if err != nil {
		c.logger.Error("Failed to set cache",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	c.logger.Debug("Cache set successfully",
		zap.String("key", key),
		zap.Duration("expiration", expiration))
	return nil
}

// Get 获取缓存
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheNotFound
		}
		c.logger.Error("Failed to get cache",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		c.logger.Error("Failed to unmarshal cache value",
			zap.String("key", key),
			zap.Error(err))
		return err
	}

	c.logger.Debug("Cache get successfully", zap.String("key", key))
	return nil
}

// Del 删除缓存
func (c *RedisCache) Del(ctx context.Context, keys ...string) error {
	if len(keys) == 0 {
		return nil
	}

	err := c.client.Del(ctx, keys...).Err()
	if err != nil {
		c.logger.Error("Failed to delete cache",
			zap.Strings("keys", keys),
			zap.Error(err))
		return err
	}

	c.logger.Debug("Cache deleted successfully", zap.Strings("keys", keys))
	return nil
}

// Exists 检查缓存是否存在
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	result, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		c.logger.Error("Failed to check cache existence",
			zap.String("key", key),
			zap.Error(err))
		return false, err
	}
	return result > 0, nil
}

// Expire 设置过期时间
func (c *RedisCache) Expire(ctx context.Context, key string, expiration time.Duration) error {
	err := c.client.Expire(ctx, key, expiration).Err()
	if err != nil {
		c.logger.Error("Failed to set cache expiration",
			zap.String("key", key),
			zap.Duration("expiration", expiration),
			zap.Error(err))
		return err
	}
	return nil
}

// Keys 查找匹配的键
func (c *RedisCache) Keys(ctx context.Context, pattern string) ([]string, error) {
	result, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		c.logger.Error("Failed to get cache keys",
			zap.String("pattern", pattern),
			zap.Error(err))
		return nil, err
	}
	return result, nil
}

// FlushDB 清空数据库
func (c *RedisCache) FlushDB(ctx context.Context) error {
	err := c.client.FlushDB(ctx).Err()
	if err != nil {
		c.logger.Error("Failed to flush cache database", zap.Error(err))
		return err
	}
	c.logger.Info("Cache database flushed successfully")
	return nil
}

// 预定义错误
var (
	ErrCacheNotFound = fmt.Errorf("cache not found")
)

// CacheService 缓存服务
type CacheService struct {
	cache  Cache
	logger *zap.Logger
}

// NewCacheService 创建缓存服务
func NewCacheService(cache Cache) *CacheService {
	return &CacheService{
		cache:  cache,
		logger: logger.Get().Named("cache_service"),
	}
}

// 业务相关的缓存方法

// SetUserSession 设置用户会话缓存
func (s *CacheService) SetUserSession(ctx context.Context, userID uint64, sessionData interface{}) error {
	key := fmt.Sprintf("user:session:%d", userID)
	return s.cache.Set(ctx, key, sessionData, 24*time.Hour)
}

// GetUserSession 获取用户会话缓存
func (s *CacheService) GetUserSession(ctx context.Context, userID uint64, dest interface{}) error {
	key := fmt.Sprintf("user:session:%d", userID)
	return s.cache.Get(ctx, key, dest)
}

// DeleteUserSession 删除用户会话缓存
func (s *CacheService) DeleteUserSession(ctx context.Context, userID uint64) error {
	key := fmt.Sprintf("user:session:%d", userID)
	return s.cache.Del(ctx, key)
}

// SetAccountStatus 设置账号状态缓存
func (s *CacheService) SetAccountStatus(ctx context.Context, accountID uint64, status string) error {
	key := fmt.Sprintf("account:status:%d", accountID)
	return s.cache.Set(ctx, key, status, 30*time.Minute)
}

// GetAccountStatus 获取账号状态缓存
func (s *CacheService) GetAccountStatus(ctx context.Context, accountID uint64) (string, error) {
	key := fmt.Sprintf("account:status:%d", accountID)
	var status string
	err := s.cache.Get(ctx, key, &status)
	return status, err
}

// SetTaskQueue 设置任务队列缓存
func (s *CacheService) SetTaskQueue(ctx context.Context, accountID uint64, queueInfo interface{}) error {
	key := fmt.Sprintf("task:queue:%d", accountID)
	return s.cache.Set(ctx, key, queueInfo, 15*time.Minute)
}

// GetTaskQueue 获取任务队列缓存
func (s *CacheService) GetTaskQueue(ctx context.Context, accountID uint64, dest interface{}) error {
	key := fmt.Sprintf("task:queue:%d", accountID)
	return s.cache.Get(ctx, key, dest)
}

// SetProxyStats 设置代理统计缓存
func (s *CacheService) SetProxyStats(ctx context.Context, proxyID uint64, stats interface{}) error {
	key := fmt.Sprintf("proxy:stats:%d", proxyID)
	return s.cache.Set(ctx, key, stats, 10*time.Minute)
}

// GetProxyStats 获取代理统计缓存
func (s *CacheService) GetProxyStats(ctx context.Context, proxyID uint64, dest interface{}) error {
	key := fmt.Sprintf("proxy:stats:%d", proxyID)
	return s.cache.Get(ctx, key, dest)
}

// SetTelegramSession 设置Telegram会话缓存
func (s *CacheService) SetTelegramSession(ctx context.Context, accountID uint64, sessionData []byte) error {
	key := fmt.Sprintf("tg:session:%d", accountID)
	return s.cache.Set(ctx, key, sessionData, 7*24*time.Hour) // 7天过期
}

// GetTelegramSession 获取Telegram会话缓存
func (s *CacheService) GetTelegramSession(ctx context.Context, accountID uint64) ([]byte, error) {
	key := fmt.Sprintf("tg:session:%d", accountID)
	var sessionData []byte
	err := s.cache.Get(ctx, key, &sessionData)
	return sessionData, err
}

// IncrementRateLimit 增加限流计数
func (s *CacheService) IncrementRateLimit(ctx context.Context, identifier string, window time.Duration) (int64, error) {
	key := fmt.Sprintf("rate_limit:%s", identifier)

	// 使用Redis的INCR和EXPIRE命令实现滑动窗口限流
	pipe := s.cache.(*RedisCache).client.Pipeline()
	incrCmd := pipe.Incr(ctx, key)
	pipe.Expire(ctx, key, window)
	_, err := pipe.Exec(ctx)

	if err != nil {
		return 0, err
	}

	return incrCmd.Val(), nil
}

// ClearExpiredKeys 清理过期的缓存键
func (s *CacheService) ClearExpiredKeys(ctx context.Context, pattern string) error {
	keys, err := s.cache.Keys(ctx, pattern)
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return s.cache.Del(ctx, keys...)
	}

	return nil
}
