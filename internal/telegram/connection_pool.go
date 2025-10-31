package telegram

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/gotd/td/telegram"
	"github.com/gotd/td/telegram/dcs"
	"github.com/gotd/td/tg"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/repository"
)

// ConnectionStatus 连接状态枚举
type ConnectionStatus int

const (
	StatusDisconnected ConnectionStatus = iota
	StatusConnecting
	StatusConnected
	StatusReconnecting
	StatusError
)

// String 返回状态字符串
func (s ConnectionStatus) String() string {
	switch s {
	case StatusDisconnected:
		return "disconnected"
	case StatusConnecting:
		return "connecting"
	case StatusConnected:
		return "connected"
	case StatusReconnecting:
		return "reconnecting"
	case StatusError:
		return "error"
	default:
		return "unknown"
	}
}

// ManagedConnection 托管连接封装
type ManagedConnection struct {
	client      *telegram.Client
	config      *ClientConfig
	status      ConnectionStatus
	lastUsed    time.Time
	useCount    int64
	isActive    bool
	taskRunning bool
	mu          sync.Mutex
	ctx         context.Context
	cancel      context.CancelFunc
	logger      *zap.Logger
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
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

// ConnectionPool 统一连接池管理器
type ConnectionPool struct {
	connections   map[string]*ManagedConnection
	configs       map[string]*ClientConfig
	mu            sync.RWMutex
	maxIdle       time.Duration
	cleanupTicker *time.Ticker
	logger        *zap.Logger
	appID         int
	appHash       string
	accountRepo   repository.AccountRepository
}

// NewConnectionPool 创建新的连接池
func NewConnectionPool(appID int, appHash string, maxIdle time.Duration, accountRepo repository.AccountRepository) *ConnectionPool {
	cp := &ConnectionPool{
		connections: make(map[string]*ManagedConnection),
		configs:     make(map[string]*ClientConfig),
		maxIdle:     maxIdle,
		logger:      logger.Get().Named("connection_pool"),
		appID:       appID,
		appHash:     appHash,
		accountRepo: accountRepo,
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

	// 检查是否已存在连接
	if conn, exists := cp.connections[accountID]; exists {
		if conn.isActive && conn.status == StatusConnected {
			conn.lastUsed = time.Now()
			conn.useCount++
			cp.logger.Debug("Reusing existing connection",
				zap.String("account_id", accountID),
				zap.Int64("use_count", conn.useCount))
			return conn, nil
		}
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
			zap.String("proxy", fmt.Sprintf("%s://%s:%d", config.ProxyConfig.Protocol, config.ProxyConfig.Host, config.ProxyConfig.Port)))

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
		client:   client,
		config:   config,
		status:   StatusConnecting,
		lastUsed: time.Now(),
		isActive: true,
		ctx:      ctx,
		cancel:   cancel,
		logger:   cp.logger.Named(accountID),
	}

	// 异步建立连接
	go cp.maintainConnection(accountID, conn)

	cp.connections[accountID] = conn
	cp.configs[accountID] = config

	return conn, nil
}

// maintainConnection 维护连接状态
func (cp *ConnectionPool) maintainConnection(accountID string, conn *ManagedConnection) {
	conn.logger.Info("Starting connection maintenance")

	err := conn.client.Run(conn.ctx, func(ctx context.Context) error {
		conn.mu.Lock()
		conn.status = StatusConnected
		conn.mu.Unlock()

		conn.logger.Info("Connection established successfully")

		// 保持连接直到取消
		<-ctx.Done()
		return ctx.Err()
	})

	if err != nil && err != context.Canceled {
		conn.logger.Error("Connection error", zap.Error(err))

		conn.mu.Lock()
		conn.status = StatusError
		conn.mu.Unlock()

		// 自动重连逻辑
		cp.scheduleReconnect(accountID, conn)
	}
}

// scheduleReconnect 调度重连
func (cp *ConnectionPool) scheduleReconnect(accountID string, conn *ManagedConnection) {
	conn.logger.Info("Scheduling reconnection")

	// 等待一段时间后重连
	time.AfterFunc(30*time.Second, func() {
		cp.mu.Lock()
		defer cp.mu.Unlock()

		// 检查连接是否仍然存在且需要重连
		if currentConn, exists := cp.connections[accountID]; exists && currentConn == conn {
			if config, configExists := cp.configs[accountID]; configExists {
				conn.logger.Info("Attempting to reconnect")
				cp.createNewConnection(accountID, config)
			}
		}
	})
}

// ExecuteTask 执行任务 (复用连接)
func (cp *ConnectionPool) ExecuteTask(accountID string, task TaskInterface) error {
	config, exists := cp.configs[accountID]
	if !exists {
		return fmt.Errorf("no configuration found for account %s", accountID)
	}

	conn, err := cp.GetOrCreateConnection(accountID, config)
	if err != nil {
		return fmt.Errorf("failed to get connection: %w", err)
	}

	// 确保单任务执行
	conn.mu.Lock()
	if conn.taskRunning {
		conn.mu.Unlock()
		return errors.New("account is busy with another task")
	}
	conn.taskRunning = true
	conn.mu.Unlock()

	defer func() {
		conn.mu.Lock()
		conn.taskRunning = false
		conn.lastUsed = time.Now()
		conn.mu.Unlock()
	}()

	// 检查连接状态
	if conn.status != StatusConnected {
		return fmt.Errorf("connection not ready, status: %s", conn.status.String())
	}

	// 直接使用已建立的连接执行任务
	conn.logger.Debug("Executing task", zap.String("task_type", task.GetType()))

	return conn.client.Run(context.Background(), func(ctx context.Context) error {
		api := conn.client.API()
		return task.Execute(ctx, api)
	})
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
	}

	delete(cp.configs, accountID)
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
	}

	if len(toRemove) > 0 {
		cp.logger.Info("Cleaned up idle connections", zap.Int("count", len(toRemove)))
	}
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

// TaskInterface 任务接口
type TaskInterface interface {
	Execute(ctx context.Context, api *tg.Client) error
	GetType() string
}
