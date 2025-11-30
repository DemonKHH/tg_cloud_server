
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

    group_chat_personality: "friendly",
    group_chat_keywords: "",
    group_chat_rate: "0.3",
    join_group_groups: "",
    join_group_delay: "",
    force_add_group_username: "",
    force_add_group_targets: "",
    force_add_group_limit: "",
    check_spam_bot: false,
    check_2fa: false,
    two_fa_password: "",
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
        if (form.check_spam_bot) {
          config.check_spam_bot = true
        }
        if (form.check_2fa) {
          config.check_2fa = true
          if (form.two_fa_password) {
            config.two_fa_password = form.two_fa_password
          }
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
          toast.error("请至少填写一个群组或频道")
          return null
        }
        config.message = form.broadcast_message

        const allGroups: any[] = []

        const processInput = (input: string) => {
          return input.split(",")
            .map(g => {
              let item = g.trim()
              if (!item) return null

              // 尝试解析为数字ID
              // 使用正则表达式确保只有纯数字才被视为ID
              if (/^-?\d+$/.test(item)) {
                const num = parseInt(item)
                if (!isNaN(num)) return num
              }

              // 处理链接格式 (t.me/username 或 https://t.me/username)
              item = item.replace(/^https?:\/\//, '').replace(/^t\.me\//, '')

              // 移除可能存在的 @ 前缀 (后端会处理，但前端清理一下也好)
              // if (item.startsWith('@')) item = item.substring(1)

              return item
            })
            .filter(g => g !== null)
        }

        if (form.broadcast_groups) {
          allGroups.push(...processInput(form.broadcast_groups))
        }
        if (form.broadcast_channels) {
          allGroups.push(...processInput(form.broadcast_channels))
        }

        if (allGroups.length === 0) {
          toast.error("请至少填写一个有效的群组或频道")
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
      case "join_group":
        if (!form.join_group_groups) {
          toast.error("请填写群组链接或用户名")
          return null
        }

        const joinGroups = form.join_group_groups.split(",")
          .map(g => g.trim())
          .filter(g => g !== "")

        if (joinGroups.length === 0) {
          toast.error("请至少填写一个有效的群组")
          return null
        }

        config.groups = joinGroups

        if (form.join_group_delay) {
          const delay = parseInt(form.join_group_delay)
          if (!isNaN(delay) && delay > 0) {
            config.interval_seconds = delay
          }
        }
        break

      case "force_add_group":
        if (!form.force_add_group_username) {
          toast.error("请填写目标群组用户名")
          return null
        }
        let targetGroup = form.force_add_group_username.trim()
        // Remove @ or t.me/ prefix if present
        targetGroup = targetGroup.replace(/^https?:\/\//, '').replace(/^t\.me\//, '').replace(/^@/, '')

        if (!targetGroup) {
          toast.error("目标群组用户名无效")
          return null
        }

        config.group_name = targetGroup

        if (!form.force_add_group_targets) {
          toast.error("请填写要拉入的用户列表")
          return null
        }
        const forceAddTargets = form.force_add_group_targets.split(",")
          .map(t => t.trim())
          .filter(t => t)

        if (forceAddTargets.length === 0) {
          toast.error("请至少填写一个要拉入的用户")
          return null
        }
        config.targets = forceAddTargets

        if (form.force_add_group_limit) {
          const limit = parseInt(form.force_add_group_limit)
          if (!isNaN(limit) && limit > 0) {
            config.limit_per_account = limit
          }
        }
        break

      case "group_chat":
        if (!form.group_chat_group_id) {
          toast.error("请填写群组信息")
          return null
        }

        let groupInput = form.group_chat_group_id.trim()

        // Try to parse as numeric ID first
        if (/^-?\d+$/.test(groupInput)) {
          const groupId = parseInt(groupInput)
          if (!isNaN(groupId)) {
            config.group_id = groupId
          }
        } else {
          // Treat as username/link
          // Remove prefixes
          groupInput = groupInput.replace(/^https?:\/\//, '').replace(/^t\.me\//, '').replace(/^@/, '')
          if (groupInput) {
            config.group_name = groupInput
          } else {
            toast.error("无效的群组信息")
            return null
          }
        }

        if (form.group_chat_duration) {
          const duration = parseInt(form.group_chat_duration)
          if (!isNaN(duration) && duration > 0) {
            config.monitor_duration_seconds = duration * 60
          }
        }

        // Construct AI config from specific fields
        const aiConfig: any = {
          personality: form.group_chat_personality || "friendly",
          response_rate: parseFloat(form.group_chat_rate) || 0.3
        }

        if (form.group_chat_keywords) {
          aiConfig.keywords = form.group_chat_keywords.split(",").map(k => k.trim()).filter(k => k)
        }

        config.ai_config = aiConfig
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
                  <SelectItem value="join_group">批量加群</SelectItem>
                  <SelectItem value="force_add_group">强拉</SelectItem>
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
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>超时时间</Label>
                  <Input
                    value={form.check_timeout}
                    onChange={e => setForm({ ...form, check_timeout: e.target.value })}
                    placeholder="例如: 2m, 30s"
                  />
                </div>
                <div className="flex items-center space-x-2">
                  <Switch
                    id="check-spam-bot"
                    checked={form.check_spam_bot}
                    onCheckedChange={checked => setForm({ ...form, check_spam_bot: checked })}
                  />
                  <Label htmlFor="check-spam-bot">双向/冻结检查 (使用 @SpamBot)</Label>
                </div>

                <div className="space-y-4 pt-2 border-t border-dashed">
                  <div className="flex items-center space-x-2">
                    <Switch
                      id="check-2fa"
                      checked={form.check_2fa}
                      onCheckedChange={checked => setForm({ ...form, check_2fa: checked })}
                    />
                    <Label htmlFor="check-2fa">2FA 检查</Label>
                  </div>

                  {form.check_2fa && (
                    <div className="space-y-2 pl-6">
                      <Label>2FA 密码 (可选)</Label>
                      <Input
                        type="password"
                        value={form.two_fa_password}
                        onChange={e => setForm({ ...form, two_fa_password: e.target.value })}
                        placeholder="如果账号开启了2FA，请输入密码以验证"
                      />
                      <p className="text-xs text-muted-foreground">
                        如果不提供密码，仅检查是否开启了2FA
                      </p>
                    </div>
                  )}
                </div>
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
                  <Label>群组 (ID/用户名/链接，逗号分隔)</Label>
                  <Input
                    value={form.broadcast_groups}
                    onChange={e => setForm({ ...form, broadcast_groups: e.target.value })}
                    placeholder="123456, @groupname, t.me/group"
                  />
                </div>
                <div className="space-y-2">
                  <Label>频道 (ID/用户名/链接，逗号分隔)</Label>
                  <Input
                    value={form.broadcast_channels}
                    onChange={e => setForm({ ...form, broadcast_channels: e.target.value })}
                    placeholder="123456, @channel, t.me/channel"
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



            {form.task_type === "group_chat" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>群组 (ID/用户名/链接)</Label>
                  <Input
                    value={form.group_chat_group_id}
                    onChange={e => setForm({ ...form, group_chat_group_id: e.target.value })}
                    placeholder="例如: @groupname, t.me/group, 或数字ID"
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
                  <Label>AI性格</Label>
                  <Select
                    value={form.group_chat_personality}
                    onValueChange={v => setForm({ ...form, group_chat_personality: v })}
                  >
                    <SelectTrigger>
                      <SelectValue placeholder="选择性格" />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="friendly">友好 (Friendly)</SelectItem>
                      <SelectItem value="professional">专业 (Professional)</SelectItem>
                      <SelectItem value="humorous">幽默 (Humorous)</SelectItem>
                    </SelectContent>
                  </Select>
                </div>
                <div className="space-y-2">
                  <Label>触发关键词 (逗号分隔)</Label>
                  <Input
                    value={form.group_chat_keywords}
                    onChange={e => setForm({ ...form, group_chat_keywords: e.target.value })}
                    placeholder="hello, hi, question, 价格"
                  />
                </div>
                <div className="space-y-2">
                  <Label>回复概率 (0.1 - 1.0)</Label>
                  <Input
                    type="number"
                    min="0.1"
                    max="1.0"
                    step="0.1"
                    value={form.group_chat_rate}
                    onChange={e => setForm({ ...form, group_chat_rate: e.target.value })}
                  />
                </div>
              </div>
            )}

            {form.task_type === "join_group" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>群组 (链接/用户名，逗号分隔)</Label>
                  <Textarea
                    value={form.join_group_groups}
                    onChange={e => setForm({ ...form, join_group_groups: e.target.value })}
                    placeholder="https://t.me/group, @groupname, https://t.me/+invitehash"
                    className="min-h-[100px]"
                  />
                  <p className="text-xs text-muted-foreground">
                    支持公开群组链接、用户名以及私有群组邀请链接
                  </p>
                </div>
                <div className="space-y-2">
                  <Label>加入间隔 (秒)</Label>
                  <Input
                    type="number"
                    value={form.join_group_delay}
                    onChange={e => setForm({ ...form, join_group_delay: e.target.value })}
                    placeholder="默认5秒"
                  />
                </div>
              </div>
            )}

            {form.task_type === "force_add_group" && (
              <div className="space-y-4">
                <div className="space-y-2">
                  <Label>目标群组用户名</Label>
                  <Input
                    value={form.force_add_group_username}
                    onChange={e => setForm({ ...form, force_add_group_username: e.target.value })}
                    placeholder="例如: @groupname 或 t.me/groupname"
                  />
                </div>
                <div className="space-y-2">
                  <Label>目标用户 (用户名/手机号，逗号分隔)</Label>
                  <Textarea
                    value={form.force_add_group_targets}
                    onChange={e => setForm({ ...form, force_add_group_targets: e.target.value })}
                    placeholder="@user1, @user2, +123456789"
                    className="min-h-[100px]"
                  />
                </div>
                <div className="space-y-2">
                  <Label>单号拉人限制</Label>
                  <Input
                    type="number"
                    value={form.force_add_group_limit}
                    onChange={e => setForm({ ...form, force_add_group_limit: e.target.value })}
                    placeholder="默认无限制"
                  />
                  <p className="text-xs text-muted-foreground">
                    限制每个账号拉入群组的人数，留空则不限制
                  </p>
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
