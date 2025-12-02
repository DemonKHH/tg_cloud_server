"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { X, RefreshCw, CheckCircle2, Clock, PlayCircle, AlertCircle, Ban, FileText, MoreVertical, Pause, Play, Square, Trash2, Search, ChevronDown } from "lucide-react"
import { taskAPI } from "@/lib/api"
import { useState, useEffect } from "react"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
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
  // 取消任务 - 打开确认对话框
  const handleCancelTask = (task: any) => {
    setCancellingTask(task)
    setCancelDialogOpen(true)
  }

  // 确认取消任务
  const confirmCancelTask = async () => {
    if (!cancellingTask) return

    try {
      await taskAPI.cancel(String(cancellingTask.id))
      toast.success("任务已取消")
      refresh()
      setCancelDialogOpen(false)
      setCancellingTask(null)
    } catch (error: any) {
      console.error('取消任务失败:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "取消任务失败"
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
      await taskAPI.delete(String(deletingTask.id))
      toast.success("任务已删除")
      refresh()
      setDeleteDialogOpen(false)
      setDeletingTask(null)
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
      </div>
    </MainLayout>
  )
}

