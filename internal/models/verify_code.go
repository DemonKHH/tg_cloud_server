package models

import (
	"time"
)

// VerifyCodeSession 验证码会话
type VerifyCodeSession struct {
	Code      string    `json:"code"`       // 临时访问代码
	AccountID uint64    `json:"account_id"` // 关联的账号ID
	UserID    uint64    `json:"user_id"`    // 用户ID
	CreatedAt time.Time `json:"created_at"` // 创建时间
	ExpiresAt time.Time `json:"expires_at"` // 过期时间
}

// IsExpired 检查是否已过期
func (s *VerifyCodeSession) IsExpired() bool {
	return time.Now().After(s.ExpiresAt)
}

// IsValid 检查会话是否有效（只检查过期时间）
func (s *VerifyCodeSession) IsValid() bool {
	return !s.IsExpired()
}

// GenerateCodeRequest 生成验证码访问链接请求
type GenerateCodeRequest struct {
	AccountID uint64 `json:"account_id" binding:"required"`
	ExpiresIn int    `json:"expires_in,omitempty"` // 过期时间(秒)，默认300秒(5分钟)
}

// GenerateCodeResponse 生成验证码访问链接响应
type GenerateCodeResponse struct {
	Code      string `json:"code"`       // 临时访问代码
	URL       string `json:"url"`        // 完整的访问链接
	ExpiresAt int64  `json:"expires_at"` // 过期时间戳
	ExpiresIn int    `json:"expires_in"` // 过期时间(秒)
}

// VerifyCodeResponse 验证码响应
type VerifyCodeResponse struct {
	Success     bool   `json:"success"`               // 是否成功获取到验证码
	Code        string `json:"code,omitempty"`        // 验证码
	Message     string `json:"message"`               // 消息
	Sender      string `json:"sender,omitempty"`      // 发送者
	ReceivedAt  int64  `json:"received_at,omitempty"` // 接收时间戳
	WaitSeconds int    `json:"wait_seconds"`          // 等待时间(秒)
}

// VerifyCodeError 验证码错误类型
type VerifyCodeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *VerifyCodeError) Error() string {
	return e.Message
}

// 预定义错误
var (
	ErrCodeNotFound = &VerifyCodeError{
		Code:    "CODE_NOT_FOUND",
		Message: "验证码访问链接无效或已过期",
	}
	ErrCodeExpired = &VerifyCodeError{
		Code:    "CODE_EXPIRED",
		Message: "验证码访问链接已过期",
	}
	ErrAccountNotFound = &VerifyCodeError{
		Code:    "ACCOUNT_NOT_FOUND",
		Message: "关联的TG账号不存在或无法访问",
	}
	ErrVerifyTimeout = &VerifyCodeError{
		Code:    "VERIFY_TIMEOUT",
		Message: "验证码接收超时，请稍后重试",
	}
	ErrTelegramConnection = &VerifyCodeError{
		Code:    "TELEGRAM_CONNECTION_ERROR",
		Message: "Telegram连接失败，请检查账号状态",
	}
)
