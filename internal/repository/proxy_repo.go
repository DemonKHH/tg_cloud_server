package repository

import (
	"gorm.io/gorm"

	"tg_cloud_server/internal/models"
)

// ProxyRepository 代理仓库接口
type ProxyRepository interface {
	Create(proxy *models.Proxy) error
	GetByID(id uint64) (*models.Proxy, error)
	GetByUserID(userID uint64, page, limit int) ([]*models.ProxyIP, int64, error)
	GetByUserIDAndID(userID, proxyID uint64) (*models.Proxy, error)
	GetByUserIDAndStatus(userID uint64, status string, page, limit int) ([]*models.ProxyIP, int64, error)
	Update(proxy *models.Proxy) error
	Delete(id uint64) error

	// 代理查询
	GetAvailableProxies(userID uint64) ([]*models.Proxy, error)
	GetProxiesByStatus(userID uint64, status string) ([]*models.Proxy, error)

	// 代理统计
	GetProxyStats(userID uint64) (*models.ProxyStats, error)
	GetStatsByUserID(userID uint64) (*models.ProxyStats, error)
	UpdateProxyStatus(id uint64, status string) error

	// 批量操作
	BulkUpdateStatus(proxyIDs []uint64, status string) error
}

// proxyRepository GORM实现
type proxyRepository struct {
	db *gorm.DB
}

// NewProxyRepository 创建代理仓库
func NewProxyRepository(db *gorm.DB) ProxyRepository {
	return &proxyRepository{db: db}
}

// Create 创建代理
func (r *proxyRepository) Create(proxy *models.Proxy) error {
	return r.db.Create(proxy).Error
}

// GetByID 根据ID获取代理
func (r *proxyRepository) GetByID(id uint64) (*models.Proxy, error) {
	var proxy models.Proxy
	err := r.db.Where("id = ?", id).First(&proxy).Error
	return &proxy, err
}

// GetByUserID 根据用户ID获取代理列表（分页）
func (r *proxyRepository) GetByUserID(userID uint64, page, limit int) ([]*models.ProxyIP, int64, error) {
	var proxies []*models.ProxyIP
	var total int64

	offset := (page - 1) * limit

	// 获取总数
	if err := r.db.Model(&models.ProxyIP{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err := r.db.Where("user_id = ?", userID).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&proxies).Error

	return proxies, total, err
}

// GetByUserIDAndID 根据用户ID和代理ID获取代理
func (r *proxyRepository) GetByUserIDAndID(userID, proxyID uint64) (*models.Proxy, error) {
	var proxy models.Proxy
	err := r.db.Where("user_id = ? AND id = ?", userID, proxyID).First(&proxy).Error
	return &proxy, err
}

// Update 更新代理
func (r *proxyRepository) Update(proxy *models.Proxy) error {
	return r.db.Save(proxy).Error
}

// Delete 删除代理
func (r *proxyRepository) Delete(id uint64) error {
	return r.db.Delete(&models.Proxy{}, id).Error
}

// GetAvailableProxies 获取可用代理
func (r *proxyRepository) GetAvailableProxies(userID uint64) ([]*models.Proxy, error) {
	var proxies []*models.Proxy
	err := r.db.Where("user_id = ? AND status = ?", userID, "active").
		Order("created_at DESC").
		Find(&proxies).Error
	return proxies, err
}

// GetProxiesByStatus 根据状态获取代理
func (r *proxyRepository) GetProxiesByStatus(userID uint64, status string) ([]*models.Proxy, error) {
	var proxies []*models.Proxy
	err := r.db.Where("user_id = ? AND status = ?", userID, status).
		Order("created_at DESC").
		Find(&proxies).Error
	return proxies, err
}

// GetProxyStats 获取代理统计
func (r *proxyRepository) GetProxyStats(userID uint64) (*models.ProxyStats, error) {
	var stats models.ProxyStats

	// 总代理数
	r.db.Model(&models.Proxy{}).
		Where("user_id = ?", userID).
		Count(&stats.Total)

	// 各状态代理数
	var statusCounts []struct {
		Status string
		Count  int64
	}

	r.db.Model(&models.Proxy{}).
		Select("status, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("status").
		Find(&statusCounts)

	for _, sc := range statusCounts {
		switch sc.Status {
		case "active":
			stats.Active = sc.Count
		case "inactive":
			stats.Inactive = sc.Count
		case "error":
			stats.Error = sc.Count
		case "testing":
			stats.Testing = sc.Count
		}
	}

	return &stats, nil
}

// GetByUserIDAndStatus 根据用户ID和状态获取代理列表（分页）
func (r *proxyRepository) GetByUserIDAndStatus(userID uint64, status string, page, limit int) ([]*models.ProxyIP, int64, error) {
	var proxies []*models.ProxyIP
	var total int64

	offset := (page - 1) * limit

	// 获取总数
	if err := r.db.Model(&models.ProxyIP{}).Where("user_id = ? AND status = ?", userID, status).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取分页数据
	err := r.db.Where("user_id = ? AND status = ?", userID, status).
		Offset(offset).
		Limit(limit).
		Order("created_at DESC").
		Find(&proxies).Error

	return proxies, total, err
}

// GetStatsByUserID 根据用户ID获取代理统计
func (r *proxyRepository) GetStatsByUserID(userID uint64) (*models.ProxyStats, error) {
	var stats models.ProxyStats

	// 获取总数
	r.db.Model(&models.ProxyIP{}).Where("user_id = ?", userID).Count(&stats.Total)

	// 按状态统计
	var statusCounts []struct {
		Status string
		Count  int64
	}

	r.db.Model(&models.ProxyIP{}).
		Select("status, COUNT(*) as count").
		Where("user_id = ?", userID).
		Group("status").
		Find(&statusCounts)

	// 分配统计数据
	for _, sc := range statusCounts {
		switch sc.Status {
		case "active":
			stats.Active = sc.Count
		case "inactive":
			stats.Inactive = sc.Count
		case "error":
			stats.Error = sc.Count
		case "testing":
			stats.Testing = sc.Count
		}
	}

	return &stats, nil
}

// UpdateProxyStatus 更新代理状态
func (r *proxyRepository) UpdateProxyStatus(id uint64, status string) error {
	return r.db.Model(&models.Proxy{}).
		Where("id = ?", id).
		Update("status", status).Error
}

// BulkUpdateStatus 批量更新代理状态
func (r *proxyRepository) BulkUpdateStatus(proxyIDs []uint64, status string) error {
	return r.db.Model(&models.Proxy{}).
		Where("id IN ?", proxyIDs).
		Update("status", status).Error
}
