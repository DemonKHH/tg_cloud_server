package services

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/repository"
)

// FileType 文件类型枚举
type FileType string

const (
	FileTypeImage    FileType = "image"
	FileTypeVideo    FileType = "video"
	FileTypeAudio    FileType = "audio"
	FileTypeDocument FileType = "document"
	FileTypeOther    FileType = "other"
)

// FileCategory 文件分类
type FileCategory string

const (
	CategoryAvatar     FileCategory = "avatar"
	CategoryMessage    FileCategory = "message"
	CategoryTemplate   FileCategory = "template"
	CategoryAttachment FileCategory = "attachment"
	CategoryExport     FileCategory = "export"
	CategoryImport     FileCategory = "import"
)

// FileInfo 文件信息
type FileInfo struct {
	ID           uint64       `json:"id"`
	UserID       uint64       `json:"user_id"`
	OriginalName string       `json:"original_name"`
	FileName     string       `json:"file_name"`
	FilePath     string       `json:"file_path"`
	FileSize     int64        `json:"file_size"`
	FileType     FileType     `json:"file_type"`
	Category     FileCategory `json:"category"`
	MimeType     string       `json:"mime_type"`
	MD5Hash      string       `json:"md5_hash"`
	UploadTime   time.Time    `json:"upload_time"`
	AccessCount  int64        `json:"access_count"`
	IsPublic     bool         `json:"is_public"`
	ExpiresAt    *time.Time   `json:"expires_at,omitempty"`
}

// FileService 文件服务接口
type FileService interface {
	// 文件上传
	UploadFile(ctx context.Context, userID uint64, file multipart.File, header *multipart.FileHeader, category FileCategory) (*FileInfo, error)
	UploadFromURL(ctx context.Context, userID uint64, url string, category FileCategory) (*FileInfo, error)
	UploadFromBytes(ctx context.Context, userID uint64, data []byte, fileName string, category FileCategory) (*FileInfo, error)

	// 文件管理
	GetFile(ctx context.Context, userID uint64, fileID uint64) (*FileInfo, error)
	GetFilesByUser(ctx context.Context, userID uint64, category FileCategory, page, limit int) ([]*FileInfo, int64, error)
	DeleteFile(ctx context.Context, userID uint64, fileID uint64) error
	UpdateFileInfo(ctx context.Context, userID uint64, fileID uint64, updates map[string]interface{}) error

	// 文件操作
	GetFileContent(ctx context.Context, userID uint64, fileID uint64) ([]byte, error)
	GetFileURL(ctx context.Context, userID uint64, fileID uint64) (string, error)
	GeneratePreview(ctx context.Context, fileID uint64) (string, error)

	// 批量操作
	BatchUpload(ctx context.Context, userID uint64, files []multipart.File, headers []*multipart.FileHeader, category FileCategory) ([]*FileInfo, error)
	BatchDelete(ctx context.Context, userID uint64, fileIDs []uint64) (int, error)

	// 清理操作
	CleanupTempFiles(ctx context.Context) error
	CleanupExpiredFiles(ctx context.Context) error
}

// fileService 文件服务实现
type fileService struct {
	fileRepo   repository.FileRepository
	logger     *zap.Logger
	uploadPath string
	baseURL    string

	// 配置
	maxFileSize        int64    // 最大文件大小
	allowedTypes       []string // 允许的文件类型
	compressionQuality int      // 压缩质量
}

// NewFileService 创建文件服务
func NewFileService(fileRepo repository.FileRepository, config map[string]interface{}) FileService {
	service := &fileService{
		fileRepo:    fileRepo,
		logger:      logger.Get().Named("file_service"),
		uploadPath:  "./uploads",
		baseURL:     "http://localhost:8080",
		maxFileSize: 50 * 1024 * 1024, // 50MB
		allowedTypes: []string{
			"image/jpeg", "image/png", "image/gif", "image/webp",
			"video/mp4", "video/avi", "video/mov",
			"audio/mp3", "audio/wav", "audio/ogg",
			"application/pdf", "application/doc", "application/docx",
			"text/plain", "text/csv",
		},
		compressionQuality: 85,
	}

	// 从配置中加载设置
	if path, ok := config["upload_path"].(string); ok {
		service.uploadPath = path
	}
	if url, ok := config["base_url"].(string); ok {
		service.baseURL = url
	}
	if size, ok := config["max_file_size"].(int64); ok {
		service.maxFileSize = size
	}

	// 确保上传目录存在
	os.MkdirAll(service.uploadPath, 0755)

	return service
}

