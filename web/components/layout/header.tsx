"use client"

import { Bell, Moon, Sun, Settings, User, LogOut, HelpCircle } from "lucide-react"
import { Button } from "@/components/ui/button"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { SidebarTrigger } from "@/components/ui/sidebar"
import { useTheme } from "next-themes"
import { motion } from "framer-motion"
import { useUser } from "@/contexts/user-context"
import { Badge } from "@/components/ui/badge"

export function Header() {
  const { theme, setTheme } = useTheme()
  const { user, loading, logout } = useUser()

  const getInitials = (name: string) => {
    if (!name) return "U"
    return name.substring(0, 2).toUpperCase()
  }

  return (
    <motion.header 
      initial={{ y: -10, opacity: 0 }}
      animate={{ y: 0, opacity: 1 }}
      transition={{ duration: 0.2 }}
      className="sticky top-0 z-30 flex h-14 items-center gap-4 border-b border-border/50 bg-background/95 backdrop-blur-sm px-4 lg:px-6"
    >
      <div className="flex flex-1 items-center gap-4">
        <SidebarTrigger className="h-8 w-8 hover:bg-muted rounded-lg transition-colors" />
        
        {/* 面包屑或页面标识可以放这里 */}
        <div className="hidden md:flex items-center gap-2 text-sm text-muted-foreground">
          <span>欢迎使用 TG Cloud</span>
        </div>
      </div>

      <div className="flex items-center gap-1">
        {/* 主题切换 */}
        <Button
          variant="ghost"
          size="icon"
          onClick={() => setTheme(theme === "light" ? "dark" : "light")}
          className="h-9 w-9 rounded-lg hover:bg-muted transition-colors"
        >
          <Sun className="h-[18px] w-[18px] rotate-0 scale-100 transition-transform dark:-rotate-90 dark:scale-0" />
          <Moon className="absolute h-[18px] w-[18px] rotate-90 scale-0 transition-transform dark:rotate-0 dark:scale-100" />
          <span className="sr-only">切换主题</span>
        </Button>

        {/* 帮助 */}
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9 rounded-lg hover:bg-muted transition-colors"
        >
          <HelpCircle className="h-[18px] w-[18px] text-muted-foreground" />
          <span className="sr-only">帮助</span>
        </Button>

        {/* 通知 */}
        <Button 
          variant="ghost" 
          size="icon" 
          className="h-9 w-9 rounded-lg hover:bg-muted transition-colors relative"
        >
          <Bell className="h-[18px] w-[18px] text-muted-foreground" />
          <span className="absolute top-1.5 right-1.5 h-2 w-2 rounded-full bg-primary" />
          <span className="sr-only">通知</span>
        </Button>

        {/* 分隔线 */}
        <div className="h-6 w-px bg-border mx-2" />

        {/* 用户菜单 */}
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button 
              variant="ghost" 
              className="relative h-9 gap-2 rounded-lg px-2 hover:bg-muted transition-colors"
            >
              <Avatar className="h-7 w-7">
                <AvatarFallback className="bg-primary/10 text-primary text-xs font-medium">
                  {loading ? "..." : getInitials(user?.username || "")}
                </AvatarFallback>
              </Avatar>
              <span className="hidden md:inline-block text-sm font-medium">
                {loading ? "加载中" : user?.username || "用户"}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent 
            className="w-56 rounded-xl shadow-lg border-border/50" 
            align="end" 
            sideOffset={8}
          >
            <DropdownMenuLabel className="font-normal px-3 py-2">
              <div className="flex flex-col space-y-1">
                <p className="text-sm font-medium">
                  {loading ? "加载中..." : user?.username || "用户名"}
                </p>
                <p className="text-xs text-muted-foreground">
                  {loading ? "" : user?.email || ""}
                </p>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem className="gap-2 px-3 py-2 cursor-pointer">
              <User className="h-4 w-4 text-muted-foreground" />
              <span>个人资料</span>
            </DropdownMenuItem>
            <DropdownMenuItem className="gap-2 px-3 py-2 cursor-pointer">
              <Settings className="h-4 w-4 text-muted-foreground" />
              <span>账号设置</span>
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem 
              onClick={logout}
              className="gap-2 px-3 py-2 cursor-pointer text-destructive focus:text-destructive"
            >
              <LogOut className="h-4 w-4" />
              <span>退出登录</span>
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
    </motion.header>
  )
}
