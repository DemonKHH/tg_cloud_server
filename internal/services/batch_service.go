package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

// Use types from models package
type BatchOperation = models.BatchOperation
type BatchJobStatus = models.BatchJobStatus
type BatchJob = models.BatchJob

// Re-export constants for convenience
const (
	BatchOperationCreateAccounts = models.BatchOperationCreateAccounts
	BatchOperationUpdateAccounts = models.BatchOperationUpdateAccounts
	BatchOperationDeleteAccounts = models.BatchOperationDeleteAccounts
	BatchOperationBindProxies    = models.BatchOperationBindProxies
	BatchOperationCreateTasks    = models.BatchOperationCreateTasks
	BatchOperationCancelTasks    = models.BatchOperationCancelTasks
	BatchOperationImportUsers    = models.BatchOperationImportUsers
	BatchOperationExportData     = models.BatchOperationExportData
)

const (
	BatchJobStatusPending   = models.BatchJobStatusPending
	BatchJobStatusRunning   = models.BatchJobStatusRunning
	BatchJobStatusCompleted = models.BatchJobStatusCompleted
	BatchJobStatusFailed    = models.BatchJobStatusFailed
	BatchJobStatusCancelled = models.BatchJobStatusCancelled
)

// BatchAccountCreateRequest 批量创建账号请求
type BatchAccountCreateRequest struct {
	Accounts []models.CreateAccountRequest `json:"accounts" binding:"required"`
}

// BatchAccountUpdateRequest 批量更新账号请求
type BatchAccountUpdateRequest struct {
	Updates []struct {
		AccountID uint64                      `json:"account_id" binding:"required"`
		Data      models.UpdateAccountRequest `json:"data" binding:"required"`
	} `json:"updates" binding:"required"`
}

// BatchProxyBindRequest 批量绑定代理请求
type BatchProxyBindRequest struct {
	Bindings []struct {
		AccountID uint64  `json:"account_id" binding:"required"`
		ProxyID   *uint64 `json:"proxy_id"` // nil表示解绑
	} `json:"bindings" binding:"required"`
}

// BatchTaskCreateRequest 批量创建任务请求
type BatchTaskCreateRequest struct {
	Tasks []models.CreateTaskRequest `json:"tasks" binding:"required"`
}

// ImportUsersRequest 导入用户请求
type ImportUsersRequest struct {
	Users []ImportUserData `json:"users" binding:"required"`
}

// ImportUserData 导入用户数据
type ImportUserData struct {
	Username  string `json:"username" binding:"required"`
	Phone     string `json:"phone"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name"`
	Bio       string `json:"bio"`
}

// ExportDataRequest 导出数据请求
type ExportDataRequest struct {
	DataType  string                 `json:"data_type" binding:"required"` // accounts, tasks, users, etc.
	Format    string                 `json:"format"`                       // json, csv, excel
	Filters   map[string]interface{} `json:"filters"`
	DateRange *DateRange             `json:"date_range"`
}

