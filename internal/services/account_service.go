package services

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
	"tg_cloud_server/internal/telegram"
)

var (
	ErrAccountExists   = errors.New("account already exists")
	ErrAccountNotFound = errors.New("account not found")
	ErrProxyNotFound   = errors.New("proxy not found")
)

// AccountService 账号管理服务
type AccountService struct {
	accountRepo    repository.AccountRepository
	proxyRepo      repository.ProxyRepository
	connectionPool *telegram.ConnectionPool
	logger         *zap.Logger
}

// NewAccountService 创建账号管理服务
func NewAccountService(accountRepo repository.AccountRepository, proxyRepo repository.ProxyRepository, connectionPool *telegram.ConnectionPool) *AccountService {
	return &AccountService{
		accountRepo:    accountRepo,
		proxyRepo:      proxyRepo,
		connectionPool: connectionPool,
		logger:         logger.Get().Named("account_service"),
	}
}

// AccountFilter 账号过滤器
type AccountFilter struct {
	UserID uint64
	Status string
	Search string
	Page   int
	Limit  int
}

// CreateAccount 创建账号
func (s *AccountService) CreateAccount(userID uint64, req *models.CreateAccountRequest) (*models.TGAccount, error) {
	// 检查手机号是否已存在
	existingAccount, _ := s.accountRepo.GetByPhone(req.Phone)
	if existingAccount != nil {
		return nil, ErrAccountExists
	}

	account := &models.TGAccount{
		UserID: userID,
		Phone:  req.Phone,
		Status: models.AccountStatusNew,
	}

	// 如果提供了session数据，设置它
	if req.SessionData != "" {
		account.SessionData = req.SessionData
	}

	// 如果指定了代理，验证代理是否存在且属于该用户
	if req.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByUserIDAndID(userID, *req.ProxyID)
		if err != nil {
			return nil, ErrProxyNotFound
		}
		if !proxy.IsActive {
			return nil, errors.New("proxy is not active")
		}
		account.ProxyID = req.ProxyID
	}

	if err := s.accountRepo.Create(account); err != nil {
		s.logger.Error("Failed to create account",
			zap.Uint64("user_id", userID),
			zap.String("phone", req.Phone),
			zap.Error(err))
		return nil, fmt.Errorf("failed to create account: %w", err)
	}

	s.logger.Info("Account created successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", account.ID),
		zap.String("phone", account.Phone))

	return account, nil
}

// GetAccounts 获取账号列表
func (s *AccountService) GetAccounts(filter *AccountFilter) ([]*models.AccountSummary, int64, error) {
	return s.accountRepo.GetAccountSummaries(filter.UserID, filter.Page, filter.Limit, filter.Search)
}

// GetAccount 获取账号详情
func (s *AccountService) GetAccount(userID, accountID uint64) (*models.TGAccount, error) {
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}
	return account, nil
}

