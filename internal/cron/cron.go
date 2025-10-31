package cron

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/config"
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
	config         *config.Config

	// 依赖服务
	taskService    *services.TaskService
	accountService *services.AccountService
	userRepo       repository.UserRepository
	taskRepo       repository.TaskRepository
	accountRepo    repository.AccountRepository

	// 连接池接口（可选，用于连接检查）
	connectionPool interface {
		GetConnectionStatus(accountID string) interface{ String() string }
		GetStats() map[string]interface{}
	}
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
		config:         config.Get(),
		taskService:    taskService,
		accountService: accountService,
		userRepo:       userRepo,
		taskRepo:       taskRepo,
		accountRepo:    accountRepo,
	}
}

// SetConnectionPool 设置连接池（可选）
func (s *CronService) SetConnectionPool(pool interface {
	GetConnectionStatus(accountID string) interface{ String() string }
	GetStats() map[string]interface{}
}) {
	s.connectionPool = pool
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
	start := time.Now()
	defer func() {
		s.logger.Info("Log cleanup completed",
			zap.Duration("duration", time.Since(start)))
	}()

	logConfig := s.config.Logging

	// 如果没有配置日志文件路径，跳过清理
	if logConfig.Filename == "" || logConfig.Output != "file" {
		s.logger.Debug("Log file cleanup skipped: no file output configured")
		return
	}

	// 获取日志目录
	logDir := filepath.Dir(logConfig.Filename)
	logBaseName := filepath.Base(logConfig.Filename)

	// 计算保留期（默认28天）
	retentionDays := logConfig.MaxAge
	if retentionDays == 0 {
		retentionDays = 28
	}
	cutoffTime := time.Now().AddDate(0, 0, -retentionDays)

	s.logger.Info("Starting log cleanup",
		zap.String("log_dir", logDir),
		zap.Int("retention_days", retentionDays))

	// 清理过期的日志文件
	deletedCount := 0
	totalSize := int64(0)

	err := filepath.WalkDir(logDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// 只处理文件
		if d.IsDir() {
			return nil
		}

		// 检查是否是日志文件（匹配基础日志文件名模式）
		fileName := d.Name()
		if !strings.HasPrefix(fileName, logBaseName) {
			return nil
		}

		// 获取文件信息
		info, err := d.Info()
		if err != nil {
			s.logger.Warn("Failed to get file info",
				zap.String("file", path),
				zap.Error(err))
			return nil
		}

		// 检查文件修改时间
		if info.ModTime().Before(cutoffTime) {
			// 删除过期文件
			if err := os.Remove(path); err != nil {
				s.logger.Warn("Failed to delete expired log file",
					zap.String("file", path),
					zap.Error(err))
				return nil
			}

			deletedCount++
			totalSize += info.Size()
			s.logger.Debug("Deleted expired log file",
				zap.String("file", path),
				zap.Time("modified", info.ModTime()),
				zap.Int64("size", info.Size()))
		}

		return nil
	})

	if err != nil {
		s.logger.Error("Error during log cleanup",
			zap.String("log_dir", logDir),
			zap.Error(err))
		return
	}

	// 限制备份数量
	if logConfig.MaxBackups > 0 {
		if err := s.limitLogBackups(logDir, logBaseName, logConfig.MaxBackups); err != nil {
			s.logger.Warn("Failed to limit log backups",
				zap.Error(err))
		}
	}

	s.logger.Info("Log cleanup completed",
		zap.Int("deleted_files", deletedCount),
		zap.Int64("freed_space_mb", totalSize/(1024*1024)),
		zap.String("log_dir", logDir))
}

// limitLogBackups 限制日志备份文件数量
func (s *CronService) limitLogBackups(logDir, baseName string, maxBackups int) error {
	// 获取所有日志文件
	var logFiles []struct {
		path    string
		modTime time.Time
	}

	err := filepath.WalkDir(logDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		if !strings.HasPrefix(d.Name(), baseName) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		logFiles = append(logFiles, struct {
			path    string
			modTime time.Time
		}{
			path:    path,
			modTime: info.ModTime(),
		})

		return nil
	})

	if err != nil {
		return err
	}

	// 按修改时间排序（最新的在前）
	for i := 0; i < len(logFiles)-1; i++ {
		for j := i + 1; j < len(logFiles); j++ {
			if logFiles[i].modTime.Before(logFiles[j].modTime) {
				logFiles[i], logFiles[j] = logFiles[j], logFiles[i]
			}
		}
	}

	// 删除超出限制的文件
	if len(logFiles) > maxBackups {
		deleted := 0
		for i := maxBackups; i < len(logFiles); i++ {
			if err := os.Remove(logFiles[i].path); err != nil {
				s.logger.Warn("Failed to delete log backup",
					zap.String("file", logFiles[i].path),
					zap.Error(err))
				continue
			}
			deleted++
		}

		if deleted > 0 {
			s.logger.Info("Limited log backups",
				zap.Int("deleted", deleted),
				zap.Int("kept", maxBackups))
		}
	}

	return nil
}