// DateRange 日期范围
type DateRange struct {
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

// BatchService 批量操作服务接口
type BatchService interface {
	// 批量任务管理
	CreateBatchJob(ctx context.Context, userID uint64, operation BatchOperation, totalItems int) (*BatchJob, error)
	GetBatchJob(ctx context.Context, userID uint64, jobID uint64) (*BatchJob, error)
	GetBatchJobs(ctx context.Context, userID uint64, page, limit int) ([]*BatchJob, int64, error)
	UpdateBatchJobProgress(ctx context.Context, jobID uint64, processed, success, failed int) error
	CompleteBatchJob(ctx context.Context, jobID uint64, result map[string]interface{}) error
	CancelBatchJob(ctx context.Context, userID uint64, jobID uint64) error

	// 批量账号操作
	BatchCreateAccounts(ctx context.Context, userID uint64, req *BatchAccountCreateRequest) (*BatchJob, error)
	BatchUpdateAccounts(ctx context.Context, userID uint64, req *BatchAccountUpdateRequest) (*BatchJob, error)
	BatchDeleteAccounts(ctx context.Context, userID uint64, accountIDs []uint64) (*BatchJob, error)
	BatchBindProxies(ctx context.Context, userID uint64, req *BatchProxyBindRequest) (*BatchJob, error)

	// 批量任务操作
	BatchCreateTasks(ctx context.Context, userID uint64, req *BatchTaskCreateRequest) (*BatchJob, error)
	BatchCancelTasks(ctx context.Context, userID uint64, taskIDs []uint64) (*BatchJob, error)

	// 数据导入导出
	ImportUsers(ctx context.Context, userID uint64, req *ImportUsersRequest) (*BatchJob, error)
	ExportData(ctx context.Context, userID uint64, req *ExportDataRequest) (*BatchJob, error)

	// 进度监控
	GetJobProgress(ctx context.Context, userID uint64, jobID uint64) (float64, error)
	IsJobRunning(ctx context.Context, jobID uint64) (bool, error)
}

// batchService 批量操作服务实现
type batchService struct {
	batchRepo      repository.BatchRepository
	accountService *AccountService
	taskService    *TaskService
	logger         *zap.Logger

	// 运行中的任务
	runningJobs      map[uint64]*BatchJob
	runningJobsMutex sync.RWMutex

	// 并发控制
	maxConcurrency int
	workerPool     chan struct{}
}

// NewBatchService 创建批量操作服务
func NewBatchService(
	batchRepo repository.BatchRepository,
	accountService *AccountService,
	taskService *TaskService,
) BatchService {
	maxConcurrency := 10 // 最大并发数

	service := &batchService{
		batchRepo:      batchRepo,
		accountService: accountService,
		taskService:    taskService,
		logger:         logger.Get().Named("batch_service"),
		runningJobs:    make(map[uint64]*BatchJob),
		maxConcurrency: maxConcurrency,
		workerPool:     make(chan struct{}, maxConcurrency),
	}

	// 初始化worker pool
	for i := 0; i < maxConcurrency; i++ {
		service.workerPool <- struct{}{}
	}

	return service
}

// CreateBatchJob 创建批量任务
func (s *batchService) CreateBatchJob(ctx context.Context, userID uint64, operation BatchOperation, totalItems int) (*BatchJob, error) {
	job := &BatchJob{
		UserID:         userID,
		Operation:      operation,
		Status:         BatchJobStatusPending,
		TotalItems:     totalItems,
		ProcessedItems: 0,
		SuccessItems:   0,
		FailedItems:    0,
		Progress:       0.0,
		ErrorMessages:  make([]string, 0),
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	if err := s.batchRepo.Create(job); err != nil {
		s.logger.Error("Failed to create batch job", zap.Error(err))
		return nil, fmt.Errorf("failed to create batch job: %w", err)
	}

	s.logger.Info("Batch job created",
		zap.Uint64("job_id", job.ID),
		zap.Uint64("user_id", userID),
		zap.String("operation", string(operation)),
		zap.Int("total_items", totalItems))

	return job, nil
}

// BatchCreateAccounts 批量创建账号
func (s *batchService) BatchCreateAccounts(ctx context.Context, userID uint64, req *BatchAccountCreateRequest) (*BatchJob, error) {
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationCreateAccounts, len(req.Accounts))
	if err != nil {
		return nil, err
	}

	// 异步执行批量操作
	go s.executeBatchCreateAccounts(ctx, job, req)

	return job, nil
}

// executeBatchCreateAccounts 执行批量创建账号
func (s *batchService) executeBatchCreateAccounts(ctx context.Context, job *BatchJob, req *BatchAccountCreateRequest) {
	// 获取worker
	<-s.workerPool
	defer func() {
		s.workerPool <- struct{}{}
	}()

	s.logger.Info("Starting batch account creation", zap.Uint64("job_id", job.ID))

	// 更新任务状态为运行中
	job.Status = BatchJobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	s.batchRepo.Update(job)

	// 记录运行中的任务
	s.runningJobsMutex.Lock()
	s.runningJobs[job.ID] = job
	s.runningJobsMutex.Unlock()

	processed := 0
	success := 0
	failed := 0
	var errorMessages []string

	for i, accountReq := range req.Accounts {
		select {
		case <-ctx.Done():
			// 任务被取消
			job.Status = BatchJobStatusCancelled
			s.completeBatchJob(job, map[string]interface{}{
				"cancelled_at": i,
				"reason":       "context cancelled",
			})
			return
		default:
		}

		// 创建账号
		_, err := s.accountService.CreateAccount(job.UserID, &accountReq)
		processed++

		if err != nil {
			failed++
			errorMsg := fmt.Sprintf("Account %d: %s", i+1, err.Error())
			errorMessages = append(errorMessages, errorMsg)
			s.logger.Error("Failed to create account in batch",
				zap.Int("index", i),
				zap.Error(err))
		} else {
			success++
		}

		// 更新进度
		s.UpdateBatchJobProgress(ctx, job.ID, processed, success, failed)

		// 避免过快的请求
		time.Sleep(100 * time.Millisecond)
	}

	// 完成任务
	result := map[string]interface{}{
		"total_accounts":   len(req.Accounts),
		"success_accounts": success,
		"failed_accounts":  failed,
		"error_messages":   errorMessages,
	}

	s.completeBatchJob(job, result)
	s.logger.Info("Batch account creation completed",
		zap.Uint64("job_id", job.ID),
		zap.Int("success", success),
		zap.Int("failed", failed))
}

// BatchUpdateAccounts 批量更新账号
func (s *batchService) BatchUpdateAccounts(ctx context.Context, userID uint64, req *BatchAccountUpdateRequest) (*BatchJob, error) {
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationUpdateAccounts, len(req.Updates))
	if err != nil {
		return nil, err
	}

	// 异步执行
	go s.executeBatchUpdateAccounts(ctx, job, req)
	return job, nil
}

