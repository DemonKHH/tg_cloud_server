"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Checkbox } from "@/components/ui/checkbox"
import { Plus, TestTube, CheckCircle2, XCircle, AlertCircle, Pencil, Trash2, MoreVertical, Upload } from "lucide-react"
import { proxyAPI } from "@/lib/api"
import { useState } from "react"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { cn } from "@/lib/utils"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
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
import { usePagination } from "@/hooks/use-pagination"
import { motion, AnimatePresence } from "framer-motion"
import { PageHeader } from "@/components/common/page-header"
import { FilterBar } from "@/components/common/filter-bar"

export default function ProxiesPage() {
  const {
    data: proxies,
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
    fetchFn: proxyAPI.list,
    initialFilters: { status: "" },
  })

  const statusFilter = filters.status || ""

  // 添加/编辑代理相关状态
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editingProxy, setEditingProxy] = useState<any>(null)
  const [form, setForm] = useState({
    name: "",
    ip: "",
    port: "",
    protocol: "http",
    username: "",
    password: "",
    country: "",
  })

  // 批量添加相关状态
  const [batchDialogOpen, setBatchDialogOpen] = useState(false)
  const [batchInput, setBatchInput] = useState("")
  const [batchProtocol, setBatchProtocol] = useState("http")

  // 批量操作状态
  const [selectedProxies, setSelectedProxies] = useState<number[]>([])
  const [batchActionLoading, setBatchActionLoading] = useState(false)

  // 测试代理相关状态
  // 测试代理相关状态
  const [testingProxy, setTestingProxy] = useState<string | null>(null)

  // 删除确认对话框状态
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deletingProxy, setDeletingProxy] = useState<any>(null)

  // 批量删除确认对话框状态
  const [batchDeleteDialogOpen, setBatchDeleteDialogOpen] = useState(false)

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />
      case "inactive":
        return <XCircle className="h-4 w-4 text-gray-500" />
      case "error":
        return <AlertCircle className="h-4 w-4 text-red-500" />
      case "testing":
        return <TestTube className="h-4 w-4 text-blue-500" />
      case "untested":
      default:
        return <AlertCircle className="h-4 w-4 text-yellow-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "active":
        return "bg-green-50 text-green-700 border border-green-200 dark:bg-green-900 dark:text-green-300 dark:border-green-800"
      case "inactive":
        return "bg-gray-50 text-gray-700 border border-gray-200 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-800"
      case "error":
        return "bg-red-50 text-red-700 border border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800"
      case "testing":
        return "bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800"
      case "untested":
      default:
        return "bg-yellow-50 text-yellow-700 border border-yellow-200 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800"
    }
  }

  const getStatusText = (status: string) => {
    const statusMap: Record<string, string> = {
      active: "活跃",
      inactive: "未激活",
      error: "错误",
      testing: "测试中",
      untested: "未测试",
    }
    return statusMap[status] || status
  }

  const getProtocolText = (protocol: string) => {
    const protocolMap: Record<string, string> = {
      http: "HTTP",
      https: "HTTPS",
      socks5: "SOCKS5",
    }
    return protocolMap[protocol] || protocol.toUpperCase()
  }

  // 添加代理
  const handleAddProxy = () => {
    setEditingProxy(null)
    setForm({
      name: "",
      ip: "",
      port: "",
      protocol: "http",
      username: "",
      password: "",
      country: "",
    })
    setEditDialogOpen(true)
  }

  // 编辑代理
  const handleEditProxy = (proxy: any) => {
    setEditingProxy(proxy)
    setForm({
      name: proxy.name || "",
      ip: proxy.ip || "",
      port: String(proxy.port || ""),
      protocol: proxy.protocol || "http",
      username: proxy.username || "",
      password: "", // 不显示密码
      country: proxy.country || "",
    })
    setEditDialogOpen(true)
  }

  // 保存代理
  const handleSaveProxy = async () => {
    if (!form.ip || !form.port) {
      toast.error("请填写IP和端口")
      return
    }

    try {
      const port = parseInt(form.port)
      if (isNaN(port) || port < 1 || port > 65535) {
        toast.error("端口号必须在1-65535之间")
        return
      }

      const data: any = {
        name: form.name || undefined,
        ip: form.ip,
        port: port,
        protocol: form.protocol,
        username: form.username || undefined,
        password: form.password || undefined,
        country: form.country || undefined,
      }

      if (editingProxy) {
        const res = await proxyAPI.update(String(editingProxy.id), data)
        if (res.code === 0) {
          toast.success("代理更新成功")
          setEditDialogOpen(false)
          refresh()
        } else {
          toast.error(res.msg || "更新代理失败")
        }
      } else {
        const res = await proxyAPI.create(data)
        if (res.code === 0) {
          toast.success("代理添加成功")
          setEditDialogOpen(false)
          refresh()
        } else {
          toast.error(res.msg || "添加代理失败")
        }
      }
    } catch (error: any) {
      toast.error(error instanceof Error ? error.message : (editingProxy ? "更新代理失败" : "添加代理失败"))
    }
  }

  // 删除代理
  // 删除代理 - 打开确认对话框
  const handleDeleteProxy = (proxy: any) => {
    setDeletingProxy(proxy)
    setDeleteDialogOpen(true)
  }

  // 确认删除代理
  const confirmDeleteProxy = async () => {
    if (!deletingProxy) return

    try {
      const res = await proxyAPI.delete(String(deletingProxy.id))
      if (res.code === 0) {
        toast.success("代理删除成功")
        refresh()
        setDeleteDialogOpen(false)
        setDeletingProxy(null)
      } else {
        toast.error(res.msg || "删除代理失败")
      }
    } catch (error: any) {
      toast.error(error instanceof Error ? error.message : "删除代理失败")
    }
  }

  // 测试代理
  const handleTestProxy = async (proxy: any) => {
    try {
      setTestingProxy(proxy.id)
      const response = await proxyAPI.test(String(proxy.id))
      if (response.code === 0 && response.data) {
        const result = response.data as any
        if (result.success) {
          toast.success(`代理测试成功，延迟: ${result.latency_ms}ms`)
        } else {
          toast.error(`代理测试失败: ${result.error || "未知错误"}`)
        }
      } else {
        toast.error(response.msg || "代理测试失败")
      }
      refresh()
    } catch (error: any) {
      toast.error(error.message || "代理测试失败")
    } finally {
      setTestingProxy(null)
    }
  }


  // 批量添加处理
  const handleBatchSubmit = async () => {
    if (!batchInput.trim()) {
      toast.error("请输入代理列表")
      return
    }

    const lines = batchInput.trim().split('\n')
    const proxies: any[] = []
    const errors: string[] = []

    lines.forEach((line, index) => {
      const trimmed = line.trim()
      if (!trimmed) return

      // ip:port:user:pass 或 ip:port
      let ip, port, username, password
      const parts = trimmed.split(':')
      if (parts.length >= 2) {
        ip = parts[0].trim()
        port = parts[1].trim()
        if (parts.length >= 4) {
          username = parts[2].trim()
          password = parts.slice(3).join(':').trim()
        }
      } else {
        errors.push(`第 ${index + 1} 行格式错误`)
        return
      }

      const portNum = parseInt(port || "")
      if (!ip || isNaN(portNum)) {
        errors.push(`第 ${index + 1} 行IP或端口无效`)
        return
      }

      proxies.push({
        name: `Batch Import ${new Date().toLocaleDateString()}`,
        ip,
        port: portNum,
        protocol: batchProtocol,
        username: username || "",
        password: password || "",
        country: "",
      })
    })

    if (errors.length > 0) {
      toast.error(`解析出错:\n${errors.slice(0, 5).join('\n')}${errors.length > 5 ? '...' : ''}`)
      if (proxies.length === 0) return
    }

    try {
      const res = await proxyAPI.batchCreate({ proxies })
      if (res.code === 0) {
        toast.success(`成功添加 ${proxies.length} 个代理`)
        setBatchDialogOpen(false)
        setBatchInput("")
        refresh()
      } else {
        toast.error(res.msg || "批量添加失败")
      }
    } catch (error: any) {
      toast.error(error instanceof Error ? error.message : "批量添加失败")
    }
  }

  // 批量操作处理
  const toggleSelectAll = () => {
    if (selectedProxies.length === proxies.length) {
      setSelectedProxies([])
    } else {
      setSelectedProxies(proxies.map(p => p.id))
    }
  }

  const toggleSelect = (id: number) => {
    if (selectedProxies.includes(id)) {
      setSelectedProxies(selectedProxies.filter(p => p !== id))
    } else {
      setSelectedProxies([...selectedProxies, id])
    }
  }

  const handleBatchDelete = () => {
    if (selectedProxies.length === 0) return
    setBatchDeleteDialogOpen(true)
  }

  const confirmBatchDelete = async () => {
    setBatchActionLoading(true)
    try {
      const res = await proxyAPI.batchDelete(selectedProxies)
      if (res.code === 0) {
        toast.success("批量删除成功")
        setSelectedProxies([])
        refresh()
        setBatchDeleteDialogOpen(false)
      } else {
        toast.error(res.msg || "批量删除失败")
      }
    } catch (error: any) {
      toast.error(error instanceof Error ? error.message : "批量删除失败")
    } finally {
      setBatchActionLoading(false)
    }
  }

  const handleBatchTest = async () => {
    if (selectedProxies.length === 0) return

    setBatchActionLoading(true)
    try {
      const res = await proxyAPI.batchTest(selectedProxies)
      if (res.code === 0) {
        toast.success("批量测试完成")
        // 刷新列表以显示最新状态
        refresh()
      } else {
        toast.error(res.msg || "批量测试失败")
      }
    } catch (error: any) {
      toast.error(error instanceof Error ? error.message : "批量测试失败")
    } finally {
      setBatchActionLoading(false)
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <PageHeader
          title="代理管理"
          description="管理和测试您的代理IP配置"
          actions={

            <div className="flex gap-2">
              <Button variant="outline" onClick={() => setBatchDialogOpen(true)}>
                <Upload className="h-4 w-4 mr-2" />
                批量导入
              </Button>
              <Button onClick={handleAddProxy}>
                <Plus className="h-4 w-4 mr-2" />
                添加代理
              </Button>
            </div>
          }
        />

        {/* Batch Actions Floating Bar */}
        <AnimatePresence>
          {selectedProxies.length > 0 && (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 20 }}
              className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 bg-background/95 backdrop-blur-lg border shadow-2xl rounded-2xl p-4 flex flex-col gap-3 min-w-[400px]"
            >
              <div className="flex items-center justify-between border-b pb-2">
                <div className="flex items-center gap-2">
                  <div className="bg-primary text-primary-foreground text-xs font-bold rounded-full w-6 h-6 flex items-center justify-center">
                    {selectedProxies.length}
                  </div>
                  <span className="text-sm font-medium text-muted-foreground">
                    已选择代理
                  </span>
                </div>
                <Button
                  size="sm"
                  variant="ghost"
                  className="h-6 text-xs text-muted-foreground hover:text-foreground px-2"
                  onClick={() => setSelectedProxies([])}
                >
                  取消选择
                </Button>
              </div>

              <div className="flex items-center gap-2 justify-center">
                <Button
                  variant="destructive"
                  onClick={handleBatchDelete}
                  disabled={batchActionLoading}
                  className="flex-1"
                >
                  <Trash2 className="h-4 w-4 mr-2" />
                  删除 ({selectedProxies.length})
                </Button>
                <Button
                  variant="secondary"
                  onClick={handleBatchTest}
                  disabled={batchActionLoading}
                  className="flex-1"
                >
                  <TestTube className="h-4 w-4 mr-2" />
                  测试 ({selectedProxies.length})
                </Button>
              </div>
            </motion.div>
          )}
        </AnimatePresence>

        {/* Filters */}
        <FilterBar
          search={search}
          onSearchChange={setSearch}
          searchPlaceholder="搜索主机、IP..."
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
                <SelectItem value="active">活跃</SelectItem>
                <SelectItem value="inactive">未激活</SelectItem>
                <SelectItem value="error">错误</SelectItem>
                <SelectItem value="testing">测试中</SelectItem>
                <SelectItem value="untested">未测试</SelectItem>
              </SelectContent>
            </Select>
          }
        />

        {/* 代理数据表 */}
        <div className="rounded-md border">
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead className="w-[50px]">
                  <Checkbox
                    checked={proxies.length > 0 && selectedProxies.length === proxies.length}
                    onCheckedChange={toggleSelectAll}
                    className="border-2 border-primary data-[state=checked]:bg-primary data-[state=checked]:border-primary"
                  />
                </TableHead>
                <TableHead className="w-[200px]">代理地址</TableHead>
                <TableHead className="w-[100px]">协议</TableHead>
                <TableHead className="w-[120px]">状态</TableHead>
                <TableHead className="w-[120px]">成功率</TableHead>
                <TableHead className="w-[120px]">平均延迟</TableHead>
                <TableHead className="w-[120px]">认证</TableHead>
                <TableHead className="w-[100px]">国家</TableHead>
                <TableHead className="w-[180px]">最后测试</TableHead>
                <TableHead className="w-[140px]">操作</TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {loading ? (
                // 加载状态
                Array.from({ length: 5 }).map((_, index) => (
                  <TableRow key={index}>
                    <TableCell><div className="h-4 w-4 bg-muted rounded animate-pulse" /></TableCell>
                    <TableCell><div className="h-4 bg-muted rounded animate-pulse" /></TableCell>
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
              ) : proxies.length === 0 ? (
                <TableRow>
                  <TableCell colSpan={10} className="h-24 text-center">
                    暂无代理数据
                  </TableCell>
                </TableRow>
              ) : (
                proxies.map((record) => (
                  <TableRow
                    key={record.id}
                    className={cn(
                      "group transition-colors hover:bg-muted/50",
                      selectedProxies.includes(record.id) && "bg-primary/5"
                    )}
                  >
                    <TableCell>
                      <Checkbox
                        checked={selectedProxies.includes(record.id)}
                        onCheckedChange={() => toggleSelect(record.id)}
                        className="border-2 border-primary/60 data-[state=checked]:bg-primary data-[state=checked]:border-primary hover:border-primary"
                      />
                    </TableCell>
                    <TableCell>
                      <div className="space-y-1">
                        <div className="font-medium">{record.ip}:{record.port}</div>
                        <div className="text-xs text-muted-foreground">
                          {record.name}
                        </div>
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className="text-xs">
                        {getProtocolText(record.protocol)}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="flex items-center gap-2">
                        {getStatusIcon(record.status)}
                        <Badge
                          variant={record.status === 'active' ? 'default' : record.status === 'error' ? 'destructive' : 'secondary'}
                          className={cn("text-xs", getStatusColor(record.status))}
                        >
                          {getStatusText(record.status)}
                        </Badge>
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        {record.success_rate ? `${record.success_rate.toFixed(1)}%` : '-'}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm">
                        {record.avg_latency ? `${record.avg_latency}ms` : '-'}
                      </div>
                    </TableCell>
                    <TableCell>
                      <Badge variant={record.username ? 'default' : 'secondary'} className="text-xs">
                        {record.username ? '已认证' : '无需认证'}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm text-muted-foreground">
                        {record.country || '-'}
                      </div>
                    </TableCell>
                    <TableCell>
                      <div className="text-sm text-muted-foreground">
                        {record.last_test_at ? new Date(record.last_test_at).toLocaleString() : '从未测试'}
                      </div>
                    </TableCell>
                    <TableCell>
                      <TooltipProvider>
                        <div className="flex items-center gap-1">
                          {/* 测试代理 */}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className={cn(
                                  "h-8 w-8",
                                  testingProxy === record.id
                                    ? "opacity-50 cursor-not-allowed text-muted-foreground"
                                    : "hover:bg-orange-50 text-orange-600 hover:text-orange-700"
                                )}
                                disabled={testingProxy === record.id}
                                onClick={() => handleTestProxy(record)}
                              >
                                <TestTube className="h-4 w-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="text-xs">
                                {testingProxy === record.id ? "测试中..." : "测试代理连接"}
                              </p>
                            </TooltipContent>
                          </Tooltip>

                          {/* 编辑代理 */}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 hover:bg-blue-50 text-blue-600 hover:text-blue-700"
                                onClick={() => handleEditProxy(record)}
                              >
                                <Pencil className="h-4 w-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="text-xs">编辑代理配置</p>
                            </TooltipContent>
                          </Tooltip>

                          {/* 删除代理 */}
                          <Tooltip>
                            <TooltipTrigger asChild>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8 hover:bg-red-50 text-red-600 hover:text-red-700"
                                onClick={() => handleDeleteProxy(record)}
                              >
                                <Trash2 className="h-4 w-4" />
                              </Button>
                            </TooltipTrigger>
                            <TooltipContent side="top">
                              <p className="text-xs">删除代理 (不可恢复)</p>
                            </TooltipContent>
                          </Tooltip>
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
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            共 {total} 个代理，当前第 {page} 页
          </div>
          <div className="flex gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => Math.max(1, p - 1))}
              disabled={page === 1}
            >
              上一页
            </Button>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage((p) => p + 1)}
              disabled={page * 20 >= total}
            >
              下一页
            </Button>
          </div>
        </div>

        {/* 添加/编辑代理对话框 */}
        <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>{editingProxy ? "编辑代理" : "添加代理"}</DialogTitle>
              <DialogDescription>
                {editingProxy ? "更新代理配置信息" : "添加新的代理IP配置"}
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="proxy-name">名称/备注（可选）</Label>
                <Input
                  id="proxy-name"
                  value={form.name}
                  onChange={(e) => setForm({ ...form, name: e.target.value })}
                  placeholder="代理名称或备注"
                />
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="proxy-ip">IP地址 *</Label>
                  <Input
                    id="proxy-ip"
                    value={form.ip}
                    onChange={(e) => setForm({ ...form, ip: e.target.value })}
                    placeholder="192.168.1.1"
                    required
                  />
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="proxy-port">端口 *</Label>
                  <Input
                    id="proxy-port"
                    type="number"
                    min="1"
                    max="65535"
                    value={form.port}
                    onChange={(e) => setForm({ ...form, port: e.target.value })}
                    placeholder="8080"
                    required
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="proxy-protocol">协议 *</Label>
                  <Select
                    value={form.protocol}
                    onValueChange={(value) => setForm({ ...form, protocol: value })}
                  >
                    <SelectTrigger id="proxy-protocol">
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="http">HTTP</SelectItem>
                      <SelectItem value="https">HTTPS</SelectItem>
                      <SelectItem value="socks5">SOCKS5</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
              </div>
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label htmlFor="proxy-username">用户名（可选）</Label>
                  <Input
                    id="proxy-username"
                    value={form.username}
                    onChange={(e) => setForm({ ...form, username: e.target.value })}
                    placeholder="代理用户名"
                  />
                </div>
                <div className="space-y-2">
                  <Label htmlFor="proxy-password">密码（可选）</Label>
                  <Input
                    id="proxy-password"
                    type="password"
                    value={form.password}
                    onChange={(e) => setForm({ ...form, password: e.target.value })}
                    placeholder={editingProxy ? "留空则不修改" : "代理密码"}
                  />
                </div>
              </div>
              <div className="space-y-2">
                <Label htmlFor="proxy-country">国家代码（可选）</Label>
                <Input
                  id="proxy-country"
                  value={form.country}
                  onChange={(e) => setForm({ ...form, country: e.target.value })}
                  placeholder="US, CN, etc."
                  maxLength={10}
                />
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleSaveProxy}>
                {editingProxy ? "更新" : "添加"}
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 删除确认对话框 */}
        <Dialog open={deleteDialogOpen} onOpenChange={setDeleteDialogOpen}>
          <DialogContent className="sm:max-w-[400px]">
            <DialogHeader>
              <DialogTitle className="text-xl text-red-600 flex items-center gap-2">
                <AlertCircle className="h-5 w-5" />
                确认删除代理
              </DialogTitle>
              <DialogDescription className="pt-2">
                您确定要删除代理 <span className="font-semibold text-foreground">{deletingProxy?.ip}:{deletingProxy?.port}</span> 吗？
                <br />
                <span className="text-red-500 text-xs mt-2 block">
                  此操作将永久删除该代理配置，且不可恢复。
                </span>
              </DialogDescription>
            </DialogHeader>
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => setDeleteDialogOpen(false)} className="btn-modern">
                取消
              </Button>
              <Button
                variant="destructive"
                onClick={confirmDeleteProxy}
                className="btn-modern bg-red-600 hover:bg-red-700"
              >
                确认删除
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 批量添加对话框 */}
        <Dialog open={batchDialogOpen} onOpenChange={setBatchDialogOpen}>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>批量导入代理</DialogTitle>
              <DialogDescription>
                每行一个代理，支持格式：<br />
                1. IP:端口:用户名:密码<br />
                2. IP:端口
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label>默认协议</Label>
                <Select
                  value={batchProtocol}
                  onValueChange={setBatchProtocol}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="http">HTTP</SelectItem>
                    <SelectItem value="https">HTTPS</SelectItem>
                    <SelectItem value="socks5">SOCKS5</SelectItem>
                  </SelectContent>
                </Select>
              </div>
              <div className="space-y-2">
                <Label>代理列表</Label>
                <Textarea
                  value={batchInput}
                  onChange={(e) => setBatchInput(e.target.value)}
                  placeholder={`192.168.1.1:8080:user:pass\n192.168.1.2:8080`}
                  className="h-[300px] font-mono text-sm"
                />
                <p className="text-xs text-muted-foreground">
                  已输入 {batchInput.split('\n').filter(l => l.trim()).length} 行
                </p>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setBatchDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleBatchSubmit}>
                导入
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 批量删除确认对话框 */}
        <Dialog open={batchDeleteDialogOpen} onOpenChange={setBatchDeleteDialogOpen}>
          <DialogContent className="sm:max-w-[400px]">
            <DialogHeader>
              <DialogTitle className="text-xl text-red-600 flex items-center gap-2">
                <AlertCircle className="h-5 w-5" />
                确认批量删除
              </DialogTitle>
              <DialogDescription className="pt-2">
                您确定要删除选中的 <span className="font-semibold text-foreground">{selectedProxies.length}</span> 个代理吗？
                <br />
                <span className="text-red-500 text-xs mt-2 block">
                  此操作将永久删除这些代理配置，且不可恢复。
                </span>
              </DialogDescription>
            </DialogHeader>
            <div className="flex justify-end gap-2 pt-4">
              <Button variant="outline" onClick={() => setBatchDeleteDialogOpen(false)} className="btn-modern">
                取消
              </Button>
              <Button
                variant="destructive"
                onClick={confirmBatchDelete}
                disabled={batchActionLoading}
                className="btn-modern bg-red-600 hover:bg-red-700"
              >
                {batchActionLoading ? "删除中..." : "确认删除"}
              </Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </MainLayout >
  )
}
