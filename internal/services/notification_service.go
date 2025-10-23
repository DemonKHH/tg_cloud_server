package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
	"tg_cloud_server/internal/events"
	"tg_cloud_server/internal/models"
)

// NotificationType 通知类型
type NotificationType string

const (
	NotificationTypeTaskUpdate    NotificationType = "task_update"
	NotificationTypeAccountStatus NotificationType = "account_status"
	NotificationTypeSystemAlert   NotificationType = "system_alert"
	NotificationTypeUserMessage   NotificationType = "user_message"
	NotificationTypeModuleUpdate  NotificationType = "module_update"
	NotificationTypeProxyStatus   NotificationType = "proxy_status"
	NotificationTypeRealTimeStats NotificationType = "realtime_stats"
)

// NotificationPriority 通知优先级
type NotificationPriority string

const (
	PriorityLow      NotificationPriority = "low"
	PriorityNormal   NotificationPriority = "normal"
	PriorityHigh     NotificationPriority = "high"
	PriorityCritical NotificationPriority = "critical"
)

// Notification 通知消息
type Notification struct {
	ID        string                 `json:"id"`
	Type      NotificationType       `json:"type"`
	Priority  NotificationPriority   `json:"priority"`
	Title     string                 `json:"title"`
	Message   string                 `json:"message"`
	Data      map[string]interface{} `json:"data,omitempty"`
	UserID    uint64                 `json:"user_id"`
	CreatedAt time.Time              `json:"created_at"`
	ReadAt    *time.Time             `json:"read_at,omitempty"`
}