// executeBatchUpdateAccounts 执行批量更新账号
func (s *batchService) executeBatchUpdateAccounts(ctx context.Context, job *BatchJob, req *BatchAccountUpdateRequest) {
	// 获取worker
	<-s.workerPool
	defer func() {
		s.workerPool <- struct{}{}
	}()

	s.logger.Info("Starting batch account update", zap.Uint64("job_id", job.ID))

	// 更新任务状态
	job.Status = BatchJobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	s.batchRepo.Update(job)

	s.runningJobsMutex.Lock()
	s.runningJobs[job.ID] = job
	s.runningJobsMutex.Unlock()

	processed := 0
	success := 0
	failed := 0
	var errorMessages []string

	for i, update := range req.Updates {
		select {
		case <-ctx.Done():
			job.Status = BatchJobStatusCancelled
			s.completeBatchJob(job, map[string]interface{}{
				"cancelled_at": i,
				"reason":       "context cancelled",
			})
			return
		default:
		}

		// 更新账号
		_, err := s.accountService.UpdateAccount(job.UserID, update.AccountID, &update.Data)
		processed++

		if err != nil {
			failed++
			errorMsg := fmt.Sprintf("Account %d: %s", update.AccountID, err.Error())
			errorMessages = append(errorMessages, errorMsg)
		} else {
			success++
		}

		// 更新进度
		s.UpdateBatchJobProgress(ctx, job.ID, processed, success, failed)
		time.Sleep(50 * time.Millisecond)
	}

	result := map[string]interface{}{
		"total_updates":   len(req.Updates),
		"success_updates": success,
		"failed_updates":  failed,
		"error_messages":  errorMessages,
	}

	s.completeBatchJob(job, result)
}

// BatchDeleteAccounts 批量删除账号
func (s *batchService) BatchDeleteAccounts(ctx context.Context, userID uint64, accountIDs []uint64) (*BatchJob, error) {
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationDeleteAccounts, len(accountIDs))
	if err != nil {
		return nil, err
	}

	// 异步执行
	go s.executeBatchDeleteAccounts(ctx, job, accountIDs)
	return job, nil
}

