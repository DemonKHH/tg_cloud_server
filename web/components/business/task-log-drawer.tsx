"use client"

import { useEffect, useRef, useState } from "react"
import { motion, AnimatePresence } from "framer-motion"
import {
  Sheet,
  SheetContent,
  SheetHeader,
  SheetTitle,
  SheetDescription,
} from "@/components/ui/sheet"
import { Button } from "@/components/ui/button"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"
import {
  useTaskLogs,
  type TaskLogEntry,
  type LogLevel,
  type ConnectionStatus,
} from "@/hooks/useTaskLogs"
import {
  RefreshCw,
  Wifi,
  WifiOff,
  AlertCircle,
  Loader2,
  ArrowDown,
  Info,
  AlertTriangle,
  XCircle,
  Bug,
  ScrollText,
} from "lucide-react"

interface TaskLogDrawerProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  taskId: number
  taskName?: string
}

// 日志级别颜色配置
const logLevelConfig: Record<
  LogLevel,
  {
    color: string
    bgColor: string
    borderColor: string
    icon: React.ReactNode
    label: string
  }
> = {
  info: {
    color: "text-blue-600 dark:text-blue-400",
    bgColor: "bg-blue-50 dark:bg-blue-950/30",
    borderColor: "border-blue-200 dark:border-blue-800",
    icon: <Info className="size-3.5" />,
    label: "信息",
  },
  warn: {
    color: "text-amber-600 dark:text-amber-400",
    bgColor: "bg-amber-50 dark:bg-amber-950/30",
    borderColor: "border-amber-200 dark:border-amber-800",
    icon: <AlertTriangle className="size-3.5" />,
    label: "警告",
  },
  error: {
    color: "text-red-600 dark:text-red-400",
    bgColor: "bg-red-50 dark:bg-red-950/30",
    borderColor: "border-red-200 dark:border-red-800",
    icon: <XCircle className="size-3.5" />,
    label: "错误",
  },
  debug: {
    color: "text-gray-500 dark:text-gray-400",
    bgColor: "bg-gray-50 dark:bg-gray-900/30",
    borderColor: "border-gray-200 dark:border-gray-700",
    icon: <Bug className="size-3.5" />,
    label: "调试",
  },
}

// 连接状态配置
const connectionStatusConfig: Record<
  ConnectionStatus,
  {
    color: string
    bgColor: string
    icon: React.ReactNode
    label: string
  }
> = {
  connecting: {
    color: "text-amber-600 dark:text-amber-400",
    bgColor: "bg-amber-100 dark:bg-amber-900/30",
    icon: <Loader2 className="size-3 animate-spin" />,
    label: "连接中",
  },
  connected: {
    color: "text-emerald-600 dark:text-emerald-400",
    bgColor: "bg-emerald-100 dark:bg-emerald-900/30",
    icon: <Wifi className="size-3" />,
    label: "已连接",
  },
  disconnected: {
    color: "text-gray-500 dark:text-gray-400",
    bgColor: "bg-gray-100 dark:bg-gray-800",
    icon: <WifiOff className="size-3" />,
    label: "已断开",
  },
  error: {
    color: "text-red-600 dark:text-red-400",
    bgColor: "bg-red-100 dark:bg-red-900/30",
    icon: <AlertCircle className="size-3" />,
    label: "连接错误",
  },
}

// 格式化时间
function formatTime(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleTimeString("zh-CN", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  })
}

// 单条日志组件
function LogEntry({ log }: { log: TaskLogEntry }) {
  // 获取日志级别配置，如果级别未知则使用 info 作为默认值
  const level = log.level && logLevelConfig[log.level] ? log.level : "info"
  const config = logLevelConfig[level]

  return (
    <motion.div
      initial={{ opacity: 0, x: 20 }}
      animate={{ opacity: 1, x: 0 }}
      className={cn(
        "flex gap-2 px-3 py-2 rounded-lg border text-sm",
        config.bgColor,
        config.borderColor
      )}
    >
      <div className={cn("flex items-center gap-1 shrink-0", config.color)}>
        {config.icon}
      </div>
      <div className="flex-1 min-w-0">
        <div className="flex items-center gap-2 mb-0.5 flex-wrap">
          <span className="text-[10px] text-muted-foreground font-mono">
            {formatTime(log.created_at)}
          </span>
          {log.action && (
            <Badge variant="outline" className="text-[10px] px-1 py-0 h-4">
              {log.action}
            </Badge>
          )}
          {log.account_id && (
            <span className="text-[10px] text-muted-foreground">
              #{log.account_id}
            </span>
          )}
        </div>
        <p className="text-xs break-words leading-relaxed">{log.message}</p>
        {log.extra_data && Object.keys(log.extra_data).length > 0 && (
          <pre className="mt-1 text-[10px] text-muted-foreground bg-background/50 rounded p-1 overflow-x-auto">
            {JSON.stringify(log.extra_data, null, 2)}
          </pre>
        )}
      </div>
    </motion.div>
  )
}

