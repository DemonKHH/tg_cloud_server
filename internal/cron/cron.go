package cron

import (
	"context"
	"runtime"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/metrics"
	"tg_cloud_server/internal/repository"
	"tg_cloud_server/internal/services"
)

// CronService 定时任务服务
type CronService struct {
	cron           *cron.Cron
	logger         *zap.Logger
	metricsService *metrics.MetricsService

	// 依赖服务
	taskService    *services.TaskService
	accountService *services.AccountService
	userRepo       repository.UserRepository
	taskRepo       repository.TaskRepository
	accountRepo    repository.AccountRepository
}

// NewCronService 创建定时任务服务
func NewCronService(
	taskService *services.TaskService,
	accountService *services.AccountService,
	userRepo repository.UserRepository,
	taskRepo repository.TaskRepository,
	accountRepo repository.AccountRepository,
) *CronService {
	return &CronService{
		cron:           cron.New(cron.WithSeconds()),
		logger:         logger.Get().Named("cron_service"),
		metricsService: metrics.NewMetricsService(),
		taskService:    taskService,
		accountService: accountService,
		userRepo:       userRepo,
		taskRepo:       taskRepo,
		accountRepo:    accountRepo,
	}
}

// Start 启动定时任务
func (s *CronService) Start() error {
	s.logger.Info("Starting cron service")

	// 添加各种定时任务
	if err := s.addHealthCheckJob(); err != nil {
		return err
	}

	if err := s.addCleanupJob(); err != nil {
		return err
	}

	if err := s.addMetricsCollectionJob(); err != nil {
		return err
	}

	if err := s.addAccountStatusUpdateJob(); err != nil {
		return err
	}

	if err := s.addTaskTimeoutCheckJob(); err != nil {
		return err
	}

	// 启动cron调度器
	s.cron.Start()
	s.logger.Info("Cron service started successfully")

	return nil
}

// Stop 停止定时任务
func (s *CronService) Stop() {
	s.logger.Info("Stopping cron service")
	s.cron.Stop()
	s.logger.Info("Cron service stopped")
}

// addHealthCheckJob 添加健康检查任务
func (s *CronService) addHealthCheckJob() error {
	// 每5分钟执行一次健康检查
	_, err := s.cron.AddFunc("0 */5 * * * *", func() {
		ctx := context.Background()
		s.logger.Debug("Running health check job")

		// 检查系统健康状态
		s.performHealthCheck(ctx)
	})

	if err != nil {
		s.logger.Error("Failed to add health check job", zap.Error(err))
		return err
	}

	s.logger.Info("Health check job added successfully")
	return nil
}

// addCleanupJob 添加清理任务
func (s *CronService) addCleanupJob() error {
	// 每天凌晨2点执行清理任务
	_, err := s.cron.AddFunc("0 0 2 * * *", func() {
		ctx := context.Background()
		s.logger.Info("Running cleanup job")

		// 清理过期的已完成任务
		s.cleanupExpiredTasks(ctx)

		// 清理过期的日志
		s.cleanupExpiredLogs(ctx)

		// 清理无效的会话数据
		s.cleanupInvalidSessions(ctx)
	})

	if err != nil {
		s.logger.Error("Failed to add cleanup job", zap.Error(err))
		return err
	}

	s.logger.Info("Cleanup job added successfully")
	return nil
}

// addMetricsCollectionJob 添加指标收集任务
func (s *CronService) addMetricsCollectionJob() error {
	// 每分钟收集一次系统指标
	_, err := s.cron.AddFunc("0 * * * * *", func() {
		s.logger.Debug("Collecting system metrics")
		s.collectSystemMetrics()
	})

	if err != nil {
		s.logger.Error("Failed to add metrics collection job", zap.Error(err))
		return err
	}

	s.logger.Info("Metrics collection job added successfully")
	return nil
}

// addAccountStatusUpdateJob 添加账号状态更新任务
func (s *CronService) addAccountStatusUpdateJob() error {
	// 每10分钟更新一次账号状态
	_, err := s.cron.AddFunc("0 */10 * * * *", func() {
		ctx := context.Background()
		s.logger.Debug("Running account status update job")
		s.updateAccountStatuses(ctx)
	})

	if err != nil {
		s.logger.Error("Failed to add account status update job", zap.Error(err))
		return err
	}

	s.logger.Info("Account status update job added successfully")
	return nil
}

// addTaskTimeoutCheckJob 添加任务超时检查任务
func (s *CronService) addTaskTimeoutCheckJob() error {
	// 每2分钟检查一次任务超时
	_, err := s.cron.AddFunc("0 */2 * * * *", func() {
		ctx := context.Background()
		s.logger.Debug("Running task timeout check job")
		s.checkTaskTimeouts(ctx)
	})

	if err != nil {
		s.logger.Error("Failed to add task timeout check job", zap.Error(err))
		return err
	}

	s.logger.Info("Task timeout check job added successfully")
	return nil
}

// performHealthCheck 执行健康检查
func (s *CronService) performHealthCheck(ctx context.Context) {
	start := time.Now()
	defer func() {
		s.logger.Debug("Health check completed",
			zap.Duration("duration", time.Since(start)))
	}()

	// 检查数据库连接
	if err := s.checkDatabaseHealth(ctx); err != nil {
		s.logger.Error("Database health check failed", zap.Error(err))
	}

	// 检查Redis连接
	if err := s.checkRedisHealth(ctx); err != nil {
		s.logger.Error("Redis health check failed", zap.Error(err))
	}

	// 检查账号连接状态
	if err := s.checkAccountConnections(ctx); err != nil {
		s.logger.Error("Account connections check failed", zap.Error(err))
	}
}

