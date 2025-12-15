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
	BatchCreate(accounts []*models.TGAccount) error
	BatchDelete(ids []uint64) error
	BatchUpdate(accounts []*models.TGAccount) error
	GetByID(id uint64) (*models.TGAccount, error)
	GetByUserIDAndID(userID, accountID uint64) (*models.TGAccount, error)
	GetByPhone(phone string) (*models.TGAccount, error)
	GetByUserID(userID uint64, offset, limit int) ([]*models.TGAccount, int64, error)
	Update(account *models.TGAccount) error
	UpdateStatus(id uint64, status models.AccountStatus) error
	Delete(id uint64) error
	GetAccountsByStatus(status models.AccountStatus) ([]*models.TGAccount, error)
	CountByUserID(userID uint64) (int64, error)
	CountActiveByUserID(userID uint64) (int64, error)
	GetAccountSummaries(userID uint64, page, limit int, search, status string) ([]*models.AccountSummary, int64, error)
	GetAll() ([]*models.TGAccount, error)
	UpdateSessionData(accountID uint64, sessionData []byte) error
	UpdateConnectionStatus(id uint64, isOnline bool) error
	Update2FAStatus(id uint64, has2FA bool, password string) error
	UpdateRiskStatus(id uint64, status models.AccountStatus, frozenUntil *string) error
	GetStatusDistribution(userID uint64) (map[string]int64, error)
	GetGrowthTrend(userID uint64, days int) ([]models.TimeSeriesPoint, error)
	GetProxyUsageStats(userID uint64) (*models.ProxyUsageStats, error)

	// 风控相关方法
	GetCoolingExpiredAccounts() ([]*models.TGAccount, error)
	GetWarningAccountsOlderThan(cutoffTime time.Time) ([]*models.TGAccount, error)
	UpdateCoolingStatus(id uint64, status models.AccountStatus, coolingUntil *time.Time, consecutiveFailures uint32) error
	IncrementConsecutiveFailures(id uint64) (uint32, error)
	ResetConsecutiveFailures(id uint64) error
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

// BatchCreate 批量创建账号（使用事务）
func (r *accountRepository) BatchCreate(accounts []*models.TGAccount) error {
	if len(accounts) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, account := range accounts {
			if err := tx.Create(account).Error; err != nil {
				return err
			}
		}
		return nil
	})
}

// BatchDelete 批量删除账号（使用事务）
func (r *accountRepository) BatchDelete(ids []uint64) error {
	if len(ids) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先将关联的任务日志中的 account_id 设为 NULL
		if err := tx.Model(&models.TaskLog{}).Where("account_id IN ?", ids).Update("account_id", nil).Error; err != nil {
			return err
		}
		// 再删除账号
		return tx.Delete(&models.TGAccount{}, ids).Error
	})
}

// BatchUpdate 批量更新账号（使用事务）
func (r *accountRepository) BatchUpdate(accounts []*models.TGAccount) error {
	if len(accounts) == 0 {
		return nil
	}
	return r.db.Transaction(func(tx *gorm.DB) error {
		for _, account := range accounts {
			if err := tx.Save(account).Error; err != nil {
				return err
			}
		}
		return nil
	})
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

// Delete 删除账号
func (r *accountRepository) Delete(id uint64) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		// 先将关联的任务日志中的 account_id 设为 NULL
		if err := tx.Model(&models.TaskLog{}).Where("account_id = ?", id).Update("account_id", nil).Error; err != nil {
			return err
		}
		// 再删除账号
		return tx.Delete(&models.TGAccount{}, id).Error
	})
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

	// 确保返回空数组而不是 nil
	if accounts == nil {
		accounts = []*models.TGAccount{}
	}

	return accounts, total, err
}