// 连接状态指示器
function ConnectionIndicator({
  status,
  onReconnect,
}: {
  status: ConnectionStatus
  onReconnect: () => void
}) {
  const config = connectionStatusConfig[status]

  return (
    <div className="flex items-center gap-2">
      <div
        className={cn(
          "flex items-center gap-1 px-2 py-0.5 rounded-full text-[10px] font-medium",
          config.bgColor,
          config.color
        )}
      >
        {config.icon}
        <span>{config.label}</span>
      </div>
      {(status === "disconnected" || status === "error") && (
        <Button
          variant="ghost"
          size="sm"
          onClick={onReconnect}
          className="h-6 px-2 text-[10px]"
        >
          <RefreshCw className="size-3 mr-1" />
          重连
        </Button>
      )}
    </div>
  )
}

// 加载状态
function LoadingIndicator() {
  return (
    <div className="flex flex-col items-center justify-center py-12 gap-3">
      <Loader2 className="size-6 animate-spin text-primary" />
      <span className="text-xs text-muted-foreground">加载日志中...</span>
    </div>
  )
}

// 空状态
function EmptyState() {
  return (
    <div className="flex flex-col items-center justify-center py-12 gap-2 text-muted-foreground">
      <ScrollText className="size-8" />
      <span className="text-xs">暂无日志</span>
    </div>
  )
}

export function TaskLogDrawer({
  open,
  onOpenChange,
  taskId,
  taskName,
}: TaskLogDrawerProps) {
  const {
    logs,
    connectionStatus,
    isLoading,
    error,
    stats,
    reconnect,
    markLogsAsRead,
  } = useTaskLogs({
    taskId,
    autoSubscribe: open,
  })

  const scrollContainerRef = useRef<HTMLDivElement>(null)
  const [autoScroll, setAutoScroll] = useState(true)
  const [showScrollButton, setShowScrollButton] = useState(false)

  // 自动滚动到底部
  useEffect(() => {
    if (autoScroll && scrollContainerRef.current) {
      scrollContainerRef.current.scrollTop =
        scrollContainerRef.current.scrollHeight
    }
  }, [logs, autoScroll])

  // 监听滚动事件
  const handleScroll = () => {
    if (!scrollContainerRef.current) return

    const { scrollTop, scrollHeight, clientHeight } = scrollContainerRef.current
    const isAtBottom = scrollHeight - scrollTop - clientHeight < 50

    setAutoScroll(isAtBottom)
    setShowScrollButton(!isAtBottom && logs.length > 0)
  }

  // 滚动到底部
  const scrollToBottom = () => {
    if (scrollContainerRef.current) {
      scrollContainerRef.current.scrollTo({
        top: scrollContainerRef.current.scrollHeight,
        behavior: "smooth",
      })
      setAutoScroll(true)
    }
  }

  // 打开时标记日志为已读
  useEffect(() => {
    if (open) {
      markLogsAsRead()
    }
  }, [open, markLogsAsRead])

  return (
    <Sheet open={open} onOpenChange={onOpenChange}>
      <SheetContent 
        side="right" 
        className="w-full sm:max-w-md md:max-w-lg flex flex-col p-0"
      >
        {/* Header */}
        <SheetHeader className="px-4 py-3 border-b shrink-0">
          <div className="flex items-center justify-between pr-6">
            <div className="flex items-center gap-2">
              <ScrollText className="size-4 text-primary" />
              <SheetTitle className="text-base">任务日志</SheetTitle>
            </div>
            <ConnectionIndicator
              status={connectionStatus}
              onReconnect={reconnect}
            />
          </div>
          <SheetDescription className="text-xs">
            {taskName && <span>{taskName}</span>}
            {taskName && " · "}
            <span>共 {stats.total} 条日志</span>
            {stats.error > 0 && (
              <Badge variant="destructive" className="ml-2 text-[10px] h-4">
                {stats.error} 错误
              </Badge>
            )}
            {stats.warn > 0 && (
              <Badge variant="warning" className="ml-1 text-[10px] h-4">
                {stats.warn} 警告
              </Badge>
            )}
          </SheetDescription>
        </SheetHeader>

        {/* 错误提示 */}
        {error && (
          <div className="shrink-0 flex items-center gap-2 mx-4 mt-3 px-3 py-2 bg-red-50 dark:bg-red-950/30 border border-red-200 dark:border-red-800 rounded-lg text-red-600 dark:text-red-400 text-xs">
            <AlertCircle className="size-3.5 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        {/* 日志容器 */}
        <div className="flex-1 min-h-0 relative">
          <div
            ref={scrollContainerRef}
            onScroll={handleScroll}
            className="h-full overflow-y-auto p-4 space-y-2"
          >
            {isLoading ? (
              <LoadingIndicator />
            ) : logs.length === 0 ? (
              <EmptyState />
            ) : (
              <AnimatePresence mode="popLayout">
                {logs.map((log) => (
                  <LogEntry key={log.id} log={log} />
                ))}
              </AnimatePresence>
            )}
          </div>

          {/* 滚动到底部按钮 */}
          <AnimatePresence>
            {showScrollButton && (
              <motion.div
                initial={{ opacity: 0, y: 10 }}
                animate={{ opacity: 1, y: 0 }}
                exit={{ opacity: 0, y: 10 }}
                className="absolute bottom-4 right-4"
              >
                <Button
                  variant="secondary"
                  size="sm"
                  onClick={scrollToBottom}
                  className="shadow-lg h-7 text-xs"
                >
                  <ArrowDown className="size-3 mr-1" />
                  最新
                </Button>
              </motion.div>
            )}
          </AnimatePresence>
        </div>
      </SheetContent>
    </Sheet>
  )
}
