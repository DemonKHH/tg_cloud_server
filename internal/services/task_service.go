package services

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gorm.io/gorm"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// TaskSchedulerInterface 任务调度器接口
type TaskSchedulerInterface interface {
	SubmitTask(task *models.Task) error
}

// TaskService 任务管理服务
type TaskService struct {
	taskRepo    repository.TaskRepository
	accountRepo repository.AccountRepository
	scheduler   TaskSchedulerInterface
	logger      *zap.Logger
}

// NewTaskService 创建任务管理服务
func NewTaskService(taskRepo repository.TaskRepository, accountRepo repository.AccountRepository) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		accountRepo: accountRepo,
		scheduler:   nil, // 稍后通过 SetTaskScheduler 设置
		logger:      logger.Get().Named("task_service"),
	}
}

// SetTaskScheduler 设置任务调度器
func (s *TaskService) SetTaskScheduler(scheduler TaskSchedulerInterface) {
	s.scheduler = scheduler
	s.logger.Info("Task scheduler has been set")

	// 启动时加载所有待处理任务
	go s.loadPendingTasks()
}

// loadPendingTasks 加载并提交所有待处理的任务
func (s *TaskService) loadPendingTasks() {
	s.logger.Info("Loading pending tasks...")

	// 查找所有pending状态的任务
	pendingTasks, err := s.taskRepo.GetTasksByStatus(models.TaskStatusPending)
	if err != nil {
		s.logger.Error("Failed to load pending tasks", zap.Error(err))
		return
	}

	submitted := 0
	failed := 0

	for _, task := range pendingTasks {
		if err := s.scheduler.SubmitTask(task); err != nil {
			failed++
			logger.LogTask(zapcore.ErrorLevel, "Failed to submit pending task to scheduler",
				zap.Uint64("task_id", task.ID),
				zap.String("task_type", string(task.TaskType)),
				zap.Error(err))
		} else {
			submitted++
			logger.LogTask(zapcore.InfoLevel, "Pending task submitted to scheduler",
				zap.Uint64("task_id", task.ID),
				zap.String("task_type", string(task.TaskType)))
		}
	}

	s.logger.Info("Finished loading pending tasks",
		zap.Int("total", len(pendingTasks)),
		zap.Int("submitted", submitted),
		zap.Int("failed", failed))
}

// TaskFilter 任务过滤器
type TaskFilter struct {
	UserID    uint64
	AccountID uint64
	TaskType  string
	Status    string
	Page      int
	Limit     int
}

