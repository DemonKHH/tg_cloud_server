package services

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/repository"
)

// TemplateType 模板类型
type TemplateType string

const (
	TemplateTypePrivate   TemplateType = "private"   // 私信模板
	TemplateBroadcast     TemplateType = "broadcast" // 群发模板
	TemplateTypeGroupChat TemplateType = "groupchat" // 群聊模板
	TemplateTypeWelcome   TemplateType = "welcome"   // 欢迎模板
	TemplateTypeFollowUp  TemplateType = "followup"  // 跟进模板
)

// TemplateStatus 模板状态
type TemplateStatus string

const (
	TemplateStatusDraft    TemplateStatus = "draft"    // 草稿
	TemplateStatusActive   TemplateStatus = "active"   // 激活
	TemplateStatusInactive TemplateStatus = "inactive" // 停用
	TemplateStatusArchived TemplateStatus = "archived" // 归档
)

// MessageTemplate 消息模板
type MessageTemplate struct {
	ID          uint64             `json:"id"`
	UserID      uint64             `json:"user_id"`
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Type        TemplateType       `json:"type"`
	Status      TemplateStatus     `json:"status"`
	Content     string             `json:"content"`
	Variables   []TemplateVariable `json:"variables"`
	Tags        []string           `json:"tags"`
	Category    string             `json:"category"`
	Language    string             `json:"language"`
	UsageCount  int64              `json:"usage_count"`
	CreatedAt   time.Time          `json:"created_at"`
	UpdatedAt   time.Time          `json:"updated_at"`
}

// TemplateVariable 模板变量
type TemplateVariable struct {
	Name         string      `json:"name"`
	Type         string      `json:"type"` // string, number, date, boolean
	Required     bool        `json:"required"`
	DefaultValue interface{} `json:"default_value"`
	Description  string      `json:"description"`
	Examples     []string    `json:"examples"`
}

// TemplateRenderRequest 模板渲染请求
type TemplateRenderRequest struct {
	TemplateID uint64                 `json:"template_id"`
	Variables  map[string]interface{} `json:"variables"`
}

// TemplateRenderResult 模板渲染结果
type TemplateRenderResult struct {
	Content     string            `json:"content"`
	MissingVars []string          `json:"missing_vars,omitempty"`
	InvalidVars map[string]string `json:"invalid_vars,omitempty"`
	RenderedAt  time.Time         `json:"rendered_at"`
}

// TemplateAnalytics 模板分析数据
type TemplateAnalytics struct {
	TemplateID      uint64    `json:"template_id"`
	UsageCount      int64     `json:"usage_count"`
	SuccessRate     float64   `json:"success_rate"`
	AvgResponseTime float64   `json:"avg_response_time"`
	TopVariables    []string  `json:"top_variables"`
	LastUsedAt      time.Time `json:"last_used_at"`
}

// TemplateService 模板服务接口
type TemplateService interface {
	// 模板管理
	CreateTemplate(ctx context.Context, userID uint64, req *CreateTemplateRequest) (*MessageTemplate, error)
	GetTemplate(ctx context.Context, userID uint64, templateID uint64) (*MessageTemplate, error)
	GetTemplates(ctx context.Context, userID uint64, filter *TemplateFilter) ([]*MessageTemplate, int64, error)
	UpdateTemplate(ctx context.Context, userID uint64, templateID uint64, req *UpdateTemplateRequest) (*MessageTemplate, error)
	DeleteTemplate(ctx context.Context, userID uint64, templateID uint64) error

	// 模板渲染
	RenderTemplate(ctx context.Context, userID uint64, req *TemplateRenderRequest) (*TemplateRenderResult, error)
	PreviewTemplate(ctx context.Context, userID uint64, templateID uint64, variables map[string]interface{}) (*TemplateRenderResult, error)
	ValidateTemplate(ctx context.Context, content string) (*ValidationResult, error)

	// 变量管理
	ExtractVariables(ctx context.Context, content string) ([]TemplateVariable, error)
	SuggestVariables(ctx context.Context, templateType TemplateType) ([]TemplateVariable, error)

	// 分析和统计
	GetTemplateAnalytics(ctx context.Context, userID uint64, templateID uint64) (*TemplateAnalytics, error)
	GetPopularTemplates(ctx context.Context, userID uint64, templateType TemplateType, limit int) ([]*MessageTemplate, error)

	// 批量操作
	DuplicateTemplate(ctx context.Context, userID uint64, templateID uint64, newName string) (*MessageTemplate, error)
	ImportTemplates(ctx context.Context, userID uint64, templates []ImportTemplateData) ([]*MessageTemplate, error)
	ExportTemplates(ctx context.Context, userID uint64, templateIDs []uint64) ([]byte, error)
}

