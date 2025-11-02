"use client"

import { useRouter } from "next/navigation"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Users, ListTodo, Globe, Bot, BarChart3, Shield, Zap, ArrowRight, Star, Play } from "lucide-react"
import { motion } from "framer-motion"

export default function HomePage() {
  const router = useRouter()

  return (
    <div className="flex min-h-screen flex-col bg-background">
      {/* Modern Header */}
      <motion.header 
        initial={{ y: -50, opacity: 0 }}
        animate={{ y: 0, opacity: 1 }}
        transition={{ duration: 0.6 }}
        className="border-b bg-background/95 backdrop-blur-sm sticky top-0 z-50 w-full"
      >
        <div className="flex h-16 items-center justify-between px-4 lg:px-6 max-w-7xl mx-auto w-full">
          <motion.div 
            whileHover={{ scale: 1.05 }}
            className="flex items-center gap-2"
          >
            <motion.div
              className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary"
              whileHover={{ rotate: 5 }}
              transition={{ type: "spring", stiffness: 400 }}
            >
              <Zap className="h-4 w-4 text-primary-foreground" />
            </motion.div>
            <h1 className="text-xl font-bold gradient-text">TG Cloud</h1>
          </motion.div>
          <div className="flex items-center gap-4">
            <Button 
              variant="ghost" 
              onClick={() => router.push("/login")}
              className="btn-modern"
            >
              登录
            </Button>
            <Button 
              onClick={() => router.push("/login")}
              className="btn-modern"
            >
              开始使用
              <ArrowRight className="ml-2 h-4 w-4" />
            </Button>
          </div>
        </div>
      </motion.header>

      {/* Modern Hero Section */}
      <section className="relative flex items-center justify-center min-h-[85vh] overflow-hidden bg-background">
        {/* Background Animation */}
        <div className="absolute inset-0 -z-10">
          <div className="absolute top-1/4 left-1/4 w-64 h-64 bg-primary/8 rounded-full blur-3xl animate-pulse" />
          <div className="absolute bottom-1/4 right-1/4 w-80 h-80 bg-primary/4 rounded-full blur-3xl animate-pulse delay-1000" />
        </div>
        
        <div className="container mx-auto px-4 lg:px-6 w-full">
          <div className="max-w-5xl mx-auto text-center space-y-8">
            
            {/* 产品标签 */}
            <motion.div
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              transition={{ delay: 0.2, type: "spring", stiffness: 200 }}
              className="flex items-center justify-center gap-2 px-4 py-2 rounded-full bg-primary/10 text-primary text-sm font-medium mx-auto w-fit"
            >
              <Star className="h-4 w-4" />
              专业级 Telegram 管理平台
            </motion.div>

            {/* 主标题 */}
            <motion.div
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.4, duration: 0.8 }}
              className="space-y-2"
            >
              <h1 className="text-4xl font-bold tracking-tight sm:text-5xl md:text-6xl lg:text-7xl leading-tight">
                <span className="gradient-text block">智能化</span>
                <span className="text-foreground block">TG账号批量管理系统</span>
              </h1>
            </motion.div>

            {/* 描述文字 */}
            <motion.p 
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.6, duration: 0.8 }}
              className="text-lg text-muted-foreground leading-relaxed sm:text-xl max-w-3xl mx-auto"
            >
              集成 <span className="text-primary font-semibold">AI智能助手</span> 的专业平台，
              支持账号健康检测、批量任务执行、智能代理管理和实时数据分析
            </motion.p>

            {/* 操作按钮 */}
            <motion.div 
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 0.8, duration: 0.8 }}
              className="flex flex-col sm:flex-row gap-4 justify-center items-center"
            >
              <Button 
                size="lg" 
                onClick={() => router.push("/login")}
                className="btn-modern text-lg px-8 py-6 h-auto"
              >
                <Play className="mr-2 h-5 w-5" />
                立即体验
              </Button>
              <Button 
                size="lg" 
                variant="outline" 
                className="btn-modern text-lg px-8 py-6 h-auto glass-effect"
              >
                观看演示
                <ArrowRight className="ml-2 h-5 w-5" />
              </Button>
            </motion.div>

            {/* 统计数据 */}
            <motion.div 
              initial={{ opacity: 0, y: 20 }}
              animate={{ opacity: 1, y: 0 }}
              transition={{ delay: 1, duration: 0.8 }}
              className="grid grid-cols-1 sm:grid-cols-3 gap-8 mt-16 pt-8 border-t border-border/20 max-w-2xl mx-auto"
            >
              {[
                { label: "活跃用户", value: "10K+" },
                { label: "管理账号", value: "500K+" },
                { label: "成功率", value: "99.9%" }
              ].map((stat) => (
                <motion.div
                  key={stat.label}
                  whileHover={{ scale: 1.05 }}
                  className="text-center space-y-2"
                >
                  <div className="text-3xl font-bold text-primary">{stat.value}</div>
                  <div className="text-sm text-muted-foreground">{stat.label}</div>
                </motion.div>
              ))}
            </motion.div>
          </div>
        </div>
      </section>

      {/* Modern Features */}
      <section className="py-20 md:py-32 bg-background">
        <div className="container mx-auto px-4 lg:px-6">
          <motion.div 
            initial={{ opacity: 0, y: 20 }}
            whileInView={{ opacity: 1, y: 0 }}
            transition={{ duration: 0.6 }}
            viewport={{ once: true }}
            className="text-center mb-16 max-w-4xl mx-auto"
          >
            <h2 className="text-3xl font-bold tracking-tight mb-6 sm:text-4xl md:text-5xl">
              强大功能，<span className="gradient-text">一站式解决</span>
            </h2>
            <p className="text-lg text-muted-foreground sm:text-xl leading-relaxed">
              从账号管理到智能自动化，TG Cloud 为您提供完整的 Telegram 营销解决方案
            </p>
          </motion.div>

          <div className="grid gap-8 md:grid-cols-2 lg:grid-cols-3 max-w-6xl mx-auto">
            {[
            {
              icon: Users,
              title: "智能账号管理",
              description: "统一管理海量TG账号，实时健康检测、状态监控和风险预警",
              features: ["健康度评估", "自动检测", "批量导入"],
              color: "from-blue-500/10 to-blue-600/5"
            },
            {
              icon: Bot,
              title: "AI智能助手",
              description: "集成GPT模型，智能回复、内容生成、情感分析一键搞定",
              features: ["智能回复", "内容生成", "情感分析"],
              color: "from-purple-500/10 to-purple-600/5"
            },
            {
              icon: ListTodo,
              title: "任务自动化",
              description: "可视化任务调度，支持定时任务、批量操作和智能重试",
              features: ["定时执行", "批量操作", "智能重试"],
              color: "from-green-500/10 to-green-600/5"
            },
            {
              icon: Globe,
              title: "代理管理",
              description: "支持多协议代理，自动检测可用性，智能负载均衡",
              features: ["多协议支持", "健康检测", "负载均衡"],
              color: "from-orange-500/10 to-orange-600/5"
            },
            {
              icon: BarChart3,
              title: "数据分析",
              description: "实时数据可视化，深度性能分析，助您做出明智决策",
              features: ["实时监控", "数据可视化", "趋势分析"],
              color: "from-cyan-500/10 to-cyan-600/5"
            },
            {
              icon: Shield,
              title: "企业级安全",
              description: "多层安全防护，数据加密传输，确保您的数据绝对安全",
              features: ["数据加密", "权限控制", "安全审计"],
              color: "from-red-500/10 to-red-600/5"
            }
          ].map((feature, index) => (
            <motion.div
              key={feature.title}
              initial={{ opacity: 0, y: 30 }}
              whileInView={{ opacity: 1, y: 0 }}
              transition={{ duration: 0.5, delay: index * 0.1 }}
              viewport={{ once: true }}
              whileHover={{ scale: 1.02, y: -5 }}
              className="group"
            >
              <Card className={`card-shadow hover:card-shadow-lg transition-all duration-300 h-full bg-linear-to-br ${feature.color} border-border/50 hover:border-primary/20`}>
                <CardHeader className="pb-4">
                  <motion.div
                    whileHover={{ scale: 1.1, rotate: 5 }}
                    className="w-12 h-12 rounded-lg bg-primary/10 flex items-center justify-center mb-4"
                  >
                    <feature.icon className="h-6 w-6 text-primary" />
                  </motion.div>
                  <CardTitle className="text-xl group-hover:text-primary transition-colors">
                    {feature.title}
                  </CardTitle>
                  <CardDescription className="text-base leading-relaxed">
                    {feature.description}
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="space-y-2">
                    {feature.features.map((item) => (
                      <div key={item} className="flex items-center gap-2 text-sm text-muted-foreground">
                        <div className="w-1.5 h-1.5 rounded-full bg-primary/60" />
                        {item}
                      </div>
                    ))}
                  </div>
                </CardContent>
              </Card>
            </motion.div>
            ))}
          </div>
        </div>
      </section>

      {/* Modern Footer */}
      <motion.footer 
        initial={{ opacity: 0, y: 20 }}
        whileInView={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.6 }}
        viewport={{ once: true }}
        className="border-t bg-muted/50"
      >
        <div className="container mx-auto px-4 lg:px-6 py-12">
          <div className="grid gap-8 md:grid-cols-4 max-w-6xl mx-auto">
            <div className="space-y-4">
              <div className="flex items-center gap-2">
                <motion.div
                  className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary"
                  whileHover={{ rotate: 5 }}
                  transition={{ type: "spring", stiffness: 400 }}
                >
                  <Zap className="h-4 w-4 text-primary-foreground" />
                </motion.div>
                <span className="text-lg font-bold gradient-text">TG Cloud</span>
              </div>
              <p className="text-sm text-muted-foreground">
                专业的 Telegram 账号管理平台，助力您的数字营销成功。
              </p>
            </div>

            <div className="space-y-4">
              <h4 className="font-semibold">产品功能</h4>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div>账号管理</div>
                <div>任务自动化</div>
                <div>AI 智能助手</div>
                <div>数据分析</div>
              </div>
            </div>

            <div className="space-y-4">
              <h4 className="font-semibold">支持</h4>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div>使用文档</div>
                <div>API 文档</div>
                <div>技术支持</div>
                <div>社区论坛</div>
              </div>
            </div>

            <div className="space-y-4">
              <h4 className="font-semibold">联系我们</h4>
              <div className="space-y-2 text-sm text-muted-foreground">
                <div>客服邮箱</div>
                <div>商务合作</div>
                <div>意见反馈</div>
                <div>工单系统</div>
              </div>
            </div>
          </div>

          <div className="mt-12 pt-8 border-t border-border/50 flex flex-col md:flex-row justify-between items-center gap-4 max-w-6xl mx-auto">
            <p className="text-sm text-muted-foreground">
              © 2024 TG Cloud. All rights reserved. 
            </p>
            <div className="flex gap-4 text-sm text-muted-foreground">
              <span className="hover:text-primary transition-colors cursor-pointer">隐私政策</span>
              <span className="hover:text-primary transition-colors cursor-pointer">服务条款</span>
              <span className="hover:text-primary transition-colors cursor-pointer">Cookie 政策</span>
            </div>
          </div>
        </div>
      </motion.footer>
    </div>
  )
}