// CreateTask 创建任务
func (s *TaskService) CreateTask(userID uint64, req *models.CreateTaskRequest) (*models.Task, error) {
	// 验证请求
	if err := req.Validate(); err != nil {
		return nil, err
	}

	// 验证所有账号是否属于用户且可用
	for _, accountID := range req.AccountIDs {
		account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
		if err != nil {
			return nil, fmt.Errorf("account %d not found or not owned by user: %w", accountID, err)
		}

		// 检查账号状态
		if !account.IsAvailable() {
			return nil, fmt.Errorf("account %d is not available, status: %s", accountID, account.Status)
		}
	}

	// 确保 Config 不为 nil，如果是 nil 则初始化为空 map
	config := req.Config
	if config == nil {
		config = make(models.TaskConfig)
	}

	task := &models.Task{
		UserID:   userID,
		TaskType: req.TaskType,
		Status:   models.TaskStatusPending,
		Priority: req.Priority,
		Config:   config,
		Result:   make(models.TaskResult), // 确保 Result 也不为 nil
	}

	// 设置账号ID列表
	task.SetAccountIDList(req.AccountIDs)

	if req.ScheduleAt != nil {
		task.ScheduledAt = req.ScheduleAt
	}

	if err := s.taskRepo.Create(task); err != nil {
		// 记录错误日志到任务日志和错误日志
		logger.LogTask(zapcore.ErrorLevel, "Failed to create task",
			zap.Uint64("user_id", userID),
			zap.Any("account_ids", req.AccountIDs),
			zap.String("task_type", string(req.TaskType)),
			zap.Int("priority", req.Priority),
			zap.Any("config", req.Config),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	// 记录成功创建任务的日志
	logger.LogTask(zapcore.InfoLevel, "Task created successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", task.ID),
		zap.String("task_type", string(task.TaskType)),
		zap.Any("account_ids", req.AccountIDs),
		zap.Int("account_count", len(req.AccountIDs)),
		zap.Int("priority", task.Priority),
		zap.Time("created_at", task.CreatedAt))

	// 根据auto_start参数决定是否自动提交任务执行
	if req.AutoStart && s.scheduler != nil {
		if err := s.scheduler.SubmitTask(task); err != nil {
			logger.LogTask(zapcore.ErrorLevel, "Failed to submit task to scheduler",
				zap.Uint64("task_id", task.ID),
				zap.String("task_type", string(task.TaskType)),
				zap.Error(err))
			// 注意：这里不返回错误，因为任务已经创建成功，只是提交调度失败
			s.logger.Error("Failed to submit task to scheduler, task will remain pending",
				zap.Uint64("task_id", task.ID),
				zap.Error(err))
		} else {
			logger.LogTask(zapcore.InfoLevel, "Task auto-submitted to scheduler",
				zap.Uint64("task_id", task.ID),
				zap.String("task_type", string(task.TaskType)))
		}
	} else if req.AutoStart && s.scheduler == nil {
		logger.LogTask(zapcore.WarnLevel, "Auto-start requested but no scheduler available",
			zap.Uint64("task_id", task.ID),
			zap.String("task_type", string(task.TaskType)))
	} else {
		logger.LogTask(zapcore.InfoLevel, "Task created without auto-start",
			zap.Uint64("task_id", task.ID),
			zap.String("task_type", string(task.TaskType)))
	}

	return task, nil
}

// GetTasks 获取任务列表
func (s *TaskService) GetTasks(filter *TaskFilter) ([]*models.TaskSummary, int64, error) {
	offset := (filter.Page - 1) * filter.Limit

	// 构建过滤条件
	conditions := make(map[string]interface{})
	conditions["user_id"] = filter.UserID

	if filter.AccountID > 0 {
		conditions["account_id"] = filter.AccountID
	}
	if filter.TaskType != "" {
		conditions["task_type"] = filter.TaskType
	}
	if filter.Status != "" {
		conditions["status"] = filter.Status
	}

	return s.taskRepo.GetTaskSummaries(conditions, offset, filter.Limit)
}

// GetTask 获取任务详情
func (s *TaskService) GetTask(userID, taskID uint64) (*models.Task, error) {
	task, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}
	return task, nil
}