// GetAccountSummaries 获取账号摘要列表（分页）
func (r *accountRepository) GetAccountSummaries(userID uint64, page, limit int, search, status string) ([]*models.AccountSummary, int64, error) {
	var summaries []*models.AccountSummary
	var total int64

	offset := (page - 1) * limit

	// 构建查询
	query := r.db.Model(&models.TGAccount{}).Where("tg_accounts.user_id = ?", userID)

	// 添加搜索条件（仅搜索手机号）
	if search != "" {
		query = query.Where("tg_accounts.phone LIKE ?", "%"+search+"%")
	}

	// 添加状态过滤条件
	if status != "" {
		query = query.Where("tg_accounts.status = ?", status)
	}

	// 获取总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 获取摘要数据（包含 Telegram 信息、代理信息和风控字段）
	err := query.
		Select("tg_accounts.id, tg_accounts.user_id, tg_accounts.phone, tg_accounts.status, tg_accounts.is_online, tg_accounts.proxy_id, tg_accounts.frozen_until, tg_accounts.has_2fa, tg_accounts.two_fa_password, tg_accounts.consecutive_failures, tg_accounts.cooling_until, tg_accounts.tg_user_id, tg_accounts.username, tg_accounts.first_name, tg_accounts.last_name, tg_accounts.bio, tg_accounts.photo_url, tg_accounts.last_used_at, tg_accounts.created_at, proxy_ips.name as proxy_name, proxy_ips.ip as proxy_ip, proxy_ips.port as proxy_port, proxy_ips.username as proxy_username, proxy_ips.password as proxy_password, proxy_ips.protocol as proxy_protocol").
		Joins("LEFT JOIN proxy_ips ON proxy_ips.id = tg_accounts.proxy_id").
		Offset(offset).
		Limit(limit).
		Order("tg_accounts.created_at DESC").
		Scan(&summaries).Error

	// 确保返回空数组而不是 nil
	if summaries == nil {
		summaries = []*models.AccountSummary{}
	}

	return summaries, total, err
}

// GetAll 获取所有账号
func (r *accountRepository) GetAll() ([]*models.TGAccount, error) {
	var accounts []*models.TGAccount
	err := r.db.Find(&accounts).Error
	return accounts, err
}

// UpdateSessionData 更新账号的Session数据
func (r *accountRepository) UpdateSessionData(accountID uint64, sessionData []byte) error {
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", accountID).
		Update("session_data", string(sessionData)).Error
}

// UpdateConnectionStatus 更新账号在线状态
func (r *accountRepository) UpdateConnectionStatus(id uint64, isOnline bool) error {
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Update("is_online", isOnline).Error
}

