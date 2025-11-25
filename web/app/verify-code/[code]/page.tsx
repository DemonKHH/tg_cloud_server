"use client"

import { useEffect, useState } from "react"
import { verifyCodeAPI } from "@/lib/api"
import { Card, CardContent, CardHeader, CardTitle, CardDescription } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { AlertCircle, CheckCircle2, XCircle, Copy, RefreshCw } from "lucide-react"
import { toast } from "sonner"
import { motion } from "framer-motion"

interface VerifyCodePageProps {
  params: {
    code: string
  }
}

export default function VerifyCodePage({ params }: VerifyCodePageProps) {
  const { code } = params
  const [status, setStatus] = useState<'waiting' | 'success' | 'timeout' | 'error'>('waiting')
  const [result, setResult] = useState<any>(null)
  const [error, setError] = useState<string>("")
  const [countdown, setCountdown] = useState(60)

  useEffect(() => {
    if (code) {
      startPolling()
    }
  }, [code])

  useEffect(() => {
    let timer: NodeJS.Timeout
    if (status === 'waiting' && countdown > 0) {
      timer = setInterval(() => {
        setCountdown(prev => prev - 1)
      }, 1000)
    }
    return () => clearInterval(timer)
  }, [status, countdown])

  const startPolling = async () => {
    setStatus('waiting')
    setError("")
    setCountdown(60)

    try {
      // The backend blocks for up to 60 seconds (or specified timeout)
      const verifyRes = await verifyCodeAPI.getCode(code, 60)

      if (verifyRes.code === 0 && verifyRes.data && verifyRes.data.success) {
        setResult(verifyRes.data)
        setStatus('success')
        toast.success("验证码获取成功")
      } else {
        // Timeout or other errors
        setStatus('timeout')
        setError(verifyRes.data?.message || verifyRes.msg || "获取验证码超时")
      }
    } catch (error: any) {
      // Request timeout or network error
      setStatus('timeout')
      setError(error.message || "请求超时")
    }
  }

  const handleCopyCode = () => {
    if (result?.code) {
      navigator.clipboard.writeText(result.code)
      toast.success("验证码已复制")
    }
  }

  return (
    <div className="min-h-screen flex items-center justify-center bg-gray-50 dark:bg-gray-900 p-4">
      <motion.div
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.5 }}
        className="w-full max-w-md"
      >
        <Card className="border-none shadow-xl bg-white/80 dark:bg-gray-800/80 backdrop-blur-lg">
          <CardHeader className="text-center pb-2">
            <CardTitle className="text-2xl font-bold bg-gradient-to-r from-blue-600 to-purple-600 bg-clip-text text-transparent">
              Telegram 验证码
            </CardTitle>
            <CardDescription>
              请在其他设备登录该账号触发验证码
            </CardDescription>
          </CardHeader>
          <CardContent className="pt-6 pb-8">
            <div className="flex flex-col items-center justify-center space-y-6">

              {status === 'waiting' && (
                <div className="flex flex-col items-center space-y-4 text-center w-full">
                  <div className="relative">
                    <div className="animate-spin rounded-full h-16 w-16 border-4 border-primary/30 border-t-primary"></div>
                    <div className="absolute inset-0 flex items-center justify-center text-sm font-mono font-bold text-primary">
                      {countdown}s
                    </div>
                  </div>
                  <div className="space-y-2">
                    <p className="font-medium text-lg">正在等待验证码...</p>
                    <p className="text-sm text-muted-foreground">系统正在监听来自 Telegram 的消息</p>
                  </div>
                </div>
              )}

              {status === 'success' && result && (
                <div className="flex flex-col items-center space-y-6 w-full">
                  <motion.div
                    initial={{ scale: 0 }}
                    animate={{ scale: 1 }}
                    className="h-16 w-16 rounded-full bg-green-100 dark:bg-green-900/30 flex items-center justify-center text-green-600 dark:text-green-400"
                  >
                    <CheckCircle2 className="h-8 w-8" />
                  </motion.div>

                  <div className="text-center space-y-1">
                    <p className="font-medium text-green-600 dark:text-green-400 text-lg">获取成功</p>
                    <p className="text-xs text-muted-foreground">
                      来自: {result.sender}
                    </p>
                    <p className="text-xs text-muted-foreground">
                      {new Date(result.received_at * 1000).toLocaleString()}
                    </p>
                  </div>

                  <div className="flex items-center gap-2 w-full p-4 bg-muted/50 rounded-xl border border-primary/10">
                    <code className="flex-1 text-center text-4xl font-mono font-bold tracking-[0.2em] text-primary">
                      {result.code}
                    </code>
                  </div>

                  <Button onClick={handleCopyCode} className="w-full h-12 text-lg btn-modern shadow-lg shadow-primary/20">
                    <Copy className="h-5 w-5 mr-2" />
                    复制验证码
                  </Button>
                </div>
              )}

              {status === 'timeout' && (
                <div className="flex flex-col items-center space-y-6 text-center w-full">
                  <div className="h-16 w-16 rounded-full bg-yellow-100 dark:bg-yellow-900/30 flex items-center justify-center text-yellow-600 dark:text-yellow-400">
                    <AlertCircle className="h-8 w-8" />
                  </div>
                  <div className="space-y-2">
                    <p className="font-medium text-yellow-600 dark:text-yellow-400 text-lg">获取超时</p>
                    <p className="text-sm text-muted-foreground max-w-[250px] mx-auto">{error}</p>
                  </div>
                  <Button
                    onClick={startPolling}
                    className="w-full h-12 btn-modern"
                    variant="outline"
                  >
                    <RefreshCw className="h-4 w-4 mr-2" />
                    重试
                  </Button>
                </div>
              )}

              {status === 'error' && (
                <div className="flex flex-col items-center space-y-6 text-center w-full">
                  <div className="h-16 w-16 rounded-full bg-red-100 dark:bg-red-900/30 flex items-center justify-center text-red-600 dark:text-red-400">
                    <XCircle className="h-8 w-8" />
                  </div>
                  <div className="space-y-2">
                    <p className="font-medium text-red-600 dark:text-red-400 text-lg">获取失败</p>
                    <p className="text-sm text-muted-foreground max-w-[250px] mx-auto">{error}</p>
                  </div>
                  <Button
                    onClick={startPolling}
                    className="w-full h-12 btn-modern"
                    variant="outline"
                  >
                    <RefreshCw className="h-4 w-4 mr-2" />
                    重试
                  </Button>
                </div>
              )}

            </div>
          </CardContent>
        </Card>

        <div className="mt-8 text-center text-xs text-muted-foreground">
          <p>此页面为公开链接，请勿泄露给他人</p>
          <p className="mt-1">© 2024 TG Cloud Server</p>
        </div>
      </motion.div>
    </div>
  )
}
