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
import { accountAPI } from "@/lib/api"
import { useState, useEffect, useRef } from "react"

export default function AccountsPage() {
  const [accounts, setAccounts] = useState<any[]>([])
  const [loading, setLoading] = useState(true)
  const [page, setPage] = useState(1)
  const [total, setTotal] = useState(0)
  const [search, setSearch] = useState("")
  const [uploadDialogOpen, setUploadDialogOpen] = useState(false)
  const [uploading, setUploading] = useState(false)
  const fileInputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    loadAccounts()
  }, [page])

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
      const response = await accountAPI.uploadFiles(file)
      
      if (response.data) {
        const data = response.data as any
        const created = data.created || 0
        const failed = data.failed || 0
        const total = data.total || 0

        if (created > 0) {
          toast.success(`成功创建 ${created} 个账号${failed > 0 ? `，失败 ${failed} 个` : ''}`)
          setUploadDialogOpen(false)
          loadAccounts() // 重新加载账号列表
          
          // 如果有错误信息，显示详细信息
          if (failed > 0 && data.errors && data.errors.length > 0) {
            console.warn("创建账号时的错误：", data.errors)
          }
        } else {
          toast.error(`未能创建任何账号。${data.errors?.length > 0 ? '错误：' + data.errors.join(', ') : ''}`)
        }
      } else {
        toast.success("文件上传成功")
        setUploadDialogOpen(false)
        loadAccounts()
      }
    } catch (error: any) {
      console.error("上传账号文件失败:", error)
      toast.error(error.message || "上传账号文件失败")
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
              <DialogContent>
                <DialogHeader>
                  <DialogTitle>上传账号文件</DialogTitle>
                  <DialogDescription>
                    支持上传 .zip、.session 文件或 tdata 文件夹。系统将自动解析并创建账号。
                  </DialogDescription>
                </DialogHeader>
                <div className="space-y-4 py-4">
                  <div className="border-2 border-dashed border-muted-foreground/25 rounded-lg p-8 text-center">
                    <FileArchive className="h-12 w-12 mx-auto mb-4 text-muted-foreground" />
                    <p className="text-sm text-muted-foreground mb-2">
                      支持的文件格式：
                    </p>
                    <ul className="text-xs text-muted-foreground space-y-1 mb-4">
                      <li>• .zip 压缩包（可包含多个账号文件）</li>
                      <li>• .session 文件（gotd/td格式）</li>
                      <li>• tdata 文件夹（Telegram Desktop格式）</li>
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
                        正在解析文件，请稍候...
                      </p>
                    )}
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

