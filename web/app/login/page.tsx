"use client"

import { useState } from "react"
import { useRouter } from "next/navigation"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { authAPI, apiClient } from "@/lib/api"
import { ResponseCode } from "@/lib/api"

type Mode = "login" | "register"

export default function LoginPage() {
  const router = useRouter()
  const [mode, setMode] = useState<Mode>("login")
  const [username, setUsername] = useState("")
  const [email, setEmail] = useState("")
  const [password, setPassword] = useState("")
  const [confirmPassword, setConfirmPassword] = useState("")
  const [loading, setLoading] = useState(false)

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    setLoading(true)

    try {
      if (mode === "login") {
        // 登录逻辑
        const response = await authAPI.login(username, password)
        if (response.code === ResponseCode.SUCCESS && response.data) {
          const data = response.data as { token: string }
          if (data.token) {
            apiClient.setToken(data.token)
            toast.success("登录成功")
            router.push("/dashboard")
          } else {
            toast.error(response.msg || "登录失败")
          }
        } else {
          toast.error(response.msg || "登录失败")
        }
      } else {
        // 注册逻辑
        if (password !== confirmPassword) {
          toast.error("两次输入的密码不一致")
          setLoading(false)
          return
        }

        if (password.length < 6) {
          toast.error("密码长度至少为6位")
          setLoading(false)
          return
        }

        if (username.length < 3 || username.length > 50) {
          toast.error("用户名长度必须在3-50个字符之间")
          setLoading(false)
          return
        }

        const response = await authAPI.register({
          username,
          email,
          password,
        })

        if (response.code === ResponseCode.SUCCESS) {
          toast.success("注册成功，请登录")
          // 切换到登录模式并清空表单
          setMode("login")
          setUsername("")
          setEmail("")
          setPassword("")
          setConfirmPassword("")
        } else {
          toast.error(response.msg || "注册失败")
        }
      }
    } catch (err) {
      toast.error(err instanceof Error ? err.message : "操作失败，请稍后重试")
    } finally {
      setLoading(false)
    }
  }

  const switchMode = () => {
    setMode(mode === "login" ? "register" : "login")
    // 切换模式时清空表单
    setUsername("")
    setEmail("")
    setPassword("")
    setConfirmPassword("")
  }

  return (
    <div className="flex min-h-screen items-center justify-center bg-gradient-to-br from-background to-muted p-4">
      <Card className="w-full max-w-md">
        <CardHeader className="space-y-1">
          <CardTitle className="text-2xl font-bold text-center">
            {mode === "login" ? "登录" : "注册"}
          </CardTitle>
          <CardDescription className="text-center">
            {mode === "login"
              ? "登录到您的TG Cloud账户"
              : "创建新的TG Cloud账户"}
          </CardDescription>
        </CardHeader>
        <CardContent>
          <form onSubmit={handleSubmit} className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="username" className="text-sm font-medium">
                用户名
              </label>
              <Input
                id="username"
                type="text"
                placeholder={mode === "login" ? "请输入用户名" : "请输入用户名（3-50个字符）"}
                value={username}
                onChange={(e) => setUsername(e.target.value)}
                required
                minLength={mode === "register" ? 3 : undefined}
                maxLength={mode === "register" ? 50 : undefined}
              />
            </div>

            {mode === "register" && (
              <div className="space-y-2">
                <label htmlFor="email" className="text-sm font-medium">
                  邮箱
                </label>
                <Input
                  id="email"
                  type="email"
                  placeholder="请输入邮箱地址"
                  value={email}
                  onChange={(e) => setEmail(e.target.value)}
                  required
                />
              </div>
            )}

            <div className="space-y-2">
              <label htmlFor="password" className="text-sm font-medium">
                密码
              </label>
              <Input
                id="password"
                type="password"
                placeholder={mode === "login" ? "请输入密码" : "请输入密码（至少6位）"}
                value={password}
                onChange={(e) => setPassword(e.target.value)}
                required
                minLength={mode === "register" ? 6 : undefined}
              />
            </div>

            {mode === "register" && (
              <div className="space-y-2">
                <label htmlFor="confirmPassword" className="text-sm font-medium">
                  确认密码
                </label>
                <Input
                  id="confirmPassword"
                  type="password"
                  placeholder="请再次输入密码"
                  value={confirmPassword}
                  onChange={(e) => setConfirmPassword(e.target.value)}
                  required
                  minLength={6}
                />
              </div>
            )}

            <Button type="submit" className="w-full" disabled={loading}>
              {loading
                ? mode === "login"
                  ? "登录中..."
                  : "注册中..."
                : mode === "login"
                ? "登录"
                : "注册"}
            </Button>
          </form>

          <div className="mt-4 text-center text-sm">
            <button
              type="button"
              onClick={switchMode}
              className="text-primary hover:underline focus:outline-none"
            >
              {mode === "login"
                ? "还没有账户？立即注册"
                : "已有账户？立即登录"}
            </button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

