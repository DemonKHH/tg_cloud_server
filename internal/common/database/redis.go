package database

import (
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"

	"tg_cloud_server/internal/common/config"
)

// InitRedis 初始化Redis连接
func InitRedis(config *config.RedisConfig) (*redis.Client, error) {
	// 创建Redis客户端
	rdb := redis.NewClient(&redis.Options{
		Addr:     config.GetAddr(),
		Password: config.Password,
		DB:       config.Database,
		PoolSize: config.PoolSize,
	})

	// 测试连接
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return rdb, nil
}

// CloseRedis 关闭Redis连接
func CloseRedis(rdb *redis.Client) error {
	return rdb.Close()
}
