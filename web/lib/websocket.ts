/**
 * 全局 WebSocket 管理器
 * 单例模式，整个应用共享一个 WebSocket 连接
 */

// 获取 WebSocket URL
function getWebSocketUrl(): string {
  if (typeof window === "undefined") {
    return "";
  }

  // 从环境变量获取后端 API URL
  const apiUrl = process.env.NEXT_PUBLIC_API_URL || "";

  if (apiUrl) {
    const url = new URL(apiUrl);
    const wsProtocol = url.protocol === "https:" ? "wss:" : "ws:";
    return `${wsProtocol}//${url.host}/api/v1/ws`;
  }

  // 开发环境默认连接到 localhost:8080
  const isDev = process.env.NODE_ENV === "development";
  if (isDev) {
    return "ws://localhost:8080/api/v1/ws";
  }

  // 生产环境使用当前页面的 host
  const protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  const host = window.location.host;
  return `${protocol}//${host}/api/v1/ws`;
}

// 获取认证 token
function getAuthToken(): string | null {
  if (typeof window === "undefined") {
    return null;
  }
  return localStorage.getItem("token");
}

// 连接状态
export type ConnectionStatus = "connecting" | "connected" | "disconnected" | "error";

// WebSocket 消息类型
export interface WSMessage {
  type: string;
  data: unknown;
  timestamp: string;
}

// 消息监听器类型
type MessageListener = (message: WSMessage) => void;

// 状态变化监听器类型
type StatusListener = (status: ConnectionStatus) => void;

class WebSocketManager {
  private static instance: WebSocketManager | null = null;
  private ws: WebSocket | null = null;
  private status: ConnectionStatus = "disconnected";
  private messageListeners: Map<string, Set<MessageListener>> = new Map();
  private statusListeners: Set<StatusListener> = new Set();
  private reconnectAttempts = 0;
  private maxReconnectAttempts = 5;
  private reconnectInterval = 3000;
  private reconnectTimer: ReturnType<typeof setTimeout> | null = null;
  private pingTimer: ReturnType<typeof setInterval> | null = null;

  private constructor() {}

  static getInstance(): WebSocketManager {
    if (!WebSocketManager.instance) {
      WebSocketManager.instance = new WebSocketManager();
    }
    return WebSocketManager.instance;
  }

  // 获取当前连接状态
  getStatus(): ConnectionStatus {
    return this.status;
  }

  // 连接 WebSocket
  connect(): void {
    if (this.ws?.readyState === WebSocket.OPEN || 
        this.ws?.readyState === WebSocket.CONNECTING) {
      return;
    }

    const token = getAuthToken();
    if (!token) {
      this.setStatus("error");
      console.warn("[WebSocket] No auth token available");
      return;
    }

    const wsUrl = getWebSocketUrl();
    if (!wsUrl) {
      this.setStatus("error");
      console.warn("[WebSocket] WebSocket URL not available");
      return;
    }

    this.setStatus("connecting");

    try {
      this.ws = new WebSocket(`${wsUrl}?token=${encodeURIComponent(token)}`);

      this.ws.onopen = () => {
        console.log("[WebSocket] Connected");
        this.setStatus("connected");
        this.reconnectAttempts = 0;
        this.clearReconnectTimer();
        this.startPing();
      };

      this.ws.onmessage = (event) => {
        try {
          const message: WSMessage = JSON.parse(event.data);
          this.notifyListeners(message);
        } catch (err) {
          console.error("[WebSocket] Failed to parse message:", err);
        }
      };

      this.ws.onerror = (error) => {
        console.error("[WebSocket] Error:", error);
        this.setStatus("error");
      };

      this.ws.onclose = (event) => {
        console.log("[WebSocket] Closed:", event.code, event.reason);
        this.setStatus("disconnected");
        this.stopPing();
        this.ws = null;

        // 非正常关闭时尝试重连
        if (!event.wasClean && this.reconnectAttempts < this.maxReconnectAttempts) {
          this.scheduleReconnect();
        }
      };
    } catch (err) {
      console.error("[WebSocket] Failed to create connection:", err);
      this.setStatus("error");
    }
  }

  // 断开连接
  disconnect(): void {
    this.clearReconnectTimer();
    this.stopPing();
    this.reconnectAttempts = this.maxReconnectAttempts; // 防止自动重连

    if (this.ws) {
      this.ws.close(1000, "User disconnected");
      this.ws = null;
    }

    this.setStatus("disconnected");
  }

  // 重新连接
  reconnect(): void {
    this.disconnect();
    this.reconnectAttempts = 0;
    setTimeout(() => this.connect(), 100);
  }

  // 发送消息
  send(message: object): boolean {
    if (this.ws?.readyState !== WebSocket.OPEN) {
      console.warn("[WebSocket] Cannot send message: not connected");
      return false;
    }

    try {
      this.ws.send(JSON.stringify(message));
      return true;
    } catch (err) {
      console.error("[WebSocket] Failed to send message:", err);
      return false;
    }
  }

  // 订阅特定类型的消息
  subscribe(type: string, listener: MessageListener): () => void {
    if (!this.messageListeners.has(type)) {
      this.messageListeners.set(type, new Set());
    }
    this.messageListeners.get(type)!.add(listener);

    // 返回取消订阅函数
    return () => {
      this.messageListeners.get(type)?.delete(listener);
    };
  }

  // 订阅连接状态变化
  onStatusChange(listener: StatusListener): () => void {
    this.statusListeners.add(listener);
    // 立即通知当前状态
    listener(this.status);

    return () => {
      this.statusListeners.delete(listener);
    };
  }

  // 私有方法

  private setStatus(status: ConnectionStatus): void {
    if (this.status !== status) {
      this.status = status;
      this.statusListeners.forEach((listener) => listener(status));
    }
  }

  private notifyListeners(message: WSMessage): void {
    console.log("[WebSocket] Received message:", message.type, message);
    
    // 通知特定类型的监听器
    const listeners = this.messageListeners.get(message.type);
    if (listeners) {
      console.log("[WebSocket] Notifying", listeners.size, "listeners for type:", message.type);
      listeners.forEach((listener) => listener(message));
    }

    // 通知通配符监听器
    const wildcardListeners = this.messageListeners.get("*");
    if (wildcardListeners) {
      wildcardListeners.forEach((listener) => listener(message));
    }
  }

  private scheduleReconnect(): void {
    const delay = this.reconnectInterval * Math.pow(2, this.reconnectAttempts);
    const maxDelay = 30000;
    const actualDelay = Math.min(delay, maxDelay);

    console.log(`[WebSocket] Reconnecting in ${actualDelay}ms (attempt ${this.reconnectAttempts + 1})`);

    this.reconnectTimer = setTimeout(() => {
      this.reconnectAttempts++;
      this.connect();
    }, actualDelay);
  }

  private clearReconnectTimer(): void {
    if (this.reconnectTimer) {
      clearTimeout(this.reconnectTimer);
      this.reconnectTimer = null;
    }
  }

  private startPing(): void {
    this.stopPing();
    this.pingTimer = setInterval(() => {
      this.send({ type: "ping" });
    }, 30000);
  }

  private stopPing(): void {
    if (this.pingTimer) {
      clearInterval(this.pingTimer);
      this.pingTimer = null;
    }
  }
}

// 导出单例实例
export const wsManager = WebSocketManager.getInstance();

// 导出便捷方法
export function connectWebSocket(): void {
  wsManager.connect();
}

export function disconnectWebSocket(): void {
  wsManager.disconnect();
}

export function reconnectWebSocket(): void {
  wsManager.reconnect();
}