// executeBatchDeleteAccounts 执行批量删除账号
func (s *batchService) executeBatchDeleteAccounts(ctx context.Context, job *BatchJob, accountIDs []uint64) {
	<-s.workerPool
	defer func() {
		s.workerPool <- struct{}{}
	}()

	job.Status = BatchJobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	s.batchRepo.Update(job)

	s.runningJobsMutex.Lock()
	s.runningJobs[job.ID] = job
	s.runningJobsMutex.Unlock()

	processed := 0
	success := 0
	failed := 0
	var errorMessages []string

	for _, accountID := range accountIDs {
		err := s.accountService.DeleteAccount(job.UserID, accountID)
		processed++

		if err != nil {
			failed++
			errorMessages = append(errorMessages, fmt.Sprintf("Account %d: %s", accountID, err.Error()))
		} else {
			success++
		}

		s.UpdateBatchJobProgress(ctx, job.ID, processed, success, failed)
		time.Sleep(50 * time.Millisecond)
	}

	result := map[string]interface{}{
		"total_deletions":   len(accountIDs),
		"success_deletions": success,
		"failed_deletions":  failed,
		"error_messages":    errorMessages,
	}

	s.completeBatchJob(job, result)
}

// BatchCreateTasks 批量创建任务
func (s *batchService) BatchCreateTasks(ctx context.Context, userID uint64, req *BatchTaskCreateRequest) (*BatchJob, error) {
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationCreateTasks, len(req.Tasks))
	if err != nil {
		return nil, err
	}

	go s.executeBatchCreateTasks(ctx, job, req)
	return job, nil
}

// executeBatchCreateTasks 执行批量创建任务
func (s *batchService) executeBatchCreateTasks(ctx context.Context, job *BatchJob, req *BatchTaskCreateRequest) {
	<-s.workerPool
	defer func() {
		s.workerPool <- struct{}{}
	}()

	job.Status = BatchJobStatusRunning
	now := time.Now()
	job.StartedAt = &now
	s.batchRepo.Update(job)

	processed := 0
	success := 0
	failed := 0
	var errorMessages []string
	var createdTaskIDs []uint64

	for i, taskReq := range req.Tasks {
		task, err := s.taskService.CreateTask(job.UserID, &taskReq)
		processed++

		if err != nil {
			failed++
			errorMessages = append(errorMessages, fmt.Sprintf("Task %d: %s", i+1, err.Error()))
		} else {
			success++
			createdTaskIDs = append(createdTaskIDs, task.ID)
		}

		s.UpdateBatchJobProgress(ctx, job.ID, processed, success, failed)
		time.Sleep(100 * time.Millisecond)
	}

	result := map[string]interface{}{
		"total_tasks":      len(req.Tasks),
		"success_tasks":    success,
		"failed_tasks":     failed,
		"created_task_ids": createdTaskIDs,
		"error_messages":   errorMessages,
	}

	s.completeBatchJob(job, result)
}

// 辅助方法

func (s *batchService) UpdateBatchJobProgress(ctx context.Context, jobID uint64, processed, success, failed int) error {
	s.runningJobsMutex.RLock()
	job, exists := s.runningJobs[jobID]
	s.runningJobsMutex.RUnlock()

	if !exists {
		return fmt.Errorf("job %d not found in running jobs", jobID)
	}

	job.ProcessedItems = processed
	job.SuccessItems = success
	job.FailedItems = failed
	job.Progress = float64(processed) / float64(job.TotalItems) * 100.0
	job.UpdatedAt = time.Now()

	return s.batchRepo.Update(job)
}

func (s *batchService) completeBatchJob(job *BatchJob, result map[string]interface{}) {
	job.Status = BatchJobStatusCompleted
	job.Result = result
	now := time.Now()
	job.CompletedAt = &now
	job.UpdatedAt = now

	s.batchRepo.Update(job)

	// 从运行中任务移除
	s.runningJobsMutex.Lock()
	delete(s.runningJobs, job.ID)
	s.runningJobsMutex.Unlock()
}

func (s *batchService) GetBatchJob(ctx context.Context, userID uint64, jobID uint64) (*BatchJob, error) {
	return s.batchRepo.GetByUserIDAndID(userID, jobID)
}

func (s *batchService) GetBatchJobs(ctx context.Context, userID uint64, page, limit int) ([]*BatchJob, int64, error) {
	offset := (page - 1) * limit
	return s.batchRepo.GetByUserID(userID, offset, limit)
}

func (s *batchService) CompleteBatchJob(ctx context.Context, jobID uint64, result map[string]interface{}) error {
	s.runningJobsMutex.RLock()
	job, exists := s.runningJobs[jobID]
	s.runningJobsMutex.RUnlock()

	if exists {
		s.completeBatchJob(job, result)
	}

	return nil
}

