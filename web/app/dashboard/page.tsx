"use client"

import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, ListTodo, Globe, Activity, TrendingUp, AlertCircle } from "lucide-react"
import { statsAPI } from "@/lib/api"
import { useEffect, useState } from "react"

export default function DashboardPage() {
  const [overview, setOverview] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const response = await statsAPI.getOverview("week")
      if (response.data) {
        setOverview(response.data)
      }
    } catch (error) {
      console.error("加载数据失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const stats = [
    {
      title: "总账号数",
      value: overview?.total_accounts || 0,
      icon: Users,
      trend: "+12%",
      description: "较上月增长",
    },
    {
      title: "活跃任务",
      value: overview?.active_tasks || 0,
      icon: ListTodo,
      trend: "+5%",
      description: "运行中",
    },
    {
      title: "代理数量",
      value: overview?.total_proxies || 0,
      icon: Globe,
      trend: "+8%",
      description: "可用代理",
    },
    {
      title: "健康账号",
      value: overview?.healthy_accounts || 0,
      icon: Activity,
      trend: "98%",
      description: "健康率",
    },
  ]

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <div className="text-muted-foreground">加载中...</div>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <div>
          <h1 className="text-3xl font-bold tracking-tight">仪表盘</h1>
          <p className="text-muted-foreground mt-1">
            欢迎回来，这里是您的系统概览
          </p>
        </div>

        {/* Stats Grid */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          {stats.map((stat) => (
            <Card key={stat.title}>
              <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                <CardTitle className="text-sm font-medium">{stat.title}</CardTitle>
                <stat.icon className="h-4 w-4 text-muted-foreground" />
              </CardHeader>
              <CardContent>
                <div className="text-2xl font-bold">{stat.value}</div>
                <p className="text-xs text-muted-foreground mt-1">
                  <span className="text-green-600">{stat.trend}</span> {stat.description}
                </p>
              </CardContent>
            </Card>
          ))}
        </div>

        {/* Charts and Recent Activity */}
        <div className="grid gap-4 md:grid-cols-2">
          {/* Task Status Chart */}
          <Card>
            <CardHeader>
              <CardTitle>任务状态分布</CardTitle>
              <CardDescription>最近7天的任务执行情况</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {[
                  { label: "已完成", value: overview?.completed_tasks || 0, color: "bg-green-500" },
                  { label: "运行中", value: overview?.active_tasks || 0, color: "bg-blue-500" },
                  { label: "等待中", value: overview?.queued_tasks || 0, color: "bg-yellow-500" },
                  { label: "失败", value: overview?.failed_tasks || 0, color: "bg-red-500" },
                ].map((item) => (
                  <div key={item.label} className="space-y-2">
                    <div className="flex justify-between text-sm">
                      <span>{item.label}</span>
                      <span className="font-medium">{item.value}</span>
                    </div>
                    <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                      <div
                        className={`h-full ${item.color} transition-all`}
                        style={{
                          width: `${
                            overview?.total_tasks
                              ? (item.value / overview.total_tasks) * 100
                              : 0
                          }%`,
                        }}
                      />
                    </div>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>

          {/* Account Health */}
          <Card>
            <CardHeader>
              <CardTitle>账号健康度</CardTitle>
              <CardDescription>账号状态概览</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="space-y-4">
                {[
                  {
                    label: "健康",
                    value: overview?.healthy_accounts || 0,
                    color: "text-green-600",
                  },
                  {
                    label: "警告",
                    value: overview?.warning_accounts || 0,
                    color: "text-yellow-600",
                  },
                  {
                    label: "异常",
                    value: overview?.error_accounts || 0,
                    color: "text-red-600",
                  },
                ].map((item) => (
                  <div key={item.label} className="flex items-center justify-between">
                    <div className="flex items-center gap-2">
                      <div className={`h-2 w-2 rounded-full ${item.color.replace("text", "bg")}`} />
                      <span className="text-sm">{item.label}</span>
                    </div>
                    <span className={`font-medium ${item.color}`}>{item.value}</span>
                  </div>
                ))}
              </div>
            </CardContent>
          </Card>
        </div>

        {/* Quick Actions */}
        <Card>
          <CardHeader>
            <CardTitle>快速操作</CardTitle>
            <CardDescription>常用功能快捷入口</CardDescription>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-3">
              <a
                href="/accounts"
                className="flex items-center gap-3 p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <Users className="h-5 w-5" />
                <div>
                  <div className="font-medium">添加账号</div>
                  <div className="text-sm text-muted-foreground">管理TG账号</div>
                </div>
              </a>
              <a
                href="/tasks"
                className="flex items-center gap-3 p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <ListTodo className="h-5 w-5" />
                <div>
                  <div className="font-medium">创建任务</div>
                  <div className="text-sm text-muted-foreground">执行批量操作</div>
                </div>
              </a>
              <a
                href="/proxies"
                className="flex items-center gap-3 p-4 rounded-lg border hover:bg-accent transition-colors"
              >
                <Globe className="h-5 w-5" />
                <div>
                  <div className="font-medium">配置代理</div>
                  <div className="text-sm text-muted-foreground">管理代理IP</div>
                </div>
              </a>
            </div>
          </CardContent>
        </Card>
      </div>
    </MainLayout>
  )
}

