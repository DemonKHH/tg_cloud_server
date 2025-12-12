package repository

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"

	"tg_cloud_server/internal/models"
)

// TaskRepository 任务仓库接口
type TaskRepository interface {
	Create(task *models.Task) error
	BatchCreate(tasks []*models.Task) error
	BatchDelete(taskIDs []uint64) error
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

	// 统计图表
	GetStatusDistribution(userID uint64, since time.Time) (map[string]int64, error)
	GetTypeDistribution(userID uint64, since time.Time) (map[string]int64, error)
	GetTasksPerHourTrend(userID uint64, hours int) ([]models.TimeSeriesPoint, error)
	GetSuccessRateTrend(userID uint64, hours int) ([]models.TimeSeriesPoint, error)
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
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先删除关联的日志
		if err := tx.Where("task_id = ?", id).Delete(&models.TaskLog{}).Error; err != nil {
			return err
		}
		// 再删除任务
		return tx.Delete(&models.Task{}, id).Error
	})
}

// DeleteByUserIDAndID 根据用户ID和任务ID删除任务（安全删除）
func (r *taskRepository) DeleteByUserIDAndID(userID, taskID uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先删除关联的日志
		if err := tx.Where("task_id = ?", taskID).Delete(&models.TaskLog{}).Error; err != nil {
			return err
		}

		// 再删除任务
		result := tx.Where("user_id = ? AND id = ?", userID, taskID).Delete(&models.Task{})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

// GetTaskSummaries 获取任务摘要列表
func (r *taskRepository) GetTaskSummaries(conditions map[string]interface{}, offset, limit int) ([]*models.TaskSummary, int64, error) {
	var tasks []*models.TaskSummary
	var total int64

	// 处理 account_id 条件（如果存在）
	var accountIDCondition string
	var accountIDParams []interface{}
	if accountID, ok := conditions["account_id"]; ok {
		// 将 account_id 条件转换为 account_ids 搜索
		accountIDStr := fmt.Sprintf("%d", accountID)
		accountIDCondition = "(tasks.account_ids = ? OR tasks.account_ids LIKE ? OR tasks.account_ids LIKE ? OR tasks.account_ids LIKE ?)"
		accountIDParams = []interface{}{
			accountIDStr,
			accountIDStr + ",%",
			"%," + accountIDStr + ",%",
			"%," + accountIDStr,
		}
		// 从 conditions 中移除 account_id
		delete(conditions, "account_id")
	}

	// 构建查询
	query := r.db.Model(&models.Task{}).
		Select(`tasks.id, tasks.task_type, tasks.status, tasks.account_ids, 
		        tasks.priority, tasks.created_at, tasks.started_at, tasks.completed_at`).
		Where(conditions)

	// 添加 account_id 条件（如果有）
	if accountIDCondition != "" {
		query = query.Where(accountIDCondition, accountIDParams...)
	}

	// 获取总数
	countQuery := r.db.Model(&models.Task{}).Where(conditions)
	if accountIDCondition != "" {
		countQuery = countQuery.Where(accountIDCondition, accountIDParams...)
	}
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据并计算持续时间
	type TaskWithDuration struct {
		models.TaskSummary
		AccountIDs     string     `gorm:"column:account_ids"`
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

		// 设置账号信息（显示账号数量）
		if rawTask.AccountIDs != "" {
			accountCount := len(strings.Split(rawTask.AccountIDs, ","))
			if accountCount == 1 {
				task.AccountPhone = "1个账号"
			} else {
				task.AccountPhone = fmt.Sprintf("%d个账号", accountCount)
			}
		}

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

// UpdateTasksStatus 批量更新任务状态（使用事务）
func (r *taskRepository) UpdateTasksStatus(taskIDs []uint64, status string) error {
	if len(taskIDs) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		return tx.Model(&models.Task{}).
			Where("id IN ?", taskIDs).
			Update("status", status).Error
	})
}

// BatchCreate 批量创建任务（使用事务）
func (r *taskRepository) BatchCreate(tasks []*models.Task) error {
	if len(tasks) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, task := range tasks {
			if err := tx.Create(task).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchDelete 批量删除任务（使用事务）
func (r *taskRepository) BatchDelete(taskIDs []uint64) error {
	if len(taskIDs) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先删除关联的日志
		if err := tx.Where("task_id IN ?", taskIDs).Delete(&models.TaskLog{}).Error; err != nil {
			return err
		}
		// 再删除任务
		return tx.Delete(&models.Task{}, taskIDs).Error
	})
}

// DeleteCompletedTasksBefore 删除指定时间之前的已完成任务
func (r *taskRepository) DeleteCompletedTasksBefore(userID uint64, cutoffTime time.Time) (int64, error) {
	var rowsAffected int64
	err := r.db.Transaction(func(tx *gorm.DB) error {
		// 1. 查找要删除的任务ID
		var taskIDs []uint64
		statuses := []string{
			string(models.TaskStatusCompleted),
			string(models.TaskStatusFailed),
			string(models.TaskStatusCancelled),
		}

		if err := tx.Model(&models.Task{}).
			Where("user_id = ? AND status IN ? AND completed_at < ?", userID, statuses, cutoffTime).
			Pluck("id", &taskIDs).Error; err != nil {
			return err
		}

		if len(taskIDs) == 0 {
			return nil
		}

		// 2. 删除关联的日志
		if err := tx.Where("task_id IN ?", taskIDs).Delete(&models.TaskLog{}).Error; err != nil {
			return err
		}

		// 3. 删除任务
		result := tx.Where("id IN ?", taskIDs).Delete(&models.Task{})
		if result.Error != nil {
			return result.Error
		}
		rowsAffected = result.RowsAffected
		return nil
	})

	return rowsAffected, err
}

// GetStatusDistribution 获取任务状态分布
func (r *taskRepository) GetStatusDistribution(userID uint64, since time.Time) (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}

	query := r.db.Model(&models.Task{}).
		Select("status, count(*) as count").
		Where("user_id = ?", userID)

	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}

	err := query.Group("status").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	distribution := make(map[string]int64)
	for _, result := range results {
		distribution[result.Status] = result.Count
	}

	return distribution, nil
}