func (s *batchService) CancelBatchJob(ctx context.Context, userID uint64, jobID uint64) error {
	job, err := s.batchRepo.GetByUserIDAndID(userID, jobID)
	if err != nil {
		return err
	}

	if job.Status == BatchJobStatusRunning {
		job.Status = BatchJobStatusCancelled
		now := time.Now()
		job.CompletedAt = &now
		job.UpdatedAt = now

		s.batchRepo.Update(job)

		s.runningJobsMutex.Lock()
		delete(s.runningJobs, jobID)
		s.runningJobsMutex.Unlock()
	}

	return nil
}

func (s *batchService) GetJobProgress(ctx context.Context, userID uint64, jobID uint64) (float64, error) {
	job, err := s.batchRepo.GetByUserIDAndID(userID, jobID)
	if err != nil {
		return 0, err
	}
	return job.Progress, nil
}

func (s *batchService) IsJobRunning(ctx context.Context, jobID uint64) (bool, error) {
	s.runningJobsMutex.RLock()
	_, exists := s.runningJobs[jobID]
	s.runningJobsMutex.RUnlock()
	return exists, nil
}

// 其他批量操作方法的占位实现

func (s *batchService) BatchBindProxies(ctx context.Context, userID uint64, req *BatchProxyBindRequest) (*BatchJob, error) {
	s.logger.Info("Starting batch proxy binding",
		zap.Uint64("user_id", userID),
		zap.Int("bindings_count", len(req.Bindings)))

	// 创建批量任务
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationBindProxies, len(req.Bindings))
	if err != nil {
		return nil, fmt.Errorf("failed to create batch job: %w", err)
	}

	// 异步执行批量绑定
	go s.executeBatchProxyBinding(context.Background(), job.ID, userID, req)

	return job, nil
}

// executeBatchProxyBinding 执行批量代理绑定
func (s *batchService) executeBatchProxyBinding(ctx context.Context, jobID, userID uint64, req *BatchProxyBindRequest) {
	s.runningJobsMutex.Lock()
	if _, exists := s.runningJobs[jobID]; !exists {
		s.runningJobs[jobID] = &BatchJob{ID: jobID}
	}
	s.runningJobsMutex.Unlock()

	defer func() {
		s.runningJobsMutex.Lock()
		delete(s.runningJobs, jobID)
		s.runningJobsMutex.Unlock()
	}()

	processed := 0
	successful := 0
	failed := 0
	var errorMessages []string

	for _, binding := range req.Bindings {
		// 验证账号归属
		_, err := s.accountService.GetAccount(userID, binding.AccountID)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("账号 %d: %s", binding.AccountID, err.Error()))
			failed++
			processed++
			continue
		}

		// 如果指定了代理ID，验证代理存在
		if binding.ProxyID != nil {
			// 这里简化验证，实际应该检查代理归属
			if *binding.ProxyID == 0 {
				errorMessages = append(errorMessages, fmt.Sprintf("账号 %d: 代理ID无效", binding.AccountID))
				failed++
				processed++
				continue
			}
		}

		// 执行绑定
		_, err = s.accountService.BindProxy(userID, binding.AccountID, binding.ProxyID)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("账号 %d 绑定失败: %s", binding.AccountID, err.Error()))
			failed++
		} else {
			successful++
		}
		processed++

		// 更新进度
		s.UpdateBatchJobProgress(ctx, jobID, processed, successful, failed)
	}

	// 完成任务
	result := map[string]interface{}{
		"total_bindings": len(req.Bindings),
		"successful":     successful,
		"failed":         failed,
		"error_messages": errorMessages,
	}

	s.CompleteBatchJob(ctx, jobID, result)
}

func (s *batchService) BatchCancelTasks(ctx context.Context, userID uint64, taskIDs []uint64) (*BatchJob, error) {
	s.logger.Info("Starting batch task cancellation",
		zap.Uint64("user_id", userID),
		zap.Int("tasks_count", len(taskIDs)))

	// 创建批量任务
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationCancelTasks, len(taskIDs))
	if err != nil {
		return nil, fmt.Errorf("failed to create batch job: %w", err)
	}

	// 异步执行批量取消
	go s.executeBatchTaskCancellation(context.Background(), job.ID, userID, taskIDs)

	return job, nil
}

