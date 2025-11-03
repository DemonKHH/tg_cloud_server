"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, Search, X, RefreshCw, CheckCircle2, Clock, PlayCircle, AlertCircle, Ban, FileText, MoreVertical } from "lucide-react"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { cn } from "@/lib/utils"

export default function TasksPage() {
  const [tasks, setTasks] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<string>("")
  
  // 创建任务相关状态
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [createForm, setCreateForm] = useState({
    account_id: "",
    task_type: "",
    priority: "5",
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
    loadTasks()
  }, [page, statusFilter])

  // 搜索防抖
  useEffect(() => {
    const timer = setTimeout(() => {
      if (page === 1) {
        loadTasks()
      } else {
        setPage(1)
      }
    }, 500)

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

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
        setAccounts(data.items || [])
      }
    } catch (error) {
      console.error("加载账号失败:", error)
    } finally {
      setLoadingAccounts(false)
    }
  }

  const loadTasks = async () => {
    try {
      setLoading(true)
      const params: any = { page, limit: 20 }
      if (search) {
        params.search = search
      }
      if (statusFilter) {
        params.status = statusFilter
      }
      const response = await taskAPI.list(params)
      if (response.data) {
        const data = response.data as any
        setTasks(data.items || [])
        setTotal(data.pagination?.total || 0)
        if (data.pagination?.current_page) {
          setPage(data.pagination.current_page)
        }
      }
    } catch (error) {
      toast.error("加载任务失败，请稍后重试")
      console.error("加载任务失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const loadLogs = async (taskId: string) => {
    try {
      setLoadingLogs(true)
      const response = await taskAPI.getLogs(taskId)
      if (response.data) {
        setLogs(response.data as any[] || [])
      }
    } catch (error) {
      toast.error("加载日志失败")
      console.error("加载日志失败:", error)
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
      loadTasks()
    } catch (error: any) {
      toast.error(error.message || "取消任务失败")
    }
  }

  // 重试任务
  const handleRetryTask = async (task: any) => {
    try {
      await taskAPI.retry(String(task.id))
      toast.success("任务已重新执行")
      loadTasks()
    } catch (error: any) {
      toast.error(error.message || "重试任务失败")
    }
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

  // 构建任务配置
  const buildTaskConfig = () => {
    const config: any = {}
    
    switch (createForm.task_type) {
      case "private_message":
        if (!createForm.private_targets || !createForm.private_message) {
          toast.error("请填写目标用户和消息内容")
          return null
        }
        config.targets = createForm.private_targets.split(",").map(t => t.trim()).filter(t => t)
        config.message = createForm.private_message
        if (createForm.private_delay) {
          config.delay_between = parseInt(createForm.private_delay)
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
        if (createForm.broadcast_groups) {
          config.groups = createForm.broadcast_groups.split(",").map(g => parseInt(g.trim())).filter(g => !isNaN(g))
        }
        if (createForm.broadcast_channels) {
          config.channels = createForm.broadcast_channels.split(",").map(c => parseInt(c.trim())).filter(c => !isNaN(c))
        }
        if (createForm.broadcast_delay) {
          config.delay_between = parseInt(createForm.broadcast_delay)
        }
        break
        
      case "verify_code":
        if (createForm.verify_timeout) {
          config.timeout = parseInt(createForm.verify_timeout)
        }
        if (createForm.verify_source) {
          config.source = createForm.verify_source
        }
        if (createForm.verify_pattern) {
          config.pattern = createForm.verify_pattern
        }
        break
        
      case "group_chat":
        if (!createForm.group_chat_group_id) {
          toast.error("请填写群组ID")
          return null
        }
        config.group_id = parseInt(createForm.group_chat_group_id)
        if (isNaN(config.group_id)) {
          toast.error("群组ID必须是数字")
          return null
        }
        if (createForm.group_chat_duration) {
          config.duration = parseInt(createForm.group_chat_duration)
        }
        if (createForm.group_chat_ai_config && createForm.group_chat_ai_config.trim() !== "" && createForm.group_chat_ai_config !== "{}") {
          try {
            config.ai_config = JSON.parse(createForm.group_chat_ai_config)
          } catch (e) {
            toast.error("AI配置JSON格式错误")
            return null
          }
        }
        break
        
      case "check":
        // 账号检查任务可能不需要额外配置
        break
        
      default:
        // 对于未知类型，可以保持空配置
        break
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

      await taskAPI.create({
        account_id: parseInt(createForm.account_id),
        task_type: createForm.task_type,
        priority: parseInt(createForm.priority),
        task_config: config,
      })
      toast.success("任务创建成功")
      setCreateDialogOpen(false)
      loadTasks()
    } catch (error: any) {
      toast.error(error.message || "创建任务失败")
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">任务管理</h1>
            <p className="text-muted-foreground mt-1">
              查看和管理您的任务
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" onClick={loadTasks}>
              <RefreshCw className="h-4 w-4 mr-2" />
              刷新
            </Button>
            <Button onClick={handleCreateTask}>
              <Plus className="h-4 w-4 mr-2" />
              创建任务
            </Button>
          </div>
        </div>

        {/* Filters */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="search"
                  placeholder="搜索任务ID或账号ID..."
                  className="pl-9"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </div>
              <Select value={statusFilter || "all"} onValueChange={(value) => setStatusFilter(value === "all" ? "" : value)}>
                <SelectTrigger className="w-[180px]">
                  <SelectValue placeholder="筛选状态" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部状态</SelectItem>
                  <SelectItem value="pending">待执行</SelectItem>
                  <SelectItem value="queued">已排队</SelectItem>
                  <SelectItem value="running">执行中</SelectItem>
                  <SelectItem value="completed">已完成</SelectItem>
                  <SelectItem value="failed">失败</SelectItem>
                  <SelectItem value="cancelled">已取消</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardHeader>
        </Card>

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
                  {record.account?.phone || `ID: ${value}`}
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
              width: '120px',
              render: (_, record) => (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent className="glass-effect" align="end">
                    <DropdownMenuItem onClick={() => handleViewLogs(record)}>
                      <FileText className="h-4 w-4 mr-2" />
                      查看日志
                    </DropdownMenuItem>
                    {(record.status === 'pending' || record.status === 'queued') && (
                      <DropdownMenuItem onClick={() => handleCancelTask(record)}>
                        <X className="h-4 w-4 mr-2" />
                        取消任务
                      </DropdownMenuItem>
                    )}
                    {(record.status === 'failed' || record.status === 'cancelled') && (
                      <DropdownMenuItem onClick={() => handleRetryTask(record)}>
                        <RefreshCw className="h-4 w-4 mr-2" />
                        重试任务
                      </DropdownMenuItem>
                    )}
                  </DropdownMenuContent>
                </DropdownMenu>
              )
            }
          ]}
          loading={loading}
          searchable
          searchPlaceholder="搜索任务ID或账号ID..."
          filterable
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
                  onValueChange={(value) => setCreateForm({ ...createForm, task_type: value })}
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
                    <p className="text-xs text-muted-foreground">多个用户名或手机号，用逗号分隔</p>
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
                    <p className="text-xs text-muted-foreground">多个群组ID，用逗号分隔</p>
                  </div>
                  <div className="space-y-2">
                    <Label htmlFor="broadcast-channels">频道ID（逗号分隔，可选）</Label>
                    <Input
                      id="broadcast-channels"
                      value={createForm.broadcast_channels}
                      onChange={(e) => setCreateForm({ ...createForm, broadcast_channels: e.target.value })}
                      placeholder="123456789, 987654321"
                    />
                    <p className="text-xs text-muted-foreground">多个频道ID，用逗号分隔</p>
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
                      placeholder='{"persona": "casual", "max_length": 200}'
                      rows={4}
                      className="font-mono text-xs"
                    />
                  </div>
                </>
              )}

              {createForm.task_type === "check" && (
                <div className="text-sm text-muted-foreground py-2">
                  账号检查任务无需额外配置，将自动检查账号状态和健康度。
                </div>
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

