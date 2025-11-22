package scheduler

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
	"tg_cloud_server/internal/telegram"
)

// TaskScheduler 任务调度器
type TaskScheduler struct {
	taskQueue      []*models.Task               // 任务队列
	runningTasks   map[uint64]bool              // 正在运行的任务 (taskID -> true)
	connectionPool *telegram.ConnectionPool     // 连接池引用
	accountRepo    repository.AccountRepository // 账号仓库
	taskRepo       repository.TaskRepository    // 任务仓库
	logger         *zap.Logger
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	maxConcurrent  int // 最大并发任务数
}

// NewTaskScheduler 创建新的任务调度器
func NewTaskScheduler(
	connectionPool *telegram.ConnectionPool,
	accountRepo repository.AccountRepository,
	taskRepo repository.TaskRepository,
) *TaskScheduler {
	ctx, cancel := context.WithCancel(context.Background())

	ts := &TaskScheduler{
		taskQueue:      make([]*models.Task, 0),
		runningTasks:   make(map[uint64]bool),
		connectionPool: connectionPool,
		accountRepo:    accountRepo,
		taskRepo:       taskRepo,
		logger:         logger.Get().Named("task_scheduler"),
		ctx:            ctx,
		cancel:         cancel,
		maxConcurrent:  10, // 默认最多10个并发任务
	}

	// 启动调度循环
	go ts.schedulingLoop()

	return ts
}

