package repository

import (
	"time"

	"gorm.io/gorm"

	"tg_cloud_server/internal/models"
)

// TaskRepository 任务仓库接口
type TaskRepository interface {
	Create(task *models.Task) error
	GetByID(id uint64) (*models.Task, error)
	GetByUserIDAndID(userID, taskID uint64) (*models.Task, error)
	Update(task *models.Task) error
	Delete(id uint64) error

	// 任务查询
	GetTaskSummaries(conditions map[string]interface{}, offset, limit int) ([]*models.TaskSummary, int64, error)
	GetPendingTasks(limit int) ([]*models.Task, error)
	GetTasksByAccountID(accountID uint64, statuses []string) ([]*models.Task, error)

	// 任务日志
	GetTaskLogs(taskID uint64) ([]*models.TaskLog, error)
	CreateTaskLog(log *models.TaskLog) error

	// 任务统计
	GetTaskStatsByUserID(userID uint64, startTime, endTime time.Time) (*models.TaskStats, error)
	GetQueueInfoByAccountID(accountID uint64) (*models.QueueInfo, error)

	// 批量操作
	UpdateTasksStatus(taskIDs []uint64, status string) error
	DeleteCompletedTasksBefore(userID uint64, cutoffTime time.Time) (int64, error)
}

// taskRepository GORM实现
type taskRepository struct {
	db *gorm.DB
}

// NewTaskRepository 创建任务仓库
func NewTaskRepository(db *gorm.DB) TaskRepository {
	return &taskRepository{db: db}
}

// Create 创建任务
func (r *taskRepository) Create(task *models.Task) error {
	return r.db.Create(task).Error
}

// GetByID 根据ID获取任务
func (r *taskRepository) GetByID(id uint64) (*models.Task, error) {
	var task models.Task
	err := r.db.Where("id = ?", id).First(&task).Error
	return &task, err
}

// GetByUserIDAndID 根据用户ID和任务ID获取任务
func (r *taskRepository) GetByUserIDAndID(userID, taskID uint64) (*models.Task, error) {
	var task models.Task
	err := r.db.Where("user_id = ? AND id = ?", userID, taskID).First(&task).Error
	return &task, err
}

// Update 更新任务
func (r *taskRepository) Update(task *models.Task) error {
	return r.db.Save(task).Error
}

// Delete 删除任务
func (r *taskRepository) Delete(id uint64) error {
	return r.db.Delete(&models.Task{}, id).Error
}

// GetTaskSummaries 获取任务摘要列表
func (r *taskRepository) GetTaskSummaries(conditions map[string]interface{}, offset, limit int) ([]*models.TaskSummary, int64, error) {
	var tasks []*models.TaskSummary
	var total int64

	query := r.db.Model(&models.Task{}).
		Select("id, user_id, account_id, task_type, status, priority, created_at, scheduled_at, started_at, completed_at").
		Where(conditions)

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&tasks).Error

	return tasks, total, err
}

// GetPendingTasks 获取待处理任务
func (r *taskRepository) GetPendingTasks(limit int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)",
		models.TaskStatusPending, time.Now()).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	return tasks, err
}

// GetTasksByAccountID 根据账号ID获取任务
func (r *taskRepository) GetTasksByAccountID(accountID uint64, statuses []string) ([]*models.Task, error) {
	var tasks []*models.Task
	query := r.db.Where("account_id = ?", accountID)

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	err := query.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// GetTaskLogs 获取任务日志
func (r *taskRepository) GetTaskLogs(taskID uint64) ([]*models.TaskLog, error) {
	var logs []*models.TaskLog
	err := r.db.Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&logs).Error
	return logs, err
}

// CreateTaskLog 创建任务日志
func (r *taskRepository) CreateTaskLog(log *models.TaskLog) error {
	return r.db.Create(log).Error
}

// GetTaskStatsByUserID 获取用户任务统计
func (r *taskRepository) GetTaskStatsByUserID(userID uint64, startTime, endTime time.Time) (*models.TaskStats, error) {
	var stats models.TaskStats

	query := r.db.Model(&models.Task{}).Where("user_id = ?", userID)

	if !startTime.IsZero() {
		query = query.Where("created_at >= ?", startTime)
	}
	if !endTime.IsZero() {
		query = query.Where("created_at <= ?", endTime)
	}

	// 总任务数
	query.Count(&stats.Total)

	// 各状态任务数
	var statusCounts []struct {
		Status string
		Count  int64
	}

	query.Select("status, COUNT(*) as count").
		Group("status").
		Find(&statusCounts)

	for _, sc := range statusCounts {
		switch sc.Status {
		case string(models.TaskStatusPending):
			stats.Pending = sc.Count
		case string(models.TaskStatusRunning):
			stats.Running = sc.Count
		case string(models.TaskStatusCompleted):
			stats.Completed = sc.Count
		case string(models.TaskStatusFailed):
			stats.Failed = sc.Count
		case string(models.TaskStatusCancelled):
			stats.Cancelled = sc.Count
		}
	}

	// 今日任务数
	today := time.Now().Truncate(24 * time.Hour)
	r.db.Model(&models.Task{}).
		Where("user_id = ? AND created_at >= ?", userID, today).
		Count(&stats.TodayTasks)

	return &stats, nil
}

// GetQueueInfoByAccountID 获取账号队列信息
func (r *taskRepository) GetQueueInfoByAccountID(accountID uint64) (*models.QueueInfo, error) {
	var info models.QueueInfo

	// 待处理任务数
	r.db.Model(&models.Task{}).
		Where("account_id = ? AND status = ?", accountID, models.TaskStatusPending).
		Count(&info.PendingTasks)

	// 运行中任务数
	r.db.Model(&models.Task{}).
		Where("account_id = ? AND status = ?", accountID, models.TaskStatusRunning).
		Count(&info.RunningTasks)

	// 预计等待时间（基于平均执行时间估算）
	var avgDuration float64
	r.db.Model(&models.Task{}).
		Select("AVG(TIMESTAMPDIFF(SECOND, started_at, completed_at))").
		Where("account_id = ? AND status = ? AND started_at IS NOT NULL AND completed_at IS NOT NULL",
			accountID, models.TaskStatusCompleted).
		Scan(&avgDuration)

	info.EstimatedWaitTime = int64(avgDuration * float64(info.PendingTasks))

	return &info, nil
}

// UpdateTasksStatus 批量更新任务状态
func (r *taskRepository) UpdateTasksStatus(taskIDs []uint64, status string) error {
	return r.db.Model(&models.Task{}).
		Where("id IN ?", taskIDs).
		Update("status", status).Error
}

// DeleteCompletedTasksBefore 删除指定时间之前的已完成任务
func (r *taskRepository) DeleteCompletedTasksBefore(userID uint64, cutoffTime time.Time) (int64, error) {
	result := r.db.Where("user_id = ? AND status IN ? AND completed_at < ?",
		userID,
		[]string{string(models.TaskStatusCompleted), string(models.TaskStatusFailed), string(models.TaskStatusCancelled)},
		cutoffTime).
		Delete(&models.Task{})

	return result.RowsAffected, result.Error
}
