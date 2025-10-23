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
	StatusDisconnected ConnectionStatus = iota // 断开连接
	StatusConnecting                           // 连接中
	StatusConnected                            // 已连接
	StatusReconnecting                         // 重连中
	StatusError                                // 错误
)

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
	LastCheckAt *time.Time    `json:"last_check_at"`
	TaskCount   int64         `json:"task_count"`
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
