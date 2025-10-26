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
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

// TemplateService 模板服务接口
type TemplateService interface {
	// 模板管理
	CreateTemplate(ctx context.Context, userID uint64, req *models.CreateTemplateRequest) (*models.MessageTemplate, error)
	GetTemplate(ctx context.Context, userID uint64, templateID uint64) (*models.MessageTemplate, error)
	GetTemplates(ctx context.Context, userID uint64, filter *models.TemplateFilter) ([]*models.MessageTemplate, int64, error)
	UpdateTemplate(ctx context.Context, userID uint64, templateID uint64, req *models.UpdateTemplateRequest) (*models.MessageTemplate, error)
	DeleteTemplate(ctx context.Context, userID uint64, templateID uint64) error

	// 模板操作
	RenderTemplate(ctx context.Context, userID uint64, req *models.RenderRequest) (*models.RenderResponse, error)
	ValidateTemplate(ctx context.Context, content string) (*models.TemplateValidationResult, error)
	DuplicateTemplate(ctx context.Context, userID uint64, templateID uint64, newName string) (*models.MessageTemplate, error)

	// 批量操作
	BatchOperation(ctx context.Context, userID uint64, operation *models.BatchTemplateOperation) (*models.BatchOperationResult, error)
	ImportTemplates(ctx context.Context, userID uint64, templates []*models.CreateTemplateRequest) (*models.ImportResult, error)
	ExportTemplates(ctx context.Context, userID uint64, templateIDs []uint64) ([]byte, error)

	// 统计和分析
	GetTemplateStats(ctx context.Context, userID uint64) (*models.TemplateStats, error)
	GetPopularTemplates(ctx context.Context, userID uint64, templateType models.TemplateType, limit int) ([]*models.MessageTemplate, error)
}

// templateService 模板服务实现
type templateService struct {
	templateRepo repository.TemplateRepository
	logger       *zap.Logger
}

// NewTemplateService 创建模板服务
func NewTemplateService(templateRepo repository.TemplateRepository) TemplateService {
	return &templateService{
		templateRepo: templateRepo,
		logger:       logger.Get().Named("template_service"),
	}
}

// CreateTemplate 创建模板
func (s *templateService) CreateTemplate(ctx context.Context, userID uint64, req *models.CreateTemplateRequest) (*models.MessageTemplate, error) {
	s.logger.Info("Creating template",
		zap.Uint64("user_id", userID),
		zap.String("name", req.Name),
		zap.String("type", string(req.Type)))

	// 验证模板内容
	if validationResult, err := s.ValidateTemplate(ctx, req.Content); err != nil {
		return nil, fmt.Errorf("template validation failed: %v", err)
	} else if !validationResult.IsValid {
		return nil, fmt.Errorf("invalid template: %s", strings.Join(validationResult.Errors, "; "))
	}

	// 创建模板
	template := &models.MessageTemplate{
		UserID:      userID,
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Category:    req.Category,
		Content:     req.Content,
		IsActive:    true,
	}

	// 处理变量
	if len(req.Variables) > 0 {
		variablesJSON, err := json.Marshal(req.Variables)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal variables: %v", err)
		}
		template.Variables = string(variablesJSON)
	}

	if err := s.templateRepo.Create(template); err != nil {
		return nil, fmt.Errorf("failed to create template: %v", err)
	}

	return template, nil
}

// GetTemplate 获取模板
func (s *templateService) GetTemplate(ctx context.Context, userID uint64, templateID uint64) (*models.MessageTemplate, error) {
	return s.templateRepo.GetByUserIDAndID(userID, templateID)
}

// GetTemplates 获取模板列表
func (s *templateService) GetTemplates(ctx context.Context, userID uint64, filter *models.TemplateFilter) ([]*models.MessageTemplate, int64, error) {
	if filter == nil {
		filter = &models.TemplateFilter{UserID: userID, Page: 1, Limit: 20}
	}

	offset := (filter.Page - 1) * filter.Limit
	return s.templateRepo.GetByUserIDWithFilter(userID, filter, offset, filter.Limit)
}

