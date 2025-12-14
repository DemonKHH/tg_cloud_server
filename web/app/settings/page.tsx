"use client"

import { useEffect, useState } from "react"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Label } from "@/components/ui/label"
import { Button } from "@/components/ui/button"
import { Palette, Shield, Loader2, RotateCcw, Save } from "lucide-react"
import { motion } from "framer-motion"
import { useTheme } from "next-themes"
import { toast } from "sonner"
import { settingsAPI, RiskSettings } from "@/lib/api"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Slider } from "@/components/ui/slider"

const DEFAULT_RISK_SETTINGS: RiskSettings = {
  max_consecutive_failures: 5,
  cooling_duration_minutes: 30,
}

export default function SettingsPage() {
  const { theme, setTheme } = useTheme()

  const [riskSettings, setRiskSettings] = useState<RiskSettings>(DEFAULT_RISK_SETTINGS)
  const [originalSettings, setOriginalSettings] = useState<RiskSettings>(DEFAULT_RISK_SETTINGS)
  const [loading, setLoading] = useState(true)
  const [saving, setSaving] = useState(false)

  // 加载风控配置
  useEffect(() => {
    loadRiskSettings()
  }, [])

  const loadRiskSettings = async () => {
    try {
      setLoading(true)
      const response = await settingsAPI.getRiskSettings()
      if (response.code === 0 && response.data) {
        setRiskSettings(response.data)
        setOriginalSettings(response.data)
      } else {
        toast.error(response.msg || "无法加载风控配置")
      }
    } catch (error) {
      console.error("Failed to load risk settings:", error)
      toast.error("无法加载风控配置")
    } finally {
      setLoading(false)
    }
  }

  const handleSaveRiskSettings = async () => {
    try {
      setSaving(true)
      const res = await settingsAPI.updateRiskSettings(riskSettings)
      if (res.code === 0) {
        setOriginalSettings(riskSettings)
        toast.success("风控配置已更新")
      } else {
        toast.error(res.msg || "无法保存风控配置")
      }
    } catch (error: any) {
      console.error("Failed to save risk settings:", error)
      toast.error(error instanceof Error ? error.message : "无法保存风控配置")
    } finally {
      setSaving(false)
    }
  }

  const handleResetRiskSettings = () => {
    setRiskSettings(DEFAULT_RISK_SETTINGS)
  }

  const hasChanges = JSON.stringify(riskSettings) !== JSON.stringify(originalSettings)

  return (
    <MainLayout>
      <div className="space-y-6 max-w-2xl">
        {/* Page Header */}
        <motion.div
          initial={{ opacity: 0, y: -10 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ duration: 0.3 }}
        >
          <h1 className="page-title gradient-text">系统设置</h1>
          <p className="page-subtitle">自定义您的系统偏好</p>
        </motion.div>

        {/* Appearance Settings */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.1 }}
        >
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <Palette className="h-4 w-4 text-primary" />
                外观设置
              </CardTitle>
              <CardDescription>自定义界面外观</CardDescription>
            </CardHeader>
            <CardContent>
              <div className="flex items-center justify-between">
                <div className="space-y-0.5">
                  <Label>主题模式</Label>
                  <p className="text-xs text-muted-foreground">选择您喜欢的界面主题</p>
                </div>
                <Select value={theme} onValueChange={setTheme}>
                  <SelectTrigger className="w-[140px]">
                    <SelectValue placeholder="选择主题" />
                  </SelectTrigger>
                  <SelectContent>
                    <SelectItem value="light">浅色模式</SelectItem>
                    <SelectItem value="dark">深色模式</SelectItem>
                    <SelectItem value="system">跟随系统</SelectItem>
                  </SelectContent>
                </Select>
              </div>
            </CardContent>
          </Card>
        </motion.div>


        {/* Risk Control Settings */}
        <motion.div
          initial={{ opacity: 0, y: 20 }}
          animate={{ opacity: 1, y: 0 }}
          transition={{ delay: 0.2 }}
        >
          <Card className="border-border/50">
            <CardHeader>
              <CardTitle className="text-base font-semibold flex items-center gap-2">
                <Shield className="h-4 w-4 text-primary" />
                风控设置
              </CardTitle>
              <CardDescription>配置账号风控保护参数</CardDescription>
            </CardHeader>
            <CardContent className="space-y-6">
              {loading ? (
                <div className="flex items-center justify-center py-8">
                  <Loader2 className="h-6 w-6 animate-spin text-muted-foreground" />
                </div>
              ) : (
                <>
                  {/* 连续失败次数 */}
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>连续失败次数阈值</Label>
                        <p className="text-xs text-muted-foreground">
                          账号连续失败达到此次数后进入冷却状态
                        </p>
                      </div>
                      <span className="text-sm font-medium w-12 text-right">
                        {riskSettings.max_consecutive_failures} 次
                      </span>
                    </div>
                    <Slider
                      value={[riskSettings.max_consecutive_failures]}
                      onValueChange={([value]) =>
                        setRiskSettings(prev => ({ ...prev, max_consecutive_failures: value }))
                      }
                      min={3}
                      max={10}
                      step={1}
                      className="w-full"
                    />
                    <div className="flex justify-between text-xs text-muted-foreground">
                      <span>3 次</span>
                      <span>10 次</span>
                    </div>
                  </div>

                  {/* 冷却时长 */}
                  <div className="space-y-3">
                    <div className="flex items-center justify-between">
                      <div className="space-y-0.5">
                        <Label>冷却时长</Label>
                        <p className="text-xs text-muted-foreground">
                          账号进入冷却状态后的等待时间
                        </p>
                      </div>
                      <span className="text-sm font-medium w-16 text-right">
                        {riskSettings.cooling_duration_minutes} 分钟
                      </span>
                    </div>
                    <Slider
                      value={[riskSettings.cooling_duration_minutes]}
                      onValueChange={([value]) =>
                        setRiskSettings(prev => ({ ...prev, cooling_duration_minutes: value }))
                      }
                      min={10}
                      max={120}
                      step={5}
                      className="w-full"
                    />
                    <div className="flex justify-between text-xs text-muted-foreground">
                      <span>10 分钟</span>
                      <span>120 分钟</span>
                    </div>
                  </div>

                  {/* 说明 */}
                  <div className="rounded-lg bg-muted/50 p-3 text-xs text-muted-foreground space-y-1">
                    <p>• 冷却结束后账号自动恢复为正常状态</p>
                    <p>• 警告状态 24 小时无错误后自动恢复</p>
                    <p>• Telegram 返回的限流错误会自动触发冷却，冷却时长由 Telegram 决定</p>
                    <p>• 账号被封禁（Dead）或冻结（Frozen）状态无法自动恢复</p>
                  </div>

                  {/* 操作按钮 */}
                  <div className="flex justify-end gap-2 pt-2">
                    <Button
                      variant="outline"
                      size="sm"
                      onClick={handleResetRiskSettings}
                      disabled={saving}
                    >
                      <RotateCcw className="h-4 w-4 mr-1" />
                      恢复默认
                    </Button>
                    <Button
                      size="sm"
                      onClick={handleSaveRiskSettings}
                      disabled={saving || !hasChanges}
                    >
                      {saving ? (
                        <Loader2 className="h-4 w-4 mr-1 animate-spin" />
                      ) : (
                        <Save className="h-4 w-4 mr-1" />
                      )}
                      保存设置
                    </Button>
                  </div>
                </>
              )}
            </CardContent>
          </Card>
        </motion.div>
      </div>
    </MainLayout>
  )
}
