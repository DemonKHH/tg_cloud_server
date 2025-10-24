package models

import (
	"encoding/json"
	"time"

	"gorm.io/gorm"
)

// TaskType 任务类型枚举
type TaskType string

const (
	TaskTypeCheck     TaskType = "check"           // 账号检查
	TaskTypePrivate   TaskType = "private_message" // 私信发送
	TaskTypeBroadcast TaskType = "broadcast"       // 群发消息
	TaskTypeVerify    TaskType = "verify_code"     // 验证码接收
	TaskTypeGroupChat TaskType = "group_chat"      // AI炒群
)

// TaskStatus 任务状态枚举
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待执行
	TaskStatusQueued    TaskStatus = "queued"    // 已排队
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// Task 任务模型
type Task struct {
	ID          uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64     `json:"user_id" gorm:"not null;index"`
	AccountID   uint64     `json:"account_id" gorm:"not null;index"` // 用户指定的执行账号
	TaskType    TaskType   `json:"task_type" gorm:"type:enum('check','private_message','broadcast','verify_code','group_chat');not null"`
	Status      TaskStatus `json:"status" gorm:"type:enum('pending','queued','running','completed','failed','cancelled');default:'pending'"`
	Priority    int        `json:"priority" gorm:"default:5"` // 优先级 1-10
	Config      TaskConfig `json:"config" gorm:"type:json"`   // 任务配置（JSON格式）
	Result      TaskResult `json:"result" gorm:"type:json"`   // 执行结果（JSON格式）
	ScheduledAt *time.Time `json:"scheduled_at"`              // 计划执行时间
	StartedAt   *time.Time `json:"started_at"`                // 开始执行时间
	CompletedAt *time.Time `json:"completed_at"`              // 完成时间
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 关联关系
	User    User      `json:"user" gorm:"foreignKey:UserID"`
	Account TGAccount `json:"account" gorm:"foreignKey:AccountID"`
	Logs    []TaskLog `json:"logs" gorm:"foreignKey:TaskID"`
}

// TableName 指定表名
func (Task) TableName() string {
	return "tasks"
}

// TaskConfig 任务配置接口
type TaskConfig map[string]interface{}

// TaskResult 任务结果
type TaskResult map[string]interface{}

// Scan 实现 sql.Scanner 接口
func (tc *TaskConfig) Scan(value interface{}) error {
	if value == nil {
		*tc = make(TaskConfig)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, tc)
}

// Value 实现 driver.Valuer 接口
func (tc TaskConfig) Value() (interface{}, error) {
	if tc == nil {
		return nil, nil
	}
	return json.Marshal(tc)
}

// Scan 实现 sql.Scanner 接口
func (tr *TaskResult) Scan(value interface{}) error {
	if value == nil {
		*tr = make(TaskResult)
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return nil
	}

	return json.Unmarshal(bytes, tr)
}

// Value 实现 driver.Valuer 接口
func (tr TaskResult) Value() (interface{}, error) {
	if tr == nil {
		return nil, nil
	}
	return json.Marshal(tr)
}

// IsCompleted 检查任务是否已完成
func (t *Task) IsCompleted() bool {
	return t.Status == TaskStatusCompleted ||
		t.Status == TaskStatusFailed ||
		t.Status == TaskStatusCancelled
}

// IsRunning 检查任务是否正在执行
func (t *Task) IsRunning() bool {
	return t.Status == TaskStatusRunning
}

// CanCancel 检查任务是否可以取消
func (t *Task) CanCancel() bool {
	return t.Status == TaskStatusPending || t.Status == TaskStatusQueued
}

// GetDuration 获取任务执行时长
func (t *Task) GetDuration() *time.Duration {
	if t.StartedAt == nil {
		return nil
	}

	endTime := time.Now()
	if t.CompletedAt != nil {
		endTime = *t.CompletedAt
	}

	duration := endTime.Sub(*t.StartedAt)
	return &duration
}

// BeforeCreate 创建前钩子
func (t *Task) BeforeCreate(tx *gorm.DB) error {
	t.Status = TaskStatusPending
	if t.Priority == 0 {
		t.Priority = 5
	}
	return nil
}

