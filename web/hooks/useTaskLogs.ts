import { useState, useEffect, useCallback, useMemo } from "react";
import { wsManager, type ConnectionStatus, type WSMessage } from "@/lib/websocket";

// 日志级别
export type LogLevel = "info" | "warn" | "error" | "debug";

// 任务日志条目
export interface TaskLogEntry {
  id: number;
  task_id: number;
  account_id?: number;
  level: LogLevel;
  action: string;
  message: string;
  extra_data?: Record<string, unknown>;
  created_at: string;
}

// 重新导出连接状态类型
export type { ConnectionStatus };

// 日志统计信息
export interface LogStats {
  total: number;
  info: number;
  warn: number;
  error: number;
  debug: number;
}

// Hook 配置选项
interface UseTaskLogsOptions {
  taskId: number;
  autoSubscribe?: boolean;
  maxLogs?: number;
}

// Hook 返回值
interface UseTaskLogsReturn {
  logs: TaskLogEntry[];
  connectionStatus: ConnectionStatus;
  isLoading: boolean;
  error: string | null;
  stats: LogStats;
  latestLog: TaskLogEntry | null;
  hasNewLogs: boolean;
  subscribe: () => void;
  unsubscribe: () => void;
  reconnect: () => void;
  clearLogs: () => void;
  markLogsAsRead: () => void;
}

export function useTaskLogs(options: UseTaskLogsOptions): UseTaskLogsReturn {
  const {
    taskId,
    autoSubscribe = true,
    maxLogs = 1000,
  } = options;

  // 状态
  const [logs, setLogs] = useState<TaskLogEntry[]>([]);
  const [connectionStatus, setConnectionStatus] = useState<ConnectionStatus>(wsManager.getStatus());
  const [isLoading, setIsLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [hasNewLogs, setHasNewLogs] = useState(false);
  const [isSubscribed, setIsSubscribed] = useState(false);

  // 计算日志统计信息
  const stats = useMemo<LogStats>(() => {
    return logs.reduce(
      (acc, log) => {
        acc.total++;
        if (log.level && acc[log.level] !== undefined) {
          acc[log.level]++;
        }
        return acc;
      },
      { total: 0, info: 0, warn: 0, error: 0, debug: 0 }
    );
  }, [logs]);

  // 获取最新日志
  const latestLog = useMemo<TaskLogEntry | null>(() => {
    return logs.length > 0 ? logs[logs.length - 1] : null;
  }, [logs]);

  // 发送订阅请求
  const subscribe = useCallback(() => {
    if (isSubscribed || !taskId) return;

    // 确保 WebSocket 已连接
    if (wsManager.getStatus() !== "connected") {
      wsManager.connect();
    }

    setIsLoading(true);
    setError(null);

    const sent = wsManager.send({
      type: "subscribe_task_logs",
      task_id: taskId,
    });

    if (!sent) {
      setError("发送订阅请求失败");
      setIsLoading(false);
    }
  }, [taskId, isSubscribed]);

  // 发送取消订阅请求
  const unsubscribe = useCallback(() => {
    if (!isSubscribed || !taskId) return;

    wsManager.send({
      type: "unsubscribe_task_logs",
      task_id: taskId,
    });

    setIsSubscribed(false);
    setLogs([]);
  }, [taskId, isSubscribed]);

  // 重连
  const reconnect = useCallback(() => {
    wsManager.reconnect();
  }, []);

  // 清空日志
  const clearLogs = useCallback(() => {
    setLogs([]);
    setHasNewLogs(false);
  }, []);

  // 标记日志为已读
  const markLogsAsRead = useCallback(() => {
    setHasNewLogs(false);
  }, []);

  // 监听连接状态变化
  useEffect(() => {
    const unsubscribeStatus = wsManager.onStatusChange((status) => {
      setConnectionStatus(status);

      // 连接成功后，如果之前已订阅，重新订阅
      if (status === "connected" && autoSubscribe && taskId && !isSubscribed) {
        // 延迟一点发送订阅请求，确保连接稳定
        setTimeout(() => {
          subscribe();
        }, 100);
      }
    });

    return unsubscribeStatus;
  }, [autoSubscribe, taskId, isSubscribed, subscribe]);

  // 监听订阅成功消息
  useEffect(() => {
    const unsubscribe = wsManager.subscribe("subscribe_task_logs_success", (message: WSMessage) => {
      const data = message.data as {
        task_id: number;
        initial_logs: TaskLogEntry[];
        message: string;
      };

      if (data.task_id === taskId) {
        setLogs(data.initial_logs || []);
        setIsLoading(false);
        setIsSubscribed(true);
      }
    });

    return unsubscribe;
  }, [taskId]);

  // 监听取消订阅成功消息
  useEffect(() => {
    const unsubscribe = wsManager.subscribe("unsubscribe_task_logs_success", (message: WSMessage) => {
      const data = message.data as { task_id: number };
      if (data.task_id === taskId) {
        setIsSubscribed(false);
      }
    });

    return unsubscribe;
  }, [taskId]);

  // 监听实时日志推送
  useEffect(() => {
    const unsubscribe = wsManager.subscribe("task_log", (message: WSMessage) => {
      const data = message.data as {
        task_id: number;
        log: TaskLogEntry;
      };

      if (data.task_id === taskId && data.log) {
        setLogs((prevLogs) => {
          const newLogs = [...prevLogs, data.log];
          // 限制最大数量
          if (newLogs.length > maxLogs) {
            return newLogs.slice(newLogs.length - maxLogs);
          }
          return newLogs;
        });
        setHasNewLogs(true);
      }
    });

    return unsubscribe;
  }, [taskId, maxLogs]);

  // 监听错误消息
  useEffect(() => {
    const unsubscribe = wsManager.subscribe("error", (message: WSMessage) => {
      const data = message.data as { message: string };
      setError(data.message || "Unknown error");
      setIsLoading(false);
    });

    return unsubscribe;
  }, []);

  // 自动订阅
  useEffect(() => {
    if (autoSubscribe && taskId) {
      // 确保连接
      if (wsManager.getStatus() !== "connected") {
        wsManager.connect();
      } else {
        subscribe();
      }
    }

    // 组件卸载时取消订阅
    return () => {
      if (isSubscribed) {
        wsManager.send({
          type: "unsubscribe_task_logs",
          task_id: taskId,
        });
      }
    };
  }, [taskId]); // eslint-disable-line react-hooks/exhaustive-deps

  return {
    logs,
    connectionStatus,
    isLoading,
    error,
    stats,
    latestLog,
    hasNewLogs,
    subscribe,
    unsubscribe,
    reconnect,
    clearLogs,
    markLogsAsRead,
  };
}