// CreateTemplateRequest 创建模板请求
type CreateTemplateRequest struct {
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	Type        TemplateType       `json:"type" binding:"required"`
	Content     string             `json:"content" binding:"required"`
	Variables   []TemplateVariable `json:"variables"`
	Tags        []string           `json:"tags"`
	Category    string             `json:"category"`
	Language    string             `json:"language"`
}

// UpdateTemplateRequest 更新模板请求
type UpdateTemplateRequest struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Content     string             `json:"content"`
	Variables   []TemplateVariable `json:"variables"`
	Tags        []string           `json:"tags"`
	Category    string             `json:"category"`
	Status      TemplateStatus     `json:"status"`
}

// TemplateFilter 模板过滤器
type TemplateFilter struct {
	Type     TemplateType   `json:"type"`
	Status   TemplateStatus `json:"status"`
	Category string         `json:"category"`
	Tags     []string       `json:"tags"`
	Keyword  string         `json:"keyword"`
	Page     int            `json:"page"`
	Limit    int            `json:"limit"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	IsValid    bool                `json:"is_valid"`
	Errors     []ValidationError   `json:"errors,omitempty"`
	Warnings   []ValidationWarning `json:"warnings,omitempty"`
	Variables  []TemplateVariable  `json:"variables"`
	Statistics *TemplateStatistics `json:"statistics"`
}

// ValidationError 验证错误
type ValidationError struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Position int    `json:"position,omitempty"`
	Variable string `json:"variable,omitempty"`
}

// ValidationWarning 验证警告
type ValidationWarning struct {
	Type     string `json:"type"`
	Message  string `json:"message"`
	Position int    `json:"position,omitempty"`
}

// TemplateStatistics 模板统计
type TemplateStatistics struct {
	CharCount     int `json:"char_count"`
	WordCount     int `json:"word_count"`
	VariableCount int `json:"variable_count"`
	LineCount     int `json:"line_count"`
}

// ImportTemplateData 导入模板数据
type ImportTemplateData struct {
	Name        string             `json:"name"`
	Description string             `json:"description"`
	Type        TemplateType       `json:"type"`
	Content     string             `json:"content"`
	Variables   []TemplateVariable `json:"variables"`
	Tags        []string           `json:"tags"`
	Category    string             `json:"category"`
}

// templateService 模板服务实现
type templateService struct {
	templateRepo repository.TemplateRepository
	logger       *zap.Logger

	// 变量正则表达式
	variableRegex *regexp.Regexp
}

// NewTemplateService 创建模板服务
func NewTemplateService(templateRepo repository.TemplateRepository) TemplateService {
	return &templateService{
		templateRepo:  templateRepo,
		logger:        logger.Get().Named("template_service"),
		variableRegex: regexp.MustCompile(`\{\{(\w+)\}\}`),
	}
}

// CreateTemplate 创建模板
func (s *templateService) CreateTemplate(ctx context.Context, userID uint64, req *CreateTemplateRequest) (*MessageTemplate, error) {
	s.logger.Info("Creating template",
		zap.Uint64("user_id", userID),
		zap.String("name", req.Name),
		zap.String("type", string(req.Type)))

	// 验证模板内容
	validation, err := s.ValidateTemplate(ctx, req.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to validate template: %w", err)
	}

	if !validation.IsValid {
		return nil, fmt.Errorf("template validation failed: %d errors found", len(validation.Errors))
	}

	// 提取变量（如果没有提供）
	if len(req.Variables) == 0 {
		req.Variables, _ = s.ExtractVariables(ctx, req.Content)
	}

	template := &MessageTemplate{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Status:      TemplateStatusDraft,
		Content:     req.Content,
		Variables:   req.Variables,
		Tags:        req.Tags,
		Category:    req.Category,
		Language:    req.Language,
		UsageCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.templateRepo.Create(template); err != nil {
		s.logger.Error("Failed to create template", zap.Error(err))
		return nil, fmt.Errorf("failed to create template: %w", err)
	}

	s.logger.Info("Template created successfully", zap.Uint64("template_id", template.ID))
	return template, nil
}

// GetTemplate 获取模板
func (s *templateService) GetTemplate(ctx context.Context, userID uint64, templateID uint64) (*MessageTemplate, error) {
	template, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
	if err != nil {
		return nil, err
	}
	return template, nil
}

// GetTemplates 获取模板列表
func (s *templateService) GetTemplates(ctx context.Context, userID uint64, filter *TemplateFilter) ([]*MessageTemplate, int64, error) {
	if filter.Page < 1 {
		filter.Page = 1
	}
	if filter.Limit < 1 {
		filter.Limit = 20
	}

	offset := (filter.Page - 1) * filter.Limit
	return s.templateRepo.GetByUserIDWithFilter(userID, filter, offset, filter.Limit)
}

// RenderTemplate 渲染模板
func (s *templateService) RenderTemplate(ctx context.Context, userID uint64, req *TemplateRenderRequest) (*TemplateRenderResult, error) {
	s.logger.Info("Rendering template",
		zap.Uint64("user_id", userID),
		zap.Uint64("template_id", req.TemplateID))

	// 获取模板
	template, err := s.templateRepo.GetByUserIDAndID(userID, req.TemplateID)
	if err != nil {
		return nil, err
	}

	// 渲染内容
	result := &TemplateRenderResult{
		Content:     template.Content,
		MissingVars: make([]string, 0),
		InvalidVars: make(map[string]string),
		RenderedAt:  time.Now(),
	}

	// 替换变量
	content := template.Content
	for _, variable := range template.Variables {
		placeholder := fmt.Sprintf("{{%s}}", variable.Name)

		value, exists := req.Variables[variable.Name]
		if !exists {
			if variable.Required {
				result.MissingVars = append(result.MissingVars, variable.Name)
				continue
			} else if variable.DefaultValue != nil {
				value = variable.DefaultValue
			} else {
				continue
			}
		}

		// 验证变量类型
		if !s.validateVariableType(value, variable.Type) {
			result.InvalidVars[variable.Name] = fmt.Sprintf("expected %s, got %T", variable.Type, value)
			continue
		}

		// 格式化变量值
		formattedValue := s.formatVariableValue(value, variable.Type)
		content = strings.ReplaceAll(content, placeholder, formattedValue)
	}

	result.Content = content

	// 更新使用统计
	s.templateRepo.IncrementUsageCount(req.TemplateID)

	s.logger.Info("Template rendered successfully",
		zap.Uint64("template_id", req.TemplateID),
		zap.Int("missing_vars", len(result.MissingVars)),
		zap.Int("invalid_vars", len(result.InvalidVars)))

	return result, nil
}

// ValidateTemplate 验证模板
func (s *templateService) ValidateTemplate(ctx context.Context, content string) (*ValidationResult, error) {
	result := &ValidationResult{
		IsValid:   true,
		Errors:    make([]ValidationError, 0),
		Warnings:  make([]ValidationWarning, 0),
		Variables: make([]TemplateVariable, 0),
		Statistics: &TemplateStatistics{
			CharCount: len(content),
			WordCount: len(strings.Fields(content)),
			LineCount: len(strings.Split(content, "\n")),
		},
	}

	// 检查变量语法
	variables := s.variableRegex.FindAllStringSubmatch(content, -1)
	for _, match := range variables {
		if len(match) < 2 {
			continue
		}

		varName := match[1]

		// 检查变量名是否有效
		if !s.isValidVariableName(varName) {
			result.Errors = append(result.Errors, ValidationError{
				Type:     "invalid_variable_name",
				Message:  fmt.Sprintf("Invalid variable name: %s", varName),
				Variable: varName,
			})
			result.IsValid = false
		}

		// 添加到变量列表
		found := false
		for _, v := range result.Variables {
			if v.Name == varName {
				found = true
				break
			}
		}
		if !found {
			result.Variables = append(result.Variables, TemplateVariable{
				Name:     varName,
				Type:     "string",
				Required: true,
			})
		}
	}

	result.Statistics.VariableCount = len(result.Variables)

	// 检查内容长度
	if len(content) > 4096 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Type:    "content_too_long",
			Message: "Template content is longer than recommended (4096 characters)",
		})
	}

	// 检查是否有未闭合的变量
	openBraces := strings.Count(content, "{{")
	closeBraces := strings.Count(content, "}}")
	if openBraces != closeBraces {
		result.Errors = append(result.Errors, ValidationError{
			Type:    "unmatched_braces",
			Message: "Unmatched variable braces in template",
		})
		result.IsValid = false
	}

	return result, nil
}

// ExtractVariables 提取模板变量
func (s *templateService) ExtractVariables(ctx context.Context, content string) ([]TemplateVariable, error) {
	variables := make([]TemplateVariable, 0)
	matches := s.variableRegex.FindAllStringSubmatch(content, -1)

	seen := make(map[string]bool)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}

		varName := match[1]
		if seen[varName] {
			continue
		}
		seen[varName] = true

		variable := TemplateVariable{
			Name:        varName,
			Type:        s.inferVariableType(varName),
			Required:    true,
			Description: s.generateVariableDescription(varName),
		}

		variables = append(variables, variable)
	}

	return variables, nil
}

// SuggestVariables 建议变量
func (s *templateService) SuggestVariables(ctx context.Context, templateType TemplateType) ([]TemplateVariable, error) {
	commonVars := []TemplateVariable{
		{Name: "user_name", Type: "string", Required: false, Description: "用户姓名"},
		{Name: "first_name", Type: "string", Required: false, Description: "名字"},
		{Name: "last_name", Type: "string", Required: false, Description: "姓氏"},
		{Name: "current_time", Type: "date", Required: false, Description: "当前时间"},
		{Name: "current_date", Type: "date", Required: false, Description: "当前日期"},
	}

	switch templateType {
	case TemplateTypePrivate:
		return append(commonVars, []TemplateVariable{
			{Name: "company", Type: "string", Required: false, Description: "公司名称"},
			{Name: "position", Type: "string", Required: false, Description: "职位"},
			{Name: "product", Type: "string", Required: false, Description: "产品名称"},
		}...), nil
	case TemplateBroadcast:
		return append(commonVars, []TemplateVariable{
			{Name: "group_name", Type: "string", Required: false, Description: "群组名称"},
			{Name: "announcement", Type: "string", Required: false, Description: "公告内容"},
			{Name: "event_date", Type: "date", Required: false, Description: "事件日期"},
		}...), nil
	case TemplateTypeGroupChat:
		return append(commonVars, []TemplateVariable{
			{Name: "topic", Type: "string", Required: false, Description: "话题"},
			{Name: "reaction", Type: "string", Required: false, Description: "反应"},
		}...), nil
	default:
		return commonVars, nil
	}
}

// 辅助方法

func (s *templateService) isValidVariableName(name string) bool {
	// 变量名只能包含字母、数字和下划线，且必须以字母开头
	matched, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_]*$`, name)
	return matched
}

