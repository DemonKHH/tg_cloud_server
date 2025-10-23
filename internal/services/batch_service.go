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
	// TODO: 实现批量绑定代理
	return nil, fmt.Errorf("batch bind proxies not implemented yet")
}

func (s *batchService) BatchCancelTasks(ctx context.Context, userID uint64, taskIDs []uint64) (*BatchJob, error) {
	// TODO: 实现批量取消任务
	return nil, fmt.Errorf("batch cancel tasks not implemented yet")
}

func (s *batchService) ImportUsers(ctx context.Context, userID uint64, req *ImportUsersRequest) (*BatchJob, error) {
	// TODO: 实现用户导入
	return nil, fmt.Errorf("import users not implemented yet")
}

func (s *batchService) ExportData(ctx context.Context, userID uint64, req *ExportDataRequest) (*BatchJob, error) {
	// TODO: 实现数据导出
	return nil, fmt.Errorf("export data not implemented yet")
}
