package services

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

var (
	ErrTaskNotFound = errors.New("task not found")
)

// TaskService 任务管理服务
type TaskService struct {
	taskRepo    repository.TaskRepository
	accountRepo repository.AccountRepository
	logger      *zap.Logger
}

// NewTaskService 创建任务管理服务
func NewTaskService(taskRepo repository.TaskRepository, accountRepo repository.AccountRepository) *TaskService {
	return &TaskService{
		taskRepo:    taskRepo,
		accountRepo: accountRepo,
		logger:      logger.Get().Named("task_service"),
	}
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
	// 验证账号是否属于用户
	account, err := s.accountRepo.GetByUserIDAndID(userID, req.AccountID)
	if err != nil {
		return nil, fmt.Errorf("account not found or not owned by user: %w", err)
	}

	// 检查账号状态
	if !account.IsAvailable() {
		return nil, fmt.Errorf("account is not available, status: %s", account.Status)
	}

	task := &models.Task{
		UserID:    userID,
		AccountID: req.AccountID,
		TaskType:  req.TaskType,
		Status:    models.TaskStatusPending,
		Priority:  req.Priority,
		Config:    req.Config,
	}

	if req.ScheduleAt != nil {
		task.ScheduledAt = req.ScheduleAt
	}

	if err := s.taskRepo.Create(task); err != nil {
		s.logger.Error("Failed to create task",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", req.AccountID),
			zap.String("task_type", string(req.TaskType)),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	s.logger.Info("Task created successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", task.ID),
		zap.String("task_type", string(task.TaskType)))

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
		s.logger.Error("Failed to cancel task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		return fmt.Errorf("failed to cancel task: %w", err)
	}

	s.logger.Info("Task cancelled successfully",
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
		s.logger.Error("Failed to retry task",
			zap.Uint64("user_id", userID),
			zap.Uint64("task_id", taskID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to retry task: %w", err)
	}

	s.logger.Info("Task retry scheduled",
		zap.Uint64("user_id", userID),
		zap.Uint64("task_id", taskID))

	return task, nil
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

// 数据模型定义


// BatchCancelRequest 批量取消请求
type BatchCancelRequest struct {
	TaskIDs []uint64 `json:"task_ids" binding:"required"`
}

// CleanupRequest 清理请求
type CleanupRequest struct {
	OlderThanDays int `json:"older_than_days" binding:"required,min=1"`
}

