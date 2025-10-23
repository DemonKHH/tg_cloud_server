package errors

import (
	"fmt"
	"net/http"
)

// APIError 统一API错误结构
type APIError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

func (e *APIError) Error() string {
	return fmt.Sprintf("API Error %d: %s", e.Code, e.Message)
}

// 预定义错误码
const (
	// 通用错误码
	ErrCodeInternalServer   = 50001
	ErrCodeInvalidParameter = 40001
	ErrCodeUnauthorized     = 40101
	ErrCodeForbidden        = 40301
	ErrCodeNotFound         = 40401
	ErrCodeConflict         = 40901
	ErrCodeTooManyRequests  = 42901

	// 用户相关错误码
	ErrCodeUserExists      = 40001
	ErrCodeUserNotFound    = 40404
	ErrCodeInvalidPassword = 40102
	ErrCodeTokenExpired    = 40103
	ErrCodeTokenInvalid    = 40104

	// 账号相关错误码
	ErrCodeAccountExists      = 41001
	ErrCodeAccountNotFound    = 41404
	ErrCodeAccountUnavailable = 41001
	ErrCodeAccountBlocked     = 41002
	ErrCodeAccountDead        = 41003

	// 任务相关错误码
	ErrCodeTaskNotFound  = 42404
	ErrCodeTaskRunning   = 42001
	ErrCodeTaskCompleted = 42002
	ErrCodeTaskFailed    = 42003
	ErrCodeTaskCancelled = 42004

	// 代理相关错误码
	ErrCodeProxyNotFound    = 43404
	ErrCodeProxyUnavailable = 43001
	ErrCodeProxyTestFailed  = 43002

	// Telegram相关错误码
	ErrCodeTelegramAuth      = 44001
	ErrCodeTelegramAPI       = 44002
	ErrCodeTelegramTimeout   = 44003
	ErrCodeTelegramRateLimit = 44004
)

// 错误构造函数
func New(code int, message string, details ...string) *APIError {
	err := &APIError{
		Code:    code,
		Message: message,
	}
	if len(details) > 0 {
		err.Details = details[0]
	}
	return err
}

// 通用错误
func InternalServerError(details ...string) *APIError {
	return New(ErrCodeInternalServer, "Internal server error", details...)
}

func InvalidParameter(message string, details ...string) *APIError {
	return New(ErrCodeInvalidParameter, message, details...)
}

func Unauthorized(details ...string) *APIError {
	return New(ErrCodeUnauthorized, "Unauthorized", details...)
}

func Forbidden(details ...string) *APIError {
	return New(ErrCodeForbidden, "Forbidden", details...)
}

func NotFound(resource string, details ...string) *APIError {
	return New(ErrCodeNotFound, fmt.Sprintf("%s not found", resource), details...)
}

func Conflict(message string, details ...string) *APIError {
	return New(ErrCodeConflict, message, details...)
}

func TooManyRequests(details ...string) *APIError {
	return New(ErrCodeTooManyRequests, "Too many requests", details...)
}

// 用户相关错误
func UserExists() *APIError {
	return New(ErrCodeUserExists, "User already exists")
}

func UserNotFound() *APIError {
	return New(ErrCodeUserNotFound, "User not found")
}

func InvalidPassword() *APIError {
	return New(ErrCodeInvalidPassword, "Invalid password")
}

func TokenExpired() *APIError {
	return New(ErrCodeTokenExpired, "Token expired")
}

func TokenInvalid() *APIError {
	return New(ErrCodeTokenInvalid, "Invalid token")
}

// 账号相关错误
func AccountExists() *APIError {
	return New(ErrCodeAccountExists, "Account already exists")
}

func AccountNotFound() *APIError {
	return New(ErrCodeAccountNotFound, "Account not found")
}

func AccountUnavailable(reason string) *APIError {
	return New(ErrCodeAccountUnavailable, "Account unavailable", reason)
}

func AccountBlocked() *APIError {
	return New(ErrCodeAccountBlocked, "Account is blocked")
}

func AccountDead() *APIError {
	return New(ErrCodeAccountDead, "Account is dead")
}

// 任务相关错误
func TaskNotFound() *APIError {
	return New(ErrCodeTaskNotFound, "Task not found")
}

func TaskRunning() *APIError {
	return New(ErrCodeTaskRunning, "Task is running")
}

func TaskCompleted() *APIError {
	return New(ErrCodeTaskCompleted, "Task already completed")
}

func TaskFailed(reason string) *APIError {
	return New(ErrCodeTaskFailed, "Task failed", reason)
}

func TaskCancelled() *APIError {
	return New(ErrCodeTaskCancelled, "Task was cancelled")
}

// 代理相关错误
func ProxyNotFound() *APIError {
	return New(ErrCodeProxyNotFound, "Proxy not found")
}

func ProxyUnavailable(reason string) *APIError {
	return New(ErrCodeProxyUnavailable, "Proxy unavailable", reason)
}

func ProxyTestFailed(reason string) *APIError {
	return New(ErrCodeProxyTestFailed, "Proxy test failed", reason)
}

// Telegram相关错误
func TelegramAuthError(reason string) *APIError {
	return New(ErrCodeTelegramAuth, "Telegram authentication error", reason)
}

func TelegramAPIError(reason string) *APIError {
	return New(ErrCodeTelegramAPI, "Telegram API error", reason)
}

func TelegramTimeout() *APIError {
	return New(ErrCodeTelegramTimeout, "Telegram operation timeout")
}

func TelegramRateLimit() *APIError {
	return New(ErrCodeTelegramRateLimit, "Telegram rate limit exceeded")
}

// GetHTTPStatus 根据错误码返回对应的HTTP状态码
func (e *APIError) GetHTTPStatus() int {
	switch e.Code / 100 {
	case 400:
		return http.StatusBadRequest
	case 401:
		return http.StatusUnauthorized
	case 403:
		return http.StatusForbidden
	case 404:
		return http.StatusNotFound
	case 409:
		return http.StatusConflict
	case 429:
		return http.StatusTooManyRequests
	case 500:
		return http.StatusInternalServerError
	default:
		return http.StatusInternalServerError
	}
}
