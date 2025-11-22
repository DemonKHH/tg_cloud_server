"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
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
import { Plus, MoreVertical, CheckCircle2, XCircle, AlertCircle, Upload, FileArchive, Search } from "lucide-react"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { cn } from "@/lib/utils"
import { accountAPI, proxyAPI } from "@/lib/api"
import { useState, useEffect, useRef } from "react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Pencil, Trash2, Activity, Link2, MessageSquare, Megaphone, Users, ShieldCheck, ChevronDown } from "lucide-react"
import { usePagination } from "@/hooks/use-pagination"
import { FilterBar } from "@/components/common/filter-bar"
import { motion } from "framer-motion"
import { Checkbox } from "@/components/ui/checkbox"
import { CreateTaskDialog } from "@/components/business/create-task-dialog"

import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"

export default function AccountsPage() {
  const {
    data: accounts,
    page,
    total,
    loading,
    search,
    setSearch,
    setPage,
    refresh,
  } = usePagination({
    fetchFn: accountAPI.list,
    pageSize: 50, // 每页50个
  })

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

  // 批量操作相关状态
  const [selectedAccountIds, setSelectedAccountIds] = useState<string[]>([])
  const [createTaskDialogOpen, setCreateTaskDialogOpen] = useState(false)
  const [initialTaskType, setInitialTaskType] = useState<string>("")



  // 全选/取消全选
  const toggleSelectAll = () => {
    if (selectedAccountIds.length === accounts.length) {
      setSelectedAccountIds([])
    } else {
      setSelectedAccountIds(accounts.map(a => String(a.id)))
    }
  }

  // 选择/取消选择单个
  const toggleSelectOne = (id: string) => {
    if (selectedAccountIds.includes(id)) {
      setSelectedAccountIds(selectedAccountIds.filter(i => i !== id))
    } else {
      setSelectedAccountIds([...selectedAccountIds, id])
    }
  }

  useEffect(() => {
    loadProxies()
  }, [])

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
      refresh()
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
      refresh()
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
        refresh() // 重新加载以更新健康度
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
      refresh()
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
      refresh()
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
          refresh() // 重新加载账号列表
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
        refresh()
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
        <div className="flex flex-col gap-2 md:flex-row md:items-center md:justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight gradient-text">账号管理</h1>
            <p className="text-muted-foreground mt-1">管理和监控您的 Telegram 账号</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Dialog open={uploadDialogOpen} onOpenChange={setUploadDialogOpen}>
              <DialogTrigger asChild>
                <Button variant="outline" className="btn-modern">
                  <Upload className="h-4 w-4 mr-2" />
                  上传账号文件
                </Button>
              </DialogTrigger>
                <DialogContent className="sm:max-w-[500px]">
                  <DialogHeader>
                    <DialogTitle className="text-2xl">上传账号文件</DialogTitle>
                    <DialogDescription>
                      支持上传 .zip、.session 文件或 tdata 文件夹。系统将自动解析 Session/TData 格式并转换为 SessionString。
                    </DialogDescription>
                  </DialogHeader>
                  <div className="space-y-4 py-4">
                    {/* 文件上传区域 */}
                    <div className="relative border-2 border-dashed border-muted-foreground/25 rounded-xl p-8 text-center hover:border-primary/50 transition-colors bg-gradient-to-br from-muted/30 to-muted/10">
                      <div className="absolute top-4 right-4">
                        <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                          <Upload className="h-4 w-4 text-primary" />
                        </div>
                      </div>
                      <FileArchive className="h-16 w-16 mx-auto mb-4 text-primary/60" />
                      <p className="text-sm font-semibold mb-3">
                        支持的文件格式
                      </p>
                      <div className="bg-background/50 rounded-lg p-4 mb-4">
                        <ul className="text-xs text-muted-foreground space-y-2 text-left max-w-xs mx-auto">
                          <li className="flex items-center gap-2">
                            <div className="h-1.5 w-1.5 rounded-full bg-primary" />
                            .zip 压缩包（可包含多个账号文件）
                          </li>
                          <li className="flex items-center gap-2">
                            <div className="h-1.5 w-1.5 rounded-full bg-primary" />
                            .session 文件（Pyrogram 格式）
                          </li>
                          <li className="flex items-center gap-2">
                            <div className="h-1.5 w-1.5 rounded-full bg-primary" />
                            tdata 文件夹（Telegram Desktop 格式）
                          </li>
                          <li className="flex items-center gap-2">
                            <div className="h-1.5 w-1.5 rounded-full bg-primary" />
                            gotd/td 格式 session 文件
                          </li>
                        </ul>
                      </div>
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
                        variant={uploading ? "outline" : "default"}
                        onClick={() => fileInputRef.current?.click()}
                        disabled={uploading}
                        className="btn-modern"
                      >
                        {uploading ? (
                          <>
                            <Activity className="h-4 w-4 mr-2 animate-spin" />
                            上传中...
                          </>
                        ) : (
                          <>
                            <Upload className="h-4 w-4 mr-2" />
                            选择文件
                          </>
                        )}
                      </Button>
                      {uploading && (
                        <motion.p 
                          initial={{ opacity: 0 }}
                          animate={{ opacity: 1 }}
                          className="text-sm text-muted-foreground mt-3"
                        >
                          正在解析文件并转换格式，请稍候...
                        </motion.p>
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
                    <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950/30 dark:to-purple-950/30 rounded-xl p-4 border border-blue-200/50 dark:border-blue-800/50">
                      <div className="flex items-start gap-3">
                        <div className="h-8 w-8 rounded-full bg-blue-500/10 flex items-center justify-center flex-shrink-0 mt-0.5">
                          <AlertCircle className="h-4 w-4 text-blue-600 dark:text-blue-400" />
                        </div>
                        <div className="flex-1">
                          <p className="font-semibold text-sm mb-2 text-blue-900 dark:text-blue-100">温馨提示</p>
                          <ul className="space-y-1.5 text-xs text-blue-700/80 dark:text-blue-300/80">
                            <li className="flex items-start gap-2">
                              <CheckCircle2 className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
                              <span>系统会自动识别文件格式并转换为 SessionString</span>
                            </li>
                            <li className="flex items-start gap-2">
                              <CheckCircle2 className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
                              <span>如果文件包含手机号信息，会自动提取</span>
                            </li>
                            <li className="flex items-start gap-2">
                              <CheckCircle2 className="h-3.5 w-3.5 mt-0.5 flex-shrink-0" />
                              <span>单个文件最大支持 100MB</span>
                            </li>
                          </ul>
                        </div>
                      </div>
                    </div>
                  </div>
                </DialogContent>
              </Dialog>
            <Button onClick={handleAddAccount} className="btn-modern">
              <Plus className="h-4 w-4 mr-2" />
              手动添加
            </Button>
          </div>
        </div>

        {/* Stats Cards */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3 }}
          >
            <Card className="relative overflow-hidden border-none shadow-sm bg-gradient-to-br from-blue-50 to-blue-100/50 dark:from-blue-950/50 dark:to-blue-900/30">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-blue-900 dark:text-blue-100">总账号数</CardTitle>
                <div className="h-10 w-10 rounded-full bg-blue-500/10 flex items-center justify-center">
                  <Users className="h-5 w-5 text-blue-600 dark:text-blue-400" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold text-blue-900 dark:text-blue-100">{total}</div>
                <p className="text-xs text-blue-700/70 dark:text-blue-300/70 mt-1">
                  当前系统中的所有账号
                </p>
              </CardContent>
              <div className="absolute -right-4 -bottom-4 h-24 w-24 rounded-full bg-blue-500/5" />
            </Card>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.1 }}
          >
            <Card className="relative overflow-hidden border-none shadow-sm bg-gradient-to-br from-green-50 to-green-100/50 dark:from-green-950/50 dark:to-green-900/30">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-green-900 dark:text-green-100">正常账号</CardTitle>
                <div className="h-10 w-10 rounded-full bg-green-500/10 flex items-center justify-center">
                  <CheckCircle2 className="h-5 w-5 text-green-600 dark:text-green-400" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold text-green-900 dark:text-green-100">
                  {accounts.filter(a => a.status === 'normal').length}
                </div>
                <p className="text-xs text-green-700/70 dark:text-green-300/70 mt-1">
                  健康状态良好的账号
                </p>
              </CardContent>
              <div className="absolute -right-4 -bottom-4 h-24 w-24 rounded-full bg-green-500/5" />
            </Card>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.2 }}
          >
            <Card className="relative overflow-hidden border-none shadow-sm bg-gradient-to-br from-purple-50 to-purple-100/50 dark:from-purple-950/50 dark:to-purple-900/30">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-purple-900 dark:text-purple-100">已同步信息</CardTitle>
                <div className="h-10 w-10 rounded-full bg-purple-500/10 flex items-center justify-center">
                  <CheckCircle2 className="h-5 w-5 text-purple-600 dark:text-purple-400" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold text-purple-900 dark:text-purple-100">
                  {accounts.filter(a => (a.username && a.username.length > 0) || (a.first_name && a.first_name.length > 0)).length}
                </div>
                <p className="text-xs text-purple-700/70 dark:text-purple-300/70 mt-1">
                  已同步 Telegram 信息的账号
                </p>
              </CardContent>
              <div className="absolute -right-4 -bottom-4 h-24 w-24 rounded-full bg-purple-500/5" />
            </Card>
          </motion.div>

          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.3, delay: 0.3 }}
          >
            <Card className="relative overflow-hidden border-none shadow-sm bg-gradient-to-br from-orange-50 to-orange-100/50 dark:from-orange-950/50 dark:to-orange-900/30">
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium text-orange-900 dark:text-orange-100">平均健康度</CardTitle>
                <div className="h-10 w-10 rounded-full bg-orange-500/10 flex items-center justify-center">
                  <Activity className="h-5 w-5 text-orange-600 dark:text-orange-400" />
                </div>
              </CardHeader>
              <CardContent>
                <div className="text-3xl font-bold text-orange-900 dark:text-orange-100">
                  {accounts.length > 0 
                    ? ((accounts.reduce((sum, a) => sum + (a.health_score || 0), 0) / accounts.length) * 100).toFixed(0)
                    : 0}%
                </div>
                <p className="text-xs text-orange-700/70 dark:text-orange-300/70 mt-1">
                  所有账号的平均健康度
                </p>
              </CardContent>
              <div className="absolute -right-4 -bottom-4 h-24 w-24 rounded-full bg-orange-500/5" />
            </Card>
          </motion.div>
        </div>

        {/* Search Bar */}
        <Card className="border-none shadow-sm">
          <CardContent className="p-4">
            <div className="flex items-center gap-2">
              <div className="relative flex-1 max-w-md">
                <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
                <Input
                  placeholder="搜索手机号..."
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
                  <XCircle className="h-4 w-4 mr-1" />
                  清除
                </Button>
              )}
            </div>
          </CardContent>
        </Card>

        {/* Account Cards Grid */}
        <div className="space-y-4">
          {/* Batch Actions Bar */}
          {selectedAccountIds.length > 0 && (
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              exit={{ opacity: 0, y: 20 }}
              className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 bg-background/95 backdrop-blur-lg border shadow-2xl rounded-full px-6 py-3 flex items-center gap-4"
            >
              <div className="flex items-center gap-2 border-r pr-4 mr-2">
                <div className="bg-primary text-primary-foreground text-xs font-bold rounded-full w-6 h-6 flex items-center justify-center">
                  {selectedAccountIds.length}
                </div>
                <span className="text-sm font-medium text-muted-foreground">
                  已选择
                </span>
              </div>

              <div className="flex items-center gap-2">
                <TooltipProvider delayDuration={0}>
                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="rounded-full hover:bg-primary/10 hover:text-primary"
                        onClick={() => {
                          setInitialTaskType("check")
                          setCreateTaskDialogOpen(true)
                        }}
                      >
                        <Activity className="h-5 w-5" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>检查健康</p>
                    </TooltipContent>
                  </Tooltip>

                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="rounded-full hover:bg-primary/10 hover:text-primary"
                        onClick={() => {
                          setInitialTaskType("private_message")
                          setCreateTaskDialogOpen(true)
                        }}
                      >
                        <MessageSquare className="h-5 w-5" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>发送私信</p>
                    </TooltipContent>
                  </Tooltip>

                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="rounded-full hover:bg-primary/10 hover:text-primary"
                        onClick={() => {
                          setInitialTaskType("broadcast")
                          setCreateTaskDialogOpen(true)
                        }}
                      >
                        <Megaphone className="h-5 w-5" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>群发消息</p>
                    </TooltipContent>
                  </Tooltip>

                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="rounded-full hover:bg-primary/10 hover:text-primary"
                        onClick={() => {
                          setInitialTaskType("group_chat")
                          setCreateTaskDialogOpen(true)
                        }}
                      >
                        <Users className="h-5 w-5" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>AI炒群</p>
                    </TooltipContent>
                  </Tooltip>

                  <Tooltip>
                    <TooltipTrigger asChild>
                      <Button
                        size="icon"
                        variant="ghost"
                        className="rounded-full hover:bg-primary/10 hover:text-primary"
                        onClick={() => {
                          setInitialTaskType("verify_code")
                          setCreateTaskDialogOpen(true)
                        }}
                      >
                        <ShieldCheck className="h-5 w-5" />
                      </Button>
                    </TooltipTrigger>
                    <TooltipContent>
                      <p>接收验证码</p>
                    </TooltipContent>
                  </Tooltip>
                </TooltipProvider>

                <div className="w-px h-6 bg-border mx-2" />

                <Button
                  size="sm"
                  variant="ghost"
                  className="text-muted-foreground hover:text-foreground rounded-full px-3"
                  onClick={() => setSelectedAccountIds([])}
                >
                  取消
                </Button>
              </div>
            </motion.div>
          )}

          {/* 账号表格 */}
          <Card className="border-none shadow-md overflow-hidden">
            <CardContent className="p-0">
              <div className="overflow-x-auto">
                <Table>
                  <TableHeader>
                    <TableRow className="bg-muted/50 hover:bg-muted/50 border-b-2">
                      <TableHead className="w-[50px] h-12">
                        <Checkbox
                          checked={accounts.length > 0 && selectedAccountIds.length === accounts.length}
                          onCheckedChange={toggleSelectAll}
                          className="border-2 border-primary data-[state=checked]:bg-primary data-[state=checked]:border-primary"
                        />
                      </TableHead>
                      <TableHead className="w-[180px] font-semibold">账号信息</TableHead>
                      <TableHead className="w-[120px] font-semibold">状态</TableHead>
                      <TableHead className="w-[150px] font-semibold">健康度</TableHead>
                      <TableHead className="w-[100px] font-semibold">代理</TableHead>
                      <TableHead className="w-[140px] font-semibold">最后使用</TableHead>
                      <TableHead className="w-[200px] text-right font-semibold">操作</TableHead>
                    </TableRow>
                  </TableHeader>
                <TableBody>
                  {loading ? (
                    // 加载状态 - 美化的骨架屏
                    Array.from({ length: 10 }).map((_, index) => (
                      <TableRow key={index} className="animate-pulse">
                        <TableCell className="py-4">
                          <div className="h-5 w-5 bg-muted rounded" />
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="flex items-center gap-3">
                            <div className="h-10 w-10 bg-muted rounded-full" />
                            <div className="space-y-2">
                              <div className="h-4 w-28 bg-muted rounded" />
                              <div className="h-3 w-20 bg-muted rounded" />
                            </div>
                          </div>
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="h-6 w-16 bg-muted rounded-full" />
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="flex items-center gap-3">
                            <div className="h-2.5 w-20 bg-muted rounded-full" />
                            <div className="h-4 w-12 bg-muted rounded" />
                          </div>
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="h-6 w-16 bg-muted rounded-full" />
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="h-4 w-20 bg-muted rounded" />
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="flex items-center justify-end gap-1">
                            <div className="h-9 w-9 bg-muted rounded-lg" />
                            <div className="h-9 w-9 bg-muted rounded-lg" />
                            <div className="h-9 w-9 bg-muted rounded-lg" />
                            <div className="h-9 w-9 bg-muted rounded-lg" />
                          </div>
                        </TableCell>
                      </TableRow>
                    ))
                  ) : accounts.length === 0 ? (
                    <TableRow>
                      <TableCell colSpan={7} className="h-64">
                        <div className="flex flex-col items-center justify-center">
                          <Users className="h-12 w-12 text-muted-foreground/50 mb-4" />
                          <p className="text-lg font-medium text-muted-foreground mb-2">暂无账号数据</p>
                          <p className="text-sm text-muted-foreground mb-6">开始添加您的第一个 Telegram 账号</p>
                          <div className="flex gap-2">
                            <Button onClick={handleAddAccount}>
                              <Plus className="h-4 w-4 mr-2" />
                              手动添加
                            </Button>
                            <Button variant="outline" onClick={() => setUploadDialogOpen(true)}>
                              <Upload className="h-4 w-4 mr-2" />
                              上传文件
                            </Button>
                          </div>
                        </div>
                      </TableCell>
                    </TableRow>
                  ) : (
                    accounts.map((record, index) => (
                      <TableRow 
                        key={record.id} 
                        className={cn(
                          "group transition-colors hover:bg-muted/50",
                          selectedAccountIds.includes(String(record.id)) && "bg-primary/5"
                        )}
                      >
                        <TableCell className="py-4">
                          <Checkbox
                            checked={selectedAccountIds.includes(String(record.id))}
                            onCheckedChange={() => toggleSelectOne(String(record.id))}
                            className="border-2 border-primary/60 data-[state=checked]:bg-primary data-[state=checked]:border-primary hover:border-primary"
                          />
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="flex items-center gap-3">
                            {/* 头像 */}
                            <div className={cn(
                              "h-10 w-10 rounded-full flex items-center justify-center text-sm font-semibold flex-shrink-0",
                              record.status === 'normal' ? 'bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400' :
                              record.status === 'warning' ? 'bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400' :
                              record.status === 'restricted' ? 'bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400' :
                              record.status === 'dead' ? 'bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400' :
                              'bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400'
                            )}>
                              {(record.first_name && record.first_name.length > 0) ? record.first_name.charAt(0).toUpperCase() : record.phone.slice(-2)}
                            </div>
                            <div className="space-y-1 min-w-0 flex-1">
                              {/* 显示名称或手机号 */}
                              <div className="flex items-center gap-2">
                                <div className="font-semibold text-sm truncate">
                                  {(record.first_name && record.first_name.length > 0) ? record.first_name : record.phone}
                                  {(record.last_name && record.last_name.length > 0) && ` ${record.last_name}`}
                                </div>
                                {(record.username && record.username.length > 0) && (
                                  <span className="text-xs text-muted-foreground">@{record.username}</span>
                                )}
                              </div>
                              {/* 显示手机号（如果有名字）或备注 */}
                              {((record.first_name && record.first_name.length > 0) || (record.username && record.username.length > 0)) && (
                                <div className="text-xs text-muted-foreground truncate">{record.phone}</div>
                              )}
                            </div>
                          </div>
                        </TableCell>
                        <TableCell className="py-4">
                          <Badge
                            variant={record.status === 'normal' ? 'default' : record.status === 'dead' || record.status === 'restricted' ? 'destructive' : 'secondary'}
                            className={cn("text-xs font-medium", getStatusColor(record.status))}
                          >
                            {getStatusIcon(record.status)}
                            <span className="ml-1.5">{getStatusText(record.status)}</span>
                          </Badge>
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="flex items-center gap-3">
                            <div className="flex-1 h-2.5 bg-muted rounded-full overflow-hidden max-w-[90px] shadow-inner">
                              <motion.div
                                initial={{ width: 0 }}
                                animate={{ width: `${(record.health_score || 0) * 100}%` }}
                                transition={{ duration: 0.8, ease: "easeOut", delay: index * 0.02 }}
                                className={cn(
                                  "h-full transition-all",
                                  (record.health_score || 0) >= 0.8 ? 'bg-gradient-to-r from-green-500 to-green-600' :
                                  (record.health_score || 0) >= 0.6 ? 'bg-gradient-to-r from-yellow-500 to-yellow-600' : 
                                  'bg-gradient-to-r from-red-500 to-red-600'
                                )}
                              />
                            </div>
                            <span className={cn(
                              "text-sm font-bold min-w-[45px]",
                              (record.health_score || 0) >= 0.8 ? 'text-green-600 dark:text-green-400' :
                              (record.health_score || 0) >= 0.6 ? 'text-yellow-600 dark:text-yellow-400' : 
                              'text-red-600 dark:text-red-400'
                            )}>
                              {((record.health_score || 0) * 100).toFixed(0)}%
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="py-4">
                          <Badge 
                            variant={record.proxy_id ? 'default' : 'outline'} 
                            className={cn(
                              "text-xs font-medium",
                              record.proxy_id && "bg-purple-100 text-purple-700 border-purple-200 dark:bg-purple-900/30 dark:text-purple-400 dark:border-purple-800"
                            )}
                          >
                            <div className={cn(
                              "h-1.5 w-1.5 rounded-full mr-1.5",
                              record.proxy_id ? "bg-purple-500" : "bg-gray-400"
                            )} />
                            {record.proxy_id ? '已绑定' : '未绑定'}
                          </Badge>
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="flex items-center gap-2 text-sm text-muted-foreground">
                            <Activity className="h-3.5 w-3.5" />
                            <span>
                              {record.last_used_at ? new Date(record.last_used_at).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }) : '从未'}
                            </span>
                          </div>
                        </TableCell>
                        <TableCell className="py-4">
                          <div className="flex items-center justify-end gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <TooltipProvider>
                              {/* 编辑账号 */}
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-9 w-9 rounded-lg hover:bg-blue-50 text-blue-600 hover:text-blue-700 dark:hover:bg-blue-950 transition-all hover:scale-105"
                                    onClick={() => handleEditAccount(record)}
                                  >
                                    <Pencil className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  <p className="text-xs">编辑账号</p>
                                </TooltipContent>
                              </Tooltip>

                              {/* 检查健康 */}
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className={cn(
                                      "h-9 w-9 rounded-lg transition-all",
                                      healthChecking === record.id
                                        ? "opacity-50 cursor-not-allowed text-muted-foreground"
                                        : "hover:bg-green-50 text-green-600 hover:text-green-700 dark:hover:bg-green-950 hover:scale-105"
                                    )}
                                    disabled={healthChecking === record.id}
                                    onClick={() => handleCheckHealth(record)}
                                  >
                                    <Activity className={cn("h-4 w-4", healthChecking === record.id && "animate-spin")} />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  <p className="text-xs">
                                    {healthChecking === record.id ? "检查中..." : "健康检查"}
                                  </p>
                                </TooltipContent>
                              </Tooltip>

                              {/* 绑定代理 */}
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-9 w-9 rounded-lg hover:bg-purple-50 text-purple-600 hover:text-purple-700 dark:hover:bg-purple-950 transition-all hover:scale-105"
                                    onClick={() => handleBindProxy(record)}
                                  >
                                    <Link2 className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  <p className="text-xs">绑定代理</p>
                                </TooltipContent>
                              </Tooltip>

                              {/* 删除账号 */}
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-9 w-9 rounded-lg hover:bg-red-50 text-red-600 hover:text-red-700 dark:hover:bg-red-950 transition-all hover:scale-105"
                                    onClick={() => handleDeleteAccount(record)}
                                  >
                                    <Trash2 className="h-4 w-4" />
                                  </Button>
                                </TooltipTrigger>
                                <TooltipContent side="top">
                                  <p className="text-xs">删除账号</p>
                                </TooltipContent>
                              </Tooltip>
                            </TooltipProvider>
                          </div>
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
                    共 <span className="text-primary font-bold">{total}</span> 个账号
                  </div>
                  <div className="h-4 w-px bg-border" />
                  <div className="text-sm text-muted-foreground">
                    第 {page} 页 · 每页 50 条
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
                    <span className="text-sm text-muted-foreground">{Math.ceil(total / 50)}</span>
                  </div>
                  <Button
                    variant="outline"
                    size="sm"
                    onClick={() => setPage((p) => p + 1)}
                    disabled={page * 50 >= total}
                    className="btn-modern h-9 px-4"
                  >
                    下一页
                    <ChevronDown className="h-4 w-4 ml-1 -rotate-90" />
                  </Button>
                </div>
              </div>
            </CardContent>
          </Card>
        </div>

        {/* 编辑账号对话框 */}
        <Dialog open={editDialogOpen} onOpenChange={setEditDialogOpen}>
          <DialogContent className="sm:max-w-[500px]">
            <DialogHeader>
              <DialogTitle className="text-2xl">编辑账号</DialogTitle>
              <DialogDescription>
                更新账号信息。Telegram 信息由系统自动同步。
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              {/* Telegram 信息展示 */}
              {((editingAccount?.username && editingAccount.username.length > 0) || (editingAccount?.first_name && editingAccount.first_name.length > 0)) && (
                <div className="bg-gradient-to-r from-blue-50 to-purple-50 dark:from-blue-950/30 dark:to-purple-950/30 rounded-xl p-4 border border-blue-200/50 dark:border-blue-800/50">
                  <div className="flex items-start gap-3">
                    <div className="h-12 w-12 rounded-full bg-primary/10 flex items-center justify-center flex-shrink-0 text-lg font-bold text-primary">
                      {(editingAccount?.first_name && editingAccount.first_name.length > 0) ? editingAccount.first_name.charAt(0).toUpperCase() : 'T'}
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="font-semibold text-sm mb-1">
                        {(editingAccount?.first_name && editingAccount.first_name.length > 0) ? editingAccount.first_name : 'Telegram 用户'}
                        {(editingAccount?.last_name && editingAccount.last_name.length > 0) && ` ${editingAccount.last_name}`}
                      </div>
                      {(editingAccount?.username && editingAccount.username.length > 0) && (
                        <div className="text-xs text-blue-600 dark:text-blue-400 mb-1">
                          @{editingAccount.username}
                        </div>
                      )}
                      {(editingAccount?.bio && editingAccount.bio.length > 0) && (
                        <div className="text-xs text-muted-foreground line-clamp-2 mt-2">
                          {editingAccount.bio}
                        </div>
                      )}
                      {editingAccount?.tg_user_id && (
                        <div className="text-xs text-muted-foreground mt-1">
                          ID: {editingAccount.tg_user_id}
                        </div>
                      )}
                    </div>
                  </div>
                </div>
              )}
              
              <div className="space-y-2">
                <Label htmlFor="edit-phone" className="text-sm font-medium">手机号</Label>
                <Input
                  id="edit-phone"
                  value={editForm.phone}
                  onChange={(e) => setEditForm({ ...editForm, phone: e.target.value })}
                  disabled
                  className="bg-muted/50 input-modern"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="edit-note" className="text-sm font-medium">备注</Label>
                <Textarea
                  id="edit-note"
                  value={editForm.note}
                  onChange={(e) => setEditForm({ ...editForm, note: e.target.value })}
                  placeholder="输入备注信息..."
                  rows={3}
                  className="input-modern resize-none"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-4 border-t">
              <Button variant="outline" onClick={() => setEditDialogOpen(false)} className="btn-modern">
                取消
              </Button>
              <Button onClick={handleSaveEdit} className="btn-modern">保存</Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 绑定代理对话框 */}
        <Dialog open={bindProxyDialogOpen} onOpenChange={setBindProxyDialogOpen}>
          <DialogContent className="sm:max-w-[450px]">
            <DialogHeader>
              <DialogTitle className="text-2xl">绑定代理</DialogTitle>
              <DialogDescription>
                为账号 <span className="font-semibold text-foreground">{bindingAccount?.phone}</span> 绑定或解绑代理
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="bind-proxy-select" className="text-sm font-medium">选择代理</Label>
                <Select
                  value={selectedBindProxy || "none"}
                  onValueChange={(value) => setSelectedBindProxy(value === "none" ? "" : value)}
                  disabled={loadingProxies}
                >
                  <SelectTrigger id="bind-proxy-select" className="input-modern">
                    <SelectValue placeholder={loadingProxies ? "加载中..." : "不绑定代理"} />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="none">
                      <div className="flex items-center gap-2">
                        <XCircle className="h-4 w-4 text-muted-foreground" />
                        不绑定代理（解绑）
                      </div>
                    </SelectItem>
                    {proxies.map((proxy) => (
                      <SelectItem key={proxy.id} value={String(proxy.id)}>
                        <div className="flex items-center gap-2">
                          <Link2 className="h-4 w-4 text-primary" />
                          {proxy.host}:{proxy.port} {proxy.username && `(${proxy.username})`}
                        </div>
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
                {proxies.length === 0 && !loadingProxies && (
                  <p className="text-xs text-muted-foreground flex items-center gap-1.5 mt-2">
                    <AlertCircle className="h-3.5 w-3.5" />
                    暂无可用代理，可以在代理管理中添加
                  </p>
                )}
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-4 border-t">
              <Button variant="outline" onClick={() => setBindProxyDialogOpen(false)} className="btn-modern">
                取消
              </Button>
              <Button onClick={handleSaveBindProxy} className="btn-modern">确认</Button>
            </div>
          </DialogContent>
        </Dialog>

        {/* 手动添加账号对话框 */}
        <Dialog open={addDialogOpen} onOpenChange={setAddDialogOpen}>
          <DialogContent className="sm:max-w-[600px] max-h-[90vh] overflow-y-auto">
            <DialogHeader>
              <DialogTitle className="text-2xl">手动添加账号</DialogTitle>
              <DialogDescription>
                手动输入账号信息添加新的 Telegram 账号
              </DialogDescription>
            </DialogHeader>
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label htmlFor="add-phone" className="text-sm font-medium flex items-center gap-1">
                  手机号 <span className="text-red-500">*</span>
                </Label>
                <Input
                  id="add-phone"
                  value={addForm.phone}
                  onChange={(e) => setAddForm({ ...addForm, phone: e.target.value })}
                  placeholder="+1234567890"
                  required
                  className="input-modern"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="add-session" className="text-sm font-medium flex items-center gap-1">
                  Session数据 <span className="text-red-500">*</span>
                </Label>
                <Textarea
                  id="add-session"
                  value={addForm.session_data}
                  onChange={(e) => setAddForm({ ...addForm, session_data: e.target.value })}
                  placeholder="粘贴SessionString..."
                  rows={6}
                  required
                  className="font-mono text-xs input-modern resize-none"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="add-note" className="text-sm font-medium">备注（可选）</Label>
                <Input
                  id="add-note"
                  value={addForm.note}
                  onChange={(e) => setAddForm({ ...addForm, note: e.target.value })}
                  placeholder="输入备注信息..."
                  className="input-modern"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="add-proxy" className="text-sm font-medium">代理ID（可选）</Label>
                <Input
                  id="add-proxy"
                  value={addForm.proxy_id}
                  onChange={(e) => setAddForm({ ...addForm, proxy_id: e.target.value })}
                  placeholder="输入代理ID..."
                  className="input-modern"
                />
              </div>
            </div>
            <div className="flex justify-end gap-2 pt-4 border-t">
              <Button variant="outline" onClick={() => setAddDialogOpen(false)} className="btn-modern">
                取消
              </Button>
              <Button onClick={handleSaveAdd} className="btn-modern">
                <Plus className="h-4 w-4 mr-2" />
                添加账号
              </Button>
            </div>
          </DialogContent>
        </Dialog>

        <CreateTaskDialog
          open={createTaskDialogOpen}
          onOpenChange={setCreateTaskDialogOpen}
          accountIds={selectedAccountIds}
          initialTaskType={initialTaskType}
          onSuccess={() => {
            setSelectedAccountIds([])
            toast.success("任务创建成功，请前往任务页面查看")
          }}
        />
      </div >
    </MainLayout >
  )
}

