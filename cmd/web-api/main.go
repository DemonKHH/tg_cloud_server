package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/cache"
	"tg_cloud_server/internal/common/config"
	"tg_cloud_server/internal/common/database"
	"tg_cloud_server/internal/common/health"
	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/metrics"
	"tg_cloud_server/internal/common/middleware"
	"tg_cloud_server/internal/common/response"
	"tg_cloud_server/internal/common/validator"
	"tg_cloud_server/internal/cron"
	"tg_cloud_server/internal/events"
	"tg_cloud_server/internal/handlers"
	"tg_cloud_server/internal/repository"
	"tg_cloud_server/internal/routes"
	"tg_cloud_server/internal/services"
)

func main() {
	// 加载配置
	configPath := os.Getenv("CONFIG_PATH")
	if configPath == "" {
		configPath = "configs/config.yaml"
	}

	if err := config.Load(configPath); err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	cfg := config.Get()

	// 初始化日志
	if err := logger.Init(&cfg.Logging); err != nil {
		log.Fatalf("Failed to initialize logger: %v", err)
	}

	logger := logger.Get()
	defer logger.Sync()

	version := "1.0.0"
	logger.Info("Starting Web API service", zap.String("version", version))

	// 初始化自定义验证器
	validator.InitCustomValidator()

	// 初始化数据库
	db, err := database.InitMySQL(&cfg.Database.MySQL)
	if err != nil {
		logger.Fatal("Failed to connect to MySQL", zap.Error(err))
	}

	// 初始化Redis
	redisClient, err := database.InitRedis(&cfg.Database.Redis)
	if err != nil {
		logger.Fatal("Failed to connect to Redis", zap.Error(err))
	}

	// 初始化缓存服务
	_ = cache.NewCacheService(cache.NewRedisCache(redisClient))

	// 初始化事件系统
	eventBus := events.NewInMemoryEventBus()
	eventService := events.NewEventService(eventBus)

	// 注册事件处理器
	loggingHandler := events.NewLoggingEventHandler()
	metricsHandler := events.NewMetricsEventHandler()

	for _, eventType := range loggingHandler.SupportedTypes() {
		eventBus.Subscribe(eventType, loggingHandler)
	}
	for _, eventType := range metricsHandler.SupportedTypes() {
		eventBus.Subscribe(eventType, metricsHandler)
	}

	// 初始化健康检查服务
	healthService := health.NewHealthService(version)
	healthService.AddChecker(health.NewDatabaseHealthChecker(db))
	healthService.AddChecker(health.NewRedisHealthChecker(redisClient))
	healthService.AddChecker(health.NewSystemHealthChecker())

	// 初始化仓库层
	userRepo := repository.NewUserRepository(db)
	accountRepo := repository.NewAccountRepository(db)
	taskRepo := repository.NewTaskRepository(db)
	proxyRepo := repository.NewProxyRepository(db)

	// 初始化服务层
	authService := services.NewAuthService(userRepo, cfg)
	accountService := services.NewAccountService(accountRepo, proxyRepo)
	proxyService := services.NewProxyService(proxyRepo)
	taskService := services.NewTaskService(taskRepo, accountRepo)
	statsService := services.NewStatsService(userRepo, accountRepo, taskRepo, proxyRepo)

	// 初始化定时任务服务
	cronService := cron.NewCronService(taskService, accountService, userRepo, taskRepo, accountRepo)

	// 初始化处理器
	authHandler := handlers.NewAuthHandler(authService)
	accountHandler := handlers.NewAccountHandler(accountService)
	taskHandler := handlers.NewTaskHandler(taskService)
	proxyHandler := handlers.NewProxyHandler(proxyService)
	moduleHandler := handlers.NewModuleHandler(taskService, accountService)
	statsHandler := handlers.NewStatsHandler(statsService)

	// 设置Gin模式
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	// 初始化路由
	router := gin.New()

	// 添加中间件
	router.Use(response.SetRequestID())           // 请求ID中间件
	router.Use(middleware.Logger(logger))         // 日志中间件
	router.Use(middleware.Recovery(logger))       // 恢复中间件
	router.Use(middleware.CORS())                 // CORS中间件
	router.Use(middleware.RateLimit(redisClient)) // 限流中间件
	router.Use(metrics.PrometheusMiddleware())    // 指标收集中间件

	// 注册路由
	routes.RegisterAuthRoutes(router, authHandler)
	routes.RegisterAPIRoutes(router, accountHandler, taskHandler, proxyHandler, moduleHandler, statsHandler, authService, cfg)
	routes.RegisterWebSocketRoutes(router, redisClient)

	// 注册指标端点
	metrics.RegisterMetricsHandler(router)

	// 健康检查端点（简单版本）
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":    "healthy",
			"service":   "web-api",
			"version":   version,
			"timestamp": time.Now().Unix(),
		})
	})

	// 详细健康检查端点
	router.GET("/health/detailed", func(c *gin.Context) {
		health := healthService.CheckHealth(c.Request.Context())
		statusCode := http.StatusOK
		if health.Status == "unhealthy" {
			statusCode = http.StatusServiceUnavailable
		}
		c.JSON(statusCode, health)
	})

	// 系统信息端点
	router.GET("/info", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"service":   "tg-cloud-server",
			"version":   version,
			"uptime":    time.Since(time.Now()).String(),
			"timestamp": time.Now().Unix(),
		})
	})

	// 创建HTTP服务器
	server := &http.Server{
		Addr:    cfg.GetServiceAddr("web_api"),
		Handler: router,
	}

	// 启动定时任务服务
	if err := cronService.Start(); err != nil {
		logger.Fatal("Failed to start cron service", zap.Error(err))
	}

	// 发布系统启动事件
	eventService.PublishSystemEvent(context.Background(), events.EventSystemStarted, map[string]interface{}{
		"version":      version,
		"startup_time": time.Now(),
	})

	// 在goroutine中启动服务器
	go func() {
		logger.Info("Web API server starting",
			zap.String("addr", server.Addr))

		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	}()

	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("Shutting down Web API server...")

	// 发布系统停止事件
	eventService.PublishSystemEvent(context.Background(), events.EventSystemStopped, map[string]interface{}{
		"shutdown_time": time.Now(),
	})

	// 创建10秒超时的上下文用于关闭
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 停止定时任务服务
	cronService.Stop()

	// 优雅关闭服务器
	if err := server.Shutdown(ctx); err != nil {
		logger.Fatal("Server forced to shutdown", zap.Error(err))
	}

	// 关闭事件总线
	if err := eventBus.Close(); err != nil {
		logger.Error("Failed to close event bus", zap.Error(err))
	}

	// 关闭数据库连接
	if sqlDB, err := db.DB(); err == nil {
		sqlDB.Close()
		logger.Info("Database connections closed")
	}

	// 关闭Redis连接
	redisClient.Close()
	logger.Info("Redis connection closed")

	logger.Info("Web API server stopped gracefully")
}