// cleanupInvalidSessions 清理无效会话
func (s *CronService) cleanupInvalidSessions(ctx context.Context) {
	start := time.Now()
	defer func() {
		s.logger.Info("Session cleanup completed",
			zap.Duration("duration", time.Since(start)))
	}()

	// 获取所有账号
	accounts, err := s.accountRepo.GetAll()
	if err != nil {
		s.logger.Error("Failed to get accounts for session cleanup", zap.Error(err))
		return
	}

	cleanedCount := 0
	// 清理30天未使用的无效session
	cutoffTime := time.Now().AddDate(0, 0, -30)

	for _, account := range accounts {
		// 检查session数据是否有效
		// 如果账号长时间未使用且session为空或无效，清理它
		if account.SessionData == "" {
			// 如果账号30天未使用且没有session，可以标记为需要重新登录
			if account.LastUsedAt != nil && account.LastUsedAt.Before(cutoffTime) {
				// 清除可能的无效session状态
				cleanedCount++
				s.logger.Debug("Found account with empty session",
					zap.Uint64("account_id", account.ID),
					zap.String("phone", account.Phone))
			}
			continue
		}

		// 检查session数据是否过长（可能已损坏）
		// Telegram session数据通常不会超过几KB
		if len(account.SessionData) > 1024*100 { // 超过100KB可能是无效的
			s.logger.Warn("Found account with suspiciously large session data, clearing it",
				zap.Uint64("account_id", account.ID),
				zap.Int("session_size", len(account.SessionData)))

			// 清空无效的session数据
			if err := s.accountRepo.UpdateSessionData(account.ID, nil); err != nil {
				s.logger.Error("Failed to clear invalid session",
					zap.Uint64("account_id", account.ID),
					zap.Error(err))
			} else {
				cleanedCount++
			}
		}

		// 检查长时间未使用的账号的session（超过60天）
		if account.LastUsedAt != nil && account.LastUsedAt.Before(time.Now().AddDate(0, 0, -60)) {
			// 对于长时间未使用的账号，可以选择清理session让其重新登录
			// 但这里我们不主动清理，只记录
			s.logger.Debug("Found inactive account with old session",
				zap.Uint64("account_id", account.ID),
				zap.Time("last_used", *account.LastUsedAt))
		}
	}

	s.logger.Info("Session cleanup completed",
		zap.Int("checked_accounts", len(accounts)),
		zap.Int("cleaned_sessions", cleanedCount))
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
	start := time.Now()
	defer func() {
		s.logger.Debug("Account connections check completed",
			zap.Duration("duration", time.Since(start)))
	}()

	// 如果没有连接池，跳过检查
	if s.connectionPool == nil {
		s.logger.Debug("Connection pool not set, skipping connection check")
		return nil
	}

	// 获取连接池统计信息
	stats := s.connectionPool.GetStats()
	totalConnections, _ := stats["total_connections"].(int)
	activeConnections, _ := stats["active_connections"].(int)

	s.logger.Info("Checking account connections",
		zap.Int("total_connections", totalConnections),
		zap.Int("active_connections", activeConnections))

	// 获取所有活跃账号
	accounts, err := s.accountRepo.GetAll()
	if err != nil {
		s.logger.Error("Failed to get accounts for connection check", zap.Error(err))
		return err
	}

	checkedCount := 0
	disconnectedCount := 0
	errorCount := 0

	for _, account := range accounts {
		// 只检查正常状态的账号
		if account.Status != "normal" && account.Status != "warning" {
			continue
		}

		accountIDStr := fmt.Sprintf("%d", account.ID)

		// 检查连接状态
		status := s.connectionPool.GetConnectionStatus(accountIDStr)
		statusStr := status.String()

		switch statusStr {
		case "disconnected", "error":
			disconnectedCount++
			s.logger.Warn("Account connection issue detected",
				zap.Uint64("account_id", account.ID),
				zap.String("phone", account.Phone),
				zap.String("connection_status", statusStr))

			// 如果连接失败且账号状态正常，可以考虑标记为警告
			// 但这里我们只记录，不主动修改账号状态
		case "connected":
			// 更新账号最后使用时间
			now := time.Now()
			if account.LastUsedAt == nil || time.Since(*account.LastUsedAt) > 5*time.Minute {
				account.LastUsedAt = &now
				if err := s.accountRepo.Update(account); err != nil {
					s.logger.Warn("Failed to update account last used time",
						zap.Uint64("account_id", account.ID),
						zap.Error(err))
				}
			}
		}

		checkedCount++
	}

	s.logger.Info("Account connections check completed",
		zap.Int("checked_accounts", checkedCount),
		zap.Int("disconnected_accounts", disconnectedCount),
		zap.Int("error_accounts", errorCount),
		zap.Int("total_pool_connections", totalConnections),
		zap.Int("active_pool_connections", activeConnections))

	return nil
}
