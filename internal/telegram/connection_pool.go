package telegram

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/models"
	"tg_cloud_server/internal/repository"
)

// 使用 models 包中定义的 ConnectionStatus
type ConnectionStatus = models.ConnectionStatus

const (
	StatusDisconnected    = models.StatusDisconnected
	StatusConnecting      = models.StatusConnecting
	StatusConnected       = models.StatusConnected
	StatusReconnecting    = models.StatusReconnecting
	StatusConnectionError = models.StatusConnectionError
)

// 添加别名以保持向下兼容
const StatusError = StatusConnectionError

// ManagedConnection 托管连接封装
// 重连相关常量
const (
	MaxReconnectAttempts  = 3                // 最大重连次数
	InitialReconnectDelay = 10 * time.Second // 初始重连延迟
	MaxReconnectDelay     = 30 * time.Second // 最大重连延迟
)

type ManagedConnection struct {
	client          *telegram.Client
	config          *ClientConfig
	status          ConnectionStatus
	lastUsed        time.Time
	useCount        int64
	isActive        bool
	taskRunning     bool
	reconnectCount  int           // 重连次数计数器
	lastReconnectAt time.Time     // 上次重连时间
	stateChangeCh   chan struct{} // 状态变更通知通道
	mu              sync.Mutex
	ctx             context.Context
	cancel          context.CancelFunc
	logger          *zap.Logger
}

// notifyStateChange 通知状态变更
func (c *ManagedConnection) notifyStateChange() {
	select {
	case c.stateChangeCh <- struct{}{}:
	default:
		// 通道已满，说明已有挂起的通知，忽略
	}
}

// ClientConfig 客户端配置
type ClientConfig struct {
	AppID       int
	AppHash     string
	Phone       string
	SessionData []byte
	ProxyConfig *ProxyConfig
}

