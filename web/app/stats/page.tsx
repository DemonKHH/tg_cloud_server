"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { TrendingUp, Users, ListTodo, Globe, Activity } from "lucide-react"
import { statsAPI } from "@/lib/api"
import { useState, useEffect } from "react"

export default function StatsPage() {
  const [overview, setOverview] = useState<any>(null)
  const [accountStats, setAccountStats] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadStats()
  }, [])

  const loadStats = async () => {
    try {
      setLoading(true)
      const [overviewRes, accountRes] = await Promise.all([
        statsAPI.getOverview("week"),
        statsAPI.getAccountStats("week"),
      ])
      if (overviewRes.data) setOverview(overviewRes.data)
      if (accountRes.data) setAccountStats(accountRes.data)
    } catch (error) {
      toast.error("加载统计失败，请稍后重试")
      console.error("加载统计失败:", error)
    } finally {
      setLoading(false)
    }
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">统计分析</h1>
          <p className="text-muted-foreground mt-1">查看系统运行数据和趋势</p>
        </div>

        {loading ? (
          <div className="text-center py-12 text-muted-foreground">加载中...</div>
        ) : (
          <>
            {/* Stats Overview */}
            <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
              <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">总账号数</CardTitle>
                  <Users className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{overview?.total_accounts || 0}</div>
                  <p className="text-xs text-muted-foreground mt-1">
                    <TrendingUp className="inline h-3 w-3 text-green-600" /> +12% 较上月
                  </p>
                </CardContent>
              </Card>

              <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">活跃任务</CardTitle>
                  <ListTodo className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{overview?.active_tasks || 0}</div>
                  <p className="text-xs text-muted-foreground mt-1">
                    <TrendingUp className="inline h-3 w-3 text-green-600" /> +5% 较上月
                  </p>
                </CardContent>
              </Card>

              <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">代理数量</CardTitle>
                  <Globe className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">{overview?.total_proxies || 0}</div>
                  <p className="text-xs text-muted-foreground mt-1">
                    <TrendingUp className="inline h-3 w-3 text-green-600" /> +8% 较上月
                  </p>
                </CardContent>
              </Card>

              <Card className="card-shadow hover:card-shadow-lg transition-all duration-300">
                <CardHeader className="flex flex-row items-center justify-between space-y-0 pb-2">
                  <CardTitle className="text-sm font-medium">健康率</CardTitle>
                  <Activity className="h-4 w-4 text-muted-foreground" />
                </CardHeader>
                <CardContent>
                  <div className="text-2xl font-bold">
                    {overview?.health_rate
                      ? `${(overview.health_rate * 100).toFixed(1)}%`
                      : "0%"}
                  </div>
                  <p className="text-xs text-muted-foreground mt-1">
                    健康账号占比
                  </p>
                </CardContent>
              </Card>
            </div>

            {/* Charts */}
            <div className="grid gap-6 md:grid-cols-2">
              <Card className="card-shadow">
                <CardHeader>
                  <CardTitle>账号状态分布</CardTitle>
                  <CardDescription>账号状态统计</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-4">
                    {accountStats?.status_distribution &&
                      Object.entries(accountStats.status_distribution).map(([status, count]: [string, any]) => (
                        <div key={status} className="space-y-2">
                          <div className="flex justify-between text-sm">
                            <span>{status}</span>
                            <span className="font-medium">{count}</span>
                          </div>
                          <div className="h-2 w-full bg-muted rounded-full overflow-hidden">
                            <div
                              className="h-full bg-primary transition-all"
                              style={{
                                width: `${accountStats?.total_accounts ? (count / accountStats.total_accounts) * 100 : 0}%`,
                              }}
                            />
                          </div>
                        </div>
                      ))}
                  </div>
                </CardContent>
              </Card>

              <Card className="card-shadow">
                <CardHeader>
                  <CardTitle>任务执行趋势</CardTitle>
                  <CardDescription>最近7天的任务执行情况</CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="text-center py-8 text-muted-foreground">
                    图表区域（可集成 Chart.js 或 Recharts）
                  </div>
                </CardContent>
              </Card>
            </div>
          </>
        )}
      </div>
    </MainLayout>
  )
}

