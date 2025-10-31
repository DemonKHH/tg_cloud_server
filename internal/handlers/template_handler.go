package handlers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/common/utils"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/services"
)

// TemplateHandler 模板处理器
type TemplateHandler struct {
	templateService services.TemplateService
	logger          *zap.Logger
}

// NewTemplateHandler 创建模板处理器
func NewTemplateHandler(templateService services.TemplateService) *TemplateHandler {
	return &TemplateHandler{
		templateService: templateService,
		logger:          logger.Get().Named("template_handler"),
	}
}

// CreateTemplate 创建模板
// @Summary 创建消息模板
// @Description 创建新的消息模板，支持变量占位符
// @Tags 模板管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body models.CreateTemplateRequest true "创建模板请求"
// @Success 201 {object} models.MessageTemplate "创建成功的模板"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates [post]
func (h *TemplateHandler) CreateTemplate(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.CreateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	template, err := h.templateService.CreateTemplate(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("Failed to create template", zap.Error(err))
		response.InternalError(c, "创建模板失败")
		return
	}

	response.SuccessWithMessage(c, "模板创建成功", template)
}

// GetTemplate 获取模板详情
// @Summary 获取模板详情
// @Description 根据ID获取特定模板的详细信息
// @Tags 模板管理
// @Produce json
// @Security ApiKeyAuth
// @Param id path uint64 true "模板ID"
// @Success 200 {object} models.MessageTemplate "模板详情"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "模板不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/{id} [get]
func (h *TemplateHandler) GetTemplate(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	templateIDStr := c.Param("id")
	templateID, err := strconv.ParseUint(templateIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的模板ID")
		return
	}

	template, err := h.templateService.GetTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		h.logger.Error("Failed to get template", zap.Error(err))
		response.NotFound(c, "模板不存在")
		return
	}

	response.Success(c, template)
}

// GetTemplates 获取模板列表
// @Summary 获取模板列表
// @Description 获取用户的模板列表，支持分页和筛选
// @Tags 模板管理
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param type query string false "模板类型" Enums(private, broadcast, groupchat, welcome, followup, other)
// @Param category query string false "模板分类" Enums(marketing, service, notification, system, custom)
// @Param keyword query string false "关键词搜索"
// @Success 200 {object} models.PaginationResponse "模板列表"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates [get]
func (h *TemplateHandler) GetTemplates(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 解析查询参数
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	// 构建筛选条件
	filter := &models.TemplateFilter{
		UserID:    userID,
		Keyword:   c.Query("keyword"),
		SortBy:    c.DefaultQuery("sort_by", "created_at"),
		SortOrder: c.DefaultQuery("sort_order", "desc"),
	}

	if templateType := c.Query("type"); templateType != "" {
		filter.Type = models.TemplateType(templateType)
	}

	if category := c.Query("category"); category != "" {
		filter.Category = models.TemplateCategory(category)
	}

	if isActive := c.Query("is_active"); isActive != "" {
		if isActive == "true" {
			active := true
			filter.IsActive = &active
		} else if isActive == "false" {
			active := false
			filter.IsActive = &active
		}
	}

	templates, total, err := h.templateService.GetTemplates(c.Request.Context(), userID, filter)
	if err != nil {
		h.logger.Error("Failed to get templates", zap.Error(err))
		response.InternalError(c, "获取模板列表失败")
		return
	}

	response.Paginated(c, templates, page, limit, total)
}

// UpdateTemplate 更新模板
// @Summary 更新模板
// @Description 更新指定ID的模板信息
// @Tags 模板管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path uint64 true "模板ID"
// @Param body body models.UpdateTemplateRequest true "更新模板请求"
// @Success 200 {object} models.MessageTemplate "更新后的模板"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "模板不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/{id} [put]
func (h *TemplateHandler) UpdateTemplate(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	templateIDStr := c.Param("id")
	templateID, err := strconv.ParseUint(templateIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的模板ID")
		return
	}

	var req models.UpdateTemplateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	template, err := h.templateService.UpdateTemplate(c.Request.Context(), userID, templateID, &req)
	if err != nil {
		h.logger.Error("Failed to update template", zap.Error(err))
		response.InternalError(c, "更新模板失败")
		return
	}

	response.SuccessWithMessage(c, "模板更新成功", template)
}

// DeleteTemplate 删除模板
// @Summary 删除模板
// @Description 删除指定ID的模板
// @Tags 模板管理
// @Security ApiKeyAuth
// @Param id path uint64 true "模板ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "模板不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/{id} [delete]
func (h *TemplateHandler) DeleteTemplate(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	templateIDStr := c.Param("id")
	templateID, err := strconv.ParseUint(templateIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的模板ID")
		return
	}

	err = h.templateService.DeleteTemplate(c.Request.Context(), userID, templateID)
	if err != nil {
		h.logger.Error("Failed to delete template", zap.Error(err))
		response.InternalError(c, "删除模板失败")
		return
	}

	response.SuccessWithMessage(c, "模板删除成功", nil)
}

// RenderTemplate 渲染模板
// @Summary 渲染模板
// @Description 使用提供的变量渲染模板内容
// @Tags 模板操作
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body models.RenderRequest true "渲染请求"
// @Success 200 {object} models.RenderResponse "渲染结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "模板不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/render [post]
func (h *TemplateHandler) RenderTemplate(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.RenderRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	renderResp, err := h.templateService.RenderTemplate(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("Failed to render template", zap.Error(err))
		response.InternalError(c, "渲染模板失败")
		return
	}

	response.Success(c, renderResp)
}

