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
import { Plus, Copy, ExternalLink, Clock, Smartphone, AlertCircle, CheckCircle2, XCircle, Search, RefreshCw } from "lucide-react"
import {
  Tooltip,
  TooltipContent,
  TooltipProvider,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { Badge } from "@/components/ui/badge"
import { ModernTable } from "@/components/ui/modern-table"
import { cn } from "@/lib/utils"
import { verifyCodeAPI, accountAPI } from "@/lib/api"
import { useState, useEffect, useRef } from "react"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Label } from "@/components/ui/label"
import { PageHeader } from "@/components/common/page-header"
import { FilterBar } from "@/components/common/filter-bar"
import { motion } from "framer-motion"

interface Account {
  id: number
  phone: string
  status: string
  health_score: number
  proxy_id?: number
  created_at: string
}

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
  const [accounts, setAccounts] = useState<Account[]>([])
  const [sessions, setSessions] = useState<VerifyCodeSession[]>([])
  const [loading, setLoading] = useState(false)
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  
  // 生成表单状态
  const [selectedAccountId, setSelectedAccountId] = useState<string>("")
  const [expiresIn, setExpiresIn] = useState<string>("300") // 默认5分钟
  const [generatingCode, setGeneratingCode] = useState(false)

  // 搜索状态
  const [searchKeyword, setSearchKeyword] = useState("")
  const [statusFilter, setStatusFilter] = useState("all")

  const fileInputRef = useRef<HTMLInputElement>(null)

  // 加载账号列表
  const loadAccounts = async () => {
    try {
      setLoading(true)
      const response = await accountAPI.list({ limit: 1000 }) // 获取所有账号
      if (response.data?.data) {
        setAccounts(response.data.data)
      }
    } catch (error: any) {
      toast.error(error?.response?.data?.msg || "获取账号列表失败")
    } finally {
      setLoading(false)
    }
  }

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
      } catch (error) {
        console.error('Failed to parse sessions from localStorage:', error)
        setSessions([])
      }
    } else {
      setSessions([])
    }
  }

  // 生成验证码链接
  const handleGenerateCode = async () => {
    if (!selectedAccountId) {
      toast.error("请选择账号")
      return
    }

    const accountIdNum = parseInt(selectedAccountId)
    const expiresInNum = parseInt(expiresIn)

    if (isNaN(expiresInNum) || expiresInNum < 60 || expiresInNum > 3600) {
      toast.error("过期时间必须在60-3600秒之间")
      return
    }

    try {
      setGeneratingCode(true)
      const response = await verifyCodeAPI.generate({
        account_id: accountIdNum,
        expires_in: expiresInNum,
      })

      if (response.data) {
        const { code, url, expires_at, expires_in } = response.data
        
        // 找到对应的账号信息
        const account = accounts.find(acc => acc.id === accountIdNum)
        
        const newSession: VerifyCodeSession = {
          code,
          url: `${window.location.origin}/verify-code/${code}`, // 完整URL
          account_id: accountIdNum,
          account_phone: account?.phone || 'Unknown',
          expires_at,
          expires_in,
          created_at: new Date().toISOString(),
        }

        // 保存到本地存储
        const existingSessions = sessions.filter(s => s.code !== code) // 防重复
        const updatedSessions = [newSession, ...existingSessions]
        localStorage.setItem('verifyCodeSessions', JSON.stringify(updatedSessions))
        setSessions(updatedSessions)

        // 复制链接到剪贴板
        await navigator.clipboard.writeText(newSession.url)
        
        toast.success("验证码链接已生成并复制到剪贴板")
        setCreateDialogOpen(false)
        
        // 重置表单
        setSelectedAccountId("")
        setExpiresIn("300")
      }
    } catch (error: any) {
      console.error('Generate code error:', error)
      const errorMessage = error?.response?.data?.msg || error.message || "生成验证码链接失败"
      toast.error(errorMessage)
    } finally {
      setGeneratingCode(false)
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
        color: minutes > 2 ? "default" as const : "warning" as const
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

  // 表格列定义
  const columns = [
    {
      key: "account_phone",
      title: "账号",
      render: (session: VerifyCodeSession) => (
        <div className="flex items-center gap-2">
          <Smartphone className="h-4 w-4 text-muted-foreground" />
          <span className="font-medium">{session.account_phone}</span>
        </div>
      )
    },
    {
      key: "code",
      title: "访问代码",
      render: (session: VerifyCodeSession) => (
        <div className="flex items-center gap-2">
          <code className="px-2 py-1 bg-muted rounded text-sm font-mono">
            {session.code.substring(0, 8)}...
          </code>
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => copyToClipboard(session.code, "访问代码")}
                >
                  <Copy className="h-3 w-3" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>复制代码</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      )
    },
    {
      key: "url",
      title: "访问链接",
      render: (session: VerifyCodeSession) => (
        <div className="flex items-center gap-2">
          <span className="text-sm text-muted-foreground truncate max-w-[200px]">
            {session.url}
          </span>
          <div className="flex gap-1">
            <TooltipProvider>
              <Tooltip>
                <TooltipTrigger asChild>
                  <Button
                    variant="ghost"
                    size="sm"
                    onClick={() => copyToClipboard(session.url, "访问链接")}
                  >
                    <Copy className="h-3 w-3" />
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
                    size="sm"
                    onClick={() => openLink(session.url)}
                  >
                    <ExternalLink className="h-3 w-3" />
                  </Button>
                </TooltipTrigger>
                <TooltipContent>在新标签页打开</TooltipContent>
              </Tooltip>
            </TooltipProvider>
          </div>
        </div>
      )
    },
    {
      key: "expires_at",
      title: "过期时间",
      render: (session: VerifyCodeSession) => {
        const { text, color } = formatExpiration(session.expires_at)
        return (
          <div className="flex items-center gap-2">
            <Clock className="h-4 w-4 text-muted-foreground" />
            <Badge variant={color}>{text}</Badge>
          </div>
        )
      }
    },
    {
      key: "actions",
      title: "操作",
      render: (session: VerifyCodeSession) => (
        <div className="flex items-center gap-1">
          <TooltipProvider>
            <Tooltip>
              <TooltipTrigger asChild>
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => deleteSession(session.code)}
                  className="hover:bg-red-50 text-red-600 hover:text-red-700"
                >
                  <XCircle className="h-4 w-4" />
                </Button>
              </TooltipTrigger>
              <TooltipContent>删除会话</TooltipContent>
            </Tooltip>
          </TooltipProvider>
        </div>
      )
    }
  ]

  // 初始化
  useEffect(() => {
    loadAccounts()
    refreshSessions()
    
    // 定时刷新过期状态
    const interval = setInterval(refreshSessions, 5000) // 每5秒刷新一次
    return () => clearInterval(interval)
  }, [])

  return (
    <MainLayout>
      <div className="space-y-6">
        <PageHeader
          title="验证码管理"
          description="管理Telegram账号验证码获取链接"
        />

        <div className="flex items-center justify-between">
          <FilterBar className="flex-1 max-w-md">
            <div className="flex items-center gap-2">
              <Search className="h-4 w-4 text-muted-foreground" />
              <Input
                placeholder="搜索账号或代码..."
                value={searchKeyword}
                onChange={(e) => setSearchKeyword(e.target.value)}
                className="border-0 shadow-none focus-visible:ring-0"
              />
            </div>
            <Select value={statusFilter} onValueChange={setStatusFilter}>
              <SelectTrigger className="w-32 border-0 shadow-none focus:ring-0">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="all">全部</SelectItem>
                <SelectItem value="active">有效</SelectItem>
                <SelectItem value="expired">已过期</SelectItem>
              </SelectContent>
            </Select>
          </FilterBar>

          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              onClick={refreshSessions}
              className="gap-2"
            >
              <RefreshCw className="h-4 w-4" />
              刷新
            </Button>
            
            <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
              <DialogTrigger asChild>
                <Button className="gap-2">
                  <Plus className="h-4 w-4" />
                  生成验证码链接
                </Button>
              </DialogTrigger>
              <DialogContent className="sm:max-w-md">
                <DialogHeader>
                  <DialogTitle>生成验证码访问链接</DialogTitle>
                  <DialogDescription>
                    选择账号并设置链接过期时间，生成后可分享给用户获取验证码
                  </DialogDescription>
                </DialogHeader>
                
                <div className="space-y-4 pt-4">
                  <div className="space-y-2">
                    <Label htmlFor="account">选择账号</Label>
                    <Select value={selectedAccountId} onValueChange={setSelectedAccountId}>
                      <SelectTrigger>
                        <SelectValue placeholder="请选择要生成验证码的账号" />
                      </SelectTrigger>
                      <SelectContent>
                        {accounts
                          .filter(account => account.status === 'normal' || account.status === 'new')
                          .map((account) => (
                          <SelectItem key={account.id} value={account.id.toString()}>
                            <div className="flex items-center gap-2">
                              <span>{account.phone}</span>
                              <Badge variant={account.status === 'normal' ? 'default' : 'secondary'}>
                                {account.status}
                              </Badge>
                            </div>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label htmlFor="expires">过期时间</Label>
                    <Select value={expiresIn} onValueChange={setExpiresIn}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="300">5分钟</SelectItem>
                        <SelectItem value="600">10分钟</SelectItem>
                        <SelectItem value="900">15分钟</SelectItem>
                        <SelectItem value="1800">30分钟</SelectItem>
                        <SelectItem value="3600">1小时</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="flex items-start gap-2 p-3 bg-blue-50 rounded-lg">
                    <AlertCircle className="h-4 w-4 text-blue-600 mt-0.5 flex-shrink-0" />
                    <div className="text-sm text-blue-800">
                      <p className="font-medium">使用说明：</p>
                      <ul className="mt-1 space-y-1 text-xs">
                        <li>• 生成的链接可以多次使用，直到过期</li>
                        <li>• 用户访问链接即可获取该账号的验证码</li>
                        <li>• 建议设置合适的过期时间保证安全</li>
                      </ul>
                    </div>
                  </div>

                  <div className="flex justify-end gap-2 pt-2">
                    <Button
                      variant="outline"
                      onClick={() => setCreateDialogOpen(false)}
                      disabled={generatingCode}
                    >
                      取消
                    </Button>
                    <Button
                      onClick={handleGenerateCode}
                      disabled={generatingCode || !selectedAccountId}
                      className="gap-2"
                    >
                      {generatingCode ? (
                        <>
                          <RefreshCw className="h-4 w-4 animate-spin" />
                          生成中...
                        </>
                      ) : (
                        <>
                          <Plus className="h-4 w-4" />
                          生成链接
                        </>
                      )}
                    </Button>
                  </div>
                </div>
              </DialogContent>
            </Dialog>
          </div>
        </div>

        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.5 }}
        >
          <ModernTable
            data={filteredSessions}
            columns={columns}
            loading={loading}
            emptyState={{
              icon: <Smartphone className="h-12 w-12 text-muted-foreground" />,
              title: "暂无验证码会话",
              description: "点击"生成验证码链接"创建新的验证码访问会话"
            }}
          />
        </motion.div>

        {filteredSessions.length > 0 && (
          <div className="text-sm text-muted-foreground text-center">
            显示 {filteredSessions.length} 个验证码会话
          </div>
        )}
      </div>
    </MainLayout>
  )
}
