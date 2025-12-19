// 任务配置字段中文映射

// 任务类型中文映射
export const taskTypeLabels: Record<string, string> = {
  check: "账号检查",
  private_message: "私信发送",
  broadcast: "群发消息",
  verify_code: "验证码接收",
  group_chat: "AI炒群",
  join_group: "批量加群",
  scenario: "场景炒群",
  force_add_group: "强拉进群",
  terminate_sessions: "踢出设备",
  update_2fa: "修改2FA",
}

// 任务状态中文映射
export const taskStatusLabels: Record<string, string> = {
  pending: "待执行",
  queued: "已排队",
  running: "执行中",
  paused: "已暂停",
  completed: "已完成",
  failed: "失败",
  cancelled: "已取消",
}

// AI性格中文映射
export const personalityLabels: Record<string, string> = {
  friendly: "友好热情",
  professional: "专业严谨",
  humorous: "幽默风趣",
  casual: "随意轻松",
}

// 通用配置字段中文映射
export const configFieldLabels: Record<string, string> = {
  // 群组相关
  group_id: "群组ID",
  group_name: "目标群组",
  group_link: "群组链接",
  group_username: "群组用户名",
  target_group: "目标群组",
  topic: "目标群组",
  groups: "群组列表",

  // 用户相关
  target_user: "目标用户",
  target_users: "目标用户",
  targets: "目标用户",
  user_id: "用户ID",
  user_ids: "用户列表",

  // 消息相关
  message: "消息内容",
  content: "内容",
  text: "文本",

  // 时间相关
  duration: "持续时间",
  monitor_duration_seconds: "持续时间",
  interval: "发送间隔",
  interval_seconds: "发送间隔",
  delay: "延迟时间",
  timeout: "超时时间",

  // 数量相关
  count: "数量",
  limit: "限制",
  max_count: "最大数量",
  limit_per_account: "单号限制",

  // 状态相关
  enabled: "启用状态",
  active: "激活状态",
  auto_start: "创建后自动启动",
  auto_reply: "自动回复",
  auto_join: "自动加群",

  // AI相关
  ai_config: "AI配置",
  personality: "AI性格",
  response_rate: "回复概率",
  keywords: "触发关键词",
  active_rate: "活跃度",

  // 场景相关
  name: "场景名称",
  description: "描述",
  agents: "智能体列表",
  persona: "人设",
  goal: "目标",
  style: "风格",

  // 2FA相关
  hint: "密码提示",
  password: "密码",
  new_password: "新密码",
  old_password: "旧密码",

  // 其他
  keep_current: "保留当前",
  check_type: "检查类型",
  priority: "优先级",
  source: "发送者",

  // 账号检查相关
  check_2fa: "2FA 检查",
  check_spam_bot: "双向/冻结检查",
  two_fa_password: "2FA 密码",

  // 验证码相关
  verify_timeout: "超时时间",
  verify_source: "发送者",
}

// 获取任务类型中文名
export function getTaskTypeLabel(type: string): string {
  return taskTypeLabels[type] || type
}

// 获取任务状态中文名
export function getTaskStatusLabel(status: string): string {
  return taskStatusLabels[status] || status
}

// 获取AI性格中文名
export function getPersonalityLabel(personality: string): string {
  return personalityLabels[personality] || personality
}

// 获取配置字段中文名
export function getConfigFieldLabel(field: string): string {
  return configFieldLabels[field] || field
}

// 格式化时间值（秒转分钟/小时）
export function formatDuration(seconds: number): string {
  if (seconds < 60) {
    return `${seconds} 秒`
  } else if (seconds < 3600) {
    return `${Math.floor(seconds / 60)} 分钟`
  } else {
    const hours = Math.floor(seconds / 3600)
    const mins = Math.floor((seconds % 3600) / 60)
    return mins > 0 ? `${hours} 小时 ${mins} 分钟` : `${hours} 小时`
  }
}

// 格式化百分比
export function formatPercent(value: number): string {
  return `${(value * 100).toFixed(0)}%`
}

// 格式化配置值
export function formatConfigValue(key: string, value: any): string {
  if (value === null || value === undefined) {
    return "-"
  }
  
  // 布尔值
  if (typeof value === "boolean") {
    return value ? "是" : "否"
  }
  
  // 数组
  if (Array.isArray(value)) {
    if (value.length === 0) return "无"
    // 如果是简单数组，直接join
    if (typeof value[0] !== "object") {
      return value.join(", ")
    }
    return `${value.length} 项`
  }
  
  // 时间相关字段
  if (key.includes("duration") || key.includes("seconds") || key.includes("timeout") || key.includes("interval") || key.includes("delay")) {
    const num = Number(value)
    if (!isNaN(num)) {
      return formatDuration(num)
    }
  }
  
  // 概率/比率相关字段
  if (key.includes("rate") || key.includes("probability")) {
    const num = Number(value)
    if (!isNaN(num) && num <= 1) {
      return formatPercent(num)
    }
  }
  
  // AI性格
  if (key === "personality") {
    return getPersonalityLabel(String(value))
  }
  
  return String(value)
}

// 解析并格式化整个配置对象
export function parseTaskConfig(taskType: string, config: Record<string, any>): Array<{ label: string; value: string | React.ReactNode; isComplex?: boolean }> {
  if (!config || Object.keys(config).length === 0) {
    return []
  }

  const result: Array<{ label: string; value: string | React.ReactNode; isComplex?: boolean }> = []

  // 根据任务类型定义字段顺序和特殊处理
  const fieldOrder = getFieldOrder(taskType)
  const processedKeys = new Set<string>()

  // 按顺序处理已知字段
  for (const key of fieldOrder) {
    if (key in config && config[key] !== undefined && config[key] !== null && config[key] !== "") {
      processedKeys.add(key)
      const item = processConfigField(key, config[key], taskType)
      if (item) {
        result.push(item)
      }
    }
  }

  // 处理剩余字段
  for (const [key, value] of Object.entries(config)) {
    if (!processedKeys.has(key) && !key.startsWith("_") && value !== undefined && value !== null && value !== "") {
      const item = processConfigField(key, value, taskType)
      if (item) {
        result.push(item)
      }
    }
  }

  return result
}

// 获取字段顺序
function getFieldOrder(taskType: string): string[] {
  switch (taskType) {
    case "scenario":
      return ["name", "topic", "duration", "description", "agents"]
    case "group_chat":
      return ["group_name", "group_id", "monitor_duration_seconds", "ai_config"]
    case "private_message":
      return ["target_user", "message"]
    case "join_group":
      return ["group_link", "group_username"]
    case "force_add_group":
      return ["group_id", "user_ids"]
    case "terminate_sessions":
      return ["keep_current"]
    case "update_2fa":
      return ["hint"]
    case "broadcast":
      return ["message", "target_groups", "interval"]
    default:
      return []
  }
}

// 处理单个配置字段
function processConfigField(key: string, value: any, taskType: string): { label: string; value: string; isComplex?: boolean } | null {
  const label = getConfigFieldLabel(key)

  // 特殊处理 ai_config
  if (key === "ai_config" && typeof value === "object") {
    return null // ai_config 的子字段会单独处理
  }

  // 特殊处理 agents 数组
  if (key === "agents" && Array.isArray(value)) {
    return {
      label: `智能体配置`,
      value: `${value.length} 个智能体`,
      isComplex: true,
    }
  }

  // 特殊处理 persona 对象
  if (key === "persona" && typeof value === "object") {
    return null
  }

  return {
    label,
    value: formatConfigValue(key, value),
  }
}