// UpdateAccount 更新账号
func (s *AccountService) UpdateAccount(userID, accountID uint64, req *models.UpdateAccountRequest) (*models.TGAccount, error) {
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	// 更新代理绑定
	if req.ProxyID != nil {
		if *req.ProxyID == 0 {
			// 解除代理绑定
			account.ProxyID = nil
		} else {
			// 验证代理是否存在且属于该用户
			proxy, err := s.proxyRepo.GetByUserIDAndID(userID, *req.ProxyID)
			if err != nil {
				return nil, ErrProxyNotFound
			}
			if !proxy.IsActive {
				return nil, errors.New("proxy is not active")
			}
			account.ProxyID = req.ProxyID
		}
	}

	// 更新状态
	if req.Status != nil {
		account.Status = *req.Status
	}

	if err := s.accountRepo.Update(account); err != nil {
		s.logger.Error("Failed to update account",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to update account: %w", err)
	}

	s.logger.Info("Account updated successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID))

	return account, nil
}

// DeleteAccount 删除账号
func (s *AccountService) DeleteAccount(userID, accountID uint64) error {
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return ErrAccountNotFound
	}

	if err := s.accountRepo.Delete(account.ID); err != nil {
		s.logger.Error("Failed to delete account",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		return fmt.Errorf("failed to delete account: %w", err)
	}

	s.logger.Info("Account deleted successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID),
		zap.String("phone", account.Phone))

	return nil
}

// CheckAccountHealth 检查账号健康状态
func (s *AccountService) CheckAccountHealth(userID, accountID uint64) (*models.AccountHealthReport, error) {
	s.logger.Info("Starting account health check",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID))

	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		s.logger.Warn("Account not found for health check",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		return nil, ErrAccountNotFound
	}

	s.logger.Debug("Account found, starting health checks",
		zap.Uint64("account_id", accountID),
		zap.String("phone", account.Phone),
		zap.String("current_status", string(account.Status)))

	// 创建健康报告
	now := time.Now()
	report := &models.AccountHealthReport{
		AccountID:   account.ID,
		Phone:       account.Phone,
		Status:      account.Status,
		CheckedAt:   &now,
		Issues:      []string{},
		Suggestions: []string{},
	}

	// 检查各种状态指标
	s.checkAccountStatus(account, report)
	s.checkProxyStatus(account, report)
	s.checkUsagePattern(account, report)

	// 主动检查连接状态
	if s.connectionPool != nil {
		s.logger.Debug("Checking connection status",
			zap.Uint64("account_id", accountID))
		if err := s.connectionPool.CheckConnection(account.ID); err != nil {
			s.logger.Warn("Connection check failed",
				zap.Uint64("account_id", accountID),
				zap.String("phone", account.Phone),
				zap.Error(err))
			report.Issues = append(report.Issues, fmt.Sprintf("连接检查失败: %v", err))
			report.Suggestions = append(report.Suggestions, "请检查代理设置或账号Session是否有效")
			// 更新状态为异常
			if account.Status == models.AccountStatusNormal {
				account.Status = models.AccountStatusWarning
			}
		} else {
			s.logger.Info("Connection check passed",
				zap.Uint64("account_id", accountID),
				zap.String("phone", account.Phone))
		}
	}

	// 更新最后检查时间
	now = time.Now()
	account.LastCheckAt = &now
	s.accountRepo.Update(account)

	s.logger.Info("Account health check completed",
		zap.Uint64("account_id", accountID),
		zap.String("phone", account.Phone),
		zap.String("status", string(account.Status)),
		zap.Int("issues_count", len(report.Issues)),
		zap.Int("suggestions_count", len(report.Suggestions)))

	return report, nil
}

// GetAccountAvailability 获取账号可用性
func (s *AccountService) GetAccountAvailability(userID, accountID uint64) (*models.AccountAvailability, error) {
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	availability := &models.AccountAvailability{
		AccountID:        account.ID,
		Status:           account.Status,
		QueueSize:        0,                          // 需要从任务调度器获取
		IsTaskRunning:    false,                      // 需要从连接池获取
		ConnectionStatus: models.ConnectionStatus(0), // 需要从连接池获取
		LastUsed:         account.LastUsedAt,
		Warnings:         []string{},
		Errors:           []string{},
	}

	// 生成建议和警告
	s.generateAvailabilityRecommendations(account, availability)

	return availability, nil
}

// ValidateAccountForTask 验证账号是否可用于特定任务
func (s *AccountService) ValidateAccountForTask(userID, accountID uint64, taskType models.TaskType) (*models.ValidationResult, error) {
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	result := &models.ValidationResult{
		AccountID: account.ID,
		IsValid:   true,
		Warnings:  []string{},
		Errors:    []string{},
		QueueSize: 0, // 需要从任务调度器获取
	}

	// 检查账号状态
	switch account.Status {
	case models.AccountStatusDead:
		result.IsValid = false
		result.Errors = append(result.Errors, "账号已死亡，无法执行任务")
	case models.AccountStatusCooling:
		result.IsValid = false
		result.Errors = append(result.Errors, "账号处于冷却期，暂时无法执行任务")
	case models.AccountStatusMaintenance:
		result.IsValid = false
		result.Errors = append(result.Errors, "账号处于维护状态，暂时无法执行任务")
	case models.AccountStatusRestricted:
		result.Warnings = append(result.Warnings, "账号受限，可能影响任务执行")
	case models.AccountStatusWarning:
		result.Warnings = append(result.Warnings, "账号状态异常，建议谨慎使用")
	}

	// 检查特定任务类型的要求
	s.validateTaskSpecificRequirements(account, taskType, result)

	return result, nil
}

// BindProxy 绑定代理到账号
func (s *AccountService) BindProxy(userID, accountID uint64, proxyID *uint64) (*models.TGAccount, error) {
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	if proxyID == nil {
		// 解除代理绑定
		account.ProxyID = nil
	} else {
		// 验证代理是否存在且属于该用户
		proxy, err := s.proxyRepo.GetByUserIDAndID(userID, *proxyID)
		if err != nil {
			return nil, ErrProxyNotFound
		}
		if !proxy.IsActive {
			return nil, errors.New("proxy is not active")
		}
		account.ProxyID = proxyID
	}

	if err := s.accountRepo.Update(account); err != nil {
		s.logger.Error("Failed to bind proxy",
			zap.Uint64("user_id", userID),
			zap.Uint64("account_id", accountID),
			zap.Any("proxy_id", proxyID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to bind proxy: %w", err)
	}

	s.logger.Info("Proxy bound successfully",
		zap.Uint64("user_id", userID),
		zap.Uint64("account_id", accountID),
		zap.Any("proxy_id", proxyID))

	return account, nil
}

// 辅助方法

// checkAccountStatus 检查账号状态
func (s *AccountService) checkAccountStatus(account *models.TGAccount, report *models.AccountHealthReport) {
	switch account.Status {
	case models.AccountStatusDead:
		report.Issues = append(report.Issues, "账号已死亡")
		report.Suggestions = append(report.Suggestions, "请更换新的账号")
	case models.AccountStatusRestricted:
		report.Issues = append(report.Issues, "账号受到限制")
		report.Suggestions = append(report.Suggestions, "暂停使用并等待限制解除")
	case models.AccountStatusWarning:
		report.Issues = append(report.Issues, "账号状态异常")
		report.Suggestions = append(report.Suggestions, "减少使用频率，观察状态变化")
	case models.AccountStatusCooling:
		report.Issues = append(report.Issues, "账号处于冷却期")
		report.Suggestions = append(report.Suggestions, "等待冷却期结束后再使用")
	}
}

// checkProxyStatus 检查代理状态
func (s *AccountService) checkProxyStatus(account *models.TGAccount, report *models.AccountHealthReport) {
	if account.ProxyID != nil {
		proxy, err := s.proxyRepo.GetByID(*account.ProxyID)
		if err != nil {
			report.Issues = append(report.Issues, "代理配置错误")
			report.Suggestions = append(report.Suggestions, "检查代理配置或重新绑定代理")
		} else if !proxy.IsActive {
			report.Issues = append(report.Issues, "代理已禁用")
			report.Suggestions = append(report.Suggestions, "启用代理或更换其他代理")
		} else if proxy.SuccessRate < 80.0 {
			report.Issues = append(report.Issues, "代理成功率较低")
			report.Suggestions = append(report.Suggestions, "考虑更换质量更好的代理")
		}
	}
}

// checkUsagePattern 检查使用模式
func (s *AccountService) checkUsagePattern(account *models.TGAccount, report *models.AccountHealthReport) {
	if account.LastUsedAt != nil {
		timeSinceLastUse := time.Since(*account.LastUsedAt)
		if timeSinceLastUse > 24*time.Hour {
			report.Suggestions = append(report.Suggestions, "账号长时间未使用，建议定期使用保持活跃")
		}
	}
}

// generateAvailabilityRecommendations 生成可用性建议
func (s *AccountService) generateAvailabilityRecommendations(account *models.TGAccount, availability *models.AccountAvailability) {
	switch account.Status {
	case models.AccountStatusDead:
		availability.Errors = append(availability.Errors, "账号已死亡")
		availability.Recommendation = "请更换新账号"
	case models.AccountStatusRestricted:
		availability.Warnings = append(availability.Warnings, "账号受限")
		availability.Recommendation = "暂停使用，等待限制解除"
	case models.AccountStatusWarning:
		availability.Warnings = append(availability.Warnings, "账号状态异常")
		availability.Recommendation = "减少使用频率"
	case models.AccountStatusCooling:
		availability.Warnings = append(availability.Warnings, "账号冷却中")
		availability.Recommendation = "等待冷却期结束"
	case models.AccountStatusMaintenance:
		availability.Warnings = append(availability.Warnings, "账号维护中")
		availability.Recommendation = "暂时无法使用"
	default:
		availability.Recommendation = "账号状态正常，可正常使用"
	}
}

// validateTaskSpecificRequirements 验证特定任务类型的要求
func (s *AccountService) validateTaskSpecificRequirements(account *models.TGAccount, taskType models.TaskType, result *models.ValidationResult) {
	switch taskType {
	case models.TaskTypePrivate:
		if account.Status == models.AccountStatusWarning || account.Status == models.AccountStatusRestricted {
			result.Warnings = append(result.Warnings, "私信任务对账号状态要求较高")
		}
	case models.TaskTypeBroadcast:
		if account.Status == models.AccountStatusWarning || account.Status == models.AccountStatusRestricted {
			result.Warnings = append(result.Warnings, "群发任务风险较高，建议使用状态正常的账号")
		}
	case models.TaskTypeGroupChat:
		if account.Status == models.AccountStatusWarning {
			result.Warnings = append(result.Warnings, "AI炒群可能会增加账号风险")
		}
	}
}

// BatchHealthCheck 批量健康检查
func (s *AccountService) BatchHealthCheck(userID uint64, accountIDs []uint64) (map[uint64]*models.AccountHealthReport, error) {
	s.logger.Info("Starting batch health check",
		zap.Uint64("user_id", userID),
		zap.Int("account_count", len(accountIDs)))

	reports := make(map[uint64]*models.AccountHealthReport)

	for _, accountID := range accountIDs {
		// 获取账号信息
		account, err := s.accountRepo.GetByID(accountID)
		if err != nil {
			s.logger.Error("Failed to get account",
				zap.Uint64("account_id", accountID),
				zap.Error(err))
			continue
		}

		// 检查账号所有权
		if account.UserID != userID {
			s.logger.Warn("Account access denied",
				zap.Uint64("account_id", accountID),
				zap.Uint64("user_id", userID))
			continue
		}

		// 生成健康报告
		report := s.generateDetailedHealthReport(account)
		reports[accountID] = report

		// 简化更新逻辑 - 实际实现需要根据repository接口调整
		// updates := map[string]interface{}{
		//	"health_score": report.HealthScore,
		//	"last_check_at": time.Now(),
		// }
		// err = s.accountRepo.Update(accountID, updates)
		// if err != nil {
		//	s.logger.Error("Failed to update account health",
		//		zap.Uint64("account_id", accountID),
		//		zap.Error(err))
		// }
	}

	s.logger.Info("Batch health check completed",
		zap.Int("total_accounts", len(accountIDs)),
		zap.Int("checked_accounts", len(reports)))

	return reports, nil
}

// generateDetailedHealthReport 生成详细的健康报告
func (s *AccountService) generateDetailedHealthReport(account *models.TGAccount) *models.AccountHealthReport {
	now := time.Now()
	report := &models.AccountHealthReport{
		AccountID:    account.ID,
		Phone:        account.Phone,
		Status:       account.Status,
		LastCheckAt:  &now,
		CheckedAt:    &now,
		Issues:       []string{},
		Suggestions:  []string{},
		CheckResults: make(map[string]interface{}),
		GeneratedAt:  now,
	}

	// 基本状态检查
	switch account.Status {
	case models.AccountStatusDead:
		report.Issues = append(report.Issues, "账号已死亡")
		report.Suggestions = append(report.Suggestions, "更换新账号")
	case models.AccountStatusRestricted:
		report.Issues = append(report.Issues, "账号受限")
		report.Suggestions = append(report.Suggestions, "等待限制解除")
	case models.AccountStatusCooling:
		report.Issues = append(report.Issues, "账号冷却中")
		report.Suggestions = append(report.Suggestions, "暂停使用")
	case models.AccountStatusWarning:
		report.Issues = append(report.Issues, "账号状态异常")
		report.Suggestions = append(report.Suggestions, "减少使用频率")
	case models.AccountStatusMaintenance:
		report.Issues = append(report.Issues, "账号维护中")
		report.Suggestions = append(report.Suggestions, "等待维护完成")
	}

	return report
}

// CreateAccountsFromUploadData 从上传的数据批量创建账号（使用事务）
func (s *AccountService) CreateAccountsFromUploadData(userID uint64, accounts []models.AccountUploadItem, proxyID *uint64) ([]*models.TGAccount, []string, error) {
	s.logger.Info("Starting batch account creation from upload",
		zap.Uint64("user_id", userID),
		zap.Int("total_accounts", len(accounts)),
		zap.Any("proxy_id", proxyID))

	var accountsToCreate []*models.TGAccount
	var validationErrors []string

	// 如果指定了代理，先验证代理是否存在且属于该用户
	if proxyID != nil {
		proxy, err := s.proxyRepo.GetByUserIDAndID(userID, *proxyID)
		if err != nil {
			s.logger.Warn("Proxy not found for batch upload",
				zap.Uint64("user_id", userID),
				zap.Uint64("proxy_id", *proxyID),
				zap.Error(err))
			return nil, nil, fmt.Errorf("代理不存在")
		}
		if !proxy.IsActive {
			s.logger.Warn("Proxy is not active for batch upload",
				zap.Uint64("user_id", userID),
				zap.Uint64("proxy_id", *proxyID))
			return nil, nil, fmt.Errorf("代理未激活")
		}
		s.logger.Debug("Proxy validated for batch upload",
			zap.Uint64("proxy_id", *proxyID),
			zap.String("proxy_ip", proxy.IP))
	}

	// 第一阶段：验证所有数据
	for _, item := range accounts {
		// 验证必需字段
		if item.Phone == "" {
			validationErrors = append(validationErrors, "手机号不能为空")
			continue
		}
		if item.SessionData == "" {
			validationErrors = append(validationErrors, fmt.Sprintf("账号 %s: session数据不能为空", item.Phone))
			continue
		}

		// 检查账号是否已存在
		existingAccount, _ := s.accountRepo.GetByPhone(item.Phone)
		if existingAccount != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("账号 %s 已存在", item.Phone))
			continue
		}

		account := &models.TGAccount{
			UserID:      userID,
			Phone:       item.Phone,
			SessionData: item.SessionData,
			Status:      models.AccountStatusNew,
			ProxyID:     proxyID,
		}
		accountsToCreate = append(accountsToCreate, account)
	}

	// 如果没有有效账号，直接返回
	if len(accountsToCreate) == 0 {
		return nil, validationErrors, nil
	}

	// 第二阶段：使用事务批量创建
	if err := s.accountRepo.BatchCreate(accountsToCreate); err != nil {
		s.logger.Error("Failed to batch create accounts",
			zap.Uint64("user_id", userID),
			zap.Int("count", len(accountsToCreate)),
			zap.Error(err))
		return nil, validationErrors, fmt.Errorf("批量创建账号失败: %w", err)
	}

	s.logger.Info("Accounts batch created from upload",
		zap.Uint64("user_id", userID),
		zap.Int("created_count", len(accountsToCreate)),
		zap.Int("error_count", len(validationErrors)))

	return accountsToCreate, validationErrors, nil
}

// BatchSet2FA 批量设置2FA密码（使用事务）
func (s *AccountService) BatchSet2FA(userID uint64, req *models.BatchSet2FARequest) error {
	// 先获取所有需要更新的账号
	var accountsToUpdate []*models.TGAccount
	for _, accountID := range req.AccountIDs {
		account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
		if err != nil {
			s.logger.Warn("Account not found for 2FA update",
				zap.Uint64("account_id", accountID),
				zap.Error(err))
			continue
		}

		account.TwoFAPassword = req.Password
		account.Has2FA = true
		account.Is2FACorrect = false // 需要等待实际验证
		accountsToUpdate = append(accountsToUpdate, account)
	}

	if len(accountsToUpdate) == 0 {
		return nil
	}

	// 使用事务批量更新
	if err := s.accountRepo.BatchUpdate(accountsToUpdate); err != nil {
		s.logger.Error("Failed to batch update 2FA passwords",
			zap.Uint64("user_id", userID),
			zap.Int("count", len(accountsToUpdate)),
			zap.Error(err))
		return fmt.Errorf("批量设置2FA失败: %w", err)
	}

	s.logger.Info("Batch 2FA passwords set successfully",
		zap.Uint64("user_id", userID),
		zap.Int("updated_count", len(accountsToUpdate)))

	return nil
}

// BatchUpdate2FA 批量修改2FA密码（使用事务）
func (s *AccountService) BatchUpdate2FA(userID uint64, req *models.BatchUpdate2FARequest) (map[uint64]string, error) {
	results := make(map[uint64]string)
	var accountsToUpdate []*models.TGAccount

	for _, accountID := range req.AccountIDs {
		account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
		if err != nil {
			results[accountID] = "账号不存在"
			continue
		}

		// 确定旧密码
		oldPassword := req.OldPassword
		if oldPassword == "" {
			oldPassword = account.TwoFAPassword
		}

		// TODO: 实现真正的 Telegram 密码修改逻辑
		// task := telegram.NewUpdatePasswordTask(oldPassword, req.NewPassword)
		// err := s.connectionPool.ExecuteTask(fmt.Sprintf("%d", accountID), task)

		// 临时逻辑：只更新本地记录
		account.TwoFAPassword = req.NewPassword
		account.Has2FA = true
		account.Is2FACorrect = false // 修改后需要重新验证
		accountsToUpdate = append(accountsToUpdate, account)
		results[accountID] = "success"

		// 忽略 oldPassword 的 lint 警告
		_ = oldPassword
	}

	if len(accountsToUpdate) == 0 {
		return results, nil
	}

	// 使用事务批量更新
	if err := s.accountRepo.BatchUpdate(accountsToUpdate); err != nil {
		s.logger.Error("Failed to batch update 2FA passwords",
			zap.Uint64("user_id", userID),
			zap.Int("count", len(accountsToUpdate)),
			zap.Error(err))
		// 更新结果为失败
		for _, account := range accountsToUpdate {
			results[account.ID] = fmt.Sprintf("更新失败: %v", err)
		}
		return results, fmt.Errorf("批量修改2FA失败: %w", err)
	}

	s.logger.Info("Batch 2FA passwords updated successfully",
		zap.Uint64("user_id", userID),
		zap.Int("updated_count", len(accountsToUpdate)))

	return results, nil
}

// BatchDeleteAccounts 批量删除账号
func (s *AccountService) BatchDeleteAccounts(userID uint64, accountIDs []uint64) (successCount int, failedCount int, err error) {
	s.logger.Info("Starting batch delete accounts",
		zap.Uint64("user_id", userID),
		zap.Int("account_count", len(accountIDs)))

	for _, accountID := range accountIDs {
		// 验证账号属于当前用户
		_, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
		if err != nil {
			s.logger.Warn("Account not found or not owned by user",
				zap.Uint64("user_id", userID),
				zap.Uint64("account_id", accountID),
				zap.Error(err))
			failedCount++
			continue
		}

		// 删除账号
		if err := s.accountRepo.Delete(accountID); err != nil {
			s.logger.Error("Failed to delete account",
				zap.Uint64("user_id", userID),
				zap.Uint64("account_id", accountID),
				zap.Error(err))
			failedCount++
			continue
		}

		successCount++
	}

	s.logger.Info("Batch delete accounts completed",
		zap.Uint64("user_id", userID),
		zap.Int("success_count", successCount),
		zap.Int("failed_count", failedCount))

	return successCount, failedCount, nil
}
