package repository

import (
	"errors"
	"time"

	"gorm.io/gorm"

	"tg_cloud_server/internal/models"
)

// AccountRepository 账号数据访问接口
type AccountRepository interface {
	Create(account *models.TGAccount) error
	GetByID(id uint64) (*models.TGAccount, error)
	GetByUserIDAndID(userID, accountID uint64) (*models.TGAccount, error)
	GetByPhone(phone string) (*models.TGAccount, error)
	GetByUserID(userID uint64, offset, limit int) ([]*models.TGAccount, int64, error)
	Update(account *models.TGAccount) error
	UpdateStatus(id uint64, status models.AccountStatus) error
	UpdateHealthScore(id uint64, score float64) error
	Delete(id uint64) error
	GetAccountsByStatus(status models.AccountStatus) ([]*models.TGAccount, error)
	CountByUserID(userID uint64) (int64, error)
	CountActiveByUserID(userID uint64) (int64, error)
	GetAccountsNeedingHealthCheck() ([]*models.TGAccount, error)
	GetAccountSummaries(userID uint64, page, limit int) ([]*models.AccountSummary, int64, error)
	GetAll() ([]*models.TGAccount, error)
}

// accountRepository 账号数据访问实现
type accountRepository struct {
	db *gorm.DB
}

// NewAccountRepository 创建账号数据访问实例
func NewAccountRepository(db *gorm.DB) AccountRepository {
	return &accountRepository{db: db}
}

// Create 创建账号
func (r *accountRepository) Create(account *models.TGAccount) error {
	return r.db.Create(account).Error
}

// GetByID 根据ID获取账号
func (r *accountRepository) GetByID(id uint64) (*models.TGAccount, error) {
	var account models.TGAccount
	err := r.db.Preload("User").Preload("ProxyIP").Where("id = ?", id).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("account not found")
		}
		return nil, err
	}
	return &account, nil
}

// GetByUserIDAndID 根据用户ID和账号ID获取账号
func (r *accountRepository) GetByUserIDAndID(userID, accountID uint64) (*models.TGAccount, error) {
	var account models.TGAccount
	err := r.db.Preload("User").Preload("ProxyIP").
		Where("id = ? AND user_id = ?", accountID, userID).
		First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("account not found")
		}
		return nil, err
	}
	return &account, nil
}

// GetByPhone 根据手机号获取账号
func (r *accountRepository) GetByPhone(phone string) (*models.TGAccount, error) {
	var account models.TGAccount
	err := r.db.Where("phone = ?", phone).First(&account).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("account not found")
		}
		return nil, err
	}
	return &account, nil
}

// GetByUserID 根据用户ID获取账号列表
func (r *accountRepository) GetByUserID(userID uint64, offset, limit int) ([]*models.TGAccount, int64, error) {
	var accounts []*models.TGAccount
	var total int64

	// 获取总数
	if err := r.db.Model(&models.TGAccount{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err := r.db.Preload("ProxyIP").
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&accounts).Error
	if err != nil {
		return nil, 0, err
	}

	return accounts, total, nil
}

// Update 更新账号
func (r *accountRepository) Update(account *models.TGAccount) error {
	return r.db.Save(account).Error
}

// UpdateStatus 更新账号状态
func (r *accountRepository) UpdateStatus(id uint64, status models.AccountStatus) error {
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":     status,
			"updated_at": time.Now(),
		}).Error
}

// UpdateHealthScore 更新健康度分数
func (r *accountRepository) UpdateHealthScore(id uint64, score float64) error {
	now := time.Now()
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"health_score":  score,
			"last_check_at": &now,
			"updated_at":    now,
		}).Error
}

// Delete 删除账号
func (r *accountRepository) Delete(id uint64) error {
	return r.db.Delete(&models.TGAccount{}, id).Error
}

// GetAccountsByStatus 根据状态获取账号列表
func (r *accountRepository) GetAccountsByStatus(status models.AccountStatus) ([]*models.TGAccount, error) {
	var accounts []*models.TGAccount
	err := r.db.Preload("User").Preload("ProxyIP").
		Where("status = ?", status).
		Find(&accounts).Error
	return accounts, err
}

// CountByUserID 统计用户账号总数
func (r *accountRepository) CountByUserID(userID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&models.TGAccount{}).Where("user_id = ?", userID).Count(&count).Error
	return count, err
}

// CountActiveByUserID 统计用户活跃账号数
func (r *accountRepository) CountActiveByUserID(userID uint64) (int64, error) {
	var count int64
	err := r.db.Model(&models.TGAccount{}).
		Where("user_id = ? AND status IN (?)", userID, []models.AccountStatus{
			models.AccountStatusNormal,
			models.AccountStatusWarning,
		}).
		Count(&count).Error
	return count, err
}

// GetAccountsNeedingHealthCheck 获取需要健康检查的账号
func (r *accountRepository) GetAccountsNeedingHealthCheck() ([]*models.TGAccount, error) {
	var accounts []*models.TGAccount

	// 获取超过5分钟未检查或从未检查的账号
	fiveMinutesAgo := time.Now().Add(-5 * time.Minute)

	err := r.db.Preload("User").Preload("ProxyIP").
		Where("(last_check_at IS NULL OR last_check_at < ?) AND status NOT IN (?)",
			fiveMinutesAgo,
			[]models.AccountStatus{
				models.AccountStatusDead,
				models.AccountStatusMaintenance,
				}).
		Limit(50). // 限制每次检查的数量
		Find(&accounts).Error

	return accounts, err
}

// UpdateLastUsed 更新最后使用时间
func (r *accountRepository) UpdateLastUsed(id uint64) error {
	now := time.Now()
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Update("last_used_at", &now).Error
}

// GetAccountsWithFilters 根据多个条件过滤账号
func (r *accountRepository) GetAccountsWithFilters(filters map[string]interface{}, offset, limit int) ([]*models.TGAccount, int64, error) {
	query := r.db.Model(&models.TGAccount{}).Preload("User").Preload("ProxyIP")

	// 应用过滤条件
	for key, value := range filters {
		switch key {
		case "user_id":
			query = query.Where("user_id = ?", value)
		case "status":
			query = query.Where("status = ?", value)
		case "health_score_min":
			query = query.Where("health_score >= ?", value)
		case "health_score_max":
			query = query.Where("health_score <= ?", value)
		case "has_proxy":
			if value.(bool) {
				query = query.Where("proxy_id IS NOT NULL")
			} else {
				query = query.Where("proxy_id IS NULL")
			}
		}
	}

	// 获取总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	var accounts []*models.TGAccount
	err := query.Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&accounts).Error

	return accounts, total, err
}

// GetAccountSummaries 获取账号摘要列表（分页）
func (r *accountRepository) GetAccountSummaries(userID uint64, page, limit int) ([]*models.AccountSummary, int64, error) {
	var summaries []*models.AccountSummary
	var total int64

	offset := (page - 1) * limit

	// 获取总数
	if err := r.db.Model(&models.TGAccount{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取摘要数据
	err := r.db.Model(&models.TGAccount{}).
		Select("id, user_id, phone, status, health_score, last_active_at, created_at").
		Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Scan(&summaries).Error

	return summaries, total, err
}

// GetAll 获取所有账号
func (r *accountRepository) GetAll() ([]*models.TGAccount, error) {
	var accounts []*models.TGAccount
	err := r.db.Find(&accounts).Error
	return accounts, err
}