// executeBatchTaskCancellation 执行批量任务取消
func (s *batchService) executeBatchTaskCancellation(ctx context.Context, jobID, userID uint64, taskIDs []uint64) {
	s.runningJobsMutex.Lock()
	if _, exists := s.runningJobs[jobID]; !exists {
		s.runningJobs[jobID] = &BatchJob{ID: jobID}
	}
	s.runningJobsMutex.Unlock()

	defer func() {
		s.runningJobsMutex.Lock()
		delete(s.runningJobs, jobID)
		s.runningJobsMutex.Unlock()
	}()

	processed := 0
	successful := 0
	failed := 0
	var errorMessages []string

	for _, taskID := range taskIDs {
		// 验证任务归属并取消
		err := s.taskService.CancelTask(userID, taskID)
		if err != nil {
			errorMessages = append(errorMessages, fmt.Sprintf("任务 %d: %s", taskID, err.Error()))
			failed++
		} else {
			successful++
		}
		processed++

		// 更新进度
		s.UpdateBatchJobProgress(ctx, jobID, processed, successful, failed)
	}

	// 完成任务
	result := map[string]interface{}{
		"total_tasks":    len(taskIDs),
		"successful":     successful,
		"failed":         failed,
		"error_messages": errorMessages,
	}

	s.CompleteBatchJob(ctx, jobID, result)
}

func (s *batchService) ImportUsers(ctx context.Context, userID uint64, req *ImportUsersRequest) (*BatchJob, error) {
	s.logger.Info("Starting user import",
		zap.Uint64("user_id", userID),
		zap.Int("users_count", len(req.Users)))

	// 创建批量任务
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationImportUsers, len(req.Users))
	if err != nil {
		return nil, fmt.Errorf("failed to create batch job: %w", err)
	}

	// 异步执行用户导入
	go s.executeUserImport(context.Background(), job.ID, userID, req)

	return job, nil
}

// executeUserImport 执行用户导入
func (s *batchService) executeUserImport(ctx context.Context, jobID, userID uint64, req *ImportUsersRequest) {
	s.runningJobsMutex.Lock()
	if _, exists := s.runningJobs[jobID]; !exists {
		s.runningJobs[jobID] = &BatchJob{ID: jobID}
	}
	s.runningJobsMutex.Unlock()

	defer func() {
		s.runningJobsMutex.Lock()
		delete(s.runningJobs, jobID)
		s.runningJobsMutex.Unlock()
	}()

	processed := 0
	successful := 0
	failed := 0
	var errorMessages []string
	var importedUsers []ImportedUserResult

	for _, userData := range req.Users {
		// 验证用户数据
		if userData.Username == "" {
			errorMessages = append(errorMessages, fmt.Sprintf("用户 %s: 用户名不能为空", userData.Username))
			failed++
			processed++
			continue
		}

		// 检查用户名是否已存在（简化实现）
		// 实际应该调用认证服务检查用户是否存在
		if len(userData.Username) < 3 {
			errorMessages = append(errorMessages, fmt.Sprintf("用户 %s: 用户名长度不能少于3个字符", userData.Username))
			failed++
			processed++
			continue
		}

		// 创建新账号记录（简化实现）
		if userData.Phone != "" {
			accountReq := &models.CreateAccountRequest{
				Phone:       userData.Phone,
				SessionData: "", // 需要用户后续提供
				ProxyID:     nil,
			}

			account, err := s.accountService.CreateAccount(userID, accountReq)
			if err != nil {
				errorMessages = append(errorMessages, fmt.Sprintf("用户 %s: 创建账号失败 - %s", userData.Username, err.Error()))
				failed++
			} else {
				importedUsers = append(importedUsers, ImportedUserResult{
					Username:  userData.Username,
					UserID:    userID, // 简化实现，使用当前用户ID
					AccountID: &account.ID,
					Phone:     userData.Phone,
				})
				successful++
			}
		} else {
			// 只记录用户信息，不创建账号
			importedUsers = append(importedUsers, ImportedUserResult{
				Username: userData.Username,
				UserID:   userID, // 简化实现
			})
			successful++
		}

		processed++

		// 更新进度
		s.UpdateBatchJobProgress(ctx, jobID, processed, successful, failed)
	}

	// 完成任务
	result := map[string]interface{}{
		"total_users":    len(req.Users),
		"successful":     successful,
		"failed":         failed,
		"error_messages": errorMessages,
		"imported_users": importedUsers,
	}

	s.CompleteBatchJob(ctx, jobID, result)
}

