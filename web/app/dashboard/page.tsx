"use client"

import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, ListTodo, Globe, Activity, TrendingUp, AlertCircle, Zap, Shield, Clock, Bot } from "lucide-react"
import { statsAPI } from "@/lib/api"
import { useEffect, useState, useMemo } from "react"
import { 
  StatsCard, 
  ModernLineChart, 
  ModernAreaChart, 
  ModernBarChart, 
  ModernPieChart 
} from "@/components/charts/modern-charts"
import { Progress } from "@/components/ui/progress"
import { motion } from "framer-motion"

export default function DashboardPage() {
  const [overview, setOverview] = useState<any>(null)
  const [dashboard, setDashboard] = useState<any>(null)
  const [loading, setLoading] = useState(true)

  useEffect(() => {
    loadData()
  }, [])

  const loadData = async () => {
    try {
      // 同时加载仪表盘数据和概览数据
      const [dashboardRes, overviewRes] = await Promise.all([
        statsAPI.getDashboard(),
        statsAPI.getOverview("week"),
      ])
      
      if (dashboardRes.data) {
        setDashboard(dashboardRes.data)
      }
      if (overviewRes.data) {
        setOverview(overviewRes.data)
      }
    } catch (error) {
      toast.error("加载数据失败，请稍后重试")
      console.error("加载数据失败:", error)
    } finally {
      setLoading(false)
    }
  }

  // 图表数据使用 useMemo 缓存
  const activityData = useMemo(() => [
    { name: '周一', value: 240 },
    { name: '周二', value: 139 },
    { name: '周三', value: 360 },
    { name: '周四', value: 280 },
    { name: '周五', value: 320 },
    { name: '周六', value: 180 },
    { name: '周日', value: 200 },
  ], [])

  const taskStatusData = useMemo(() => [
    { name: '已完成', value: dashboard?.completed_tasks || overview?.completed_tasks || 0 },
    { name: '运行中', value: dashboard?.running_tasks || overview?.running_tasks || 0 },
    { name: '等待中', value: dashboard?.pending_tasks || overview?.pending_tasks || 0 },
    { name: '失败', value: dashboard?.failed_tasks || overview?.failed_tasks || 0 },
  ], [dashboard, overview])

  const growthData = useMemo(() => [
    { name: '1月', value: 120 },
    { name: '2月', value: 190 },
    { name: '3月', value: 300 },
    { name: '4月', value: 250 },
    { name: '5月', value: 420 },
    { name: '6月', value: 380 },
  ], [])

  const performanceData = useMemo(() => [
    { name: '性能', value: 85 },
    { name: '可用性', value: 98 },
    { name: '安全性', value: 92 },
    { name: '响应时间', value: 78 },
  ], [])

  if (loading) {
    return (
      <MainLayout>
        <div className="flex items-center justify-center h-64">
          <motion.div 
            animate={{ rotate: 360 }}
            transition={{ duration: 1, repeat: Infinity, ease: "linear" }}
            className="w-8 h-8 border-4 border-primary border-t-transparent rounded-full"
          />
          <span className="ml-3 text-muted-foreground">加载中...</span>
        </div>
      </MainLayout>
    )
  }

  return (
    <MainLayout>
      <div className="space-y-8 bg-background min-h-screen p-6">
        {/* Page Header */}
        <motion.div
          initial={{ opacity: 0, y: -20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6 }}
          className="space-y-2"
        >
          <h1 className="text-4xl font-bold tracking-tight gradient-text">
            欢迎回到 TG Cloud
          </h1>
          <p className="text-lg text-muted-foreground">
            您的 Telegram 账号管理中心 - 实时监控和智能管理
          </p>
        </motion.div>

        {/* Modern Stats Grid */}
        <div className="grid gap-6 md:grid-cols-2 lg:grid-cols-4">
          <StatsCard
            title="总账号数"
            value={dashboard?.total_accounts || overview?.total_accounts || 0}
            change={dashboard?.account_growth || overview?.account_growth ? `+${dashboard?.account_growth || overview?.account_growth}% 较上月` : "暂无数据"}
            changeType="positive"
            icon={<Users className="h-5 w-5" />}
          />
          <StatsCard
            title="活跃任务"
            value={dashboard?.active_tasks || overview?.active_tasks || 0}
            change={dashboard?.running_tasks || overview?.running_tasks ? `${dashboard?.running_tasks || overview?.running_tasks} 运行中` : "暂无数据"}
            changeType="positive"
            icon={<Zap className="h-5 w-5" />}
          />
          <StatsCard
            title="正常账号"
            value={dashboard?.normal_accounts || overview?.normal_accounts || 0}
            change={dashboard?.normal_rate || overview?.normal_rate ? `${(dashboard?.normal_rate || overview?.normal_rate * 100).toFixed(1)}% 正常率` : "暂无数据"}
            changeType="positive"
            icon={<Shield className="h-5 w-5" />}
          />
          <StatsCard
            title="完成任务"
            value={dashboard?.completed_tasks || overview?.completed_tasks || 0}
            change={dashboard?.today_tasks || overview?.today_tasks ? `${dashboard?.today_tasks || overview?.today_tasks} 今日任务` : "暂无数据"}
            changeType="positive"
            icon={<Clock className="h-5 w-5" />}
          />
        </div>

        {/* Modern Charts Grid */}
        <div className="grid gap-6 lg:grid-cols-2">
          <ModernLineChart 
            data={activityData}
            title="系统活动趋势"
            description="最近7天的系统使用情况"
            height={300}
          />
          <ModernPieChart 
            data={taskStatusData}
            title="任务状态分布"
            description="当前任务执行状态分析"
            height={300}
          />
        </div>

        {/* Additional Analytics */}
        <div className="grid gap-6 lg:grid-cols-2">
          <ModernAreaChart 
            data={growthData}
            title="账号增长趋势"
            description="过去6个月的账号增长情况"
            height={300}
          />
          <ModernBarChart 
            data={performanceData}
            title="系统性能指标"
            description="各项性能指标评分"
            height={300}
          />
        </div>

        {/* Modern Quick Actions */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.6, delay: 0.3 }}
        >
          <Card className="card-shadow-lg glass-effect">
            <CardHeader>
              <CardTitle className="flex items-center gap-2">
                <Zap className="h-5 w-5 text-primary" />
                快速操作
              </CardTitle>
              <CardDescription>常用功能快捷入口</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="grid gap-6 md:grid-cols-3">
                {[
                  {
                    href: "/accounts",
                    icon: Users,
                    title: "账号管理",
                    description: "添加和管理TG账号",
                    color: "from-blue-500/10 to-blue-600/5"
                  },
                  {
                    href: "/tasks",
                    icon: ListTodo,
                    title: "任务中心",
                    description: "创建和执行批量任务",
                    color: "from-green-500/10 to-green-600/5"
                  },
                  {
                    href: "/ai",
                    icon: Bot,
                    title: "AI服务",
                    description: "智能助手和自动化",
                    color: "from-purple-500/10 to-purple-600/5"
                  }
                ].map((action, index) => (
                  <motion.a
                    key={action.title}
                    href={action.href}
                    whileHover={{ scale: 1.02, y: -2 }}
                    whileTap={{ scale: 0.98 }}
                    initial={{ opacity: 0, y: 20 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ duration: 0.3, delay: 0.4 + index * 0.1 }}
                    className={`group relative flex items-center gap-4 p-6 rounded-xl bg-linear-to-br ${action.color} border border-border/50 hover:border-primary/20 transition-all duration-200 cursor-pointer overflow-hidden`}
                  >
                    <div className="absolute inset-0 bg-linear-to-r from-primary/5 to-transparent opacity-0 group-hover:opacity-100 transition-opacity" />
                    <motion.div
                      whileHover={{ scale: 1.1, rotate: 5 }}
                      className="relative z-10 p-3 rounded-lg bg-background/80 shadow-sm"
                    >
                      <action.icon className="h-6 w-6 text-primary" />
                    </motion.div>
                    <div className="relative z-10">
                      <div className="font-semibold text-foreground group-hover:text-primary transition-colors">
                        {action.title}
                      </div>
                      <div className="text-sm text-muted-foreground mt-1">
                        {action.description}
                      </div>
                    </div>
                    <motion.div
                      className="absolute right-4 opacity-0 group-hover:opacity-100 transition-opacity"
                      whileHover={{ x: 2 }}
                    >
                      <TrendingUp className="h-5 w-5 text-primary" />
                    </motion.div>
                  </motion.a>
                ))}
              </div>
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </MainLayout>
  )
}

