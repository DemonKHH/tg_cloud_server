package metrics

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Prometheus指标定义
var (
	// HTTP请求指标
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status_code"},
	)

	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	// 任务相关指标
	TasksTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "tasks_total",
			Help: "Total number of tasks",
		},
		[]string{"task_type", "status"},
	)

	TaskExecutionDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "task_execution_duration_seconds",
			Help:    "Task execution duration in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0, 120.0, 300.0},
		},
		[]string{"task_type", "account_id"},
	)

	TaskQueueLength = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "task_queue_length",
			Help: "Current length of task queue",
		},
		[]string{"account_id"},
	)

	// 账号相关指标
	AccountsTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "accounts_total",
			Help: "Total number of accounts",
		},
		[]string{"status", "user_id"},
	)

	AccountHealthScore = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "account_health_score",
			Help: "Account health score",
		},
		[]string{"account_id", "phone"},
	)

	// Telegram连接指标
	TelegramConnectionsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "telegram_connections_active",
			Help: "Number of active Telegram connections",
		},
		[]string{"account_id"},
	)

	TelegramAPICallsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "telegram_api_calls_total",
			Help: "Total number of Telegram API calls",
		},
		[]string{"method", "status"},
	)

	TelegramAPICallDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "telegram_api_call_duration_seconds",
			Help:    "Telegram API call duration in seconds",
			Buckets: []float64{0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"method"},
	)

	// 代理相关指标
	ProxiesTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "proxies_total",
			Help: "Total number of proxies",
		},
		[]string{"status", "user_id"},
	)

	ProxyLatency = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "proxy_latency_seconds",
			Help: "Proxy latency in seconds",
		},
		[]string{"proxy_id", "country"},
	)

	ProxySuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "proxy_success_rate",
			Help: "Proxy success rate",
		},
		[]string{"proxy_id", "country"},
	)

	// 缓存相关指标
	CacheOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "cache_operations_total",
			Help: "Total number of cache operations",
		},
		[]string{"operation", "status"},
	)

	CacheHitRatio = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cache_hit_ratio",
			Help: "Cache hit ratio",
		},
		[]string{"cache_type"},
	)

	// 数据库相关指标
	DatabaseConnectionsActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections_active",
			Help: "Number of active database connections",
		},
	)

	DatabaseQueryDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_query_duration_seconds",
			Help:    "Database query duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.05, 0.1, 0.5, 1.0, 5.0},
		},
		[]string{"query_type"},
	)

	// 系统资源指标
	MemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "memory_usage_bytes",
			Help: "Memory usage in bytes",
		},
	)

	CPUUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "cpu_usage_percent",
			Help: "CPU usage percentage",
		},
	)

	GoroutinesActive = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "goroutines_active",
			Help: "Number of active goroutines",
		},
	)
)

// MetricsService 指标服务
type MetricsService struct{}

// NewMetricsService 创建指标服务
func NewMetricsService() *MetricsService {
	return &MetricsService{}
}

// RecordHTTPRequest 记录HTTP请求指标
func (m *MetricsService) RecordHTTPRequest(method, endpoint, statusCode string, duration float64) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, statusCode).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordTaskExecution 记录任务执行指标
func (m *MetricsService) RecordTaskExecution(taskType, status string, accountID uint64, duration float64) {
	TasksTotal.WithLabelValues(taskType, status).Inc()
	TaskExecutionDuration.WithLabelValues(taskType, strconv.FormatUint(accountID, 10)).Observe(duration)
}

// UpdateTaskQueueLength 更新任务队列长度
func (m *MetricsService) UpdateTaskQueueLength(accountID uint64, length float64) {
	TaskQueueLength.WithLabelValues(strconv.FormatUint(accountID, 10)).Set(length)
}

// UpdateAccountCount 更新账号数量
func (m *MetricsService) UpdateAccountCount(status string, userID uint64, count float64) {
	AccountsTotal.WithLabelValues(status, strconv.FormatUint(userID, 10)).Set(count)
}

// UpdateAccountHealthScore 更新账号健康度评分
func (m *MetricsService) UpdateAccountHealthScore(accountID uint64, phone string, score float64) {
	AccountHealthScore.WithLabelValues(strconv.FormatUint(accountID, 10), phone).Set(score)
}

// UpdateTelegramConnections 更新Telegram连接数
func (m *MetricsService) UpdateTelegramConnections(accountID uint64, connections float64) {
	TelegramConnectionsActive.WithLabelValues(strconv.FormatUint(accountID, 10)).Set(connections)
}

// RecordTelegramAPICall 记录Telegram API调用
func (m *MetricsService) RecordTelegramAPICall(method, status string, duration float64) {
	TelegramAPICallsTotal.WithLabelValues(method, status).Inc()
	TelegramAPICallDuration.WithLabelValues(method).Observe(duration)
}

// UpdateProxyCount 更新代理数量
func (m *MetricsService) UpdateProxyCount(status string, userID uint64, count float64) {
	ProxiesTotal.WithLabelValues(status, strconv.FormatUint(userID, 10)).Set(count)
}

// UpdateProxyMetrics 更新代理指标
func (m *MetricsService) UpdateProxyMetrics(proxyID uint64, country string, latency, successRate float64) {
	ProxyLatency.WithLabelValues(strconv.FormatUint(proxyID, 10), country).Set(latency)
	ProxySuccessRate.WithLabelValues(strconv.FormatUint(proxyID, 10), country).Set(successRate)
}

// RecordCacheOperation 记录缓存操作
func (m *MetricsService) RecordCacheOperation(operation, status string) {
	CacheOperationsTotal.WithLabelValues(operation, status).Inc()
}

// UpdateCacheHitRatio 更新缓存命中率
func (m *MetricsService) UpdateCacheHitRatio(cacheType string, ratio float64) {
	CacheHitRatio.WithLabelValues(cacheType).Set(ratio)
}

// UpdateDatabaseConnections 更新数据库连接数
func (m *MetricsService) UpdateDatabaseConnections(connections float64) {
	DatabaseConnectionsActive.Set(connections)
}

// RecordDatabaseQuery 记录数据库查询
func (m *MetricsService) RecordDatabaseQuery(queryType string, duration float64) {
	DatabaseQueryDuration.WithLabelValues(queryType).Observe(duration)
}

// UpdateSystemMetrics 更新系统指标
func (m *MetricsService) UpdateSystemMetrics(memoryUsage, cpuUsage, goroutines float64) {
	MemoryUsage.Set(memoryUsage)
	CPuUsage.Set(cpuUsage)
	GoroutinesActive.Set(goroutines)
}

// PrometheusMiddleware Prometheus中间件，用于收集HTTP指标
func PrometheusMiddleware() gin.HandlerFunc {
	metricsService := NewMetricsService()

	return func(c *gin.Context) {
		start := prometheus.NewTimer(prometheus.ObserverFunc(func(v float64) {
			metricsService.RecordHTTPRequest(
				c.Request.Method,
				c.FullPath(),
				strconv.Itoa(c.Writer.Status()),
				v,
			)
		}))
		defer start.ObserveDuration()

		c.Next()
	}
}

// RegisterMetricsHandler 注册指标处理器
func RegisterMetricsHandler(router *gin.Engine) {
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))
}
