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

  useEffect(() => {
    loadAccounts()
  }, [page])

  useEffect(() => {
    if (uploadDialogOpen) {
      loadProxies()
    }
  }, [uploadDialogOpen])

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
      const response = await accountAPI.list({ page, limit: 20 })
      if (response.data) {
        // 后端返回格式：{ items: [], pagination: { total, current_page, ... } }
        const data = response.data as any
        setAccounts(data.items || [])
        setTotal(data.pagination?.total || 0)
      }
    } catch (error) {
      toast.error("加载账号失败，请稍后重试")
      console.error("加载账号失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const getStatusIcon = (status: string) => {
    switch (status) {
      case "active":
        return <CheckCircle2 className="h-4 w-4 text-green-500" />
      case "error":
        return <XCircle className="h-4 w-4 text-red-500" />
      default:
        return <AlertCircle className="h-4 w-4 text-yellow-500" />
    }
  }

  const getStatusColor = (status: string) => {
    switch (status) {
      case "active":
        return "bg-green-100 text-green-800 dark:bg-green-900 dark:text-green-300"
      case "error":
        return "bg-red-100 text-red-800 dark:bg-red-900 dark:text-red-300"
      default:
        return "bg-yellow-100 text-yellow-800 dark:bg-yellow-900 dark:text-yellow-300"
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
                      value={selectedProxy}
                      onValueChange={setSelectedProxy}
                      disabled={uploading || loadingProxies}
                    >
                      <SelectTrigger id="proxy-select">
                        <SelectValue placeholder={loadingProxies ? "加载中..." : "不绑定代理"} />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="">不绑定代理</SelectItem>
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
            <Button>
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

        {/* Accounts Table */}
        <Card>
          <CardHeader>
            <CardTitle>账号列表</CardTitle>
          </CardHeader>
          <CardContent>
            {loading ? (
              <div className="text-center py-8 text-muted-foreground">加载中...</div>
            ) : accounts.length === 0 ? (
              <div className="text-center py-8 text-muted-foreground">暂无账号</div>
            ) : (
              <div className="space-y-4">
                <div className="overflow-x-auto">
                  <table className="w-full">
                    <thead>
                      <tr className="border-b">
                        <th className="text-left p-4 font-medium">手机号</th>
                        <th className="text-left p-4 font-medium">状态</th>
                        <th className="text-left p-4 font-medium">健康度</th>
                        <th className="text-left p-4 font-medium">代理</th>
                        <th className="text-left p-4 font-medium">最后使用</th>
                        <th className="text-right p-4 font-medium">操作</th>
                      </tr>
                    </thead>
                    <tbody>
                      {accounts.map((account) => (
                        <tr key={account.id} className="border-b hover:bg-muted/50">
                          <td className="p-4">
                            <div className="font-medium">{account.phone}</div>
                            {account.note && (
                              <div className="text-sm text-muted-foreground">{account.note}</div>
                            )}
                          </td>
                          <td className="p-4">
                            <div className="flex items-center gap-2">
                              {getStatusIcon(account.status)}
                              <span
                                className={`px-2 py-1 rounded-full text-xs font-medium ${getStatusColor(
                                  account.status
                                )}`}
                              >
                                {account.status}
                              </span>
                            </div>
                          </td>
                          <td className="p-4">
                            <div className="flex items-center gap-2">
                              <div className="flex-1 h-2 bg-muted rounded-full overflow-hidden max-w-24">
                                <div
                                  className="h-full bg-green-500"
                                  style={{ width: `${(account.health_score || 0) * 100}%` }}
                                />
                              </div>
                              <span className="text-sm font-medium">
                                {((account.health_score || 0) * 100).toFixed(0)}%
                              </span>
                            </div>
                          </td>
                          <td className="p-4">
                            <div className="text-sm">
                              {account.proxy_id ? (
                                <span className="text-muted-foreground">已绑定</span>
                              ) : (
                                <span className="text-muted-foreground">未绑定</span>
                              )}
                            </div>
                          </td>
                          <td className="p-4">
                            <div className="text-sm text-muted-foreground">
                              {account.last_used_at
                                ? new Date(account.last_used_at).toLocaleDateString()
                                : "从未使用"}
                            </div>
                          </td>
                          <td className="p-4 text-right">
                            <Button variant="ghost" size="icon">
                              <MoreVertical className="h-4 w-4" />
                            </Button>
                          </td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                </div>

                {/* Pagination */}
                <div className="flex items-center justify-between pt-4">
                  <div className="text-sm text-muted-foreground">
                    共 {total} 个账号
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
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </MainLayout>
  )
}