// GetTypeDistribution 获取任务类型分布
func (r *taskRepository) GetTypeDistribution(userID uint64, since time.Time) (map[string]int64, error) {
	var results []struct {
		TaskType string
		Count    int64
	}

	query := r.db.Model(&models.Task{}).
		Select("task_type, count(*) as count").
		Where("user_id = ?", userID)

	if !since.IsZero() {
		query = query.Where("created_at >= ?", since)
	}

	err := query.Group("task_type").Scan(&results).Error
	if err != nil {
		return nil, err
	}

	distribution := make(map[string]int64)
	for _, result := range results {
		distribution[result.TaskType] = result.Count
	}

	return distribution, nil
}

// GetTasksPerHourTrend 获取每小时任务数趋势
func (r *taskRepository) GetTasksPerHourTrend(userID uint64, hours int) ([]models.TimeSeriesPoint, error) {
	startTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	// 获取时间段内的所有任务
	var tasks []models.Task
	err := r.db.Select("created_at").
		Where("user_id = ? AND created_at >= ?", userID, startTime).
		Order("created_at ASC").
		Find(&tasks).Error

	if err != nil {
		return nil, err
	}

	// 按小时聚合
	hourlyCounts := make(map[string]int64)
	for _, task := range tasks {
		// 格式化为 "2006-01-02 15:00"
		hourStr := task.CreatedAt.Format("2006-01-02 15:00")
		hourlyCounts[hourStr]++
	}

	// 构建结果
	var points []models.TimeSeriesPoint
	for i := 0; i < hours; i++ {
		// 从最早的时间开始
		t := startTime.Add(time.Duration(i) * time.Hour).Truncate(time.Hour)
		hourStr := t.Format("2006-01-02 15:00")

		points = append(points, models.TimeSeriesPoint{
			Timestamp: t,
			Value:     float64(hourlyCounts[hourStr]),
			Label:     t.Format("15:00"),
		})
	}
	// 确保包含当前小时
	now := time.Now().Truncate(time.Hour)
	if len(points) == 0 || !points[len(points)-1].Timestamp.Equal(now) {
		hourStr := now.Format("2006-01-02 15:00")
		points = append(points, models.TimeSeriesPoint{
			Timestamp: now,
			Value:     float64(hourlyCounts[hourStr]),
			Label:     "现在",
		})
	}

	return points, nil
}

// GetSuccessRateTrend 获取成功率趋势
func (r *taskRepository) GetSuccessRateTrend(userID uint64, hours int) ([]models.TimeSeriesPoint, error) {
	// 简化实现：每6小时一个点，计算该时间段内的成功率
	// 为了性能，我们只取最近24小时，分4个点，或者最近hours小时

	step := hours / 5 // 分5个点
	if step < 1 {
		step = 1
	}

	var points []models.TimeSeriesPoint
	now := time.Now()

	for i := 5; i >= 0; i-- {
		endTime := now.Add(-time.Duration(i*step) * time.Hour)
		startTime := endTime.Add(-time.Duration(step) * time.Hour)

		// 统计该时间段内的任务
		var total int64
		var success int64

		r.db.Model(&models.Task{}).
			Where("user_id = ? AND created_at >= ? AND created_at < ?", userID, startTime, endTime).
			Count(&total)

		if total > 0 {
			r.db.Model(&models.Task{}).
				Where("user_id = ? AND created_at >= ? AND created_at < ? AND status = ?",
					userID, startTime, endTime, models.TaskStatusCompleted).
				Count(&success)

			rate := float64(success) / float64(total) * 100
			points = append(points, models.TimeSeriesPoint{
				Timestamp: endTime,
				Value:     rate,
				Label:     endTime.Format("15:00"),
			})
		} else {
			// 如果没有任务，延续上一个点的成功率，或者设为100%（无失败）
			// 这里设为0或者跳过
			points = append(points, models.TimeSeriesPoint{
				Timestamp: endTime,
				Value:     0, // 或者 100?
				Label:     endTime.Format("15:00"),
			})
		}
	}

	return points, nil
}
