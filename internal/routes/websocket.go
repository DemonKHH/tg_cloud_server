package routes

import (
	"encoding/json"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/common/utils"
	"tg_cloud_server/internal/services"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中应该检查Origin
		return true
	},
}

// WebSocketManager WebSocket连接管理器
type WebSocketManager struct {
	connections map[uint64]*WebSocketConnection // userID -> connection
	broadcast   chan *BroadcastMessage
	register    chan *WebSocketConnection
	unregister  chan *WebSocketConnection
	mutex       sync.RWMutex
	authService *services.AuthService
	logger      *zap.Logger
}

// WebSocketConnection WebSocket连接信息
type WebSocketConnection struct {
	UserID     uint64
	Conn       *websocket.Conn
	Send       chan []byte
	Manager    *WebSocketManager
	LastPing   time.Time
	Subscribed map[string]bool // 订阅的消息类型
}

// WebSocketMessage WebSocket消息结构
type WebSocketMessage struct {
	Type string      `json:"type"`
	Data interface{} `json:"data"`
}

// BroadcastMessage 广播消息
type BroadcastMessage struct {
	UserID   uint64      // 0表示广播给所有用户
	Type     string      `json:"type"`
	Data     interface{} `json:"data"`
	Channels []string    `json:"channels,omitempty"` // 指定频道
}

// AuthMessage 认证消息
type AuthMessage struct {
	Token string `json:"token"`
}

// SubscribeMessage 订阅消息
type SubscribeMessage struct {
	Channels []string `json:"channels"`
}

// NewWebSocketManager 创建WebSocket管理器
func NewWebSocketManager(authService *services.AuthService) *WebSocketManager {
	return &WebSocketManager{
		connections: make(map[uint64]*WebSocketConnection),
		broadcast:   make(chan *BroadcastMessage, 256),
		register:    make(chan *WebSocketConnection),
		unregister:  make(chan *WebSocketConnection),
		authService: authService,
		logger:      logger.Get().Named("websocket_manager"),
	}
}

// Run 运行WebSocket管理器
func (m *WebSocketManager) Run() {
	for {
		select {
		case conn := <-m.register:
			m.mutex.Lock()
			m.connections[conn.UserID] = conn
			m.mutex.Unlock()
			m.logger.Info("User connected", zap.Uint64("user_id", conn.UserID))

		case conn := <-m.unregister:
			m.mutex.Lock()
			if _, ok := m.connections[conn.UserID]; ok {
				delete(m.connections, conn.UserID)
				close(conn.Send)
			}
			m.mutex.Unlock()
			m.logger.Info("User disconnected", zap.Uint64("user_id", conn.UserID))

		case message := <-m.broadcast:
			m.broadcastMessage(message)
		}
	}
}

// broadcastMessage 广播消息
func (m *WebSocketManager) broadcastMessage(message *BroadcastMessage) {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	data, err := json.Marshal(WebSocketMessage{
		Type: message.Type,
		Data: message.Data,
	})
	if err != nil {
		m.logger.Error("Failed to marshal broadcast message", zap.Error(err))
		return
	}

	if message.UserID == 0 {
		// 广播给所有用户
		for userID, conn := range m.connections {
			if m.shouldSendToUser(conn, message.Channels) {
				select {
				case conn.Send <- data:
				default:
					m.logger.Warn("Failed to send message to user", zap.Uint64("user_id", userID))
				}
			}
		}
	} else {
		// 发送给特定用户
		if conn, ok := m.connections[message.UserID]; ok {
			if m.shouldSendToUser(conn, message.Channels) {
				select {
				case conn.Send <- data:
				default:
					m.logger.Warn("Failed to send message to user", zap.Uint64("user_id", message.UserID))
				}
			}
		}
	}
}

// shouldSendToUser 检查是否应该发送消息给用户
func (m *WebSocketManager) shouldSendToUser(conn *WebSocketConnection, channels []string) bool {
	if len(channels) == 0 {
		return true // 无频道限制
	}

	for _, channel := range channels {
		if conn.Subscribed[channel] {
			return true
		}
	}
	return false
}

// SendToUser 发送消息给特定用户
func (m *WebSocketManager) SendToUser(userID uint64, msgType string, data interface{}) {
	message := &BroadcastMessage{
		UserID: userID,
		Type:   msgType,
		Data:   data,
	}

	select {
	case m.broadcast <- message:
	default:
		m.logger.Warn("Broadcast channel full, dropping message", zap.Uint64("user_id", userID))
	}
}

// Broadcast 广播消息给所有用户
func (m *WebSocketManager) Broadcast(msgType string, data interface{}) {
	message := &BroadcastMessage{
		UserID: 0,
		Type:   msgType,
		Data:   data,
	}

	select {
	case m.broadcast <- message:
	default:
		m.logger.Warn("Broadcast channel full, dropping message")
	}
}

// wsManager 全局WebSocket管理器
var wsManager *WebSocketManager