// ProxyConfig 代理配置
type ProxyConfig struct {
	Protocol string `json:"protocol"`
	IP       string `json:"ip"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ConnectionPool 统一连接池管理器
type ConnectionPool struct {
	connections    map[string]*ManagedConnection
	configs        map[string]*ClientConfig
	mu             sync.RWMutex
	maxIdle        time.Duration
	cleanupTicker  *time.Ticker
	logger         *zap.Logger
	appID          int
	appHash        string
	accountRepo    repository.AccountRepository
	proxyRepo      repository.ProxyRepository
	updateHandlers map[string]telegram.UpdateHandler
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(appID int, appHash string, maxIdle time.Duration, accountRepo repository.AccountRepository, proxyRepo repository.ProxyRepository) *ConnectionPool {
	cp := &ConnectionPool{
		connections:    make(map[string]*ManagedConnection),
		configs:        make(map[string]*ClientConfig),
		maxIdle:        maxIdle,
		logger:         logger.Get().Named("connection_pool"),
		appID:          appID,
		appHash:        appHash,
		accountRepo:    accountRepo,
		proxyRepo:      proxyRepo,
		updateHandlers: make(map[string]telegram.UpdateHandler),
	}

	// 启动清理定时器
	cp.cleanupTicker = time.NewTicker(5 * time.Minute)
	go cp.cleanupLoop()

	return cp
}

// GetOrCreateConnection 获取或创建连接 (核心方法)
func (cp *ConnectionPool) GetOrCreateConnection(accountID string, config *ClientConfig) (*ManagedConnection, error) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.logger.Debug("GetOrCreateConnection called",
		zap.String("account_id", accountID),
		zap.String("phone", config.Phone),
		zap.Bool("has_proxy", config.ProxyConfig != nil))

	// 检查是否已存在连接
	if conn, exists := cp.connections[accountID]; exists {
		// 复用连接：只要连接是活跃的，无论是已连接、连接中还是重连中，都应该复用，
		// 让 waitForConnection 去处理等待逻辑
		if conn.isActive && (conn.status == StatusConnected || conn.status == StatusConnecting || conn.status == StatusReconnecting) {
			conn.lastUsed = time.Now()
			conn.useCount++
			cp.logger.Info("Reusing existing connection",
				zap.String("account_id", accountID),
				zap.String("phone", config.Phone),
				zap.String("status", conn.status.String()),
				zap.Int64("use_count", conn.useCount),
				zap.Duration("idle_time", time.Since(conn.lastUsed)))
			return conn, nil
		}
		cp.logger.Warn("Existing connection is not active or in error state",
			zap.String("account_id", accountID),
			zap.String("status", conn.status.String()),
			zap.Bool("is_active", conn.isActive))
	}

	// 检查是否已存在旧连接，如果存在，先取消它以便通知等待者
	if oldConn, exists := cp.connections[accountID]; exists {
		cp.logger.Info("Canceling old connection before creating new one", zap.String("account_id", accountID))
		oldConn.cancel()
		// 不需要 delete，因为下面会直接覆盖
	}

	// 创建新连接
	cp.logger.Info("Creating new connection", zap.String("account_id", accountID))
	return cp.createNewConnection(accountID, config)
}

// createNewConnection 创建新连接
func (cp *ConnectionPool) createNewConnection(accountID string, config *ClientConfig) (*ManagedConnection, error) {
	ctx, cancel := context.WithCancel(context.Background())

	// 转换accountID为uint64
	var accountIDNum uint64
	if accountID != "" {
		_, err := fmt.Sscanf(accountID, "%d", &accountIDNum)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("invalid account ID format: %w", err)
		}
	}

	// 创建Session存储（使用数据库持久化）
	sessionStorage := NewDatabaseSessionStorage(
		accountIDNum,
		cp.accountRepo,
		config.SessionData,
	)

	options := telegram.Options{
		SessionStorage: sessionStorage,
		UpdateHandler:  cp.createUpdateDispatcher(accountID),
	}

	// 配置代理 (固定绑定)
	if config.ProxyConfig != nil {
		// 创建代理dialer
		proxyDialer, err := createProxyDialer(config.ProxyConfig)
		if err != nil {
			cancel()
			return nil, fmt.Errorf("failed to create proxy dialer: %w", err)
		}

		// 将proxy.Dialer适配为context-aware dialer供gotd/td使用
		adapter := &proxyDialerAdapter{dialer: proxyDialer}

		// 创建使用代理的Resolver
		resolver := dcs.Plain(dcs.PlainOptions{
			Dial: adapter.DialContext,
		})
		options.Resolver = resolver

		cp.logger.Info("Proxy configuration applied for account",
			zap.String("account_id", accountID),
			zap.String("proxy", fmt.Sprintf("%s://%s:%d", config.ProxyConfig.Protocol, config.ProxyConfig.IP, config.ProxyConfig.Port)))

		// 测试代理连接（可选，用于验证代理是否可用）
		if err := testProxyConnection(config.ProxyConfig); err != nil {
			cp.logger.Warn("Proxy connection test failed, but will continue",
				zap.String("account_id", accountID),
				zap.Error(err))
		} else {
			cp.logger.Info("Proxy connection test successful",
				zap.String("account_id", accountID))
		}
	}

	client := telegram.NewClient(cp.appID, cp.appHash, options)

	conn := &ManagedConnection{
		client:        client,
		config:        config,
		status:        StatusConnecting,
		stateChangeCh: make(chan struct{}, 1),
		lastUsed:      time.Now(),
		isActive:      true,
		ctx:           ctx,
		cancel:        cancel,
		logger:        cp.logger.Named(accountID),
	}

	// 异步建立连接
	go cp.maintainConnection(accountID, conn)

	cp.connections[accountID] = conn
	cp.configs[accountID] = config

	return conn, nil
}

// maintainConnection 维护连接状态
func (cp *ConnectionPool) maintainConnection(accountID string, conn *ManagedConnection) {
	conn.logger.Info("Starting connection maintenance",
		zap.String("account_id", accountID),
		zap.String("phone", conn.config.Phone),
		zap.Bool("has_proxy", conn.config.ProxyConfig != nil))

	startTime := time.Now()

	err := conn.client.Run(conn.ctx, func(ctx context.Context) error {
		conn.mu.Lock()
		conn.status = StatusConnected
		// 连接成功，重置重连计数器
		if conn.reconnectCount > 0 {
			conn.logger.Info("Connection recovered after reconnect attempts",
				zap.String("account_id", accountID),
				zap.Int("previous_attempts", conn.reconnectCount),
				zap.Duration("recovery_time", time.Since(startTime)))
		}
		conn.reconnectCount = 0
		conn.notifyStateChange() // 通知状态变更
		conn.mu.Unlock()

		conn.logger.Info("Connection established successfully",
			zap.String("account_id", accountID),
			zap.String("phone", conn.config.Phone),
			zap.Duration("connect_time", time.Since(startTime)))

		// 连接成功，更新账号状态为正常
		cp.updateAccountStatusOnSuccess(accountID)

		// 连接成功后，获取并更新账号信息（在同一个 Run 上下文中）
		go cp.updateAccountInfoFromTelegram(accountID, conn, ctx)

		// 更新在线状态为在线
		cp.updateConnectionStatus(accountID, true)
		defer func() {
			cp.updateConnectionStatus(accountID, false)
			conn.logger.Info("Connection closed",
				zap.String("account_id", accountID),
				zap.Duration("session_duration", time.Since(startTime)))
		}()

		// 保持连接直到取消
		<-ctx.Done()
		return ctx.Err()
	})

	if err != nil && err != context.Canceled {
		conn.logger.Error("Connection error occurred",
			zap.Error(err),
			zap.String("error_type", fmt.Sprintf("%T", err)),
			zap.String("account_id", accountID),
			zap.String("phone", conn.config.Phone),
			zap.Int("session_data_length", len(conn.config.SessionData)),
			zap.Int("reconnect_count", conn.reconnectCount),
			zap.Duration("session_duration", time.Since(startTime)))

		conn.mu.Lock()
		conn.status = StatusReconnecting // 改为重连中，而不是 Error，以便任务等待
		conn.notifyStateChange()         // 通知状态变更
		conn.mu.Unlock()

		// 确保在线状态为离线（连接错误时）
		cp.updateConnectionStatus(accountID, false)

		// 更新账号状态
		cp.updateAccountStatusOnError(accountID, err)

		// 自动重连逻辑
		conn.logger.Info("Scheduling automatic reconnection",
			zap.String("account_id", accountID),
			zap.Int("current_reconnect_count", conn.reconnectCount))
		cp.scheduleReconnect(accountID, conn)
	} else if err == context.Canceled {
		conn.logger.Info("Connection canceled by context",
			zap.String("account_id", accountID),
			zap.Duration("session_duration", time.Since(startTime)))
	}
}

// scheduleReconnect 调度重连（带重试次数限制和指数退避）
func (cp *ConnectionPool) scheduleReconnect(accountID string, conn *ManagedConnection) {
	conn.mu.Lock()
	conn.reconnectCount++
	currentAttempt := conn.reconnectCount
	conn.lastReconnectAt = time.Now()
	conn.mu.Unlock()

	cp.logger.Info("Reconnect attempt scheduled",
		zap.String("account_id", accountID),
		zap.String("phone", conn.config.Phone),
		zap.Int("attempt", currentAttempt),
		zap.Int("max_attempts", MaxReconnectAttempts))

	// 检查是否超过最大重连次数
	if currentAttempt > MaxReconnectAttempts {
		cp.logger.Error("Max reconnect attempts reached, giving up",
			zap.String("account_id", accountID),
			zap.String("phone", conn.config.Phone),
			zap.Int("attempts", currentAttempt-1),
			zap.Duration("total_reconnect_time", time.Since(conn.lastReconnectAt)))

		// 移除连接，不再重试
		cp.mu.Lock()
		if currentConn, exists := cp.connections[accountID]; exists && currentConn == conn {
			conn.cancel()
			delete(cp.connections, accountID)
			go cp.updateConnectionStatus(accountID, false)
		}
		cp.mu.Unlock()
		return
	}

	// 计算指数退避延迟: 30s, 60s, 120s, 240s, 300s(max)
	delay := InitialReconnectDelay * time.Duration(1<<(currentAttempt-1))
	if delay > MaxReconnectDelay {
		delay = MaxReconnectDelay
	}

	// 设置状态为重连中，以便任务可以等待
	conn.mu.Lock()
	conn.status = StatusReconnecting
	conn.notifyStateChange() // 通知状态变更
	conn.mu.Unlock()

	cp.logger.Info("Scheduling reconnection with exponential backoff",
		zap.String("account_id", accountID),
		zap.String("phone", conn.config.Phone),
		zap.Int("attempt", currentAttempt),
		zap.Int("max_attempts", MaxReconnectAttempts),
		zap.Duration("delay", delay),
		zap.Time("next_attempt_at", time.Now().Add(delay)))

	time.AfterFunc(delay, func() {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		// 检查连接是否仍然存在且需要重连
		if currentConn, exists := cp.connections[accountID]; exists && currentConn == conn {
			if config, configExists := cp.configs[accountID]; configExists {
				conn.logger.Info("Attempting to reconnect",
					zap.Int("attempt", currentAttempt))

				// 创建新连接时继承重连计数
				newConn, err := cp.createNewConnection(accountID, config)
				if err != nil {
					conn.logger.Error("Failed to create new connection during reconnect",
						zap.Error(err))
					return
				}
				// 继承重连计数到新连接
				newConn.mu.Lock()
				newConn.reconnectCount = currentAttempt
				newConn.mu.Unlock()
			}
		}
	})
}

// ExecuteTask 执行任务 (复用连接)
func (cp *ConnectionPool) ExecuteTask(accountID string, task TaskInterface) error {
	taskStartTime := time.Now()
	taskType := task.GetType()

	cp.logger.Info("ExecuteTask started",
		zap.String("account_id", accountID),
		zap.String("task_type", taskType))

	config, exists := cp.configs[accountID]
	if !exists {
		// 动态加载账号配置
		cp.logger.Debug("Loading account config dynamically",
			zap.String("account_id", accountID))
		var err error
		config, err = cp.loadAccountConfig(accountID)
		if err != nil {
			cp.logger.Error("Failed to load account configuration",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType),
				zap.Error(err))
			return fmt.Errorf("failed to load account configuration: %w", err)
		}
		cp.logger.Info("Account config loaded successfully",
			zap.String("account_id", accountID),
			zap.String("phone", config.Phone))
	}

	var conn *ManagedConnection
	var err error

	// 尝试获取连接并等待连接就绪，支持在连接被替换（重连）时重试
	maxRetries := 3
	for i := 0; i < maxRetries; i++ {
		conn, err = cp.GetOrCreateConnection(accountID, config)
		if err != nil {
			// 连接失败，更新账号状态为警告
			cp.updateAccountStatusOnError(accountID, err)
			cp.logger.Error("Failed to get connection",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType),
				zap.Int("attempt", i+1),
				zap.Error(err))
			return fmt.Errorf("failed to get connection: %w", err)
		}

		// 确保单任务执行
		conn.mu.Lock()
		if conn.taskRunning {
			conn.mu.Unlock()
			cp.logger.Warn("Account is busy with another task",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType))
			return errors.New("account is busy with another task")
		}
		conn.taskRunning = true
		conn.mu.Unlock()

		// 等待连接建立完成
		cp.logger.Debug("Waiting for connection to be ready",
			zap.String("account_id", accountID),
			zap.String("task_type", taskType),
			zap.Int("attempt", i+1))

		_, err = cp.waitForConnection(accountID, conn)
		if err == nil {
			// 成功建立连接
			cp.logger.Info("Connection ready for task execution",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType),
				zap.Duration("wait_time", time.Since(taskStartTime)))
			break
		}

		// 等待失败，释放占用状态
		conn.mu.Lock()
		conn.taskRunning = false
		conn.mu.Unlock()

		// 检查是否是因为连接被替换（这是正常的重连流程）
		if strings.Contains(err.Error(), "connection was replaced") || strings.Contains(err.Error(), "please retry") {
			cp.logger.Info("Connection replaced during wait, retrying...",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType),
				zap.Int("attempt", i+1))
			continue
		}

		// 其他错误直接返回
		cp.logger.Error("Failed to wait for connection",
			zap.String("account_id", accountID),
			zap.String("task_type", taskType),
			zap.Int("attempt", i+1),
			zap.Error(err))
		return err
	}

	if err != nil {
		cp.logger.Error("Failed to establish connection after all retries",
			zap.String("account_id", accountID),
			zap.String("task_type", taskType),
			zap.Int("max_retries", maxRetries),
			zap.Error(err))
		return fmt.Errorf("failed to establish connection after retries: %w", err)
	}

	// 连接成功，更新账号状态为正常（如果之前不是正常状态）
	cp.updateAccountStatusOnSuccess(accountID)

	// 直接使用已建立的连接执行任务
	conn.logger.Info("Executing task",
		zap.String("account_id", accountID),
		zap.String("task_type", taskType),
		zap.Duration("setup_time", time.Since(taskStartTime)))

	// 执行任务并捕获错误
	// 注意：不要再次调用 conn.client.Run，因为 maintainConnection 已经在运行它了
	// 直接执行任务逻辑
	taskExecStartTime := time.Now()
	taskErr := func() error {
		ctx := context.Background()

		// 安全检查：确保 client 不为 nil
		if conn.client == nil {
			cp.logger.Error("Connection client is nil",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType))
			return errors.New("connection client is nil")
		}

		if advancedTask, ok := task.(AdvancedTaskInterface); ok {
			cp.logger.Debug("Executing advanced task",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType))
			return advancedTask.ExecuteAdvanced(ctx, conn.client)
		}

		// 安全检查：确保 API 不为 nil
		api := conn.client.API()
		if api == nil {
			cp.logger.Error("Connection API is nil",
				zap.String("account_id", accountID),
				zap.String("task_type", taskType))
			return errors.New("connection API is nil, connection may not be fully established")
		}

		return task.Execute(ctx, api)
	}()

	taskExecDuration := time.Since(taskExecStartTime)
	totalDuration := time.Since(taskStartTime)

	// 释放任务运行状态
	conn.mu.Lock()
	conn.taskRunning = false
	conn.mu.Unlock()

	// 根据任务执行结果更新账号状态
	if taskErr != nil {
		cp.logger.Error("Task execution failed",
			zap.String("account_id", accountID),
			zap.String("task_type", taskType),
			zap.Duration("exec_duration", taskExecDuration),
			zap.Duration("total_duration", totalDuration),
			zap.Error(taskErr))
		cp.updateAccountStatusOnTaskError(accountID, taskErr)
	} else {
		cp.logger.Info("Task execution completed successfully",
			zap.String("account_id", accountID),
			zap.String("task_type", taskType),
			zap.Duration("exec_duration", taskExecDuration),
			zap.Duration("total_duration", totalDuration))
		cp.updateAccountStatusOnSuccess(accountID)
	}

	return taskErr
}

// waitForConnection 等待连接建立（事件驱动版本，去轮询）
func (cp *ConnectionPool) waitForConnection(accountID string, conn *ManagedConnection) (*ManagedConnection, error) {
	// 等待连接建立的超时时间 (增加到 90s 以覆盖重连周期)
	// 如果连接彻底失败（重试耗尽），会在其他地方被 Cancel，这里会收到 ctx.Done()，所以不用担心死等
	maxWaitTime := 90 * time.Second

	// 快速检查一次状态
	conn.mu.Lock()
	status := conn.status
	conn.mu.Unlock()

	if status == StatusConnected && conn.client != nil && conn.client.API() != nil {
		return conn, nil
	}
	if status == StatusConnectionError {
		return nil, fmt.Errorf("connection error")
	}

	timer := time.NewTimer(maxWaitTime)
	defer timer.Stop()

	cp.logger.Info("Waiting for connection ready...",
		zap.String("account_id", accountID),
		zap.String("initial_status", status.String()))

	for {
		select {
		case <-conn.stateChangeCh:
			// 状态发生变更，检查新状态
			conn.mu.Lock()
			newStatus := conn.status
			conn.mu.Unlock()

			// cp.logger.Debug("Connection state changed",
			// 	zap.String("account_id", accountID),
			// 	zap.String("status", newStatus.String()))

			switch newStatus {
			case StatusConnected:
				// 连接成功，再次确保 client 和 API可用
				if conn.client != nil && conn.client.API() != nil {
					return conn, nil
				}
				// 理论上不应该发生 Connected 但 API 为 nil，除非初始化逻辑有 bug
				// 继续等待

			case StatusConnectionError:
				return nil, fmt.Errorf("connection error")

			case StatusConnecting, StatusReconnecting:
				// 继续等待
			}

		case <-conn.ctx.Done():
			// 当前连接上下文被取消，说明连接被替代（重连产生新连接）或被移除
			// 此时应该返回错误，让上层 ExecuteTask 的重试逻辑去获取新连接
			return nil, fmt.Errorf("connection replaced or canceled")

		case <-timer.C:
			return nil, fmt.Errorf("connection timeout after %v", maxWaitTime)
		}
	}
}

// GetConnectionStatus 获取连接状态
func (cp *ConnectionPool) GetConnectionStatus(accountID string) ConnectionStatus {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if conn, exists := cp.connections[accountID]; exists {
		return conn.status
	}
	return StatusDisconnected
}

// IsAccountBusy 检查账号是否忙碌
func (cp *ConnectionPool) IsAccountBusy(accountID string) bool {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	if conn, exists := cp.connections[accountID]; exists {
		conn.mu.Lock()
		defer conn.mu.Unlock()
		return conn.taskRunning
	}
	return false
}

// UpdateConfig 更新账号配置
func (cp *ConnectionPool) UpdateConfig(accountID string, config *ClientConfig) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	cp.configs[accountID] = config

	// 如果连接存在，标记需要重建
	if conn, exists := cp.connections[accountID]; exists {
		cp.logger.Info("Configuration updated, will recreate connection",
			zap.String("account_id", accountID))

		conn.cancel()
		delete(cp.connections, accountID)
	}
}

// RemoveConnection 移除连接
func (cp *ConnectionPool) RemoveConnection(accountID string) {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	if conn, exists := cp.connections[accountID]; exists {
		conn.logger.Info("Removing connection")
		conn.cancel()
		delete(cp.connections, accountID)
		// 确保更新在线状态为离线
		go cp.updateConnectionStatus(accountID, false)
	}

	delete(cp.configs, accountID)
	delete(cp.updateHandlers, accountID)
}

// SetUpdateHandler 设置账号的更新处理器
func (cp *ConnectionPool) SetUpdateHandler(accountID string, handler telegram.UpdateHandler) {
	cp.mu.Lock()
	defer cp.mu.Unlock()
	cp.updateHandlers[accountID] = handler
}

// createUpdateDispatcher 创建更新分发器
func (cp *ConnectionPool) createUpdateDispatcher(accountID string) telegram.UpdateHandler {
	return telegram.UpdateHandlerFunc(func(ctx context.Context, u tg.UpdatesClass) error {
		cp.mu.RLock()
		handler, exists := cp.updateHandlers[accountID]
		cp.mu.RUnlock()

		if exists && handler != nil {
			return handler.Handle(ctx, u)
		}
		return nil
	})
}

// cleanupLoop 清理循环
func (cp *ConnectionPool) cleanupLoop() {
	for range cp.cleanupTicker.C {
		cp.cleanupIdleConnections()
	}
}

// cleanupIdleConnections 定期清理空闲连接
func (cp *ConnectionPool) cleanupIdleConnections() {
	cp.mu.Lock()
	defer cp.mu.Unlock()

	now := time.Now()
	var toRemove []string

	for accountID, conn := range cp.connections {
		conn.mu.Lock()
		isIdle := !conn.taskRunning && now.Sub(conn.lastUsed) > cp.maxIdle
		conn.mu.Unlock()

		if isIdle {
			cp.logger.Info("Cleaning up idle connection",
				zap.String("account_id", accountID),
				zap.Duration("idle_time", now.Sub(conn.lastUsed)))

			conn.cancel()
			toRemove = append(toRemove, accountID)
		}
	}

	for _, accountID := range toRemove {
		delete(cp.connections, accountID)
		// 确保更新在线状态为离线
		go cp.updateConnectionStatus(accountID, false)
	}

	if len(toRemove) > 0 {
		cp.logger.Info("Cleaned up idle connections", zap.Int("count", len(toRemove)))
	}
}

// loadAccountConfig 动态加载账号配置
func (cp *ConnectionPool) loadAccountConfig(accountID string) (*ClientConfig, error) {
	// 转换accountID为uint64
	accountIDNum, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid account ID: %w", err)
	}

	// 从数据库获取账号信息
	account, err := cp.accountRepo.GetByID(accountIDNum)
	if err != nil {
		return nil, fmt.Errorf("failed to get account from database: %w", err)
	}

	// 检查账号状态
	if !account.IsAvailable() {
		return nil, fmt.Errorf("account is not available, status: %s", account.Status)
	}

	// 注意：不在这里解码 session 数据
	// session 数据的解码由 DatabaseSessionStorage.LoadSession 统一处理
	// 这里只传递原始的 base64 编码数据，避免双重解码问题
	cp.logger.Debug("Session data will be decoded by DatabaseSessionStorage",
		zap.String("account_id", accountID),
		zap.Int("session_data_len", len(account.SessionData)))

	// 构建配置 - SessionData 传 nil，让 DatabaseSessionStorage 从数据库加载并解码
	config := &ClientConfig{
		AppID:       cp.appID,
		AppHash:     cp.appHash,
		Phone:       account.Phone,
		SessionData: nil, // 不预加载，由 DatabaseSessionStorage 统一处理
	}

	// 如果账号绑定了代理，加载代理配置
	if account.ProxyID != nil && *account.ProxyID > 0 {
		proxy, err := cp.proxyRepo.GetByID(*account.ProxyID)
		if err != nil {
			cp.logger.Warn("Failed to load proxy configuration",
				zap.String("account_id", accountID),
				zap.Uint64("proxy_id", *account.ProxyID),
				zap.Error(err))
		} else if proxy != nil {
			config.ProxyConfig = &ProxyConfig{
				Protocol: string(proxy.Protocol),
				IP:       proxy.IP,
				Port:     proxy.Port,
				Username: proxy.Username,
				Password: proxy.Password,
			}
			cp.logger.Info("Proxy configuration loaded for account",
				zap.String("account_id", accountID),
				zap.Uint64("proxy_id", *account.ProxyID),
				zap.String("proxy_ip", proxy.IP),
				zap.Int("proxy_port", proxy.Port))
		}
	}

	// 缓存配置
	cp.mu.Lock()
	cp.configs[accountID] = config
	cp.mu.Unlock()

	cp.logger.Info("Account configuration loaded dynamically",
		zap.String("account_id", accountID),
		zap.String("phone", account.Phone))

	return config, nil
}

// GetStats 获取连接池统计信息
func (cp *ConnectionPool) GetStats() map[string]interface{} {
	cp.mu.RLock()
	defer cp.mu.RUnlock()

	stats := map[string]interface{}{
		"total_connections":     len(cp.connections),
		"active_connections":    0,
		"busy_connections":      0,
		"connections_by_status": make(map[string]int),
	}

	for _, conn := range cp.connections {
		conn.mu.Lock()
		if conn.isActive {
			stats["active_connections"] = stats["active_connections"].(int) + 1
		}
		if conn.taskRunning {
			stats["busy_connections"] = stats["busy_connections"].(int) + 1
		}

		statusStr := conn.status.String()
		if count, exists := stats["connections_by_status"].(map[string]int)[statusStr]; exists {
			stats["connections_by_status"].(map[string]int)[statusStr] = count + 1
		} else {
			stats["connections_by_status"].(map[string]int)[statusStr] = 1
		}
		conn.mu.Unlock()
	}

	return stats
}

// updateAccountInfoFromTelegram 从 Telegram 获取并更新账号信息
// ctx 参数是从 maintainConnection 的 Run 回调中传入的，确保使用同一个连接上下文
func (cp *ConnectionPool) updateAccountInfoFromTelegram(accountID string, conn *ManagedConnection, ctx context.Context) {
	// 转换accountID为uint64
	accountIDNum, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		cp.logger.Error("Invalid account ID for info update", zap.String("account_id", accountID), zap.Error(err))
		return
	}

	// 直接使用已建立的连接和 API 客户端，不再调用 Run()
	api := conn.client.API()

	// 获取当前用户信息
	users, err := api.UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUserSelf{}})
	if err != nil {
		cp.logger.Warn("Failed to get user info from Telegram",
			zap.String("account_id", accountID),
			zap.Error(err))
		// 检查是否是账号被禁用等严重错误，需要更新账号状态
		errorStr := strings.ToUpper(err.Error())
		if strings.Contains(errorStr, "USER_DEACTIVATED") ||
			strings.Contains(errorStr, "AUTH_KEY_UNREGISTERED") ||
			strings.Contains(errorStr, "PHONE_NUMBER_BANNED") ||
			strings.Contains(errorStr, "SESSION_REVOKED") {
			account, getErr := cp.accountRepo.GetByID(accountIDNum)
			if getErr == nil {
				account.Status = models.AccountStatusDead
				now := time.Now()
				account.LastCheckAt = &now
				if updateErr := cp.accountRepo.Update(account); updateErr != nil {
					cp.logger.Error("Failed to update account status to dead",
						zap.String("account_id", accountID),
						zap.Error(updateErr))
				} else {
					cp.logger.Info("Account marked as dead due to Telegram error",
						zap.String("account_id", accountID),
						zap.String("phone", account.Phone),
						zap.String("error_type", errorStr))
				}
			}
		}
		return
	}

	if len(users) == 0 {
		cp.logger.Warn("No user info returned from Telegram",
			zap.String("account_id", accountID))
		return
	}

	// 提取用户信息
	user, ok := users[0].(*tg.User)
	if !ok {
		cp.logger.Warn("Unexpected user type from Telegram",
			zap.String("account_id", accountID))
		return
	}

	// 准备更新数据
	info := &models.TelegramAccountInfo{}

	if user.ID != 0 {
		userID := int64(user.ID)
		info.TgUserID = &userID
	}

	if user.Phone != "" {
		info.Phone = &user.Phone
	}

	if user.Username != "" {
		info.Username = &user.Username
	}

	if user.FirstName != "" {
		info.FirstName = &user.FirstName
	}

	if user.LastName != "" {
		info.LastName = &user.LastName
	}

	// 获取完整用户信息（包括头像和简介）
	fullUser, err := api.UsersGetFullUser(ctx, &tg.InputUserSelf{})
	if err == nil {
		if fullUser.FullUser.About != "" {
			info.Bio = &fullUser.FullUser.About
		}

		// 获取头像URL（如果有）
		if user.Photo != nil {
			if photo, ok := user.Photo.(*tg.UserProfilePhoto); ok {
				// 这里可以构建头像URL或保存photo ID
				// 简单起见，我们保存photo ID
				photoID := fmt.Sprintf("%d", photo.PhotoID)
				info.PhotoURL = &photoID
			}
		}
	}

	// 更新到数据库
	account, err := cp.accountRepo.GetByID(accountIDNum)
	if err != nil {
		cp.logger.Error("Failed to get account from database",
			zap.String("account_id", accountID),
			zap.Error(err))
		return
	}

	// 验证账号 ID 匹配，防止更新错误的账号
	if account.ID != accountIDNum {
		cp.logger.Error("Account ID mismatch! This should never happen!",
			zap.String("expected_account_id", accountID),
			zap.Uint64("actual_account_id", account.ID))
		return
	}

	// 记录更新前的信息用于调试
	cp.logger.Info("Updating account info",
		zap.String("account_id", accountID),
		zap.String("phone", account.Phone),
		zap.Any("new_tg_user_id", info.TgUserID),
		zap.Any("new_username", info.Username),
		zap.Any("new_first_name", info.FirstName))

	// 更新字段
	if info.TgUserID != nil {
		account.TgUserID = info.TgUserID
	}
	if info.Phone != nil && *info.Phone != "" {
		account.Phone = *info.Phone
	}
	if info.Username != nil {
		account.Username = info.Username
	}
	if info.FirstName != nil {
		account.FirstName = info.FirstName
	}
	if info.LastName != nil {
		account.LastName = info.LastName
	}
	if info.Bio != nil {
		account.Bio = info.Bio
	}
	if info.PhotoURL != nil {
		account.PhotoURL = info.PhotoURL
	}

	// 保存到数据库
	if err := cp.accountRepo.Update(account); err != nil {
		cp.logger.Error("Failed to update account info to database",
			zap.String("account_id", accountID),
			zap.Error(err))
		return
	}

	cp.logger.Info("Account info updated from Telegram successfully",
		zap.String("account_id", accountID),
		zap.String("phone", account.Phone),
		zap.Any("tg_user_id", info.TgUserID),
		zap.Any("username", info.Username),
		zap.Any("first_name", info.FirstName))
}

// updateAccountStatusOnSuccess 连接或任务成功时更新账号状态
func (cp *ConnectionPool) updateAccountStatusOnSuccess(accountID string) {
	accountIDNum, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return
	}

	account, err := cp.accountRepo.GetByID(accountIDNum)
	if err != nil {
		return
	}

	// 如果账号状态是警告或新建，更新为正常
	if account.Status == models.AccountStatusWarning || account.Status == models.AccountStatusNew {
		account.Status = models.AccountStatusNormal
		now := time.Now()
		account.LastCheckAt = &now
		account.LastUsedAt = &now

		if err := cp.accountRepo.Update(account); err != nil {
			cp.logger.Error("Failed to update account status to normal",
				zap.String("account_id", accountID),
				zap.Error(err))
		} else {
			cp.logger.Info("Account status updated to normal",
				zap.String("account_id", accountID))
		}
	} else {
		// 只更新最后使用时间
		now := time.Now()
		account.LastUsedAt = &now
		account.LastCheckAt = &now
		cp.accountRepo.Update(account)
	}
}

// updateAccountStatusOnError 连接失败时更新账号状态
func (cp *ConnectionPool) updateAccountStatusOnError(accountID string, err error) {
	accountIDNum, parseErr := strconv.ParseUint(accountID, 10, 64)
	if parseErr != nil {
		return
	}

	account, getErr := cp.accountRepo.GetByID(accountIDNum)
	if getErr != nil {
		return
	}

	// 根据错误类型判断是否需要更新状态
	errorStr := strings.ToUpper(err.Error())

	// 检查是否是严重错误（账号被封禁等）
	if strings.Contains(errorStr, "AUTH_KEY_UNREGISTERED") ||
		strings.Contains(errorStr, "USER_DEACTIVATED") ||
		strings.Contains(errorStr, "PHONE_NUMBER_BANNED") {
		account.Status = models.AccountStatusDead
		cp.logger.Warn("Account marked as dead due to critical error",
			zap.String("account_id", accountID),
			zap.Error(err))
	} else if strings.Contains(errorStr, "FLOOD_WAIT") ||
		strings.Contains(errorStr, "SLOWMODE_WAIT") {
		// 触发限流，设置为冷却状态
		account.Status = models.AccountStatusCooling
		cp.logger.Warn("Account marked as cooling due to rate limit",
			zap.String("account_id", accountID),
			zap.Error(err))
	} else if account.Status == models.AccountStatusNormal || account.Status == models.AccountStatusNew {
		// 其他错误，设置为警告状态
		account.Status = models.AccountStatusWarning
		cp.logger.Warn("Account marked as warning due to error",
			zap.String("account_id", accountID),
			zap.Error(err))
	}

	now := time.Now()
	account.LastCheckAt = &now

	if updateErr := cp.accountRepo.Update(account); updateErr != nil {
		cp.logger.Error("Failed to update account status on error",
			zap.String("account_id", accountID),
			zap.Error(updateErr))
	}
}

// updateAccountStatusOnTaskError 任务执行失败时更新账号状态
func (cp *ConnectionPool) updateAccountStatusOnTaskError(accountID string, err error) {
	accountIDNum, parseErr := strconv.ParseUint(accountID, 10, 64)
	if parseErr != nil {
		return
	}

	account, getErr := cp.accountRepo.GetByID(accountIDNum)
	if getErr != nil {
		return
	}

	errorStr := strings.ToUpper(err.Error())

	// 检查是否是严重错误
	if strings.Contains(errorStr, "AUTH_KEY_UNREGISTERED") ||
		strings.Contains(errorStr, "USER_DEACTIVATED") ||
		strings.Contains(errorStr, "PHONE_NUMBER_BANNED") {
		account.Status = models.AccountStatusDead
		cp.logger.Warn("Account marked as dead due to task error",
			zap.String("account_id", accountID),
			zap.Error(err))
	} else if strings.Contains(errorStr, "FLOOD_WAIT") ||
		strings.Contains(errorStr, "SLOWMODE_WAIT") ||
		strings.Contains(errorStr, "PEER_FLOOD") {
		// 触发限流，设置为冷却状态
		account.Status = models.AccountStatusCooling
		cp.logger.Warn("Account marked as cooling due to task error",
			zap.String("account_id", accountID),
			zap.Error(err))
	} else if strings.Contains(errorStr, "CHAT_WRITE_FORBIDDEN") ||
		strings.Contains(errorStr, "USER_RESTRICTED") ||
		strings.Contains(errorStr, "CHAT_RESTRICTED") {
		account.Status = models.AccountStatusRestricted
		cp.logger.Warn("Account marked as restricted due to task error",
			zap.String("account_id", accountID),
			zap.Error(err))
	}
	// 其他错误不改变状态，可能是临时性问题

	now := time.Now()
	account.LastCheckAt = &now

	if updateErr := cp.accountRepo.Update(account); updateErr != nil {
		cp.logger.Error("Failed to update account status on task error",
			zap.String("account_id", accountID),
			zap.Error(updateErr))
	}
}

// updateConnectionStatus 更新账号在线状态
func (cp *ConnectionPool) updateConnectionStatus(accountID string, isOnline bool) {
	accountIDNum, err := strconv.ParseUint(accountID, 10, 64)
	if err != nil {
		return
	}

	if err := cp.accountRepo.UpdateConnectionStatus(accountIDNum, isOnline); err != nil {
		cp.logger.Error("Failed to update connection status",
			zap.String("account_id", accountID),
			zap.Bool("is_online", isOnline),
			zap.Error(err))
	}
}

// CheckConnection 主动检查账号连接状态
func (cp *ConnectionPool) CheckConnection(accountID uint64) error {
	// 1. 获取账号信息
	account, err := cp.accountRepo.GetByID(accountID)
	if err != nil {
		return err
	}

	// 2. 构建配置
	config := &ClientConfig{
		AppID:       cp.appID,
		AppHash:     cp.appHash,
		Phone:       account.Phone,
		SessionData: []byte(account.SessionData),
	}

	if account.ProxyID != nil {
		proxy, err := cp.proxyRepo.GetByID(*account.ProxyID)
		if err == nil && proxy.IsActive {
			config.ProxyConfig = &ProxyConfig{
				Protocol: string(proxy.Protocol),
				IP:       proxy.IP,
				Port:     proxy.Port,
				Username: proxy.Username,
				Password: proxy.Password,
			}
		}
	}

	// 3. 获取或创建连接
	conn, err := cp.GetOrCreateConnection(fmt.Sprintf("%d", accountID), config)
	if err != nil {
		return err
	}

	// 4. 等待连接就绪 (最多等待 15 秒)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("connection timeout")
		case <-ticker.C:
			conn.mu.Lock()
			status := conn.status
			conn.mu.Unlock()

			switch status {
			case StatusConnected:
				// 5. 验证会话有效性
				checkCtx, checkCancel := context.WithTimeout(context.Background(), 10*time.Second)
				defer checkCancel()

				// 获取当前用户信息来验证会话
				user, err := conn.client.Self(checkCtx)
				if err != nil {
					// 如果获取用户信息失败，可能是 session 失效
					cp.updateAccountStatusOnError(fmt.Sprintf("%d", accountID), err)
					return fmt.Errorf("session invalid: %w", err)
				}

				// 验证成功，更新状态
				if account.Status == models.AccountStatusWarning || account.Status == models.AccountStatusNew {
					account.Status = models.AccountStatusNormal
					now := time.Now()
					account.LastCheckAt = &now
					cp.accountRepo.Update(account)
				}

				// 确保在线状态为 true
				cp.updateConnectionStatus(fmt.Sprintf("%d", accountID), true)

				cp.logger.Info("Account check successful",
					zap.Uint64("account_id", accountID),
					zap.String("username", user.Username))
				return nil
			case StatusConnectionError:
				return fmt.Errorf("connection failed")
			}
		}
	}
}

// Close 关闭连接池
func (cp *ConnectionPool) Close() {
	cp.logger.Info("Closing connection pool")

	cp.cleanupTicker.Stop()

	cp.mu.Lock()
	defer cp.mu.Unlock()

	for accountID, conn := range cp.connections {
		cp.logger.Debug("Closing connection", zap.String("account_id", accountID))
		conn.cancel()
	}

	cp.connections = make(map[string]*ManagedConnection)
	cp.configs = make(map[string]*ClientConfig)
}
