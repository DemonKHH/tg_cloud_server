package response

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"

	"tg_cloud_server/internal/common/errors"
)

// APIResponse 统一API响应格式
type APIResponse struct {
	Success   bool        `json:"success"`
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	Error     *ErrorInfo  `json:"error,omitempty"`
	Timestamp int64       `json:"timestamp"`
	RequestID string      `json:"request_id,omitempty"`
}

// ErrorInfo 错误详情
type ErrorInfo struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
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
		Success:   true,
		Code:      200,
		Message:   "Success",
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// SuccessWithMessage 带消息的成功响应
func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	response := &APIResponse{
		Success:   true,
		Code:      200,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusOK, response)
}

// Created 创建成功响应
func Created(c *gin.Context, message string, data interface{}) {
	response := &APIResponse{
		Success:   true,
		Code:      201,
		Message:   message,
		Data:      data,
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(http.StatusCreated, response)
}

// Error 错误响应
func Error(c *gin.Context, err *errors.APIError) {
	response := &APIResponse{
		Success: false,
		Code:    err.Code,
		Message: "Request failed",
		Error: &ErrorInfo{
			Code:    err.Code,
			Message: err.Message,
			Details: err.Details,
		},
		Timestamp: time.Now().Unix(),
		RequestID: getRequestID(c),
	}
	c.JSON(err.GetHTTPStatus(), response)
}

// InternalError 内部错误响应
func InternalError(c *gin.Context, message string) {
	err := errors.InternalServerError(message)
	Error(c, err)
}

// BadRequest 请求错误响应
func BadRequest(c *gin.Context, message string) {
	err := errors.InvalidParameter(message)
	Error(c, err)
}

// Unauthorized 未授权响应
func Unauthorized(c *gin.Context) {
	err := errors.Unauthorized()
	Error(c, err)
}

// Forbidden 禁止访问响应
func Forbidden(c *gin.Context) {
	err := errors.Forbidden()
	Error(c, err)
}

// NotFound 资源未找到响应
func NotFound(c *gin.Context, resource string) {
	err := errors.NotFound(resource)
	Error(c, err)
}

// Conflict 冲突响应
func Conflict(c *gin.Context, message string) {
	err := errors.Conflict(message)
	Error(c, err)
}

// TooManyRequests 限流响应
func TooManyRequests(c *gin.Context) {
	err := errors.TooManyRequests()
	Error(c, err)
}

// Paginated 分页响应
func Paginated(c *gin.Context, items interface{}, page, limit int, total int64, meta ...map[string]interface{}) {
	totalPages := int((total + int64(limit) - 1) / int64(limit))

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
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
