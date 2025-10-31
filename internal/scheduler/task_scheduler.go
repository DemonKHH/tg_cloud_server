package scheduler

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
	"tg_cloud_server/internal/telegram"
)

// TaskScheduler 任务调度器
type TaskScheduler struct {
	accountQueues  map[string]*TaskQueue        // 账号任务队列 (accountID -> queue)
	accountStatus  *sync.Map                    // 账号状态池
	connectionPool *telegram.ConnectionPool     // 连接池引用
	accountRepo    repository.AccountRepository // 账号仓库
	taskRepo       repository.TaskRepository    // 任务仓库
	// riskEngine 暂时移除风控引擎，后续实现
	logger *zap.Logger
	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// TaskQueue 任务队列
type TaskQueue struct {
	accountID  string
	tasks      []*models.Task
	mu         sync.Mutex
	processing bool
}

// NewTaskScheduler 创建新的任务调度器
func NewTaskScheduler(
	connectionPool *telegram.ConnectionPool,
	accountRepo repository.AccountRepository,
	taskRepo repository.TaskRepository,
) *TaskScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	ts := &TaskScheduler{
		accountQueues:  make(map[string]*TaskQueue),
		accountStatus:  &sync.Map{},
		connectionPool: connectionPool,
		accountRepo:    accountRepo,
		taskRepo:       taskRepo,
		logger:         logger.Get().Named("task_scheduler"),
		ctx:            ctx,
		cancel:         cancel,
	}

	// 启动调度循环
	go ts.schedulingLoop()

	return ts
}

// SubmitTask 提交任务到指定账号队列
func (ts *TaskScheduler) SubmitTask(task *models.Task) error {
	accountID := fmt.Sprintf("%d", task.AccountID)

	// 验证账号可用性
	if err := ts.ValidateAccount(accountID); err != nil {
		ts.logger.Warn("Account validation failed",
			zap.String("account_id", accountID),
			zap.Error(err))
		return fmt.Errorf("account validation failed: %w", err)
	}

	// 获取或创建队列
	queue := ts.getOrCreateQueue(accountID)

	// 添加任务到队列
	queue.mu.Lock()
	defer queue.mu.Unlock()

	task.Status = models.TaskStatusQueued
	queue.tasks = append(queue.tasks, task)

	// 更新数据库中的任务状态
	if err := ts.taskRepo.UpdateStatus(task.ID, models.TaskStatusQueued); err != nil {
		ts.logger.Error("Failed to update task status",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
	}

	ts.logger.Info("Task submitted to queue",
		zap.Uint64("task_id", task.ID),
		zap.String("account_id", accountID),
		zap.String("task_type", string(task.TaskType)),
		zap.Int("queue_size", len(queue.tasks)))

	return nil
}

// ValidateAccount 验证账号可用性
func (ts *TaskScheduler) ValidateAccount(accountID string) error {
	// 从缓存或数据库获取账号信息
	account, err := ts.getAccountInfo(accountID)
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	// 检查账号状态
	if !account.IsAvailable() {
		return fmt.Errorf("account is not available, status: %s", account.Status)
	}

	// 检查连接状态
	connectionStatus := ts.connectionPool.GetConnectionStatus(accountID)
	if connectionStatus == telegram.StatusError {
		return fmt.Errorf("account connection is in error state")
	}

	// 检查是否正在执行任务
	if ts.connectionPool.IsAccountBusy(accountID) {
		// 这不是错误，任务会排队等待
		ts.logger.Debug("Account is busy, task will be queued",
			zap.String("account_id", accountID))
	}

	return nil
}

// GetAccountAvailability 获取账号可用性信息
func (ts *TaskScheduler) GetAccountAvailability(accountID string) (*models.AccountAvailability, error) {
	account, err := ts.getAccountInfo(accountID)
	if err != nil {
		return nil, err
	}

	queueSize := ts.getQueueSize(accountID)
	isTaskRunning := ts.connectionPool.IsAccountBusy(accountID)
	connectionStatus := ts.connectionPool.GetConnectionStatus(accountID)

	availability := &models.AccountAvailability{
		AccountID:        account.ID,
		Status:           account.Status,
		HealthScore:      account.HealthScore,
		QueueSize:        queueSize,
		IsTaskRunning:    isTaskRunning,
		ConnectionStatus: models.ConnectionStatus(connectionStatus),
		LastUsed:         account.LastUsedAt,
		Warnings:         []string{},
		Errors:           []string{},
	}

	// 生成建议和警告
	ts.generateRecommendations(account, availability)

	return availability, nil
}

// ValidateAccountForTask 验证用户选择的账号是否可用于特定任务
func (ts *TaskScheduler) ValidateAccountForTask(accountID string, taskType models.TaskType) (*models.ValidationResult, error) {
	account, err := ts.getAccountInfo(accountID)
	if err != nil {
		return nil, err
	}

	result := &models.ValidationResult{
		AccountID:   account.ID,
		IsValid:     true,
		Warnings:    []string{},
		Errors:      []string{},
		QueueSize:   ts.getQueueSize(accountID),
		HealthScore: account.HealthScore,
	}

	// 账号状态检查
	if account.Status == models.AccountStatusDead {
		result.IsValid = false
		result.Errors = append(result.Errors, "账号已死亡，无法执行任务")
		return result, nil
	}

	if account.Status == models.AccountStatusCooling {
		result.IsValid = false
		result.Errors = append(result.Errors, "账号处于冷却期，暂时无法执行任务")
		return result, nil
	}

	// 健康度检查
	if account.HealthScore < 0.3 {
		result.Warnings = append(result.Warnings, "账号健康度较低，建议谨慎使用")
	}

	// 任务队列检查
	if result.QueueSize > 10 {
		result.Warnings = append(result.Warnings, fmt.Sprintf("账号任务队列较长 (%d个任务)", result.QueueSize))
	}

	// 连接状态检查
	connectionStatus := ts.connectionPool.GetConnectionStatus(accountID)
	if connectionStatus == telegram.StatusError {
		result.Warnings = append(result.Warnings, "账号连接异常，可能影响任务执行")
	}

	return result, nil
}

// getOrCreateQueue 获取或创建队列
func (ts *TaskScheduler) getOrCreateQueue(accountID string) *TaskQueue {
	ts.mu.Lock()
	defer ts.mu.Unlock()

	if queue, exists := ts.accountQueues[accountID]; exists {
		return queue
	}

	queue := &TaskQueue{
		accountID: accountID,
		tasks:     make([]*models.Task, 0),
	}
	ts.accountQueues[accountID] = queue

	ts.logger.Debug("Created new task queue", zap.String("account_id", accountID))
	return queue
}

// schedulingLoop 调度循环
func (ts *TaskScheduler) schedulingLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ts.ctx.Done():
			return
		case <-ticker.C:
			ts.processQueues()
		}
	}
}