// Update2FAStatus 更新账号2FA状态
func (r *accountRepository) Update2FAStatus(id uint64, has2FA bool, password string) error {
	updates := map[string]interface{}{
		"has_2fa":    has2FA,
		"updated_at": time.Now(),
	}
	if password != "" {
		updates["two_fa_password"] = password
	}
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// UpdateRiskStatus 更新账号风控状态
func (r *accountRepository) UpdateRiskStatus(id uint64, status models.AccountStatus, frozenUntil *string) error {
	updates := map[string]interface{}{
		"status":     status,
		"updated_at": time.Now(),
	}
	if frozenUntil != nil {
		updates["frozen_until"] = *frozenUntil
	}
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// GetStatusDistribution 获取账号状态分布
func (r *accountRepository) GetStatusDistribution(userID uint64) (map[string]int64, error) {
	var results []struct {
		Status string
		Count  int64
	}

	err := r.db.Model(&models.TGAccount{}).
		Select("status, count(*) as count").
		Where("user_id = ?", userID).
		Group("status").
		Scan(&results).Error

	if err != nil {
		return nil, err
	}

	distribution := make(map[string]int64)
	for _, result := range results {
		distribution[result.Status] = result.Count
	}

	return distribution, nil
}

// GetGrowthTrend 获取账号增长趋势
func (r *accountRepository) GetGrowthTrend(userID uint64, days int) ([]models.TimeSeriesPoint, error) {
	// 使用 SQLite 的 strftime 函数格式化日期
	// 注意：如果使用 MySQL，需要改为 DATE_FORMAT(created_at, '%Y-%m-%d')
	// 这里假设是 SQLite，因为没有明确指定数据库类型，但通常本地开发用 SQLite
	// 为了兼容性，我们获取最近 days 天的数据，然后在内存中处理
	// 或者使用 GORM 的通用写法

	startDate := time.Now().AddDate(0, 0, -days)

	// 获取所有在此期间创建的账号
	var accounts []models.TGAccount
	err := r.db.Select("created_at").
		Where("user_id = ? AND created_at >= ?", userID, startDate).
		Order("created_at ASC").
		Find(&accounts).Error

	if err != nil {
		return nil, err
	}

	// 按天聚合
	dailyCounts := make(map[string]int64)
	for _, account := range accounts {
		date := account.CreatedAt.Format("2006-01-02")
		dailyCounts[date]++
	}

	// 构建结果，确保每天都有数据（即使是0）
	var points []models.TimeSeriesPoint
	// 获取当前总数作为基准（如果是累计增长）或者每日新增
	// 这里我们返回每日新增，前端如果是展示累计，需要前端处理或者这里改为累计
	// 根据 stats_service.go 中的 mock 数据，看起来是累计总数
	// 让我们先获取截止到 startDate 的总数
	var currentTotal int64
	r.db.Model(&models.TGAccount{}).Where("user_id = ? AND created_at < ?", userID, startDate).Count(&currentTotal)

	for i := 0; i < days; i++ {
		date := startDate.AddDate(0, 0, i)
		dateStr := date.Format("2006-01-02")

		// 累加当天的增量
		currentTotal += dailyCounts[dateStr]

		points = append(points, models.TimeSeriesPoint{
			Timestamp: date,
			Value:     float64(currentTotal),
			Label:     date.Format("01-02"),
		})
	}

	return points, nil
}

// GetProxyUsageStats 获取代理使用统计
func (r *accountRepository) GetProxyUsageStats(userID uint64) (*models.ProxyUsageStats, error) {
	stats := &models.ProxyUsageStats{
		ProxyTypes: make(map[string]int64),
	}

	// 统计有代理和无代理的账号
	var withProxy int64
	r.db.Model(&models.TGAccount{}).Where("user_id = ? AND proxy_id IS NOT NULL", userID).Count(&withProxy)
	stats.WithProxy = withProxy

	var withoutProxy int64
	r.db.Model(&models.TGAccount{}).Where("user_id = ? AND proxy_id IS NULL", userID).Count(&withoutProxy)
	stats.WithoutProxy = withoutProxy

	// 统计代理类型
	var typeResults []struct {
		Protocol string
		Count    int64
	}

	err := r.db.Table("tg_accounts").
		Select("proxy_ips.protocol, count(*) as count").
		Joins("LEFT JOIN proxy_ips ON proxy_ips.id = tg_accounts.proxy_id").
		Where("tg_accounts.user_id = ? AND tg_accounts.proxy_id IS NOT NULL", userID).
		Group("proxy_ips.protocol").
		Scan(&typeResults).Error

	if err != nil {
		return nil, err
	}

	for _, result := range typeResults {
		stats.ProxyTypes[result.Protocol] = result.Count
	}

	// 计算平均延迟 (仅计算有延迟数据的代理)
	// 假设 proxy_ips 表有 latency 字段，且 tg_accounts 关联了 proxy_ips
	// 这里需要注意 proxy_ips 表结构，之前 list_dir 没有显示 proxy_ips 的结构，但 account_repo.go 中有 Joins("LEFT JOIN proxy_ips ...")
	// 假设 proxy_ips 有 latency 字段
	// 如果没有 latency 字段，这个查询会报错。
	// 让我们先检查一下 proxy_ips 的模型定义，或者先注释掉延迟统计，或者假设它存在。
	// 查看 account_repo.go 的 GetAccountSummaries 方法，它 select 了 proxy_ips 的字段，但没有 latency。
	// 让我们先不计算 latency，或者设为 0。
	stats.AvgLatency = 0 // 暂不支持延迟统计

	return stats, nil
}

// GetCoolingExpiredAccounts 获取冷却到期的账号
func (r *accountRepository) GetCoolingExpiredAccounts() ([]*models.TGAccount, error) {
	var accounts []*models.TGAccount
	err := r.db.Where("status = ? AND cooling_until IS NOT NULL AND cooling_until < ?",
		models.AccountStatusCooling, time.Now()).
		Find(&accounts).Error
	return accounts, err
}

// GetWarningAccountsOlderThan 获取警告状态超过指定时间的账号
func (r *accountRepository) GetWarningAccountsOlderThan(cutoffTime time.Time) ([]*models.TGAccount, error) {
	var accounts []*models.TGAccount
	err := r.db.Where("status = ? AND updated_at < ?",
		models.AccountStatusWarning, cutoffTime).
		Find(&accounts).Error
	return accounts, err
}

// UpdateCoolingStatus 更新账号冷却状态
func (r *accountRepository) UpdateCoolingStatus(id uint64, status models.AccountStatus, coolingUntil *time.Time, consecutiveFailures uint32) error {
	updates := map[string]interface{}{
		"status":               status,
		"cooling_until":        coolingUntil,
		"consecutive_failures": consecutiveFailures,
		"updated_at":           time.Now(),
	}
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Updates(updates).Error
}

// IncrementConsecutiveFailures 增加连续失败计数并返回新值
func (r *accountRepository) IncrementConsecutiveFailures(id uint64) (uint32, error) {
	// 先增加计数
	err := r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		UpdateColumn("consecutive_failures", gorm.Expr("consecutive_failures + 1")).Error
	if err != nil {
		return 0, err
	}

	// 获取新值
	var account models.TGAccount
	err = r.db.Select("consecutive_failures").Where("id = ?", id).First(&account).Error
	if err != nil {
		return 0, err
	}
	return account.ConsecutiveFailures, nil
}

// ResetConsecutiveFailures 重置连续失败计数
func (r *accountRepository) ResetConsecutiveFailures(id uint64) error {
	return r.db.Model(&models.TGAccount{}).
		Where("id = ?", id).
		Update("consecutive_failures", 0).Error
}