// UploadFile 上传文件
func (s *fileService) UploadFile(ctx context.Context, userID uint64, file multipart.File, header *multipart.FileHeader, category FileCategory) (*FileInfo, error) {
	s.logger.Info("Uploading file",
		zap.Uint64("user_id", userID),
		zap.String("filename", header.Filename),
		zap.Int64("size", header.Size),
		zap.String("category", string(category)))

	// 验证文件大小
	if header.Size > s.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", header.Size, s.maxFileSize)
	}

	// 检测文件类型
	fileType := s.detectFileType(header.Header.Get("Content-Type"))
	mimeType := header.Header.Get("Content-Type")

	// 验证文件类型
	if !s.isAllowedType(mimeType) {
		return nil, fmt.Errorf("file type %s is not allowed", mimeType)
	}

	// 生成文件名和路径
	fileName := s.generateFileName(header.Filename, fileType)
	filePath := s.generateFilePath(userID, category, fileName)
	fullPath := filepath.Join(s.uploadPath, filePath)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		s.logger.Error("Failed to create directory", zap.Error(err))
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 保存文件
	dst, err := os.Create(fullPath)
	if err != nil {
		s.logger.Error("Failed to create file", zap.Error(err))
		return nil, fmt.Errorf("failed to create file: %w", err)
	}
	defer dst.Close()

	// 复制文件内容并计算MD5
	hash := md5.New()
	multiWriter := io.MultiWriter(dst, hash)

	if _, err := io.Copy(multiWriter, file); err != nil {
		s.logger.Error("Failed to copy file", zap.Error(err))
		os.Remove(fullPath) // 清理失败的文件
		return nil, fmt.Errorf("failed to copy file: %w", err)
	}

	md5Hash := fmt.Sprintf("%x", hash.Sum(nil))

	// 创建文件信息
	fileInfo := &FileInfo{
		UserID:       userID,
		OriginalName: header.Filename,
		FileName:     fileName,
		FilePath:     filePath,
		FileSize:     header.Size,
		FileType:     fileType,
		Category:     category,
		MimeType:     mimeType,
		MD5Hash:      md5Hash,
		UploadTime:   time.Now(),
		IsPublic:     false,
	}

	// 保存到数据库
	if err := s.fileRepo.Create(fileInfo); err != nil {
		s.logger.Error("Failed to save file info", zap.Error(err))
		os.Remove(fullPath) // 清理文件
		return nil, fmt.Errorf("failed to save file info: %w", err)
	}

	s.logger.Info("File uploaded successfully",
		zap.Uint64("file_id", fileInfo.ID),
		zap.String("file_path", filePath))

	return fileInfo, nil
}

// UploadFromURL 从URL上传文件
func (s *fileService) UploadFromURL(ctx context.Context, userID uint64, url string, category FileCategory) (*FileInfo, error) {
	s.logger.Info("Uploading file from URL",
		zap.Uint64("user_id", userID),
		zap.String("url", url),
		zap.String("category", string(category)))

	// TODO: 实现从URL下载文件的逻辑
	// 1. 发送HTTP GET请求
	// 2. 检查Content-Type和Content-Length
	// 3. 下载文件内容
	// 4. 保存文件

	return nil, fmt.Errorf("upload from URL not implemented yet")
}

