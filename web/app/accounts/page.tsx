"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Plus, Search, Filter, MoreVertical, CheckCircle2, XCircle, AlertCircle, Upload, FileArchive } from "lucide-react"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Badge } from "@/components/ui/badge"
import { ModernTable } from "@/components/ui/modern-table"
import { motion } from "framer-motion"
import { cn } from "@/lib/utils"
import { accountAPI, proxyAPI } from "@/lib/api"
import { useState, useEffect, useRef, useCallback } from "react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Pencil, Trash2, Activity, Link2 } from "lucide-react"

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState("")
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false)
  const [uploading, setUploading] = useState(false)
  const [selectedProxy, setSelectedProxy] = useState<string>("")
  const [proxies, setProxies] = useState<any[]>([])
  const [loadingProxies, setLoadingProxies] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)
  
  // 编辑账号相关状态
  const [editDialogOpen, setEditDialogOpen] = useState(false)
  const [editingAccount, setEditingAccount] = useState<any>(null)
  const [editForm, setEditForm] = useState({ note: "", phone: "", session_data: "" })
  
  // 绑定代理相关状态
  const [bindProxyDialogOpen, setBindProxyDialogOpen] = useState(false)
  const [bindingAccount, setBindingAccount] = useState<any>(null)
  const [selectedBindProxy, setSelectedBindProxy] = useState<string>("")
  
  // 手动添加账号相关状态
  const [addDialogOpen, setAddDialogOpen] = useState(false)
  const [addForm, setAddForm] = useState({ phone: "", session_data: "", note: "", proxy_id: "" })
  
  // 健康检查相关状态
  const [healthChecking, setHealthChecking] = useState<string | null>(null)

  useEffect(() => {
    loadAccounts()
  }, [page])

  useEffect(() => {
    if (uploadDialogOpen) {
      loadProxies()
    }
  }, [uploadDialogOpen])

  // 搜索防抖
  useEffect(() => {
    const timer = setTimeout(() => {
      if (page === 1) {
        loadAccounts()
      } else {
        setPage(1) // 重置到第一页，触发loadAccounts
      }
    }, 500)

    return () => clearTimeout(timer)
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [search])

  useEffect(() => {
    if (bindProxyDialogOpen || editDialogOpen) {
      loadProxies()
    }
  }, [bindProxyDialogOpen, editDialogOpen])

  const loadProxies = async () => {
    try {
      setLoadingProxies(true)
      // 获取所有代理，前端过滤活跃的
      const response = await proxyAPI.list({ page: 1, limit: 100 })
      if (response.data) {
        const data = response.data as any
        // 过滤出活跃状态的代理
        const activeProxies = (data.items || []).filter(
          (proxy: any) => proxy.status === 'active' || proxy.is_active === true
        )
        setProxies(activeProxies)
      }
    } catch (error) {
      console.error("加载代理失败:", error)
      // 不显示错误提示，因为代理是可选的
    } finally {
      setLoadingProxies(false)
    }
  }

  const loadAccounts = async () => {
    try {
      setLoading(true)
      const params: any = { page, limit: 20 }
      if (search) {
        params.search = search
      }
      const response = await accountAPI.list(params)
      if (response.data) {
        // 后端返回格式：{ items: [], pagination: { current_page, per_page, total, total_pages, has_next, has_prev } }
        const data = response.data as any
        setAccounts(data.items || [])
        setTotal(data.pagination?.total || 0)
        // 更新页码（后端返回的current_page）
        if (data.pagination?.current_page) {
          setPage(data.pagination.current_page)
        }
      }
    } catch (error) {
      toast.error("加载账号失败，请稍后重试")
      console.error("加载账号失败:", error)
    } finally {
      setLoading(false)
    }
  }

  // 根据后端账号状态枚举映射图标和颜色
  const getStatusIcon = (status: string) => {
    switch (status) {
      case "normal":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />
      case "warning":
        return <AlertCircle className="h-4 w-4 text-yellow-500" />
      case "restricted":
        return <XCircle className="h-4 w-4 text-orange-500" />
      case "dead":
        return <XCircle className="h-4 w-4 text-red-500" />
      case "cooling":
        return <AlertCircle className="h-4 w-4 text-blue-500" />
      case "maintenance":
        return <AlertCircle className="h-4 w-4 text-gray-500" />
      case "new":
      default:
        return <AlertCircle className="h-4 w-4 text-purple-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "normal":
        return "bg-green-50 text-green-700 border border-green-200 dark:bg-green-900 dark:text-green-300 dark:border-green-800"
      case "warning":
        return "bg-yellow-50 text-yellow-700 border border-yellow-200 dark:bg-yellow-900 dark:text-yellow-300 dark:border-yellow-800"
      case "restricted":
        return "bg-orange-50 text-orange-700 border border-orange-200 dark:bg-orange-900 dark:text-orange-300 dark:border-orange-800"
      case "dead":
        return "bg-red-50 text-red-700 border border-red-200 dark:bg-red-900 dark:text-red-300 dark:border-red-800"
      case "cooling":
        return "bg-blue-50 text-blue-700 border border-blue-200 dark:bg-blue-900 dark:text-blue-300 dark:border-blue-800"
      case "maintenance":
        return "bg-gray-50 text-gray-700 border border-gray-200 dark:bg-gray-900 dark:text-gray-300 dark:border-gray-800"
      case "new":
      default:
        return "bg-purple-50 text-purple-700 border border-purple-200 dark:bg-purple-900 dark:text-purple-300 dark:border-purple-800"
    }
  }

  const getStatusText = (status: string) => {
    const statusMap: Record<string, string> = {
      new: "新建",
      normal: "正常",
      warning: "警告",
      restricted: "限制",
      dead: "死亡",
      cooling: "冷却",
      maintenance: "维护",
    }
    return statusMap[status] || status
  }

  // 编辑账号
  const handleEditAccount = (account: any) => {
    setEditingAccount(account)
    setEditForm({
      note: account.note || "",
      phone: account.phone || "",
      session_data: account.session_data || "",
    })
    setEditDialogOpen(true)
  }

  const handleSaveEdit = async () => {
    if (!editingAccount) return
    
    try {
      await accountAPI.update(String(editingAccount.id), {
        note: editForm.note,
        // session_data 通常不允许修改，除非是特殊情况
      })
      toast.success("账号更新成功")
      setEditDialogOpen(false)
      loadAccounts()
    } catch (error: any) {
      toast.error(error.message || "更新账号失败")
    }
  }

  // 删除账号
  const handleDeleteAccount = async (account: any) => {
    if (!confirm(`确定要删除账号 ${account.phone} 吗？此操作不可恢复。`)) {
      return
    }

    try {
      await accountAPI.delete(String(account.id))
      toast.success("账号删除成功")
      loadAccounts()
    } catch (error: any) {
      toast.error(error.message || "删除账号失败")
    }
  }

  // 健康检查
  const handleCheckHealth = async (account: any) => {
    try {
      setHealthChecking(account.id)
      const response = await accountAPI.checkHealth(String(account.id))
      if (response.data) {
        const health = response.data as any
        toast.success(`健康检查完成：健康度 ${((health.score || 0) * 100).toFixed(0)}%`)
        loadAccounts() // 重新加载以更新健康度
      }
    } catch (error: any) {
      toast.error(error.message || "健康检查失败")
    } finally {
      setHealthChecking(null)
    }
  }

  // 绑定代理
  const handleBindProxy = (account: any) => {
    setBindingAccount(account)
    setSelectedBindProxy(account.proxy_id ? String(account.proxy_id) : "")
    setBindProxyDialogOpen(true)
  }

  const handleSaveBindProxy = async () => {
    if (!bindingAccount) return

    try {
      const proxyId = selectedBindProxy ? parseInt(selectedBindProxy) : undefined
      await accountAPI.bindProxy(String(bindingAccount.id), proxyId)
      toast.success(proxyId ? "代理绑定成功" : "代理解绑成功")
      setBindProxyDialogOpen(false)
      loadAccounts()
    } catch (error: any) {
      toast.error(error.message || "代理绑定失败")
    }
  }

  // 手动添加账号
  const handleAddAccount = () => {
    setAddForm({ phone: "", session_data: "", note: "", proxy_id: "" })
    setAddDialogOpen(true)
  }

  const handleSaveAdd = async () => {
    if (!addForm.phone || !addForm.session_data) {
      toast.error("请填写手机号和Session数据")
      return
    }

    try {
      const data: any = {
        phone: addForm.phone,
        session_data: addForm.session_data,
      }
      if (addForm.note) {
        data.note = addForm.note
      }
      if (addForm.proxy_id) {
        data.proxy_id = parseInt(addForm.proxy_id)
      }
      await accountAPI.create(data)
      toast.success("账号添加成功")
      setAddDialogOpen(false)
      loadAccounts()
    } catch (error: any) {
      toast.error(error.message || "添加账号失败")
    }
  }

  const handleFileUpload = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0]
    if (!file) return

    // 验证文件类型
    const fileName = file.name.toLowerCase()
    const isValidType = 
      fileName.endsWith('.zip') ||
      fileName.endsWith('.session') ||
      fileName.includes('tdata')

    if (!isValidType) {
      toast.error("不支持的文件类型，请上传 .zip、.session 文件或 tdata 文件夹")
      return
    }

    // 验证文件大小（100MB限制）
    if (file.size > 100 * 1024 * 1024) {
      toast.error("文件大小超过100MB限制")
      return
    }

    try {
      setUploading(true)
      
      // 如果选择了代理，传递代理ID
      const proxyId = selectedProxy ? parseInt(selectedProxy) : undefined
      const response = await accountAPI.uploadFiles(file, proxyId)
      
      if (response.data) {
        const data = response.data as any
        const created = data.created || 0
        const failed = data.failed || 0
        const total = data.total || 0

        if (created > 0) {
          toast.success(`成功创建 ${created} 个账号${failed > 0 ? `，失败 ${failed} 个` : ''}`)
          
          // 如果有错误信息，显示详细信息（最多显示前3个）
          if (failed > 0 && data.errors && data.errors.length > 0) {
            const errorMsg = data.errors.slice(0, 3).join('; ')
            if (data.errors.length > 3) {
              toast.warning(`${errorMsg}... (共 ${data.errors.length} 个错误)`)
            } else {
              toast.warning(`部分账号创建失败: ${errorMsg}`)
            }
            console.warn("创建账号时的错误：", data.errors)
          }
          
          setUploadDialogOpen(false)
          setSelectedProxy("") // 重置代理选择
          loadAccounts() // 重新加载账号列表
        } else {
          // 所有账号都创建失败
          const errorMsg = data.errors?.length > 0 
            ? data.errors.slice(0, 3).join('; ')
            : '未知错误'
          toast.error(`未能创建任何账号。${errorMsg}${data.errors?.length > 3 ? '...' : ''}`)
        }
      } else {
        toast.success("文件上传成功")
        setUploadDialogOpen(false)
        setSelectedProxy("") // 重置代理选择
        loadAccounts()
      }
    } catch (error: any) {
      console.error("上传账号文件失败:", error)
      const errorMsg = error.message || "上传账号文件失败"
      toast.error(errorMsg)
    } finally {
      setUploading(false)
      // 清空文件输入
      if (fileInputRef.current) {
        fileInputRef.current.value = ''
      }
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight">账号管理</h1>
            <p className="text-muted-foreground mt-1">
              管理和监控您的TG账号
            </p>
          </div>
          <div className="flex gap-2">
            <Dialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
              <DialogTrigger asChild>
                <Button variant="outline">
                  <Upload className="h-4 w-4 mr-2" />
                  上传账号文件
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-[500px]">
                <DialogHeader>
                  <DialogTitle>上传账号文件</DialogTitle>
                  <DialogDescription>
                    支持上传 .zip、.session 文件或 tdata 文件夹。系统将自动解析 Session/TData 格式并转换为 SessionString。
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  {/* 文件上传区域 */}
                  <div className="border-2 border-dashed border-muted-foreground/25 rounded-lg p-8 text-center">
                    <FileArchive className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                    <p className="text-sm text-muted-foreground mb-2 font-medium">
                      支持的文件格式：
                    </p>
                    <ul className="text-xs text-muted-foreground space-y-1 mb-4 text-left max-w-xs mx-auto">
                      <li>• .zip 压缩包（可包含多个账号文件）</li>
                      <li>• .session 文件（Pyrogram 格式）</li>
                      <li>• tdata 文件夹（Telegram Desktop 格式）</li>
                      <li>• gotd/td 格式 session 文件</li>
                    </ul>
                    <Input
                      ref={fileInputRef}
                      type="file"
                      className="hidden"
                      id="account-file-upload"
                      accept=".zip,.session"
                      onChange={handleFileUpload}
                      disabled={uploading}
                    />
                    <Button
                      variant="outline"
                      onClick={() => fileInputRef.current?.click()}
                      disabled={uploading}
                    >
                      {uploading ? "上传中..." : "选择文件"}
                    </Button>
                    {uploading && (
                      <p className="text-sm text-muted-foreground mt-2">
                        正在解析文件并转换格式，请稍候...
                      </p>
                    )}
                  </div>

                  {/* 代理选择（可选） */}
                  <div className="space-y-2">
                    <Label htmlFor="proxy-select">选择代理（可选）</Label>
                    <Select
                      value={selectedProxy || "none"}
                      onValueChange={(value) => setSelectedProxy(value === "none" ? "" : value)}
                      disabled={uploading || loadingProxies}
                    >
                      <SelectTrigger id="proxy-select">
                        <SelectValue placeholder={loadingProxies ? "加载中..." : "不绑定代理"} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">不绑定代理</SelectItem>
                        {proxies.map((proxy) => (
                          <SelectItem key={proxy.id} value={String(proxy.id)}>
                            {proxy.host}:{proxy.port} {proxy.username && `(${proxy.username})`}
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                    {proxies.length === 0 && !loadingProxies && (
                      <p className="text-xs text-muted-foreground">
                        暂无可用代理，可以在代理管理中添加
                      </p>
                    )}
                  </div>

                  {/* 提示信息 */}
                  <div className="bg-muted/50 rounded-lg p-3 text-xs text-muted-foreground">
                    <p className="font-medium mb-1">提示：</p>
                    <ul className="space-y-1 list-disc list-inside">
                      <li>系统会自动识别文件格式并转换为 SessionString</li>
                      <li>如果文件包含手机号信息，会自动提取</li>
                      <li>单个文件最大支持 100MB</li>
                    </ul>
                  </div>
                </div>
              </DialogContent>
            </Dialog>
            <Button onClick={handleAddAccount}>
              <Plus className="h-4 w-4 mr-2" />
              手动添加
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
                  placeholder="搜索手机号或备注..."
                  className="pl-9"
                  value={search}
                  onChange={(e) => setSearch(e.target.value)}
                />
              </div>
              <Button variant="outline">
                <Filter className="h-4 w-4 mr-2" />
                筛选
              </Button>
            </div>
          </CardHeader>
        </Card>

        {/* Modern Accounts Table */}
        <ModernTable
          data={accounts}
          columns={[
            {
              key: 'phone',
              title: '账号信息',
              width: '200px',
              render: (value, record) => (
                <div className="space-y-1">
                  <div className="font-medium">{value}</div>
                  {record.note && (
                    <div className="text-sm text-muted-foreground">{record.note}</div>
                  )}
                </div>
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
                    variant={value === 'normal' ? 'default' : value === 'dead' || value === 'restricted' ? 'destructive' : 'secondary'}
                    className={cn("text-xs", getStatusColor(value))}
                  >
                    {getStatusText(value)}
                  </Badge>
                </div>
              )
            },
            {
              key: 'health_score',
              title: '健康度',
              width: '150px',
              sortable: true,
              render: (value) => (
                <div className="flex items-center gap-3">
                  <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden max-w-16">
                    <motion.div
                      initial={{ width: 0 }}
                      animate={{ width: `${(value || 0) * 100}%` }}
                      transition={{ duration: 0.8, ease: "easeOut" }}
                      className={cn(
                        "h-full transition-colors",
                        (value || 0) >= 0.8 ? 'bg-green-500' :
                        (value || 0) >= 0.6 ? 'bg-yellow-500' : 'bg-red-500'
                      )}
                    />
                  </div>
                  <span className="text-sm font-medium min-w-12">
                    {((value || 0) * 100).toFixed(0)}%
                  </span>
                </div>
              )
            },
            {
              key: 'proxy_id',
              title: '代理',
              width: '100px',
              render: (value) => (
                <Badge variant={value ? 'default' : 'secondary'} className="text-xs">
                  {value ? '已绑定' : '未绑定'}
                </Badge>
              )
            },
            {
              key: 'last_used_at',
              title: '最后使用',
              width: '120px',
              sortable: true,
              render: (value) => (
                <div className="text-sm text-muted-foreground">
                  {value ? new Date(value).toLocaleDateString() : '从未使用'}
                </div>
              )
            },
            {
              key: 'actions',
              title: '操作',
              width: '100px',
              render: (_, record) => (
                <DropdownMenu>
                  <DropdownMenuTrigger asChild>
                    <Button variant="ghost" size="icon" className="h-8 w-8">
                      <MoreVertical className="h-4 w-4" />
                    </Button>
                  </DropdownMenuTrigger>
                  <DropdownMenuContent className="glass-effect" align="end">
                    <DropdownMenuItem onClick={() => handleEditAccount(record)}>
                      <Pencil className="h-4 w-4 mr-2" />
                      编辑账号
                    </DropdownMenuItem>
                    <DropdownMenuItem 
                      onClick={() => handleCheckHealth(record)}
                      disabled={healthChecking === record.id}
                    >
                      <Activity className="h-4 w-4 mr-2" />
                      {healthChecking === record.id ? "检查中..." : "检查健康"}
                    </DropdownMenuItem>
                    <DropdownMenuItem onClick={() => handleBindProxy(record)}>
                      <Link2 className="h-4 w-4 mr-2" />
                      绑定代理
                    </DropdownMenuItem>
                    <DropdownMenuSeparator />
                    <DropdownMenuItem 
                      onClick={() => handleDeleteAccount(record)}
                      className="text-destructive"
                    >
                      <Trash2 className="h-4 w-4 mr-2" />
                      删除账号
                    </DropdownMenuItem>
                  </DropdownMenuContent>
                </DropdownMenu>
              )
            }
          ]}
          loading={loading}
          searchable
          searchPlaceholder="搜索手机号或备注..."
          filterable
          emptyText="暂无账号数据"
          className="card-shadow"
        />

        {/* Pagination */}
        <div className="flex items-center justify-between">
          <div className="text-sm text-muted-foreground">
            共 {total} 个账号，当前第 {page} 页
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

        {/* 编辑账号对话框 */}
        <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle>编辑账号</DialogTitle>
              <DialogDescription>
                更新账号信息。Session数据通常不允许修改。
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="edit-phone">手机号</Label>
                <Input
                  id="edit-phone"
                  value={editForm.phone}
                  onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })}
                  disabled
                  className="bg-muted"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-note">备注</Label>
                <Textarea
                  id="edit-note"
                  value={editForm.note}
                  onChange={(e) => setEditForm({ ...editForm, note: e.target.value })}
                  placeholder="输入备注信息..."
                  rows={3}
                />
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setEditDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleSaveEdit}>保存</Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 绑定代理对话框 */}
        <Dialog open={bindProxyDialogOpen} onOpenChange={setBindProxyDialogOpen}>
          <DialogContent className="sm:max-w-[400px]">
            <DialogHeader>
              <DialogTitle>绑定代理</DialogTitle>
              <DialogDescription>
                为账号 {bindingAccount?.phone} 绑定或解绑代理
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="bind-proxy-select">选择代理</Label>
                <Select
                  value={selectedBindProxy || "none"}
                  onValueChange={(value) => setSelectedBindProxy(value === "none" ? "" : value)}
                  disabled={loadingProxies}
                >
                  <SelectTrigger id="bind-proxy-select">
                    <SelectValue placeholder={loadingProxies ? "加载中..." : "不绑定代理"} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">不绑定代理（解绑）</SelectItem>
                    {proxies.map((proxy) => (
                      <SelectItem key={proxy.id} value={String(proxy.id)}>
                        {proxy.host}:{proxy.port} {proxy.username && `(${proxy.username})`}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setBindProxyDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleSaveBindProxy}>确认</Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 手动添加账号对话框 */}
        <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
          <DialogContent className="sm:max-w-[600px]">
            <DialogHeader>
              <DialogTitle>手动添加账号</DialogTitle>
              <DialogDescription>
                手动输入账号信息添加新的TG账号
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="add-phone">手机号 *</Label>
                <Input
                  id="add-phone"
                  value={addForm.phone}
                  onChange={(e) => setAddForm({ ...addForm, phone: e.target.value })}
                  placeholder="+1234567890"
                  required
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="add-session">Session数据 *</Label>
                <Textarea
                  id="add-session"
                  value={addForm.session_data}
                  onChange={(e) => setAddForm({ ...addForm, session_data: e.target.value })}
                  placeholder="粘贴SessionString..."
                  rows={6}
                  required
                  className="font-mono text-xs"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="add-note">备注（可选）</Label>
                <Input
                  id="add-note"
                  value={addForm.note}
                  onChange={(e) => setAddForm({ ...addForm, note: e.target.value })}
                  placeholder="输入备注信息..."
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="add-proxy-select">代理（可选）</Label>
                <Select
                  value={addForm.proxy_id || "none"}
                  onValueChange={(value) => setAddForm({ ...addForm, proxy_id: value === "none" ? "" : value })}
                  disabled={loadingProxies}
                >
                  <SelectTrigger id="add-proxy-select">
                    <SelectValue placeholder={loadingProxies ? "加载中..." : "不绑定代理"} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">不绑定代理</SelectItem>
                    {proxies.map((proxy) => (
                      <SelectItem key={proxy.id} value={String(proxy.id)}>
                        {proxy.host}:{proxy.port} {proxy.username && `(${proxy.username})`}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            </div>
            <div className="flex justify-end gap-2">
              <Button variant="outline" onClick={() => setAddDialogOpen(false)}>
                取消
              </Button>
              <Button onClick={handleSaveAdd}>添加</Button>
            </div>
          </DialogContent>
        </Dialog>
      </div>
    </MainLayout>
  )
}

