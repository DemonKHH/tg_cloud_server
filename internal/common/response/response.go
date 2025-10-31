package response

import (
	"math/rand"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// 统一的响应码定义
const (
	CodeSuccess = 0 // 成功

	// 通用错误码 1xxx
	CodeInvalidParam  = 1001 // 参数错误
	CodeUnauthorized  = 1002 // 未授权
	CodeForbidden     = 1003 // 禁止访问
	CodeNotFound      = 1004 // 资源不存在
	CodeInternalError = 1005 // 服务器内部错误
	CodeRateLimit     = 1006 // 请求过于频繁
	CodeConflict      = 1007 // 资源冲突

	// 业务错误码 2xxx
	CodeUserExists         = 2001 // 用户已存在
	CodeInvalidCredentials = 2002 // 凭证无效
	CodeAccountNotFound    = 2003 // 账号不存在
	CodeTaskNotFound       = 2004 // 任务不存在
	CodeProxyNotFound      = 2005 // 代理不存在
	CodeAccountBusy        = 2006 // 账号忙碌
	CodeConnectionFailed   = 2007 // 连接失败
)

// APIResponse 统一API响应格式
type APIResponse struct {
	Code int         `json:"code"`           // 响应码，0表示成功，非0表示失败
	Msg  string      `json:"msg"`            // 响应消息
	Data interface{} `json:"data,omitempty"` // 响应数据
}

// PaginatedResponse 分页响应
type PaginatedResponse struct {
	Items      interface{}            `json:"items"`
	Pagination *PaginationInfo        `json:"pagination"`
	Meta       map[string]interface{} `json:"meta,omitempty"`
}

// PaginationInfo 分页信息
type PaginationInfo struct {
	CurrentPage int   `json:"current_page"`
	PerPage     int   `json:"per_page"`
	Total       int64 `json:"total"`
	TotalPages  int   `json:"total_pages"`
	HasNext     bool  `json:"has_next"`
	HasPrev     bool  `json:"has_prev"`
}

// Success 成功响应
func Success(c *gin.Context, data interface{}) {
	response := &APIResponse{
		Code: CodeSuccess,
		Msg:  "success",
		Data: data,
	}
	c.JSON(http.StatusOK, response)
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, msg string, data interface{}) {
	response := &APIResponse{
		Code: CodeSuccess,
		Msg:  msg,
		Data: data,
	}
	c.JSON(http.StatusOK, response)
}

// Error 错误响应
func Error(c *gin.Context, code int, msg string) {
	response := &APIResponse{
		Code: code,
		Msg:  msg,
		Data: nil,
	}
	c.JSON(http.StatusOK, response)
}

// ErrorWithData 带数据的错误响应
func ErrorWithData(c *gin.Context, code int, msg string, data interface{}) {
	response := &APIResponse{
		Code: code,
		Msg:  msg,
		Data: data,
	}
	c.JSON(http.StatusOK, response)
}

// InvalidParam 参数错误响应
func InvalidParam(c *gin.Context, msg string) {
	Error(c, CodeInvalidParam, msg)
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context, msg ...string) {
	message := "未授权"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	Error(c, CodeUnauthorized, message)
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context, msg ...string) {
	message := "权限不足"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	Error(c, CodeForbidden, message)
}

// NotFound 资源未找到响应
func NotFound(c *gin.Context, msg ...string) {
	message := "资源不存在"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	Error(c, CodeNotFound, message)
}

// InternalError 内部错误响应
func InternalError(c *gin.Context, msg ...string) {
	message := "服务器内部错误"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	Error(c, CodeInternalError, message)
}

// Conflict 冲突响应
func Conflict(c *gin.Context, msg string) {
	Error(c, CodeConflict, msg)
}

// TooManyRequests 限流响应
func TooManyRequests(c *gin.Context, msg ...string) {
	message := "请求过于频繁，请稍后重试"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	Error(c, CodeRateLimit, message)
}

// UserExists 用户已存在
func UserExists(c *gin.Context) {
	Error(c, CodeUserExists, "用户已存在")
}

// InvalidCredentials 凭证无效
func InvalidCredentials(c *gin.Context) {
	Error(c, CodeInvalidCredentials, "用户名或密码错误")
}

// AccountNotFound 账号不存在
func AccountNotFound(c *gin.Context) {
	Error(c, CodeAccountNotFound, "账号不存在")
}

// TaskNotFound 任务不存在
func TaskNotFound(c *gin.Context) {
	Error(c, CodeTaskNotFound, "任务不存在")
}

// ProxyNotFound 代理不存在
func ProxyNotFound(c *gin.Context) {
	Error(c, CodeProxyNotFound, "代理不存在")
}

// AccountBusy 账号忙碌
func AccountBusy(c *gin.Context) {
	Error(c, CodeAccountBusy, "账号正在执行其他任务，请稍后重试")
}

// ConnectionFailed 连接失败
func ConnectionFailed(c *gin.Context, msg ...string) {
	message := "连接失败"
	if len(msg) > 0 && msg[0] != "" {
		message = msg[0]
	}
	Error(c, CodeConnectionFailed, message)
}

// Paginated 分页响应
func Paginated(c *gin.Context, items interface{}, page, limit int, total int64, meta ...map[string]interface{}) {
	totalPages := int((total + int64(limit) - 1) / int64(limit))

	// 确保 items 不是 nil，返回空数组
	if items == nil {
		items = []interface{}{}
	}

	pagination := &PaginationInfo{
		CurrentPage: page,
		PerPage:     limit,
		Total:       total,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrev:     page > 1,
	}

	data := &PaginatedResponse{
		Items:      items,
		Pagination: pagination,
	}

	if len(meta) > 0 {
		data.Meta = meta[0]
	}

	Success(c, data)
}

// getRequestID 获取请求ID
func getRequestID(c *gin.Context) string {
	if id := c.GetHeader("X-Request-ID"); id != "" {
		return id
	}
	if id := c.GetString("request_id"); id != "" {
		return id
	}
	return ""
}

// SetRequestID 设置请求ID中间件
func SetRequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

// generateRequestID 生成请求ID
func generateRequestID() string {
	// 简单的请求ID生成，实际项目中可以使用UUID或其他算法
	return time.Now().Format("20060102150405") + "-" + randString(6)
}

// randString 生成随机字符串
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	r := rand.New(rand.NewSource(time.Now().UnixNano()))
	for i := range b {
		b[i] = letters[r.Intn(len(letters))]
	}
	return string(b)
}