// cleanupExpiredTasks 清理过期任务
func (s *CronService) cleanupExpiredTasks(ctx context.Context) {
	start := time.Now()
	defer func() {
		s.logger.Info("Task cleanup completed",
			zap.Duration("duration", time.Since(start)))
	}()

	// 清理30天前的已完成任务
	cutoffTime := time.Now().AddDate(0, 0, -30)

	// 获取所有用户
	users, err := s.userRepo.GetAll()
	if err != nil {
		s.logger.Error("Failed to get users for cleanup", zap.Error(err))
		return
	}

	totalDeleted := int64(0)
	for _, user := range users {
		count, err := s.taskRepo.DeleteCompletedTasksBefore(user.ID, cutoffTime)
		if err != nil {
			s.logger.Error("Failed to cleanup tasks for user",
				zap.Uint64("user_id", user.ID),
				zap.Error(err))
			continue
		}
		totalDeleted += count
	}

	s.logger.Info("Tasks cleaned up successfully",
		zap.Int64("total_deleted", totalDeleted))
}

// cleanupExpiredLogs 清理过期日志
func (s *CronService) cleanupExpiredLogs(ctx context.Context) {
	// 这里可以实现日志清理逻辑
	s.logger.Debug("Log cleanup not implemented yet")
}

// cleanupInvalidSessions 清理无效会话
func (s *CronService) cleanupInvalidSessions(ctx context.Context) {
	// 这里可以实现会话清理逻辑
	s.logger.Debug("Session cleanup not implemented yet")
}

// collectSystemMetrics 收集系统指标
func (s *CronService) collectSystemMetrics() {
	// 收集内存使用情况
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	memoryUsage := float64(m.Alloc)
	goroutines := float64(runtime.NumGoroutine())

	// 更新系统指标
	s.metricsService.UpdateSystemMetrics(memoryUsage, 0, goroutines) // CPU使用率需要其他方式获取

	s.logger.Debug("System metrics collected",
		zap.Float64("memory_usage", memoryUsage),
		zap.Float64("goroutines", goroutines))
}

// updateAccountStatuses 更新账号状态
func (s *CronService) updateAccountStatuses(ctx context.Context) {
	start := time.Now()
	defer func() {
		s.logger.Debug("Account status update completed",
			zap.Duration("duration", time.Since(start)))
	}()

	// 获取所有活跃账号
	accounts, err := s.accountRepo.GetAll()
	if err != nil {
		s.logger.Error("Failed to get accounts for status update", zap.Error(err))
		return
	}

	updatedCount := 0
	for _, account := range accounts {
		// 检查账号健康度
		if health, err := s.accountService.CheckAccountHealth(account.UserID, account.ID); err == nil {
			// 根据健康度更新状态
			if health.Score < 50 && account.Status != "warning" {
				account.Status = "warning"
				if err := s.accountRepo.Update(account); err != nil {
					s.logger.Error("Failed to update account status",
						zap.Uint64("account_id", account.ID),
						zap.Error(err))
				} else {
					updatedCount++
				}
			}
		}
	}

	s.logger.Info("Account statuses updated",
		zap.Int("updated_count", updatedCount),
		zap.Int("total_accounts", len(accounts)))
}

// checkTaskTimeouts 检查任务超时
func (s *CronService) checkTaskTimeouts(ctx context.Context) {
	start := time.Now()
	defer func() {
		s.logger.Debug("Task timeout check completed",
			zap.Duration("duration", time.Since(start)))
	}()

	// 获取所有运行中的任务
	runningTasks, err := s.taskRepo.GetTasksByAccountID(0, []string{"running"})
	if err != nil {
		s.logger.Error("Failed to get running tasks", zap.Error(err))
		return
	}

	timeoutCount := 0
	for _, task := range runningTasks {
		if task.StartedAt != nil {
			// 检查任务是否超时（超过30分钟）
			if time.Since(*task.StartedAt) > 30*time.Minute {
				// 标记任务为失败
				task.Status = "failed"
				completedTime := time.Now()
				task.CompletedAt = &completedTime

				if err := s.taskRepo.Update(task); err != nil {
					s.logger.Error("Failed to update timeout task",
						zap.Uint64("task_id", task.ID),
						zap.Error(err))
				} else {
					timeoutCount++
					s.logger.Warn("Task marked as timeout",
						zap.Uint64("task_id", task.ID),
						zap.String("task_type", string(task.TaskType)))
				}
			}
		}
	}

	if timeoutCount > 0 {
		s.logger.Info("Timeout tasks handled",
			zap.Int("timeout_count", timeoutCount))
	}
}

// checkDatabaseHealth 检查数据库健康状态
func (s *CronService) checkDatabaseHealth(ctx context.Context) error {
	// 执行简单的数据库查询检查连接
	_, err := s.userRepo.GetByID(1) // 尝试获取ID为1的用户，即使不存在也能检查连接
	if err != nil && err.Error() != "record not found" {
		return err
	}
	return nil
}

// checkRedisHealth 检查Redis健康状态
func (s *CronService) checkRedisHealth(ctx context.Context) error {
	// 这里需要Redis客户端实例，暂时跳过
	return nil
}

// checkAccountConnections 检查账号连接状态
func (s *CronService) checkAccountConnections(ctx context.Context) error {
	// 这里可以检查Telegram连接池中的连接状态
	s.logger.Debug("Account connections check not implemented yet")
	return nil
}
