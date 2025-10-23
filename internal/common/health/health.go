package health

import (
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"go.uber.org/zap"
	"gorm.io/gorm"

	"tg_cloud_server/internal/common/logger"
)

// HealthStatus 健康状态枚举
type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded  HealthStatus = "degraded"
)

// ComponentHealth 组件健康状态
type ComponentHealth struct {
	Name      string                 `json:"name"`
	Status    HealthStatus           `json:"status"`
	Message   string                 `json:"message,omitempty"`
	Duration  time.Duration          `json:"duration_ms"`
	Details   map[string]interface{} `json:"details,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
}

// OverallHealth 整体健康状态
type OverallHealth struct {
	Status     HealthStatus                `json:"status"`
	Version    string                      `json:"version"`
	Uptime     time.Duration               `json:"uptime_seconds"`
	Components map[string]*ComponentHealth `json:"components"`
	Timestamp  time.Time                   `json:"timestamp"`
}

// HealthChecker 健康检查器接口
type HealthChecker interface {
	Check(ctx context.Context) *ComponentHealth
	Name() string
}

// HealthService 健康检查服务
type HealthService struct {
	checkers  []HealthChecker
	logger    *zap.Logger
	startTime time.Time
	version   string
}

// NewHealthService 创建健康检查服务
func NewHealthService(version string) *HealthService {
	return &HealthService{
		checkers:  make([]HealthChecker, 0),
		logger:    logger.Get().Named("health_service"),
		startTime: time.Now(),
		version:   version,
	}
}

// AddChecker 添加健康检查器
func (s *HealthService) AddChecker(checker HealthChecker) {
	s.checkers = append(s.checkers, checker)
	s.logger.Info("Health checker added", zap.String("name", checker.Name()))
}

// CheckHealth 执行健康检查
func (s *HealthService) CheckHealth(ctx context.Context) *OverallHealth {
	start := time.Now()
	components := make(map[string]*ComponentHealth)
	overallStatus := StatusHealthy

	// 并发执行所有健康检查
	resultChan := make(chan *ComponentHealth, len(s.checkers))

	for _, checker := range s.checkers {
		go func(c HealthChecker) {
			checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
			defer cancel()

			result := c.Check(checkCtx)
			resultChan <- result
		}(checker)
	}

	// 收集结果
	for i := 0; i < len(s.checkers); i++ {
		result := <-resultChan
		components[result.Name] = result

		// 确定整体状态
		if result.Status == StatusUnhealthy {
			overallStatus = StatusUnhealthy
		} else if result.Status == StatusDegraded && overallStatus == StatusHealthy {
			overallStatus = StatusDegraded
		}
	}

	health := &OverallHealth{
		Status:     overallStatus,
		Version:    s.version,
		Uptime:     time.Since(s.startTime),
		Components: components,
		Timestamp:  time.Now(),
	}

	s.logger.Debug("Health check completed",
		zap.String("status", string(overallStatus)),
		zap.Duration("duration", time.Since(start)),
		zap.Int("components", len(components)))

	return health
}

// DatabaseHealthChecker 数据库健康检查器
type DatabaseHealthChecker struct {
	db   *gorm.DB
	name string
}

// NewDatabaseHealthChecker 创建数据库健康检查器
func NewDatabaseHealthChecker(db *gorm.DB) HealthChecker {
	return &DatabaseHealthChecker{
		db:   db,
		name: "database",
	}
}

// Name 返回检查器名称
func (c *DatabaseHealthChecker) Name() string {
	return c.name
}

// Check 执行数据库健康检查
func (c *DatabaseHealthChecker) Check(ctx context.Context) *ComponentHealth {
	start := time.Now()
	health := &ComponentHealth{
		Name:      c.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}

	// 获取底层SQL DB
	sqlDB, err := c.db.DB()
	if err != nil {
		health.Status = StatusUnhealthy
		health.Message = "Failed to get SQL DB instance"
		health.Duration = time.Since(start)
		return health
	}

	// 检查数据库连接
	if err := sqlDB.PingContext(ctx); err != nil {
		health.Status = StatusUnhealthy
		health.Message = "Database ping failed: " + err.Error()
		health.Duration = time.Since(start)
		return health
	}

	// 获取连接池统计信息
	stats := sqlDB.Stats()
	health.Details["open_connections"] = stats.OpenConnections
	health.Details["in_use"] = stats.InUse
	health.Details["idle"] = stats.Idle
	health.Details["max_open_connections"] = stats.MaxOpenConnections

	// 检查连接池状态
	if stats.OpenConnections > stats.MaxOpenConnections*8/10 {
		health.Status = StatusDegraded
		health.Message = "Database connection pool usage is high"
	} else {
		health.Status = StatusHealthy
		health.Message = "Database is healthy"
	}

	health.Duration = time.Since(start)
	return health
}

// RedisHealthChecker Redis健康检查器
type RedisHealthChecker struct {
	client *redis.Client
	name   string
}

// NewRedisHealthChecker 创建Redis健康检查器
func NewRedisHealthChecker(client *redis.Client) HealthChecker {
	return &RedisHealthChecker{
		client: client,
		name:   "redis",
	}
}

// Name 返回检查器名称
func (c *RedisHealthChecker) Name() string {
	return c.name
}

// Check 执行Redis健康检查
func (c *RedisHealthChecker) Check(ctx context.Context) *ComponentHealth {
	start := time.Now()
	health := &ComponentHealth{
		Name:      c.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}

	// 检查Redis连接
	if err := c.client.Ping(ctx).Err(); err != nil {
		health.Status = StatusUnhealthy
		health.Message = "Redis ping failed: " + err.Error()
		health.Duration = time.Since(start)
		return health
	}

	// 获取Redis信息
	info, err := c.client.Info(ctx, "memory", "stats").Result()
	if err != nil {
		health.Status = StatusDegraded
		health.Message = "Failed to get Redis info: " + err.Error()
	} else {
		health.Status = StatusHealthy
		health.Message = "Redis is healthy"
		health.Details["info"] = info
	}

	health.Duration = time.Since(start)
	return health
}

// SystemHealthChecker 系统健康检查器
type SystemHealthChecker struct {
	name string
}

// NewSystemHealthChecker 创建系统健康检查器
func NewSystemHealthChecker() HealthChecker {
	return &SystemHealthChecker{
		name: "system",
	}
}

// Name 返回检查器名称
func (c *SystemHealthChecker) Name() string {
	return c.name
}

// Check 执行系统健康检查
func (c *SystemHealthChecker) Check(ctx context.Context) *ComponentHealth {
	start := time.Now()
	health := &ComponentHealth{
		Name:      c.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
		Status:    StatusHealthy,
		Message:   "System is healthy",
	}

	// 这里可以添加系统资源检查逻辑
	// 例如：CPU使用率、内存使用率、磁盘空间等

	health.Duration = time.Since(start)
	return health
}

// CustomHealthChecker 自定义健康检查器
type CustomHealthChecker struct {
	name     string
	checkFn  func(ctx context.Context) error
	detailFn func(ctx context.Context) map[string]interface{}
}

// NewCustomHealthChecker 创建自定义健康检查器
func NewCustomHealthChecker(name string, checkFn func(ctx context.Context) error, detailFn func(ctx context.Context) map[string]interface{}) HealthChecker {
	return &CustomHealthChecker{
		name:     name,
		checkFn:  checkFn,
		detailFn: detailFn,
	}
}

// Name 返回检查器名称
func (c *CustomHealthChecker) Name() string {
	return c.name
}

// Check 执行自定义健康检查
func (c *CustomHealthChecker) Check(ctx context.Context) *ComponentHealth {
	start := time.Now()
	health := &ComponentHealth{
		Name:      c.name,
		Timestamp: start,
		Details:   make(map[string]interface{}),
	}

	// 执行检查函数
	if err := c.checkFn(ctx); err != nil {
		health.Status = StatusUnhealthy
		health.Message = err.Error()
	} else {
		health.Status = StatusHealthy
		health.Message = c.name + " is healthy"
	}

	// 获取详细信息
	if c.detailFn != nil {
		health.Details = c.detailFn(ctx)
	}

	health.Duration = time.Since(start)
	return health
}
