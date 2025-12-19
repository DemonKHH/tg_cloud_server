"use client"

import { useState, useEffect } from "react"
import { toast } from "sonner"
import { MainLayout } from "@/components/layout/main-layout"
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import { Badge } from "@/components/ui/badge"
import { Checkbox } from "@/components/ui/checkbox"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from "@/components/ui/dialog"
import {
  Bot,
  Sparkles,
  MessageSquare,
  Users,
  Play,
  Trash2,
  Settings2,
  Target,
  Zap,
  UserPlus,
  CheckCircle2,
} from "lucide-react"
import { accountAPI, taskAPI } from "@/lib/api"

interface Account {
  id: number
  phone: string
  status: string
  first_name?: string
  last_name?: string
  username?: string
}

interface AgentConfig {
  account_id: number
  account_phone?: string
  name: string
  style: string
  goal: string
  active_rate: number
}

interface ScenarioConfig {
  name: string
  description: string
  topic: string
  duration: number
  agents: AgentConfig[]
}

export default function AIPage() {
  const [accounts, setAccounts] = useState<Account[]>([])
  const [loading, setLoading] = useState(false)

  // 场景配置
  const [scenario, setScenario] = useState<ScenarioConfig>({
    name: "",
    description: "",
    topic: "",
    duration: 600,
    agents: [],
  })

  // 当前编辑的智能体
  const [editingAgent, setEditingAgent] = useState<AgentConfig | null>(null)
  const [agentDialogOpen, setAgentDialogOpen] = useState(false)

  // 批量选择的账号（用于场景配置）
  const [scenarioSelectedAccounts, setScenarioSelectedAccounts] = useState<number[]>([])

  // 快速创建选择的账号
  const [quickSelectedAccounts, setQuickSelectedAccounts] = useState<number[]>([])

  // 加载账号列表
  useEffect(() => {
    loadAccounts()
  }, [])

  const loadAccounts = async () => {
    try {
      const response = await accountAPI.list({ limit: 100 })
      if (response.code === 0 && response.data) {
        const responseData = response.data as any
        const allAccounts = responseData.items || responseData.data || []
        const availableAccounts = allAccounts.filter((a: any) => a.status !== "dead")
        setAccounts(availableAccounts)
      }
    } catch (error) {
      console.error("Failed to load accounts:", error)
    }
  }

  // 直接添加账号为智能体（点击即添加，打开编辑对话框）
  const handleAddAccountAsAgent = (account: Account) => {
    if (scenario.agents.find(a => a.account_id === account.id)) {
      toast.info("该账号已添加")
      return
    }

    const newAgent: AgentConfig = {
      account_id: account.id,
      account_phone: account.phone,
      name: "",
      style: "",
      goal: "",
      active_rate: 0.5,
    }

    // 打开编辑对话框让用户填写
    setEditingAgent(newAgent)
    setAgentDialogOpen(true)
  }

  // 保存新增或编辑的智能体
  const handleSaveAgentFromDialog = () => {
    if (!editingAgent) return

    // 验证必填字段
    if (!editingAgent.name.trim()) {
      toast.error("请填写角色名称")
      return
    }
    if (!editingAgent.goal.trim()) {
      toast.error("请填写行动目标")
      return
    }

    // 检查是新增还是编辑
    const existingIndex = scenario.agents.findIndex(a => a.account_id === editingAgent.account_id)
    
    if (existingIndex >= 0) {
      // 编辑现有智能体
      setScenario(prev => ({
        ...prev,
        agents: prev.agents.map(a =>
          a.account_id === editingAgent.account_id ? editingAgent : a
        ),
      }))
      toast.success("智能体配置已更新")
    } else {
      // 新增智能体
      setScenario(prev => ({
        ...prev,
        agents: [...prev.agents, editingAgent],
      }))
      toast.success(`已添加 ${editingAgent.account_phone}`)
    }

    setAgentDialogOpen(false)
    setEditingAgent(null)
  }

  // 批量添加选中的账号（打开批量配置对话框）
  const handleBatchAddAgents = () => {
    if (scenarioSelectedAccounts.length === 0) {
      toast.warning("请先选择账号")
      return
    }

    // 对于批量添加，逐个打开编辑对话框
    const firstAccountId = scenarioSelectedAccounts[0]
    const account = accounts.find(a => a.id === firstAccountId)
    
    if (!account) return

    const newAgent: AgentConfig = {
      account_id: firstAccountId,
      account_phone: account.phone,
      name: "",
      style: "",
      goal: "",
      active_rate: 0.5,
    }

    setEditingAgent(newAgent)
    setAgentDialogOpen(true)
  }

  // 编辑智能体
  const handleEditAgent = (agent: AgentConfig) => {
    setEditingAgent({ ...agent })
    setAgentDialogOpen(true)
  }

  // 删除智能体
  const handleRemoveAgent = (accountId: number) => {
    setScenario(prev => ({
      ...prev,
      agents: prev.agents.filter(a => a.account_id !== accountId),
    }))
  }

  // 创建场景任务
  const handleCreateScenario = async () => {
    if (!scenario.topic) {
      toast.error("请填写目标群组")
      return
    }
    if (scenario.agents.length === 0) {
      toast.error("请至少添加一个智能体")
      return
    }

    setLoading(true)
    try {
      const taskConfig = {
        name: scenario.name || `AI炒群-${new Date().toLocaleString()}`,
        description: scenario.description,
        topic: scenario.topic,
        duration: scenario.duration,
        agents: scenario.agents.map(agent => ({
          account_id: agent.account_id,
          persona: {
            name: agent.name,
            style: agent.style ? [agent.style] : [],
          },
          goal: agent.goal,
          active_rate: agent.active_rate,
        })),
      }

      const response = await taskAPI.create({
        account_ids: scenario.agents.map(a => a.account_id),
        task_type: "scenario",
        priority: 5,
        auto_start: true,
        task_config: taskConfig,
      })

      if (response.code === 0) {
        toast.success("场景任务创建成功")
        setScenario({
          name: "",
          description: "",
          topic: "",
          duration: 600,
          agents: [],
        })
      } else {
        toast.error(response.msg || "创建失败")
      }
    } catch (error) {
      console.error("Create scenario error:", error)
      toast.error("创建场景任务失败")
    } finally {
      setLoading(false)
    }
  }

  // 快速创建
  const [quickForm, setQuickForm] = useState({
    group: "",
    duration: "30",
    personality: "friendly",
    keywords: "",
    rate: "0.3",
  })

  const handleQuickCreate = async () => {
    if (quickSelectedAccounts.length === 0) {
      toast.error("请选择至少一个账号")
      return
    }
    if (!quickForm.group) {
      toast.error("请填写目标群组")
      return
    }

    setLoading(true)
    try {
      let groupInput = quickForm.group.trim()
      const config: any = {}

      if (/^-?\d+$/.test(groupInput)) {
        config.group_id = parseInt(groupInput)
      } else {
        groupInput = groupInput.replace(/^https?:\/\//, '').replace(/^t\.me\//, '').replace(/^@/, '')
        config.group_name = groupInput
      }

      if (quickForm.duration) {
        config.monitor_duration_seconds = parseInt(quickForm.duration) * 60
      }

      config.ai_config = {
        personality: quickForm.personality,
        response_rate: parseFloat(quickForm.rate) || 0.3,
      }

      if (quickForm.keywords) {
        config.ai_config.keywords = quickForm.keywords.split(",").map(k => k.trim()).filter(k => k)
      }

      const response = await taskAPI.create({
        account_ids: quickSelectedAccounts,
        task_type: "group_chat",
        priority: 5,
        auto_start: true,
        task_config: config,
      })

      if (response.code === 0) {
        toast.success("AI炒群任务创建成功")
        setQuickSelectedAccounts([])
        setQuickForm({
          group: "",
          duration: "30",
          personality: "friendly",
          keywords: "",
          rate: "0.3",
        })
      } else {
        toast.error(response.msg || "创建失败")
      }
    } catch (error) {
      console.error("Quick create error:", error)
      toast.error("创建任务失败")
    } finally {
      setLoading(false)
    }
  }

  // 获取未添加的账号
  const availableAccounts = accounts.filter(a => !scenario.agents.find(ag => ag.account_id === a.id))

  return (
    <MainLayout>
      <div className="space-y-6">
        {/* 页面标题 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-3xl font-bold tracking-tight flex items-center gap-2">
              <Bot className="h-8 w-8 text-primary" />
              AI炒群
            </h1>
            <p className="text-muted-foreground mt-1">
              使用AI智能体自动参与群聊互动
            </p>
          </div>
          <div className="flex gap-2">
            <Button variant="outline" asChild className="gap-2">
              <a href="/ai/test">
                <Zap className="h-4 w-4" />
                测试AI
              </a>
            </Button>
          </div>
        </div>

        <Tabs defaultValue="scenario" className="space-y-6">
          <TabsList>
            <TabsTrigger value="scenario" className="gap-2">
              <Users className="h-4 w-4" />
              多智能体场景
            </TabsTrigger>
            <TabsTrigger value="quick" className="gap-2">
              <Zap className="h-4 w-4" />
              快速创建
            </TabsTrigger>
          </TabsList>

          {/* 多智能体场景配置 */}
          <TabsContent value="scenario" className="space-y-6">
            {/* 场景基本信息 */}
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Target className="h-5 w-5" />
                  场景配置
                </CardTitle>
                <CardDescription>设置目标群组和运行时长</CardDescription>
              </CardHeader>
              <CardContent>
                <div className="grid grid-cols-3 gap-4">
                  <div className="space-y-2">
                    <Label>目标群组 *</Label>
                    <Input
                      value={scenario.topic}
                      onChange={e => setScenario({ ...scenario, topic: e.target.value })}
                      placeholder="@groupname 或 t.me/group"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>场景名称</Label>
                    <Input
                      value={scenario.name}
                      onChange={e => setScenario({ ...scenario, name: e.target.value })}
                      placeholder="可选，用于标识任务"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>持续时间</Label>
                    <Select
                      value={scenario.duration.toString()}
                      onValueChange={v => setScenario({ ...scenario, duration: parseInt(v) })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="300">5 分钟</SelectItem>
                        <SelectItem value="600">10 分钟</SelectItem>
                        <SelectItem value="1800">30 分钟</SelectItem>
                        <SelectItem value="3600">1 小时</SelectItem>
                        <SelectItem value="7200">2 小时</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>
                </div>
              </CardContent>
            </Card>

            <div className="grid gap-6 lg:grid-cols-2">
              {/* 左侧：账号列表 */}
              <Card>
                <CardHeader>
                  <div className="flex items-center justify-between">
                    <div>
                      <CardTitle className="flex items-center gap-2">
                        <UserPlus className="h-5 w-5" />
                        选择账号
                      </CardTitle>
                      <CardDescription>
                        点击账号直接添加，或勾选后批量添加
                      </CardDescription>
                    </div>
                    {scenarioSelectedAccounts.length > 0 && (
                      <Button size="sm" onClick={handleBatchAddAgents}>
                        批量添加 ({scenarioSelectedAccounts.length})
                      </Button>
                    )}
                  </div>
                </CardHeader>
                <CardContent>
                  <div className="max-h-[400px] overflow-y-auto space-y-2">
                    {availableAccounts.length === 0 ? (
                      <p className="text-sm text-muted-foreground text-center py-8">
                        {accounts.length === 0 ? "暂无可用账号" : "所有账号已添加"}
                      </p>
                    ) : (
                      availableAccounts.map(account => (
                        <div
                          key={account.id}
                          className="flex items-center gap-3 p-3 rounded-lg border hover:bg-muted/50 transition-colors"
                        >
                          <Checkbox
                            checked={scenarioSelectedAccounts.includes(account.id)}
                            onCheckedChange={(checked) => {
                              setScenarioSelectedAccounts(prev =>
                                checked
                                  ? [...prev, account.id]
                                  : prev.filter(id => id !== account.id)
                              )
                            }}
                          />
                          <div 
                            className="flex-1 flex items-center justify-between cursor-pointer"
                            onClick={() => handleAddAccountAsAgent(account)}
                          >
                            <div className="flex items-center gap-3">
                              <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                                <Bot className="h-4 w-4 text-primary" />
                              </div>
                              <div>
                                <p className="font-medium text-sm">{account.phone}</p>
                                <p className="text-xs text-muted-foreground">
                                  {account.first_name || account.username || "未设置昵称"}
                                </p>
                              </div>
                            </div>
                            <Badge variant="outline" className="text-xs">
                              点击添加
                            </Badge>
                          </div>
                        </div>
                      ))
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* 右侧：已添加的智能体 */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Users className="h-5 w-5" />
                    智能体列表
                    {scenario.agents.length > 0 && (
                      <Badge variant="secondary">{scenario.agents.length}</Badge>
                    )}
                  </CardTitle>
                  <CardDescription>
                    为每个账号配置独立的人设和目标
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  {scenario.agents.length === 0 ? (
                    <div className="text-center py-12 text-muted-foreground">
                      <Bot className="h-12 w-12 mx-auto mb-3 opacity-30" />
                      <p>暂未添加智能体</p>
                      <p className="text-sm mt-1">从左侧选择账号添加</p>
                    </div>
                  ) : (
                    <div className="space-y-3 max-h-[400px] overflow-y-auto">
                      {scenario.agents.map(agent => (
                        <div
                          key={agent.account_id}
                          className="p-4 rounded-lg border bg-card"
                        >
                          <div className="flex items-start justify-between mb-3">
                            <div className="flex items-center gap-3">
                              <div className="h-10 w-10 rounded-full bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center text-primary-foreground font-medium">
                                {agent.name ? agent.name.charAt(0) : "?"}
                              </div>
                              <div>
                                <p className="font-medium">{agent.name || "未设置名称"}</p>
                                <p className="text-xs text-muted-foreground">
                                  {agent.account_phone}
                                  {agent.style && ` · ${agent.style}`}
                                </p>
                              </div>
                            </div>
                            <div className="flex gap-1">
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={() => handleEditAgent(agent)}
                              >
                                <Settings2 className="h-4 w-4" />
                              </Button>
                              <Button
                                variant="ghost"
                                size="icon"
                                className="h-8 w-8"
                                onClick={() => handleRemoveAgent(agent.account_id)}
                              >
                                <Trash2 className="h-4 w-4 text-destructive" />
                              </Button>
                            </div>
                          </div>

                          <div className="text-xs text-muted-foreground">
                            <span className="inline-flex items-center gap-1">
                              <CheckCircle2 className="h-3 w-3" />
                              目标: {agent.goal || "未设置"}
                            </span>
                            <span className="ml-3">活跃度: {(agent.active_rate * 100).toFixed(0)}%</span>
                          </div>
                        </div>
                      ))}
                    </div>
                  )}

                  {scenario.agents.length > 0 && (
                    <div className="mt-4 pt-4 border-t">
                      <Button
                        className="w-full gap-2"
                        size="lg"
                        onClick={handleCreateScenario}
                        disabled={loading || !scenario.topic}
                      >
                        <Sparkles className="h-5 w-5" />
                        {loading ? "创建中..." : "启动场景任务"}
                      </Button>
                    </div>
                  )}
                </CardContent>
              </Card>
            </div>
          </TabsContent>

          {/* 快速创建 */}
          <TabsContent value="quick" className="space-y-6">
            <div className="grid gap-6 md:grid-cols-2">
              {/* 账号选择 */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Users className="h-5 w-5" />
                    选择账号
                  </CardTitle>
                  <CardDescription>
                    选择参与炒群的账号 (已选 {quickSelectedAccounts.length} 个)
                  </CardDescription>
                </CardHeader>
                <CardContent>
                  <div className="max-h-[300px] overflow-y-auto space-y-2">
                    {accounts.length === 0 ? (
                      <p className="text-sm text-muted-foreground text-center py-4">
                        暂无可用账号
                      </p>
                    ) : (
                      accounts.map(account => (
                        <div
                          key={account.id}
                          className={`flex items-center justify-between p-3 rounded-lg border cursor-pointer transition-colors ${
                            quickSelectedAccounts.includes(account.id)
                              ? "border-primary bg-primary/5"
                              : "hover:bg-muted/50"
                          }`}
                          onClick={() => {
                            setQuickSelectedAccounts(prev =>
                              prev.includes(account.id)
                                ? prev.filter(id => id !== account.id)
                                : [...prev, account.id]
                            )
                          }}
                        >
                          <div className="flex items-center gap-3">
                            <div className="h-8 w-8 rounded-full bg-primary/10 flex items-center justify-center">
                              <Bot className="h-4 w-4 text-primary" />
                            </div>
                            <div>
                              <p className="font-medium text-sm">{account.phone}</p>
                              <p className="text-xs text-muted-foreground">
                                {account.first_name || account.username || "未设置昵称"}
                              </p>
                            </div>
                          </div>
                          <Badge 
                            variant={account.status === "normal" ? "default" : "outline"}
                            className={
                              account.status === "normal" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" : ""
                            }
                          >
                            {account.status === "normal" ? "正常" : account.status}
                          </Badge>
                        </div>
                      ))
                    )}
                  </div>
                </CardContent>
              </Card>

              {/* 快速配置 */}
              <Card>
                <CardHeader>
                  <CardTitle className="flex items-center gap-2">
                    <Settings2 className="h-5 w-5" />
                    任务配置
                  </CardTitle>
                  <CardDescription>配置AI炒群参数（所有账号共用）</CardDescription>
                </CardHeader>
                <CardContent className="space-y-4">
                  <div className="space-y-2">
                    <Label>目标群组</Label>
                    <Input
                      value={quickForm.group}
                      onChange={e => setQuickForm({ ...quickForm, group: e.target.value })}
                      placeholder="@groupname 或 t.me/group 或 群组ID"
                    />
                  </div>

                  <div className="grid grid-cols-2 gap-4">
                    <div className="space-y-2">
                      <Label>持续时间 (分钟)</Label>
                      <Input
                        type="number"
                        value={quickForm.duration}
                        onChange={e => setQuickForm({ ...quickForm, duration: e.target.value })}
                        placeholder="30"
                      />
                    </div>
                    <div className="space-y-2">
                      <Label>回复概率</Label>
                      <Input
                        type="number"
                        min="0.1"
                        max="1"
                        step="0.1"
                        value={quickForm.rate}
                        onChange={e => setQuickForm({ ...quickForm, rate: e.target.value })}
                      />
                    </div>
                  </div>

                  <div className="space-y-2">
                    <Label>AI性格</Label>
                    <Select
                      value={quickForm.personality}
                      onValueChange={v => setQuickForm({ ...quickForm, personality: v })}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="friendly">友好热情</SelectItem>
                        <SelectItem value="professional">专业严谨</SelectItem>
                        <SelectItem value="humorous">幽默风趣</SelectItem>
                        <SelectItem value="casual">随意轻松</SelectItem>
                      </SelectContent>
                    </Select>
                  </div>

                  <div className="space-y-2">
                    <Label>触发关键词 (可选)</Label>
                    <Input
                      value={quickForm.keywords}
                      onChange={e => setQuickForm({ ...quickForm, keywords: e.target.value })}
                      placeholder="价格, 咨询, 问题 (逗号分隔)"
                    />
                  </div>

                  <Button
                    className="w-full gap-2"
                    onClick={handleQuickCreate}
                    disabled={loading || quickSelectedAccounts.length === 0}
                  >
                    <Play className="h-4 w-4" />
                    {loading ? "创建中..." : "开始炒群"}
                  </Button>
                </CardContent>
              </Card>
            </div>
          </TabsContent>
        </Tabs>

        {/* 功能说明 */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2">
              <MessageSquare className="h-5 w-5" />
              功能说明
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="grid gap-4 md:grid-cols-2">
              <div className="p-4 rounded-lg bg-muted/30 border">
                <div className="flex items-center gap-2 mb-2">
                  <Users className="h-5 w-5 text-blue-500" />
                  <span className="font-medium">多智能体场景</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  为每个账号配置独立人设，模拟多人真实互动。支持人设模板快速应用。
                </p>
              </div>
              <div className="p-4 rounded-lg bg-muted/30 border">
                <div className="flex items-center gap-2 mb-2">
                  <Zap className="h-5 w-5 text-yellow-500" />
                  <span className="font-medium">快速创建</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  简单配置，所有账号共用同一AI性格，适合快速启动。
                </p>
              </div>
            </div>
          </CardContent>
        </Card>
      </div>

      {/* 智能体编辑对话框 */}
      <Dialog open={agentDialogOpen} onOpenChange={setAgentDialogOpen}>
        <DialogContent className="sm:max-w-[500px]">
          <DialogHeader>
            <DialogTitle>
              {editingAgent && scenario.agents.find(a => a.account_id === editingAgent.account_id)
                ? "编辑智能体配置"
                : "配置智能体"}
            </DialogTitle>
            <DialogDescription>
              {editingAgent?.account_phone && `账号: ${editingAgent.account_phone}`}
            </DialogDescription>
          </DialogHeader>

          {editingAgent && (
            <div className="space-y-4 py-4">
              <div className="space-y-2">
                <Label>角色名称 *</Label>
                <Input
                  value={editingAgent.name}
                  placeholder="请输入角色名称"
                  onChange={e =>
                    setEditingAgent({
                      ...editingAgent,
                      name: e.target.value,
                    })
                  }
                />
              </div>

              <div className="space-y-2">
                <Label>说话风格</Label>
                <Input
                  value={editingAgent.style}
                  onChange={e =>
                    setEditingAgent({
                      ...editingAgent,
                      style: e.target.value,
                    })
                  }
                  placeholder="例如: 友好热情、专业严谨、幽默风趣"
                />
              </div>

              <div className="space-y-2">
                <Label>行动目标 *</Label>
                <Textarea
                  value={editingAgent.goal}
                  onChange={e => setEditingAgent({ ...editingAgent, goal: e.target.value })}
                  placeholder="描述这个智能体在群聊中的目标..."
                />
              </div>

              <div className="space-y-2">
                <Label>活跃度 ({(editingAgent.active_rate * 100).toFixed(0)}%)</Label>
                <input
                  type="range"
                  min="0.1"
                  max="1"
                  step="0.1"
                  value={editingAgent.active_rate}
                  onChange={e =>
                    setEditingAgent({ ...editingAgent, active_rate: parseFloat(e.target.value) })
                  }
                  className="w-full h-2 bg-gray-200 rounded-lg appearance-none cursor-pointer dark:bg-gray-700"
                />
              </div>
            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setAgentDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSaveAgentFromDialog}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </MainLayout>
  )
}
