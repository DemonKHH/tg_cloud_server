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
import { Copy, ExternalLink, Clock, Smartphone, AlertCircle, CheckCircle2, XCircle, Search, RefreshCw } from "lucide-react"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Badge } from "@/components/ui/badge"
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table"
import { cn } from "@/lib/utils"
import { useState, useEffect, useRef } from "react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

import { PageHeader } from "@/components/common/page-header"
import { FilterBar } from "@/components/common/filter-bar"
import { motion } from "framer-motion"

import { Checkbox } from "@/components/ui/checkbox"
import { verifyCodeAPI } from "@/lib/api"

interface VerifyCodeSession {
  code: string
  url: string
  account_id: number
  account_phone: string
  expires_at: number
  expires_in: number
  created_at: number
}

export default function VerifyCodesPage() {
  const [sessions, setSessions] = useState<VerifyCodeSession[]>([])

  // 分页状态
  const [page, setPage] = useState(1)
  const [limit] = useState(50)
  const [total, setTotal] = useState(0)
  const [loading, setLoading] = useState(false)

  // 搜索状态
  const [searchKeyword, setSearchKeyword] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")

  // 批量操作状态
  const [selectedCodes, setSelectedCodes] = useState<string[]>([])
  const [batchDeleteDialogOpen, setBatchDeleteDialogOpen] = useState(false)
  const [hiddenCodes, setHiddenCodes] = useState<string[]>([])

  // 复制链接
  const copyToClipboard = async (text: string, type: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success(`${type}已复制到剪贴板`)
    } catch (error) {
      toast.error("复制失败")
    }
  }

  // 打开链接
  const openLink = (url: string) => {
    window.open(url, '_blank')
  }

  // 删除会话
  const deleteSession = (code: string) => {
    // 这里应该调用后端API删除，但目前后端没有删除接口，只是前端隐藏
    // 如果需要后端支持，需要添加 DeleteSession 接口
    setHiddenCodes(prev => [...prev, code])
    toast.success("验证码会话已移除")
    if (selectedCodes.includes(code)) {
      setSelectedCodes(prev => prev.filter(c => c !== code))
    }
  }

  // 批量删除会话
  const handleBatchDelete = () => {
    if (selectedCodes.length === 0) return
    setBatchDeleteDialogOpen(true)
  }

  const confirmBatchDelete = () => {
    setHiddenCodes(prev => [...prev, ...selectedCodes])
    toast.success(`已移除 ${selectedCodes.length} 个会话`)
    setSelectedCodes([])
    setBatchDeleteDialogOpen(false)
  }

  // 全选/取消全选
  const toggleSelectAll = () => {
    if (selectedCodes.length === filteredSessions.length) {
      setSelectedCodes([])
    } else {
      setSelectedCodes(filteredSessions.map(s => s.code))
    }
  }

  // 选择/取消选择单个
  const toggleSelectOne = (code: string) => {
    if (selectedCodes.includes(code)) {
      setSelectedCodes(prev => prev.filter(c => c !== code))
    } else {
      setSelectedCodes(prev => [...prev, code])
    }
  }

  // 格式化过期时间
  const formatExpiration = (expiresAt: number) => {
    const date = new Date(expiresAt * 1000)
    const now = new Date()
    const isExpired = date <= now

    const year = date.getFullYear()
    const month = String(date.getMonth() + 1).padStart(2, '0')
    const day = String(date.getDate()).padStart(2, '0')
    const hours = String(date.getHours()).padStart(2, '0')
    const minutes = String(date.getMinutes()).padStart(2, '0')
    const seconds = String(date.getSeconds()).padStart(2, '0')

    const text = `${year}-${month}-${day} ${hours}:${minutes}:${seconds}`

    return {
      text,
      color: isExpired ? "destructive" as const : "outline" as const
    }
  }

  // 过滤会话 (仅本地状态过滤，搜索已由后端处理)
  const filteredSessions = sessions.filter(session => {
    if (hiddenCodes.includes(session.code)) return false

    if (statusFilter === "all") return true

    const now = new Date()
    const expiresAt = new Date(session.expires_at * 1000)
    const isExpired = expiresAt <= now

    if (statusFilter === "active") return !isExpired
    if (statusFilter === "expired") return isExpired

    return true
  })

  // 获取会话列表
  const fetchSessions = async () => {
    try {
      setLoading(true)
      const res = await verifyCodeAPI.listSessions({
        page,
        limit,
        keyword: searchKeyword,
      })

      if (res.data) {
        setSessions(res.data.items)
        setTotal(res.data.pagination.total)
      }
    } catch (error) {
      console.error("Failed to fetch sessions:", error)
      toast.error("获取会话列表失败")
    } finally {
      setLoading(false)
    }
  }

  const refresh = () => {
    fetchSessions()
  }

  // 初始化
  useEffect(() => {
    const timer = setTimeout(() => {
      fetchSessions()
    }, 300)
    return () => clearTimeout(timer)
  }, [page, limit, searchKeyword])

  return (
    <MainLayout>
      <div className="space-y-6">
        <PageHeader
          title="API链接管理"
          description="管理Telegram账号验证码获取链接"
        />

        <div className="flex items-center justify-between">
          <FilterBar
            className="flex-1 max-w-md"
            search={searchKeyword}
            onSearchChange={(val) => setSearchKeyword(val)}
            searchPlaceholder="搜索账号或代码..."
            filters={
              <Select value={statusFilter} onValueChange={setStatusFilter}>
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="all">全部</SelectItem>
                  <SelectItem value="active">有效</SelectItem>
                  <SelectItem value="expired">已过期</SelectItem>
                </SelectContent>
              </Select>
            }
          />

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={refresh}
              className="gap-2"
            >
              <RefreshCw className={`h-4 w-4 ${loading ? 'animate-spin' : ''}`} />
              刷新
            </Button>
          </div>
        </div>

        {/* Batch Actions Bar */}
        {selectedCodes.length > 0 && (
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            exit={{ opacity: 0, y: 20 }}
            className="fixed bottom-6 left-1/2 -translate-x-1/2 z-50 bg-background/95 backdrop-blur-lg border shadow-2xl rounded-2xl p-4 flex flex-col gap-3 min-w-[300px]"
          >
            <div className="flex items-center justify-between border-b pb-2">
              <div className="flex items-center gap-2">
                <div className="bg-primary text-primary-foreground text-xs font-bold rounded-full w-6 h-6 flex items-center justify-center">
                  {selectedCodes.length}
                </div>
                <span className="text-sm font-medium text-muted-foreground">
                  已选择会话
                </span>
              </div>
              <Button
                size="sm"
                variant="ghost"
                className="h-6 text-xs text-muted-foreground hover:text-foreground px-2"
                onClick={() => setSelectedCodes([])}
              >
                取消选择
              </Button>
            </div>

            <div className="flex items-center justify-center gap-2">
              <Button
                size="sm"
                variant="destructive"
                className="w-full"
                onClick={handleBatchDelete}
              >
                <XCircle className="h-4 w-4 mr-2" />
                批量删除 ({selectedCodes.length})
              </Button>
            </div>
          </motion.div>
        )}

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
        >
          <div className="rounded-xl border shadow-sm bg-card overflow-hidden">
            <Table>
              <TableHeader>
                <TableRow className="bg-muted/50 hover:bg-muted/50">
                  <TableHead className="w-[50px]">
                    <Checkbox
                      checked={filteredSessions.length > 0 && selectedCodes.length === filteredSessions.length}
                      onCheckedChange={toggleSelectAll}
                      className="border-2 border-primary data-[state=checked]:bg-primary data-[state=checked]:border-primary"
                    />
                  </TableHead>
                  <TableHead>账号</TableHead>
                  <TableHead>访问链接</TableHead>
                  <TableHead>过期时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredSessions.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={5} className="h-48 text-center">
                      <div className="flex flex-col items-center justify-center text-muted-foreground">
                        <Smartphone className="h-12 w-12 mb-4 opacity-20" />
                        <p className="text-lg font-medium mb-2">暂无验证码会话</p>
                        <p className="text-sm opacity-70">点击"生成验证码链接"创建新的验证码访问会话</p>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredSessions.map((session) => {
                    const fullUrl = typeof window !== 'undefined' ? `${window.location.origin}${session.url}` : session.url
                    return (
                      <TableRow
                        key={session.code}
                        className={cn(
                          "group transition-colors hover:bg-muted/50",
                          selectedCodes.includes(session.code) && "bg-primary/5"
                        )}
                      >
                        <TableCell>
                          <Checkbox
                            checked={selectedCodes.includes(session.code)}
                            onCheckedChange={() => toggleSelectOne(session.code)}
                            className="border-2 border-primary/60 data-[state=checked]:bg-primary data-[state=checked]:border-primary"
                          />
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2">
                            <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center text-primary font-bold text-xs">
                              {session.account_phone.slice(-2)}
                            </div>
                            <span className="font-medium">{session.account_phone}</span>
                          </div>
                        </TableCell>
                        <TableCell>
                          <div className="flex items-center gap-2 max-w-[500px]">
                            <span className="text-sm text-muted-foreground truncate font-mono bg-muted/30 px-2 py-1 rounded flex-1">
                              {fullUrl}
                            </span>
                            <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity shrink-0">
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      className="h-8 w-8"
                                      onClick={() => copyToClipboard(fullUrl, "访问链接")}
                                    >
                                      <Copy className="h-3.5 w-3.5" />
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>复制链接</TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                              <TooltipProvider>
                                <Tooltip>
                                  <TooltipTrigger asChild>
                                    <Button
                                      variant="ghost"
                                      size="icon"
                                      className="h-8 w-8"
                                      onClick={() => openLink(fullUrl)}
                                    >
                                      <ExternalLink className="h-3.5 w-3.5" />
                                    </Button>
                                  </TooltipTrigger>
                                  <TooltipContent>在新标签页打开</TooltipContent>
                                </Tooltip>
                              </TooltipProvider>
                            </div>
                          </div>
                        </TableCell>
                        <TableCell>
                          {(() => {
                            const { text, color } = formatExpiration(session.expires_at)
                            return (
                              <div className="flex items-center gap-2">
                                <Badge variant={color} className="font-normal">
                                  <Clock className="h-3 w-3 mr-1" />
                                  {text}
                                </Badge>
                              </div>
                            )
                          })()}
                        </TableCell>
                        <TableCell className="text-right">
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  onClick={() => deleteSession(session.code)}
                                  className="h-8 w-8 text-muted-foreground hover:text-red-600 hover:bg-red-50"
                                >
                                  <XCircle className="h-4 w-4" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>删除会话</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </TableCell>
                      </TableRow>
                    )
                  })
                )}
              </TableBody>
            </Table>
          </div>
        </motion.div>

        {/* Pagination */}
        <div className="flex items-center justify-between py-4">
          <div className="text-sm text-muted-foreground">
            共 {total} 条记录
          </div>
          <div className="flex items-center space-x-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(p => Math.max(1, p - 1))}
              disabled={page === 1 || loading}
            >
              上一页
            </Button>
            <div className="text-sm font-medium">
              第 {page} 页
            </div>
            <Button
              variant="outline"
              size="sm"
              onClick={() => setPage(p => p + 1)}
              disabled={sessions.length < 50 || loading}
            >
              下一页
            </Button>
          </div>
        </div>

        {/* 批量删除确认对话框 */}
        <Dialog open={batchDeleteDialogOpen} onOpenChange={setBatchDeleteDialogOpen}>
          <DialogContent className="sm:max-w-[400px]">
            <DialogHeader>
              <DialogTitle className="text-xl text-red-600 flex items-center gap-2">
                <AlertCircle className="h-5 w-5" />
                确认批量删除
              </DialogTitle>
              <DialogDescription className="pt-2">
                您确定要删除选中的 <span className="font-semibold text-foreground">{selectedCodes.length}</span> 个会话吗？
                <br />
                <span className="text-red-500 text-xs mt-2 block">
                  此操作将永久删除这些验证码会话，用户将无法再通过旧链接获取验证码。
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
