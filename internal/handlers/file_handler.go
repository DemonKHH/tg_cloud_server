package handlers

import (
	"fmt"
	"net/http"
	"path/filepath"
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

// FileHandler 文件处理器
type FileHandler struct {
	fileService services.FileService
	logger      *zap.Logger
}

// NewFileHandler 创建文件处理器
func NewFileHandler(fileService services.FileService) *FileHandler {
	return &FileHandler{
		fileService: fileService,
		logger:      logger.Get().Named("file_handler"),
	}
}

// UploadFile 上传文件
// @Summary 上传文件
// @Description 上传单个文件到服务器
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Security ApiKeyAuth
// @Param file formData file true "上传的文件"
// @Param category formData string true "文件分类" Enums(avatar, message, template, attachment, export, import)
// @Success 200 {object} models.FileInfo "上传成功的文件信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 413 {object} map[string]string "文件过大"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/upload [post]
func (h *FileHandler) UploadFile(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 获取上传的文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		response.InvalidParam(c, "无法从请求中获取文件")
		return
	}
	defer file.Close()

	// 获取文件分类
	categoryStr := c.PostForm("category")
	if categoryStr == "" {
		categoryStr = string(models.CategoryAttachment) // 默认分类
	}
	category := models.FileCategory(categoryStr)

	// 验证文件大小 (10MB限制)
	if header.Size > 10*1024*1024 {
		response.InvalidParam(c, "文件大小超过10MB限制")
		return
	}

	// 验证文件类型
	ext := strings.ToLower(filepath.Ext(header.Filename))
	allowedExts := map[string]bool{
		".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".webp": true, // 图片
		".mp4": true, ".avi": true, ".mov": true, ".mkv": true, // 视频
		".mp3": true, ".wav": true, ".ogg": true, // 音频
		".pdf": true, ".doc": true, ".docx": true, ".txt": true, ".csv": true, ".xlsx": true, // 文档
		".zip": true, ".rar": true, ".7z": true, // 压缩包
	}

	if !allowedExts[ext] {
		response.InvalidParam(c, "不允许的文件类型")
		return
	}

	fileInfo, err := h.fileService.UploadFile(c.Request.Context(), userID, file, header, category)
	if err != nil {
		h.logger.Error("Failed to upload file", zap.Error(err))
		response.InternalError(c, "文件上传失败")
		return
	}

	response.SuccessWithMessage(c, "文件上传成功", fileInfo)
}