// UpdateTemplate 更新模板
func (s *templateService) UpdateTemplate(ctx context.Context, userID uint64, templateID uint64, req *models.UpdateTemplateRequest) (*models.MessageTemplate, error) {
	// 获取现有模板
	template, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %v", err)
	}

	// 更新字段
	if req.Name != "" {
		template.Name = req.Name
	}
	if req.Description != "" {
		template.Description = req.Description
	}
	if req.Type != "" {
		template.Type = req.Type
	}
	if req.Category != "" {
		template.Category = req.Category
	}
	if req.Content != "" {
		// 验证新内容
		if validationResult, err := s.ValidateTemplate(ctx, req.Content); err != nil {
			return nil, fmt.Errorf("template validation failed: %v", err)
		} else if !validationResult.IsValid {
			return nil, fmt.Errorf("invalid template: %s", strings.Join(validationResult.Errors, "; "))
		}
		template.Content = req.Content
	}
	if len(req.Variables) > 0 {
		variablesJSON, err := json.Marshal(req.Variables)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal variables: %v", err)
		}
		template.Variables = string(variablesJSON)
	}
	if req.IsActive != nil {
		template.IsActive = *req.IsActive
	}

	if err := s.templateRepo.Update(template); err != nil {
		return nil, fmt.Errorf("failed to update template: %v", err)
	}

	return template, nil
}

// DeleteTemplate 删除模板
func (s *templateService) DeleteTemplate(ctx context.Context, userID uint64, templateID uint64) error {
	// 验证权限
	if _, err := s.templateRepo.GetByUserIDAndID(userID, templateID); err != nil {
		return fmt.Errorf("template not found or access denied")
	}

	return s.templateRepo.Delete(templateID)
}

// RenderTemplate 渲染模板
func (s *templateService) RenderTemplate(ctx context.Context, userID uint64, req *models.RenderRequest) (*models.RenderResponse, error) {
	// 获取模板
	template, err := s.templateRepo.GetByUserIDAndID(userID, req.TemplateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %v", err)
	}

	if !template.IsActive {
		return nil, fmt.Errorf("template is inactive")
	}

	// 渲染模板
	renderedContent := template.Render(req.Variables)

	// 提取已使用的变量
	usedVars := extractVariablesFromContent(template.Content)

	// 检查缺失的变量
	var missingVars []string
	for _, varName := range usedVars {
		if _, exists := req.Variables[varName]; !exists {
			missingVars = append(missingVars, varName)
		}
	}

	// 增加使用计数
	s.templateRepo.IncrementUsageCount(template.ID)

	response := &models.RenderResponse{
		TemplateID:       template.ID,
		RenderedContent:  renderedContent,
		UsedVariables:    usedVars,
		MissingVariables: missingVars,
	}

	return response, nil
}

// ValidateTemplate 验证模板
func (s *templateService) ValidateTemplate(ctx context.Context, content string) (*models.TemplateValidationResult, error) {
	result := &models.TemplateValidationResult{
		IsValid: true,
		Errors:  []string{},
	}

	// 检查基本格式
	if strings.TrimSpace(content) == "" {
		result.IsValid = false
		result.Errors = append(result.Errors, "Template content cannot be empty")
		return result, nil
	}

	// 检查变量格式 {{variable}}
	variableRegex := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := variableRegex.FindAllStringSubmatch(content, -1)

	for _, match := range matches {
		varName := strings.TrimSpace(match[1])
		if varName == "" {
			result.IsValid = false
			result.Errors = append(result.Errors, "Empty variable name found")
		} else if !isValidVariableName(varName) {
			result.IsValid = false
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid variable name: %s", varName))
		}
	}

	// 检查长度限制
	if len(content) > 10000 { // 10KB限制
		result.IsValid = false
		result.Errors = append(result.Errors, "Template content too long (max 10KB)")
	}

	return result, nil
}

// DuplicateTemplate 复制模板
func (s *templateService) DuplicateTemplate(ctx context.Context, userID uint64, templateID uint64, newName string) (*models.MessageTemplate, error) {
	// 获取原模板
	original, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
	if err != nil {
		return nil, fmt.Errorf("template not found: %v", err)
	}

	// 创建副本
	duplicate := &models.MessageTemplate{
		UserID:      userID,
		Name:        newName,
		Description: original.Description + " (copy)",
		Type:        original.Type,
		Category:    original.Category,
		Content:     original.Content,
		Variables:   original.Variables,
		IsActive:    true,
	}

	if err := s.templateRepo.Create(duplicate); err != nil {
		return nil, fmt.Errorf("failed to create duplicate: %v", err)
	}

	return duplicate, nil
}