// RegisterWebSocketRoutes 注册WebSocket路由
func RegisterWebSocketRoutes(router *gin.Engine, redisClient *redis.Client, authService *services.AuthService, notificationService services.NotificationService) {
	log := logger.Get().Named("websocket")

	// 初始化WebSocket管理器
	wsManager = NewWebSocketManager(authService)
	go wsManager.Run()

	// WebSocket连接端点 (旧版本，保持兼容)
	router.GET("/ws", func(c *gin.Context) {
		// 升级HTTP连接到WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Error("Failed to upgrade to websocket", zap.Error(err))
			return
		}

		log.Info("WebSocket connection established",
			zap.String("remote_addr", conn.RemoteAddr().String()))

		// 处理WebSocket连接
		handleWebSocketConnection(conn, wsManager, log)
	})

	// NotificationService WebSocket 端点 (支持任务日志订阅)
	router.GET("/api/v1/ws", func(c *gin.Context) {
		// 从查询参数获取 token
		token := c.Query("token")
		if token == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Missing token"})
			return
		}

		// 验证 token
		userID, err := authService.VerifyToken(token)
		if err != nil {
			log.Warn("Invalid token for WebSocket connection", zap.Error(err))
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			return
		}

		// 升级HTTP连接到WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Error("Failed to upgrade to websocket", zap.Error(err))
			return
		}

		log.Info("NotificationService WebSocket connection established",
			zap.Uint64("user_id", userID),
			zap.String("remote_addr", conn.RemoteAddr().String()))

		// 注册到 NotificationService
		notificationService.RegisterWSConnection(userID, conn)
	})

	// WebSocket状态端点
	router.GET("/ws/status", func(c *gin.Context) {
		wsManager.mutex.RLock()
		connectionCount := len(wsManager.connections)
		wsManager.mutex.RUnlock()

		c.JSON(http.StatusOK, gin.H{
			"status":             "available",
			"endpoint":           "/ws",
			"description":        "WebSocket connection for real-time updates",
			"active_connections": connectionCount,
			"supported_channels": []string{
				"task_updates",
				"account_status",
				"system_notifications",
				"user_activity",
			},
		})
	})

	// 管理员广播端点
	router.POST("/ws/broadcast", func(c *gin.Context) {
		// 需要管理员权限
		userID, err := utils.GetUserID(c)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "未授权"})
			return
		}

		// TODO: 检查管理员权限
		_ = userID

		var req struct {
			Type     string      `json:"type" binding:"required"`
			Data     interface{} `json:"data" binding:"required"`
			Channels []string    `json:"channels"`
		}

		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		// 发送广播消息
		wsManager.Broadcast(req.Type, req.Data)

		c.JSON(http.StatusOK, gin.H{
			"message": "Broadcast sent successfully",
		})
	})
}

// handleWebSocketConnection 处理WebSocket连接
func handleWebSocketConnection(conn *websocket.Conn, manager *WebSocketManager, log *zap.Logger) {
	// 创建WebSocket连接对象
	wsConn := &WebSocketConnection{
		Conn:       conn,
		Send:       make(chan []byte, 256),
		Manager:    manager,
		LastPing:   time.Now(),
		Subscribed: make(map[string]bool),
	}

	// 设置读取限制和超时
	conn.SetReadLimit(512)
	conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	conn.SetWriteDeadline(time.Now().Add(10 * time.Second))

	// 心跳检测
	conn.SetPongHandler(func(string) error {
		conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		wsConn.LastPing = time.Now()
		return nil
	})

	// 启动写协程
	go wsConn.writePump()

	// 等待用户认证（30秒超时）
	authenticated := make(chan bool, 1)
	go func() {
		time.Sleep(30 * time.Second)
		authenticated <- false
	}()

	// 发送认证请求
	authRequest := WebSocketMessage{
		Type: "auth_required",
		Data: map[string]interface{}{
			"message": "Please authenticate within 30 seconds",
		},
	}
	if err := conn.WriteJSON(authRequest); err != nil {
		log.Error("Failed to send auth request", zap.Error(err))
		return
	}

	// 处理认证过程
	go func() {
		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				authenticated <- false
				return
			}

			if messageType == websocket.TextMessage {
				var msg WebSocketMessage
				if err := json.Unmarshal(message, &msg); err != nil {
					continue
				}

				if msg.Type == "auth" {
					if wsConn.handleAuthentication(msg.Data) {
						authenticated <- true
						return
					}
				}
			}
		}
	}()

	// 等待认证结果
	if !<-authenticated {
		log.Warn("WebSocket authentication timeout or failed")
		conn.WriteJSON(WebSocketMessage{
			Type: "auth_failed",
			Data: map[string]interface{}{
				"message": "Authentication failed or timeout",
			},
		})
		conn.Close()
		return
	}

	// 认证成功，注册连接
	manager.register <- wsConn

	// 发送认证成功消息
	conn.WriteJSON(WebSocketMessage{
		Type: "auth_success",
		Data: map[string]interface{}{
			"message": "Authentication successful",
			"user_id": wsConn.UserID,
		},
	})

	// 启动读协程
	wsConn.readPump()

	// 连接关闭时清理
	manager.unregister <- wsConn
}

