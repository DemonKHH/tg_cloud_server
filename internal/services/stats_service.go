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
	totalTasks, _ := s.getTaskCount(ctx, userID, time.Time{}) // 全部任务
	totalProxies, _ := s.getProxyCount(ctx, userID)

	// 获取周期内统计
	periodAccounts, _ := s.getAccountCountSince(ctx, userID, periodStart)
	periodTasks, _ := s.getTaskCount(ctx, userID, periodStart)

	// 获取状态分布
	accountsByStatus, _ := s.getAccountStatusDistribution(ctx, userID)
	tasksByStatus, _ := s.getTaskStatusDistribution(ctx, userID, periodStart)
	tasksByType, _ := s.getTaskTypeDistribution(ctx, userID, periodStart)

	// 获取系统健康指标
	systemHealth, _ := s.GetSystemHealth(ctx)

	overview := &models.SystemOverview{
		TotalUsers:       totalUsers,
		TotalAccounts:    totalAccounts,
		TotalTasks:       totalTasks,
		TotalProxies:     totalProxies,
		PeriodUsers:      0, // 简化实现，需要根据需求完善
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
	statusDistribution, _ := s.getAccountStatusDistribution(ctx, userID)

	// 获取代理使用情况
	proxyUsage := s.getProxyUsageStats(ctx, userID)

	// 获取活跃度统计
	activityStats := s.getAccountActivityStats(ctx, userID)

	// 获取风控统计
	riskStats := s.getAccountRiskStats(ctx, userID)

	// 获取趋势数据（简化实现）
	trendData := []models.TimeSeriesPoint{
		{Timestamp: now.AddDate(0, 0, -7), Value: float64(totalAccounts * 8 / 10), Label: "7天前"},
		{Timestamp: now.AddDate(0, 0, -5), Value: float64(totalAccounts * 9 / 10), Label: "5天前"},
		{Timestamp: now.AddDate(0, 0, -3), Value: float64(totalAccounts * 95 / 100), Label: "3天前"},
		{Timestamp: now, Value: float64(totalAccounts), Label: "现在"},
	}

	statistics := &models.AccountStatistics{
		TotalAccounts:          totalAccounts,
		ActiveAccounts:         activeAccounts,
		StatusDistribution:     statusDistribution,
		ConnectionDistribution: make(map[string]int64), // 简化实现
		ProxyUsage:             proxyUsage,
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
			Type:        "task_completed",
			Description: "任务执行完成",
			Icon:        "check-circle",
			Color:       "success",
			CreatedAt:   now.Add(-1 * time.Hour),
		},
		{
			ID:          2,
			Type:        "account_created",
			Description: "新增TG账号",
			Icon:        "user-plus",
			Color:       "info",
			CreatedAt:   now.Add(-2 * time.Hour),
		},
	}

	// 获取系统通知（简化实现）
	systemNotifications := []models.SystemNotification{
		{
			ID:        1,
			Type:      "info",
			Title:     "系统更新",
			Message:   "系统将在今晚进行例行维护",
			IsRead:    false,
			Priority:  3,
			CreatedAt: now.Add(-30 * time.Minute),
		},
	}

	// 获取性能指标（简化实现）
	performanceMetrics := models.DashboardMetrics{
		TasksPerHour:     s.getTasksPerHourTrend(ctx, userID),
		SuccessRateTrend: s.getSuccessRateTrend(ctx, userID),
		AccountGrowth:    s.getAccountGrowthTrend(ctx, userID),
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

// getAccountStatusDistribution 获取账号状态分布
func (s *statsService) getAccountStatusDistribution(ctx context.Context, userID uint64) (map[string]int64, error) {
	// 简化实现
	return map[string]int64{
		"normal":      15,
		"warning":     3,
		"restricted":  1,
		"dead":        0,
		"cooling":     2,
		"maintenance": 1,
		"new":         3,
	}, nil
}

// getTaskStatusDistribution 获取任务状态分布
func (s *statsService) getTaskStatusDistribution(ctx context.Context, userID uint64, since time.Time) (map[string]int64, error) {
	// 简化实现
	return map[string]int64{
		"completed": 45,
		"failed":    3,
		"running":   2,
		"pending":   5,
		"cancelled": 1,
	}, nil
}

// getTaskTypeDistribution 获取任务类型分布
func (s *statsService) getTaskTypeDistribution(ctx context.Context, userID uint64, since time.Time) (map[string]int64, error) {
	// 简化实现
	return map[string]int64{
		"check":     20,
		"private":   15,
		"broadcast": 10,
		"verify":    3,
		"groupchat": 8,
	}, nil
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

// getAccountRiskStats 获取账号风控统计
func (s *statsService) getAccountRiskStats(ctx context.Context, userID uint64) models.AccountRiskStats {
	return models.AccountRiskStats{
		HighRiskAccounts:   2,
		MediumRiskAccounts: 5,
		LowRiskAccounts:    18,
		RestrictedAccounts: 1,
		DeadAccounts:       0,
		RecentBans:         0,
	}
}

// getQuickStats 获取快速统计
func (s *statsService) getQuickStats(ctx context.Context, userID uint64) models.DashboardQuickStats {
	totalAccounts, _ := s.accountRepo.CountByUserID(userID)
	activeAccounts, _ := s.accountRepo.CountActiveByUserID(userID)

	return models.DashboardQuickStats{
		TotalAccounts:  totalAccounts,
		ActiveAccounts: activeAccounts,
		TodayTasks:     15,
		CompletedTasks: 142,
		FailedTasks:    8,
		SuccessRate:    94.7,
		ActiveProxies:  8,
	}
}

// getTrend methods (简化实现)
func (s *statsService) getTasksPerHourTrend(ctx context.Context, userID uint64) []models.TimeSeriesPoint {
	now := time.Now()
	return []models.TimeSeriesPoint{
		{Timestamp: now.Add(-3 * time.Hour), Value: 12, Label: "3h前"},
		{Timestamp: now.Add(-2 * time.Hour), Value: 18, Label: "2h前"},
		{Timestamp: now.Add(-1 * time.Hour), Value: 15, Label: "1h前"},
		{Timestamp: now, Value: 22, Label: "现在"},
	}
}

func (s *statsService) getSuccessRateTrend(ctx context.Context, userID uint64) []models.TimeSeriesPoint {
	now := time.Now()
	return []models.TimeSeriesPoint{
		{Timestamp: now.Add(-24 * time.Hour), Value: 93.2},
		{Timestamp: now.Add(-18 * time.Hour), Value: 94.1},
		{Timestamp: now.Add(-12 * time.Hour), Value: 92.8},
		{Timestamp: now.Add(-6 * time.Hour), Value: 95.2},
		{Timestamp: now, Value: 94.7},
	}
}

func (s *statsService) getAccountGrowthTrend(ctx context.Context, userID uint64) []models.TimeSeriesPoint {
	now := time.Now()
	return []models.TimeSeriesPoint{
		{Timestamp: now.Add(-7 * 24 * time.Hour), Value: 18},
		{Timestamp: now.Add(-5 * 24 * time.Hour), Value: 20},
		{Timestamp: now.Add(-3 * 24 * time.Hour), Value: 22},
		{Timestamp: now.Add(-1 * 24 * time.Hour), Value: 24},
		{Timestamp: now, Value: 25},
	}
}
