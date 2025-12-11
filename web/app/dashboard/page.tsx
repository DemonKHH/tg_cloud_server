"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, ListTodo, Bot, TrendingUp, Zap, Shield, Clock, ArrowUpRight } from "lucide-react"
import { statsAPI } from "@/lib/api"
import { useEffect, useState, useMemo } from "react"
import {
  StatsCard,
  ModernLineChart,
  ModernAreaChart,
  ModernBarChart,
  ModernPieChart
} from "@/components/charts/modern-charts"
import { motion } from "framer-motion"
import Link from "next/link"

export default function DashboardPage() {
  const [dashboard, setDashboard] = useState<any>(null)
  const [overview, setOverview] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      const [dashboardRes, overviewRes] = await Promise.all([
        statsAPI.getDashboard(),
        statsAPI.getOverview("week"),
      ])

      if (dashboardRes.data) setDashboard(dashboardRes.data)
      if (overviewRes.data) setOverview(overviewRes.data)
    } catch (error) {
      toast.error("加载数据失败")
      console.error("加载数据失败:", error)
    } finally {
      setLoading(false)
    }
  }

  const activityData = useMemo(() => {
    if (!dashboard?.performance_metrics?.tasks_per_hour) return []
    return dashboard.performance_metrics.tasks_per_hour.map((item: any) => ({
      name: item.label,
      value: item.value
    }))
  }, [dashboard])

  const taskStatusData = useMemo(() => [
    { name: '已完成', value: dashboard?.quick_stats?.completed_tasks || 0 },
    { name: '运行中', value: dashboard?.quick_stats?.running_tasks || 0 },
    { name: '等待中', value: dashboard?.quick_stats?.pending_tasks || 0 },
    { name: '失败', value: dashboard?.quick_stats?.failed_tasks || 0 },
  ], [dashboard])

  const growthData = useMemo(() => {
    if (!dashboard?.performance_metrics?.account_growth) return []
    return dashboard.performance_metrics.account_growth.map((item: any) => ({
      name: new Date(item.timestamp).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' }),
      value: item.value
    }))
  }, [dashboard])

  const performanceData = useMemo(() => {
    if (!overview?.system_health) return []
    return [
      { name: '系统评分', value: overview.system_health.overall_score || 0 },
      { name: '账号健康', value: overview.system_health.accounts_health || 0 },
      { name: '任务成功率', value: overview.system_health.tasks_success_rate || 0 },
      { name: '代理可用率', value: overview.system_health.proxies_active_rate || 0 },
    ]
  }, [overview])

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-[60vh]">
          <div className="flex flex-col items-center gap-3">
            <motion.div
              animate={{ rotate: 360 }}
              transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
              className="w-8 h-8 border-2 border-primary border-t-transparent rounded-full"
            />
            <span className="text-sm text-muted-foreground">加载中...</span>
          </div>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* Page Header */}
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
        >
          <h1 className="page-title gradient-text">仪表盘</h1>
          <p className="page-subtitle">实时监控您的 Telegram 账号和任务状态</p>
        </motion.div>

        {/* Stats Grid */}
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.1 }}
          >
            <StatsCard
              title="总账号数"
              value={dashboard?.quick_stats?.total_accounts || 0}
              change={dashboard?.performance_metrics?.account_growth?.length > 1 ?
                `+${(dashboard.performance_metrics.account_growth[dashboard.performance_metrics.account_growth.length - 1].value - dashboard.performance_metrics.account_growth[0].value)} 本周` :
                "暂无数据"}
              changeType="positive"
              icon={<Users className="h-5 w-5" />}
            />
          </motion.div>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.15 }}
          >
            <StatsCard
              title="今日任务"
              value={dashboard?.quick_stats?.today_tasks || 0}
              change={dashboard?.quick_stats?.success_rate ? `${dashboard.quick_stats.success_rate.toFixed(0)}% 成功率` : "暂无数据"}
              changeType="positive"
              icon={<Zap className="h-5 w-5" />}
            />
          </motion.div>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.2 }}
          >
            <StatsCard
              title="活跃账号"
              value={dashboard?.quick_stats?.active_accounts || 0}
              change={dashboard?.quick_stats?.total_accounts ? `${((dashboard.quick_stats.active_accounts / dashboard.quick_stats.total_accounts) * 100).toFixed(0)}% 活跃率` : "暂无数据"}
              changeType="positive"
              icon={<Shield className="h-5 w-5" />}
            />
          </motion.div>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.25 }}
          >
            <StatsCard
              title="完成任务"
              value={dashboard?.quick_stats?.completed_tasks || 0}
              change={dashboard?.quick_stats?.failed_tasks ? `${dashboard.quick_stats.failed_tasks} 失败` : "无失败"}
              changeType={dashboard?.quick_stats?.failed_tasks > 0 ? "negative" : "positive"}
              icon={<Clock className="h-5 w-5" />}
            />
          </motion.div>
        </div>

        {/* Charts Grid */}
        <div className="grid gap-4 lg:grid-cols-2">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.3 }}
          >
            <ModernLineChart
              data={activityData}
              title="系统活动趋势"
              description="最近7天的系统使用情况"
              height={280}
            />
          </motion.div>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.35 }}
          >
            <ModernPieChart
              data={taskStatusData}
              title="任务状态分布"
              description="当前任务执行状态"
              height={280}
            />
          </motion.div>
        </div>

        <div className="grid gap-4 lg:grid-cols-2">
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.4 }}
          >
            <ModernAreaChart
              data={growthData}
              title="账号增长趋势"
              description="过去一段时间的账号增长"
              height={280}
            />
          </motion.div>
          <motion.div
            initial={{ opacity: 0, y: 20 }}
            animate={{ opacity: 1, y: 0 }}
            transition={{ delay: 0.45 }}
          >
            <ModernBarChart
              data={performanceData}
              title="系统性能指标"
              description="各项性能指标评分"
              height={280}
            />
          </motion.div>
        </div>

        {/* Quick Actions */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.5 }}
        >
          <Card className="border-border/50">
            <CardHeader className="pb-4">
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <Zap className="h-4 w-4 text-primary" />
                快速操作
              </CardTitle>
              <CardDescription>常用功能快捷入口</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-3 md:grid-cols-3">
                {[
                  {
                    href: "/accounts",
                    icon: Users,
                    title: "账号管理",
                    description: "添加和管理TG账号",
                    className: "stat-card-blue"
                  },
                  {
                    href: "/tasks",
                    icon: ListTodo,
                    title: "任务中心",
                    description: "创建和执行批量任务",
                    className: "stat-card-green"
                  },
                  {
                    href: "/ai",
                    icon: Bot,
                    title: "AI服务",
                    description: "智能助手和自动化",
                    className: "stat-card-purple"
                  }
                ].map((action, index) => (
                  <Link
                    key={action.title}
                    href={action.href}
                    className={`group relative flex items-center gap-4 p-4 rounded-xl ${action.className} border border-transparent hover:border-primary/20 transition-all duration-200 hover-card`}
                  >
                    <div className="p-2.5 rounded-lg bg-background/80 shadow-sm group-hover:shadow transition-shadow">
                      <action.icon className="h-5 w-5 text-primary" />
                    </div>
                    <div className="flex-1 min-w-0">
                      <div className="font-medium text-sm group-hover:text-primary transition-colors">
                        {action.title}
                      </div>
                      <div className="text-xs text-muted-foreground mt-0.5 truncate">
                        {action.description}
                      </div>
                    </div>
                    <ArrowUpRight className="h-4 w-4 text-muted-foreground/50 group-hover:text-primary transition-colors" />
                  </Link>
                ))}
              </div>
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </MainLayout>
  )
}
