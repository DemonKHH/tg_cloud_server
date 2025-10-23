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
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

// FileService 文件服务接口
type FileService interface {
	// 文件上传
	UploadFile(ctx context.Context, userID uint64, file multipart.File, header *multipart.FileHeader, category models.FileCategory) (*models.FileInfo, error)
	UploadFromURL(ctx context.Context, userID uint64, url string, category models.FileCategory) (*models.FileInfo, error)
	UploadFromBytes(ctx context.Context, userID uint64, data []byte, fileName string, category models.FileCategory) (*models.FileInfo, error)

	// 文件管理
	GetFile(ctx context.Context, userID uint64, fileID uint64) (*models.FileInfo, error)
	GetFilesByUser(ctx context.Context, userID uint64, category models.FileCategory, page, limit int) ([]*models.FileInfo, int64, error)
	DeleteFile(ctx context.Context, userID uint64, fileID uint64) error
	UpdateFileInfo(ctx context.Context, userID uint64, fileID uint64, updates map[string]interface{}) error

	// 文件操作
	GetFileContent(ctx context.Context, userID uint64, fileID uint64) ([]byte, error)
	GetFileURL(ctx context.Context, userID uint64, fileID uint64) (string, error)
	GeneratePreview(ctx context.Context, fileID uint64) (string, error)

	// 批量操作
	BatchUpload(ctx context.Context, userID uint64, files []multipart.File, headers []*multipart.FileHeader, category models.FileCategory) ([]*models.FileInfo, error)
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
	if err := os.MkdirAll(service.uploadPath, 0755); err != nil {
		service.logger.Error("Failed to create upload directory", zap.Error(err))
	}

	return service
}

// UploadFile 上传文件
func (s *fileService) UploadFile(ctx context.Context, userID uint64, file multipart.File, header *multipart.FileHeader, category models.FileCategory) (*models.FileInfo, error) {
	// 验证文件大小
	if header.Size > s.maxFileSize {
		return nil, fmt.Errorf("file size exceeds limit: %d bytes", s.maxFileSize)
	}

	// 生成唯一文件名
	fileName := s.generateUniqueFileName(header.Filename)
	filePath := filepath.Join(s.uploadPath, fileName)

	// 创建目标文件
	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create file: %v", err)
	}
	defer dst.Close()

	// 复制文件内容
	_, err = io.Copy(dst, file)
	if err != nil {
		os.Remove(filePath) // 清理失败的文件
		return nil, fmt.Errorf("failed to save file: %v", err)
	}

	// 计算MD5哈希
	hash, err := s.calculateMD5(filePath)
	if err != nil {
		s.logger.Warn("Failed to calculate MD5", zap.Error(err))
	}

	// 检测文件类型
	fileType := s.detectFileType(header.Header.Get("Content-Type"))

	// 创建文件信息记录
	fileInfo := &models.FileInfo{
		UserID:       userID,
		OriginalName: header.Filename,
		FileName:     fileName,
		FilePath:     filePath,
		FileSize:     header.Size,
		ContentType:  header.Header.Get("Content-Type"),
		FileType:     fileType,
		Category:     category,
		MD5Hash:      hash,
		IsPublic:     false,
		DownloadURL:  s.generateDownloadURL(fileName),
	}

	// 保存到数据库
	if err := s.fileRepo.Create(fileInfo); err != nil {
		os.Remove(filePath) // 清理文件
		return nil, fmt.Errorf("failed to save file info: %v", err)
	}

	s.logger.Info("File uploaded successfully",
		zap.Uint64("user_id", userID),
		zap.String("file_name", fileName),
		zap.Int64("file_size", header.Size))

	return fileInfo, nil
}

// UploadFromURL 从URL上传文件
func (s *fileService) UploadFromURL(ctx context.Context, userID uint64, url string, category models.FileCategory) (*models.FileInfo, error) {
	// 实现从URL下载并保存文件的逻辑
	// 这里是一个简化的实现
	return nil, fmt.Errorf("upload from URL not implemented yet")
}

