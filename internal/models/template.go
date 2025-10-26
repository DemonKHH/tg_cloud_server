package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

// TemplateType 模板类型枚举
type TemplateType string

const (
	TemplateTypeText     TemplateType = "text"
	TemplateTypeRichText TemplateType = "rich_text"
	TemplateTypeMarkdown TemplateType = "markdown"
	TemplateTypeHTML     TemplateType = "html"
)

// TemplateCategory 模板分类
type TemplateCategory string

const (
	CategoryWelcome      TemplateCategory = "welcome"
	CategoryPromotion    TemplateCategory = "promotion"
	CategoryNotification TemplateCategory = "notification"
	CategoryFollowUp     TemplateCategory = "follow_up"
	CategoryCustom       TemplateCategory = "custom"
)

// MessageTemplate 消息模板
type MessageTemplate struct {
	ID          uint64           `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID      uint64           `json:"user_id" gorm:"not null;index"`
	Name        string           `json:"name" gorm:"size:100;not null"`
	Description string           `json:"description" gorm:"size:500"`
	Type        TemplateType     `json:"type" gorm:"type:enum('text','rich_text','markdown','html');not null"`
	Category    TemplateCategory `json:"category" gorm:"type:enum('welcome','promotion','notification','follow_up','custom');not null"`
	Content     string           `json:"content" gorm:"type:text;not null"`
	Variables   string           `json:"variables" gorm:"type:json"` // 存储模板变量定义
	IsActive    bool             `json:"is_active" gorm:"default:true"`
	UsageCount  int64            `json:"usage_count" gorm:"default:0"`
	CreatedAt   time.Time        `json:"created_at"`
	UpdatedAt   time.Time        `json:"updated_at"`

	// 关联关系
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (MessageTemplate) TableName() string {
	return "message_templates"
}

// BeforeCreate 创建前钩子
func (t *MessageTemplate) BeforeCreate(tx *gorm.DB) error {
	t.IsActive = true
	t.UsageCount = 0
	return nil
}

// IncrementUsage 增加使用次数
func (t *MessageTemplate) IncrementUsage() {
	t.UsageCount++
}

// GetVariableNames 获取模板中的变量名列表
func (t *MessageTemplate) GetVariableNames() []string {
	// 解析模板内容，提取 {{variable}} 格式的变量
	// 这里是简化实现，实际可能需要更复杂的解析逻辑
	return []string{} // 返回空切片作为示例
}

// Render 渲染模板
func (t *MessageTemplate) Render(variables map[string]string) string {
	content := t.Content

	// 替换模板变量
	for key, value := range variables {
		placeholder := "{{" + key + "}}"
		// 简单的字符串替换，实际项目可能需要更复杂的模板引擎
		content = strings.ReplaceAll(content, placeholder, value)
	}

	return content
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string           `json:"name" binding:"required"`
	Description string           `json:"description"`
	Type        TemplateType     `json:"type" binding:"required"`
	Category    TemplateCategory `json:"category" binding:"required"`
	Content     string           `json:"content" binding:"required"`
	Variables   []string         `json:"variables"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Type        TemplateType     `json:"type"`
	Category    TemplateCategory `json:"category"`
	Content     string           `json:"content"`
	Variables   []string         `json:"variables"`
	IsActive    *bool            `json:"is_active"`
}

// TemplateFilter 模板过滤器
type TemplateFilter struct {
	UserID    uint64           `json:"user_id"`
	Category  TemplateCategory `json:"category"`
	Type      TemplateType     `json:"type"`
	IsActive  *bool            `json:"is_active"`
	Keyword   string           `json:"keyword"`
	SortBy    string           `json:"sort_by"`
	SortOrder string           `json:"sort_order"`
	Page      int              `json:"page"`
	Limit     int              `json:"limit"`
}

// TemplateVariable 模板变量定义
type TemplateVariable struct {
	Name         string `json:"name"`
	Description  string `json:"description"`
	Required     bool   `json:"required"`
	DefaultValue string `json:"default_value,omitempty"`
	Type         string `json:"type"` // string, number, boolean, date
}

// RenderRequest 渲染请求
type RenderRequest struct {
	TemplateID uint64            `json:"template_id" binding:"required"`
	Variables  map[string]string `json:"variables"`
}

// RenderResponse 渲染响应
type RenderResponse struct {
	TemplateID       uint64   `json:"template_id"`
	RenderedContent  string   `json:"rendered_content"`
	UsedVariables    []string `json:"used_variables"`
	MissingVariables []string `json:"missing_variables,omitempty"`
}

// TemplateStats 模板统计
type TemplateStats struct {
	Total      int64                `json:"total"`
	Active     int64                `json:"active"`
	Inactive   int64                `json:"inactive"`
	ByCategory map[string]int64     `json:"by_category"`
	ByType     map[string]int64     `json:"by_type"`
	TopUsed    []*TemplateUsageInfo `json:"top_used"`
}

// TemplateUsageInfo 模板使用信息
type TemplateUsageInfo struct {
	TemplateID uint64           `json:"template_id"`
	Name       string           `json:"name"`
	Category   TemplateCategory `json:"category"`
	UsageCount int64            `json:"usage_count"`
	LastUsedAt *time.Time       `json:"last_used_at,omitempty"`
}

// BatchTemplateOperation 批量模板操作
type BatchTemplateOperation struct {
	TemplateIDs  []uint64 `json:"template_ids" binding:"required"`
	Operation    string   `json:"operation" binding:"required,oneof=activate deactivate delete copy"`
	TargetUserID uint64   `json:"target_user_id,omitempty"` // 用于复制操作
}

// TemplateValidationResult 模板验证结果
type TemplateValidationResult struct {
	IsValid   bool     `json:"is_valid"`
	Errors    []string `json:"errors"`
	Variables []string `json:"variables,omitempty"` // 模板中发现的变量
}

// BatchOperationResult 批量操作结果
type BatchOperationResult struct {
	Total      int      `json:"total"`
	Successful int      `json:"successful"`
	Failed     int      `json:"failed"`
	Errors     []string `json:"errors"`
}

// ImportResult 导入结果
type ImportResult struct {
	Total       int      `json:"total"`
	Successful  int      `json:"successful"`
	Failed      int      `json:"failed"`
	Errors      []string `json:"errors"`
	ImportedIDs []uint64 `json:"imported_ids"`
}
