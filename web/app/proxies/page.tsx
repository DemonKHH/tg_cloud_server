"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Plus, Search, TestTube, CheckCircle2, XCircle, AlertCircle, Pencil, Trash2, MoreVertical } from "lucide-react"
import { proxyAPI } from "@/lib/api"
import { useState, useEffect } from "react"
import { Badge } from "@/components/ui/badge"
import { ModernTable } from "@/components/ui/modern-table"
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
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { cn } from "@/lib/utils"

export default function ProxiesPage() {
  const [proxies, setProxies] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState("")
  const [statusFilter, setStatusFilter] = useState<string>("")
  
  // 添加/编辑代理相关状态
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editingProxy, setEditingProxy] = useState<any>(null)
  const [form, setForm] = useState({
    name: "",
    host: "",
    ip: "",
    port: "",
    protocol: "http",
    username: "",
    password: "",
    country: "",
  })
  
  // 测试代理相关状态
  const [testingProxy, setTestingProxy] = useState<string | null>(null)

  useEffect(() => {
    loadProxies()
  }, [page, statusFilter])

  // 搜索防抖
  useEffect(() => {
    const timer = setTimeout(() => {
      if (page === 1) {
        loadProxies()
      } else {
        setPage(1)
      }
    }, 500)

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

  const loadProxies = async () => {
    try {
      setLoading(true)
      const params: any = { page, limit: 20 }
      if (search) {
        params.search = search
      }
      if (statusFilter) {
        params.status = statusFilter
      }
      const response = await proxyAPI.list(params)
      if (response.data) {
        const data = response.data as any
        setProxies(data.items || [])
        setTotal(data.pagination?.total || 0)
        if (data.pagination?.current_page) {
          setPage(data.pagination.current_page)
        }
      }
    } catch (error) {
      toast.error("加载代理失败，请稍后重试")
      console.error("加载代理失败:", error)
    } finally {
      setLoading(false)
    }
  }

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
      host: "",
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
      host: proxy.host || "",
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
    if (!form.host || !form.ip || !form.port) {
      toast.error("请填写主机、IP和端口")
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
        host: form.host,
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
      loadProxies()
    } catch (error: any) {
      toast.error(error.message || (editingProxy ? "更新代理失败" : "添加代理失败"))
    }
  }

  // 删除代理
  const handleDeleteProxy = async (proxy: any) => {
    if (!confirm(`确定要删除代理 ${proxy.host}:${proxy.port} 吗？`)) {
      return
    }

    try {
      await proxyAPI.delete(String(proxy.id))
      toast.success("代理删除成功")
      loadProxies()
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
      loadProxies()
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
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">代理管理</h1>
            <p className="text-muted-foreground mt-1">
              管理和测试您的代理IP配置
            </p>
          </div>
          <Button onClick={handleAddProxy}>
            <Plus className="h-4 w-4 mr-2" />
            添加代理
          </Button>
        </div>

        {/* Filters */}
        <Card>
          <CardHeader>
            <div className="flex items-center gap-4">
              <div className="relative flex-1">
                <Search className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-muted-foreground" />
                <Input
                  type="search"
                  placeholder="搜索主机、IP..."
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
                  <SelectItem value="active">活跃</SelectItem>
                  <SelectItem value="inactive">未激活</SelectItem>
                  <SelectItem value="error">错误</SelectItem>
                  <SelectItem value="testing">测试中</SelectItem>
                  <SelectItem value="untested">未测试</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </CardHeader>
        </Card>

        {/* Proxies Table */}
        <ModernTable
          data={proxies}
          columns={[
            {
              key: 'host',
              title: '代理地址',
              width: '200px',
              render: (value, record) => (
                <div className="space-y-1">
                  <div className="font-medium">{value}:{record.port}</div>
                  <div className="text-xs text-muted-foreground">
                    {record.name || record.ip}
                  </div>
                </div>
              )
            },
            {
              key: 'protocol',
              title: '协议',
              width: '100px',
              render: (value) => (
                <Badge variant="secondary" className="text-xs">
                  {getProtocolText(value)}
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
                    variant={value === 'active' ? 'default' : value === 'error' ? 'destructive' : 'secondary'}
                    className={cn("text-xs", getStatusColor(value))}
                  >
                    {getStatusText(value)}
                  </Badge>
                </div>
              )
            },
            {
              key: 'success_rate',
              title: '成功率',
              width: '120px',
              sortable: true,
              render: (value) => (
                <div className="text-sm">
                  {value ? `${value.toFixed(1)}%` : '-'}
                </div>
              )
            },
            {
              key: 'avg_latency',
              title: '平均延迟',
              width: '120px',
              sortable: true,
              render: (value) => (
                <div className="text-sm">
                  {value ? `${value}ms` : '-'}
                </div>
              )
            },
            {
              key: 'username',
              title: '认证',
              width: '120px',
              render: (value) => (
                <Badge variant={value ? 'default' : 'secondary'} className="text-xs">
                  {value ? '已认证' : '无需认证'}
                </Badge>
              )
            },
            {
              key: 'country',
              title: '国家',
              width: '100px',
              render: (value) => (
                <div className="text-sm text-muted-foreground">
                  {value || '-'}
                </div>
              )
            },
            {
              key: 'last_test_at',
              title: '最后测试',
              width: '180px',
              sortable: true,
              render: (value) => (
                <div className="text-sm text-muted-foreground">
                  {value ? new Date(value).toLocaleString() : '从未测试'}
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
                    <DropdownMenuItem 
                      onClick={() => handleTestProxy(record)}
                      disabled={testingProxy === record.id}
                    >
                      <TestTube className="h-4 w-4 mr-2" />
                      {testingProxy === record.id ? "测试中..." : "测试代理"}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleEditProxy(record)}>
                      <Pencil className="h-4 w-4 mr-2" />
                      编辑代理
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem 
                      onClick={() => handleDeleteProxy(record)}
                      className="text-destructive"
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      删除代理
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )
            }
          ]}
          loading={loading}
          searchable
          searchPlaceholder="搜索主机、IP..."
          filterable
          emptyText="暂无代理数据"
          className="card-shadow"
        />

        {/* Pagination */}
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
                  <Label htmlFor="proxy-host">主机地址 *</Label>
                  <Input
                    id="proxy-host"
                    value={form.host}
                    onChange={(e) => setForm({ ...form, host: e.target.value })}
                    placeholder="example.com"
                    required
                  />
                </div>
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
      </div>
    </MainLayout>
  )
}
