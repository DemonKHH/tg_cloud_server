package services

import (
	"context"
	"regexp"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

// RiskControlService 风控服务接口
type RiskControlService interface {
	// CanExecuteTask 检查账号是否可以执行任务
	CanExecuteTask(ctx context.Context, accountID uint64, taskType models.TaskType) (allowed bool, reason string)

	// ReportTaskResult 上报任务执行结果
	ReportTaskResult(ctx context.Context, accountID uint64, success bool, taskErr error)

	// HandleTelegramError 处理Telegram错误
	HandleTelegramError(ctx context.Context, accountID uint64, err error)

	// ProcessCoolingRecovery 处理冷却恢复（定时任务调用）
	ProcessCoolingRecovery(ctx context.Context) int

	// ProcessWarningRecovery 处理警告恢复（定时任务调用）
	ProcessWarningRecovery(ctx context.Context) int

	// GetUserRiskSettings 获取用户风控配置
	GetUserRiskSettings(ctx context.Context, userID uint64) *models.UserRiskSettings

	// UpdateUserRiskSettings 更新用户风控配置
	UpdateUserRiskSettings(ctx context.Context, userID uint64, settings *models.UserRiskSettings) error
}

// riskControlService 风控服务实现
type riskControlService struct {
	accountRepo repository.AccountRepository
	userRepo    repository.UserRepository
	logger      *zap.Logger
}

// NewRiskControlService 创建风控服务实例
func NewRiskControlService(
	accountRepo repository.AccountRepository,
	userRepo repository.UserRepository,
) RiskControlService {
	return &riskControlService{
		accountRepo: accountRepo,
		userRepo:    userRepo,
		logger:      logger.Get().Named("risk_control"),
	}
}

// CanExecuteTask 检查账号是否可以执行任务
func (s *riskControlService) CanExecuteTask(ctx context.Context, accountID uint64, taskType models.TaskType) (bool, string) {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		return false, "账号不存在"
	}

	switch account.Status {
	case models.AccountStatusDead:
		return false, "账号已死亡，无法执行任务"

	case models.AccountStatusFrozen:
		return false, "账号已冻结，无法执行任务"

	case models.AccountStatusCooling:
		// 检查冷却是否到期
		if account.CoolingUntil != nil && account.CoolingUntil.After(time.Now()) {
			remaining := time.Until(*account.CoolingUntil)
			return false, "账号冷却中，剩余 " + remaining.Round(time.Minute).String()
		}
		// 冷却已到期，允许执行（定时任务会恢复状态）

	case models.AccountStatusRestricted, models.AccountStatusTwoWay:
		// 允许执行，但记录警告日志
		s.logger.Warn("Executing task on restricted/two_way account",
			zap.Uint64("account_id", accountID),
			zap.String("status", string(account.Status)),
			zap.String("task_type", string(taskType)))
	}

	// new, normal, warning, restricted, two_way 都允许执行
	return true, ""
}

