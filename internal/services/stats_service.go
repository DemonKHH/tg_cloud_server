package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

// StatsService 统计服务接口
type StatsService interface {
	// 系统统计
	GetSystemOverview(ctx context.Context, userID uint64, period string) (*models.SystemOverview, error)
	GetAccountStatistics(ctx context.Context, userID uint64, period string, status string) (*models.AccountStatistics, error)
	GetUserDashboard(ctx context.Context, userID uint64) (*models.UserDashboard, error)

	// 实时统计
	GetRealTimeStats(ctx context.Context, userID uint64) (map[string]interface{}, error)
	GetSystemHealth(ctx context.Context) (*models.SystemHealth, error)
}

// statsService 统计服务实现
type statsService struct {
	userRepo    repository.UserRepository
	accountRepo repository.AccountRepository
	taskRepo    repository.TaskRepository
	proxyRepo   repository.ProxyRepository
	logger      *zap.Logger
}

// NewStatsService 创建统计服务
func NewStatsService(
	userRepo repository.UserRepository,
	accountRepo repository.AccountRepository,
	taskRepo repository.TaskRepository,
	proxyRepo repository.ProxyRepository,
) StatsService {
	return &statsService{
		userRepo:    userRepo,
		accountRepo: accountRepo,
		taskRepo:    taskRepo,
		proxyRepo:   proxyRepo,
		logger:      logger.Get().Named("stats_service"),
	}
}

// GetSystemOverview 获取系统统计概览
func (s *statsService) GetSystemOverview(ctx context.Context, userID uint64, period string) (*models.SystemOverview, error) {
	s.logger.Info("Getting system overview",
		zap.Uint64("user_id", userID),
		zap.String("period", period))

	now := time.Now()
	periodStart := s.getPeriodStart(now, period)

	// 获取基本统计
	totalUsers, _ := s.getUserCount(ctx)
	totalAccounts, _ := s.accountRepo.CountByUserID(userID)

	// 获取任务统计
	taskStats, err := s.taskRepo.GetTaskStatsByUserID(userID, time.Time{}, time.Time{})
	if err != nil {
		s.logger.Error("Failed to get task stats", zap.Error(err))
	}
	totalTasks := taskStats.Total

	totalProxies, _ := s.getProxyCount(ctx, userID)

	// 获取周期内统计
	periodAccounts, _ := s.getAccountCountSince(ctx, userID, periodStart)

	periodTaskStats, _ := s.taskRepo.GetTaskStatsByUserID(userID, periodStart, time.Time{})
	periodTasks := periodTaskStats.Total

	// 获取状态分布
	accountsByStatus, err := s.accountRepo.GetStatusDistribution(userID)
	if err != nil {
		s.logger.Error("Failed to get account status distribution", zap.Error(err))
		accountsByStatus = make(map[string]int64)
	}

	tasksByStatus, err := s.taskRepo.GetStatusDistribution(userID, periodStart)
	if err != nil {
		s.logger.Error("Failed to get task status distribution", zap.Error(err))
		tasksByStatus = make(map[string]int64)
	}

	tasksByType, err := s.taskRepo.GetTypeDistribution(userID, periodStart)
	if err != nil {
		s.logger.Error("Failed to get task type distribution", zap.Error(err))
		tasksByType = make(map[string]int64)
	}

	// 获取系统健康指标
	systemHealth, _ := s.GetSystemHealth(ctx)

	overview := &models.SystemOverview{
		TotalUsers:       totalUsers,
		TotalAccounts:    totalAccounts,
		TotalTasks:       totalTasks,
		TotalProxies:     totalProxies,
		PeriodUsers:      0, // 简化实现
		PeriodAccounts:   periodAccounts,
		PeriodTasks:      periodTasks,
		AccountsByStatus: accountsByStatus,
		TasksByStatus:    tasksByStatus,
		TasksByType:      tasksByType,
		SystemHealth:     *systemHealth,
		GeneratedAt:      now,
		Period:           period,
	}

	return overview, nil
}