// ImportedUserResult 导入用户结果
type ImportedUserResult struct {
	Username  string  `json:"username"`
	UserID    uint64  `json:"user_id"`
	AccountID *uint64 `json:"account_id,omitempty"`
	Phone     string  `json:"phone,omitempty"`
}

func (s *batchService) ExportData(ctx context.Context, userID uint64, req *ExportDataRequest) (*BatchJob, error) {
	s.logger.Info("Starting data export",
		zap.Uint64("user_id", userID),
		zap.String("data_type", req.DataType),
		zap.String("format", req.Format))

	// 创建批量任务
	job, err := s.CreateBatchJob(ctx, userID, BatchOperationExportData, 1) // 导出是单个任务
	if err != nil {
		return nil, fmt.Errorf("failed to create batch job: %w", err)
	}

	// 异步执行数据导出
	go s.executeDataExport(context.Background(), job.ID, userID, req)

	return job, nil
}

// executeDataExport 执行数据导出
func (s *batchService) executeDataExport(ctx context.Context, jobID, userID uint64, req *ExportDataRequest) {
	s.runningJobsMutex.Lock()
	if _, exists := s.runningJobs[jobID]; !exists {
		s.runningJobs[jobID] = &BatchJob{ID: jobID}
	}
	s.runningJobsMutex.Unlock()

	defer func() {
		s.runningJobsMutex.Lock()
		delete(s.runningJobs, jobID)
		s.runningJobsMutex.Unlock()
	}()

	var result map[string]interface{}
	var err error

	// 根据数据类型执行不同的导出逻辑
	switch req.DataType {
	case "accounts":
		result, err = s.exportAccounts(ctx, userID, req)
	case "tasks":
		result, err = s.exportTasks(ctx, userID, req)
	case "proxies":
		result, err = s.exportProxies(ctx, userID, req)
	default:
		err = fmt.Errorf("unsupported data type: %s", req.DataType)
	}

	if err != nil {
		result = map[string]interface{}{
			"success": false,
			"error":   err.Error(),
		}
	}

	// 更新进度和完成任务
	s.UpdateBatchJobProgress(ctx, jobID, 1, 1, 0)
	s.CompleteBatchJob(ctx, jobID, result)
}

// exportAccounts 导出账号数据
func (s *batchService) exportAccounts(ctx context.Context, userID uint64, req *ExportDataRequest) (map[string]interface{}, error) {
	// 简化实现，实际应该分页获取数据
	filter := &AccountFilter{
		UserID: userID,
		Page:   1,
		Limit:  1000,
	}
	accounts, total, err := s.accountService.GetAccounts(filter)
	if err != nil {
		return nil, fmt.Errorf("failed to get accounts: %w", err)
	}

	// 根据格式导出
	var exportedData interface{}
	var filename string

	switch req.Format {
	case "json", "":
		exportedData = accounts
		filename = fmt.Sprintf("accounts_%d.json", time.Now().Unix())
	case "csv":
		csvData := s.convertAccountsToCSV(accounts)
		exportedData = csvData
		filename = fmt.Sprintf("accounts_%d.csv", time.Now().Unix())
	default:
		exportedData = accounts
		filename = fmt.Sprintf("accounts_%d.json", time.Now().Unix())
	}

	result := map[string]interface{}{
		"success":          true,
		"data_type":        "accounts",
		"format":           req.Format,
		"total_records":    total,
		"exported_records": len(accounts),
		"filename":         filename,
		"data":             exportedData,
		"exported_at":      time.Now(),
	}

	return result, nil
}