func (s *templateService) inferVariableType(varName string) string {
	name := strings.ToLower(varName)

	if strings.Contains(name, "time") || strings.Contains(name, "date") {
		return "date"
	}
	if strings.Contains(name, "count") || strings.Contains(name, "number") || strings.Contains(name, "age") {
		return "number"
	}
	if strings.Contains(name, "is_") || strings.Contains(name, "has_") {
		return "boolean"
	}

	return "string"
}

func (s *templateService) generateVariableDescription(varName string) string {
	name := strings.ToLower(varName)

	descriptions := map[string]string{
		"user_name":    "用户姓名",
		"first_name":   "名字",
		"last_name":    "姓氏",
		"email":        "邮箱地址",
		"phone":        "电话号码",
		"company":      "公司名称",
		"position":     "职位",
		"product":      "产品名称",
		"current_time": "当前时间",
		"current_date": "当前日期",
	}

	if desc, exists := descriptions[name]; exists {
		return desc
	}

	return fmt.Sprintf("变量: %s", varName)
}

func (s *templateService) validateVariableType(value interface{}, expectedType string) bool {
	switch expectedType {
	case "string":
		_, ok := value.(string)
		return ok
	case "number":
		switch value.(type) {
		case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64, float32, float64:
			return true
		default:
			return false
		}
	case "boolean":
		_, ok := value.(bool)
		return ok
	case "date":
		switch value.(type) {
		case string, time.Time:
			return true
		default:
			return false
		}
	default:
		return true
	}
}

