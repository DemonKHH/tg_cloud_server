package events

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
)

// EventType 事件类型
type EventType string

const (
	// 用户事件
	EventUserRegistered EventType = "user.registered"
	EventUserLoggedIn   EventType = "user.logged_in"
	EventUserLoggedOut  EventType = "user.logged_out"

	// 账号事件
	EventAccountCreated       EventType = "account.created"
	EventAccountUpdated       EventType = "account.updated"
	EventAccountDeleted       EventType = "account.deleted"
	EventAccountStatusChanged EventType = "account.status_changed"
	EventAccountProxyBound    EventType = "account.proxy_bound"

	// 任务事件
	EventTaskCreated   EventType = "task.created"
	EventTaskStarted   EventType = "task.started"
	EventTaskCompleted EventType = "task.completed"
	EventTaskFailed    EventType = "task.failed"
	EventTaskCancelled EventType = "task.cancelled"
	EventTaskRetried   EventType = "task.retried"

	// 代理事件
	EventProxyCreated       EventType = "proxy.created"
	EventProxyUpdated       EventType = "proxy.updated"
	EventProxyDeleted       EventType = "proxy.deleted"
	EventProxyTestStarted   EventType = "proxy.test_started"
	EventProxyTestCompleted EventType = "proxy.test_completed"

	// Telegram事件
	EventTelegramConnected    EventType = "telegram.connected"
	EventTelegramDisconnected EventType = "telegram.disconnected"
	EventTelegramAuthFailed   EventType = "telegram.auth_failed"
	EventTelegramRateLimit    EventType = "telegram.rate_limit"

	// 系统事件
	EventSystemStarted EventType = "system.started"
	EventSystemStopped EventType = "system.stopped"
	EventSystemError   EventType = "system.error"
)

