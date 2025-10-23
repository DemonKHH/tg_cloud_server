package models

import (
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// UserRole 用户角色枚举
type UserRole string

const (
	RoleAdmin    UserRole = "admin"    // 系统管理员
	RolePremium  UserRole = "premium"  // 高级用户
	RoleStandard UserRole = "standard" // 标准用户
)

// User 用户模型
type User struct {
	ID           uint64     `json:"id" gorm:"primaryKey;autoIncrement"`
	Username     string     `json:"username" gorm:"uniqueIndex;size:50;not null"`
	Email        string     `json:"email" gorm:"uniqueIndex;size:100"`
	PasswordHash string     `json:"-" gorm:"size:255;not null"` // 隐藏密码
	Role         UserRole   `json:"role" gorm:"type:enum('admin','premium','standard');default:'standard'"`
	IsActive     bool       `json:"is_active" gorm:"default:true"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`

	// 关联关系
	Accounts []TGAccount `json:"accounts" gorm:"foreignKey:UserID"`
	Tasks    []Task      `json:"tasks" gorm:"foreignKey:UserID"`
	ProxyIPs []ProxyIP   `json:"proxy_ips" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (User) TableName() string {
	return "users"
}

// SetPassword 设置密码（加密存储）
func (u *User) SetPassword(password string) error {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	u.PasswordHash = string(hashedPassword)
	return nil
}

// CheckPassword 验证密码
func (u *User) CheckPassword(password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(password))
	return err == nil
}

// IsAdmin 检查是否为管理员
func (u *User) IsAdmin() bool {
	return u.Role == RoleAdmin
}

// IsPremium 检查是否为高级用户
func (u *User) IsPremium() bool {
	return u.Role == RolePremium || u.Role == RoleAdmin
}

// HasPermission 检查用户权限
func (u *User) HasPermission(permission string) bool {
	switch permission {
	case "manage_users":
		return u.Role == RoleAdmin
	case "unlimited_accounts":
		return u.Role == RoleAdmin || u.Role == RolePremium
	case "advanced_features":
		return u.Role == RoleAdmin || u.Role == RolePremium
	case "basic_features":
		return u.IsActive
	default:
		return false
	}
}

// BeforeCreate 创建前钩子
func (u *User) BeforeCreate(tx *gorm.DB) error {
	u.Role = RoleStandard
	u.IsActive = true
	return nil
}

// UserProfile 用户资料（用于API返回）
type UserProfile struct {
	ID          uint64     `json:"id"`
	Username    string     `json:"username"`
	Email       string     `json:"email"`
	Role        UserRole   `json:"role"`
	IsActive    bool       `json:"is_active"`
	LastLoginAt *time.Time `json:"last_login_at"`
	CreatedAt   time.Time  `json:"created_at"`
	Stats       UserStats  `json:"stats"`
}

// UserStats 用户统计信息
type UserStats struct {
	AccountCount       int64 `json:"account_count"`
	ActiveAccountCount int64 `json:"active_account_count"`
	TaskCount          int64 `json:"task_count"`
	TasksToday         int64 `json:"tasks_today"`
	TasksThisWeek      int64 `json:"tasks_this_week"`
	ProxyCount         int64 `json:"proxy_count"`
}

// LoginRequest 登录请求
type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// RegisterRequest 注册请求
type RegisterRequest struct {
	Username string `json:"username" binding:"required,min=3,max=50"`
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
}

// UpdateProfileRequest 更新资料请求
type UpdateProfileRequest struct {
	Email    string `json:"email" binding:"omitempty,email"`
	Password string `json:"password" binding:"omitempty,min=6"`
}

// LoginResponse 登录响应
type LoginResponse struct {
	User        UserProfile `json:"user"`
	AccessToken string      `json:"access_token"`
	TokenType   string      `json:"token_type"`
	ExpiresIn   int64       `json:"expires_in"`
}
