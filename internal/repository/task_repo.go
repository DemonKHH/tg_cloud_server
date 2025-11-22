package repository

import (
	"fmt"
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
	UpdateStatus(taskID uint64, status models.TaskStatus) error
	UpdateTask(taskID uint64, updates map[string]interface{}) error
	Delete(id uint64) error

	// 任务查询
	GetTaskSummaries(conditions map[string]interface{}, offset, limit int) ([]*models.TaskSummary, int64, error)
	GetPendingTasks(limit int) ([]*models.Task, error)
	GetTasksByStatus(status models.TaskStatus) ([]*models.Task, error)
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
	DeleteByUserIDAndID(userID, taskID uint64) error
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

// UpdateStatus 更新任务状态
func (r *taskRepository) UpdateStatus(taskID uint64, status models.TaskStatus) error {
	return r.db.Model(&models.Task{}).Where("id = ?", taskID).Update("status", status).Error
}

// UpdateTask 更新任务字段
func (r *taskRepository) UpdateTask(taskID uint64, updates map[string]interface{}) error {
	return r.db.Model(&models.Task{}).Where("id = ?", taskID).Updates(updates).Error
}

// Delete 删除任务
func (r *taskRepository) Delete(id uint64) error {
	return r.db.Delete(&models.Task{}, id).Error
}

// DeleteByUserIDAndID 根据用户ID和任务ID删除任务（安全删除）
func (r *taskRepository) DeleteByUserIDAndID(userID, taskID uint64) error {
	result := r.db.Where("user_id = ? AND id = ?", userID, taskID).Delete(&models.Task{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

// GetTaskSummaries 获取任务摘要列表
func (r *taskRepository) GetTaskSummaries(conditions map[string]interface{}, offset, limit int) ([]*models.TaskSummary, int64, error) {
	var tasks []*models.TaskSummary
	var total int64

	query := r.db.Model(&models.Task{}).
		Select(`tasks.id, tasks.task_type, tasks.status, tasks.account_id, 
		        tg_accounts.phone as account_phone, tasks.priority, 
		        tasks.created_at, tasks.started_at, tasks.completed_at`).
		Joins("LEFT JOIN tg_accounts ON tasks.account_id = tg_accounts.id").
		Where(conditions)

	// 获取总数
	countQuery := r.db.Model(&models.Task{}).Where(conditions)
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据并计算持续时间
	type TaskWithDuration struct {
		models.TaskSummary
		StartedAtRaw   *time.Time `gorm:"column:started_at"`
		CompletedAtRaw *time.Time `gorm:"column:completed_at"`
	}

	var rawTasks []TaskWithDuration
	err := query.Offset(offset).Limit(limit).
		Order("tasks.created_at DESC").
		Scan(&rawTasks).Error

	if err != nil {
		return nil, 0, err
	}

	// 转换并计算持续时间
	for _, rawTask := range rawTasks {
		task := rawTask.TaskSummary
		task.StartedAt = rawTask.StartedAtRaw
		task.CompletedAt = rawTask.CompletedAtRaw

		// 计算持续时间
		if task.StartedAt != nil && task.CompletedAt != nil {
			duration := task.CompletedAt.Sub(*task.StartedAt)
			task.Duration = formatDuration(duration)
		} else if task.StartedAt != nil {
			duration := time.Since(*task.StartedAt)
			task.Duration = formatDuration(duration) + " (运行中)"
		}

		tasks = append(tasks, &task)
	}

	// 确保返回空数组而不是 nil
	if tasks == nil {
		tasks = []*models.TaskSummary{}
	}

	return tasks, total, nil
}

// formatDuration 格式化持续时间
func formatDuration(duration time.Duration) string {
	if duration < time.Minute {
		return fmt.Sprintf("%.1f秒", duration.Seconds())
	} else if duration < time.Hour {
		return fmt.Sprintf("%.1f分钟", duration.Minutes())
	} else {
		return fmt.Sprintf("%.1f小时", duration.Hours())
	}
}

// GetPendingTasks 获取待处理任务
func (r *taskRepository) GetPendingTasks(limit int) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.Where("status = ? AND (scheduled_at IS NULL OR scheduled_at <= ?)",
		models.TaskStatusPending, time.Now()).
		Order("priority DESC, created_at ASC").
		Limit(limit).
		Find(&tasks).Error
	if tasks == nil {
		tasks = []*models.Task{}
	}
	return tasks, err
}

// GetTasksByStatus 根据状态获取所有任务
func (r *taskRepository) GetTasksByStatus(status models.TaskStatus) ([]*models.Task, error) {
	var tasks []*models.Task
	err := r.db.Where("status = ?", status).
		Order("priority DESC, created_at ASC").
		Find(&tasks).Error
	if tasks == nil {
		tasks = []*models.Task{}
	}
	return tasks, err
}

// GetTasksByAccountID 根据账号ID获取任务（搜索 account_ids 字段）
func (r *taskRepository) GetTasksByAccountID(accountID uint64, statuses []string) ([]*models.Task, error) {
	var tasks []*models.Task
	// 搜索 account_ids 字段中包含该账号ID的任务
	// 使用 LIKE 查询，匹配 "accountID" 或 "accountID," 或 ",accountID," 或 ",accountID"
	accountIDStr := fmt.Sprintf("%d", accountID)
	query := r.db.Where(
		"account_ids = ? OR account_ids LIKE ? OR account_ids LIKE ? OR account_ids LIKE ?",
		accountIDStr,           // 只有一个账号
		accountIDStr+",%",      // 第一个账号
		"%,"+accountIDStr+",%", // 中间的账号
		"%,"+accountIDStr,      // 最后一个账号
	)

	if len(statuses) > 0 {
		query = query.Where("status IN ?", statuses)
	}

	err := query.Order("created_at DESC").Find(&tasks).Error
	if tasks == nil {
		tasks = []*models.Task{}
	}
	return tasks, err
}

// GetTaskLogs 获取任务日志
func (r *taskRepository) GetTaskLogs(taskID uint64) ([]*models.TaskLog, error) {
	var logs []*models.TaskLog
	err := r.db.Where("task_id = ?", taskID).
		Order("created_at ASC").
		Find(&logs).Error
	if logs == nil {
		logs = []*models.TaskLog{}
	}
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

// GetQueueInfoByAccountID 获取账号队列信息（搜索包含该账号的任务）
func (r *taskRepository) GetQueueInfoByAccountID(accountID uint64) (*models.QueueInfo, error) {
	var info models.QueueInfo

	// 构建账号ID搜索条件
	accountIDStr := fmt.Sprintf("%d", accountID)
	accountCondition := "account_ids = ? OR account_ids LIKE ? OR account_ids LIKE ? OR account_ids LIKE ?"
	accountParams := []interface{}{
		accountIDStr,
		accountIDStr + ",%",
		"%," + accountIDStr + ",%",
		"%," + accountIDStr,
	}

	// 待处理任务数
	r.db.Model(&models.Task{}).
		Where(accountCondition, accountParams...).
		Where("status = ?", models.TaskStatusPending).
		Count(&info.PendingTasks)

	// 运行中任务数
	r.db.Model(&models.Task{}).
		Where(accountCondition, accountParams...).
		Where("status = ?", models.TaskStatusRunning).
		Count(&info.RunningTasks)

	// 预计等待时间（基于平均执行时间估算）
	var avgDuration float64
	r.db.Model(&models.Task{}).
		Select("AVG(TIMESTAMPDIFF(SECOND, started_at, completed_at))").
		Where(accountCondition, accountParams...).
		Where("status = ? AND started_at IS NOT NULL AND completed_at IS NOT NULL", models.TaskStatusCompleted).
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
