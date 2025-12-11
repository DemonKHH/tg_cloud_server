"use client"

import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Avatar, AvatarFallback } from "@/components/ui/avatar"
import { useUser } from "@/contexts/user-context"
import { useState, useEffect } from "react"
import { toast } from "sonner"
import { authAPI } from "@/lib/api"
import { User, Mail, Shield, Lock } from "lucide-react"
import { motion } from "framer-motion"
import { Badge } from "@/components/ui/badge"

export default function ProfilePage() {
  const { user, loading, refresh } = useUser()
  const [saving, setSaving] = useState(false)
  const [savingPassword, setSavingPassword] = useState(false)
  const [form, setForm] = useState({
    username: "",
    email: "",
  })
  const [passwordForm, setPasswordForm] = useState({
    newPassword: "",
    confirmPassword: "",
  })

  // 初始化表单
  useEffect(() => {
    if (user) {
      setForm({
        username: user.username || "",
        email: user.email || "",
      })
    }
  }, [user])

  const getInitials = (name: string) => {
    if (!name) return "U"
    return name.substring(0, 2).toUpperCase()
  }

  const handleSave = async () => {
    if (!form.username.trim()) {
      toast.error("用户名不能为空")
      return
    }

    setSaving(true)
    try {
      await authAPI.updateProfile({
        username: form.username,
        email: form.email,
      })
      toast.success("个人资料已更新")
      refresh?.()
    } catch (error: any) {
      toast.error(error.message || "更新失败")
    } finally {
      setSaving(false)
    }
  }

  const handleSavePassword = async () => {
    if (!passwordForm.newPassword) {
      toast.error("请输入新密码")
      return
    }

    if (passwordForm.newPassword.length < 6) {
      toast.error("密码长度至少6位")
      return
    }

    if (passwordForm.newPassword !== passwordForm.confirmPassword) {
      toast.error("两次输入的密码不一致")
      return
    }

    setSavingPassword(true)
    try {
      await authAPI.updateProfile({
        password: passwordForm.newPassword,
      })
      toast.success("密码已更新")
      setPasswordForm({ newPassword: "", confirmPassword: "" })
    } catch (error: any) {
      toast.error(error.message || "密码更新失败")
    } finally {
      setSavingPassword(false)
    }
  }

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
      <div className="space-y-6 max-w-2xl">
        {/* Page Header */}
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
        >
          <h1 className="page-title gradient-text">个人资料</h1>
          <p className="page-subtitle">管理您的个人信息和账号安全</p>
        </motion.div>

        {/* Profile Card */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <Card className="border-border/50">
            <CardHeader>
              <div className="flex items-center gap-4">
                <Avatar className="h-16 w-16">
                  <AvatarFallback className="bg-primary/10 text-primary text-xl font-semibold">
                    {getInitials(user?.username || "")}
                  </AvatarFallback>
                </Avatar>
                <div>
                  <CardTitle>{user?.username || "用户"}</CardTitle>
                  <CardDescription className="flex items-center gap-1.5 mt-1">
                    <Mail className="h-3.5 w-3.5" />
                    {user?.email || "未设置邮箱"}
                  </CardDescription>
                  <Badge variant="secondary" className="mt-2 text-xs">
                    <Shield className="h-3 w-3 mr-1" />
                    {user?.role || "user"}
                  </Badge>
                </div>
              </div>
            </CardHeader>
          </Card>
        </motion.div>

        {/* Basic Info */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <User className="h-4 w-4 text-primary" />
                基本信息
              </CardTitle>
              <CardDescription>更新您的个人信息</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="username">用户名</Label>
                <Input
                  id="username"
                  value={form.username}
                  onChange={(e) => setForm({ ...form, username: e.target.value })}
                  placeholder="请输入用户名"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="email">邮箱</Label>
                <Input
                  id="email"
                  type="email"
                  value={form.email}
                  onChange={(e) => setForm({ ...form, email: e.target.value })}
                  placeholder="请输入邮箱"
                />
              </div>
              <div className="flex justify-end pt-2">
                <Button onClick={handleSave} disabled={saving}>
                  {saving ? "保存中..." : "保存更改"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </motion.div>

        {/* Password */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.3 }}
        >
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <Lock className="h-4 w-4 text-primary" />
                修改密码
              </CardTitle>
              <CardDescription>更新您的登录密码</CardDescription>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="new-password">新密码</Label>
                <Input
                  id="new-password"
                  type="password"
                  value={passwordForm.newPassword}
                  onChange={(e) => setPasswordForm({ ...passwordForm, newPassword: e.target.value })}
                  placeholder="请输入新密码（至少6位）"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="confirm-password">确认密码</Label>
                <Input
                  id="confirm-password"
                  type="password"
                  value={passwordForm.confirmPassword}
                  onChange={(e) => setPasswordForm({ ...passwordForm, confirmPassword: e.target.value })}
                  placeholder="请再次输入新密码"
                />
              </div>
              <div className="flex justify-end pt-2">
                <Button onClick={handleSavePassword} disabled={savingPassword}>
                  {savingPassword ? "保存中..." : "更新密码"}
                </Button>
              </div>
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </MainLayout>
  )
}
