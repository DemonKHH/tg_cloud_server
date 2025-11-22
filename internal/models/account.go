package models

import (
	"time"

	"gorm.io/gorm"
)

// AccountStatus 账号状态枚举
type AccountStatus string

const (
	AccountStatusNew         AccountStatus = "new"         // 新建
	AccountStatusNormal      AccountStatus = "normal"      // 正常
	AccountStatusWarning     AccountStatus = "warning"     // 警告
	AccountStatusRestricted  AccountStatus = "restricted"  // 限制
	AccountStatusDead        AccountStatus = "dead"        // 死亡
	AccountStatusCooling     AccountStatus = "cooling"     // 冷却
	AccountStatusMaintenance AccountStatus = "maintenance" // 维护
)

// ConnectionStatus 连接状态枚举
type ConnectionStatus int

const (
	StatusDisconnected    ConnectionStatus = iota // 断开连接
	StatusConnecting                              // 连接中
	StatusConnected                               // 已连接
	StatusReconnecting                            // 重连中
	StatusConnectionError                         // 连接错误
)

// String 返回连接状态字符串
func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusReconnecting:
		return "reconnecting"
	case StatusConnectionError:
		return "error"
	default:
		return "unknown"
	}
}

// TGAccount TG账号模型
type TGAccount struct {
	ID          uint64        `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64        `json:"user_id" gorm:"not null;index"`
	Phone       string        `json:"phone" gorm:"uniqueIndex;size:20;not null"`
	SessionData string        `json:"-" gorm:"type:text"` // 隐藏敏感数据
	ProxyID     *uint64       `json:"proxy_id" gorm:"index"`
	Status      AccountStatus `json:"status" gorm:"type:enum('new','normal','warning','restricted','dead','cooling','maintenance');default:'new'"`
	HealthScore float64       `json:"health_score" gorm:"type:decimal(3,2);default:1.00"`
	LastCheckAt *time.Time    `json:"last_check_at"`
	LastUsedAt  *time.Time    `json:"last_used_at"`
	CreatedAt   time.Time     `json:"created_at"`
	UpdatedAt   time.Time     `json:"updated_at"`

	// 关联关系
	User    User     `json:"user" gorm:"foreignKey:UserID"`
	ProxyIP *ProxyIP `json:"proxy_ip" gorm:"foreignKey:ProxyID"`
	Tasks   []Task   `json:"tasks" gorm:"foreignKey:AccountID"`
}

// TableName 指定表名
func (TGAccount) TableName() string {
	return "tg_accounts"
}

// IsAvailable 检查账号是否可用
func (a *TGAccount) IsAvailable() bool {
	return a.Status != AccountStatusDead &&
		a.Status != AccountStatusCooling &&
		a.Status != AccountStatusMaintenance
}

// NeedsAttention 检查账号是否需要关注
func (a *TGAccount) NeedsAttention() bool {
	return a.Status == AccountStatusWarning ||
		a.Status == AccountStatusRestricted ||
		a.HealthScore < 0.3
}

// GetStatusColor 获取状态颜色（用于前端显示）
func (a *TGAccount) GetStatusColor() string {
	switch a.Status {
	case AccountStatusNormal:
		return "green"
	case AccountStatusWarning:
		return "orange"
	case AccountStatusRestricted:
		return "red"
	case AccountStatusDead:
		return "black"
	case AccountStatusCooling:
		return "blue"
	case AccountStatusMaintenance:
		return "gray"
	default:
		return "purple"
	}
}

// BeforeCreate 创建前钩子
func (a *TGAccount) BeforeCreate(tx *gorm.DB) error {
	a.Status = AccountStatusNew
	a.HealthScore = 1.0
	return nil
}

// AccountSummary 账号摘要信息（用于列表显示）
type AccountSummary struct {
	ID          uint64        `json:"id"`
	Phone       string        `json:"phone"`
	Status      AccountStatus `json:"status"`
	HealthScore float64       `json:"health_score"`
	ProxyID     *uint64       `json:"proxy_id,omitempty"`
	LastUsedAt  *time.Time    `json:"last_used_at,omitempty"`
	LastCheckAt *time.Time    `json:"last_check_at,omitempty"`
	CreatedAt   time.Time     `json:"created_at"`
	TaskCount   int64         `json:"task_count,omitempty"`
	ProxyName   string        `json:"proxy_name,omitempty"`
}

// AccountAvailability 账号可用性信息
type AccountAvailability struct {
	AccountID        uint64           `json:"account_id"`
	Status           AccountStatus    `json:"status"`
	HealthScore      float64          `json:"health_score"`
	QueueSize        int              `json:"queue_size"`
	IsTaskRunning    bool             `json:"is_task_running"`
	ConnectionStatus ConnectionStatus `json:"connection_status"`
	LastUsed         *time.Time       `json:"last_used"`
	Recommendation   string           `json:"recommendation"`
	Warnings         []string         `json:"warnings"`
	Errors           []string         `json:"errors"`
}

// ValidationResult 账号验证结果
type ValidationResult struct {
	AccountID   uint64   `json:"account_id"`
	IsValid     bool     `json:"is_valid"`
	Warnings    []string `json:"warnings"`
	Errors      []string `json:"errors"`
	QueueSize   int      `json:"queue_size"`
	HealthScore float64  `json:"health_score"`
}

// CreateAccountRequest 创建账号请求
type CreateAccountRequest struct {
	Phone       string  `json:"phone" binding:"required"`
	SessionData string  `json:"session_data" binding:"required"`
	ProxyID     *uint64 `json:"proxy_id"`
}

// BatchUploadAccountRequest 批量上传账号请求
type BatchUploadAccountRequest struct {
	Accounts []AccountUploadItem `json:"accounts" binding:"required,min=1"`
	ProxyID  *uint64             `json:"proxy_id"`
}

// AccountUploadItem 单个账号上传项
type AccountUploadItem struct {
	Phone       string `json:"phone" binding:"required"`
	SessionData string `json:"session_data" binding:"required"`
}

// UpdateAccountRequest 更新账号请求
type UpdateAccountRequest struct {
	Phone   string         `json:"phone"`
	Status  *AccountStatus `json:"status"`
	ProxyID *uint64        `json:"proxy_id"`
}

// AccountHealthReport 账号健康报告
type AccountHealthReport struct {
	AccountID    uint64                 `json:"account_id"`
	Phone        string                 `json:"phone"`
	HealthScore  float64                `json:"health_score"`
	Score        float64                `json:"score"` // 别名字段用于兼容
	Status       AccountStatus          `json:"status"`
	LastCheckAt  *time.Time             `json:"last_check_at"`
	CheckedAt    *time.Time             `json:"checked_at"` // 别名字段用于兼容
	Issues       []string               `json:"issues"`
	Suggestions  []string               `json:"suggestions"`
	CheckResults map[string]interface{} `json:"check_results"`
	GeneratedAt  time.Time              `json:"generated_at"`
}

// PaginationResponse 分页响应
type PaginationResponse struct {
	Total       int64       `json:"total"`
	Page        int         `json:"page"`
	Limit       int         `json:"limit"`
	TotalPages  int         `json:"total_pages"`
	HasNext     bool        `json:"has_next"`
	HasPrevious bool        `json:"has_previous"`
	Data        interface{} `json:"data"`
}

// RiskLog 风控日志
type RiskLog struct {
	ID        uint64    `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID    uint64    `json:"user_id" gorm:"not null;index"`
	AccountID *uint64   `json:"account_id" gorm:"index"`
	TaskID    *uint64   `json:"task_id" gorm:"index"`
	Level     string    `json:"level" gorm:"type:enum('low','medium','high','critical');not null"`
	Event     string    `json:"event" gorm:"size:100;not null"`
	Message   string    `json:"message" gorm:"type:text"`
	Data      string    `json:"data" gorm:"type:json"`
	CreatedAt time.Time `json:"created_at"`
}

// UpdateTaskRequest 更新任务请求
type UpdateTaskRequest struct {
	Status     *TaskStatus `json:"status"`
	Priority   int         `json:"priority,omitempty"`
	Config     TaskConfig  `json:"config"`
	Result     TaskResult  `json:"result"`
	ScheduleAt *time.Time  `json:"schedule_at,omitempty"`
}
