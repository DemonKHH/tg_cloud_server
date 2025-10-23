package repository

import (
	"gorm.io/gorm"

	"tg_cloud_server/internal/services"
)

// TemplateRepository 模板仓库接口
type TemplateRepository interface {
	Create(template *services.MessageTemplate) error
	GetByID(id uint64) (*services.MessageTemplate, error)
	GetByUserIDAndID(userID, templateID uint64) (*services.MessageTemplate, error)
	GetByUserIDWithFilter(userID uint64, filter *services.TemplateFilter, offset, limit int) ([]*services.MessageTemplate, int64, error)
	Update(template *services.MessageTemplate) error
	Delete(templateID uint64) error
	IncrementUsageCount(templateID uint64) error
	GetPopularByType(userID uint64, templateType string, limit int) ([]*services.MessageTemplate, error)
}

// templateRepository GORM实现
type templateRepository struct {
	db *gorm.DB
}

// NewTemplateRepository 创建模板仓库
func NewTemplateRepository(db *gorm.DB) TemplateRepository {
	return &templateRepository{db: db}
}

// Create 创建模板
func (r *templateRepository) Create(template *services.MessageTemplate) error {
	return r.db.Create(template).Error
}

// GetByID 根据ID获取模板
func (r *templateRepository) GetByID(id uint64) (*services.MessageTemplate, error) {
	var template services.MessageTemplate
	err := r.db.Where("id = ?", id).First(&template).Error
	return &template, err
}

// GetByUserIDAndID 根据用户ID和模板ID获取模板
func (r *templateRepository) GetByUserIDAndID(userID, templateID uint64) (*services.MessageTemplate, error) {
	var template services.MessageTemplate
	err := r.db.Where("user_id = ? AND id = ?", userID, templateID).First(&template).Error
	return &template, err
}

// GetByUserIDWithFilter 根据用户ID和过滤条件获取模板列表
func (r *templateRepository) GetByUserIDWithFilter(userID uint64, filter *services.TemplateFilter, offset, limit int) ([]*services.MessageTemplate, int64, error) {
	var templates []*services.MessageTemplate
	var total int64

	query := r.db.Where("user_id = ?", userID)

	// 应用过滤条件
	if filter.Type != "" {
		query = query.Where("type = ?", filter.Type)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.Category != "" {
		query = query.Where("category = ?", filter.Category)
	}
	if filter.Keyword != "" {
		query = query.Where("name LIKE ? OR description LIKE ?", "%"+filter.Keyword+"%", "%"+filter.Keyword+"%")
	}

	// 获取总数
	if err := query.Model(&services.MessageTemplate{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取数据
	err := query.Offset(offset).Limit(limit).
		Order("updated_at DESC").
		Find(&templates).Error

	return templates, total, err
}

// Update 更新模板
func (r *templateRepository) Update(template *services.MessageTemplate) error {
	return r.db.Save(template).Error
}

// Delete 删除模板
func (r *templateRepository) Delete(templateID uint64) error {
	return r.db.Delete(&services.MessageTemplate{}, templateID).Error
}

// IncrementUsageCount 增加使用次数
func (r *templateRepository) IncrementUsageCount(templateID uint64) error {
	return r.db.Model(&services.MessageTemplate{}).Where("id = ?", templateID).
		Update("usage_count", gorm.Expr("usage_count + 1")).Error
}

// GetPopularByType 获取热门模板
func (r *templateRepository) GetPopularByType(userID uint64, templateType string, limit int) ([]*services.MessageTemplate, error) {
	var templates []*services.MessageTemplate

	query := r.db.Where("user_id = ? AND status = ?", userID, "active")
	if templateType != "" {
		query = query.Where("type = ?", templateType)
	}

	err := query.Order("usage_count DESC").Limit(limit).Find(&templates).Error
	return templates, err
}
