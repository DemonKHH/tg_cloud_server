package validator

import (
	"fmt"
	"net"
	"reflect"
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
)

// CustomValidator 自定义验证器
type CustomValidator struct {
	validator *validator.Validate
}

// NewCustomValidator 创建自定义验证器
func NewCustomValidator() *CustomValidator {
	v := validator.New()

	// 注册自定义验证规则
	v.RegisterValidation("phone", validatePhone)
	v.RegisterValidation("proxy_protocol", validateProxyProtocol)
	v.RegisterValidation("task_type", validateTaskType)
	v.RegisterValidation("account_status", validateAccountStatus)
	v.RegisterValidation("proxy_port", validateProxyPort)
	v.RegisterValidation("telegram_username", validateTelegramUsername)
	v.RegisterValidation("strong_password", validateStrongPassword)

	// 注册标签名函数，用于自定义错误消息中的字段名
	v.RegisterTagNameFunc(func(fld reflect.StructField) string {
		name := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
		if name == "-" {
			return ""
		}
		return name
	})

	return &CustomValidator{validator: v}
}

// ValidateStruct 验证结构体
func (cv *CustomValidator) ValidateStruct(obj interface{}) error {
	if kindOfData(obj) == reflect.Struct {
		if err := cv.validator.Struct(obj); err != nil {
			return cv.formatValidationError(err)
		}
	}
	return nil
}

// Engine 返回底层验证器引擎
func (cv *CustomValidator) Engine() interface{} {
	return cv.validator
}

// formatValidationError 格式化验证错误
func (cv *CustomValidator) formatValidationError(err error) error {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {
		var errorMessages []string
		for _, fieldError := range validationErrors {
			errorMessages = append(errorMessages, cv.getErrorMessage(fieldError))
		}
		return fmt.Errorf("validation failed: %s", strings.Join(errorMessages, ", "))
	}
	return err
}

// getErrorMessage 获取错误消息
func (cv *CustomValidator) getErrorMessage(fieldError validator.FieldError) string {
	fieldName := fieldError.Field()
	tag := fieldError.Tag()

	switch tag {
	case "required":
		return fmt.Sprintf("%s is required", fieldName)
	case "email":
		return fmt.Sprintf("%s must be a valid email address", fieldName)
	case "min":
		return fmt.Sprintf("%s must be at least %s characters long", fieldName, fieldError.Param())
	case "max":
		return fmt.Sprintf("%s must be at most %s characters long", fieldName, fieldError.Param())
	case "len":
		return fmt.Sprintf("%s must be exactly %s characters long", fieldName, fieldError.Param())
	case "phone":
		return fmt.Sprintf("%s must be a valid phone number", fieldName)
	case "proxy_protocol":
		return fmt.Sprintf("%s must be one of: http, https, socks5", fieldName)
	case "task_type":
		return fmt.Sprintf("%s must be one of: check, private_message, broadcast, verify_code, group_chat", fieldName)
	case "account_status":
		return fmt.Sprintf("%s must be one of: normal, warning, restricted, dead, cooling, maintenance, new", fieldName)
	case "proxy_port":
		return fmt.Sprintf("%s must be a valid port number (1-65535)", fieldName)
	case "telegram_username":
		return fmt.Sprintf("%s must be a valid Telegram username", fieldName)
	case "strong_password":
		return fmt.Sprintf("%s must be at least 8 characters long and contain uppercase, lowercase, number and special character", fieldName)
	case "ip":
		return fmt.Sprintf("%s must be a valid IP address", fieldName)
	case "oneof":
		return fmt.Sprintf("%s must be one of: %s", fieldName, fieldError.Param())
	default:
		return fmt.Sprintf("%s is invalid", fieldName)
	}
}

// kindOfData 获取数据类型
func kindOfData(data interface{}) reflect.Kind {
	value := reflect.ValueOf(data)
	valueType := value.Kind()

	if valueType == reflect.Ptr {
		valueType = value.Elem().Kind()
	}
	return valueType
}

// 自定义验证函数

// validatePhone 验证手机号
func validatePhone(fl validator.FieldLevel) bool {
	phone := fl.Field().String()
	if phone == "" {
		return false
	}

	// 简单的国际手机号验证，支持+开头的国际格式
	phoneRegex := regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)
	return phoneRegex.MatchString(phone)
}

// validateProxyProtocol 验证代理协议
func validateProxyProtocol(fl validator.FieldLevel) bool {
	protocol := fl.Field().String()
	validProtocols := []string{"http", "https", "socks5"}

	for _, valid := range validProtocols {
		if protocol == valid {
			return true
		}
	}
	return false
}

