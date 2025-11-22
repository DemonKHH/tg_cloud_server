
import { useState, useEffect } from "react"
import { toast } from "sonner"
import { Button } from "@/components/ui/button"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Switch } from "@/components/ui/switch"
import { taskAPI, accountAPI } from "@/lib/api"

interface CreateTaskDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  accountIds: string[] // Pre-selected account IDs
  initialTaskType?: string
  onSuccess?: () => void
}

export function CreateTaskDialog({
  open,
  onOpenChange,
  accountIds,
  initialTaskType,
  onSuccess,
}: CreateTaskDialogProps) {
  const [loading, setLoading] = useState(false)
  const [form, setForm] = useState({
    task_type: "",
    priority: "5",
    auto_start: true,
    // Task specific configs
    check_timeout: "2m",
    private_targets: "",
    private_message: "",
    private_delay: "",
    broadcast_message: "",
    broadcast_groups: "",
    broadcast_channels: "",
    broadcast_delay: "",
    verify_timeout: "30",
    verify_source: "",
    verify_pattern: "",
    group_chat_group_id: "",
    group_chat_duration: "",
    group_chat_ai_config: "{}",
  })

  // Reset form when dialog opens
  useEffect(() => {
    if (open) {
      setForm(prev => ({
        ...prev,
        task_type: initialTaskType || "",
      }))
    }
  }, [open, initialTaskType])

  const handleTaskTypeChange = (value: string) => {
    setForm(prev => ({
      ...prev,
      task_type: value,
      // Reset specific fields if needed, or keep them
    }))
  }

  const buildTaskConfig = () => {
    const config: any = {}

    switch (form.task_type) {
      case "check":
        if (form.check_timeout && form.check_timeout !== "2m") {
          config.timeout = form.check_timeout
        }
        break

      case "private_message":
        if (!form.private_targets || !form.private_message) {
          toast.error("请填写目标用户和消息内容")
          return null
        }
        const targets = form.private_targets.split(",").map(t => t.trim()).filter(t => t)
        if (targets.length === 0) {
          toast.error("请至少填写一个目标用户")
          return null
        }
        config.targets = targets
        config.message = form.private_message
        if (form.private_delay) {
          const delay = parseInt(form.private_delay)
          if (!isNaN(delay) && delay > 0) {
            config.interval_seconds = delay
          }
        }
        break

      case "broadcast":
        if (!form.broadcast_message) {
          toast.error("请填写消息内容")
          return null
        }
        if (!form.broadcast_groups && !form.broadcast_channels) {
          toast.error("请至少填写一个群组或频道ID")
          return null
        }
        config.message = form.broadcast_message

        const allGroups: any[] = []
        if (form.broadcast_groups) {
          const groups = form.broadcast_groups.split(",")
            .map(g => {
              const num = parseInt(g.trim())
              return !isNaN(num) && num > 0 ? num : null
            })
            .filter(g => g !== null)
          allGroups.push(...groups)
        }
        if (form.broadcast_channels) {
          const channels = form.broadcast_channels.split(",")
            .map(c => {
              const num = parseInt(c.trim())
              return !isNaN(num) && num > 0 ? num : null
            })
            .filter(c => c !== null)
          allGroups.push(...channels)
        }

        if (allGroups.length === 0) {
          toast.error("请至少填写一个有效的群组或频道ID")
          return null
        }

        config.groups = allGroups
        if (form.broadcast_delay) {
          const delay = parseInt(form.broadcast_delay)
          if (!isNaN(delay) && delay > 0) {
            config.interval_seconds = delay
          }
        }
        break

      case "verify_code":
        if (form.verify_timeout) {
          const timeout = parseInt(form.verify_timeout)
          if (!isNaN(timeout) && timeout > 0) {
            config.timeout_seconds = timeout
          }
        }
        if (form.verify_source && form.verify_source.trim()) {
          config.senders = [form.verify_source.trim()]
        }
        if (form.verify_pattern && form.verify_pattern.trim()) {
          config.pattern = form.verify_pattern.trim()
        }
        break

      case "group_chat":
        if (!form.group_chat_group_id) {
          toast.error("请填写群组ID")
          return null
        }
        const groupId = parseInt(form.group_chat_group_id)
        if (isNaN(groupId) || groupId <= 0) {
          toast.error("群组ID必须是大于0的数字")
          return null
        }
        config.group_id = groupId
        if (form.group_chat_duration) {
          const duration = parseInt(form.group_chat_duration)
          if (!isNaN(duration) && duration > 0) {
            config.monitor_duration_seconds = duration * 60
          }
        }
        if (form.group_chat_ai_config && form.group_chat_ai_config.trim() !== "" && form.group_chat_ai_config !== "{}") {
          try {
            const aiConfig = JSON.parse(form.group_chat_ai_config)
            if (typeof aiConfig === 'object' && aiConfig !== null) {
              config.ai_config = aiConfig
            }
          } catch (e) {
            toast.error("AI配置JSON格式错误，请检查语法")
            return null
          }
        }
        break

      default:
        toast.error("请选择有效的任务类型")
        return null
    }

    return config
  }

  const handleSubmit = async () => {
    if (accountIds.length === 0) {
      toast.error("未选择任何账号")
      return
    }
    if (!form.task_type) {
      toast.error("请选择任务类型")
      return
    }

    const config = buildTaskConfig()
    if (!config) return

    setLoading(true)

    try {
      // 创建单个任务，使用多个账号
      const requestData = {
        account_ids: accountIds.map(id => parseInt(id)),
        task_type: form.task_type,
        priority: parseInt(form.priority) || 5,
        auto_start: form.auto_start,
        task_config: config,
      }

      await taskAPI.create(requestData)

      if (accountIds.length === 1) {
        toast.success("任务创建成功")
      } else {
        toast.success(`任务创建成功，将使用 ${accountIds.length} 个账号依次执行`)
      }
      onOpenChange(false)
      onSuccess?.()
    } catch (error) {
      console.error("Batch create error:", error)
      toast.error("创建任务过程中发生错误")
    } finally {
      setLoading(false)
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-[600px] max-h-[80vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>创建任务</DialogTitle>
          <DialogDescription>
            为选中的 {accountIds.length} 个账号创建任务
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-4 py-4">
          <div className="grid grid-cols-2 gap-4">
            <div className="space-y-2">
              <Label>任务类型</Label>
              <Select value={form.task_type} onValueChange={handleTaskTypeChange}>
                <SelectTrigger>
                  <SelectValue placeholder="选择任务类型" />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="check">账号检查</SelectItem>
                  <SelectItem value="private_message">私信发送</SelectItem>
                  <SelectItem value="broadcast">群发消息</SelectItem>
                  <SelectItem value="verify_code">验证码接收</SelectItem>
                  <SelectItem value="group_chat">AI炒群</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="space-y-2">
              <Label>优先级 (1-10)</Label>
              <Input
                type="number"
                min="1"
                max="10"
                value={form.priority}
                onChange={e => setForm({ ...form, priority: e.target.value })}
              />
            </div>
          </div>

          <div className="flex items-center space-x-2">
            <Switch
              id="auto-start"
              checked={form.auto_start}
              onCheckedChange={checked => setForm({ ...form, auto_start: checked })}
            />
            <Label htmlFor="auto-start">创建后自动启动</Label>
          </div>

          {/* Dynamic Config Fields */}
          <div className="border-t pt-4 mt-4">
            {form.task_type === "check" && (
              <div className="space-y-2">
                <Label>超时时间</Label>
                <Input
                  value={form.check_timeout}
                  onChange={e => setForm({ ...form, check_timeout: e.target.value })}
                  placeholder="例如: 2m, 30s"
                />
              </div>
            )}

            {form.task_type === "private_message" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>目标用户 (逗号分隔)</Label>
                  <Textarea
                    value={form.private_targets}
                    onChange={e => setForm({ ...form, private_targets: e.target.value })}
                    placeholder="@user1, @user2, +123456789"
                  />
                </div>
                <div className="space-y-2">
                  <Label>消息内容</Label>
                  <Textarea
                    value={form.private_message}
                    onChange={e => setForm({ ...form, private_message: e.target.value })}
                    placeholder="输入要发送的消息..."
                  />
                </div>
                <div className="space-y-2">
                  <Label>发送间隔 (秒)</Label>
                  <Input
                    type="number"
                    value={form.private_delay}
                    onChange={e => setForm({ ...form, private_delay: e.target.value })}
                    placeholder="默认无间隔"
                  />
                </div>
              </div>
            )}

            {form.task_type === "broadcast" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>群组ID (逗号分隔)</Label>
                  <Input
                    value={form.broadcast_groups}
                    onChange={e => setForm({ ...form, broadcast_groups: e.target.value })}
                    placeholder="123456, 789012"
                  />
                </div>
                <div className="space-y-2">
                  <Label>频道ID (逗号分隔)</Label>
                  <Input
                    value={form.broadcast_channels}
                    onChange={e => setForm({ ...form, broadcast_channels: e.target.value })}
                    placeholder="123456, 789012"
                  />
                </div>
                <div className="space-y-2">
                  <Label>消息内容</Label>
                  <Textarea
                    value={form.broadcast_message}
                    onChange={e => setForm({ ...form, broadcast_message: e.target.value })}
                    placeholder="输入要发送的消息..."
                  />
                </div>
                <div className="space-y-2">
                  <Label>发送间隔 (秒)</Label>
                  <Input
                    type="number"
                    value={form.broadcast_delay}
                    onChange={e => setForm({ ...form, broadcast_delay: e.target.value })}
                    placeholder="默认无间隔"
                  />
                </div>
              </div>
            )}

            {form.task_type === "verify_code" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>超时时间 (秒)</Label>
                  <Input
                    type="number"
                    value={form.verify_timeout}
                    onChange={e => setForm({ ...form, verify_timeout: e.target.value })}
                    placeholder="30"
                  />
                </div>
                <div className="space-y-2">
                  <Label>指定发送者 (可选)</Label>
                  <Input
                    value={form.verify_source}
                    onChange={e => setForm({ ...form, verify_source: e.target.value })}
                    placeholder="例如: Telegram"
                  />
                </div>
                <div className="space-y-2">
                  <Label>匹配正则 (可选)</Label>
                  <Input
                    value={form.verify_pattern}
                    onChange={e => setForm({ ...form, verify_pattern: e.target.value })}
                    placeholder="例如: code is (\d+)"
                  />
                </div>
              </div>
            )}

            {form.task_type === "group_chat" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>群组ID</Label>
                  <Input
                    value={form.group_chat_group_id}
                    onChange={e => setForm({ ...form, group_chat_group_id: e.target.value })}
                    placeholder="输入群组ID"
                  />
                </div>
                <div className="space-y-2">
                  <Label>持续时间 (分钟)</Label>
                  <Input
                    type="number"
                    value={form.group_chat_duration}
                    onChange={e => setForm({ ...form, group_chat_duration: e.target.value })}
                    placeholder="默认一直运行"
                  />
                </div>
                <div className="space-y-2">
                  <Label>AI配置 (JSON)</Label>
                  <Textarea
                    value={form.group_chat_ai_config}
                    onChange={e => setForm({ ...form, group_chat_ai_config: e.target.value })}
                    placeholder="{}"
                    className="font-mono"
                  />
                </div>
              </div>
            )}
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>取消</Button>
          <Button onClick={handleSubmit} disabled={loading}>
            {loading ? "创建中..." : "确定创建"}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
