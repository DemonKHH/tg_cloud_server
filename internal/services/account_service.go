package services

import (
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

var (
	ErrAccountExists   = errors.New("account already exists")
	ErrAccountNotFound = errors.New("account not found")
	ErrProxyNotFound   = errors.New("proxy not found")
)

// AccountService 账号管理服务
type AccountService struct {
	accountRepo repository.AccountRepository
	proxyRepo   repository.ProxyRepository
	logger      *zap.Logger
}

// NewAccountService 创建账号管理服务
func NewAccountService(accountRepo repository.AccountRepository, proxyRepo repository.ProxyRepository) *AccountService {
	return &AccountService{
		accountRepo: accountRepo,
		proxyRepo:   proxyRepo,
		logger:      logger.Get().Named("account_service"),
	}
}

// AccountFilter 账号过滤器
type AccountFilter struct {
	UserID uint64
	Status string
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
		UserID:      userID,
		Phone:       req.Phone,
		Status:      models.AccountStatusNew,
		HealthScore: 1.0,
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
	return s.accountRepo.GetAccountSummaries(filter.UserID, filter.Page, filter.Limit)
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

// CheckAccountHealth 检查账号健康度
func (s *AccountService) CheckAccountHealth(userID, accountID uint64) (*models.AccountHealthReport, error) {
	account, err := s.accountRepo.GetByUserIDAndID(userID, accountID)
	if err != nil {
		return nil, ErrAccountNotFound
	}

	// 创建健康度报告
	now := time.Now()
	report := &models.AccountHealthReport{
		AccountID:   account.ID,
		Phone:       account.Phone,
		Status:      account.Status,
		HealthScore: account.HealthScore,
		CheckedAt:   &now,
		Issues:      []string{},
		Suggestions: []string{},
	}

	// 检查各种健康指标
	s.checkAccountStatus(account, report)
	s.checkProxyStatus(account, report)
	s.checkUsagePattern(account, report)

	// 更新最后检查时间
	now = time.Now()
	account.LastCheckAt = &now
	s.accountRepo.Update(account)

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
		HealthScore:      account.HealthScore,
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
		AccountID:   account.ID,
		IsValid:     true,
		Warnings:    []string{},
		Errors:      []string{},
		QueueSize:   0, // 需要从任务调度器获取
		HealthScore: account.HealthScore,
	}

	// 检查账号状态
	if account.Status == models.AccountStatusDead {
		result.IsValid = false
		result.Errors = append(result.Errors, "账号已死亡，无法执行任务")
	} else if account.Status == models.AccountStatusCooling {
		result.IsValid = false
		result.Errors = append(result.Errors, "账号处于冷却期，暂时无法执行任务")
	} else if account.Status == models.AccountStatusMaintenance {
		result.IsValid = false
		result.Errors = append(result.Errors, "账号处于维护状态，暂时无法执行任务")
	}

	// 检查健康度
	if account.HealthScore < 0.3 {
		result.Warnings = append(result.Warnings, "账号健康度较低，建议谨慎使用")
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

	if account.HealthScore < 0.5 {
		report.Issues = append(report.Issues, "账号健康度较低")
		report.Suggestions = append(report.Suggestions, "减少使用频率，让账号休息一段时间")
	}
}

// generateAvailabilityRecommendations 生成可用性建议
func (s *AccountService) generateAvailabilityRecommendations(account *models.TGAccount, availability *models.AccountAvailability) {
	if account.HealthScore < 0.3 {
		availability.Errors = append(availability.Errors, "账号健康度过低")
		availability.Recommendation = "建议暂停使用此账号"
	} else if account.HealthScore < 0.7 {
		availability.Warnings = append(availability.Warnings, "账号健康度偏低")
		availability.Recommendation = "适当减少使用频率"
	} else {
		availability.Recommendation = "账号状态良好，可正常使用"
	}
}

// validateTaskSpecificRequirements 验证特定任务类型的要求
func (s *AccountService) validateTaskSpecificRequirements(account *models.TGAccount, taskType models.TaskType, result *models.ValidationResult) {
	switch taskType {
	case models.TaskTypePrivate:
		if account.HealthScore < 0.5 {
			result.Warnings = append(result.Warnings, "私信任务对账号健康度要求较高")
		}
	case models.TaskTypeBroadcast:
		if account.HealthScore < 0.6 {
			result.Warnings = append(result.Warnings, "群发任务风险较高，建议使用健康度更高的账号")
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
		HealthScore:  100.0,
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
		report.HealthScore = 0
		report.Issues = append(report.Issues, "账号已死亡")
		report.Suggestions = append(report.Suggestions, "更换新账号")
	case models.AccountStatusRestricted:
		report.HealthScore -= 40
		report.Issues = append(report.Issues, "账号受限")
		report.Suggestions = append(report.Suggestions, "等待限制解除")
	case models.AccountStatusCooling:
		report.HealthScore -= 20
		report.Issues = append(report.Issues, "账号冷却中")
		report.Suggestions = append(report.Suggestions, "暂停使用")
	}

	// 简化连接状态检查 - 实际实现需要根据模型定义调整
	// switch account.ConnectionStatus {
	// case models.StatusConnectionError:
	//	report.HealthScore -= 30
	//	report.Issues = append(report.Issues, "连接异常")
	//	report.Suggestions = append(report.Suggestions, "检查网络和代理设置")
	// case models.StatusDisconnected:
	//	report.HealthScore -= 15
	//	report.Issues = append(report.Issues, "未连接")
	//	report.Suggestions = append(report.Suggestions, "重新建立连接")
	// }

	// 确保健康度在0-100范围内
	if report.HealthScore > 100 {
		report.HealthScore = 100
	} else if report.HealthScore < 0 {
		report.HealthScore = 0
	}

	report.Score = report.HealthScore
	return report
}