// handleAuthentication 处理用户认证
func (c *WebSocketConnection) handleAuthentication(data interface{}) bool {
	authData, ok := data.(map[string]interface{})
	if !ok {
		return false
	}

	tokenInterface, exists := authData["token"]
	if !exists {
		return false
	}

	token, ok := tokenInterface.(string)
	if !ok {
		return false
	}

	// 验证JWT token
	userID, err := c.Manager.authService.VerifyToken(token)
	if err != nil {
		c.Manager.logger.Warn("Invalid token", zap.Error(err))
		return false
	}

	c.UserID = userID
	return true
}

// readPump 处理读取消息
func (c *WebSocketConnection) readPump() {
	defer func() {
		c.Manager.unregister <- c
		c.Conn.Close()
	}()

	for {
		messageType, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.Manager.logger.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		c.Manager.logger.Debug("Received WebSocket message",
			zap.Uint64("user_id", c.UserID),
			zap.Int("message_type", messageType),
			zap.String("message", string(message)))

		// 处理不同类型的消息
		switch messageType {
		case websocket.TextMessage:
			c.handleTextMessage(message)
		case websocket.BinaryMessage:
			c.handleBinaryMessage(message)
		case websocket.PingMessage:
			// 响应Ping
			if err := c.Conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				c.Manager.logger.Error("Failed to send pong", zap.Error(err))
				return
			}
		}
	}
}

// writePump 处理发送消息
func (c *WebSocketConnection) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				c.Manager.logger.Error("Failed to write message", zap.Error(err))
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleTextMessage 处理文本消息
func (c *WebSocketConnection) handleTextMessage(message []byte) {
	var msg WebSocketMessage
	if err := json.Unmarshal(message, &msg); err != nil {
		c.Manager.logger.Error("Failed to unmarshal message", zap.Error(err))
		return
	}

	switch msg.Type {
	case "subscribe":
		c.handleSubscribe(msg.Data)
	case "unsubscribe":
		c.handleUnsubscribe(msg.Data)
	case "ping":
		c.sendMessage("pong", map[string]interface{}{
			"timestamp": time.Now().Unix(),
		})
	default:
		c.Manager.logger.Debug("Unknown message type", zap.String("type", msg.Type))
	}
}

// handleBinaryMessage 处理二进制消息
func (c *WebSocketConnection) handleBinaryMessage(message []byte) {
	c.Manager.logger.Debug("Received binary message", zap.Int("size", len(message)))
	// 暂时不处理二进制消息
}

// handleSubscribe 处理订阅请求
func (c *WebSocketConnection) handleSubscribe(data interface{}) {
	subData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	channelsInterface, exists := subData["channels"]
	if !exists {
		return
	}

	channels, ok := channelsInterface.([]interface{})
	if !ok {
		return
	}

	for _, channelInterface := range channels {
		if channel, ok := channelInterface.(string); ok {
			c.Subscribed[channel] = true
			c.Manager.logger.Info("User subscribed to channel",
				zap.Uint64("user_id", c.UserID),
				zap.String("channel", channel))
		}
	}

	c.sendMessage("subscribed", map[string]interface{}{
		"channels": channels,
	})
}

// handleUnsubscribe 处理取消订阅请求
func (c *WebSocketConnection) handleUnsubscribe(data interface{}) {
	subData, ok := data.(map[string]interface{})
	if !ok {
		return
	}

	channelsInterface, exists := subData["channels"]
	if !exists {
		return
	}

	channels, ok := channelsInterface.([]interface{})
	if !ok {
		return
	}

	for _, channelInterface := range channels {
		if channel, ok := channelInterface.(string); ok {
			delete(c.Subscribed, channel)
			c.Manager.logger.Info("User unsubscribed from channel",
				zap.Uint64("user_id", c.UserID),
				zap.String("channel", channel))
		}
	}

	c.sendMessage("unsubscribed", map[string]interface{}{
		"channels": channels,
	})
}

// sendMessage 发送消息给客户端
func (c *WebSocketConnection) sendMessage(msgType string, data interface{}) {
	msg := WebSocketMessage{
		Type: msgType,
		Data: data,
	}

	msgBytes, err := json.Marshal(msg)
	if err != nil {
		c.Manager.logger.Error("Failed to marshal message", zap.Error(err))
		return
	}

	select {
	case c.Send <- msgBytes:
	default:
		c.Manager.logger.Warn("Send channel full, dropping message", zap.Uint64("user_id", c.UserID))
	}
}

// GetWebSocketManager 获取全局WebSocket管理器
func GetWebSocketManager() *WebSocketManager {
	return wsManager
}
