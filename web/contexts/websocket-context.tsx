"use client";

import { createContext, useContext, useEffect, useState, type ReactNode } from "react";
import { wsManager, type ConnectionStatus } from "@/lib/websocket";

interface WebSocketContextValue {
  status: ConnectionStatus;
  connect: () => void;
  disconnect: () => void;
  reconnect: () => void;
}

const WebSocketContext = createContext<WebSocketContextValue | null>(null);

export function WebSocketProvider({ children }: { children: ReactNode }) {
  const [status, setStatus] = useState<ConnectionStatus>(wsManager.getStatus());

  useEffect(() => {
    // 监听状态变化
    const unsubscribe = wsManager.onStatusChange(setStatus);

    // 检查是否已登录，如果已登录则自动连接
    const token = typeof window !== "undefined" ? localStorage.getItem("token") : null;
    if (token && wsManager.getStatus() === "disconnected") {
      wsManager.connect();
    }

    return () => {
      unsubscribe();
    };
  }, []);

  const value: WebSocketContextValue = {
    status,
    connect: () => wsManager.connect(),
    disconnect: () => wsManager.disconnect(),
    reconnect: () => wsManager.reconnect(),
  };

  return (
    <WebSocketContext.Provider value={value}>
      {children}
    </WebSocketContext.Provider>
  );
}

export function useWebSocket(): WebSocketContextValue {
  const context = useContext(WebSocketContext);
  if (!context) {
    throw new Error("useWebSocket must be used within a WebSocketProvider");
  }
  return context;
}
