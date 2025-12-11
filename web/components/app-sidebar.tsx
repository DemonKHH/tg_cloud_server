"use client"

import Link from "next/link"
import { usePathname } from "next/navigation"
import {
  LayoutDashboard,
  Users,
  ListTodo,
  Globe,
  Bot,
  BarChart3,
  Settings,
  Zap,
  Shield,
  ChevronRight,
} from "lucide-react"
import { motion } from "framer-motion"

import {
  Sidebar,
  SidebarContent,
  SidebarFooter,
  SidebarGroup,
  SidebarGroupContent,
  SidebarHeader,
  SidebarMenu,
  SidebarMenuButton,
  SidebarMenuItem,
} from "@/components/ui/sidebar"
import { Badge } from "@/components/ui/badge"
import { cn } from "@/lib/utils"

const navigation = [
  { name: "仪表盘", href: "/dashboard", icon: LayoutDashboard, badge: null },
  { name: "账号管理", href: "/accounts", icon: Users, badge: null },
  { name: "任务管理", href: "/tasks", icon: ListTodo, badge: null },
  { name: "API链接", href: "/verify-codes", icon: Shield, badge: null },
  { name: "代理管理", href: "/proxies", icon: Globe, badge: null },
  { name: "AI服务", href: "/ai", icon: Bot, badge: "新" },
  { name: "统计分析", href: "/stats", icon: BarChart3, badge: null },
]

const bottomNavigation = [
  { name: "系统设置", href: "/settings", icon: Settings },
]

export function AppSidebar() {
  const pathname = usePathname()

  return (
    <Sidebar variant="inset" className="border-r border-sidebar-border/50">
      {/* Logo Header */}
      <SidebarHeader className="border-b border-sidebar-border/50">
        <Link href="/dashboard" className="flex items-center gap-3 px-3 py-4">
          <motion.div
            className="flex h-9 w-9 items-center justify-center rounded-xl bg-gradient-to-br from-primary to-primary/80 shadow-lg shadow-primary/25"
            whileHover={{ scale: 1.05, rotate: 3 }}
            transition={{ type: "spring", stiffness: 400, damping: 17 }}
          >
            <Zap className="h-5 w-5 text-primary-foreground" />
          </motion.div>
          <div className="flex flex-col group-data-[collapsible=icon]:hidden">
            <span className="text-lg font-bold tracking-tight gradient-text">
              TG Cloud
            </span>
            <span className="text-[10px] text-muted-foreground -mt-0.5">
              账号管理平台
            </span>
          </div>
        </Link>
      </SidebarHeader>

      {/* Main Navigation */}
      <SidebarContent className="px-2 py-4">
        <SidebarGroup>
          <SidebarGroupContent>
            <SidebarMenu className="space-y-1">
              {navigation.map((item, index) => {
                const isActive = pathname === item.href || pathname?.startsWith(`${item.href}/`)
                return (
                  <motion.div
                    key={item.name}
                    initial={{ opacity: 0, x: -10 }}
                    animate={{ opacity: 1, x: 0 }}
                    transition={{ delay: 0.05 * index, duration: 0.2 }}
                  >
                    <SidebarMenuItem>
                      <SidebarMenuButton
                        asChild
                        isActive={isActive}
                        className={cn(
                          "group relative h-10 rounded-lg transition-all duration-200",
                          isActive 
                            ? "bg-primary/10 text-primary font-medium" 
                            : "hover:bg-muted/80"
                        )}
                      >
                        <Link href={item.href} className="flex items-center gap-3 px-3">
                          <item.icon className={cn(
                            "h-[18px] w-[18px] transition-colors",
                            isActive ? "text-primary" : "text-muted-foreground group-hover:text-foreground"
                          )} />
                          <span className="flex-1 text-sm">{item.name}</span>
                          {item.badge && (
                            <Badge
                              variant="default"
                              className="h-5 px-1.5 text-[10px] font-medium bg-primary/90"
                            >
                              {item.badge}
                            </Badge>
                          )}
                          {isActive && (
                            <motion.div
                              layoutId="activeIndicator"
                              className="absolute left-0 top-1/2 -translate-y-1/2 w-[3px] h-5 bg-primary rounded-r-full"
                              transition={{ type: "spring", stiffness: 500, damping: 30 }}
                            />
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

      {/* Footer Navigation */}
      <SidebarFooter className="border-t border-sidebar-border/50 p-2">
        <SidebarMenu>
          {bottomNavigation.map((item) => {
            const isActive = pathname === item.href
            return (
              <SidebarMenuItem key={item.name}>
                <SidebarMenuButton
                  asChild
                  isActive={isActive}
                  className={cn(
                    "h-10 rounded-lg transition-all duration-200",
                    isActive 
                      ? "bg-primary/10 text-primary font-medium" 
                      : "hover:bg-muted/80"
                  )}
                >
                  <Link href={item.href} className="flex items-center gap-3 px-3">
                    <item.icon className={cn(
                      "h-[18px] w-[18px]",
                      isActive ? "text-primary" : "text-muted-foreground"
                    )} />
                    <span className="flex-1 text-sm">{item.name}</span>
                    <ChevronRight className="h-4 w-4 text-muted-foreground/50" />
                  </Link>
                </SidebarMenuButton>
              </SidebarMenuItem>
            )
          })}
        </SidebarMenu>
      </SidebarFooter>
    </Sidebar>
  )
}