// UploadFromBytes 从字节数据上传文件
func (s *fileService) UploadFromBytes(ctx context.Context, userID uint64, data []byte, fileName string, category models.FileCategory) (*models.FileInfo, error) {
	// 验证文件大小
	if int64(len(data)) > s.maxFileSize {
		return nil, fmt.Errorf("file size exceeds limit: %d bytes", len(data))
	}

	// 生成唯一文件名
	uniqueFileName := s.generateUniqueFileName(fileName)
	filePath := filepath.Join(s.uploadPath, uniqueFileName)

	// 写入文件
	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %v", err)
	}

	// 计算MD5哈希
	hash := fmt.Sprintf("%x", md5.Sum(data))

	// 检测文件类型
	fileType := s.detectFileTypeFromExtension(fileName)

	// 创建文件信息记录
	fileInfo := &models.FileInfo{
		UserID:       userID,
		OriginalName: fileName,
		FileName:     uniqueFileName,
		FilePath:     filePath,
		FileSize:     int64(len(data)),
		FileType:     fileType,
		Category:     category,
		MD5Hash:      hash,
		IsPublic:     false,
		DownloadURL:  s.generateDownloadURL(uniqueFileName),
	}

	// 保存到数据库
	if err := s.fileRepo.Create(fileInfo); err != nil {
		os.Remove(filePath) // 清理文件
		return nil, fmt.Errorf("failed to save file info: %v", err)
	}

	return fileInfo, nil
}

// GetFile 获取文件信息
func (s *fileService) GetFile(ctx context.Context, userID uint64, fileID uint64) (*models.FileInfo, error) {
	return s.fileRepo.GetByUserIDAndFileID(userID, fileID)
}

// GetFilesByUser 获取用户的文件列表
func (s *fileService) GetFilesByUser(ctx context.Context, userID uint64, category models.FileCategory, page, limit int) ([]*models.FileInfo, int64, error) {
	offset := (page - 1) * limit
	return s.fileRepo.GetByUserIDAndCategory(userID, string(category), offset, limit)
}

// DeleteFile 删除文件
func (s *fileService) DeleteFile(ctx context.Context, userID uint64, fileID uint64) error {
	// 获取文件信息
	fileInfo, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return fmt.Errorf("file not found: %v", err)
	}

	// 删除物理文件
	if err := os.Remove(fileInfo.FilePath); err != nil {
		s.logger.Warn("Failed to delete physical file", zap.Error(err))
	}

	// 删除数据库记录
	return s.fileRepo.Delete(fileID)
}

// UpdateFileInfo 更新文件信息
func (s *fileService) UpdateFileInfo(ctx context.Context, userID uint64, fileID uint64, updates map[string]interface{}) error {
	// 验证用户权限
	if _, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID); err != nil {
		return fmt.Errorf("file not found or access denied")
	}

	return s.fileRepo.Update(fileID, updates)
}

// GetFileContent 获取文件内容
func (s *fileService) GetFileContent(ctx context.Context, userID uint64, fileID uint64) ([]byte, error) {
	fileInfo, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return nil, fmt.Errorf("file not found: %v", err)
	}

	return os.ReadFile(fileInfo.FilePath)
}

// GetFileURL 获取文件访问URL
func (s *fileService) GetFileURL(ctx context.Context, userID uint64, fileID uint64) (string, error) {
	_, err := s.fileRepo.GetByUserIDAndFileID(userID, fileID)
	if err != nil {
		return "", fmt.Errorf("file not found: %v", err)
	}

	// 增加访问计数
	_ = s.fileRepo.IncrementAccessCount(fileID)

	return fmt.Sprintf("%s/api/v1/files/%d", s.baseURL, fileID), nil
}

// GeneratePreview 生成预览
func (s *fileService) GeneratePreview(ctx context.Context, fileID uint64) (string, error) {
	// 实现预览生成逻辑（缩略图等）
	return "", fmt.Errorf("preview generation not implemented yet")
}

