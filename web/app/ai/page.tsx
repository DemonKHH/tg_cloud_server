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
  Plus,
  Trash2,
  Settings2,
  Target,
  Zap,
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
  persona: {
    name: string
    age: number
    occupation: string
    style: string[]
    beliefs: string[]
  }
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
  const [createDialogOpen, setCreateDialogOpen] = useState(false)
  const [selectedAccounts, setSelectedAccounts] = useState<number[]>([])

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

  // 加载账号列表
  useEffect(() => {
    loadAccounts()
  }, [])

  const loadAccounts = async () => {
    try {
      // 获取所有账号，不过滤状态，让用户自己选择
      const response = await accountAPI.list({ limit: 100 })
      if (response.code === 0 && response.data) {
        // 后端返回的是 { items: [...], pagination: {...} }
        const responseData = response.data as any
        const allAccounts = responseData.items || responseData.data || []
        // 过滤掉死亡账号，只显示可用的账号
        const availableAccounts = allAccounts.filter(
          (a: any) => a.status !== "dead"
        )
        setAccounts(availableAccounts)
      }
    } catch (error) {
      console.error("Failed to load accounts:", error)
    }
  }

  // 添加智能体
  const handleAddAgent = () => {
    if (selectedAccounts.length === 0) {
      toast.warning("请先选择账号")
      return
    }

    // 为每个选中的账号创建智能体配置
    const newAgents: AgentConfig[] = selectedAccounts
      .filter(id => !scenario.agents.find(a => a.account_id === id))
      .map(accountId => {
        const account = accounts.find(a => a.id === accountId)
        return {
          account_id: accountId,
          account_phone: account?.phone,
          persona: {
            name: account?.first_name || `用户${accountId}`,
            age: 25,
            occupation: "自由职业",
            style: ["友好", "热情"],
            beliefs: [],
          },
          goal: "积极参与群聊讨论",
          active_rate: 0.5,
        }
      })

    if (newAgents.length === 0) {
      toast.info("选中的账号已全部添加")
      return
    }

    setScenario(prev => ({
      ...prev,
      agents: [...prev.agents, ...newAgents],
    }))
    setSelectedAccounts([])
    toast.success(`已添加 ${newAgents.length} 个智能体`)
  }

  // 编辑智能体
  const handleEditAgent = (agent: AgentConfig) => {
    setEditingAgent({ ...agent })
    setAgentDialogOpen(true)
  }

  // 保存智能体编辑
  const handleSaveAgent = () => {
    if (!editingAgent) return

    setScenario(prev => ({
      ...prev,
      agents: prev.agents.map(a =>
        a.account_id === editingAgent.account_id ? editingAgent : a
      ),
    }))
    setAgentDialogOpen(false)
    setEditingAgent(null)
    toast.success("智能体配置已更新")
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
          persona: agent.persona,
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
        setCreateDialogOpen(false)
        // 重置表单
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


  // 快速创建简单AI炒群任务
  const [quickForm, setQuickForm] = useState({
    group: "",
    duration: "30",
    personality: "friendly",
    keywords: "",
    rate: "0.3",
  })

  const handleQuickCreate = async () => {
    if (selectedAccounts.length === 0) {
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

      // 解析群组ID或用户名
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
        account_ids: selectedAccounts,
        task_type: "group_chat",
        priority: 5,
        auto_start: true,
        task_config: config,
      })

      if (response.code === 0) {
        toast.success("AI炒群任务创建成功")
        setSelectedAccounts([])
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
            <Button onClick={() => setCreateDialogOpen(true)} className="gap-2">
              <Sparkles className="h-4 w-4" />
              创建场景任务
            </Button>
          </div>
        </div>

        <Tabs defaultValue="quick" className="space-y-6">
          <TabsList>
            <TabsTrigger value="quick" className="gap-2">
              <Zap className="h-4 w-4" />
              快速创建
            </TabsTrigger>
            <TabsTrigger value="scenario" className="gap-2">
              <Users className="h-4 w-4" />
              场景配置
            </TabsTrigger>
          </TabsList>

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
                    选择参与炒群的账号 (已选 {selectedAccounts.length} 个)
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
                            selectedAccounts.includes(account.id)
                              ? "border-primary bg-primary/5"
                              : "hover:bg-muted/50"
                          }`}
                          onClick={() => {
                            setSelectedAccounts(prev =>
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
                            variant={account.status === "normal" ? "default" : account.status === "warning" ? "secondary" : "outline"}
                            className={
                              account.status === "normal" ? "bg-green-100 text-green-700 dark:bg-green-900/30 dark:text-green-400" :
                              account.status === "warning" ? "bg-yellow-100 text-yellow-700 dark:bg-yellow-900/30 dark:text-yellow-400" :
                              account.status === "restricted" ? "bg-orange-100 text-orange-700 dark:bg-orange-900/30 dark:text-orange-400" :
                              account.status === "cooling" ? "bg-blue-100 text-blue-700 dark:bg-blue-900/30 dark:text-blue-400" :
                              ""
                            }
                          >
                            {account.status === "normal" ? "正常" : 
                             account.status === "warning" ? "警告" :
                             account.status === "restricted" ? "受限" :
                             account.status === "cooling" ? "冷却" :
                             account.status === "new" ? "新建" :
                             account.status}
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
                  <CardDescription>配置AI炒群参数</CardDescription>
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
                    disabled={loading || selectedAccounts.length === 0}
                  >
                    <Play className="h-4 w-4" />
                    {loading ? "创建中..." : "开始炒群"}
                  </Button>
                </CardContent>
              </Card>
            </div>
          </TabsContent>


          {/* 场景配置 */}
          <TabsContent value="scenario" className="space-y-6">
            <Card>
              <CardHeader>
                <CardTitle className="flex items-center gap-2">
                  <Target className="h-5 w-5" />
                  场景基本信息
                </CardTitle>
              </CardHeader>
              <CardContent className="space-y-4">
                <div className="grid grid-cols-2 gap-4">
                  <div className="space-y-2">
                    <Label>场景名称</Label>
                    <Input
                      value={scenario.name}
                      onChange={e => setScenario({ ...scenario, name: e.target.value })}
                      placeholder="例如: 产品推广讨论"
                    />
                  </div>
                  <div className="space-y-2">
                    <Label>目标群组</Label>
                    <Input
                      value={scenario.topic}
                      onChange={e => setScenario({ ...scenario, topic: e.target.value })}
                      placeholder="@groupname 或 t.me/group"
                    />
                  </div>
                </div>
                <div className="space-y-2">
                  <Label>场景描述</Label>
                  <Textarea
                    value={scenario.description}
                    onChange={e => setScenario({ ...scenario, description: e.target.value })}
                    placeholder="描述这个场景的目的和预期效果..."
                  />
                </div>
                <div className="space-y-2">
                  <Label>持续时间 (秒)</Label>
                  <Input
                    type="number"
                    value={scenario.duration}
                    onChange={e => setScenario({ ...scenario, duration: parseInt(e.target.value) || 600 })}
                  />
                  <p className="text-xs text-muted-foreground">
                    当前设置: {Math.floor(scenario.duration / 60)} 分钟
                  </p>
                </div>
              </CardContent>
            </Card>

            {/* 智能体列表 */}
            <Card>
              <CardHeader>
                <div className="flex items-center justify-between">
                  <div>
                    <CardTitle className="flex items-center gap-2">
                      <Users className="h-5 w-5" />
                      智能体配置
                    </CardTitle>
                    <CardDescription>
                      配置参与场景的智能体人设和目标
                    </CardDescription>
                  </div>
                  <div className="flex items-center gap-2">
                    <Select
                      value=""
                      onValueChange={v => {
                        const id = parseInt(v)
                        if (!selectedAccounts.includes(id)) {
                          setSelectedAccounts([...selectedAccounts, id])
                        }
                      }}
                    >
                      <SelectTrigger className="w-[200px]">
                        <SelectValue placeholder="选择账号添加" />
                      </SelectTrigger>
                      <SelectContent>
                        {accounts
                          .filter(a => !scenario.agents.find(ag => ag.account_id === a.id))
                          .map(account => (
                            <SelectItem key={account.id} value={account.id.toString()}>
                              {account.phone} - {account.first_name || "未命名"}
                            </SelectItem>
                          ))}
                      </SelectContent>
                    </Select>
                    <Button onClick={handleAddAgent} disabled={selectedAccounts.length === 0}>
                      <Plus className="h-4 w-4 mr-1" />
                      添加
                    </Button>
                  </div>
                </div>
              </CardHeader>
              <CardContent>
                {scenario.agents.length === 0 ? (
                  <div className="text-center py-8 text-muted-foreground">
                    <Bot className="h-12 w-12 mx-auto mb-2 opacity-50" />
                    <p>暂未添加智能体</p>
                    <p className="text-sm">请从上方选择账号添加</p>
                  </div>
                ) : (
                  <div className="space-y-3">
                    {scenario.agents.map(agent => (
                      <div
                        key={agent.account_id}
                        className="flex items-center justify-between p-4 rounded-lg border bg-card hover:shadow-sm transition-shadow"
                      >
                        <div className="flex items-center gap-4">
                          <div className="h-10 w-10 rounded-full bg-gradient-to-br from-primary to-primary/60 flex items-center justify-center text-primary-foreground font-medium">
                            {agent.persona.name.charAt(0)}
                          </div>
                          <div>
                            <p className="font-medium">{agent.persona.name}</p>
                            <p className="text-sm text-muted-foreground">
                              {agent.persona.age}岁 · {agent.persona.occupation}
                            </p>
                            <p className="text-xs text-muted-foreground mt-1">
                              目标: {agent.goal}
                            </p>
                          </div>
                        </div>
                        <div className="flex items-center gap-4">
                          <div className="text-right text-sm">
                            <p>活跃度: {(agent.active_rate * 100).toFixed(0)}%</p>
                            <p className="text-muted-foreground text-xs">
                              {agent.account_phone}
                            </p>
                          </div>
                          <div className="flex gap-1">
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => handleEditAgent(agent)}
                            >
                              <Settings2 className="h-4 w-4" />
                            </Button>
                            <Button
                              variant="ghost"
                              size="icon"
                              onClick={() => handleRemoveAgent(agent.account_id)}
                            >
                              <Trash2 className="h-4 w-4 text-destructive" />
                            </Button>
                          </div>
                        </div>
                      </div>
                    ))}
                  </div>
                )}

                {scenario.agents.length > 0 && (
                  <div className="mt-6 pt-4 border-t">
                    <Button
                      className="w-full gap-2"
                      size="lg"
                      onClick={handleCreateScenario}
                      disabled={loading}
                    >
                      <Sparkles className="h-5 w-5" />
                      {loading ? "创建中..." : "启动场景任务"}
                    </Button>
                  </div>
                )}
              </CardContent>
            </Card>
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
            <div className="grid gap-4 md:grid-cols-3">
              <div className="p-4 rounded-lg bg-muted/30 border">
                <div className="flex items-center gap-2 mb-2">
                  <Zap className="h-5 w-5 text-yellow-500" />
                  <span className="font-medium">快速创建</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  简单配置，快速启动AI炒群任务，适合单一目标场景
                </p>
              </div>
              <div className="p-4 rounded-lg bg-muted/30 border">
                <div className="flex items-center gap-2 mb-2">
                  <Users className="h-5 w-5 text-blue-500" />
                  <span className="font-medium">场景配置</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  配置多个智能体人设，模拟真实群聊互动场景
                </p>
              </div>
              <div className="p-4 rounded-lg bg-muted/30 border">
                <div className="flex items-center gap-2 mb-2">
                  <Bot className="h-5 w-5 text-green-500" />
                  <span className="font-medium">智能决策</span>
                </div>
                <p className="text-sm text-muted-foreground">
                  AI根据聊天上下文自动决策是否发言及发言内容
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
            <DialogTitle>编辑智能体配置</DialogTitle>
            <DialogDescription>
              配置智能体的人设、目标和行为参数
            </DialogDescription>
          </DialogHeader>

          {editingAgent && (
            <div className="space-y-4 py-4">
              <div className="grid grid-cols-2 gap-4">
                <div className="space-y-2">
                  <Label>角色名称</Label>
                  <Input
                    value={editingAgent.persona.name}
                    onChange={e =>
                      setEditingAgent({
                        ...editingAgent,
                        persona: { ...editingAgent.persona, name: e.target.value },
                      })
                    }
                  />
                </div>
                <div className="space-y-2">
                  <Label>年龄</Label>
                  <Input
                    type="number"
                    value={editingAgent.persona.age}
                    onChange={e =>
                      setEditingAgent({
                        ...editingAgent,
                        persona: { ...editingAgent.persona, age: parseInt(e.target.value) || 25 },
                      })
                    }
                  />
                </div>
              </div>

              <div className="space-y-2">
                <Label>职业</Label>
                <Input
                  value={editingAgent.persona.occupation}
                  onChange={e =>
                    setEditingAgent({
                      ...editingAgent,
                      persona: { ...editingAgent.persona, occupation: e.target.value },
                    })
                  }
                  placeholder="例如: 产品经理、程序员、自由职业者"
                />
              </div>

              <div className="space-y-2">
                <Label>说话风格 (逗号分隔)</Label>
                <Input
                  value={editingAgent.persona.style.join(", ")}
                  onChange={e =>
                    setEditingAgent({
                      ...editingAgent,
                      persona: {
                        ...editingAgent.persona,
                        style: e.target.value.split(",").map(s => s.trim()).filter(s => s),
                      },
                    })
                  }
                  placeholder="友好, 热情, 专业"
                />
              </div>

              <div className="space-y-2">
                <Label>核心观点 (逗号分隔)</Label>
                <Input
                  value={editingAgent.persona.beliefs.join(", ")}
                  onChange={e =>
                    setEditingAgent({
                      ...editingAgent,
                      persona: {
                        ...editingAgent.persona,
                        beliefs: e.target.value.split(",").map(s => s.trim()).filter(s => s),
                      },
                    })
                  }
                  placeholder="支持创新, 注重效率"
                />
              </div>

              <div className="space-y-2">
                <Label>行动目标</Label>
                <Textarea
                  value={editingAgent.goal}
                  onChange={e => setEditingAgent({ ...editingAgent, goal: e.target.value })}
                  placeholder="描述这个智能体在群聊中的目标..."
                />
              </div>

              <div className="space-y-2">
                <Label>活跃度 ({(editingAgent.active_rate * 100).toFixed(0)}%)</Label>
                <Input
                  type="range"
                  min="0.1"
                  max="1"
                  step="0.1"
                  value={editingAgent.active_rate}
                  onChange={e =>
                    setEditingAgent({ ...editingAgent, active_rate: parseFloat(e.target.value) })
                  }
                  className="w-full"
                />
              </div>

            </div>
          )}

          <DialogFooter>
            <Button variant="outline" onClick={() => setAgentDialogOpen(false)}>
              取消
            </Button>
            <Button onClick={handleSaveAgent}>保存</Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      {/* 创建场景对话框 */}
      <Dialog open={createDialogOpen} onOpenChange={setCreateDialogOpen}>
        <DialogContent className="sm:max-w-[600px]">
          <DialogHeader>
            <DialogTitle>创建场景任务</DialogTitle>
            <DialogDescription>
              配置多智能体协作场景，模拟真实群聊互动
            </DialogDescription>
          </DialogHeader>

          <div className="py-4">
            <p className="text-sm text-muted-foreground mb-4">
              请在"场景配置"标签页中完成配置后，点击"启动场景任务"按钮创建任务。
            </p>
            <div className="p-4 bg-muted/50 rounded-lg">
              <p className="text-sm">当前配置:</p>
              <ul className="text-sm text-muted-foreground mt-2 space-y-1">
                <li>• 目标群组: {scenario.topic || "未设置"}</li>
                <li>• 智能体数量: {scenario.agents.length}</li>
                <li>• 持续时间: {Math.floor(scenario.duration / 60)} 分钟</li>
              </ul>
            </div>
          </div>

          <DialogFooter>
            <Button variant="outline" onClick={() => setCreateDialogOpen(false)}>
              关闭
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </MainLayout>
  )
}
