"use client"

import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, ListTodo, Globe, Bot, BarChart3, Shield } from "lucide-react"

export default function HomePage() {
  const router = useRouter()

  return (
    <div className="flex min-h-screen flex-col">
      {/* Header */}
      <header className="border-b">
        <div className="container flex h-16 items-center justify-between px-4 lg:px-6">
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-bold">TG Cloud</h1>
          </div>
          <div className="flex items-center gap-4">
            <Button variant="ghost" onClick={() => router.push("/login")}>
              登录
            </Button>
            <Button onClick={() => router.push("/login")}>开始使用</Button>
          </div>
        </div>
      </header>

      {/* Hero Section */}
      <section className="container flex flex-col items-center justify-center gap-6 py-12 md:py-24 lg:py-32">
        <div className="flex flex-col items-center gap-4 text-center">
          <h1 className="text-4xl font-bold tracking-tight sm:text-6xl md:text-7xl">
            TG账号批量管理系统
          </h1>
          <p className="max-w-[700px] text-lg text-muted-foreground sm:text-xl">
            专业的Telegram账号管理和批量操作平台，支持账号检查、私信、群发、验证码接收和AI群聊等功能
          </p>
          <div className="flex gap-4 mt-4">
            <Button size="lg" onClick={() => router.push("/login")}>
              立即开始
            </Button>
            <Button size="lg" variant="outline">
              了解更多
            </Button>
          </div>
        </div>
      </section>

      {/* Features */}
      <section className="container py-12 md:py-24">
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-3">
          <Card>
            <CardHeader>
              <Users className="h-10 w-10 mb-2 text-primary" />
              <CardTitle>账号管理</CardTitle>
              <CardDescription>
                统一管理您的TG账号，支持健康检查、状态监控和代理配置
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <ListTodo className="h-10 w-10 mb-2 text-primary" />
              <CardTitle>任务调度</CardTitle>
              <CardDescription>
                智能任务队列，支持批量操作和实时监控
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <Globe className="h-10 w-10 mb-2 text-primary" />
              <CardTitle>代理管理</CardTitle>
              <CardDescription>
                支持HTTP/HTTPS/SOCKS5代理，灵活配置和自动测试
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <Bot className="h-10 w-10 mb-2 text-primary" />
              <CardTitle>AI服务</CardTitle>
              <CardDescription>
                智能回复、情感分析、关键词提取等功能
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <BarChart3 className="h-10 w-10 mb-2 text-primary" />
              <CardTitle>数据分析</CardTitle>
              <CardDescription>
                实时统计和可视化，全面掌握系统运行状态
              </CardDescription>
            </CardHeader>
          </Card>

          <Card>
            <CardHeader>
              <Shield className="h-10 w-10 mb-2 text-primary" />
              <CardTitle>安全可靠</CardTitle>
              <CardDescription>
                JWT认证、RBAC权限控制、数据加密等多重安全保障
              </CardDescription>
            </CardHeader>
          </Card>
        </div>
      </section>

      {/* Footer */}
      <footer className="border-t py-6">
        <div className="container flex flex-col items-center justify-between gap-4 md:flex-row px-4 lg:px-6">
          <p className="text-sm text-muted-foreground">
            © 2024 TG Cloud. All rights reserved.
          </p>
        </div>
      </footer>
    </div>
  )
}