// ReportTaskResult 上报任务执行结果
func (s *riskControlService) ReportTaskResult(ctx context.Context, accountID uint64, success bool, taskErr error) {
	account, err := s.accountRepo.GetByID(accountID)
	if err != nil {
		s.logger.Error("Failed to get account for risk report",
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		return
	}

	if success {
		// 成功：重置连续失败计数
		if account.ConsecutiveFailures > 0 {
			if err := s.accountRepo.ResetConsecutiveFailures(accountID); err != nil {
				s.logger.Error("Failed to reset consecutive failures",
					zap.Uint64("account_id", accountID),
					zap.Error(err))
			}
		}
		return
	}

	// 失败：增加连续失败计数
	newCount, err := s.accountRepo.IncrementConsecutiveFailures(accountID)
	if err != nil {
		s.logger.Error("Failed to increment consecutive failures",
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		return
	}

	// 获取用户风控配置
	settings := s.GetUserRiskSettings(ctx, account.UserID)

	// 检查是否触发冷却
	if int(newCount) >= settings.MaxConsecutiveFailures {
		coolingUntil := time.Now().Add(time.Duration(settings.CoolingDurationMinutes) * time.Minute)

		if err := s.accountRepo.UpdateCoolingStatus(accountID, models.AccountStatusCooling, &coolingUntil, 0); err != nil {
			s.logger.Error("Failed to update cooling status",
				zap.Uint64("account_id", accountID),
				zap.Error(err))
			return
		}

		s.logger.Warn("Account triggered cooling due to consecutive failures",
			zap.Uint64("account_id", accountID),
			zap.Uint32("failures", newCount),
			zap.Time("cooling_until", coolingUntil))
	}
}

// HandleTelegramError 处理Telegram错误
func (s *riskControlService) HandleTelegramError(ctx context.Context, accountID uint64, err error) {
	if err == nil {
		return
	}

	account, getErr := s.accountRepo.GetByID(accountID)
	if getErr != nil {
		s.logger.Error("Failed to get account for telegram error handling",
			zap.Uint64("account_id", accountID),
			zap.Error(getErr))
		return
	}

	errorStr := strings.ToUpper(err.Error())
	var newStatus models.AccountStatus
	var coolingUntil *time.Time

	// 致命错误 → Dead
	if strings.Contains(errorStr, "AUTH_KEY_UNREGISTERED") ||
		strings.Contains(errorStr, "USER_DEACTIVATED") ||
		strings.Contains(errorStr, "PHONE_NUMBER_BANNED") ||
		strings.Contains(errorStr, "SESSION_REVOKED") {
		newStatus = models.AccountStatusDead

		// 限流错误 → Cooling
	} else if strings.Contains(errorStr, "FLOOD_WAIT") {
		newStatus = models.AccountStatusCooling
		waitSeconds := s.parseFloodWaitSeconds(errorStr)
		until := time.Now().Add(time.Duration(waitSeconds+60) * time.Second)
		coolingUntil = &until

	} else if strings.Contains(errorStr, "PEER_FLOOD") {
		newStatus = models.AccountStatusCooling
		until := time.Now().Add(1 * time.Hour)
		coolingUntil = &until

	} else if strings.Contains(errorStr, "PHONE_NUMBER_FLOOD") {
		newStatus = models.AccountStatusCooling
		until := time.Now().Add(24 * time.Hour)
		coolingUntil = &until

	} else if strings.Contains(errorStr, "SLOWMODE_WAIT") {
		newStatus = models.AccountStatusCooling
		until := time.Now().Add(30 * time.Minute)
		coolingUntil = &until

		// 限制错误 → Restricted
	} else if strings.Contains(errorStr, "USER_RESTRICTED") ||
		strings.Contains(errorStr, "CHAT_WRITE_FORBIDDEN") ||
		strings.Contains(errorStr, "CHAT_RESTRICTED") {
		newStatus = models.AccountStatusRestricted

	} else {
		// 其他错误不处理状态变更
		return
	}

	// 更新状态
	oldStatus := account.Status

	if err := s.accountRepo.UpdateCoolingStatus(accountID, newStatus, coolingUntil, 0); err != nil {
		s.logger.Error("Failed to update account status on telegram error",
			zap.Uint64("account_id", accountID),
			zap.Error(err))
		return
	}

	s.logger.Warn("Account status changed due to Telegram error",
		zap.Uint64("account_id", accountID),
		zap.String("old_status", string(oldStatus)),
		zap.String("new_status", string(newStatus)),
		zap.String("error", err.Error()))
}

// parseFloodWaitSeconds 解析 FLOOD_WAIT 错误中的等待秒数
func (s *riskControlService) parseFloodWaitSeconds(errorStr string) int {
	// 匹配 FLOOD_WAIT_123 或 FLOOD_WAIT (123) 等格式
	re := regexp.MustCompile(`FLOOD_WAIT[_\s]*(\d+)`)
	matches := re.FindStringSubmatch(errorStr)
	if len(matches) >= 2 {
		if seconds, err := strconv.Atoi(matches[1]); err == nil {
			return seconds
		}
	}
	// 默认返回 300 秒（5分钟）
	return 300
}

// ProcessCoolingRecovery 处理冷却恢复
func (s *riskControlService) ProcessCoolingRecovery(ctx context.Context) int {
	accounts, err := s.accountRepo.GetCoolingExpiredAccounts()
	if err != nil {
		s.logger.Error("Failed to get cooling expired accounts", zap.Error(err))
		return 0
	}

	recoveredCount := 0
	for _, account := range accounts {
		if err := s.accountRepo.UpdateCoolingStatus(account.ID, models.AccountStatusNormal, nil, 0); err != nil {
			s.logger.Error("Failed to recover account from cooling",
				zap.Uint64("account_id", account.ID),
				zap.Error(err))
			continue
		}

		recoveredCount++
		s.logger.Info("Account recovered from cooling",
			zap.Uint64("account_id", account.ID),
			zap.String("phone", account.Phone))
	}

	if recoveredCount > 0 {
		s.logger.Info("Cooling recovery completed",
			zap.Int("recovered_count", recoveredCount))
	}

	return recoveredCount
}

// ProcessWarningRecovery 处理警告恢复
func (s *riskControlService) ProcessWarningRecovery(ctx context.Context) int {
	cutoffTime := time.Now().Add(-24 * time.Hour)
	accounts, err := s.accountRepo.GetWarningAccountsOlderThan(cutoffTime)
	if err != nil {
		s.logger.Error("Failed to get warning accounts", zap.Error(err))
		return 0
	}

	recoveredCount := 0
	for _, account := range accounts {
		if err := s.accountRepo.UpdateStatus(account.ID, models.AccountStatusNormal); err != nil {
			s.logger.Error("Failed to recover account from warning",
				zap.Uint64("account_id", account.ID),
				zap.Error(err))
			continue
		}

		recoveredCount++
		s.logger.Info("Account recovered from warning",
			zap.Uint64("account_id", account.ID),
			zap.String("phone", account.Phone))
	}

	if recoveredCount > 0 {
		s.logger.Info("Warning recovery completed",
			zap.Int("recovered_count", recoveredCount))
	}

	return recoveredCount
}

// GetUserRiskSettings 获取用户风控配置
func (s *riskControlService) GetUserRiskSettings(ctx context.Context, userID uint64) *models.UserRiskSettings {
	defaults := models.GetDefaultRiskSettings()

	user, err := s.userRepo.GetByID(userID)
	if err != nil || user.RiskSettings == nil {
		return defaults
	}

	settings := user.RiskSettings
	settings.Validate() // 确保值在有效范围内

	return settings
}

// UpdateUserRiskSettings 更新用户风控配置
func (s *riskControlService) UpdateUserRiskSettings(ctx context.Context, userID uint64, settings *models.UserRiskSettings) error {
	settings.Validate() // 确保值在有效范围内

	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return err
	}

	user.RiskSettings = settings
	return s.userRepo.Update(user)
}