// validateTaskType 验证任务类型
func validateTaskType(fl validator.FieldLevel) bool {
	taskType := fl.Field().String()
	validTypes := []string{"check", "private_message", "broadcast", "verify_code", "group_chat"}

	for _, valid := range validTypes {
		if taskType == valid {
			return true
		}
	}
	return false
}

// validateAccountStatus 验证账号状态
func validateAccountStatus(fl validator.FieldLevel) bool {
	status := fl.Field().String()
	validStatuses := []string{"normal", "warning", "restricted", "dead", "cooling", "maintenance", "new"}

	for _, valid := range validStatuses {
		if status == valid {
			return true
		}
	}
	return false
}

// validateProxyPort 验证代理端口
func validateProxyPort(fl validator.FieldLevel) bool {
	port := fl.Field().Int()
	return port >= 1 && port <= 65535
}

// validateTelegramUsername 验证Telegram用户名
func validateTelegramUsername(fl validator.FieldLevel) bool {
	username := fl.Field().String()
	if username == "" {
		return true // 用户名可以为空
	}

	// Telegram用户名规则：5-32个字符，只能包含字母、数字和下划线，不能以数字开头
	usernameRegex := regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{4,31}$`)
	return usernameRegex.MatchString(username)
}

// validateStrongPassword 验证强密码
func validateStrongPassword(fl validator.FieldLevel) bool {
	password := fl.Field().String()

	if len(password) < 8 {
		return false
	}

	// 检查是否包含大写字母
	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(password)
	// 检查是否包含小写字母
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(password)
	// 检查是否包含数字
	hasNumber := regexp.MustCompile(`\d`).MatchString(password)
	// 检查是否包含特殊字符
	hasSpecial := regexp.MustCompile(`[!@#$%^&*()_+\-=\[\]{};':"\\|,.<>\/?]`).MatchString(password)

	return hasUpper && hasLower && hasNumber && hasSpecial
}

// InitCustomValidator 初始化自定义验证器
func InitCustomValidator() {
	customValidator := NewCustomValidator()
	binding.Validator = customValidator
}

// ValidateIPAddress 验证IP地址
func ValidateIPAddress(ip string) bool {
	return net.ParseIP(ip) != nil
}

// ValidatePortRange 验证端口范围
func ValidatePortRange(port int) bool {
	return port >= 1 && port <= 65535
}

// ValidateURL 验证URL格式
func ValidateURL(url string) bool {
	urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
	return urlRegex.MatchString(url)
}

// ValidateJSON 验证JSON格式
func ValidateJSON(jsonStr string) bool {
	if jsonStr == "" {
		return true
	}

	// 简单的JSON格式检查
	jsonStr = strings.TrimSpace(jsonStr)
	return (strings.HasPrefix(jsonStr, "{") && strings.HasSuffix(jsonStr, "}")) ||
		(strings.HasPrefix(jsonStr, "[") && strings.HasSuffix(jsonStr, "]"))
}

// ValidateAccountID 验证账号ID
func ValidateAccountID(accountID string) (uint64, error) {
	id, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid account ID")
	}
	return id, nil
}

// ValidateUserID 验证用户ID
func ValidateUserID(userID string) (uint64, error) {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid user ID")
	}
	return id, nil
}

// ValidateTaskID 验证任务ID
func ValidateTaskID(taskID string) (uint64, error) {
	id, err := strconv.ParseUint(taskID, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid task ID")
	}
	return id, nil
}

// ValidateProxyID 验证代理ID
func ValidateProxyID(proxyID string) (uint64, error) {
	id, err := strconv.ParseUint(proxyID, 10, 64)
	if err != nil || id == 0 {
		return 0, fmt.Errorf("invalid proxy ID")
	}
	return id, nil
}

// ValidatePagination 验证分页参数
func ValidatePagination(page, limit int) (int, int, error) {
	if page < 1 {
		page = 1
	}

	if limit < 1 {
		limit = 20
	} else if limit > 100 {
		limit = 100
	}

	return page, limit, nil
}

// SanitizeString 清理字符串输入
func SanitizeString(input string) string {
	// 移除前后空格
	input = strings.TrimSpace(input)

	// 移除控制字符
	input = regexp.MustCompile(`[\x00-\x1f\x7f]`).ReplaceAllString(input, "")

	return input
}

// SanitizeHTML 清理HTML内容
func SanitizeHTML(input string) string {
	// 移除HTML标签
	htmlRegex := regexp.MustCompile(`<[^>]*>`)
	return htmlRegex.ReplaceAllString(input, "")
}

// ValidateTimeRange 验证时间范围
func ValidateTimeRange(timeRange string) bool {
	validRanges := []string{"today", "week", "month", "all"}
	for _, valid := range validRanges {
		if timeRange == valid {
			return true
		}
	}
	return false
}
