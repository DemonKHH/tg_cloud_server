package models

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"gorm.io/gorm"
)

// TaskType 任务类型枚举
type TaskType string

const (
	TaskTypeCheck             TaskType = "check"              // 账号检查
	TaskTypePrivate           TaskType = "private_message"    // 私信发送
	TaskTypeBroadcast         TaskType = "broadcast"          // 群发消息
	TaskTypeVerify            TaskType = "verify_code"        // 验证码接收
	TaskTypeGroupChat         TaskType = "group_chat"         // AI炒群
	TaskTypeJoinGroup         TaskType = "join_group"         // 批量加群
	TaskTypeScenario          TaskType = "scenario"           // 智能体场景炒群
	TaskTypeForceAdd          TaskType = "force_add_group"    // 强拉进群
	TaskTypeTerminateSessions TaskType = "terminate_sessions" // 踢出其他设备
	TaskTypeUpdate2FA         TaskType = "update_2fa"         // 修改2FA密码
)

// TaskStatus 任务状态枚举
type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"   // 待执行
	TaskStatusQueued    TaskStatus = "queued"    // 已排队
	TaskStatusRunning   TaskStatus = "running"   // 执行中
	TaskStatusPaused    TaskStatus = "paused"    // 已暂停
	TaskStatusCompleted TaskStatus = "completed" // 已完成
	TaskStatusFailed    TaskStatus = "failed"    // 失败
	TaskStatusCancelled TaskStatus = "cancelled" // 已取消
)

// Task 任务模型
type Task struct {
	ID          uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64     `json:"user_id" gorm:"not null;index"`
	AccountIDs  string     `json:"account_ids" gorm:"type:text;not null"` // 账号ID列表（逗号分隔，如 "1,2,3"）
	TaskType    TaskType   `json:"task_type" gorm:"type:enum('check','private_message','broadcast','verify_code','group_chat','join_group','scenario','force_add_group','terminate_sessions','update_2fa');not null"`
	Status      TaskStatus `json:"status" gorm:"type:enum('pending','queued','running', 'paused', 'completed','failed','cancelled');default:'pending'"`
	Priority    int        `json:"priority" gorm:"default:5"` // 优先级 1-10
	Config      TaskConfig `json:"config" gorm:"type:json"`   // 任务配置（JSON格式）
	Result      TaskResult `json:"result" gorm:"type:json"`   // 执行结果（JSON格式）
	ScheduledAt *time.Time `json:"scheduled_at"`              // 计划执行时间
	StartedAt   *time.Time `json:"started_at"`                // 开始执行时间
	CompletedAt *time.Time `json:"completed_at"`              // 完成时间
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 关联关系
	User User      `json:"user" gorm:"foreignKey:UserID"`
	Logs []TaskLog `json:"logs" gorm:"foreignKey:TaskID"`
}

// GetAccountIDList 获取账号ID列表
func (t *Task) GetAccountIDList() []uint64 {
	if t.AccountIDs == "" {
		return []uint64{}
	}

	ids := []uint64{}
	parts := strings.Split(t.AccountIDs, ",")
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.ParseUint(part, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}

	return ids
}

// SetAccountIDList 设置账号ID列表
func (t *Task) SetAccountIDList(ids []uint64) {
	if len(ids) == 0 {
		t.AccountIDs = ""
		return
	}

	// 将所有账号ID转换为逗号分隔的字符串
	strIDs := make([]string, len(ids))
	for i, id := range ids {
		strIDs[i] = strconv.FormatUint(id, 10)
	}
	t.AccountIDs = strings.Join(strIDs, ",")
}

// GetFirstAccountID 获取第一个账号ID（用于显示）
func (t *Task) GetFirstAccountID() uint64 {
	ids := t.GetAccountIDList()
	if len(ids) > 0 {
		return ids[0]
	}
	return 0
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
func (tc TaskConfig) Value() (driver.Value, error) {
	if len(tc) == 0 {
		return []byte("{}"), nil
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
func (tr TaskResult) Value() (driver.Value, error) {
	if len(tr) == 0 {
		return []byte("{}"), nil
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
	// 确保 Config 不为 nil，如果是 nil 则初始化为空 map
	if t.Config == nil {
		t.Config = make(TaskConfig)
	}
	// 确保 Result 不为 nil，如果是 nil 则初始化为空 map
	if t.Result == nil {
		t.Result = make(TaskResult)
	}
	return nil
}

// TaskLog 任务执行日志模型
type TaskLog struct {
	ID        uint64          `json:"id" gorm:"primaryKey;autoIncrement"`
	TaskID    uint64          `json:"task_id" gorm:"not null;index:idx_task_created"`
	AccountID *uint64         `json:"account_id" gorm:"index"`
	Level     string          `json:"level" gorm:"size:10;default:'info'"`
	Action    string          `json:"action" gorm:"size:50;not null"`
	Message   string          `json:"message" gorm:"type:text"`
	ExtraData json.RawMessage `json:"extra_data" gorm:"type:json"`
	CreatedAt time.Time       `json:"created_at" gorm:"index:idx_task_created"`

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
	AccountIDs []uint64   `json:"account_ids" binding:"required,min=1"` // 账号ID列表
	TaskType   TaskType   `json:"task_type" binding:"required"`
	Config     TaskConfig `json:"task_config"`
	Priority   int        `json:"priority,omitempty"`
	ScheduleAt *time.Time `json:"schedule_at,omitempty"`
	AutoStart  bool       `json:"auto_start"` // 是否自动开始执行，默认false
}

// Validate 验证请求
func (r *CreateTaskRequest) Validate() error {
	if len(r.AccountIDs) == 0 {
		return fmt.Errorf("至少需要指定一个账号")
	}
	return nil
}

// TaskSummary 任务摘要信息
type TaskSummary struct {
	ID           uint64     `json:"id"`
	TaskType     TaskType   `json:"task_type"`
	Status       TaskStatus `json:"status"`
	AccountPhone string     `json:"account_phone"` // 显示账号信息（如 "1个账号" 或 "3个账号"）
	Priority     int        `json:"priority"`
	Config       TaskConfig `json:"config"` // 任务配置
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

// BatchDeleteRequest 批量删除请求
type BatchDeleteRequest struct {
	TaskIDs []uint64 `json:"task_ids" binding:"required"`
}

// CleanupRequest 清理请求
type CleanupRequest struct {
	OlderThanDays int `json:"older_than_days" binding:"required,min=1"`
}

// TaskControlRequest 任务控制请求
type TaskControlRequest struct {
	Action string `json:"action" binding:"required,oneof=start pause stop resume"`
}

// BatchTaskControlRequest 批量任务控制请求
type BatchTaskControlRequest struct {
	TaskIDs []uint64 `json:"task_ids" binding:"required"`
	Action  string   `json:"action" binding:"required,oneof=start pause stop resume cancel"`
}