// UploadFromBytes 从字节数组上传文件
func (s *fileService) UploadFromBytes(ctx context.Context, userID uint64, data []byte, fileName string, category FileCategory) (*FileInfo, error) {
	s.logger.Info("Uploading file from bytes",
		zap.Uint64("user_id", userID),
		zap.String("filename", fileName),
		zap.Int("size", len(data)),
		zap.String("category", string(category)))

	// 验证文件大小
	if int64(len(data)) > s.maxFileSize {
		return nil, fmt.Errorf("file size %d exceeds maximum allowed size %d", len(data), s.maxFileSize)
	}

	// 检测文件类型
	mimeType := s.detectMimeType(data, fileName)
	fileType := s.detectFileType(mimeType)

	// 验证文件类型
	if !s.isAllowedType(mimeType) {
		return nil, fmt.Errorf("file type %s is not allowed", mimeType)
	}

	// 生成文件名和路径
	generatedFileName := s.generateFileName(fileName, fileType)
	filePath := s.generateFilePath(userID, category, generatedFileName)
	fullPath := filepath.Join(s.uploadPath, filePath)

	// 确保目录存在
	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		s.logger.Error("Failed to create directory", zap.Error(err))
		return nil, fmt.Errorf("failed to create directory: %w", err)
	}

	// 保存文件
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		s.logger.Error("Failed to write file", zap.Error(err))
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	// 计算MD5
	hash := md5.Sum(data)
	md5Hash := fmt.Sprintf("%x", hash)

	// 创建文件信息
	fileInfo := &FileInfo{
		UserID:       userID,
		OriginalName: fileName,
		FileName:     generatedFileName,
		FilePath:     filePath,
		FileSize:     int64(len(data)),
		FileType:     fileType,
		Category:     category,
		MimeType:     mimeType,
		MD5Hash:      md5Hash,
		UploadTime:   time.Now(),
		IsPublic:     false,
	}

	// 保存到数据库
	if err := s.fileRepo.Create(fileInfo); err != nil {
		s.logger.Error("Failed to save file info", zap.Error(err))
		os.Remove(fullPath) // 清理文件
		return nil, fmt.Errorf("failed to save file info: %w", err)
	}

	s.logger.Info("File uploaded successfully from bytes",
		zap.Uint64("file_id", fileInfo.ID),
		zap.String("file_path", filePath))

	return fileInfo, nil
}

// GetFile 获取文件信息
func (s *fileService) GetFile(ctx context.Context, userID uint64, fileID uint64) (*FileInfo, error) {
	fileInfo, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return nil, err
	}

	// 增加访问计数
	s.fileRepo.IncrementAccessCount(fileID)

	return fileInfo, nil
}

// GetFilesByUser 获取用户文件列表
func (s *fileService) GetFilesByUser(ctx context.Context, userID uint64, category FileCategory, page, limit int) ([]*FileInfo, int64, error) {
	offset := (page - 1) * limit
	return s.fileRepo.GetByUserIDAndCategory(userID, string(category), offset, limit)
}

// DeleteFile 删除文件
func (s *fileService) DeleteFile(ctx context.Context, userID uint64, fileID uint64) error {
	// 获取文件信息
	fileInfo, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return err
	}

	// 删除物理文件
	fullPath := filepath.Join(s.uploadPath, fileInfo.FilePath)
	if err := os.Remove(fullPath); err != nil && !os.IsNotExist(err) {
		s.logger.Error("Failed to delete physical file",
			zap.String("path", fullPath),
			zap.Error(err))
	}

	// 从数据库删除
	if err := s.fileRepo.Delete(fileID); err != nil {
		s.logger.Error("Failed to delete file from database", zap.Error(err))
		return err
	}

	s.logger.Info("File deleted successfully",
		zap.Uint64("file_id", fileID),
		zap.String("file_path", fileInfo.FilePath))

	return nil
}

// GetFileContent 获取文件内容
func (s *fileService) GetFileContent(ctx context.Context, userID uint64, fileID uint64) ([]byte, error) {
	// 获取文件信息
	fileInfo, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return nil, err
	}

	// 读取文件内容
	fullPath := filepath.Join(s.uploadPath, fileInfo.FilePath)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		s.logger.Error("Failed to read file content",
			zap.String("path", fullPath),
			zap.Error(err))
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	// 增加访问计数
	s.fileRepo.IncrementAccessCount(fileID)

	return content, nil
}

// GetFileURL 获取文件访问URL
func (s *fileService) GetFileURL(ctx context.Context, userID uint64, fileID uint64) (string, error) {
	// 验证文件存在且属于用户
	_, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return "", err
	}

	// 生成访问URL
	url := fmt.Sprintf("%s/api/v1/files/%d/download", s.baseURL, fileID)
	return url, nil
}

// 辅助方法

func (s *fileService) detectFileType(mimeType string) FileType {
	switch {
	case strings.HasPrefix(mimeType, "image/"):
		return FileTypeImage
	case strings.HasPrefix(mimeType, "video/"):
		return FileTypeVideo
	case strings.HasPrefix(mimeType, "audio/"):
		return FileTypeAudio
	case strings.HasPrefix(mimeType, "application/") || strings.HasPrefix(mimeType, "text/"):
		return FileTypeDocument
	default:
		return FileTypeOther
	}
}

