package models

import (
	"time"

	"gorm.io/gorm"
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
	ID           uint64       `json:"id" gorm:"primaryKey;autoIncrement"`
	UserID       uint64       `json:"user_id" gorm:"not null;index"`
	OriginalName string       `json:"original_name" gorm:"size:255;not null"`
	FileName     string       `json:"file_name" gorm:"size:255;not null;uniqueIndex"`
	FilePath     string       `json:"file_path" gorm:"size:500;not null"`
	FileSize     int64        `json:"file_size" gorm:"not null"`
	ContentType  string       `json:"content_type" gorm:"size:100"`
	FileType     FileType     `json:"file_type" gorm:"type:enum('image','video','audio','document','other');not null"`
	Category     FileCategory `json:"category" gorm:"type:enum('avatar','message','template','attachment','export','import');not null"`
	MD5Hash      string       `json:"md5_hash" gorm:"size:32;index"`
	IsPublic     bool         `json:"is_public" gorm:"default:false"`
	DownloadURL  string       `json:"download_url" gorm:"size:500"`
	PreviewURL   string       `json:"preview_url" gorm:"size:500"`
	Metadata     string       `json:"metadata" gorm:"type:json"` // 存储额外的文件元数据
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`

	// 关联关系
	User User `json:"user" gorm:"foreignKey:UserID"`
}

// TableName 指定表名
func (FileInfo) TableName() string {
	return "file_infos"
}

// BeforeCreate 创建前钩子
func (f *FileInfo) BeforeCreate(tx *gorm.DB) error {
	// 可以在这里添加创建前的逻辑
	return nil
}

// GetURL 获取文件访问URL
func (f *FileInfo) GetURL() string {
	if f.IsPublic && f.DownloadURL != "" {
		return f.DownloadURL
	}
	// 返回需要认证的URL
	return "/api/v1/files/" + f.FileName
}

// GetPreviewURL 获取预览URL
func (f *FileInfo) GetPreviewURL() string {
	if f.PreviewURL != "" {
		return f.PreviewURL
	}
	// 对于图片类型，返回缩略图URL
	if f.FileType == FileTypeImage {
		return "/api/v1/files/" + f.FileName + "/preview"
	}
	return ""
}

// IsImage 检查是否为图片文件
func (f *FileInfo) IsImage() bool {
	return f.FileType == FileTypeImage
}

// IsVideo 检查是否为视频文件
func (f *FileInfo) IsVideo() bool {
	return f.FileType == FileTypeVideo
}

// FileUploadRequest 文件上传请求
type FileUploadRequest struct {
	Category    FileCategory `json:"category" binding:"required"`
	IsPublic    bool         `json:"is_public"`
	Description string       `json:"description"`
}

// FileUploadResponse 文件上传响应
type FileUploadResponse struct {
	FileID      uint64 `json:"file_id"`
	FileName    string `json:"file_name"`
	DownloadURL string `json:"download_url"`
	PreviewURL  string `json:"preview_url,omitempty"`
}

// FileListRequest 文件列表请求
type FileListRequest struct {
	Category FileCategory `json:"category"`
	FileType FileType     `json:"file_type"`
	IsPublic *bool        `json:"is_public"`
	Page     int          `json:"page"`
	Limit    int          `json:"limit"`
}

// BatchFileOperation 批量文件操作
type BatchFileOperation struct {
	FileIDs   []uint64 `json:"file_ids" binding:"required"`
	Operation string   `json:"operation" binding:"required,oneof=delete move copy"`
	TargetDir string   `json:"target_dir,omitempty"`
}