// Event 事件结构
type Event struct {
	ID        string                 `json:"id"`
	Type      EventType              `json:"type"`
	Source    string                 `json:"source"`
	UserID    *uint64                `json:"user_id,omitempty"`
	AccountID *uint64                `json:"account_id,omitempty"`
	TaskID    *uint64                `json:"task_id,omitempty"`
	ProxyID   *uint64                `json:"proxy_id,omitempty"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
}

// EventHandler 事件处理器接口
type EventHandler interface {
	Handle(ctx context.Context, event *Event) error
	SupportedTypes() []EventType
}

// EventBus 事件总线接口
type EventBus interface {
	Publish(ctx context.Context, event *Event) error
	Subscribe(eventType EventType, handler EventHandler) error
	Unsubscribe(eventType EventType, handler EventHandler) error
	Close() error
}

// InMemoryEventBus 内存事件总线实现
type InMemoryEventBus struct {
	handlers map[EventType][]EventHandler
	mutex    sync.RWMutex
	logger   *zap.Logger
}

// NewInMemoryEventBus 创建内存事件总线
func NewInMemoryEventBus() EventBus {
	return &InMemoryEventBus{
		handlers: make(map[EventType][]EventHandler),
		logger:   logger.Get().Named("event_bus"),
	}
}

// Publish 发布事件
func (bus *InMemoryEventBus) Publish(ctx context.Context, event *Event) error {
	bus.mutex.RLock()
	handlers, exists := bus.handlers[event.Type]
	bus.mutex.RUnlock()

	if !exists || len(handlers) == 0 {
		bus.logger.Debug("No handlers for event type",
			zap.String("event_type", string(event.Type)),
			zap.String("event_id", event.ID))
		return nil
	}

	bus.logger.Info("Publishing event",
		zap.String("event_type", string(event.Type)),
		zap.String("event_id", event.ID),
		zap.Int("handler_count", len(handlers)))

	// 异步处理事件
	for _, handler := range handlers {
		go func(h EventHandler) {
			if err := h.Handle(ctx, event); err != nil {
				bus.logger.Error("Event handler failed",
					zap.String("event_type", string(event.Type)),
					zap.String("event_id", event.ID),
					zap.Error(err))
			}
		}(handler)
	}

	return nil
}

// Subscribe 订阅事件
func (bus *InMemoryEventBus) Subscribe(eventType EventType, handler EventHandler) error {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	if bus.handlers[eventType] == nil {
		bus.handlers[eventType] = make([]EventHandler, 0)
	}

	bus.handlers[eventType] = append(bus.handlers[eventType], handler)

	bus.logger.Info("Event handler subscribed",
		zap.String("event_type", string(eventType)),
		zap.Int("total_handlers", len(bus.handlers[eventType])))

	return nil
}

// Unsubscribe 取消订阅事件
func (bus *InMemoryEventBus) Unsubscribe(eventType EventType, handler EventHandler) error {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	handlers, exists := bus.handlers[eventType]
	if !exists {
		return nil
	}

	for i, h := range handlers {
		if h == handler {
			bus.handlers[eventType] = append(handlers[:i], handlers[i+1:]...)
			bus.logger.Info("Event handler unsubscribed",
				zap.String("event_type", string(eventType)))
			break
		}
	}

	return nil
}

// Close 关闭事件总线
func (bus *InMemoryEventBus) Close() error {
	bus.mutex.Lock()
	defer bus.mutex.Unlock()

	bus.handlers = make(map[EventType][]EventHandler)
	bus.logger.Info("Event bus closed")
	return nil
}

// EventService 事件服务
type EventService struct {
	bus    EventBus
	logger *zap.Logger
}

// NewEventService 创建事件服务
func NewEventService(bus EventBus) *EventService {
	return &EventService{
		bus:    bus,
		logger: logger.Get().Named("event_service"),
	}
}

// PublishUserEvent 发布用户事件
func (s *EventService) PublishUserEvent(ctx context.Context, eventType EventType, userID uint64, data map[string]interface{}) error {
	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "user_service",
		UserID:    &userID,
		Data:      data,
		Timestamp: time.Now(),
		Version:   "1.0",
	}
	return s.bus.Publish(ctx, event)
}

// PublishAccountEvent 发布账号事件
func (s *EventService) PublishAccountEvent(ctx context.Context, eventType EventType, userID, accountID uint64, data map[string]interface{}) error {
	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "account_service",
		UserID:    &userID,
		AccountID: &accountID,
		Data:      data,
		Timestamp: time.Now(),
		Version:   "1.0",
	}
	return s.bus.Publish(ctx, event)
}

// PublishTaskEvent 发布任务事件
func (s *EventService) PublishTaskEvent(ctx context.Context, eventType EventType, userID, taskID, accountID uint64, data map[string]interface{}) error {
	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "task_service",
		UserID:    &userID,
		TaskID:    &taskID,
		AccountID: &accountID,
		Data:      data,
		Timestamp: time.Now(),
		Version:   "1.0",
	}
	return s.bus.Publish(ctx, event)
}

// PublishProxyEvent 发布代理事件
func (s *EventService) PublishProxyEvent(ctx context.Context, eventType EventType, userID, proxyID uint64, data map[string]interface{}) error {
	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "proxy_service",
		UserID:    &userID,
		ProxyID:   &proxyID,
		Data:      data,
		Timestamp: time.Now(),
		Version:   "1.0",
	}
	return s.bus.Publish(ctx, event)
}

// PublishTelegramEvent 发布Telegram事件
func (s *EventService) PublishTelegramEvent(ctx context.Context, eventType EventType, accountID uint64, data map[string]interface{}) error {
	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "telegram_service",
		AccountID: &accountID,
		Data:      data,
		Timestamp: time.Now(),
		Version:   "1.0",
	}
	return s.bus.Publish(ctx, event)
}

// PublishSystemEvent 发布系统事件
func (s *EventService) PublishSystemEvent(ctx context.Context, eventType EventType, data map[string]interface{}) error {
	event := &Event{
		ID:        generateEventID(),
		Type:      eventType,
		Source:    "system",
		Data:      data,
		Timestamp: time.Now(),
		Version:   "1.0",
	}
	return s.bus.Publish(ctx, event)
}

// Subscribe 订阅事件
func (s *EventService) Subscribe(eventType EventType, handler EventHandler) error {
	return s.bus.Subscribe(eventType, handler)
}

// 内置事件处理器

// LoggingEventHandler 日志事件处理器
type LoggingEventHandler struct {
	logger *zap.Logger
}

// NewLoggingEventHandler 创建日志事件处理器
func NewLoggingEventHandler() EventHandler {
	return &LoggingEventHandler{
		logger: logger.Get().Named("event_logger"),
	}
}

// Handle 处理事件
func (h *LoggingEventHandler) Handle(ctx context.Context, event *Event) error {
	eventData, _ := json.Marshal(event)
	h.logger.Info("Event received",
		zap.String("event_type", string(event.Type)),
		zap.String("event_id", event.ID),
		zap.String("source", event.Source),
		zap.String("event_data", string(eventData)))
	return nil
}

// SupportedTypes 支持的事件类型
func (h *LoggingEventHandler) SupportedTypes() []EventType {
	return []EventType{
		EventUserRegistered, EventUserLoggedIn, EventUserLoggedOut,
		EventAccountCreated, EventAccountUpdated, EventAccountDeleted, EventAccountStatusChanged,
		EventTaskCreated, EventTaskStarted, EventTaskCompleted, EventTaskFailed,
		EventProxyCreated, EventProxyUpdated, EventProxyDeleted,
		EventTelegramConnected, EventTelegramDisconnected, EventTelegramAuthFailed,
		EventSystemStarted, EventSystemStopped, EventSystemError,
	}
}

// MetricsEventHandler 指标事件处理器
type MetricsEventHandler struct {
	logger *zap.Logger
}

// NewMetricsEventHandler 创建指标事件处理器
func NewMetricsEventHandler() EventHandler {
	return &MetricsEventHandler{
		logger: logger.Get().Named("metrics_event_handler"),
	}
}

// Handle 处理事件
func (h *MetricsEventHandler) Handle(ctx context.Context, event *Event) error {
	// 根据事件类型更新相应的指标
	switch event.Type {
	case EventTaskCompleted, EventTaskFailed:
		// 更新任务相关指标
		if duration, ok := event.Data["duration"].(float64); ok {
			// 这里可以调用metrics服务更新指标
			h.logger.Debug("Updating task metrics",
				zap.String("event_type", string(event.Type)),
				zap.Float64("duration", duration))
		}
	case EventAccountStatusChanged:
		// 更新账号状态指标
		if status, ok := event.Data["new_status"].(string); ok {
			h.logger.Debug("Updating account status metrics",
				zap.String("status", status))
		}
	}
	return nil
}

// SupportedTypes 支持的事件类型
func (h *MetricsEventHandler) SupportedTypes() []EventType {
	return []EventType{
		EventTaskCreated, EventTaskCompleted, EventTaskFailed,
		EventAccountStatusChanged,
		EventTelegramConnected, EventTelegramDisconnected,
	}
}

// generateEventID 生成事件ID
func generateEventID() string {
	return fmt.Sprintf("evt_%d_%s", time.Now().UnixNano(), randString(8))
}

// randString 生成随机字符串
func randString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
