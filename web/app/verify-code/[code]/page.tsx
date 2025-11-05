"use client"

import { useState, useEffect } from "react"
import { useParams } from "next/navigation"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Badge } from "@/components/ui/badge"
import { verifyCodeAPI } from "@/lib/api"
import { 
  Smartphone, 
  Clock, 
  Shield, 
  CheckCircle2, 
  XCircle, 
  AlertTriangle, 
  RefreshCw, 
  Copy, 
  ArrowRight,
  Loader2
} from "lucide-react"
import { cn } from "@/lib/utils"
import { motion, AnimatePresence } from "framer-motion"

interface VerifyCodeResponse {
  success: boolean
  code?: string
  sender?: string
  received_at?: string
  wait_seconds?: number
  message: string
}

export default function VerifyCodePage() {
  const params = useParams()
  const code = params.code as string
  
  const [loading, setLoading] = useState(false)
  const [timeout, setTimeout] = useState("60")
  const [result, setResult] = useState<VerifyCodeResponse | null>(null)
  const [countdown, setCountdown] = useState(0)
  const [error, setError] = useState<string>("")

  // 获取验证码
  const handleGetCode = async () => {
    if (!code) {
      toast.error("无效的访问链接")
      return
    }

    const timeoutNum = parseInt(timeout)
    if (isNaN(timeoutNum) || timeoutNum < 10 || timeoutNum > 300) {
      toast.error("超时时间必须在10-300秒之间")
      return
    }

    try {
      setLoading(true)
      setError("")
      setResult(null)
      setCountdown(timeoutNum)

      // 开始倒计时
      const countdownInterval = setInterval(() => {
        setCountdown(prev => {
          if (prev <= 1) {
            clearInterval(countdownInterval)
            return 0
          }
          return prev - 1
        })
      }, 1000)

      const response = await verifyCodeAPI.getCode(code, timeoutNum)
      
      clearInterval(countdownInterval)
      setCountdown(0)

      if (response.data) {
        setResult(response.data)
        if (response.data.success && response.data.code) {
          toast.success("验证码获取成功！")
        } else {
          toast.warning(response.data.message || "验证码获取失败")
        }
      }
    } catch (error: any) {
      setCountdown(0)
      console.error('Get code error:', error)
      
      let errorMessage = "获取验证码失败"
      if (error?.response?.status === 404) {
        errorMessage = "访问链接无效或已过期"
      } else if (error?.response?.status === 403) {
        errorMessage = "用户账号已过期，请联系管理员"
      } else if (error?.response?.data?.msg) {
        errorMessage = error.response.data.msg
      } else if (error.message) {
        errorMessage = error.message
      }
      
      setError(errorMessage)
      toast.error(errorMessage)
    } finally {
      setLoading(false)
    }
  }

  // 复制验证码
  const copyCode = async (text: string) => {
    try {
      await navigator.clipboard.writeText(text)
      toast.success("验证码已复制到剪贴板")
    } catch (error) {
      toast.error("复制失败")
    }
  }

  // 格式化时间
  const formatTime = (dateString: string) => {
    try {
      const date = new Date(dateString)
      return date.toLocaleString('zh-CN', {
        year: 'numeric',
        month: '2-digit',
        day: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
        second: '2-digit'
      })
    } catch (error) {
      return dateString
    }
  }

  // 重置状态
  const reset = () => {
    setResult(null)
    setError("")
    setCountdown(0)
  }

  return (
    <div className="min-h-screen bg-gradient-to-br from-blue-50 via-white to-purple-50 flex items-center justify-center p-4">
      <motion.div
        initial={{ opacity: 0, scale: 0.95 }}
        animate={{ opacity: 1, scale: 1 }}
        transition={{ duration: 0.3 }}
        className="w-full max-w-md"
      >
        <Card className="shadow-xl border-0 bg-white/80 backdrop-blur-sm">
          <CardHeader className="text-center space-y-4">
            <div className="mx-auto w-16 h-16 bg-gradient-to-br from-blue-500 to-purple-600 rounded-2xl flex items-center justify-center">
              <Smartphone className="h-8 w-8 text-white" />
            </div>
            <div>
              <CardTitle className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
                获取Telegram验证码
              </CardTitle>
              <CardDescription className="text-base mt-2">
                点击下方按钮获取验证码，验证码将从Telegram账号中自动提取
              </CardDescription>
            </div>
          </CardHeader>

          <CardContent className="space-y-6">
            {/* 访问代码信息 */}
            <div className="p-4 bg-gradient-to-r from-blue-50 to-purple-50 rounded-lg border">
              <div className="flex items-center gap-2 text-sm text-muted-foreground">
                <Shield className="h-4 w-4" />
                <span>访问代码</span>
              </div>
              <code className="block mt-1 font-mono text-sm bg-white px-2 py-1 rounded border">
                {code}
              </code>
            </div>

            {/* 超时设置 */}
            <div className="space-y-2">
              <label className="text-sm font-medium flex items-center gap-2">
                <Clock className="h-4 w-4" />
                等待超时时间
              </label>
              <select
                value={timeout}
                onChange={(e) => setTimeout(e.target.value)}
                disabled={loading}
                className="w-full px-3 py-2 bg-white border border-gray-200 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent"
              >
                <option value="30">30秒</option>
                <option value="60">60秒</option>
                <option value="120">2分钟</option>
                <option value="180">3分钟</option>
                <option value="300">5分钟</option>
              </select>
              <p className="text-xs text-muted-foreground">
                系统将在指定时间内监听验证码消息
              </p>
            </div>

            {/* 获取按钮 */}
            <Button 
              onClick={handleGetCode}
              disabled={loading}
              className="w-full h-12 text-base font-medium bg-gradient-to-r from-blue-600 to-purple-600 hover:from-blue-700 hover:to-purple-700"
            >
              {loading ? (
                <div className="flex items-center gap-2">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  获取中...
                  {countdown > 0 && <span>({countdown}s)</span>}
                </div>
              ) : (
                <div className="flex items-center gap-2">
                  <Smartphone className="h-4 w-4" />
                  获取验证码
                  <ArrowRight className="h-4 w-4" />
                </div>
              )}
            </Button>

            {/* 进度指示器 */}
            <AnimatePresence>
              {countdown > 0 && (
                <motion.div
                  initial={{ opacity: 0, height: 0 }}
                  animate={{ opacity: 1, height: "auto" }}
                  exit={{ opacity: 0, height: 0 }}
                  className="space-y-2"
                >
                  <div className="flex justify-between text-sm text-muted-foreground">
                    <span>等待验证码...</span>
                    <span>{countdown}s</span>
                  </div>
                  <div className="w-full bg-gray-200 rounded-full h-2 overflow-hidden">
                    <motion.div
                      className="h-full bg-gradient-to-r from-blue-500 to-purple-600 rounded-full"
                      initial={{ width: "100%" }}
                      animate={{ width: "0%" }}
                      transition={{ 
                        duration: parseInt(timeout),
                        ease: "linear"
                      }}
                    />
                  </div>
                </motion.div>
              )}
            </AnimatePresence>

            {/* 结果显示 */}
            <AnimatePresence mode="wait">
              {result && (
                <motion.div
                  key="result"
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                  className={cn(
                    "p-4 rounded-lg border",
                    result.success 
                      ? "bg-green-50 border-green-200" 
                      : "bg-yellow-50 border-yellow-200"
                  )}
                >
                  <div className="flex items-center gap-2 mb-3">
                    {result.success ? (
                      <CheckCircle2 className="h-5 w-5 text-green-600" />
                    ) : (
                      <AlertTriangle className="h-5 w-5 text-yellow-600" />
                    )}
                    <span className={cn(
                      "font-medium",
                      result.success ? "text-green-800" : "text-yellow-800"
                    )}>
                      {result.success ? "验证码获取成功" : "获取结果"}
                    </span>
                  </div>

                  {result.success && result.code && (
                    <div className="space-y-3">
                      <div className="flex items-center justify-between p-3 bg-white rounded border">
                        <div>
                          <div className="text-sm text-muted-foreground">验证码</div>
                          <div className="text-2xl font-bold font-mono text-green-700">
                            {result.code}
                          </div>
                        </div>
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() => copyCode(result.code!)}
                          className="gap-2"
                        >
                          <Copy className="h-3 w-3" />
                          复制
                        </Button>
                      </div>

                      <div className="grid grid-cols-2 gap-3 text-sm">
                        {result.sender && (
                          <div>
                            <div className="text-muted-foreground">发送方</div>
                            <div className="font-medium">{result.sender}</div>
                          </div>
                        )}
                        {result.wait_seconds !== undefined && (
                          <div>
                            <div className="text-muted-foreground">等待时间</div>
                            <div className="font-medium">{result.wait_seconds}秒</div>
                          </div>
                        )}
                        {result.received_at && (
                          <div className="col-span-2">
                            <div className="text-muted-foreground">接收时间</div>
                            <div className="font-medium">{formatTime(result.received_at)}</div>
                          </div>
                        )}
                      </div>
                    </div>
                  )}

                  <div className="mt-3 text-sm text-muted-foreground">
                    {result.message}
                  </div>

                  <Button
                    variant="outline"
                    onClick={reset}
                    className="w-full mt-3"
                  >
                    重新获取
                  </Button>
                </motion.div>
              )}

              {error && (
                <motion.div
                  key="error"
                  initial={{ opacity: 0, y: 20 }}
                  animate={{ opacity: 1, y: 0 }}
                  exit={{ opacity: 0, y: -20 }}
                  className="p-4 bg-red-50 border border-red-200 rounded-lg"
                >
                  <div className="flex items-center gap-2 mb-2">
                    <XCircle className="h-5 w-5 text-red-600" />
                    <span className="font-medium text-red-800">获取失败</span>
                  </div>
                  <div className="text-sm text-red-700 mb-3">{error}</div>
                  <Button
                    variant="outline"
                    onClick={reset}
                    className="w-full"
                  >
                    重试
                  </Button>
                </motion.div>
              )}
            </AnimatePresence>

            {/* 使用说明 */}
            <div className="text-xs text-muted-foreground space-y-1 p-3 bg-gray-50 rounded-lg">
              <div className="font-medium">使用说明：</div>
              <div>• 点击"获取验证码"后，系统将自动监听Telegram消息</div>
              <div>• 请确保在等待期间发起需要验证码的操作</div>
              <div>• 验证码通常在30-60秒内到达</div>
              <div>• 如果长时间未收到，请检查账号状态或重试</div>
            </div>
          </CardContent>
        </Card>
      </motion.div>
    </div>
  )
}