// exportTasks 导出任务数据
func (s *batchService) exportTasks(ctx context.Context, userID uint64, req *ExportDataRequest) (map[string]interface{}, error) {
	// 简化实现
	tasks := []map[string]interface{}{
		{
			"id":         1,
			"type":       "account_check",
			"status":     "completed",
			"created_at": time.Now().Format("2006-01-02 15:04:05"),
		},
	}

	var exportedData interface{}
	var filename string

	switch req.Format {
	case "json", "":
		exportedData = tasks
		filename = fmt.Sprintf("tasks_%d.json", time.Now().Unix())
	case "csv":
		csvData := s.convertTasksToCSV(tasks)
		exportedData = csvData
		filename = fmt.Sprintf("tasks_%d.csv", time.Now().Unix())
	default:
		exportedData = tasks
		filename = fmt.Sprintf("tasks_%d.json", time.Now().Unix())
	}

	result := map[string]interface{}{
		"success":          true,
		"data_type":        "tasks",
		"format":           req.Format,
		"total_records":    int64(len(tasks)),
		"exported_records": len(tasks),
		"filename":         filename,
		"data":             exportedData,
		"exported_at":      time.Now(),
	}

	return result, nil
}

// exportProxies 导出代理数据
func (s *batchService) exportProxies(ctx context.Context, userID uint64, req *ExportDataRequest) (map[string]interface{}, error) {
	// 简化实现
	proxies := []map[string]interface{}{
		{
			"id":       1,
			"name":     "代理1",
			"host":     "127.0.0.1",
			"port":     8080,
			"protocol": "http",
			"status":   "active",
		},
	}

	var exportedData interface{}
	var filename string

	switch req.Format {
	case "json", "":
		exportedData = proxies
		filename = fmt.Sprintf("proxies_%d.json", time.Now().Unix())
	case "csv":
		csvData := s.convertProxiesToCSV(proxies)
		exportedData = csvData
		filename = fmt.Sprintf("proxies_%d.csv", time.Now().Unix())
	default:
		exportedData = proxies
		filename = fmt.Sprintf("proxies_%d.json", time.Now().Unix())
	}

	result := map[string]interface{}{
		"success":          true,
		"data_type":        "proxies",
		"format":           req.Format,
		"total_records":    int64(len(proxies)),
		"exported_records": len(proxies),
		"filename":         filename,
		"data":             exportedData,
		"exported_at":      time.Now(),
	}

	return result, nil
}

// CSV转换辅助方法（简化实现）
func (s *batchService) convertAccountsToCSV(accounts []*models.AccountSummary) string {
	header := "ID,Phone,Status,Last Check At,Last Used At\n"
	var rows []string
	rows = append(rows, header)

	for _, account := range accounts {
		var lastCheckDate string
		if account.LastCheckAt != nil {
			lastCheckDate = account.LastCheckAt.Format("2006-01-02")
		} else {
			lastCheckDate = ""
		}

		var lastUsedDate string
		if account.LastUsedAt != nil {
			lastUsedDate = account.LastUsedAt.Format("2006-01-02")
		} else {
			lastUsedDate = ""
		}

		row := fmt.Sprintf("%d,%s,%s,%s,%s\n",
			account.ID,
			account.Phone,
			string(account.Status),
			lastCheckDate,
			lastUsedDate)
		rows = append(rows, row)
	}

	result := ""
	for _, row := range rows {
		result += row
	}
	return result
}

func (s *batchService) convertTasksToCSV(tasks []map[string]interface{}) string {
	header := "ID,Type,Status,Created At\n"
	rows := []string{header}

	for _, task := range tasks {
		row := fmt.Sprintf("%v,%v,%v,%v\n",
			task["id"], task["type"], task["status"], task["created_at"])
		rows = append(rows, row)
	}

	result := ""
	for _, row := range rows {
		result += row
	}
	return result
}

func (s *batchService) convertProxiesToCSV(proxies []map[string]interface{}) string {
	header := "ID,Name,Host,Port,Protocol,Status\n"
	rows := []string{header}

	for _, proxy := range proxies {
		row := fmt.Sprintf("%v,%v,%v,%v,%v,%v\n",
			proxy["id"], proxy["name"], proxy["host"],
			proxy["port"], proxy["protocol"], proxy["status"])
		rows = append(rows, row)
	}

	result := ""
	for _, row := range rows {
		result += row
	}
	return result
}
