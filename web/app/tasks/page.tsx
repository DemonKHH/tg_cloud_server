"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, X, RefreshCw, CheckCircle2, Clock, PlayCircle, AlertCircle, Ban, FileText, MoreVertical, Pause, Play, Square, Trash2 } from "lucide-react"
import { taskAPI, accountAPI } from "@/lib/api"
import { useState, useEffect } from "react"
import { Badge } from "@/components/ui/badge"
import { ModernTable } from "@/components/ui/modern-table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
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
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Switch } from "@/components/ui/switch"
import { cn } from "@/lib/utils"
import { usePagination } from "@/hooks/use-pagination"
import { PageHeader } from "@/components/common/page-header"
import { FilterBar } from "@/components/common/filter-bar"

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
  
  // 创建任务相关状态
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [createForm, setCreateForm] = useState({
    account_id: "",
    task_type: "",
    priority: "5",
    auto_start: false, // 是否自动开始执行
    // 账号检查配置
    check_timeout: "2m",
    // 私信配置
    private_targets: "",
    private_message: "",
    private_delay: "",
    // 群发配置
    broadcast_message: "",
    broadcast_groups: "",
    broadcast_channels: "",
    broadcast_delay: "",
    // 验证码配置
    verify_timeout: "30",
    verify_source: "",
    verify_pattern: "",
    // AI炒群配置
    group_chat_group_id: "",
    group_chat_duration: "",
    group_chat_ai_config: "{}",
  })
  const [accounts, setAccounts] = useState<any[]>([])
  const [loadingAccounts, setLoadingAccounts] = useState(false)
  
  // 查看日志相关状态
  const [logsDialogOpen, setLogsDialogOpen] = useState(false)
  const [viewingTask, setViewingTask] = useState<any>(null)
  const [logs, setLogs] = useState<any[]>([])
  const [loadingLogs, setLoadingLogs] = useState(false)

  useEffect(() => {
    if (createDialogOpen) {
      loadAccounts()
    }
  }, [createDialogOpen])

  const loadAccounts = async () => {
    try {
      setLoadingAccounts(true)
      const response = await accountAPI.list({ page: 1, limit: 100 })
      if (response.data) {
        const data = response.data as any
        // 兼容不同的数据结构
        setAccounts(data.items || data.data || [])
      } else {
        setAccounts([])
      }
    } catch (error: any) {
      console.error("加载账号失败:", error)
      const errorMessage = error?.response?.data?.msg || error.message || "加载账号失败"
      toast.error(errorMessage)
      setAccounts([])
    } finally {
      setLoadingAccounts(false)
    }
  }

  const loadLogs = async (taskId: string) => {
    try {
      setLoadingLogs(true)
      const response = await taskAPI.getLogs(taskId)
      if (response.data) {
        // 确保logs是数组格式
        const logsData = Array.isArray(response.data) ? response.data : []
        setLogs(logsData)
      } else {
        setLogs([])
      }
    } catch (error: any) {
      console.error("加载日志失败:", error)
      const errorMessage = error?.response?.data?.msg || error.message || "加载日志失败"
      toast.error(errorMessage)
      setLogs([])
    } finally {
      setLoadingLogs(false)
    }
  }

  const getTaskTypeText = (type: string) => {
    const typeMap: Record<string, string> = {
      check: "账号检查",
      private_message: "私信发送",
      broadcast: "群发消息",
      verify_code: "验证码接收",
      group_chat: "AI炒群",
    }
    return typeMap[type] || type
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
    const statusMap: Record<string, string> = {
      pending: "待执行",
      queued: "已排队",
      running: "执行中",
      paused: "已暂停",
      completed: "已完成",
      failed: "失败",
      cancelled: "已取消",
    }
    return statusMap[status] || status
  }

  // 取消任务
  const handleCancelTask = async (task: any) => {
    if (!confirm(`确定要取消任务 #${task.id} 吗？`)) {
      return
    }

    try {
      await taskAPI.cancel(String(task.id))
      toast.success("任务已取消")
      refresh()
    } catch (error: any) {
      console.error('取消任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "取消任务失败"
      toast.error(errorMessage)
    }
  }

  // 删除任务
  const handleDeleteTask = async (task: any) => {
    if (!confirm(`确定要删除任务 #${task.id} 吗？此操作不可恢复。`)) {
      return
    }

    try {
      await taskAPI.delete(String(task.id))
      toast.success("任务已删除")
      refresh()
    } catch (error: any) {
      console.error('删除任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "删除任务失败"
      toast.error(errorMessage)
    }
  }

  // 重试任务
  const handleRetryTask = async (task: any) => {
    try {
      await taskAPI.retry(String(task.id))
      toast.success("任务已重新执行")
      refresh()
    } catch (error: any) {
      console.error('重试任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "重试任务失败"
      toast.error(errorMessage)
    }
  }

  // 启动任务
  const handleStartTask = async (task: any) => {
    try {
      await taskAPI.control(String(task.id), 'start')
      toast.success("任务已启动")
      refresh()
    } catch (error: any) {
      console.error('启动任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "启动任务失败"
      toast.error(errorMessage)
    }
  }

  // 暂停任务
  const handlePauseTask = async (task: any) => {
    try {
      await taskAPI.control(String(task.id), 'pause')
      toast.success("任务已暂停")
      refresh()
    } catch (error: any) {
      console.error('暂停任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "暂停任务失败"
      toast.error(errorMessage)
    }
  }

  // 恢复任务
  const handleResumeTask = async (task: any) => {
    try {
      await taskAPI.control(String(task.id), 'resume')
      toast.success("任务已恢复")
      refresh()
    } catch (error: any) {
      console.error('恢复任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "恢复任务失败"
      toast.error(errorMessage)
    }
  }

  // 停止任务
  const handleStopTask = async (task: any) => {
    try {
      await taskAPI.control(String(task.id), 'stop')
      toast.success("任务已停止")
      refresh()
    } catch (error: any) {
      console.error('停止任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "停止任务失败"
      toast.error(errorMessage)
    }
  }

  // 检查操作是否可用
  const isActionEnabled = (action: string, status: string) => {
    switch (action) {
      case 'start':
        return status === 'pending'
      case 'pause':
        return status === 'running'
      case 'resume':
        return status === 'paused'
      case 'stop':
        return ['running', 'paused', 'queued'].includes(status)
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
      case 'pause':
        return enabled ? '暂停任务' : `暂停任务 - 只有运行中的任务才能暂停（当前: ${statusText}）`
      case 'resume':
        return enabled ? '恢复任务' : `恢复任务 - 只有已暂停的任务才能恢复（当前: ${statusText}）`
      case 'stop':
        return enabled ? '停止任务' : `停止任务 - 只有运行中、暂停或排队的任务才能停止（当前: ${statusText}）`
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

  // 创建任务
  const handleCreateTask = () => {
    setCreateForm({
      account_id: "",
      task_type: "",
      priority: "5",
      auto_start: false,
      check_timeout: "2m",
      private_targets: "",
      private_message: "",
      private_delay: "",
      broadcast_message: "",
      broadcast_groups: "",
      broadcast_channels: "",
      broadcast_delay: "",
      verify_timeout: "30",
      verify_source: "",
      verify_pattern: "",
      group_chat_group_id: "",
      group_chat_duration: "",
      group_chat_ai_config: "{}",
    })
    setCreateDialogOpen(true)
  }

  // 当任务类型改变时，清空其他任务类型的配置
  const handleTaskTypeChange = (value: string) => {
    setCreateForm({
      ...createForm,
      task_type: value,
      // 清空所有配置字段
      check_timeout: "2m",
      private_targets: "",
      private_message: "",
      private_delay: "",
      broadcast_message: "",
      broadcast_groups: "",
      broadcast_channels: "",
      broadcast_delay: "",
      verify_timeout: "30",
      verify_source: "",
      verify_pattern: "",
      group_chat_group_id: "",
      group_chat_duration: "",
      group_chat_ai_config: "{}",
    })
  }

  // 构建任务配置
  const buildTaskConfig = () => {
    const config: any = {}
    
    switch (createForm.task_type) {
      case "check":
        // 账号检查配置 - 暂时不需要特殊配置，使用默认配置即可
        if (createForm.check_timeout && createForm.check_timeout !== "2m") {
          // 如果用户设置了非默认的超时时间，则添加到配置中
          config.timeout = createForm.check_timeout
        }
        break
        
      case "private_message":
        if (!createForm.private_targets || !createForm.private_message) {
          toast.error("请填写目标用户和消息内容")
          return null
        }
        // 处理目标用户列表
        const targets = createForm.private_targets.split(",").map(t => t.trim()).filter(t => t)
        if (targets.length === 0) {
          toast.error("请至少填写一个目标用户")
          return null
        }
        config.targets = targets
        config.message = createForm.private_message
        if (createForm.private_delay) {
          const delay = parseInt(createForm.private_delay)
          if (!isNaN(delay) && delay > 0) {
            config.interval_seconds = delay
          }
        }
        break
        
      case "broadcast":
        if (!createForm.broadcast_message) {
          toast.error("请填写消息内容")
          return null
        }
        if (!createForm.broadcast_groups && !createForm.broadcast_channels) {
          toast.error("请至少填写一个群组或频道ID")
          return null
        }
        config.message = createForm.broadcast_message
        
        // 合并群组和频道ID到一个groups数组中
        const allGroups: any[] = []
        if (createForm.broadcast_groups) {
          const groups = createForm.broadcast_groups.split(",")
            .map(g => {
              const num = parseInt(g.trim())
              return !isNaN(num) && num > 0 ? num : null
            })
            .filter(g => g !== null)
          allGroups.push(...groups)
        }
        if (createForm.broadcast_channels) {
          const channels = createForm.broadcast_channels.split(",")
            .map(c => {
              const num = parseInt(c.trim())
              return !isNaN(num) && num > 0 ? num : null
            })
            .filter(c => c !== null)
          allGroups.push(...channels)
        }
        
        if (allGroups.length === 0) {
          toast.error("请至少填写一个有效的群组或频道ID")
          return null
        }
        
        config.groups = allGroups
        if (createForm.broadcast_delay) {
          const delay = parseInt(createForm.broadcast_delay)
          if (!isNaN(delay) && delay > 0) {
            config.interval_seconds = delay
          }
        }
        break
        
      case "verify_code":
        // 验证码配置（所有字段都是可选的）
        if (createForm.verify_timeout) {
          const timeout = parseInt(createForm.verify_timeout)
          if (!isNaN(timeout) && timeout > 0) {
            config.timeout_seconds = timeout
          }
        }
        if (createForm.verify_source && createForm.verify_source.trim()) {
          // 后端期望的是senders数组
          config.senders = [createForm.verify_source.trim()]
        }
        // 注意：后端暂时没有pattern字段支持，这个功能需要后端实现
        if (createForm.verify_pattern && createForm.verify_pattern.trim()) {
          config.pattern = createForm.verify_pattern.trim()
        }
        break
        
      case "group_chat":
        if (!createForm.group_chat_group_id) {
          toast.error("请填写群组ID")
          return null
        }
        const groupId = parseInt(createForm.group_chat_group_id)
        if (isNaN(groupId) || groupId <= 0) {
          toast.error("群组ID必须是大于0的数字")
          return null
        }
        config.group_id = groupId
        if (createForm.group_chat_duration) {
          const duration = parseInt(createForm.group_chat_duration)
          if (!isNaN(duration) && duration > 0) {
            // 将分钟转换为秒，因为后端期望的是秒
            config.monitor_duration_seconds = duration * 60
          }
        }
        if (createForm.group_chat_ai_config && createForm.group_chat_ai_config.trim() !== "" && createForm.group_chat_ai_config !== "{}") {
          try {
            const aiConfig = JSON.parse(createForm.group_chat_ai_config)
            if (typeof aiConfig === 'object' && aiConfig !== null) {
              config.ai_config = aiConfig
            }
          } catch (e) {
            toast.error("AI配置JSON格式错误，请检查语法")
            return null
          }
        }
        break
        
      default:
        toast.error("请选择有效的任务类型")
        return null
    }
    
    return config
  }

  const handleSaveCreateTask = async () => {
    if (!createForm.account_id || !createForm.task_type) {
      toast.error("请选择账号和任务类型")
      return
    }

    try {
      const config = buildTaskConfig()
      if (config === null) {
        return // buildTaskConfig 已经显示了错误消息
      }

      const requestData = {
        account_id: parseInt(createForm.account_id),
        task_type: createForm.task_type,
        priority: parseInt(createForm.priority) || 5,
        auto_start: createForm.auto_start,
        task_config: config,
      }

      console.log('创建任务请求数据:', requestData) // 调试日志

      await taskAPI.create(requestData)
      toast.success("任务创建成功")
      setCreateDialogOpen(false)
      refresh()
    } catch (error: any) {
      console.error('创建任务失败:', error) // 调试日志
      const errorMessage = error?.response?.data?.msg || error.message || "创建任务失败"
      toast.error(errorMessage)
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <PageHeader
          title="任务管理"
          description="查看和管理您的任务"
          actions={
            <>
              <Button variant="outline" onClick={refresh}>
                <RefreshCw className="h-4 w-4 mr-2" />
                刷新
              </Button>
              <Button onClick={handleCreateTask}>
                <Plus className="h-4 w-4 mr-2" />
                创建任务
              </Button>
            </>
          }
        />

        {/* Filters */}
        <FilterBar
          search={search}
          onSearchChange={setSearch}
          searchPlaceholder="搜索任务ID或账号ID..."
          filters={
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
          }
        />

        {/* Tasks Table */}
        <ModernTable
          data={tasks}
          columns={[
            {
              key: 'id',
              title: '任务ID',
              width: '100px',
              sortable: true,
              render: (value) => (
                <span className="font-mono text-sm">#{value}</span>
              )
            },
            {
              key: 'task_type',
              title: '任务类型',
              width: '150px',
              render: (value) => (
                <Badge variant="secondary" className="text-xs">
                  {getTaskTypeText(value)}
                </Badge>
              )
            },
            {
              key: 'status',
              title: '状态',
              width: '120px',
              sortable: true,
              render: (value) => (
                <div className="flex items-center gap-2">
                  {getStatusIcon(value)}
                  <Badge
                    variant={value === 'completed' ? 'default' : value === 'failed' ? 'destructive' : 'secondary'}
                    className={cn("text-xs", getStatusColor(value))}
                  >
                    {getStatusText(value)}
                  </Badge>
                </div>
              )
            },
            {
              key: 'account_id',
              title: '账号',
              width: '120px',
              render: (value, record) => (
                <div className="text-sm">
                  {record.account_phone || record.account?.phone || `ID: ${value}`}
                </div>
              )
            },
            {
              key: 'priority',
              title: '优先级',
              width: '100px',
              sortable: true,
              render: (value) => (
                <span className="text-sm">{value}/10</span>
              )
            },
            {
              key: 'created_at',
              title: '创建时间',
              width: '180px',
              sortable: true,
              render: (value) => (
                <div className="text-sm text-muted-foreground">
                  {new Date(value).toLocaleString()}
                </div>
              )
            },
            {
              key: 'completed_at',
              title: '完成时间',
              width: '180px',
              sortable: true,
              render: (value) => (
                <div className="text-sm text-muted-foreground">
                  {value ? new Date(value).toLocaleString() : '-'}
                </div>
              )
            },
            {
              key: 'actions',
              title: '操作',
              width: '240px',
              render: (_, record) => (
                <TooltipProvider>
                  <div className="flex items-center gap-1">
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
                      <Pause className="h-4 w-4" />, 
                      () => handlePauseTask(record),
                      "hover:bg-orange-50 text-orange-600 hover:text-orange-700"
                    )}

                    {/* 恢复任务 */}
                    {renderActionButton(
                      'resume', 
                      record, 
                      <PlayCircle className="h-4 w-4" />, 
                      () => handleResumeTask(record),
                      "hover:bg-emerald-50 text-emerald-600 hover:text-emerald-700"
                    )}

                    {/* 停止任务 */}
                    {renderActionButton(
                      'stop', 
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
              )
            }
          ]}
          loading={loading}
          emptyText="暂无任务数据"
          className="card-shadow"
        />

        {/* Pagination */}
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            共 {total} 个任务，当前第 {page} 页
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
              className="btn-modern"
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => p + 1)}
              disabled={page * 20 >= total}
              className="btn-modern"
            >
              下一页
            </Button>
          </div>
        </div>

        {/* 创建任务对话框 */}
        <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>创建任务</DialogTitle>
              <DialogDescription>
                创建一个新的任务
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="create-account">选择账号 *</Label>
                <Select
                  value={createForm.account_id}
                  onValueChange={(value) => setCreateForm({ ...createForm, account_id: value })}
                  disabled={loadingAccounts}
                >
                  <SelectTrigger id="create-account">
                    <SelectValue placeholder={loadingAccounts ? "加载中..." : "选择账号"} />
                  </SelectTrigger>
                  <SelectContent>
                    {accounts.map((account) => (
                      <SelectItem key={account.id} value={String(account.id)}>
                        {account.phone} {account.note && `(${account.note})`}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="create-task-type">任务类型 *</Label>
                <Select
                  value={createForm.task_type}
                  onValueChange={handleTaskTypeChange}
                >
                  <SelectTrigger id="create-task-type">
                    <SelectValue placeholder="选择任务类型" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="check">账号检查</SelectItem>
                    <SelectItem value="private_message">私信发送</SelectItem>
                    <SelectItem value="broadcast">群发消息</SelectItem>
                    <SelectItem value="verify_code">验证码接收</SelectItem>
                    <SelectItem value="group_chat">AI炒群</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label htmlFor="create-priority">优先级 (1-10)</Label>
                <Input
                  id="create-priority"
                  type="number"
                  min="1"
                  max="10"
                  value={createForm.priority}
                  onChange={(e) => setCreateForm({ ...createForm, priority: e.target.value })}
                  placeholder="5"
                />
              </div>
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label htmlFor="auto-start">自动执行</Label>
                  <p className="text-sm text-muted-foreground">
                    创建任务后立即开始执行
                  </p>
                </div>
                <Switch
                  id="auto-start"
                  checked={createForm.auto_start}
                  onCheckedChange={(checked) => setCreateForm({ ...createForm, auto_start: checked })}
                />
              </div>
              {/* 根据任务类型显示不同的配置表单 */}
              {createForm.task_type === "private_message" && (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="private-targets">目标用户（逗号分隔）*</Label>
                    <Input
                      id="private-targets"
                      value={createForm.private_targets}
                      onChange={(e) => setCreateForm({ ...createForm, private_targets: e.target.value })}
                      placeholder="username1, username2, +1234567890"
                      required
                    />
                    <p className="text-xs text-muted-foreground">
                      支持用户名(@username)或手机号(+1234567890)，多个用逗号分隔
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="private-message">消息内容 *</Label>
                    <Textarea
                      id="private-message"
                      value={createForm.private_message}
                      onChange={(e) => setCreateForm({ ...createForm, private_message: e.target.value })}
                      placeholder="输入要发送的消息内容..."
                      rows={4}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="private-delay">发送间隔（秒，可选）</Label>
                    <Input
                      id="private-delay"
                      type="number"
                      min="0"
                      value={createForm.private_delay}
                      onChange={(e) => setCreateForm({ ...createForm, private_delay: e.target.value })}
                      placeholder="0"
                    />
                  </div>
                </>
              )}

              {createForm.task_type === "broadcast" && (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="broadcast-message">消息内容 *</Label>
                    <Textarea
                      id="broadcast-message"
                      value={createForm.broadcast_message}
                      onChange={(e) => setCreateForm({ ...createForm, broadcast_message: e.target.value })}
                      placeholder="输入要群发的消息内容..."
                      rows={4}
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="broadcast-groups">群组ID（逗号分隔，可选）</Label>
                    <Input
                      id="broadcast-groups"
                      value={createForm.broadcast_groups}
                      onChange={(e) => setCreateForm({ ...createForm, broadcast_groups: e.target.value })}
                      placeholder="123456789, 987654321"
                    />
                    <p className="text-xs text-muted-foreground">
                      群组的数字ID，多个用逗号分隔，例如：-1001234567890
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="broadcast-channels">频道ID（逗号分隔，可选）</Label>
                    <Input
                      id="broadcast-channels"
                      value={createForm.broadcast_channels}
                      onChange={(e) => setCreateForm({ ...createForm, broadcast_channels: e.target.value })}
                      placeholder="123456789, 987654321"
                    />
                    <p className="text-xs text-muted-foreground">
                      频道的数字ID，多个用逗号分隔，例如：-1001234567890
                    </p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="broadcast-delay">发送间隔（秒，可选）</Label>
                    <Input
                      id="broadcast-delay"
                      type="number"
                      min="0"
                      value={createForm.broadcast_delay}
                      onChange={(e) => setCreateForm({ ...createForm, broadcast_delay: e.target.value })}
                      placeholder="0"
                    />
                  </div>
                </>
              )}

              {createForm.task_type === "verify_code" && (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="verify-timeout">超时时间（秒）</Label>
                    <Input
                      id="verify-timeout"
                      type="number"
                      min="1"
                      value={createForm.verify_timeout}
                      onChange={(e) => setCreateForm({ ...createForm, verify_timeout: e.target.value })}
                      placeholder="30"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="verify-source">来源过滤（可选）</Label>
                    <Input
                      id="verify-source"
                      value={createForm.verify_source}
                      onChange={(e) => setCreateForm({ ...createForm, verify_source: e.target.value })}
                      placeholder="Telegram"
                    />
                    <p className="text-xs text-muted-foreground">过滤特定来源的验证码</p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="verify-pattern">匹配模式（可选）</Label>
                    <Input
                      id="verify-pattern"
                      value={createForm.verify_pattern}
                      onChange={(e) => setCreateForm({ ...createForm, verify_pattern: e.target.value })}
                      placeholder="\\d{6}"
                    />
                    <p className="text-xs text-muted-foreground">正则表达式，用于匹配验证码格式</p>
                  </div>
                </>
              )}

              {createForm.task_type === "group_chat" && (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="group-chat-group-id">群组ID *</Label>
                    <Input
                      id="group-chat-group-id"
                      type="number"
                      value={createForm.group_chat_group_id}
                      onChange={(e) => setCreateForm({ ...createForm, group_chat_group_id: e.target.value })}
                      placeholder="123456789"
                      required
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="group-chat-duration">持续时间（分钟，可选）</Label>
                    <Input
                      id="group-chat-duration"
                      type="number"
                      min="1"
                      value={createForm.group_chat_duration}
                      onChange={(e) => setCreateForm({ ...createForm, group_chat_duration: e.target.value })}
                      placeholder="60"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="group-chat-ai-config">AI配置（JSON，可选）</Label>
                    <Textarea
                      id="group-chat-ai-config"
                      value={createForm.group_chat_ai_config}
                      onChange={(e) => setCreateForm({ ...createForm, group_chat_ai_config: e.target.value })}
                      placeholder='{"personality": "friendly", "response_rate": 0.3, "keywords": ["hello", "question"]}'
                      rows={4}
                      className="font-mono text-xs"
                    />
                  </div>
                </>
              )}

              {createForm.task_type === "check" && (
                <>
                  <div className="space-y-2">
                    <Label htmlFor="check-timeout">超时时间（可选）</Label>
                    <Input
                      id="check-timeout"
                      value={createForm.check_timeout}
                      onChange={(e) => setCreateForm({ ...createForm, check_timeout: e.target.value })}
                      placeholder="2m"
                    />
                    <p className="text-xs text-muted-foreground">例如：30s, 2m, 5m</p>
                  </div>
                  <div className="bg-muted/50 rounded-lg p-3 text-xs text-muted-foreground">
                    <p>账号检查任务将自动检查账号的状态和健康度。</p>
                  </div>
                </>
              )}

              {!createForm.task_type && (
                <div className="text-sm text-muted-foreground py-2">
                  请先选择任务类型以显示配置表单
                </div>
              )}
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleSaveCreateTask}>创建</Button>
            </div>
          </DialogContent>
        </Dialog>

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
      </div>
    </MainLayout>
  )
}

