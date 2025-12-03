"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  Users,
  ListTodo,
  Globe,
  FileText,
  Bot,
  BarChart3,
  Settings,
  Zap,
  Shield,
} from "lucide-react"
import { motion, AnimatePresence } from "framer-motion"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarGroupLabel,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
  SidebarTrigger,
} from "@/components/ui/sidebar"
import { Badge } from "@/components/ui/badge"
import { Tooltip, TooltipContent, TooltipProvider, TooltipTrigger } from "@/components/ui/tooltip"

const navigation = [
  { name: "仪表盘", href: "/dashboard", icon: LayoutDashboard, badge: null },
  { name: "账号管理", href: "/accounts", icon: Users, badge: null },
  { name: "任务管理", href: "/tasks", icon: ListTodo, badge: "3" },
  { name: "API链接管理", href: "/verify-codes", icon: Shield, badge: null },
  { name: "代理管理", href: "/proxies", icon: Globe, badge: null },

  { name: "AI服务", href: "/ai", icon: Bot, badge: "新" },
  { name: "统计分析", href: "/stats", icon: BarChart3, badge: null },
  { name: "系统设置", href: "/settings", icon: Settings, badge: null },
]

export function AppSidebar() {
  const pathname = usePathname()

  return (
    <TooltipProvider>
      <Sidebar variant="inset" className="border-r border-sidebar-border">
        {/* Header */}
        <SidebarHeader>
          <div className="flex items-center gap-2 px-4 py-2">
            <motion.div
              className="flex h-8 w-8 items-center justify-center rounded-lg bg-primary"
              whileHover={{ rotate: 5, scale: 1.1 }}
              transition={{ type: "spring", stiffness: 400 }}
            >
              <Zap className="h-4 w-4 text-primary-foreground" />
            </motion.div>
            <h1 className="text-xl font-bold gradient-text group-data-[collapsible=icon]:hidden">
              TG Cloud
            </h1>
          </div>
        </SidebarHeader>

        {/* Content */}
        <SidebarContent>
          <SidebarGroup>
            <SidebarGroupLabel>导航菜单</SidebarGroupLabel>
            <SidebarGroupContent>
              <SidebarMenu>
                {navigation.map((item, index) => {
                  const isActive = pathname === item.href || pathname?.startsWith(`${item.href}/`)
                  return (
                    <motion.div
                      key={item.name}
                      initial={{ opacity: 0, x: -20 }}
                      animate={{ opacity: 1, x: 0 }}
                      transition={{ delay: 0.1 + index * 0.05 }}
                    >
                      <SidebarMenuItem>
                        <SidebarMenuButton
                          asChild
                          isActive={isActive}
                          className="group relative flex items-center gap-3 rounded-lg text-sm font-medium transition-all duration-200 hover:shadow-sm hover:scale-[1.02]"
                        >
                          <Link href={item.href} className="flex items-center gap-3 w-full">
                            <motion.div
                              whileHover={{ scale: 1.1, rotate: 5 }}
                              transition={{ type: "spring", stiffness: 400 }}
                            >
                              <item.icon className="h-5 w-5" />
                            </motion.div>
                            <span className="flex-1">{item.name}</span>
                            {item.badge && (
                              <motion.div
                                initial={{ scale: 0 }}
                                animate={{ scale: 1 }}
                                transition={{ delay: 0.2 + index * 0.05 }}
                              >
                                <Badge
                                  variant={item.badge === "新" ? "default" : "secondary"}
                                  className="h-5 text-xs px-1.5"
                                >
                                  {item.badge}
                                </Badge>
                              </motion.div>
                            )}
                          </Link>
                        </SidebarMenuButton>
                      </SidebarMenuItem>
                    </motion.div>
                  )
                })}
              </SidebarMenu>
            </SidebarGroupContent>
          </SidebarGroup>
        </SidebarContent>
      </Sidebar>
    </TooltipProvider>
  )
}