// UpdateTask 更新任务
func (s *TaskService) UpdateTask(userID, taskID uint64, req *models.UpdateTaskRequest) (*models.Task, error) {
	task, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	// 只允许更新某些字段
	if req.Priority > 0 {
		task.Priority = req.Priority
	}

	if req.ScheduleAt != nil {
		task.ScheduledAt = req.ScheduleAt
	}

	if req.Config != nil {
		task.Config = req.Config
	}

	if err := s.taskRepo.Update(task); err != nil {
		s.logger.Error("Failed to update task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update task: %w", err)
	}

	s.logger.Info("Task updated successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID))

	return task, nil
}

// CancelTask 取消任务
func (s *TaskService) CancelTask(userID, taskID uint64) error {
	task, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 只有待执行或排队中的任务可以取消
	if !task.CanCancel() {
		return fmt.Errorf("task cannot be cancelled, current status: %s", task.Status)
	}

	task.Status = models.TaskStatusCancelled
	completedTime := time.Now()
	task.CompletedAt = &completedTime

	if err := s.taskRepo.Update(task); err != nil {
		logger.LogTask(zapcore.ErrorLevel, "Failed to cancel task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.String("task_type", string(task.TaskType)),
			zap.String("original_status", string(task.Status)),
			zap.Error(err))
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	logger.LogTask(zapcore.InfoLevel, "Task cancelled successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID),
		zap.String("task_type", string(task.TaskType)),
		zap.Any("account_ids", task.GetAccountIDList()),
		zap.Time("cancelled_at", *task.CompletedAt))

	return nil
}

// DeleteTask 删除任务
func (s *TaskService) DeleteTask(userID, taskID uint64) error {
	// 删除任务（Repository会验证用户权限）
	err := s.taskRepo.DeleteByUserIDAndID(userID, taskID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrTaskNotFound
		}
		logger.LogTask(zapcore.ErrorLevel, "Failed to delete task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		return fmt.Errorf("failed to delete task: %w", err)
	}

	logger.LogTask(zapcore.InfoLevel, "Task deleted successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID))

	return nil
}

// GetTaskLogs 获取任务日志
func (s *TaskService) GetTaskLogs(userID, taskID uint64) ([]*models.TaskLog, error) {
	// 首先验证任务是否属于用户
	_, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	return s.taskRepo.GetTaskLogs(taskID)
}

// GetTaskStats 获取任务统计
func (s *TaskService) GetTaskStats(userID uint64, timeRange string) (*models.TaskStats, error) {
	var startTime time.Time
	now := time.Now()

	switch timeRange {
	case "today":
		startTime = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	case "week":
		startTime = now.AddDate(0, 0, -7)
	case "month":
		startTime = now.AddDate(0, -1, 0)
	default:
		startTime = time.Time{} // 所有时间
	}

	return s.taskRepo.GetTaskStatsByUserID(userID, startTime, now)
}

// RetryTask 重试失败的任务
func (s *TaskService) RetryTask(userID, taskID uint64) (*models.Task, error) {
	task, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return nil, ErrTaskNotFound
	}

	if task.Status != models.TaskStatusFailed {
		return nil, fmt.Errorf("only failed tasks can be retried, current status: %s", task.Status)
	}

	// 重置任务状态
	task.Status = models.TaskStatusPending
	task.StartedAt = nil
	task.CompletedAt = nil
	task.Result = nil

	if err := s.taskRepo.Update(task); err != nil {
		logger.LogTask(zapcore.ErrorLevel, "Failed to retry task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.String("task_type", string(task.TaskType)),
			zap.String("original_status", string(task.Status)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retry task: %w", err)
	}

	logger.LogTask(zapcore.InfoLevel, "Task retry scheduled",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID),
		zap.String("task_type", string(task.TaskType)),
		zap.Any("account_ids", task.GetAccountIDList()),
		zap.String("new_status", string(task.Status)))

	return task, nil
}

// StartTask 启动任务
func (s *TaskService) StartTask(userID, taskID uint64) error {
	task, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 检查任务状态是否可以启动
	if task.Status != models.TaskStatusPending && task.Status != models.TaskStatusPaused {
		return fmt.Errorf("task status %s cannot be started", task.Status)
	}

	// 提交给调度器
	if s.scheduler == nil {
		return fmt.Errorf("task scheduler not available")
	}

	if err := s.scheduler.SubmitTask(task); err != nil {
		logger.LogTask(zapcore.ErrorLevel, "Failed to start task",
			zap.Uint64("task_id", taskID),
			zap.String("task_type", string(task.TaskType)),
			zap.Error(err))
		return fmt.Errorf("failed to start task: %w", err)
	}

	logger.LogTask(zapcore.InfoLevel, "Task started manually",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID),
		zap.String("task_type", string(task.TaskType)))

	return nil
}

// PauseTask 暂停任务
func (s *TaskService) PauseTask(userID, taskID uint64) error {
	task, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 检查任务状态
	if task.Status != models.TaskStatusQueued && task.Status != models.TaskStatusRunning {
		return fmt.Errorf("task status %s cannot be paused", task.Status)
	}

	// 如果任务正在运行，只能等待其完成，但标记为paused状态
	task.Status = models.TaskStatusPaused
	if err := s.taskRepo.Update(task); err != nil {
		logger.LogTask(zapcore.ErrorLevel, "Failed to pause task",
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		return fmt.Errorf("failed to pause task: %w", err)
	}

	logger.LogTask(zapcore.InfoLevel, "Task paused",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID),
		zap.String("task_type", string(task.TaskType)))

	return nil
}

// StopTask 停止任务（取消）
func (s *TaskService) StopTask(userID, taskID uint64) error {
	// 停止任务实际上就是取消任务
	return s.CancelTask(userID, taskID)
}

// ResumeTask 恢复任务
func (s *TaskService) ResumeTask(userID, taskID uint64) error {
	task, err := s.taskRepo.GetByUserIDAndID(userID, taskID)
	if err != nil {
		return ErrTaskNotFound
	}

	// 检查任务状态
	if task.Status != models.TaskStatusPaused {
		return fmt.Errorf("task status %s cannot be resumed", task.Status)
	}

	// 重新提交给调度器
	if s.scheduler == nil {
		return fmt.Errorf("task scheduler not available")
	}

	// 将状态改回pending
	task.Status = models.TaskStatusPending
	if err := s.taskRepo.Update(task); err != nil {
		return fmt.Errorf("failed to update task status: %w", err)
	}

	if err := s.scheduler.SubmitTask(task); err != nil {
		logger.LogTask(zapcore.ErrorLevel, "Failed to resume task",
			zap.Uint64("task_id", taskID),
			zap.String("task_type", string(task.TaskType)),
			zap.Error(err))
		return fmt.Errorf("failed to resume task: %w", err)
	}

	logger.LogTask(zapcore.InfoLevel, "Task resumed",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID),
		zap.String("task_type", string(task.TaskType)))

	return nil
}

// BatchControlTasks 批量控制任务
func (s *TaskService) BatchControlTasks(userID uint64, req *models.BatchTaskControlRequest) (int, error) {
	successCount := 0
	var errors []string

	for _, taskID := range req.TaskIDs {
		var err error

		switch req.Action {
		case "start":
			err = s.StartTask(userID, taskID)
		case "pause":
			err = s.PauseTask(userID, taskID)
		case "stop":
			err = s.StopTask(userID, taskID)
		case "resume":
			err = s.ResumeTask(userID, taskID)
		case "cancel":
			err = s.CancelTask(userID, taskID)
		default:
			err = fmt.Errorf("unsupported action: %s", req.Action)
		}

		if err != nil {
			errors = append(errors, fmt.Sprintf("Task %d: %s", taskID, err.Error()))
			logger.LogTask(zapcore.ErrorLevel, "Batch task control failed",
				zap.Uint64("task_id", taskID),
				zap.String("action", req.Action),
				zap.Error(err))
		} else {
			successCount++
			logger.LogTask(zapcore.InfoLevel, "Batch task control succeeded",
				zap.Uint64("task_id", taskID),
				zap.String("action", req.Action))
		}
	}

	if len(errors) > 0 {
		s.logger.Warn("Some tasks in batch control failed",
			zap.Int("success_count", successCount),
			zap.Int("failed_count", len(errors)),
			zap.Strings("errors", errors))
	}

	return successCount, nil
}

// BatchCancelTasks 批量取消任务
func (s *TaskService) BatchCancelTasks(userID uint64, taskIDs []uint64) (int, error) {
	successCount := 0

	for _, taskID := range taskIDs {
		if err := s.CancelTask(userID, taskID); err != nil {
			s.logger.Warn("Failed to cancel task in batch",
				zap.Uint64("user_id", userID),
				zap.Uint64("task_id", taskID),
				zap.Error(err))
			continue
		}
		successCount++
	}

	s.logger.Info("Batch cancel tasks completed",
		zap.Uint64("user_id", userID),
		zap.Int("total", len(taskIDs)),
		zap.Int("success", successCount))

	return successCount, nil
}

// BatchDeleteTasks 批量删除任务
func (s *TaskService) BatchDeleteTasks(userID uint64, taskIDs []uint64) (int, error) {
	successCount := 0

	for _, taskID := range taskIDs {
		if err := s.DeleteTask(userID, taskID); err != nil {
			s.logger.Warn("Failed to delete task in batch",
				zap.Uint64("user_id", userID),
				zap.Uint64("task_id", taskID),
				zap.Error(err))
			continue
		}
		successCount++
	}

	s.logger.Info("Batch delete tasks completed",
		zap.Uint64("user_id", userID),
		zap.Int("total", len(taskIDs)),
		zap.Int("success", successCount))

	return successCount, nil
}

// GetQueueInfo 获取队列信息
func (s *TaskService) GetQueueInfo(userID, accountID uint64) (*models.QueueInfo, error) {
	// 验证账号是否属于用户
	_, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return nil, fmt.Errorf("account not found or not owned by user: %w", err)
	}

	return s.taskRepo.GetQueueInfoByAccountID(accountID)
}

