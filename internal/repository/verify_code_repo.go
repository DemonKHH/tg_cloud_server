package repository

import (
	"time"

	"gorm.io/gorm"

	"tg_cloud_server/internal/models"
)

// VerifyCodeRepository 验证码会话仓库接口
type VerifyCodeRepository interface {
	Create(session *models.VerifyCodeSession) error
	GetByCode(code string) (*models.VerifyCodeSession, error)
	ListByUserID(userID uint64, page, limit int, keyword string) ([]models.VerifyCodeSession, int64, error)
	DeleteByCode(code string) error
	DeleteByCodes(codes []string) error
	DeleteExpired() error
	DeleteByUserIDAndCodes(userID uint64, codes []string) error
}

// verifyCodeRepository 验证码会话仓库实现
type verifyCodeRepository struct {
	db *gorm.DB
}

// NewVerifyCodeRepository 创建验证码会话仓库
func NewVerifyCodeRepository(db *gorm.DB) VerifyCodeRepository {
	return &verifyCodeRepository{db: db}
}

// Create 创建验证码会话
func (r *verifyCodeRepository) Create(session *models.VerifyCodeSession) error {
	return r.db.Create(session).Error
}

// GetByCode 根据code获取会话
func (r *verifyCodeRepository) GetByCode(code string) (*models.VerifyCodeSession, error) {
	var session models.VerifyCodeSession
	err := r.db.Where("code = ? AND expires_at > ?", code, time.Now()).First(&session).Error
	if err != nil {
		return nil, err
	}
	return &session, nil
}

// ListByUserID 获取用户的验证码会话列表（支持分页和搜索）
func (r *verifyCodeRepository) ListByUserID(userID uint64, page, limit int, keyword string) ([]models.VerifyCodeSession, int64, error) {
	var sessions []models.VerifyCodeSession
	var total int64

	query := r.db.Model(&models.VerifyCodeSession{}).Where("user_id = ? AND expires_at > ?", userID, time.Now())

	// 关键词搜索（搜索code）
	if keyword != "" {
		query = query.Where("code LIKE ?", "%"+keyword+"%")
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&sessions).Error; err != nil {
		return nil, 0, err
	}

	return sessions, total, nil
}

// DeleteByCode 删除指定code的会话
func (r *verifyCodeRepository) DeleteByCode(code string) error {
	return r.db.Where("code = ?", code).Delete(&models.VerifyCodeSession{}).Error
}

// DeleteByCodes 批量删除指定codes的会话
func (r *verifyCodeRepository) DeleteByCodes(codes []string) error {
	if len(codes) == 0 {
		return nil
	}
	return r.db.Where("code IN ?", codes).Delete(&models.VerifyCodeSession{}).Error
}

// DeleteExpired 删除过期的会话
func (r *verifyCodeRepository) DeleteExpired() error {
	return r.db.Where("expires_at <= ?", time.Now()).Delete(&models.VerifyCodeSession{}).Error
}

// DeleteByUserIDAndCodes 删除指定用户的指定codes的会话
func (r *verifyCodeRepository) DeleteByUserIDAndCodes(userID uint64, codes []string) error {
	if len(codes) == 0 {
		return nil
	}
	return r.db.Where("user_id = ? AND code IN ?", userID, codes).Delete(&models.VerifyCodeSession{}).Error
}
