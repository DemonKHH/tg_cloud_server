package routes

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/gorilla/websocket"
	"go.uber.org/zap"

	"tg_cloud_server/internal/common/logger"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// 在生产环境中应该检查Origin
		return true
	},
}

// RegisterWebSocketRoutes 注册WebSocket路由
func RegisterWebSocketRoutes(router *gin.Engine, redisClient *redis.Client) {
	log := logger.Get().Named("websocket")

	// WebSocket连接端点
	router.GET("/ws", func(c *gin.Context) {
		// 升级HTTP连接到WebSocket
		conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
		if err != nil {
			log.Error("Failed to upgrade to websocket", zap.Error(err))
			return
		}
		defer conn.Close()

		log.Info("WebSocket connection established",
			zap.String("remote_addr", conn.RemoteAddr().String()))

		// 处理WebSocket连接
		handleWebSocketConnection(conn, redisClient, log)
	})

	// WebSocket状态端点
	router.GET("/ws/status", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":      "available",
			"endpoint":    "/ws",
			"description": "WebSocket connection for real-time updates",
		})
	})
}

// handleWebSocketConnection 处理WebSocket连接
func handleWebSocketConnection(conn *websocket.Conn, redisClient *redis.Client, log *zap.Logger) {
	// 设置读取限制
	conn.SetReadLimit(512)

	// 心跳检测
	conn.SetPongHandler(func(string) error {
		log.Debug("Received pong from client")
		return nil
	})

	// 监听消息
	for {
		messageType, message, err := conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Error("WebSocket read error", zap.Error(err))
			}
			break
		}

		log.Debug("Received WebSocket message",
			zap.Int("message_type", messageType),
			zap.String("message", string(message)))

		// 处理不同类型的消息
		switch messageType {
		case websocket.TextMessage:
			handleTextMessage(conn, message, log)
		case websocket.BinaryMessage:
			handleBinaryMessage(conn, message, log)
		case websocket.PingMessage:
			// 响应Ping
			if err := conn.WriteMessage(websocket.PongMessage, nil); err != nil {
				log.Error("Failed to send pong", zap.Error(err))
				return
			}
		}
	}

	log.Info("WebSocket connection closed",
		zap.String("remote_addr", conn.RemoteAddr().String()))
}

// handleTextMessage 处理文本消息
func handleTextMessage(conn *websocket.Conn, message []byte, log *zap.Logger) {
	// 这里可以解析JSON消息并进行相应处理
	// 例如：订阅任务状态更新、账号状态变化等

	response := map[string]interface{}{
		"type": "ack",
		"data": "Message received",
	}

	if err := conn.WriteJSON(response); err != nil {
		log.Error("Failed to send WebSocket response", zap.Error(err))
	}
}

// handleBinaryMessage 处理二进制消息
func handleBinaryMessage(conn *websocket.Conn, message []byte, log *zap.Logger) {
	log.Debug("Received binary message", zap.Int("size", len(message)))

	// 暂时不处理二进制消息
	response := []byte("Binary message received")
	if err := conn.WriteMessage(websocket.BinaryMessage, response); err != nil {
		log.Error("Failed to send binary response", zap.Error(err))
	}
}