func (s *templateService) formatVariableValue(value interface{}, varType string) string {
	switch varType {
	case "date":
		if t, ok := value.(time.Time); ok {
			return t.Format("2006-01-02 15:04:05")
		}
		if s, ok := value.(string); ok {
			return s
		}
	case "number":
		return fmt.Sprintf("%v", value)
	case "boolean":
		if b, ok := value.(bool); ok {
			if b {
				return "是"
			}
			return "否"
		}
	}

	return fmt.Sprintf("%v", value)
}

// DuplicateTemplate 复制模板
func (s *templateService) DuplicateTemplate(ctx context.Context, userID uint64, templateID uint64, newName string) (*MessageTemplate, error) {
	// 获取原模板
	original, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
	if err != nil {
		return nil, err
	}

	// 创建副本
	duplicate := &MessageTemplate{
		UserID:      userID,
		Name:        newName,
		Description: original.Description + " (副本)",
		Type:        original.Type,
		Status:      TemplateStatusDraft,
		Content:     original.Content,
		Variables:   original.Variables,
		Tags:        original.Tags,
		Category:    original.Category,
		Language:    original.Language,
		UsageCount:  0,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	if err := s.templateRepo.Create(duplicate); err != nil {
		return nil, fmt.Errorf("failed to duplicate template: %w", err)
	}

	return duplicate, nil
}

// UpdateTemplate 更新模板
func (s *templateService) UpdateTemplate(ctx context.Context, userID uint64, templateID uint64, req *UpdateTemplateRequest) (*MessageTemplate, error) {
	template, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
	if err != nil {
		return nil, err
	}

	// 更新字段
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Content != "" {
		// 验证新内容
		if validation, err := s.ValidateTemplate(ctx, req.Content); err != nil || !validation.IsValid {
			return nil, fmt.Errorf("invalid template content")
		}
		template.Content = req.Content
	}
	if req.Variables != nil {
		template.Variables = req.Variables
	}
	if req.Tags != nil {
		template.Tags = req.Tags
	}
	if req.Category != "" {
		template.Category = req.Category
	}
	if req.Status != "" {
		template.Status = req.Status
	}

	template.UpdatedAt = time.Now()

	if err := s.templateRepo.Update(template); err != nil {
		return nil, fmt.Errorf("failed to update template: %w", err)
	}

	return template, nil
}

// DeleteTemplate 删除模板
func (s *templateService) DeleteTemplate(ctx context.Context, userID uint64, templateID uint64) error {
	// 验证模板所有权
	_, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
	if err != nil {
		return err
	}

	return s.templateRepo.Delete(templateID)
}

// PreviewTemplate 预览模板
func (s *templateService) PreviewTemplate(ctx context.Context, userID uint64, templateID uint64, variables map[string]interface{}) (*TemplateRenderResult, error) {
	req := &TemplateRenderRequest{
		TemplateID: templateID,
		Variables:  variables,
	}
	return s.RenderTemplate(ctx, userID, req)
}

// GetTemplateAnalytics 获取模板分析数据
func (s *templateService) GetTemplateAnalytics(ctx context.Context, userID uint64, templateID uint64) (*TemplateAnalytics, error) {
	// TODO: 实现模板分析数据获取
	return &TemplateAnalytics{
		TemplateID:      templateID,
		UsageCount:      0,
		SuccessRate:     0,
		AvgResponseTime: 0,
		TopVariables:    []string{},
		LastUsedAt:      time.Now(),
	}, nil
}

// GetPopularTemplates 获取热门模板
func (s *templateService) GetPopularTemplates(ctx context.Context, userID uint64, templateType TemplateType, limit int) ([]*MessageTemplate, error) {
	return s.templateRepo.GetPopularByType(userID, string(templateType), limit)
}

// ImportTemplates 导入模板
func (s *templateService) ImportTemplates(ctx context.Context, userID uint64, templates []ImportTemplateData) ([]*MessageTemplate, error) {
	var results []*MessageTemplate

	for _, data := range templates {
		req := &CreateTemplateRequest{
			Name:        data.Name,
			Description: data.Description,
			Type:        data.Type,
			Content:     data.Content,
			Variables:   data.Variables,
			Tags:        data.Tags,
			Category:    data.Category,
		}

		template, err := s.CreateTemplate(ctx, userID, req)
		if err != nil {
			s.logger.Error("Failed to import template",
				zap.String("name", data.Name),
				zap.Error(err))
			continue
		}

		results = append(results, template)
	}

	return results, nil
}

// ExportTemplates 导出模板
func (s *templateService) ExportTemplates(ctx context.Context, userID uint64, templateIDs []uint64) ([]byte, error) {
	var templates []*MessageTemplate

	for _, id := range templateIDs {
		template, err := s.templateRepo.GetByUserIDAndID(userID, id)
		if err != nil {
			continue
		}
		templates = append(templates, template)
	}

	return json.Marshal(templates)
}
