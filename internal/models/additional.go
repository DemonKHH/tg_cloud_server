package models

import "time"

// 补充模型定义，用于统一仓库接口

// Proxy 代理模型别名（用于仓库接口统一）
type Proxy = ProxyIP

// ProxyStats 代理统计信息（仓库接口版本）
type ProxyStats struct {
	Total    int64 `json:"total"`
	Active   int64 `json:"active"`
	Inactive int64 `json:"inactive"`
	Error    int64 `json:"error"`
	Testing  int64 `json:"testing"`
}

// TaskStats 任务统计信息（仓库接口版本）
type TaskStats struct {
	Total      int64 `json:"total"`
	Pending    int64 `json:"pending"`
	Running    int64 `json:"running"`
	Completed  int64 `json:"completed"`
	Failed     int64 `json:"failed"`
	Cancelled  int64 `json:"cancelled"`
	TodayTasks int64 `json:"today_tasks"`
}

// QueueInfo 队列信息（仓库接口版本）
type QueueInfo struct {
	AccountID         uint64 `json:"account_id"`
	PendingTasks      int64  `json:"pending_tasks"`
	RunningTasks      int64  `json:"running_tasks"`
	EstimatedWaitTime int64  `json:"estimated_wait_time"` // 秒
}

// BatchOperation 批量操作类型
type BatchOperation string

const (
	BatchOperationCreateAccounts BatchOperation = "create_accounts"
	BatchOperationUpdateAccounts BatchOperation = "update_accounts"
	BatchOperationDeleteAccounts BatchOperation = "delete_accounts"
	BatchOperationBindProxies    BatchOperation = "bind_proxies"
	BatchOperationCreateTasks    BatchOperation = "create_tasks"
	BatchOperationCancelTasks    BatchOperation = "cancel_tasks"
	BatchOperationImportUsers    BatchOperation = "import_users"
	BatchOperationExportData     BatchOperation = "export_data"
)

// BatchJobStatus 批量任务状态
type BatchJobStatus string

const (
	BatchJobStatusPending   BatchJobStatus = "pending"
	BatchJobStatusRunning   BatchJobStatus = "running"
	BatchJobStatusCompleted BatchJobStatus = "completed"
	BatchJobStatusFailed    BatchJobStatus = "failed"
	BatchJobStatusCancelled BatchJobStatus = "cancelled"
)

// BatchJob 批量任务
type BatchJob struct {
	ID             uint64                 `json:"id"`
	UserID         uint64                 `json:"user_id"`
	Operation      BatchOperation         `json:"operation"`
	Status         BatchJobStatus         `json:"status"`
	TotalItems     int                    `json:"total_items"`
	ProcessedItems int                    `json:"processed_items"`
	SuccessItems   int                    `json:"success_items"`
	FailedItems    int                    `json:"failed_items"`
	Progress       float64                `json:"progress"`
	ErrorMessages  []string               `json:"error_messages,omitempty"`
	Result         map[string]interface{} `json:"result,omitempty"`
	StartedAt      *time.Time             `json:"started_at,omitempty"`
	CompletedAt    *time.Time             `json:"completed_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}