// ValidateTemplate 验证模板
// @Summary 验证模板
// @Description 验证模板内容的有效性
// @Tags 模板操作
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body map[string]string true "验证请求" example:{"content":"Hello {{name}}!"}
// @Success 200 {object} models.TemplateValidationResult "验证结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/validate [post]
func (h *TemplateHandler) ValidateTemplate(c *gin.Context) {
	_, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req struct {
		Content string `json:"content" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	result, err := h.templateService.ValidateTemplate(c.Request.Context(), req.Content)
	if err != nil {
		h.logger.Error("Failed to validate template", zap.Error(err))
		response.InternalError(c, "验证模板失败")
		return
	}

	response.Success(c, result)
}

// DuplicateTemplate 复制模板
// @Summary 复制模板
// @Description 复制指定ID的模板为新模板
// @Tags 模板操作
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param id path uint64 true "模板ID"
// @Param body body map[string]string true "复制请求" example:{"new_name":"My Template Copy"}
// @Success 201 {object} models.MessageTemplate "复制的新模板"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "模板不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/{id}/duplicate [post]
func (h *TemplateHandler) DuplicateTemplate(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	templateIDStr := c.Param("id")
	templateID, err := strconv.ParseUint(templateIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的模板ID")
		return
	}

	var req struct {
		NewName string `json:"new_name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	template, err := h.templateService.DuplicateTemplate(c.Request.Context(), userID, templateID, req.NewName)
	if err != nil {
		h.logger.Error("Failed to duplicate template", zap.Error(err))
		response.InternalError(c, "复制模板失败")
		return
	}

	response.SuccessWithMessage(c, "模板复制成功", template)
}

// BatchOperation 批量操作模板
// @Summary 批量操作模板
// @Description 对多个模板执行批量操作（激活、停用、删除、复制）
// @Tags 模板操作
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body models.BatchTemplateOperation true "批量操作请求"
// @Success 200 {object} models.BatchOperationResult "批量操作结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/batch [post]
func (h *TemplateHandler) BatchOperation(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req models.BatchTemplateOperation
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	result, err := h.templateService.BatchOperation(c.Request.Context(), userID, &req)
	if err != nil {
		h.logger.Error("Failed to perform batch operation", zap.Error(err))
		response.InternalError(c, "批量操作失败")
		return
	}

	response.Success(c, result)
}

// ImportTemplates 导入模板
// @Summary 导入模板
// @Description 批量导入多个模板
// @Tags 模板操作
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body []models.CreateTemplateRequest true "导入模板列表"
// @Success 200 {object} models.ImportResult "导入结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/import [post]
func (h *TemplateHandler) ImportTemplates(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var templates []*models.CreateTemplateRequest
	if err := c.ShouldBindJSON(&templates); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	result, err := h.templateService.ImportTemplates(c.Request.Context(), userID, templates)
	if err != nil {
		h.logger.Error("Failed to import templates", zap.Error(err))
		response.InternalError(c, "导入模板失败")
		return
	}

	response.Success(c, result)
}

// ExportTemplates 导出模板
// @Summary 导出模板
// @Description 导出指定的模板为JSON格式
// @Tags 模板操作
// @Produce json
// @Security ApiKeyAuth
// @Param template_ids query string true "模板ID列表，逗号分隔" example:"1,2,3"
// @Success 200 {array} models.MessageTemplate "导出的模板数据"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/export [get]
func (h *TemplateHandler) ExportTemplates(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	templateIDsStr := c.Query("template_ids")
	if templateIDsStr == "" {
		response.InvalidParam(c, "template_ids参数是必需的")
		return
	}

	// 解析模板ID列表
	templateIDsStrSlice := strings.Split(templateIDsStr, ",")
	var templateIDs []uint64
	for _, idStr := range templateIDsStrSlice {
		id, err := strconv.ParseUint(strings.TrimSpace(idStr), 10, 64)
		if err != nil {
			response.InvalidParam(c, "无效的模板ID: "+idStr)
			return
		}
		templateIDs = append(templateIDs, id)
	}

	data, err := h.templateService.ExportTemplates(c.Request.Context(), userID, templateIDs)
	if err != nil {
		h.logger.Error("Failed to export templates", zap.Error(err))
		response.InternalError(c, "导出模板失败")
		return
	}

	// 设置响应头
	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=templates.json")
	c.Data(http.StatusOK, "application/json", data)
}

// GetTemplateStats 获取模板统计
// @Summary 获取模板统计
// @Description 获取用户的模板使用统计信息
// @Tags 模板统计
// @Produce json
// @Security ApiKeyAuth
// @Success 200 {object} models.TemplateStats "模板统计数据"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/stats [get]
func (h *TemplateHandler) GetTemplateStats(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	stats, err := h.templateService.GetTemplateStats(c.Request.Context(), userID)
	if err != nil {
		h.logger.Error("Failed to get template stats", zap.Error(err))
		response.InternalError(c, "获取模板统计失败")
		return
	}

	response.Success(c, stats)
}

// GetPopularTemplates 获取热门模板
// @Summary 获取热门模板
// @Description 获取指定类型的热门使用模板
// @Tags 模板统计
// @Produce json
// @Security ApiKeyAuth
// @Param type query string false "模板类型" Enums(private, broadcast, groupchat, welcome, followup, other)
// @Param limit query int false "返回数量" default(10)
// @Success 200 {array} models.MessageTemplate "热门模板列表"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/templates/popular [get]
func (h *TemplateHandler) GetPopularTemplates(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	templateType := models.TemplateType(c.DefaultQuery("type", ""))
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 50 {
		limit = 10
	}

	templates, err := h.templateService.GetPopularTemplates(c.Request.Context(), userID, templateType, limit)
	if err != nil {
		h.logger.Error("Failed to get popular templates", zap.Error(err))
		response.InternalError(c, "获取热门模板失败")
		return
	}

	response.Success(c, templates)
}
