"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, TestTube, CheckCircle2, XCircle, AlertCircle, Pencil, Trash2, MoreVertical } from "lucide-react"
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
import { usePagination } from "@/hooks/use-pagination"
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

  // 测试代理相关状态
  // 测试代理相关状态
  const [testingProxy, setTestingProxy] = useState<string | null>(null)

  // 删除确认对话框状态
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false)
  const [deletingProxy, setDeletingProxy] = useState<any>(null)

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
        await proxyAPI.update(String(editingProxy.id), data)
        toast.success("代理更新成功")
      } else {
        await proxyAPI.create(data)
        toast.success("代理添加成功")
      }
      setEditDialogOpen(false)
      refresh()
    } catch (error: any) {
      toast.error(error.message || (editingProxy ? "更新代理失败" : "添加代理失败"))
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
      await proxyAPI.delete(String(deletingProxy.id))
      toast.success("代理删除成功")
      refresh()
      setDeleteDialogOpen(false)
      setDeletingProxy(null)
    } catch (error: any) {
      toast.error(error.message || "删除代理失败")
    }
  }

  // 测试代理
  const handleTestProxy = async (proxy: any) => {
    try {
      setTestingProxy(proxy.id)
      const response = await proxyAPI.test(String(proxy.id))
      if (response.data) {
        const result = response.data as any
        if (result.success) {
          toast.success(`代理测试成功，延迟: ${result.latency_ms}ms`)
        } else {
          toast.error(`代理测试失败: ${result.error || "未知错误"}`)
        }
      }
      refresh()
    } catch (error: any) {
      toast.error(error.message || "代理测试失败")
    } finally {
      setTestingProxy(null)
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
            <Button onClick={handleAddProxy}>
              <Plus className="h-4 w-4 mr-2" />
              添加代理
            </Button>
          }
        />

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
                  <TableCell colSpan={9} className="h-24 text-center">
                    暂无代理数据
                  </TableCell>
                </TableRow>
              ) : (
                proxies.map((record) => (
                  <TableRow key={record.id}>
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
      </div>
    </MainLayout>
  )
}