// TaskLog 任务执行日志模型
type TaskLog struct {
	ID        uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID    uint64          `json:"task_id" gorm:"not null;index"`
	AccountID *uint64         `json:"account_id" gorm:"index"`
	Action    string          `json:"action" gorm:"size:50;not null"`
	Message   string          `json:"message" gorm:"type:text"`
	ExtraData json.RawMessage `json:"extra_data" gorm:"type:json"`
	CreatedAt time.Time       `json:"created_at"`

	// 关联关系
	Task    Task       `json:"task" gorm:"foreignKey:TaskID"`
	Account *TGAccount `json:"account" gorm:"foreignKey:AccountID"`
}

// TableName 指定表名
func (TaskLog) TableName() string {
	return "task_logs"
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	AccountID  uint64     `json:"account_id" binding:"required"`
	TaskType   TaskType   `json:"task_type" binding:"required"`
	Config     TaskConfig `json:"task_config"`
	Priority   int        `json:"priority,omitempty"`
	ScheduleAt *time.Time `json:"schedule_at,omitempty"`
}

// TaskSummary 任务摘要信息
type TaskSummary struct {
	ID           uint64     `json:"id"`
	TaskType     TaskType   `json:"task_type"`
	Status       TaskStatus `json:"status"`
	AccountID    uint64     `json:"account_id"`
	AccountPhone string     `json:"account_phone"`
	Priority     int        `json:"priority"`
	CreatedAt    time.Time  `json:"created_at"`
	StartedAt    *time.Time `json:"started_at"`
	CompletedAt  *time.Time `json:"completed_at"`
	Duration     string     `json:"duration,omitempty"`
}

// AccountQueueInfo 账号队列详细信息
type AccountQueueInfo struct {
	AccountID      uint64     `json:"account_id"`
	PendingCount   int        `json:"pending_count"`
	RunningCount   int        `json:"running_count"`
	CompletedCount int        `json:"completed_count"`
	FailedCount    int        `json:"failed_count"`
	LastTaskAt     *time.Time `json:"last_task_at"`
}

// TaskStatistics 任务详细统计
type TaskStatistics struct {
	TotalTasks    int64            `json:"total_tasks"`
	TasksByStatus map[string]int64 `json:"tasks_by_status"`
	TasksByType   map[string]int64 `json:"tasks_by_type"`
	SuccessRate   float64          `json:"success_rate"`
	AvgDuration   float64          `json:"avg_duration_seconds"`
	TasksToday    int64            `json:"tasks_today"`
	TasksThisWeek int64            `json:"tasks_this_week"`
}

// TaskAnalytics 任务分析数据
type TaskAnalytics struct {
	Period         string           `json:"period"`
	TotalTasks     int64            `json:"total_tasks"`
	Completed      int64            `json:"completed"`
	Failed         int64            `json:"failed"`
	Cancelled      int64            `json:"cancelled"`
	Running        int64            `json:"running"`
	Pending        int64            `json:"pending"`
	SuccessRate    float64          `json:"success_rate"`
	TasksByType    map[string]int64 `json:"tasks_by_type,omitempty"`
	TasksByAccount map[uint64]int64 `json:"tasks_by_account,omitempty"`
	GeneratedAt    time.Time        `json:"generated_at"`
}

// SchedulingOptimization 调度优化建议
type SchedulingOptimization struct {
	UserID          uint64    `json:"user_id"`
	TotalLoad       int64     `json:"total_load"`
	TotalCapacity   int64     `json:"total_capacity"`
	UtilizationRate float64   `json:"utilization_rate"`
	Recommendations []string  `json:"recommendations"`
	GeneratedAt     time.Time `json:"generated_at"`
}

// BatchCancelRequest 批量取消请求
type BatchCancelRequest struct {
	TaskIDs []uint64 `json:"task_ids" binding:"required"`
}

// CleanupRequest 清理请求
type CleanupRequest struct {
	OlderThanDays int `json:"older_than_days" binding:"required,min=1"`
}