// GetAccountStatistics 获取账号统计详情
func (s *statsService) GetAccountStatistics(ctx context.Context, userID uint64, period string, status string) (*models.AccountStatistics, error) {
	s.logger.Info("Getting account statistics",
		zap.Uint64("user_id", userID),
		zap.String("period", period),
		zap.String("status", status))

	now := time.Now()

	// 获取基本统计
	totalAccounts, _ := s.accountRepo.CountByUserID(userID)
	activeAccounts, _ := s.accountRepo.CountActiveByUserID(userID)

	// 获取状态分布
	statusDistribution, err := s.accountRepo.GetStatusDistribution(userID)
	if err != nil {
		s.logger.Error("Failed to get status distribution", zap.Error(err))
		statusDistribution = make(map[string]int64)
	}

	// 获取代理使用情况
	proxyUsage, err := s.accountRepo.GetProxyUsageStats(userID)
	if err != nil {
		s.logger.Error("Failed to get proxy usage stats", zap.Error(err))
		proxyUsage = &models.ProxyUsageStats{}
	}

	// 获取活跃度统计 (简化实现，暂无 repository 支持)
	activityStats := s.getAccountActivityStats(ctx, userID)

	// 获取风控统计 (基于状态分布计算)
	riskStats := s.calculateRiskStats(statusDistribution)

	// 获取趋势数据
	trendData, err := s.accountRepo.GetGrowthTrend(userID, 7) // 默认7天
	if err != nil {
		s.logger.Error("Failed to get growth trend", zap.Error(err))
		trendData = []models.TimeSeriesPoint{}
	}

	statistics := &models.AccountStatistics{
		TotalAccounts:          totalAccounts,
		ActiveAccounts:         activeAccounts,
		StatusDistribution:     statusDistribution,
		ConnectionDistribution: make(map[string]int64), // 简化实现
		ProxyUsage:             *proxyUsage,
		ActivityStats:          activityStats,
		RiskStats:              riskStats,
		TrendData:              trendData,
		GeneratedAt:            now,
		Period:                 period,
	}

	return statistics, nil
}

// GetUserDashboard 获取用户仪表盘数据
func (s *statsService) GetUserDashboard(ctx context.Context, userID uint64) (*models.UserDashboard, error) {
	s.logger.Info("Getting user dashboard", zap.Uint64("user_id", userID))

	now := time.Now()

	// 获取用户信息
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	userInfo := models.UserDashboardInfo{
		UserID:       user.ID,
		Username:     user.Username,
		Email:        user.Email,
		Role:         string(user.Role),
		LastLoginAt:  user.LastLoginAt,
		RegisteredAt: user.CreatedAt,
	}

	// 获取快速统计
	quickStats := s.getQuickStats(ctx, userID)

	// 获取最近活动（简化实现）
	recentActivities := []models.DashboardActivity{
		{
			ID:          1,
			Type:        "info",
			Description: "欢迎使用 TG Cloud",
			Icon:        "info",
			Color:       "info",
			CreatedAt:   now,
		},
	}

	// 获取系统通知（简化实现）
	systemNotifications := []models.SystemNotification{}

	// 获取性能指标
	tasksPerHour, _ := s.taskRepo.GetTasksPerHourTrend(userID, 24)
	successRateTrend, _ := s.taskRepo.GetSuccessRateTrend(userID, 24)
	accountGrowth, _ := s.accountRepo.GetGrowthTrend(userID, 7)

	performanceMetrics := models.DashboardMetrics{
		TasksPerHour:     tasksPerHour,
		SuccessRateTrend: successRateTrend,
		AccountGrowth:    accountGrowth,
	}

	dashboard := &models.UserDashboard{
		UserInfo:            userInfo,
		QuickStats:          quickStats,
		RecentActivities:    recentActivities,
		SystemNotifications: systemNotifications,
		PerformanceMetrics:  performanceMetrics,
		GeneratedAt:         now,
	}

	return dashboard, nil
}

// GetRealTimeStats 获取实时统计
func (s *statsService) GetRealTimeStats(ctx context.Context, userID uint64) (map[string]interface{}, error) {
	stats := map[string]interface{}{
		"online_accounts": 0, // 需要实现在线账号统计
		"running_tasks":   0, // 需要实现运行中任务统计
		"system_load":     "low",
		"last_update":     time.Now(),
	}

	return stats, nil
}

// GetSystemHealth 获取系统健康指标
func (s *statsService) GetSystemHealth(ctx context.Context) (*models.SystemHealth, error) {
	// 简化实现，实际应该从监控系统获取真实数据
	health := &models.SystemHealth{
		OverallScore:      85.5,
		AccountsHealth:    78.2,
		TasksSuccessRate:  92.1,
		ProxiesActiveRate: 88.7,
		SystemUptime:      "12天3小时",
		DatabaseLatency:   15.2,
	}

	return health, nil
}