func (s *fileService) detectMimeType(data []byte, fileName string) string {
	// 简单的MIME类型检测，实际应该使用http.DetectContentType或magic number检测
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".mp4":
		return "video/mp4"
	case ".mp3":
		return "audio/mp3"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func (s *fileService) isAllowedType(mimeType string) bool {
	for _, allowedType := range s.allowedTypes {
		if mimeType == allowedType {
			return true
		}
	}
	return false
}

func (s *fileService) generateFileName(originalName string, fileType FileType) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)

	// 生成时间戳
	timestamp := time.Now().Format("20060102150405")

	// 生成随机后缀
	hash := md5.Sum([]byte(originalName + timestamp))
	suffix := fmt.Sprintf("%x", hash)[:8]

	return fmt.Sprintf("%s_%s_%s%s", name, timestamp, suffix, ext)
}

func (s *fileService) generateFilePath(userID uint64, category FileCategory, fileName string) string {
	// 按年月组织目录结构
	now := time.Now()
	return filepath.Join(
		fmt.Sprintf("user_%d", userID),
		string(category),
		now.Format("2006"),
		now.Format("01"),
		fileName,
	)
}

// BatchUpload 批量上传文件
func (s *fileService) BatchUpload(ctx context.Context, userID uint64, files []multipart.File, headers []*multipart.FileHeader, category FileCategory) ([]*FileInfo, error) {
	if len(files) != len(headers) {
		return nil, fmt.Errorf("files and headers count mismatch")
	}

	var results []*FileInfo
	var errors []error

	for i, file := range files {
		fileInfo, err := s.UploadFile(ctx, userID, file, headers[i], category)
		if err != nil {
			errors = append(errors, err)
			s.logger.Error("Failed to upload file in batch",
				zap.Int("index", i),
				zap.String("filename", headers[i].Filename),
				zap.Error(err))
			continue
		}
		results = append(results, fileInfo)
	}

	if len(errors) > 0 {
		s.logger.Warn("Some files failed to upload in batch",
			zap.Int("total", len(files)),
			zap.Int("success", len(results)),
			zap.Int("failed", len(errors)))
	}

	return results, nil
}

// BatchDelete 批量删除文件
func (s *fileService) BatchDelete(ctx context.Context, userID uint64, fileIDs []uint64) (int, error) {
	successCount := 0

	for _, fileID := range fileIDs {
		if err := s.DeleteFile(ctx, userID, fileID); err != nil {
			s.logger.Error("Failed to delete file in batch",
				zap.Uint64("file_id", fileID),
				zap.Error(err))
			continue
		}
		successCount++
	}

	s.logger.Info("Batch delete completed",
		zap.Int("total", len(fileIDs)),
		zap.Int("success", successCount))

	return successCount, nil
}

// UpdateFileInfo 更新文件信息
func (s *fileService) UpdateFileInfo(ctx context.Context, userID uint64, fileID uint64, updates map[string]interface{}) error {
	// 验证文件所有权
	_, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return err
	}

	return s.fileRepo.Update(fileID, updates)
}

// GeneratePreview 生成文件预览
func (s *fileService) GeneratePreview(ctx context.Context, fileID uint64) (string, error) {
	// TODO: 实现文件预览生成逻辑
	// 对于图片：生成缩略图
	// 对于视频：生成封面图
	// 对于文档：生成首页预览
	return "", fmt.Errorf("preview generation not implemented yet")
}

// CleanupTempFiles 清理临时文件
func (s *fileService) CleanupTempFiles(ctx context.Context) error {
	// TODO: 实现临时文件清理逻辑
	return nil
}

// CleanupExpiredFiles 清理过期文件
func (s *fileService) CleanupExpiredFiles(ctx context.Context) error {
	expiredFiles, err := s.fileRepo.GetExpiredFiles()
	if err != nil {
		return err
	}

	for _, file := range expiredFiles {
		// 删除物理文件
		fullPath := filepath.Join(s.uploadPath, file.FilePath)
		os.Remove(fullPath)

		// 删除数据库记录
		s.fileRepo.Delete(file.ID)
	}

	s.logger.Info("Expired files cleaned up", zap.Int("count", len(expiredFiles)))
	return nil
}
