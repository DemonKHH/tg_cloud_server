"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { X, RefreshCw, CheckCircle2, Clock, PlayCircle, AlertCircle, Ban, FileText, Pause, Play, Square, Trash2, Search, ChevronDown, Eye } from "lucide-react"
import { taskAPI } from "@/lib/api"
import { useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { cn } from "@/lib/utils"
import { usePagination } from "@/hooks/use-pagination"
import { Card, CardContent } from "@/components/ui/card"
import {
  getTaskTypeLabel,
  getTaskStatusLabel,
  getPersonalityLabel,
  getConfigFieldLabel,
  formatDuration,
  formatPercent,
} from "@/lib/task-config"

export default function TasksPage() {
  const {
    data: tasks,
    page,
    total,
    loading,
    search,
    setSearch,
    filters,
    updateFilter,
    setPage,
    refresh,
  } = usePagination({
    fetchFn: taskAPI.list,
    initialFilters: { status: "" },
  })

  const statusFilter = filters.status || ""



  // 查看日志相关状态
  const [logsDialogOpen, setLogsDialogOpen] = useState(false)
  const [viewingTask, setViewingTask] = useState<any>(null)
  const [logs, setLogs] = useState<any[]>([])
  const [loadingLogs, setLoadingLogs] = useState(false)

  // 确认对话框状态
  const [cancelDialogOpen, setCancelDialogOpen] = useState(false)
  const [cancellingTask, setCancellingTask] = useState<any>(null)
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deletingTask, setDeletingTask] = useState<any>(null)

  // 查看详情状态
  const [detailDialogOpen, setDetailDialogOpen] = useState(false)
  const [detailTask, setDetailTask] = useState<any>(null)



  const loadLogs = async (taskId: string) => {
    try {
      setLoadingLogs(true)
      const response = await taskAPI.getLogs(taskId)
      if (response.code === 0 && response.data) {
        // 确保logs是数组格式
        const logsData = Array.isArray(response.data) ? response.data : []
        setLogs(logsData)
      } else {
        toast.error(response.msg || "加载日志失败")
        setLogs([])
      }
    } catch (error: any) {
      console.error("加载日志失败:", error)
      const errorMessage = error instanceof Error ? error.message : "加载日志失败"
      toast.error(errorMessage)
      setLogs([])
    } finally {
      setLoadingLogs(false)
    }
  }

  const getTaskTypeText = (type: string) => {
    return getTaskTypeLabel(type)
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "completed":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />
      case "running":
        return <PlayCircle className="h-4 w-4 text-blue-500" />
      case "failed":
        return <AlertCircle className="h-4 w-4 text-red-500" />
      case "pending":
        return <Clock className="h-4 w-4 text-yellow-500" />
      case "queued":
        return <Clock className="h-4 w-4 text-blue-500" />
      case "paused":
        return <Pause className="h-4 w-4 text-orange-500" />
      case "cancelled":
        return <Ban className="h-4 w-4 text-gray-500" />
      default:
        return <Clock className="h-4 w-4 text-gray-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "completed":
        return "bg-green-50 text-green-700 border border-green-200 dark:bg-green-900 dark:text-green-300 dark:border-green-800"
      case "running":
        return "bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800"
      case "failed":
        return "bg-red-50 text-red-700 border border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800"
      case "queued":
        return "bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800"
      case "pending":
        return "bg-yellow-50 text-yellow-700 border border-yellow-200 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800"
      case "paused":
        return "bg-orange-50 text-orange-700 border border-orange-200 dark:bg-orange-900 dark:text-orange-300 dark:border-orange-800"
      case "cancelled":
        return "bg-gray-50 text-gray-700 border border-gray-200 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-800"
      default:
        return "bg-gray-50 text-gray-700 border border-gray-200 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-800"
    }
  }

  const getStatusText = (status: string) => {
    return getTaskStatusLabel(status)
  }

  // 取消任务
  // 取消任务 - 打开确认对话框
  const handleCancelTask = (task: any) => {
    setCancellingTask(task)
    setCancelDialogOpen(true)
  }

  // 确认取消任务
  const confirmCancelTask = async () => {
    if (!cancellingTask) return

    try {
      const res = await taskAPI.cancel(String(cancellingTask.id))
      if (res.code === 0) {
        toast.success("任务已取消")
        refresh()
        setCancelDialogOpen(false)
        setCancellingTask(null)
      } else {
        toast.error(res.msg || "取消任务失败")
      }
    } catch (error: any) {
      console.error('取消任务失败:', error)
      const errorMessage = error instanceof Error ? error.message : "取消任务失败"
      toast.error(errorMessage)
    }
  }

  // 删除任务
  // 删除任务 - 打开确认对话框
  const handleDeleteTask = (task: any) => {
    setDeletingTask(task)
    setDeleteDialogOpen(true)
  }

  // 确认删除任务
  const confirmDeleteTask = async () => {
    if (!deletingTask) return

    try {
      const res = await taskAPI.delete(String(deletingTask.id))
      if (res.code === 0) {
        toast.success("任务已删除")
        refresh()
        setDeleteDialogOpen(false)
        setDeletingTask(null)
      } else {
        toast.error(res.msg || "删除任务失败")
      }
    } catch (error: any) {
      console.error('删除任务失败:', error)
      const errorMessage = error instanceof Error ? error.message : "删除任务失败"
      toast.error(errorMessage)
    }
  }

  // 重试任务
  const handleRetryTask = async (task: any) => {
    try {
      const res = await taskAPI.retry(String(task.id))
      if (res.code === 0) {
        toast.success("任务已重新执行")
        refresh()
      } else {
        toast.error(res.msg || "重试任务失败")
      }
    } catch (error: any) {
      console.error('重试任务失败:', error)
      const errorMessage = error instanceof Error ? error.message : "重试任务失败"
      toast.error(errorMessage)
    }
  }

  // 启动任务
  const handleStartTask = async (task: any) => {
    try {
      const res = await taskAPI.control(String(task.id), 'start')
      if (res.code === 0) {
        toast.success("任务已启动")
        refresh()
      } else {
        toast.error(res.msg || "启动任务失败")
      }
    } catch (error: any) {
      console.error('启动任务失败:', error)
      const errorMessage = error instanceof Error ? error.message : "启动任务失败"
      toast.error(errorMessage)
    }
  }

  // 停止任务
  const handleStopTask = async (task: any) => {
    try {
      const res = await taskAPI.control(String(task.id), 'stop')
      if (res.code === 0) {
        toast.success("任务已停止")
        refresh()
      } else {
        toast.error(res.msg || "停止任务失败")
      }
    } catch (error: any) {
      console.error('停止任务失败:', error)
      const errorMessage = error instanceof Error ? error.message : "停止任务失败"
      toast.error(errorMessage)
    }
  }

  // 检查操作是否可用
  const isActionEnabled = (action: string, status: string) => {
    switch (action) {
      case 'start':
        return status === 'pending'
      case 'stop':
        return ['running', 'pending', 'queued'].includes(status)
      case 'cancel':
        return ['pending', 'queued'].includes(status)
      case 'retry':
        return ['failed', 'cancelled'].includes(status)
      case 'delete':
        return true // 删除操作在所有状态下都可用
      default:
        return false
    }
  }

  // 获取操作按钮的提示信息
  const getActionTooltip = (action: string, status: string) => {
    const statusText = getStatusText(status)
    const enabled = isActionEnabled(action, status)

    switch (action) {
      case 'start':
        return enabled ? '启动任务' : `启动任务 - 只有待执行的任务才能启动（当前: ${statusText}）`
      case 'stop':
        return enabled ? '停止任务' : `停止任务 - 只有运行中、待执行或排队的任务才能停止（当前: ${statusText}）`
      case 'cancel':
        return enabled ? '取消任务' : `取消任务 - 只有待执行或排队的任务才能取消（当前: ${statusText}）`
      case 'retry':
        return enabled ? '重试任务' : `重试任务 - 只有失败或已取消的任务才能重试（当前: ${statusText}）`
      case 'delete':
        return '删除任务 (不可恢复)'
      case 'logs':
        return '查看任务日志'
      default:
        return '操作不可用'
    }
  }

  // 渲染操作按钮
  const renderActionButton = (action: string, record: any, icon: React.ReactNode, handler: () => void, colorClass?: string) => {
    const enabled = isActionEnabled(action, record.status)
    const tooltip = getActionTooltip(action, record.status)

    return (
      <Tooltip key={action}>
        <TooltipTrigger asChild>
          <Button
            variant="ghost"
            size="icon"
            className={cn(
              "h-8 w-8",
              enabled
                ? colorClass || "hover:bg-primary/10 text-primary"
                : "opacity-40 cursor-not-allowed text-muted-foreground"
            )}
            disabled={!enabled}
            onClick={enabled ? handler : undefined}
          >
            {icon}
          </Button>
        </TooltipTrigger>
        <TooltipContent side="top">
          <p className="text-xs">{tooltip}</p>
        </TooltipContent>
      </Tooltip>
    )
  }

  // 查看日志
  const handleViewLogs = async (task: any) => {
    setViewingTask(task)
    setLogsDialogOpen(true)
    await loadLogs(String(task.id))
  }

  // 查看详情
  const handleViewDetail = (task: any) => {
    setDetailTask(task)
    setDetailDialogOpen(true)
  }

  // 渲染任务配置详情
  const renderTaskConfig = (task: any) => {
    const config = task.config || task.task_config
    if (!config || Object.keys(config).length === 0) {
      return <p className="text-muted-foreground text-sm">无配置信息</p>
    }

    switch (task.task_type) {
      case 'scenario':
        return (
          <div className="space-y-3">
            {config.name && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('name')}</span>
                <span>{config.name}</span>
              </div>
            )}
            {config.topic && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('topic')}</span>
                <span className="font-mono">{config.topic}</span>
              </div>
            )}
            {config.duration && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('duration')}</span>
                <span>{formatDuration(config.duration)}</span>
              </div>
            )}
            {config.agents && config.agents.length > 0 && (
              <div className="space-y-2">
                <p className="text-muted-foreground">{getConfigFieldLabel('agents')} ({config.agents.length}个)</p>
                <div className="space-y-2 pl-2 border-l-2 border-muted">
                  {config.agents.map((agent: any, idx: number) => (
                    <div key={idx} className="bg-muted/50 p-2 rounded text-sm">
                      <div className="font-medium">{agent.persona?.name || `智能体 ${idx + 1}`}</div>
                      {agent.persona?.style && (
                        <div className="text-xs text-muted-foreground">{getConfigFieldLabel('style')}: {Array.isArray(agent.persona.style) ? agent.persona.style.join(', ') : agent.persona.style}</div>
                      )}
                      {agent.goal && <div className="text-xs text-muted-foreground">{getConfigFieldLabel('goal')}: {agent.goal}</div>}
                      <div className="text-xs text-muted-foreground">{getConfigFieldLabel('active_rate')}: {formatPercent(agent.active_rate || 0.5)}</div>
                    </div>
                  ))}
                </div>
              </div>
            )}
          </div>
        )

      case 'group_chat':
        return (
          <div className="space-y-3">
            {(config.group_name || config.group_id) && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('target_group')}</span>
                <span className="font-mono">{config.group_name || config.group_id}</span>
              </div>
            )}
            {config.monitor_duration_seconds && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('monitor_duration_seconds')}</span>
                <span>{formatDuration(config.monitor_duration_seconds)}</span>
              </div>
            )}
            {config.ai_config && (
              <>
                {config.ai_config.personality && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{getConfigFieldLabel('personality')}</span>
                    <span>{getPersonalityLabel(config.ai_config.personality)}</span>
                  </div>
                )}
                {config.ai_config.response_rate && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{getConfigFieldLabel('response_rate')}</span>
                    <span>{formatPercent(config.ai_config.response_rate)}</span>
                  </div>
                )}
                {config.ai_config.keywords && config.ai_config.keywords.length > 0 && (
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">{getConfigFieldLabel('keywords')}</span>
                    <span>{config.ai_config.keywords.join(', ')}</span>
                  </div>
                )}
              </>
            )}
          </div>
        )

      case 'private_message':
        return (
          <div className="space-y-3">
            {config.target_user && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('target_user')}</span>
                <span>{config.target_user}</span>
              </div>
            )}
            {config.message && (
              <div className="space-y-1">
                <span className="text-muted-foreground">{getConfigFieldLabel('message')}</span>
                <p className="bg-muted p-2 rounded text-sm">{config.message}</p>
              </div>
            )}
          </div>
        )

      case 'join_group':
        return (
          <div className="space-y-3">
            {config.group_link && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('group_link')}</span>
                <span className="font-mono text-sm break-all">{config.group_link}</span>
              </div>
            )}
            {config.group_username && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('group_username')}</span>
                <span>@{config.group_username}</span>
              </div>
            )}
          </div>
        )

      case 'force_add_group':
        return (
          <div className="space-y-3">
            {config.group_id && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('group_id')}</span>
                <span className="font-mono">{config.group_id}</span>
              </div>
            )}
            {config.user_ids && config.user_ids.length > 0 && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('user_ids')}</span>
                <span>{config.user_ids.length} 人</span>
              </div>
            )}
          </div>
        )

      case 'terminate_sessions':
        return (
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-muted-foreground">操作类型</span>
              <span>踢出其他设备</span>
            </div>
            {config.keep_current !== undefined && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('keep_current')}</span>
                <span>{config.keep_current ? "是" : "否"}</span>
              </div>
            )}
          </div>
        )

      case 'update_2fa':
        return (
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-muted-foreground">操作类型</span>
              <span>修改两步验证密码</span>
            </div>
            {config.hint && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('hint')}</span>
                <span>{config.hint}</span>
              </div>
            )}
          </div>
        )

      case 'check':
        return (
          <div className="space-y-3">
            <div className="flex justify-between">
              <span className="text-muted-foreground">操作类型</span>
              <span>账号状态检查</span>
            </div>
            {config.check_type && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('check_type')}</span>
                <span>{config.check_type === 'health_check' ? '健康检查' : config.check_type}</span>
              </div>
            )}
            {config.timeout && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('timeout')}</span>
                <span>{config.timeout}</span>
              </div>
            )}
            {/* 显示检查项配置 */}
            {Object.entries(config)
              .filter(([key]) => !['check_type', 'timeout'].includes(key) && !key.startsWith('_'))
              .map(([key, value]) => (
                <div key={key} className="flex justify-between">
                  <span className="text-muted-foreground">{getConfigFieldLabel(key)}</span>
                  <span>{typeof value === 'boolean' ? (value ? '是' : '否') : (typeof value === 'object' ? JSON.stringify(value) : String(value))}</span>
                </div>
              ))}
          </div>
        )

      case 'broadcast':
        return (
          <div className="space-y-3">
            {config.message && (
              <div className="space-y-1">
                <span className="text-muted-foreground">{getConfigFieldLabel('message')}</span>
                <p className="bg-muted p-2 rounded text-sm">{config.message}</p>
              </div>
            )}
            {config.target_groups && config.target_groups.length > 0 && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">目标群组数</span>
                <span>{config.target_groups.length} 个</span>
              </div>
            )}
            {config.interval && (
              <div className="flex justify-between">
                <span className="text-muted-foreground">{getConfigFieldLabel('interval')}</span>
                <span>{formatDuration(config.interval)}</span>
              </div>
            )}
          </div>
        )

      default:
        // 默认：尝试智能解析常见字段
        return (
          <div className="space-y-3">
            {renderGenericConfig(config)}
          </div>
        )
    }
  }

  // 通用配置渲染（用于未知任务类型）
  const renderGenericConfig = (config: any) => {
    const entries = Object.entries(config).filter(([key]) => !key.startsWith('_'))
    
    if (entries.length === 0) {
      return <p className="text-muted-foreground text-sm">无配置信息</p>
    }

    return entries.map(([key, value]) => {
      const label = getConfigFieldLabel(key)
      let displayValue: React.ReactNode = String(value)

      if (typeof value === 'boolean') {
        displayValue = value ? "是" : "否"
      } else if (Array.isArray(value)) {
        displayValue = value.length > 0 ? value.join(', ') : "无"
      } else if (typeof value === 'object' && value !== null) {
        displayValue = (
          <pre className="bg-muted p-2 rounded text-xs overflow-x-auto whitespace-pre-wrap">
            {JSON.stringify(value, null, 2)}
          </pre>
        )
      } else if (key.includes('duration') || key.includes('seconds') || key.includes('interval')) {
        const seconds = Number(value)
        if (!isNaN(seconds)) {
          displayValue = formatDuration(seconds)
        }
      } else if (key.includes('rate') && typeof value === 'number' && value <= 1) {
        displayValue = formatPercent(value)
      } else if (key === 'personality') {
        displayValue = getPersonalityLabel(String(value))
      }

      return (
        <div key={key} className={typeof value === 'object' && value !== null && !Array.isArray(value) ? "space-y-1" : "flex justify-between"}>
          <span className="text-muted-foreground">{label}</span>
          {typeof value === 'object' && value !== null && !Array.isArray(value) ? displayValue : <span>{displayValue}</span>}
        </div>
      )
    })
  }







  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight gradient-text">任务管理</h1>
            <p className="text-muted-foreground mt-1">查看和管理您的任务执行情况</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={refresh} className="btn-modern">
              <RefreshCw className="h-4 w-4 mr-2" />
              刷新
            </Button>

          </div>
        </div>

        {/* Search and Filters */}
        <Card className="border-none shadow-sm">
          <CardContent className="p-4">
            <div className="flex items-center gap-4">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="搜索任务ID或账号..."
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                  className="pl-9 input-modern"
                />
              </div>
              {search && (
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => setSearch("")}
                  className="text-muted-foreground hover:text-foreground"
                >
                  <X className="h-4 w-4 mr-1" />
                  清除
                </Button>
              )}
              <Select
                value={statusFilter || "all"}
                onValueChange={(value) => updateFilter("status", value === "all" ? "" : value)}
              >
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="筛选状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="pending">待执行</SelectItem>
                  <SelectItem value="queued">已排队</SelectItem>
                  <SelectItem value="running">执行中</SelectItem>
                  <SelectItem value="paused">已暂停</SelectItem>
                  <SelectItem value="completed">已完成</SelectItem>
                  <SelectItem value="failed">失败</SelectItem>
                  <SelectItem value="cancelled">已取消</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardContent>
        </Card>

        {/* 任务数据表 */}
        <Card className="border-none shadow-md overflow-hidden">
          <CardContent className="p-0">
            <div className="overflow-x-auto">
              <Table>
                <TableHeader>
                  <TableRow className="bg-muted/50 hover:bg-muted/50 border-b-2">
                    <TableHead className="w-[100px] h-12 font-semibold">任务ID</TableHead>
                    <TableHead className="w-[150px] font-semibold">任务类型</TableHead>
                    <TableHead className="w-[120px] font-semibold">状态</TableHead>
                    <TableHead className="w-[120px] font-semibold">账号</TableHead>
                    <TableHead className="w-[100px] font-semibold">优先级</TableHead>
                    <TableHead className="w-[180px] font-semibold">创建时间</TableHead>
                    <TableHead className="w-[180px] font-semibold">完成时间</TableHead>
                    <TableHead className="w-[240px] text-right font-semibold">操作</TableHead>
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {loading ? (
                    // 加载状态
                    Array.from({ length: 5 }).map((_, index) => (
                      <TableRow key={index}>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                        <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
                      </TableRow>
                    ))
                  ) : tasks.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={8} className="h-64">
                        <div className="flex flex-col items-center justify-center">
                          <Clock className="h-12 w-12 text-muted-foreground/50 mb-4" />
                          <p className="text-lg font-medium text-muted-foreground mb-2">暂无任务数据</p>
                          <p className="text-sm text-muted-foreground mb-6">创建您的第一个任务</p>

                        </div>
                      </TableCell>
                    </TableRow>
                  ) : (
                    tasks.map((record) => (
                      <TableRow key={record.id} className="group transition-colors hover:bg-muted/50">
                        <TableCell>
                          <span className="font-mono text-sm">#{record.id}</span>
                        </TableCell>
                        <TableCell>
                          <Badge variant="secondary" className="text-xs">
                            {getTaskTypeText(record.task_type)}
                          </Badge>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            {getStatusIcon(record.status)}
                            <Badge
                              variant={record.status === 'completed' ? 'default' : record.status === 'failed' ? 'destructive' : 'secondary'}
                              className={cn("text-xs", getStatusColor(record.status))}
                            >
                              {getStatusText(record.status)}
                            </Badge>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm">
                            {record.account_phone || record.account?.phone || `ID: ${record.account_id}`}
                          </div>
                        </TableCell>
                        <TableCell>
                          <span className="text-sm">{record.priority}/10</span>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-muted-foreground">
                            {new Date(record.created_at).toLocaleString()}
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="text-sm text-muted-foreground">
                            {record.completed_at ? new Date(record.completed_at).toLocaleString() : '-'}
                          </div>
                        </TableCell>
                        <TableCell>
                          <TooltipProvider>
                            <div className="flex items-center gap-1">
                              {/* 查看详情 */}
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8 hover:bg-purple-50 text-purple-600 hover:text-purple-700"
                                    onClick={() => handleViewDetail(record)}
                                  >
                                    <Eye className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  <p className="text-xs">查看任务详情</p>
                                </TooltipContent>
                              </Tooltip>

                              {/* 查看日志 - 始终可用 */}
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8 hover:bg-blue-50 text-blue-600 hover:text-blue-700"
                                    onClick={() => handleViewLogs(record)}
                                  >
                                    <FileText className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  <p className="text-xs">查看任务日志</p>
                                </TooltipContent>
                              </Tooltip>

                              {/* 启动任务 */}
                              {renderActionButton(
                                'start',
                                record,
                                <Play className="h-4 w-4" />,
                                () => handleStartTask(record),
                                "hover:bg-green-50 text-green-600 hover:text-green-700"
                              )}

                              {/* 暂停任务 */}
                              {renderActionButton(
                                'pause',
                                record,
                                <Square className="h-4 w-4" />,
                                () => handleStopTask(record),
                                "hover:bg-red-50 text-red-600 hover:text-red-700"
                              )}

                              {/* 取消任务 */}
                              {renderActionButton(
                                'cancel',
                                record,
                                <X className="h-4 w-4" />,
                                () => handleCancelTask(record),
                                "hover:bg-gray-50 text-gray-600 hover:text-gray-700"
                              )}

                              {/* 重试任务 */}
                              {renderActionButton(
                                'retry',
                                record,
                                <RefreshCw className="h-4 w-4" />,
                                () => handleRetryTask(record),
                                "hover:bg-purple-50 text-purple-600 hover:text-purple-700"
                              )}

                              {/* 删除任务 */}
                              {renderActionButton(
                                'delete',
                                record,
                                <Trash2 className="h-4 w-4" />,
                                () => handleDeleteTask(record),
                                "hover:bg-red-50 text-red-600 hover:text-red-700"
                              )}
                            </div>
                          </TooltipProvider>
                        </TableCell>
                      </TableRow>
                    ))
                  )}
                </TableBody>
              </Table>
            </div>

            {/* 分页 */}
            <div className="flex items-center justify-between px-6 py-4 border-t bg-muted/30">
              <div className="flex items-center gap-4">
                <div className="text-sm font-medium text-foreground">
                  共 <span className="text-primary font-bold">{total}</span> 个任务
                </div>
                <div className="h-4 w-px bg-border" />
                <div className="text-sm text-muted-foreground">
                  第 {page} 页 · 每页 20 条
                </div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => Math.max(1, p - 1))}
                  disabled={page === 1}
                  className="btn-modern h-9 px-4"
                >
                  <ChevronDown className="h-4 w-4 mr-1 rotate-90" />
                  上一页
                </Button>
                <div className="flex items-center gap-1 px-2">
                  <span className="text-sm font-medium">{page}</span>
                  <span className="text-sm text-muted-foreground">/</span>
                  <span className="text-sm text-muted-foreground">{Math.ceil(total / 20)}</span>
                </div>
                <Button
                  variant="outline"
                  size="sm"
                  onClick={() => setPage((p) => p + 1)}
                  disabled={page * 20 >= total}
                  className="btn-modern h-9 px-4"
                >
                  下一页
                  <ChevronDown className="h-4 w-4 ml-1 -rotate-90" />
                </Button>
              </div>
            </div>
          </CardContent>
        </Card>



        {/* 查看日志对话框 */}
        <Dialog open={logsDialogOpen} onOpenChange={setLogsDialogOpen}>
          <DialogContent className="sm:max-w-[800px] max-h-[600px] overflow-y-auto">
            <DialogHeader>
              <DialogTitle>任务日志 - #{viewingTask?.id}</DialogTitle>
              <DialogDescription>
                {viewingTask && `${getTaskTypeText(viewingTask.task_type)} - ${getStatusText(viewingTask.status)}`}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-2 py-4">
              {loadingLogs ? (
                <div className="text-center text-muted-foreground py-4">加载中...</div>
              ) : logs.length === 0 ? (
                <div className="text-center text-muted-foreground py-4">暂无日志</div>
              ) : (
                logs.map((log: any, index: number) => (
                  <div key={index} className="border-l-2 border-muted pl-4 py-2">
                    <div className="flex items-center justify-between mb-1">
                      <span className="text-sm font-medium">{log.action}</span>
                      <span className="text-xs text-muted-foreground">
                        {new Date(log.created_at).toLocaleString()}
                      </span>
                    </div>
                    <div className="text-sm text-muted-foreground">{log.message}</div>
                  </div>
                ))
              )}
            </div>
          </DialogContent>
        </Dialog>

        {/* 取消任务确认对话框 */}
        <Dialog open={cancelDialogOpen} onOpenChange={setCancelDialogOpen}>
          <DialogContent className="sm:max-w-[400px]">
            <DialogHeader>
              <DialogTitle className="text-xl text-yellow-600 flex items-center gap-2">
                <AlertCircle className="h-5 w-5" />
                确认取消任务
              </DialogTitle>
              <DialogDescription className="pt-2">
                您确定要取消任务 <span className="font-semibold text-foreground">#{cancellingTask?.id}</span> 吗？
              </DialogDescription>
            </DialogHeader>
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => setCancelDialogOpen(false)} className="btn-modern">
                暂不取消
              </Button>
              <Button
                variant="default"
                onClick={confirmCancelTask}
                className="btn-modern bg-yellow-600 hover:bg-yellow-700 text-white"
              >
                确认取消
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 删除任务确认对话框 */}
        <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
          <DialogContent className="sm:max-w-[400px]">
            <DialogHeader>
              <DialogTitle className="text-xl text-red-600 flex items-center gap-2">
                <Trash2 className="h-5 w-5" />
                确认删除任务
              </DialogTitle>
              <DialogDescription className="pt-2">
                您确定要删除任务 <span className="font-semibold text-foreground">#{deletingTask?.id}</span> 吗？
                <br />
                <span className="text-red-500 text-xs mt-2 block">
                  此操作将永久删除该任务及其所有相关数据（包括日志），且不可恢复。
                </span>
              </DialogDescription>
            </DialogHeader>
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => setDeleteDialogOpen(false)} className="btn-modern">
                取消
              </Button>
              <Button
                variant="destructive"
                onClick={confirmDeleteTask}
                className="btn-modern bg-red-600 hover:bg-red-700"
              >
                确认删除
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 任务详情对话框 */}
        <Dialog open={detailDialogOpen} onOpenChange={setDetailDialogOpen}>
          <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle className="flex items-center gap-2">
                <Eye className="h-5 w-5" />
                任务详情 - #{detailTask?.id}
              </DialogTitle>
              <DialogDescription>
                {detailTask && `${getTaskTypeText(detailTask.task_type)}`}
              </DialogDescription>
            </DialogHeader>
            {detailTask && (
              <div className="space-y-4 py-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-1">
                    <p className="text-sm text-muted-foreground">任务ID</p>
                    <p className="font-mono">#{detailTask.id}</p>
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm text-muted-foreground">任务类型</p>
                    <Badge variant="secondary">{getTaskTypeText(detailTask.task_type)}</Badge>
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm text-muted-foreground">状态</p>
                    <div className="flex items-center gap-2">
                      {getStatusIcon(detailTask.status)}
                      <Badge className={getStatusColor(detailTask.status)}>
                        {getStatusText(detailTask.status)}
                      </Badge>
                    </div>
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm text-muted-foreground">优先级</p>
                    <p>{detailTask.priority}/10</p>
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm text-muted-foreground">账号</p>
                    <p>{detailTask.account_phone || detailTask.account?.phone || `ID: ${detailTask.account_id}`}</p>
                  </div>
                  <div className="space-y-1">
                    <p className="text-sm text-muted-foreground">创建时间</p>
                    <p className="text-sm">{new Date(detailTask.created_at).toLocaleString()}</p>
                  </div>
                  {detailTask.started_at && (
                    <div className="space-y-1">
                      <p className="text-sm text-muted-foreground">开始时间</p>
                      <p className="text-sm">{new Date(detailTask.started_at).toLocaleString()}</p>
                    </div>
                  )}
                  {detailTask.completed_at && (
                    <div className="space-y-1">
                      <p className="text-sm text-muted-foreground">完成时间</p>
                      <p className="text-sm">{new Date(detailTask.completed_at).toLocaleString()}</p>
                    </div>
                  )}
                </div>
                {(detailTask.config || detailTask.task_config) && (
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-muted-foreground border-b pb-2">任务配置</p>
                    {renderTaskConfig(detailTask)}
                  </div>
                )}
                {detailTask.result && (
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-muted-foreground border-b pb-2">执行结果</p>
                    <pre className="bg-muted p-4 rounded-lg text-xs overflow-x-auto whitespace-pre-wrap">
                      {typeof detailTask.result === 'string' ? detailTask.result : JSON.stringify(detailTask.result, null, 2)}
                    </pre>
                  </div>
                )}
                {detailTask.error && (
                  <div className="space-y-2">
                    <p className="text-sm font-medium text-red-500 border-b pb-2">错误信息</p>
                    <pre className="bg-red-50 dark:bg-red-900/20 p-4 rounded-lg text-xs text-red-600 dark:text-red-400 overflow-x-auto whitespace-pre-wrap">
                      {detailTask.error}
                    </pre>
                  </div>
                )}
              </div>
            )}
          </DialogContent>
        </Dialog>
      </div>
    </MainLayout>
  )
}