// processQueues 处理所有队列
func (ts *TaskScheduler) processQueues() {
	ts.mu.RLock()
	queues := make([]*TaskQueue, 0, len(ts.accountQueues))
	for _, queue := range ts.accountQueues {
		queues = append(queues, queue)
	}
	ts.mu.RUnlock()

	for _, queue := range queues {
		ts.processQueue(queue)
	}
}

// processQueue 处理单个队列
func (ts *TaskScheduler) processQueue(queue *TaskQueue) {
	queue.mu.Lock()
	defer queue.mu.Unlock()

	// 如果正在处理或队列为空，跳过
	if queue.processing || len(queue.tasks) == 0 {
		return
	}

	// 检查账号是否忙碌
	if ts.connectionPool.IsAccountBusy(queue.accountID) {
		return
	}

	// 获取下一个任务
	task := queue.tasks[0]
	queue.tasks = queue.tasks[1:]
	queue.processing = true

	// 异步执行任务
	go func() {
		defer func() {
			queue.mu.Lock()
			queue.processing = false
			queue.mu.Unlock()
		}()

		ts.executeTask(task)
	}()
}

// executeTask 执行任务
func (ts *TaskScheduler) executeTask(task *models.Task) {
	accountID := fmt.Sprintf("%d", task.AccountID)

	ts.logger.Info("Starting task execution",
		zap.Uint64("task_id", task.ID),
		zap.String("account_id", accountID),
		zap.String("task_type", string(task.TaskType)))

	// 更新任务状态为运行中
	task.Status = models.TaskStatusRunning
	startTime := time.Now()
	task.StartedAt = &startTime

	if err := ts.taskRepo.UpdateTask(task.ID, map[string]interface{}{
		"status":     models.TaskStatusRunning,
		"started_at": startTime,
	}); err != nil {
		ts.logger.Error("Failed to update task status",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
	}

	// 执行风控检查
	if err := ts.performRiskControlCheck(task, accountID); err != nil {
		ts.logger.Warn("Risk control check failed, task rejected",
			zap.Uint64("task_id", task.ID),
			zap.String("account_id", accountID),
			zap.Error(err))
		ts.completeTaskWithError(task, fmt.Errorf("risk control check failed: %w", err))
		return
	}

	// 创建任务执行器
	taskExecutor, err := ts.createTaskExecutor(task)
	if err != nil {
		ts.logger.Error("Failed to create task executor",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
		ts.completeTaskWithError(task, err)
		return
	}

	// 执行任务
	err = ts.connectionPool.ExecuteTask(accountID, taskExecutor)

	// 完成任务
	if err != nil {
		ts.logger.Error("Task execution failed",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
		ts.completeTaskWithError(task, err)
	} else {
		ts.logger.Info("Task execution completed successfully",
			zap.Uint64("task_id", task.ID))
		ts.completeTaskWithSuccess(task)
	}
}

// completeTaskWithSuccess 成功完成任务
func (ts *TaskScheduler) completeTaskWithSuccess(task *models.Task) {
	task.Status = models.TaskStatusCompleted
	completedTime := time.Now()
	task.CompletedAt = &completedTime

	if err := ts.taskRepo.UpdateTask(task.ID, map[string]interface{}{
		"status":       models.TaskStatusCompleted,
		"completed_at": completedTime,
	}); err != nil {
		ts.logger.Error("Failed to update completed task",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
	}
}

// performRiskControlCheck 执行风控检查
func (ts *TaskScheduler) performRiskControlCheck(task *models.Task, accountID string) error {
	// 获取账号信息
	accountIDUint, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid account ID: %w", err)
	}

	account, err := ts.accountRepo.GetByID(accountIDUint)
	if err != nil {
		return fmt.Errorf("failed to get account: %w", err)
	}

	// 1. 检查账号状态
	if !account.IsAvailable() {
		return fmt.Errorf("account is not available, status: %s", account.Status)
	}

	// 2. 检查账号健康度
	if account.HealthScore < 0.3 {
		return fmt.Errorf("account health score too low: %.2f (minimum: 0.3)", account.HealthScore)
	}

	// 3. 检查账号是否忙碌
	if ts.connectionPool.IsAccountBusy(accountID) {
		return fmt.Errorf("account is busy with another task")
	}

	// 4. 检查连接状态
	connStatus := ts.connectionPool.GetConnectionStatus(accountID)
	statusStr := connStatus.String()
	if statusStr != "connected" {
		return fmt.Errorf("account connection not ready, status: %s", statusStr)
	}

	// 5. 检查账号是否需要冷却（如果最近有失败的任务）
	// 获取最近1小时内的失败任务数
	oneHourAgo := time.Now().Add(-1 * time.Hour)
	recentFailedTasks, err := ts.taskRepo.GetTasksByAccountID(accountIDUint, []string{"failed"})
	if err == nil {
		failedCount := 0
		for _, t := range recentFailedTasks {
			if t.CompletedAt != nil && t.CompletedAt.After(oneHourAgo) {
				failedCount++
			}
		}

		// 如果最近1小时内失败任务超过3个，拒绝新任务
		if failedCount >= 3 {
			return fmt.Errorf("account has too many recent failures (%d in last hour), cooling down required", failedCount)
		}
	}

	// 6. 检查账号是否在冷却状态
	if account.Status == models.AccountStatusCooling {
		return fmt.Errorf("account is in cooling period")
	}

	// 7. 检查任务频率限制（避免短时间内大量相同类型任务）
	recentTasks, err := ts.taskRepo.GetTasksByAccountID(accountIDUint, []string{"running", "queued"})
	if err == nil {
		sameTypeCount := 0
		for _, t := range recentTasks {
			if t.TaskType == task.TaskType {
				sameTypeCount++
			}
		}

		// 同一类型任务队列中超过5个，限制新任务
		if sameTypeCount >= 5 {
			ts.logger.Warn("Too many tasks of same type in queue",
				zap.String("account_id", accountID),
				zap.String("task_type", string(task.TaskType)),
				zap.Int("count", sameTypeCount))
			// 这里可以选择拒绝或允许，根据业务需求
		}
	}

	ts.logger.Debug("Risk control check passed",
		zap.Uint64("task_id", task.ID),
		zap.String("account_id", accountID),
		zap.Float64("health_score", account.HealthScore))

	return nil
}

// completeTaskWithError 失败完成任务
func (ts *TaskScheduler) completeTaskWithError(task *models.Task, taskErr error) {
	task.Status = models.TaskStatusFailed
	completedTime := time.Now()
	task.CompletedAt = &completedTime

	// 设置错误结果
	if task.Result == nil {
		task.Result = make(models.TaskResult)
	}
	task.Result["error"] = taskErr.Error()

	if err := ts.taskRepo.UpdateTask(task.ID, map[string]interface{}{
		"status":       models.TaskStatusFailed,
		"completed_at": completedTime,
		"result":       task.Result,
	}); err != nil {
		ts.logger.Error("Failed to update failed task",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
	}
}

// createTaskExecutor 创建任务执行器
func (ts *TaskScheduler) createTaskExecutor(task *models.Task) (telegram.TaskInterface, error) {
	switch task.TaskType {
	case models.TaskTypeCheck:
		return telegram.NewAccountCheckTask(task), nil
	case models.TaskTypePrivate:
		return telegram.NewPrivateMessageTask(task), nil
	case models.TaskTypeBroadcast:
		return telegram.NewBroadcastTask(task), nil
	case models.TaskTypeVerify:
		return telegram.NewVerifyCodeTask(task), nil
	case models.TaskTypeGroupChat:
		return telegram.NewGroupChatTask(task), nil
	default:
		return nil, fmt.Errorf("unsupported task type: %s", task.TaskType)
	}
}

// getAccountInfo 获取账号信息
func (ts *TaskScheduler) getAccountInfo(accountID string) (*models.TGAccount, error) {
	// 这里应该实现缓存逻辑，先从缓存获取，缓存不存在再从数据库获取
	// 为了简化示例，直接从数据库获取
	accountIDUint, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	return ts.accountRepo.GetByID(accountIDUint)
}

// getQueueSize 获取队列大小
func (ts *TaskScheduler) getQueueSize(accountID string) int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()

	if queue, exists := ts.accountQueues[accountID]; exists {
		queue.mu.Lock()
		defer queue.mu.Unlock()
		return len(queue.tasks)
	}
	return 0
}

// generateRecommendations 生成建议和警告
func (ts *TaskScheduler) generateRecommendations(account *models.TGAccount, availability *models.AccountAvailability) {
	if account.HealthScore < 0.3 {
		availability.Warnings = append(availability.Warnings, "账号健康度过低，建议暂停使用")
		availability.Recommendation = "建议让账号休息一段时间"
	} else if account.HealthScore < 0.7 {
		availability.Warnings = append(availability.Warnings, "账号健康度偏低，建议减少使用频率")
		availability.Recommendation = "适当降低任务频率"
	}

	if availability.QueueSize > 5 {
		availability.Warnings = append(availability.Warnings, "任务队列较长，执行可能延迟")
	}

	if availability.ConnectionStatus != models.ConnectionStatus(telegram.StatusConnected) {
		availability.Warnings = append(availability.Warnings, "连接状态异常")
	}
}

// GetQueueStatus 获取队列状态
func (ts *TaskScheduler) GetQueueStatus(accountID string) *models.QueueInfo {
	// 解析accountID
	accountIDUint, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return &models.QueueInfo{
			AccountID:         0,
			PendingTasks:      0,
			RunningTasks:      0,
			EstimatedWaitTime: 0,
		}
	}

	// 实现队列状态获取逻辑
	// 这里应该查询数据库获取更完整的统计信息
	return &models.QueueInfo{
		AccountID:         accountIDUint,
		PendingTasks:      int64(ts.getQueueSize(accountID)),
		RunningTasks:      0, // 需要实现
		EstimatedWaitTime: 0, // 需要实现
	}
}

// Close 关闭调度器
func (ts *TaskScheduler) Close() {
	ts.logger.Info("Closing task scheduler")
	ts.cancel()
}
