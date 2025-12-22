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
	AccountStatusFrozen      AccountStatus = "frozen"      // 冻结
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
	Status      AccountStatus `json:"status" gorm:"type:enum('new','normal','warning','restricted','dead','cooling','maintenance','frozen');default:'new'"`
	IsOnline    bool          `json:"is_online" gorm:"default:false"` // 是否在线

	// Telegram 账号信息（从 Telegram 获取并存储）
	TgUserID  *int64  `json:"tg_user_id" gorm:"index"`        // Telegram 用户ID
	Username  *string `json:"username" gorm:"size:100;index"` // Telegram 用户名
	FirstName *string `json:"first_name" gorm:"size:100"`     // 名字
	LastName  *string `json:"last_name" gorm:"size:100"`      // 姓氏
	Bio       *string `json:"bio" gorm:"type:text"`           // 个人简介
	PhotoURL  *string `json:"photo_url" gorm:"size:500"`      // 头像URL

	// 2FA 信息
	Has2FA        bool   `json:"has_2fa" gorm:"column:has_2fa;default:false"`               // 是否开启2FA
	TwoFAPassword string `json:"two_fa_password" gorm:"column:two_fa_password;size:100"`    // 2FA密码
	Is2FACorrect  bool   `json:"is_2fa_correct" gorm:"column:is_2fa_correct;default:false"` // 2FA密码是否正确

	// 双向限制状态（独立字段，可与其他状态同时存在）
	IsBidirectional bool    `json:"is_bidirectional" gorm:"default:false"`            // 是否双向限制
	FrozenUntil     *string `json:"frozen_until" gorm:"column:frozen_until;size:100"` // 冻结结束时间

	// 风控字段
	ConsecutiveFailures uint32     `json:"consecutive_failures" gorm:"default:0"` // 连续失败次数
	CoolingUntil        *time.Time `json:"cooling_until"`                         // 冷却结束时间

	LastCheckAt *time.Time `json:"last_check_at"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`

	// 关联关系
	User    User     `json:"user" gorm:"foreignKey:UserID"`
	ProxyIP *ProxyIP `json:"proxy_ip" gorm:"foreignKey:ProxyID"`
}

// TableName 指定表名
func (TGAccount) TableName() string {
	return "tg_accounts"
}

// IsAvailable 检查账号是否可用
func (a *TGAccount) IsAvailable() bool {
	return a.Status != AccountStatusDead &&
		a.Status != AccountStatusCooling &&
		a.Status != AccountStatusMaintenance &&
		a.Status != AccountStatusFrozen
}

// NeedsAttention 检查账号是否需要关注
func (a *TGAccount) NeedsAttention() bool {
	return a.Status == AccountStatusWarning ||
		a.Status == AccountStatusRestricted ||
		a.Status == AccountStatusFrozen ||
		a.IsBidirectional
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
	case AccountStatusFrozen:
		return "red"
	default:
		return "purple"
	}
}

// BeforeCreate 创建前钩子
func (a *TGAccount) BeforeCreate(tx *gorm.DB) error {
	a.Status = AccountStatusNew
	return nil
}

// AccountSummary 账号摘要信息（用于列表显示）
type AccountSummary struct {
	ID       uint64        `json:"id"`
	Phone    string        `json:"phone"`
	Status   AccountStatus `json:"status"`
	IsOnline bool          `json:"is_online"`
	ProxyID  *uint64       `json:"proxy_id,omitempty"`

	// 双向限制状态（独立字段）
	IsBidirectional bool    `json:"is_bidirectional"`
	FrozenUntil     *string `json:"frozen_until,omitempty" gorm:"column:frozen_until"`

	// 2FA 信息
	Has2FA        bool   `json:"has_2fa" gorm:"column:has_2fa"`
	TwoFAPassword string `json:"two_fa_password,omitempty" gorm:"column:two_fa_password"`

	// 风控字段
	ConsecutiveFailures uint32     `json:"consecutive_failures"`
	CoolingUntil        *time.Time `json:"cooling_until,omitempty"`

	// Telegram 信息（始终返回，即使为空）
	TgUserID  *int64  `json:"tg_user_id"`
	Username  *string `json:"username"`
	FirstName *string `json:"first_name"`
	LastName  *string `json:"last_name"`
	Bio       *string `json:"bio"`
	PhotoURL  *string `json:"photo_url"`

	LastUsedAt  *time.Time `json:"last_used_at,omitempty"`
	LastCheckAt *time.Time `json:"last_check_at,omitempty"`
	CreatedAt   time.Time  `json:"created_at"`
	TaskCount   int64      `json:"task_count,omitempty"`
	ProxyName   string     `json:"proxy_name,omitempty"`

	// 代理详情
	ProxyIP       string `json:"proxy_ip,omitempty"`
	ProxyPort     int    `json:"proxy_port,omitempty"`
	ProxyUsername string `json:"proxy_username,omitempty"`
	ProxyPassword string `json:"proxy_password,omitempty"`
	ProxyProtocol string `json:"proxy_protocol,omitempty"`
}

// AccountAvailability 账号可用性信息
type AccountAvailability struct {
	AccountID        uint64           `json:"account_id"`
	Status           AccountStatus    `json:"status"`
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
	AccountID uint64   `json:"account_id"`
	IsValid   bool     `json:"is_valid"`
	Warnings  []string `json:"warnings"`
	Errors    []string `json:"errors"`
	QueueSize int      `json:"queue_size"`
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

// BatchSet2FARequest 批量设置2FA密码请求（仅更新本地记录）
type BatchSet2FARequest struct {
	AccountIDs []uint64 `json:"account_ids" binding:"required,min=1"`
	Password   string   `json:"password" binding:"required"`
}

// BatchUpdate2FARequest 批量修改2FA密码请求（尝试修改Telegram密码）
type BatchUpdate2FARequest struct {
	AccountIDs  []uint64 `json:"account_ids" binding:"required,min=1"`
	OldPassword string   `json:"old_password"` // 如果为空，尝试使用本地存储的密码
	NewPassword string   `json:"new_password" binding:"required"`
}

// BatchDeleteAccountsRequest 批量删除账号请求
type BatchDeleteAccountsRequest struct {
	AccountIDs []uint64 `json:"account_ids" binding:"required,min=1"`
}

// BatchBindProxyRequest 批量绑定/解绑代理请求
type BatchBindProxyRequest struct {
	AccountIDs []uint64 `json:"account_ids" binding:"required,min=1"`
	ProxyID    *uint64  `json:"proxy_id"` // nil表示解绑代理
}

// ExportAccountsRequest 导出账号请求
type ExportAccountsRequest struct {
	AccountIDs []uint64 `json:"account_ids" binding:"required,min=1"`
}

// AccountHealthReport 账号健康报告
type AccountHealthReport struct {
	AccountID    uint64                 `json:"account_id"`
	Phone        string                 `json:"phone"`
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

// TelegramAccountInfo Telegram 账号信息（用于更新）
type TelegramAccountInfo struct {
	TgUserID  *int64  `json:"tg_user_id,omitempty"`
	Phone     *string `json:"phone,omitempty"`
	Username  *string `json:"username,omitempty"`
	FirstName *string `json:"first_name,omitempty"`
	LastName  *string `json:"last_name,omitempty"`
	Bio       *string `json:"bio,omitempty"`
	PhotoURL  *string `json:"photo_url,omitempty"`
}