// UploadFromURL 从URL上传文件
// @Summary 从URL上传文件
// @Description 从指定URL下载并上传文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body map[string]string true "上传请求" example:{"url":"https://example.com/file.jpg","category":"message"}
// @Success 200 {object} models.FileInfo "上传成功的文件信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/upload-url [post]
func (h *FileHandler) UploadFromURL(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req struct {
		URL      string `json:"url" binding:"required,url"`
		Category string `json:"category"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	category := models.FileCategory(req.Category)
	if req.Category == "" {
		category = models.CategoryAttachment
	}

	fileInfo, err := h.fileService.UploadFromURL(c.Request.Context(), userID, req.URL, category)
	if err != nil {
		h.logger.Error("Failed to upload file from URL", zap.Error(err))
		response.InternalError(c, "从URL上传文件失败")
		return
	}

	response.SuccessWithMessage(c, "文件上传成功", fileInfo)
}

// GetFile 获取文件信息
// @Summary 获取文件信息
// @Description 根据文件ID获取文件的详细信息
// @Tags 文件管理
// @Produce json
// @Security ApiKeyAuth
// @Param id path uint64 true "文件ID"
// @Success 200 {object} models.FileInfo "文件信息"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "文件不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/{id} [get]
func (h *FileHandler) GetFile(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	fileIDStr := c.Param("id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的文件ID")
		return
	}

	fileInfo, err := h.fileService.GetFile(c.Request.Context(), userID, fileID)
	if err != nil {
		h.logger.Error("Failed to get file", zap.Error(err))
		response.NotFound(c, "文件不存在")
		return
	}

	response.Success(c, fileInfo)
}

// GetFiles 获取文件列表
// @Summary 获取文件列表
// @Description 获取用户的文件列表，支持分页和分类筛选
// @Tags 文件管理
// @Produce json
// @Security ApiKeyAuth
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(20)
// @Param category query string false "文件分类" Enums(avatar, message, template, attachment, export, import)
// @Success 200 {object} models.PaginationResponse "文件列表"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files [get]
func (h *FileHandler) GetFiles(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 解析查询参数
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "20")
	categoryStr := c.Query("category")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 20
	}

	category := models.FileCategory(categoryStr)

	files, total, err := h.fileService.GetFilesByUser(c.Request.Context(), userID, category, page, limit)
	if err != nil {
		h.logger.Error("Failed to get files", zap.Error(err))
		response.InternalError(c, "获取文件列表失败")
		return
	}

	response.Paginated(c, files, page, limit, total)
}

// DownloadFile 下载文件
// @Summary 下载文件
// @Description 下载指定ID的文件
// @Tags 文件管理
// @Produce application/octet-stream
// @Security ApiKeyAuth
// @Param id path uint64 true "文件ID"
// @Success 200 {file} file "文件内容"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "文件不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/{id}/download [get]
func (h *FileHandler) DownloadFile(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	fileIDStr := c.Param("id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的文件ID")
		return
	}

	// 获取文件信息
	fileInfo, err := h.fileService.GetFile(c.Request.Context(), userID, fileID)
	if err != nil {
		h.logger.Error("Failed to get file", zap.Error(err))
		response.NotFound(c, "文件不存在")
		return
	}

	// 获取文件内容
	content, err := h.fileService.GetFileContent(c.Request.Context(), userID, fileID)
	if err != nil {
		h.logger.Error("Failed to get file content", zap.Error(err))
		response.InternalError(c, "获取文件内容失败")
		return
	}

	// 设置响应头
	c.Header("Content-Type", fileInfo.ContentType)
	c.Header("Content-Disposition", `attachment; filename="`+fileInfo.OriginalName+`"`)
	c.Header("Content-Length", strconv.FormatInt(fileInfo.FileSize, 10))

	c.Data(http.StatusOK, fileInfo.ContentType, content)
}

// PreviewFile 预览文件
// @Summary 预览文件
// @Description 在线预览文件（主要用于图片）
// @Tags 文件管理
// @Produce application/octet-stream
// @Security ApiKeyAuth
// @Param id path uint64 true "文件ID"
// @Success 200 {file} file "文件预览内容"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "文件不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/{id}/preview [get]
func (h *FileHandler) PreviewFile(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	fileIDStr := c.Param("id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的文件ID")
		return
	}

	// 获取文件信息
	fileInfo, err := h.fileService.GetFile(c.Request.Context(), userID, fileID)
	if err != nil {
		h.logger.Error("Failed to get file", zap.Error(err))
		response.NotFound(c, "文件不存在")
		return
	}

	// 检查是否支持预览
	if fileInfo.FileType != models.FileTypeImage {
		response.InvalidParam(c, "该文件类型不支持预览")
		return
	}

	// 获取或生成预览
	previewURL, err := h.fileService.GeneratePreview(c.Request.Context(), fileID)
	if err != nil {
		h.logger.Error("Failed to generate preview", zap.Error(err))
		// 如果预览生成失败，直接返回原文件
		content, err := h.fileService.GetFileContent(c.Request.Context(), userID, fileID)
		if err != nil {
			response.InternalError(c, "获取文件内容失败")
			return
		}

		c.Header("Content-Type", fileInfo.ContentType)
		c.Data(http.StatusOK, fileInfo.ContentType, content)
		return
	}

	response.Success(c, gin.H{"preview_url": previewURL})
}

// DeleteFile 删除文件
// @Summary 删除文件
// @Description 删除指定ID的文件
// @Tags 文件管理
// @Security ApiKeyAuth
// @Param id path uint64 true "文件ID"
// @Success 200 {object} map[string]string "删除成功"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "文件不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/{id} [delete]
func (h *FileHandler) DeleteFile(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	fileIDStr := c.Param("id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的文件ID")
		return
	}

	err = h.fileService.DeleteFile(c.Request.Context(), userID, fileID)
	if err != nil {
		h.logger.Error("Failed to delete file", zap.Error(err))
		response.InternalError(c, "删除文件失败")
		return
	}

	response.SuccessWithMessage(c, "文件删除成功", nil)
}

// BatchUpload 批量上传文件
// @Summary 批量上传文件
// @Description 一次性上传多个文件
// @Tags 文件管理
// @Accept multipart/form-data
// @Produce json
// @Security ApiKeyAuth
// @Param files formData []file true "上传的文件列表"
// @Param category formData string true "文件分类" Enums(avatar, message, template, attachment, export, import)
// @Success 200 {array} models.FileInfo "上传成功的文件信息列表"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/batch-upload [post]
func (h *FileHandler) BatchUpload(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	// 解析多文件上传
	form, err := c.MultipartForm()
	if err != nil {
		response.InvalidParam(c, "无法解析multipart表单")
		return
	}

	files := form.File["files"]
	if len(files) == 0 {
		response.InvalidParam(c, "未提供文件")
		return
	}

	// 获取文件分类
	categoryStr := c.PostForm("category")
	if categoryStr == "" {
		categoryStr = string(models.CategoryAttachment)
	}
	category := models.FileCategory(categoryStr)

	// 限制批量上传数量
	if len(files) > 10 {
		response.InvalidParam(c, "每批最多允许10个文件")
		return
	}

	var uploadedFiles []*models.FileInfo
	var errors []string

	for _, fileHeader := range files {
		file, err := fileHeader.Open()
		if err != nil {
			errors = append(errors, fmt.Sprintf("无法打开文件 %s: %v", fileHeader.Filename, err))
			continue
		}

		fileInfo, err := h.fileService.UploadFile(c.Request.Context(), userID, file, fileHeader, category)
		if err != nil {
			errors = append(errors, fmt.Sprintf("上传文件 %s 失败: %v", fileHeader.Filename, err))
			file.Close()
			continue
		}

		uploadedFiles = append(uploadedFiles, fileInfo)
		file.Close()
	}

	data := gin.H{
		"uploaded_files": uploadedFiles,
		"success_count":  len(uploadedFiles),
		"total_count":    len(files),
	}

	if len(errors) > 0 {
		data["errors"] = errors
		data["error_count"] = len(errors)
	}

	response.Success(c, data)
}

// BatchDelete 批量删除文件
// @Summary 批量删除文件
// @Description 批量删除多个文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Security ApiKeyAuth
// @Param body body map[string][]uint64 true "删除请求" example:{"file_ids":[1,2,3]}
// @Success 200 {object} map[string]interface{} "删除结果"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/batch-delete [delete]
func (h *FileHandler) BatchDelete(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	var req struct {
		FileIDs []uint64 `json:"file_ids" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		response.InvalidParam(c, err.Error())
		return
	}

	if len(req.FileIDs) > 50 {
		response.InvalidParam(c, "每批最多允许删除50个文件")
		return
	}

	deletedCount, err := h.fileService.BatchDelete(c.Request.Context(), userID, req.FileIDs)
	if err != nil {
		h.logger.Error("Failed to batch delete files", zap.Error(err))
		response.InternalError(c, "批量删除文件失败")
		return
	}

	response.Success(c, gin.H{
		"deleted_count": deletedCount,
		"total_count":   len(req.FileIDs),
		"success_rate":  float64(deletedCount) / float64(len(req.FileIDs)) * 100,
	})
}

// GetFileURL 获取文件访问URL
// @Summary 获取文件访问URL
// @Description 获取文件的临时访问URL
// @Tags 文件管理
// @Produce json
// @Security ApiKeyAuth
// @Param id path uint64 true "文件ID"
// @Success 200 {object} map[string]string "文件URL"
// @Failure 400 {object} map[string]string "请求错误"
// @Failure 401 {object} map[string]string "未授权"
// @Failure 404 {object} map[string]string "文件不存在"
// @Failure 500 {object} map[string]string "服务器错误"
// @Router /api/v1/files/{id}/url [get]
func (h *FileHandler) GetFileURL(c *gin.Context) {
	userID, err := utils.GetUserID(c)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	fileIDStr := c.Param("id")
	fileID, err := strconv.ParseUint(fileIDStr, 10, 64)
	if err != nil {
		response.InvalidParam(c, "无效的文件ID")
		return
	}

	url, err := h.fileService.GetFileURL(c.Request.Context(), userID, fileID)
	if err != nil {
		h.logger.Error("Failed to get file URL", zap.Error(err))
		response.NotFound(c, "获取文件URL失败")
		return
	}

	response.Success(c, gin.H{
		"url":        url,
		"expires_in": "1 hour", // 临时URL有效期
	})
}