// Stop 停止任务调度器
func (ts *TaskScheduler) Stop() {
	ts.logger.Info("Stopping task scheduler...")

	// 取消上下文，停止调度循环
	ts.cancel()

	// 等待正在执行的任务完成（最多等待10秒）
	deadline := time.Now().Add(10 * time.Second)

	for time.Now().Before(deadline) {
		ts.mu.RLock()
		hasRunningTasks := len(ts.runningTasks) > 0
		ts.mu.RUnlock()

		if !hasRunningTasks {
			break
		}

		if !hasRunningTasks {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	ts.logger.Info("Task scheduler stopped")
}

// SubmitTask 提交任务到指定账号队列
func (ts *TaskScheduler) SubmitTask(task *models.Task) error {
	if task == nil {
		return fmt.Errorf("task cannot be nil")
	}

	// 验证任务有账号
	accountIDs := task.GetAccountIDList()
	if len(accountIDs) == 0 {
		return fmt.Errorf("task has no accounts assigned")
	}

	// 验证所有账号可用性
	for _, accountID := range accountIDs {
		accountIDStr := fmt.Sprintf("%d", accountID)
		if err := ts.ValidateAccount(accountIDStr); err != nil {
			ts.logger.Warn("Account validation failed",
				zap.String("account_id", accountIDStr),
				zap.Uint64("task_id", task.ID),
				zap.Error(err))
			// 继续验证其他账号
		}
	}

	// 更新数据库中的任务状态（在添加到队列之前）
	if err := ts.taskRepo.UpdateStatus(task.ID, models.TaskStatusQueued); err != nil {
		ts.logger.Error("Failed to update task status to queued",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
		return fmt.Errorf("failed to update task status: %w", err)
	}

	// 添加任务到队列
	ts.mu.Lock()
	task.Status = models.TaskStatusQueued
	ts.taskQueue = append(ts.taskQueue, task)
	queueSize := len(ts.taskQueue)
	ts.mu.Unlock()

	// 使用专门的任务日志记录器
	logger.LogTask(zapcore.InfoLevel, "Task submitted to queue",
		zap.Uint64("task_id", task.ID),
		zap.Any("account_ids", accountIDs),
		zap.Int("account_count", len(accountIDs)),
		zap.String("task_type", string(task.TaskType)),
		zap.Int("priority", task.Priority),
		zap.Int("queue_size", queueSize),
		zap.Time("submitted_at", time.Now()))

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

// processQueues 处理任务队列
func (ts *TaskScheduler) processQueues() {
	ts.mu.Lock()

	// 检查是否达到最大并发数
	if len(ts.runningTasks) >= ts.maxConcurrent {
		ts.mu.Unlock()
		return
	}

	// 检查队列是否为空
	if len(ts.taskQueue) == 0 {
		ts.mu.Unlock()
		return
	}

	// 获取下一个任务（按优先级排序，优先级高的先执行）
	// 简单实现：取第一个任务
	task := ts.taskQueue[0]
	ts.taskQueue = ts.taskQueue[1:]

	// 标记任务为运行中
	ts.runningTasks[task.ID] = true

	ts.mu.Unlock()

	// 异步执行任务
	go func() {
		defer func() {
			// 从运行列表中移除
			ts.mu.Lock()
			delete(ts.runningTasks, task.ID)
			ts.mu.Unlock()

			// 处理panic
			if r := recover(); r != nil {
				logger.LogTask(zapcore.ErrorLevel, "Task execution panicked",
					zap.Uint64("task_id", task.ID),
					zap.String("task_type", string(task.TaskType)),
					zap.Any("panic", r))
				// 标记任务为失败
				ts.completeTaskWithError(task, fmt.Errorf("task execution panicked: %v", r))
			}
		}()

		ts.executeTask(task)
	}()
}

// executeTask 执行任务
func (ts *TaskScheduler) executeTask(task *models.Task) {
	// 获取账号ID列表
	accountIDs := task.GetAccountIDList()

	// 更新任务状态为运行中
	task.Status = models.TaskStatusRunning
	startTime := time.Now()
	task.StartedAt = &startTime

	logger.LogTask(zapcore.InfoLevel, "Starting task execution",
		zap.Uint64("task_id", task.ID),
		zap.Any("account_ids", accountIDs),
		zap.Int("account_count", len(accountIDs)),
		zap.String("task_type", string(task.TaskType)),
		zap.Int("priority", task.Priority),
		zap.Time("started_at", startTime))

	if err := ts.taskRepo.UpdateTask(task.ID, map[string]interface{}{
		"status":     models.TaskStatusRunning,
		"started_at": startTime,
	}); err != nil {
		ts.logger.Error("Failed to update task status",
			zap.Uint64("task_id", task.ID),
			zap.Error(err))
	}

	// 初始化结果记录
	if task.Result == nil {
		task.Result = make(models.TaskResult)
	}
	task.Result["account_results"] = make(map[string]interface{})
	accountResults := task.Result["account_results"].(map[string]interface{})

	// 依次使用每个账号执行任务
	successCount := 0
	failCount := 0
	var lastError error

	// 记录任务开始日志
	ts.createTaskLog(task.ID, nil, "task_started", fmt.Sprintf("开始执行任务，共 %d 个账号", len(accountIDs)), nil)

	for i, accountID := range accountIDs {
		accountIDStr := fmt.Sprintf("%d", accountID)

		logger.LogTask(zapcore.InfoLevel, "Executing task with account",
			zap.Uint64("task_id", task.ID),
			zap.String("account_id", accountIDStr),
			zap.Int("account_index", i+1),
			zap.Int("total_accounts", len(accountIDs)))

		// 记录账号开始执行日志
		ts.createTaskLog(task.ID, &accountID, "account_started", fmt.Sprintf("开始使用账号 %d 执行任务 (%d/%d)", accountID, i+1, len(accountIDs)), nil)

		// 执行风控检查
		if err := ts.performRiskControlCheck(task, accountIDStr); err != nil {
			ts.logger.Warn("Risk control check failed for account",
				zap.Uint64("task_id", task.ID),
				zap.String("account_id", accountIDStr),
				zap.Error(err))
			accountResults[accountIDStr] = map[string]interface{}{
				"status": "failed",
				"error":  fmt.Sprintf("risk control check failed: %v", err),
			}
			// 记录风控检查失败日志
			ts.createTaskLog(task.ID, &accountID, "risk_check_failed", fmt.Sprintf("账号 %d 风控检查失败: %v", accountID, err), nil)
			failCount++
			lastError = err
			continue
		}

		// 记录风控检查通过日志
		ts.createTaskLog(task.ID, &accountID, "risk_check_passed", fmt.Sprintf("账号 %d 风控检查通过", accountID), nil)

		// 创建任务执行器
		taskExecutor, err := ts.createTaskExecutor(task)
		if err != nil {
			ts.logger.Error("Failed to create task executor for account",
				zap.Uint64("task_id", task.ID),
				zap.String("account_id", accountIDStr),
				zap.Error(err))
			accountResults[accountIDStr] = map[string]interface{}{
				"status": "failed",
				"error":  fmt.Sprintf("failed to create executor: %v", err),
			}
			// 记录创建执行器失败日志
			ts.createTaskLog(task.ID, &accountID, "executor_creation_failed", fmt.Sprintf("账号 %d 创建任务执行器失败: %v", accountID, err), nil)
			failCount++
			lastError = err
			continue
		}

		// 执行任务
		accountStartTime := time.Now()
		err = ts.connectionPool.ExecuteTask(accountIDStr, taskExecutor)
		accountDuration := time.Since(accountStartTime)

		// 保存该账号的执行结果（从 task.Result 中提取）
		accountResult := make(map[string]interface{})
		accountResult["duration"] = accountDuration.String()

		// 复制任务执行器写入的结果
		for key, value := range task.Result {
			if key != "account_results" && key != "success_count" && key != "fail_count" && key != "total_accounts" {
				accountResult[key] = value
			}
		}

		if err != nil {
			logger.LogTask(zapcore.ErrorLevel, "Task execution failed for account",
				zap.Uint64("task_id", task.ID),
				zap.String("account_id", accountIDStr),
				zap.Duration("duration", accountDuration),
				zap.Error(err))
			accountResult["status"] = "failed"
			accountResult["error"] = err.Error()
			// 记录执行失败日志
			ts.createTaskLog(task.ID, &accountID, "execution_failed", fmt.Sprintf("账号 %d 执行失败: %v (耗时: %s)", accountID, err, accountDuration), accountResult)
			failCount++
			lastError = err
		} else {
			logger.LogTask(zapcore.InfoLevel, "Task execution succeeded for account",
				zap.Uint64("task_id", task.ID),
				zap.String("account_id", accountIDStr),
				zap.Duration("duration", accountDuration))
			accountResult["status"] = "success"

			// 记录每个目标的详细结果（如果有）
			if targetResults, ok := accountResult["target_results"].(map[string]interface{}); ok && len(targetResults) > 0 {
				for targetName, targetResult := range targetResults {
					if resultMap, ok := targetResult.(map[string]interface{}); ok {
						status := "unknown"
						if s, ok := resultMap["status"].(string); ok {
							status = s
						}

						var message string
						if status == "success" {
							message = fmt.Sprintf("账号 %d 成功发送给 %s", accountID, targetName)
						} else {
							errorMsg := "未知错误"
							if e, ok := resultMap["error"].(string); ok {
								errorMsg = e
							}
							message = fmt.Sprintf("账号 %d 发送给 %s 失败: %s", accountID, targetName, errorMsg)
						}

						ts.createTaskLog(task.ID, &accountID, fmt.Sprintf("target_%s", status), message, resultMap)
					}
				}
			}

			// 记录执行成功日志
			ts.createTaskLog(task.ID, &accountID, "execution_success", fmt.Sprintf("账号 %d 执行成功 (耗时: %s)", accountID, accountDuration), accountResult)
			successCount++
		}

		// 保存该账号的结果
		accountResults[accountIDStr] = accountResult

		// 恢复 account_results（防止被任务执行器覆盖）
		task.Result["account_results"] = accountResults
	}

	// 更新任务结果
	task.Result["success_count"] = successCount
	task.Result["fail_count"] = failCount
	task.Result["total_accounts"] = len(accountIDs)

	// 完成任务
	duration := time.Since(startTime)
	if successCount == 0 {
		// 所有账号都失败
		logger.LogTask(zapcore.ErrorLevel, "Task execution failed for all accounts",
			zap.Uint64("task_id", task.ID),
			zap.Int("total_accounts", len(accountIDs)),
			zap.Duration("duration", duration),
			zap.Error(lastError))
		ts.createTaskLog(task.ID, nil, "task_failed", fmt.Sprintf("任务执行失败，所有 %d 个账号都失败了 (总耗时: %s)", len(accountIDs), duration), task.Result)
		ts.completeTaskWithError(task, fmt.Errorf("all %d accounts failed, last error: %w", len(accountIDs), lastError))
	} else if failCount > 0 {
		// 部分成功
		logger.LogTask(zapcore.WarnLevel, "Task execution partially succeeded",
			zap.Uint64("task_id", task.ID),
			zap.Int("success_count", successCount),
			zap.Int("fail_count", failCount),
			zap.Int("total_accounts", len(accountIDs)),
			zap.Duration("duration", duration))
		ts.createTaskLog(task.ID, nil, "task_partial_success", fmt.Sprintf("任务部分成功: %d 成功, %d 失败 (总耗时: %s)", successCount, failCount, duration), task.Result)
		ts.completeTaskWithSuccess(task)
	} else {
		// 全部成功
		logger.LogTask(zapcore.InfoLevel, "Task execution completed successfully for all accounts",
			zap.Uint64("task_id", task.ID),
			zap.Int("total_accounts", len(accountIDs)),
			zap.Duration("duration", duration))
		ts.createTaskLog(task.ID, nil, "task_completed", fmt.Sprintf("任务执行成功，所有 %d 个账号都成功了 (总耗时: %s)", len(accountIDs), duration), task.Result)
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

	// 4. 连接状态检查移到实际执行时进行，这里只检查连接是否处于错误状态
	connStatus := ts.connectionPool.GetConnectionStatus(accountID)
	if connStatus == telegram.StatusConnectionError {
		return fmt.Errorf("account connection has persistent error")
	}
	// 注意：StatusDisconnected 和 StatusConnecting 不算错误，连接会在执行时自动创建

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
	// 现在队列不再按账号分组，返回总队列大小
	ts.mu.RLock()
	size := len(ts.taskQueue)
	ts.mu.RUnlock()
	return size
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

	if availability.ConnectionStatus != models.StatusConnected {
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

// createTaskLog 创建任务日志
func (ts *TaskScheduler) createTaskLog(taskID uint64, accountID *uint64, action, message string, extraData interface{}) {
	var extraDataJSON []byte
	if extraData != nil {
		var err error
		extraDataJSON, err = json.Marshal(extraData)
		if err != nil {
			ts.logger.Warn("Failed to marshal extra data for task log",
				zap.Uint64("task_id", taskID),
				zap.Error(err))
			extraDataJSON = []byte("{}")
		}
	} else {
		extraDataJSON = []byte("{}")
	}

	taskLog := &models.TaskLog{
		TaskID:    taskID,
		AccountID: accountID,
		Action:    action,
		Message:   message,
		ExtraData: extraDataJSON,
		CreatedAt: time.Now(),
	}

	if err := ts.taskRepo.CreateTaskLog(taskLog); err != nil {
		ts.logger.Error("Failed to create task log",
			zap.Uint64("task_id", taskID),
			zap.String("action", action),
			zap.Error(err))
	}
}