// 辅助方法

// getPeriodStart 获取周期开始时间
func (s *statsService) getPeriodStart(now time.Time, period string) time.Time {
	switch period {
	case "day":
		return now.AddDate(0, 0, -1)
	case "week":
		return now.AddDate(0, 0, -7)
	case "month":
		return now.AddDate(0, -1, 0)
	default:
		return now.AddDate(0, 0, -7) // 默认一周
	}
}

// getUserCount 获取用户总数
func (s *statsService) getUserCount(ctx context.Context) (int64, error) {
	// 简化实现，应该通过repository获取
	return 100, nil
}

// getTaskCount 获取任务数量
func (s *statsService) getTaskCount(ctx context.Context, userID uint64, since time.Time) (int64, error) {
	// 简化实现，应该通过repository获取
	return 50, nil
}

// getProxyCount 获取代理数量
func (s *statsService) getProxyCount(ctx context.Context, userID uint64) (int64, error) {
	// 简化实现，应该通过repository获取
	return 10, nil
}

// getAccountCountSince 获取指定时间后的账号数量
func (s *statsService) getAccountCountSince(ctx context.Context, userID uint64, since time.Time) (int64, error) {
	// 简化实现，应该通过repository获取
	return 5, nil
}

// getAccountHealthDistribution 获取账号健康度分布
func (s *statsService) getAccountHealthDistribution(ctx context.Context, userID uint64) map[string]int64 {
	// 简化实现
	return map[string]int64{
		"excellent": 12, // 90-100
		"good":      8,  // 70-89
		"fair":      4,  // 50-69
		"poor":      1,  // 0-49
	}
}

// getProxyUsageStats 获取代理使用统计
func (s *statsService) getProxyUsageStats(ctx context.Context, userID uint64) models.ProxyUsageStats {
	return models.ProxyUsageStats{
		WithProxy:    20,
		WithoutProxy: 5,
		ProxyTypes: map[string]int64{
			"http":   8,
			"https":  5,
			"socks5": 7,
		},
		AvgLatency: 125.5,
	}
}

// getAccountActivityStats 获取账号活跃度统计
func (s *statsService) getAccountActivityStats(ctx context.Context, userID uint64) models.AccountActivityStats {
	return models.AccountActivityStats{
		ActiveToday:      18,
		ActiveThisWeek:   22,
		ActiveThisMonth:  25,
		InactiveAccounts: 3,
	}
}

// calculateRiskStats 根据状态分布计算风控统计
func (s *statsService) calculateRiskStats(distribution map[string]int64) models.AccountRiskStats {
	return models.AccountRiskStats{
		HighRiskAccounts:   distribution[string(models.AccountStatusDead)],
		MediumRiskAccounts: distribution[string(models.AccountStatusRestricted)] + distribution[string(models.AccountStatusWarning)],
		LowRiskAccounts:    distribution[string(models.AccountStatusNormal)] + distribution[string(models.AccountStatusNew)] + distribution[string(models.AccountStatusCooling)],
		RestrictedAccounts: distribution[string(models.AccountStatusRestricted)],
		DeadAccounts:       distribution[string(models.AccountStatusDead)],
		RecentBans:         0, // 需要额外逻辑
	}
}

// getQuickStats 获取快速统计
func (s *statsService) getQuickStats(ctx context.Context, userID uint64) models.DashboardQuickStats {
	totalAccounts, _ := s.accountRepo.CountByUserID(userID)
	activeAccounts, _ := s.accountRepo.CountActiveByUserID(userID)

	taskStats, _ := s.taskRepo.GetTaskStatsByUserID(userID, time.Time{}, time.Time{})

	var successRate float64
	if taskStats.Total > 0 {
		successRate = float64(taskStats.Completed) / float64(taskStats.Total) * 100
	}

	activeProxies, _ := s.getProxyCount(ctx, userID)

	return models.DashboardQuickStats{
		TotalAccounts:  totalAccounts,
		ActiveAccounts: activeAccounts,
		TodayTasks:     taskStats.TodayTasks,
		CompletedTasks: taskStats.Completed,
		FailedTasks:    taskStats.Failed,
		PendingTasks:   taskStats.Pending,
		RunningTasks:   taskStats.Running,
		CancelledTasks: taskStats.Cancelled,
		SuccessRate:    successRate,
		ActiveProxies:  activeProxies,
	}
}