// BatchUpload 批量上传
func (s *fileService) BatchUpload(ctx context.Context, userID uint64, files []multipart.File, headers []*multipart.FileHeader, category models.FileCategory) ([]*models.FileInfo, error) {
	var results []*models.FileInfo

	for i, file := range files {
		if i >= len(headers) {
			break
		}

		fileInfo, err := s.UploadFile(ctx, userID, file, headers[i], category)
		if err != nil {
			s.logger.Error("Failed to upload file in batch",
				zap.String("filename", headers[i].Filename),
				zap.Error(err))
			continue
		}

		results = append(results, fileInfo)
	}

	return results, nil
}

// BatchDelete 批量删除
func (s *fileService) BatchDelete(ctx context.Context, userID uint64, fileIDs []uint64) (int, error) {
	deletedCount := 0

	for _, fileID := range fileIDs {
		if err := s.DeleteFile(ctx, userID, fileID); err != nil {
			s.logger.Error("Failed to delete file in batch",
				zap.Uint64("file_id", fileID),
				zap.Error(err))
			continue
		}
		deletedCount++
	}

	return deletedCount, nil
}

// CleanupTempFiles 清理临时文件
func (s *fileService) CleanupTempFiles(ctx context.Context) error {
	// 实现清理临时文件的逻辑
	return nil
}

// CleanupExpiredFiles 清理过期文件
func (s *fileService) CleanupExpiredFiles(ctx context.Context) error {
	expiredFiles, err := s.fileRepo.GetExpiredFiles()
	if err != nil {
		return fmt.Errorf("failed to get expired files: %v", err)
	}

	for _, file := range expiredFiles {
		// 删除物理文件
		if err := os.Remove(file.FilePath); err != nil {
			s.logger.Warn("Failed to delete expired physical file", zap.Error(err))
		}

		// 删除数据库记录
		if err := s.fileRepo.Delete(file.ID); err != nil {
			s.logger.Error("Failed to delete expired file record", zap.Error(err))
		}
	}

	s.logger.Info("Cleaned up expired files", zap.Int("count", len(expiredFiles)))
	return nil
}

// 辅助方法

// generateUniqueFileName 生成唯一文件名
func (s *fileService) generateUniqueFileName(originalName string) string {
	ext := filepath.Ext(originalName)
	name := strings.TrimSuffix(originalName, ext)
	timestamp := time.Now().UnixNano()
	return fmt.Sprintf("%s_%d%s", name, timestamp, ext)
}

// calculateMD5 计算文件MD5哈希
func (s *fileService) calculateMD5(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := md5.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}

// detectFileType 根据MIME类型检测文件类型
func (s *fileService) detectFileType(mimeType string) models.FileType {
	if strings.HasPrefix(mimeType, "image/") {
		return models.FileTypeImage
	} else if strings.HasPrefix(mimeType, "video/") {
		return models.FileTypeVideo
	} else if strings.HasPrefix(mimeType, "audio/") {
		return models.FileTypeAudio
	} else if mimeType == "application/pdf" ||
		strings.Contains(mimeType, "document") ||
		strings.Contains(mimeType, "text") {
		return models.FileTypeDocument
	}
	return models.FileTypeOther
}

// detectFileTypeFromExtension 根据文件扩展名检测文件类型
func (s *fileService) detectFileTypeFromExtension(fileName string) models.FileType {
	ext := strings.ToLower(filepath.Ext(fileName))
	switch ext {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp":
		return models.FileTypeImage
	case ".mp4", ".avi", ".mov", ".mkv", ".flv", ".wmv":
		return models.FileTypeVideo
	case ".mp3", ".wav", ".ogg", ".flac", ".aac":
		return models.FileTypeAudio
	case ".pdf", ".doc", ".docx", ".txt", ".rtf":
		return models.FileTypeDocument
	default:
		return models.FileTypeOther
	}
}

// generateDownloadURL 生成下载URL
func (s *fileService) generateDownloadURL(fileName string) string {
	return fmt.Sprintf("%s/api/v1/files/%s", s.baseURL, fileName)
}