// CleanupCompletedTasks 清理已完成的任务
func (s *TaskService) CleanupCompletedTasks(userID uint64, olderThanDays int) (int64, error) {
	cutoffTime := time.Now().AddDate(0, 0, -olderThanDays)

	count, err := s.taskRepo.DeleteCompletedTasksBefore(userID, cutoffTime)
	if err != nil {
		s.logger.Error("Failed to cleanup completed tasks",
			zap.Uint64("user_id", userID),
			zap.Int("days", olderThanDays),
			zap.Error(err))
		return 0, fmt.Errorf("failed to cleanup completed tasks: %w", err)
	}

	s.logger.Info("Completed tasks cleaned up",
		zap.Uint64("user_id", userID),
		zap.Int64("count", count))

	return count, nil
}

// GetTaskAnalytics 获取任务分析数据
func (s *TaskService) GetTaskAnalytics(userID uint64, days int) (*models.TaskAnalytics, error) {
	// 简化实现 - 实际需要根据repository接口调整
	analytics := &models.TaskAnalytics{
		Period:      fmt.Sprintf("Last %d days", days),
		TotalTasks:  0,
		Completed:   0,
		Failed:      0,
		Cancelled:   0,
		Running:     0,
		Pending:     0,
		SuccessRate: 0,
		GeneratedAt: time.Now(),
	}

	s.logger.Info("Task analytics generated",
		zap.Uint64("user_id", userID),
		zap.Int("days", days))

	return analytics, nil
}

// RetryFailedTasks 重试失败的任务
func (s *TaskService) RetryFailedTasks(userID uint64, maxRetries int) (int, error) {
	// 简化实现 - 实际需要根据repository接口调整
	s.logger.Info("Failed tasks retry requested",
		zap.Uint64("user_id", userID),
		zap.Int("max_retries", maxRetries))

	// 这里应该实现实际的重试逻辑
	return 0, nil
}

// OptimizeTaskScheduling 优化任务调度
func (s *TaskService) OptimizeTaskScheduling(userID uint64) (*models.SchedulingOptimization, error) {
	// 简化实现 - 实际需要根据repository接口调整
	optimization := &models.SchedulingOptimization{
		UserID:          userID,
		TotalLoad:       0,
		TotalCapacity:   100,
		UtilizationRate: 0,
		Recommendations: []string{"暂无负载数据"},
		GeneratedAt:     time.Now(),
	}

	s.logger.Info("Task scheduling optimization generated",
		zap.Uint64("user_id", userID))

	return optimization, nil
}
