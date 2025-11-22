package models

import "time"

// SystemOverview 系统统计概览
type SystemOverview struct {
	// 基本统计
	TotalUsers    int64 `json:"total_users"`
	TotalAccounts int64 `json:"total_accounts"`
	TotalTasks    int64 `json:"total_tasks"`
	TotalProxies  int64 `json:"total_proxies"`

	// 周期内统计
	PeriodUsers    int64 `json:"period_users"`    // 周期内新增用户
	PeriodAccounts int64 `json:"period_accounts"` // 周期内新增账号
	PeriodTasks    int64 `json:"period_tasks"`    // 周期内新增任务

	// 账号状态分布
	AccountsByStatus map[string]int64 `json:"accounts_by_status"`

	// 任务状态分布
	TasksByStatus map[string]int64 `json:"tasks_by_status"`

	// 任务类型分布
	TasksByType map[string]int64 `json:"tasks_by_type"`

	// 系统健康指标
	SystemHealth SystemHealth `json:"system_health"`

	// 生成时间
	GeneratedAt time.Time `json:"generated_at"`
	Period      string    `json:"period"`
}

// SystemHealth 系统健康指标
type SystemHealth struct {
	OverallScore      float64 `json:"overall_score"`       // 总体健康分数 (0-100)
	AccountsHealth    float64 `json:"accounts_health"`     // 账号健康平均分
	TasksSuccessRate  float64 `json:"tasks_success_rate"`  // 任务成功率
	ProxiesActiveRate float64 `json:"proxies_active_rate"` // 代理活跃率
	SystemUptime      string  `json:"system_uptime"`       // 系统运行时间
	DatabaseLatency   float64 `json:"database_latency_ms"` // 数据库延迟(毫秒)
}

// AccountStatistics 账号统计详情
type AccountStatistics struct {
	// 基本统计
	TotalAccounts  int64 `json:"total_accounts"`
	ActiveAccounts int64 `json:"active_accounts"`

	// 状态分布
	StatusDistribution map[string]int64 `json:"status_distribution"`

	// 连接状态分布
	ConnectionDistribution map[string]int64 `json:"connection_distribution"`

	// 代理使用情况
	ProxyUsage ProxyUsageStats `json:"proxy_usage"`

	// 账号活跃度
	ActivityStats AccountActivityStats `json:"activity_stats"`

	// 风控统计
	RiskStats AccountRiskStats `json:"risk_stats"`

	// 趋势数据
	TrendData []TimeSeriesPoint `json:"trend_data"`

	// 生成时间
	GeneratedAt time.Time `json:"generated_at"`
	Period      string    `json:"period"`
}

// ProxyUsageStats 代理使用统计
type ProxyUsageStats struct {
	WithProxy    int64            `json:"with_proxy"`     // 使用代理的账号数
	WithoutProxy int64            `json:"without_proxy"`  // 未使用代理的账号数
	ProxyTypes   map[string]int64 `json:"proxy_types"`    // 代理类型分布
	AvgLatency   float64          `json:"avg_latency_ms"` // 平均延迟
}

// AccountActivityStats 账号活跃度统计
type AccountActivityStats struct {
	ActiveToday      int64 `json:"active_today"`      // 今日活跃账号
	ActiveThisWeek   int64 `json:"active_this_week"`  // 本周活跃账号
	ActiveThisMonth  int64 `json:"active_this_month"` // 本月活跃账号
	InactiveAccounts int64 `json:"inactive_accounts"` // 非活跃账号
}

// AccountRiskStats 账号风控统计
type AccountRiskStats struct {
	HighRiskAccounts   int64 `json:"high_risk_accounts"`   // 高风险账号
	MediumRiskAccounts int64 `json:"medium_risk_accounts"` // 中风险账号
	LowRiskAccounts    int64 `json:"low_risk_accounts"`    // 低风险账号
	RestrictedAccounts int64 `json:"restricted_accounts"`  // 受限账号
	DeadAccounts       int64 `json:"dead_accounts"`        // 死亡账号
	RecentBans         int64 `json:"recent_bans"`          // 近期封禁
}

// UserDashboard 用户仪表盘数据
type UserDashboard struct {
	// 用户基本信息
	UserInfo UserDashboardInfo `json:"user_info"`

	// 快速统计
	QuickStats DashboardQuickStats `json:"quick_stats"`

	// 最近活动
	RecentActivities []DashboardActivity `json:"recent_activities"`

	// 系统通知
	SystemNotifications []SystemNotification `json:"system_notifications"`

	// 性能指标
	PerformanceMetrics DashboardMetrics `json:"performance_metrics"`

	// 生成时间
	GeneratedAt time.Time `json:"generated_at"`
}

// UserDashboardInfo 用户仪表盘基本信息
type UserDashboardInfo struct {
	UserID       uint64     `json:"user_id"`
	Username     string     `json:"username"`
	Email        string     `json:"email"`
	Role         string     `json:"role"`
	LastLoginAt  *time.Time `json:"last_login_at"`
	RegisteredAt time.Time  `json:"registered_at"`
}

// DashboardQuickStats 仪表盘快速统计
type DashboardQuickStats struct {
	TotalAccounts  int64   `json:"total_accounts"`
	ActiveAccounts int64   `json:"active_accounts"`
	TodayTasks     int64   `json:"today_tasks"`
	CompletedTasks int64   `json:"completed_tasks"`
	FailedTasks    int64   `json:"failed_tasks"`
	SuccessRate    float64 `json:"success_rate"`
	ActiveProxies  int64   `json:"active_proxies"`
}

// DashboardActivity 仪表盘活动记录
type DashboardActivity struct {
	ID          uint64    `json:"id"`
	Type        string    `json:"type"` // task_completed, account_created, etc.
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Color       string    `json:"color"`
	CreatedAt   time.Time `json:"created_at"`
}

// SystemNotification 系统通知
type SystemNotification struct {
	ID        uint64    `json:"id"`
	Type      string    `json:"type"` // info, warning, error, success
	Title     string    `json:"title"`
	Message   string    `json:"message"`
	IsRead    bool      `json:"is_read"`
	Priority  int       `json:"priority"` // 1-5, 5最高
	CreatedAt time.Time `json:"created_at"`
}

// DashboardMetrics 仪表盘性能指标
type DashboardMetrics struct {
	TasksPerHour     []TimeSeriesPoint `json:"tasks_per_hour"`
	SuccessRateTrend []TimeSeriesPoint `json:"success_rate_trend"`
	AccountGrowth    []TimeSeriesPoint `json:"account_growth"`
}

// TimeSeriesPoint 时间序列数据点
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
	Label     string    `json:"label,omitempty"`
}