// WSMessage WebSocket消息
type WSMessage struct {
	Type      string      `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// WSConnection WebSocket连接
type WSConnection struct {
	UserID     uint64
	Conn       *websocket.Conn
	Send       chan WSMessage
	Hub        *WSHub
	LastActive time.Time
}

// WSHub WebSocket集线器
type WSHub struct {
	clients    map[uint64]*WSConnection
	broadcast  chan WSMessage
	register   chan *WSConnection
	unregister chan *WSConnection
	mutex      sync.RWMutex
	logger     *zap.Logger
}

// NotificationService 通知服务接口
type NotificationService interface {
	// WebSocket管理
	RegisterWSConnection(userID uint64, conn *websocket.Conn) *WSConnection
	UnregisterWSConnection(userID uint64)
	GetActiveConnections() map[uint64]*WSConnection
	IsUserOnline(userID uint64) bool

	// 通知发送
	SendToUser(userID uint64, notification *Notification) error
	SendToUsers(userIDs []uint64, notification *Notification) error
	SendToAll(notification *Notification) error
	BroadcastMessage(msgType string, data interface{}) error

	// 任务相关通知
	NotifyTaskStatusChange(userID uint64, task *models.Task, oldStatus, newStatus string) error
	NotifyTaskProgress(userID uint64, taskID uint64, progress int, message string) error
	NotifyTaskCompleted(userID uint64, task *models.Task) error
	NotifyTaskFailed(userID uint64, task *models.Task, reason string) error

	// 账号相关通知
	NotifyAccountStatusChange(userID uint64, account *models.TGAccount, oldStatus, newStatus string) error
	NotifyAccountError(userID uint64, accountID uint64, error string) error
	NotifyProxyStatusChange(userID uint64, proxyID uint64, status string) error

	// 系统通知
	NotifySystemAlert(userID uint64, level string, message string) error
	NotifySystemMaintenance(message string, scheduledAt time.Time) error
	NotifyRateLimitExceeded(userID uint64) error

	// 实时数据推送
	PushRealTimeStats(userID uint64, stats map[string]interface{}) error
	PushTaskQueueUpdate(userID uint64, accountID uint64, queueInfo *models.QueueInfo) error

	// 消息管理
	GetUnreadNotifications(userID uint64) ([]*Notification, error)
	MarkNotificationAsRead(userID uint64, notificationID string) error
	MarkAllAsRead(userID uint64) error
	CleanupOldNotifications(olderThan time.Duration) error

	// 事件处理
	HandleEvent(ctx context.Context, event *events.Event) error

	// 启动和停止
	Start() error
	Stop() error
}

// notificationService 通知服务实现
type notificationService struct {
	hub                *WSHub
	eventService       *events.EventService
	logger             *zap.Logger
	notifications      map[string]*Notification // 内存存储通知，实际应该用数据库
	notificationsMutex sync.RWMutex
	running            bool
}

// NewNotificationService 创建通知服务
func NewNotificationService(eventService *events.EventService) NotificationService {
	service := &notificationService{
		eventService:  eventService,
		logger:        logger.Get().Named("notification_service"),
		notifications: make(map[string]*Notification),
		running:       false,
	}

	// 创建WebSocket集线器
	service.hub = &WSHub{
		clients:    make(map[uint64]*WSConnection),
		broadcast:  make(chan WSMessage, 256),
		register:   make(chan *WSConnection),
		unregister: make(chan *WSConnection),
		logger:     service.logger.Named("ws_hub"),
	}

	return service
}

// Start 启动服务
func (s *notificationService) Start() error {
	if s.running {
		return nil
	}

	s.logger.Info("Starting notification service")
	s.running = true

	// 启动WebSocket集线器
	go s.hub.run()

	// 订阅相关事件
	s.subscribeToEvents()

	s.logger.Info("Notification service started successfully")
	return nil
}

// Stop 停止服务
func (s *notificationService) Stop() error {
	if !s.running {
		return nil
	}

	s.logger.Info("Stopping notification service")
	s.running = false

	// 关闭所有WebSocket连接
	s.hub.mutex.Lock()
	for _, client := range s.hub.clients {
		close(client.Send)
		client.Conn.Close()
	}
	s.hub.clients = make(map[uint64]*WSConnection)
	s.hub.mutex.Unlock()

	s.logger.Info("Notification service stopped")
	return nil
}

// RegisterWSConnection 注册WebSocket连接
func (s *notificationService) RegisterWSConnection(userID uint64, conn *websocket.Conn) *WSConnection {
	client := &WSConnection{
		UserID:     userID,
		Conn:       conn,
		Send:       make(chan WSMessage, 256),
		Hub:        s.hub,
		LastActive: time.Now(),
	}

	s.hub.register <- client

	// 启动连接处理协程
	go s.handleWSConnection(client)

	s.logger.Info("WebSocket connection registered", zap.Uint64("user_id", userID))
	return client
}

// UnregisterWSConnection 注销WebSocket连接
func (s *notificationService) UnregisterWSConnection(userID uint64) {
	s.hub.mutex.Lock()
	if client, exists := s.hub.clients[userID]; exists {
		s.hub.unregister <- client
	}
	s.hub.mutex.Unlock()
}

// SendToUser 发送通知给指定用户
func (s *notificationService) SendToUser(userID uint64, notification *Notification) error {
	s.logger.Debug("Sending notification to user",
		zap.Uint64("user_id", userID),
		zap.String("type", string(notification.Type)),
		zap.String("title", notification.Title))

	// 存储通知
	s.storeNotification(notification)

	// 如果用户在线，通过WebSocket发送
	if s.IsUserOnline(userID) {
		message := WSMessage{
			Type:      "notification",
			Data:      notification,
			Timestamp: time.Now(),
		}

		s.hub.mutex.RLock()
		if client, exists := s.hub.clients[userID]; exists {
			select {
			case client.Send <- message:
			case <-time.After(time.Second):
				s.logger.Warn("Failed to send notification: timeout", zap.Uint64("user_id", userID))
			}
		}
		s.hub.mutex.RUnlock()
	}

	return nil
}

// SendToUsers 发送通知给多个用户
func (s *notificationService) SendToUsers(userIDs []uint64, notification *Notification) error {
	for _, userID := range userIDs {
		notification.UserID = userID
		if err := s.SendToUser(userID, notification); err != nil {
			s.logger.Error("Failed to send notification to user",
				zap.Uint64("user_id", userID),
				zap.Error(err))
		}
	}
	return nil
}

// SendToAll 发送通知给所有在线用户
func (s *notificationService) SendToAll(notification *Notification) error {
	message := WSMessage{
		Type:      "broadcast",
		Data:      notification,
		Timestamp: time.Now(),
	}

	s.hub.broadcast <- message
	return nil
}

// BroadcastMessage 广播消息
func (s *notificationService) BroadcastMessage(msgType string, data interface{}) error {
	message := WSMessage{
		Type:      msgType,
		Data:      data,
		Timestamp: time.Now(),
	}

	s.hub.broadcast <- message
	return nil
}

// NotifyTaskStatusChange 通知任务状态变更
func (s *notificationService) NotifyTaskStatusChange(userID uint64, task *models.Task, oldStatus, newStatus string) error {
	var priority NotificationPriority
	var title, message string

	switch newStatus {
	case string(models.TaskStatusRunning):
		priority = PriorityNormal
		title = "任务开始执行"
		message = fmt.Sprintf("任务 #%d 开始执行", task.ID)
	case string(models.TaskStatusCompleted):
		priority = PriorityNormal
		title = "任务执行完成"
		message = fmt.Sprintf("任务 #%d 执行完成", task.ID)
	case string(models.TaskStatusFailed):
		priority = PriorityHigh
		title = "任务执行失败"
		message = fmt.Sprintf("任务 #%d 执行失败", task.ID)
	case string(models.TaskStatusCancelled):
		priority = PriorityNormal
		title = "任务已取消"
		message = fmt.Sprintf("任务 #%d 已取消", task.ID)
	default:
		priority = PriorityLow
		title = "任务状态更新"
		message = fmt.Sprintf("任务 #%d 状态从 %s 变更为 %s", task.ID, oldStatus, newStatus)
	}

	notification := &Notification{
		ID:       s.generateNotificationID(),
		Type:     NotificationTypeTaskUpdate,
		Priority: priority,
		Title:    title,
		Message:  message,
		Data: map[string]interface{}{
			"task_id":    task.ID,
			"task_type":  task.TaskType,
			"old_status": oldStatus,
			"new_status": newStatus,
			"account_id": task.AccountID,
		},
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	return s.SendToUser(userID, notification)
}

// NotifyTaskProgress 通知任务进度
func (s *notificationService) NotifyTaskProgress(userID uint64, taskID uint64, progress int, message string) error {
	wsMsg := WSMessage{
		Type: "task_progress",
		Data: map[string]interface{}{
			"task_id":  taskID,
			"progress": progress,
			"message":  message,
		},
		Timestamp: time.Now(),
	}

	s.hub.mutex.RLock()
	if client, exists := s.hub.clients[userID]; exists {
		select {
		case client.Send <- wsMsg:
		default:
		}
	}
	s.hub.mutex.RUnlock()

	return nil
}

// NotifyAccountStatusChange 通知账号状态变更
func (s *notificationService) NotifyAccountStatusChange(userID uint64, account *models.TGAccount, oldStatus, newStatus string) error {
	var priority NotificationPriority
	switch newStatus {
	case "dead", "blocked", "restricted":
		priority = PriorityHigh
	case "warning":
		priority = PriorityNormal
	default:
		priority = PriorityLow
	}

	notification := &Notification{
		ID:       s.generateNotificationID(),
		Type:     NotificationTypeAccountStatus,
		Priority: priority,
		Title:    "账号状态变更",
		Message:  fmt.Sprintf("账号 %s 状态从 %s 变更为 %s", account.Phone, oldStatus, newStatus),
		Data: map[string]interface{}{
			"account_id": account.ID,
			"phone":      account.Phone,
			"old_status": oldStatus,
			"new_status": newStatus,
		},
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	return s.SendToUser(userID, notification)
}

// NotifySystemAlert 发送系统告警
func (s *notificationService) NotifySystemAlert(userID uint64, level string, message string) error {
	var priority NotificationPriority
	switch level {
	case "critical":
		priority = PriorityCritical
	case "error":
		priority = PriorityHigh
	case "warning":
		priority = PriorityNormal
	default:
		priority = PriorityLow
	}

	notification := &Notification{
		ID:       s.generateNotificationID(),
		Type:     NotificationTypeSystemAlert,
		Priority: priority,
		Title:    "系统告警",
		Message:  message,
		Data: map[string]interface{}{
			"level": level,
		},
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	return s.SendToUser(userID, notification)
}

// PushRealTimeStats 推送实时统计数据
func (s *notificationService) PushRealTimeStats(userID uint64, stats map[string]interface{}) error {
	message := WSMessage{
		Type:      "realtime_stats",
		Data:      stats,
		Timestamp: time.Now(),
	}

	s.hub.mutex.RLock()
	if client, exists := s.hub.clients[userID]; exists {
		select {
		case client.Send <- message:
		default:
		}
	}
	s.hub.mutex.RUnlock()

	return nil
}

// WebSocket集线器运行逻辑
func (hub *WSHub) run() {
	for {
		select {
		case client := <-hub.register:
			hub.mutex.Lock()
			hub.clients[client.UserID] = client
			hub.mutex.Unlock()
			hub.logger.Info("Client registered", zap.Uint64("user_id", client.UserID))

		case client := <-hub.unregister:
			hub.mutex.Lock()
			if _, ok := hub.clients[client.UserID]; ok {
				delete(hub.clients, client.UserID)
				close(client.Send)
			}
			hub.mutex.Unlock()
			hub.logger.Info("Client unregistered", zap.Uint64("user_id", client.UserID))

		case message := <-hub.broadcast:
			hub.mutex.RLock()
			for userID, client := range hub.clients {
				select {
				case client.Send <- message:
				default:
					delete(hub.clients, userID)
					close(client.Send)
				}
			}
			hub.mutex.RUnlock()
		}
	}
}

// handleWSConnection 处理WebSocket连接
func (s *notificationService) handleWSConnection(client *WSConnection) {
	defer func() {
		s.hub.unregister <- client
		client.Conn.Close()
	}()

	// 启动发送协程
	go s.handleWSSend(client)

	// 设置读取参数
	client.Conn.SetReadLimit(512)
	client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	client.Conn.SetPongHandler(func(string) error {
		client.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		client.LastActive = time.Now()
		return nil
	})

	// 读取消息循环
	for {
		_, message, err := client.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				s.logger.Error("WebSocket error", zap.Error(err))
			}
			break
		}

		client.LastActive = time.Now()
		s.handleWSMessage(client, message)
	}
}

// handleWSSend 处理WebSocket发送
func (s *notificationService) handleWSSend(client *WSConnection) {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		client.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-client.Send:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				client.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := client.Conn.WriteJSON(message); err != nil {
				s.logger.Error("Failed to write WebSocket message", zap.Error(err))
				return
			}

		case <-ticker.C:
			client.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := client.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleWSMessage 处理WebSocket消息
func (s *notificationService) handleWSMessage(client *WSConnection, message []byte) {
	var msg map[string]interface{}
	if err := json.Unmarshal(message, &msg); err != nil {
		s.logger.Error("Failed to unmarshal WebSocket message", zap.Error(err))
		return
	}

	msgType, ok := msg["type"].(string)
	if !ok {
		return
	}

	switch msgType {
	case "ping":
		// 响应ping
		response := WSMessage{
			Type:      "pong",
			Timestamp: time.Now(),
		}
		client.Send <- response

	case "subscribe":
		// 处理订阅请求
		s.handleSubscribe(client, msg)

	case "unsubscribe":
		// 处理取消订阅请求
		s.handleUnsubscribe(client, msg)

	case "mark_read":
		// 标记通知为已读
		if notificationID, ok := msg["notification_id"].(string); ok {
			s.MarkNotificationAsRead(client.UserID, notificationID)
		}
	}
}

// 辅助方法

func (s *notificationService) IsUserOnline(userID uint64) bool {
	s.hub.mutex.RLock()
	_, exists := s.hub.clients[userID]
	s.hub.mutex.RUnlock()
	return exists
}

func (s *notificationService) GetActiveConnections() map[uint64]*WSConnection {
	s.hub.mutex.RLock()
	connections := make(map[uint64]*WSConnection)
	for userID, client := range s.hub.clients {
		connections[userID] = client
	}
	s.hub.mutex.RUnlock()
	return connections
}

func (s *notificationService) storeNotification(notification *Notification) {
	s.notificationsMutex.Lock()
	s.notifications[notification.ID] = notification
	s.notificationsMutex.Unlock()
}

func (s *notificationService) generateNotificationID() string {
	return fmt.Sprintf("notif_%d_%d", time.Now().UnixNano(), time.Now().UnixNano()%1000)
}

func (s *notificationService) subscribeToEvents() {
	// 订阅任务相关事件
	taskEvents := []events.EventType{
		events.EventTaskStarted,
		events.EventTaskCompleted,
		events.EventTaskFailed,
		events.EventTaskCancelled,
	}

	for _, eventType := range taskEvents {
		s.eventService.Subscribe(eventType, s)
	}

	// 订阅账号相关事件
	accountEvents := []events.EventType{
		events.EventAccountStatusChanged,
		events.EventAccountCreated,
		events.EventAccountDeleted,
	}

	for _, eventType := range accountEvents {
		s.eventService.Subscribe(eventType, s)
	}
}

// Handle 处理事件 - 实现 EventHandler 接口
func (s *notificationService) Handle(ctx context.Context, event *events.Event) error {
	return s.HandleEvent(ctx, event)
}

// HandleEvent 处理事件
func (s *notificationService) HandleEvent(ctx context.Context, event *events.Event) error {
	switch event.Type {
	case events.EventTaskCompleted:
		if event.UserID != nil && event.TaskID != nil {
			// 这里需要获取完整的任务信息
			// task := getTaskFromEvent(event)
			// s.NotifyTaskStatusChange(*event.UserID, task, "running", "completed")
		}
	case events.EventTaskFailed:
		if event.UserID != nil && event.TaskID != nil {
			// 类似处理其他事件...
		}
	case events.EventAccountStatusChanged:
		if event.UserID != nil && event.AccountID != nil {
			// 处理账号状态变更事件...
		}
	}
	return nil
}

// 实现事件处理器接口的方法
func (s *notificationService) SupportedTypes() []events.EventType {
	return []events.EventType{
		events.EventTaskStarted,
		events.EventTaskCompleted,
		events.EventTaskFailed,
		events.EventTaskCancelled,
		events.EventAccountStatusChanged,
		events.EventAccountCreated,
		events.EventAccountDeleted,
	}
}

// 其他方法的简单实现

func (s *notificationService) NotifyTaskCompleted(userID uint64, task *models.Task) error {
	return s.NotifyTaskStatusChange(userID, task, "running", "completed")
}

func (s *notificationService) NotifyTaskFailed(userID uint64, task *models.Task, reason string) error {
	return s.NotifyTaskStatusChange(userID, task, "running", "failed")
}

func (s *notificationService) NotifyAccountError(userID uint64, accountID uint64, error string) error {
	notification := &Notification{
		ID:       s.generateNotificationID(),
		Type:     NotificationTypeAccountStatus,
		Priority: PriorityHigh,
		Title:    "账号错误",
		Message:  fmt.Sprintf("账号 #%d 发生错误: %s", accountID, error),
		Data: map[string]interface{}{
			"account_id": accountID,
			"error":      error,
		},
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	return s.SendToUser(userID, notification)
}

func (s *notificationService) NotifyProxyStatusChange(userID uint64, proxyID uint64, status string) error {
	notification := &Notification{
		ID:       s.generateNotificationID(),
		Type:     NotificationTypeProxyStatus,
		Priority: PriorityNormal,
		Title:    "代理状态变更",
		Message:  fmt.Sprintf("代理 #%d 状态变更为: %s", proxyID, status),
		Data: map[string]interface{}{
			"proxy_id": proxyID,
			"status":   status,
		},
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	return s.SendToUser(userID, notification)
}

func (s *notificationService) NotifySystemMaintenance(message string, scheduledAt time.Time) error {
	notification := &Notification{
		ID:       s.generateNotificationID(),
		Type:     NotificationTypeSystemAlert,
		Priority: PriorityNormal,
		Title:    "系统维护通知",
		Message:  message,
		Data: map[string]interface{}{
			"scheduled_at": scheduledAt,
		},
		CreatedAt: time.Now(),
	}

	return s.SendToAll(notification)
}

func (s *notificationService) NotifyRateLimitExceeded(userID uint64) error {
	notification := &Notification{
		ID:        s.generateNotificationID(),
		Type:      NotificationTypeSystemAlert,
		Priority:  PriorityHigh,
		Title:     "请求频率超限",
		Message:   "您的请求频率过高，请稍后再试",
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	return s.SendToUser(userID, notification)
}

func (s *notificationService) PushTaskQueueUpdate(userID uint64, accountID uint64, queueInfo *models.QueueInfo) error {
	return s.PushRealTimeStats(userID, map[string]interface{}{
		"type":       "queue_update",
		"account_id": accountID,
		"queue_info": queueInfo,
	})
}

func (s *notificationService) GetUnreadNotifications(userID uint64) ([]*Notification, error) {
	s.notificationsMutex.RLock()
	defer s.notificationsMutex.RUnlock()

	var unread []*Notification
	for _, notification := range s.notifications {
		if notification.UserID == userID && notification.ReadAt == nil {
			unread = append(unread, notification)
		}
	}

	return unread, nil
}

func (s *notificationService) MarkNotificationAsRead(userID uint64, notificationID string) error {
	s.notificationsMutex.Lock()
	defer s.notificationsMutex.Unlock()

	if notification, exists := s.notifications[notificationID]; exists {
		if notification.UserID == userID {
			now := time.Now()
			notification.ReadAt = &now
		}
	}

	return nil
}

func (s *notificationService) MarkAllAsRead(userID uint64) error {
	s.notificationsMutex.Lock()
	defer s.notificationsMutex.Unlock()

	now := time.Now()
	for _, notification := range s.notifications {
		if notification.UserID == userID && notification.ReadAt == nil {
			notification.ReadAt = &now
		}
	}

	return nil
}

func (s *notificationService) CleanupOldNotifications(olderThan time.Duration) error {
	s.notificationsMutex.Lock()
	defer s.notificationsMutex.Unlock()

	cutoff := time.Now().Add(-olderThan)
	for id, notification := range s.notifications {
		if notification.CreatedAt.Before(cutoff) {
			delete(s.notifications, id)
		}
	}

	return nil
}

func (s *notificationService) handleSubscribe(client *WSConnection, msg map[string]interface{}) {
	// TODO: 实现订阅逻辑
}

func (s *notificationService) handleUnsubscribe(client *WSConnection, msg map[string]interface{}) {
	// TODO: 实现取消订阅逻辑
}
