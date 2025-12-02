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

interface VerifyCodeSession {
  code: string
  url: string
  account_id: number
  account_phone: string
  expires_at: string
  expires_in: number
  created_at: string
}

export default function VerifyCodesPage() {
  const [sessions, setSessions] = useState<VerifyCodeSession[]>([])


  // 搜索状态
  const [searchKeyword, setSearchKeyword] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")

  // 批量操作状态
  const [selectedCodes, setSelectedCodes] = useState<string[]>([])
  const [batchDeleteDialogOpen, setBatchDeleteDialogOpen] = useState(false)

  const fileInputRef = useRef<HTMLInputElement>(null)

  // 刷新会话列表（从本地存储获取）
  const refreshSessions = () => {
    const savedSessions = localStorage.getItem('verifyCodeSessions')
    if (savedSessions) {
      try {
        const parsed = JSON.parse(savedSessions) as VerifyCodeSession[]
        // 过滤掉过期的会话
        const now = new Date()
        const validSessions = parsed.filter(session => new Date(session.expires_at) > now)

        if (validSessions.length !== parsed.length) {
          // 有过期会话，更新本地存储
          localStorage.setItem('verifyCodeSessions', JSON.stringify(validSessions))
        }

        setSessions(validSessions)
        // 清理已选择但不存在的会话
        setSelectedCodes(prev => prev.filter(code => validSessions.some(s => s.code === code)))
      } catch (error) {
        console.error('Failed to parse sessions from localStorage:', error)
        setSessions([])
      }
    } else {
      setSessions([])
    }
  }



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
    const updatedSessions = sessions.filter(s => s.code !== code)
    setSessions(updatedSessions)
    localStorage.setItem('verifyCodeSessions', JSON.stringify(updatedSessions))
    toast.success("验证码会话已删除")
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
    const updatedSessions = sessions.filter(s => !selectedCodes.includes(s.code))
    setSessions(updatedSessions)
    localStorage.setItem('verifyCodeSessions', JSON.stringify(updatedSessions))
    toast.success(`已删除 ${selectedCodes.length} 个会话`)
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
  const formatExpiration = (expiresAt: string) => {
    const date = new Date(expiresAt)
    const now = new Date()
    const diff = date.getTime() - now.getTime()

    if (diff <= 0) {
      return { text: "已过期", color: "destructive" as const }
    }

    const minutes = Math.floor(diff / (1000 * 60))
    const seconds = Math.floor((diff % (1000 * 60)) / 1000)

    if (minutes > 0) {
      return {
        text: `${minutes}分${seconds}秒后过期`,
        color: minutes > 2 ? "default" as const : "secondary" as const
      }
    } else {
      return {
        text: `${seconds}秒后过期`,
        color: "destructive" as const
      }
    }
  }

  // 过滤会话
  const filteredSessions = sessions.filter(session => {
    const matchesSearch =
      session.account_phone.toLowerCase().includes(searchKeyword.toLowerCase()) ||
      session.code.toLowerCase().includes(searchKeyword.toLowerCase())

    if (statusFilter === "all") return matchesSearch

    const now = new Date()
    const expiresAt = new Date(session.expires_at)
    const isExpired = expiresAt <= now

    if (statusFilter === "active") return matchesSearch && !isExpired
    if (statusFilter === "expired") return matchesSearch && isExpired

    return matchesSearch
  })



  // 初始化
  useEffect(() => {
    refreshSessions()

    // 定时刷新过期状态
    const interval = setInterval(refreshSessions, 5000) // 每5秒刷新一次
    return () => clearInterval(interval)
  }, [])

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
            onSearchChange={setSearchKeyword}
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
              onClick={refreshSessions}
              className="gap-2"
            >
              <RefreshCw className="h-4 w-4" />
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
                  <TableHead>访问代码</TableHead>
                  <TableHead>访问链接</TableHead>
                  <TableHead>过期时间</TableHead>
                  <TableHead className="text-right">操作</TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {filteredSessions.length === 0 ? (
                  <TableRow>
                    <TableCell colSpan={6} className="h-48 text-center">
                      <div className="flex flex-col items-center justify-center text-muted-foreground">
                        <Smartphone className="h-12 w-12 mb-4 opacity-20" />
                        <p className="text-lg font-medium mb-2">暂无验证码会话</p>
                        <p className="text-sm opacity-70">点击"生成验证码链接"创建新的验证码访问会话</p>
                      </div>
                    </TableCell>
                  </TableRow>
                ) : (
                  filteredSessions.map((session) => (
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
                        <div className="flex items-center gap-2">
                          <code className="px-2 py-1 bg-muted rounded text-sm font-mono border">
                            {session.code.substring(0, 8)}...
                          </code>
                          <TooltipProvider>
                            <Tooltip>
                              <TooltipTrigger asChild>
                                <Button
                                  variant="ghost"
                                  size="icon"
                                  className="h-8 w-8 opacity-0 group-hover:opacity-100 transition-opacity"
                                  onClick={() => copyToClipboard(session.code, "访问代码")}
                                >
                                  <Copy className="h-3.5 w-3.5" />
                                </Button>
                              </TooltipTrigger>
                              <TooltipContent>复制代码</TooltipContent>
                            </Tooltip>
                          </TooltipProvider>
                        </div>
                      </TableCell>
                      <TableCell>
                        <div className="flex items-center gap-2">
                          <span className="text-sm text-muted-foreground truncate max-w-[200px] font-mono bg-muted/30 px-2 py-1 rounded">
                            {session.url}
                          </span>
                          <div className="flex gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                            <TooltipProvider>
                              <Tooltip>
                                <TooltipTrigger asChild>
                                  <Button
                                    variant="ghost"
                                    size="icon"
                                    className="h-8 w-8"
                                    onClick={() => copyToClipboard(session.url, "访问链接")}
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
                                    onClick={() => openLink(session.url)}
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
                  ))
                )}
              </TableBody>
            </Table>
          </div>
        </motion.div>

        {filteredSessions.length > 0 && (
          <div className="text-sm text-muted-foreground text-center">
            显示 {filteredSessions.length} 个验证码会话
          </div>
        )}

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