// BatchOperation 批量操作
func (s *templateService) BatchOperation(ctx context.Context, userID uint64, operation *models.BatchTemplateOperation) (*models.BatchOperationResult, error) {
	result := &models.BatchOperationResult{
		Total:      len(operation.TemplateIDs),
		Successful: 0,
		Failed:     0,
		Errors:     []string{},
	}

	for _, templateID := range operation.TemplateIDs {
		var err error

		switch operation.Operation {
		case "activate":
			err = s.updateTemplateStatus(userID, templateID, true)
		case "deactivate":
			err = s.updateTemplateStatus(userID, templateID, false)
		case "delete":
			err = s.DeleteTemplate(ctx, userID, templateID)
		case "copy":
			if operation.TargetUserID > 0 {
				_, err = s.copyTemplateToUser(userID, templateID, operation.TargetUserID)
			} else {
				err = fmt.Errorf("target user ID required for copy operation")
			}
		default:
			err = fmt.Errorf("unknown operation: %s", operation.Operation)
		}

		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Template %d: %v", templateID, err))
		} else {
			result.Successful++
		}
	}

	return result, nil
}

// ImportTemplates 导入模板
func (s *templateService) ImportTemplates(ctx context.Context, userID uint64, templates []*models.CreateTemplateRequest) (*models.ImportResult, error) {
	result := &models.ImportResult{
		Total:       len(templates),
		Successful:  0,
		Failed:      0,
		Errors:      []string{},
		ImportedIDs: []uint64{},
	}

	for i, templateReq := range templates {
		template, err := s.CreateTemplate(ctx, userID, templateReq)
		if err != nil {
			result.Failed++
			result.Errors = append(result.Errors, fmt.Sprintf("Template %d: %v", i+1, err))
		} else {
			result.Successful++
			result.ImportedIDs = append(result.ImportedIDs, template.ID)
		}
	}

	return result, nil
}

// ExportTemplates 导出模板
func (s *templateService) ExportTemplates(ctx context.Context, userID uint64, templateIDs []uint64) ([]byte, error) {
	var templates []*models.MessageTemplate

	for _, templateID := range templateIDs {
		template, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
		if err != nil {
			return nil, fmt.Errorf("failed to get template %d: %v", templateID, err)
		}
		templates = append(templates, template)
	}

	// 导出为JSON
	exportData := map[string]interface{}{
		"version":     "1.0",
		"exported_at": time.Now(),
		"templates":   templates,
	}

	return json.MarshalIndent(exportData, "", "  ")
}

// GetTemplateStats 获取模板统计
func (s *templateService) GetTemplateStats(ctx context.Context, userID uint64) (*models.TemplateStats, error) {
	// 这里应该实现统计逻辑，返回一个基本的统计结果
	stats := &models.TemplateStats{
		Total:      0,
		Active:     0,
		Inactive:   0,
		ByCategory: make(map[string]int64),
		ByType:     make(map[string]int64),
		TopUsed:    []*models.TemplateUsageInfo{},
	}

	return stats, nil
}

// GetPopularTemplates 获取热门模板
func (s *templateService) GetPopularTemplates(ctx context.Context, userID uint64, templateType models.TemplateType, limit int) ([]*models.MessageTemplate, error) {
	return s.templateRepo.GetPopularByType(userID, string(templateType), limit)
}

// 辅助方法

// updateTemplateStatus 更新模板状态
func (s *templateService) updateTemplateStatus(userID, templateID uint64, isActive bool) error {
	template, err := s.templateRepo.GetByUserIDAndID(userID, templateID)
	if err != nil {
		return err
	}

	template.IsActive = isActive
	return s.templateRepo.Update(template)
}

// copyTemplateToUser 复制模板给其他用户
func (s *templateService) copyTemplateToUser(sourceUserID, templateID, targetUserID uint64) (*models.MessageTemplate, error) {
	// 获取源模板
	source, err := s.templateRepo.GetByUserIDAndID(sourceUserID, templateID)
	if err != nil {
		return nil, err
	}

	// 创建副本
	copy := &models.MessageTemplate{
		UserID:      targetUserID,
		Name:        source.Name,
		Description: source.Description,
		Type:        source.Type,
		Category:    source.Category,
		Content:     source.Content,
		Variables:   source.Variables,
		IsActive:    true,
	}

	if err := s.templateRepo.Create(copy); err != nil {
		return nil, err
	}

	return copy, nil
}

// extractVariablesFromContent 从内容中提取变量
func extractVariablesFromContent(content string) []string {
	variableRegex := regexp.MustCompile(`\{\{([^}]+)\}\}`)
	matches := variableRegex.FindAllStringSubmatch(content, -1)

	var variables []string
	seen := make(map[string]bool)

	for _, match := range matches {
		varName := strings.TrimSpace(match[1])
		if !seen[varName] {
			variables = append(variables, varName)
			seen[varName] = true
		}
	}

	return variables
}

// isValidVariableName 检查变量名是否有效
func isValidVariableName(name string) bool {
	// 变量名应该是字母、数字和下划线的组合，不能以数字开头
	matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, name)
	return matched
}
