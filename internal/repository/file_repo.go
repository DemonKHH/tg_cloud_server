package repository

import (
	"gorm.io/gorm"

	"tg_cloud_server/internal/services"
)

// FileRepository 文件仓库接口
type FileRepository interface {
	Create(fileInfo *services.FileInfo) error
	GetByID(id uint64) (*services.FileInfo, error)
	GetByUserIDAndFileID(userID, fileID uint64) (*services.FileInfo, error)
	GetByUserIDAndCategory(userID uint64, category string, offset, limit int) ([]*services.FileInfo, int64, error)
	Update(fileID uint64, updates map[string]interface{}) error
	Delete(fileID uint64) error
	IncrementAccessCount(fileID uint64) error
	GetExpiredFiles() ([]*services.FileInfo, error)
}

// fileRepository GORM实现
type fileRepository struct {
	db *gorm.DB
}

// NewFileRepository 创建文件仓库
func NewFileRepository(db *gorm.DB) FileRepository {
	return &fileRepository{db: db}
}

// Create 创建文件信息
func (r *fileRepository) Create(fileInfo *services.FileInfo) error {
	return r.db.Create(fileInfo).Error
}

// GetByID 根据ID获取文件信息
func (r *fileRepository) GetByID(id uint64) (*services.FileInfo, error) {
	var fileInfo services.FileInfo
	err := r.db.Where("id = ?", id).First(&fileInfo).Error
	return &fileInfo, err
}

// GetByUserIDAndFileID 根据用户ID和文件ID获取文件信息
func (r *fileRepository) GetByUserIDAndFileID(userID, fileID uint64) (*services.FileInfo, error) {
	var fileInfo services.FileInfo
	err := r.db.Where("user_id = ? AND id = ?", userID, fileID).First(&fileInfo).Error
	return &fileInfo, err
}

// GetByUserIDAndCategory 根据用户ID和分类获取文件列表
func (r *fileRepository) GetByUserIDAndCategory(userID uint64, category string, offset, limit int) ([]*services.FileInfo, int64, error) {
	var files []*services.FileInfo
	var total int64

	query := r.db.Where("user_id = ?", userID)
	if category != "" {
		query = query.Where("category = ?", category)
	}

	// 获取总数
	if err := query.Model(&services.FileInfo{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	err := query.Offset(offset).Limit(limit).
		Order("created_at DESC").
		Find(&files).Error

	return files, total, err
}

// Update 更新文件信息
func (r *fileRepository) Update(fileID uint64, updates map[string]interface{}) error {
	return r.db.Model(&services.FileInfo{}).Where("id = ?", fileID).Updates(updates).Error
}

// Delete 删除文件信息
func (r *fileRepository) Delete(fileID uint64) error {
	return r.db.Delete(&services.FileInfo{}, fileID).Error
}

// IncrementAccessCount 增加访问次数
func (r *fileRepository) IncrementAccessCount(fileID uint64) error {
	return r.db.Model(&services.FileInfo{}).Where("id = ?", fileID).
		Update("access_count", gorm.Expr("access_count + 1")).Error
}

// GetExpiredFiles 获取过期文件
func (r *fileRepository) GetExpiredFiles() ([]*services.FileInfo, error) {
	var files []*services.FileInfo
	err := r.db.Where("expires_at IS NOT NULL AND expires_at < NOW()").Find(&files).Error
	return files, err
}
