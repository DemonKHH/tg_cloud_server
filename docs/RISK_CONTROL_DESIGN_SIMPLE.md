# 账号风控系统设计文档（简化版）

## 一、概述

### 1.1 设计目标

- **简单易用**：最少的配置项，最直观的逻辑
- **核心保护**：覆盖最关键的风控场景
- **零新增表**：复用现有数据结构，降低实现成本
- **用户可控**：关键参数允许用户自定义

### 1.2 核心理念

> 只做必要的风控，不做过度设计

风控的本质是：
1. **响应 Telegram 的限制** - 平台告诉我们账号有问题，我们就处理
2. **预防连续失败** - 连续失败说明账号可能有问题，需要冷却
3. **自动恢复** - 冷却结束后自动恢复，减少人工干预

---

## 二、账号状态定义

### 2.1 状态枚举

| 状态 | 标识 | 说明 | 可执行任务 | 恢复方式 |
|------|------|------|------------|----------|
| 新建 | `new` | 刚导入，未验证 | 所有任务 | 检查通过→normal |
| 正常 | `normal` | 状态正常 | 所有任务 | - |
| 警告 | `warning` | 有异常但可用 | 所有任务 | 24h无错误自动恢复 |
| 冷却 | `cooling` | 触发限流/连续失败 | 不可执行 | 冷却时间到期 |
| 受限 | `restricted` | 被Telegram限制 | 所有任务（有风险） | 手动检测 |
| 双向 | `two_way` | SpamBot双向限制 | 所有任务（有风险） | 手动检测 |
| 冻结 | `frozen` | SpamBot冻结 | 不可执行 | 解冻时间后检测 |
| 死亡 | `dead` | 永久封禁 | 不可执行 | 不可恢复 |

### 2.2 状态转换图

```
                              ┌─────────┐
                              │   new   │ ─────────────────────┐
                              └────┬────┘                      │
                                   │ 检查通过                  │ 可执行任务
                                   ▼                           │
┌──────────────────────────────────────────────────────────────┤
│                                                              │
│    ┌─────────┐  连续失败/限流  ┌─────────┐                  │
│    │ normal  │ ──────────────→ │ cooling │ ← 不可执行任务   │
│    └────┬────┘                 └────┬────┘                  │
│         │                           │                        │
│         │ 单次失败                  │ 冷却到期               │
│         ▼                           │                        │
│    ┌─────────┐                      │                        │
│    │ warning │ ◄─────────┐          │                        │
│    └────┬────┘           │          │                        │
│         │ 24h无错误      │          │                        │
│         │                │          │                        │
│         └────────────────┴──────────┴───────→ normal         │
│                                                              │
└──────────────────────────────────────────────────────────────┘
                    │
                    │ Telegram错误/SpamBot检测
                    ▼
        ┌───────────┬───────────┬───────────┐
        │           │           │           │
        ▼           ▼           ▼           ▼
   ┌─────────┐ ┌─────────┐ ┌─────────┐ ┌─────────┐
   │restricted│ │ two_way │ │ frozen  │ │  dead   │
   │ 可执行   │ │ 可执行   │ │ 不可执行 │ │ 不可执行 │
   └─────────┘ └─────────┘ └─────────┘ └─────────┘
       │           │           │           │
       │ 手动检测   │ 手动检测   │ 解冻后检测 │ 不可恢复
       └───────────┴───────────┴───────────┘

注：restricted 和 two_way 状态可执行任务，但可能会失败，失败会记录日志
```

---

## 三、风控规则

### 3.1 规则总览

简化版只有 **3 类规则**：

| 规则类型 | 触发条件 | 处理动作 | 可配置 |
|----------|----------|----------|--------|
| Telegram错误处理 | 收到特定错误码 | 按映射表更新状态 | ❌ 系统固定 |
| 连续失败检测 | 连续失败N次 | 进入冷却状态 | ✅ 用户可配置 |
| SpamBot结果处理 | SpamBot返回限制信息 | 更新为对应状态 | ❌ 系统固定 |

### 3.2 Telegram 错误处理规则（系统固定）

#### 致命错误 → Dead（永久）

| 错误码 | 说明 |
|--------|------|
| `AUTH_KEY_UNREGISTERED` | Session 失效 |
| `USER_DEACTIVATED` | 用户已注销 |
| `PHONE_NUMBER_BANNED` | 手机号被封禁 |
| `SESSION_REVOKED` | Session 被撤销 |

#### 限流错误 → Cooling（临时）

| 错误码 | 冷却时长 |
|--------|----------|
| `FLOOD_WAIT_X` | X秒 + 60秒缓冲 |
| `SLOWMODE_WAIT` | 30分钟 |
| `PEER_FLOOD` | 1小时 |
| `PHONE_NUMBER_FLOOD` | 24小时 |

#### 限制错误 → Restricted

| 错误码 | 说明 |
|--------|------|
| `USER_RESTRICTED` | 用户受限 |
| `CHAT_WRITE_FORBIDDEN` | 禁止发言 |
| `CHAT_RESTRICTED` | 聊天受限 |

### 3.3 连续失败检测规则（用户可配置）

```
触发条件: 连续失败次数 >= max_consecutive_failures
处理动作: 
  1. 状态 → cooling
  2. 设置 cooling_until = 当前时间 + cooling_duration
  3. 重置 consecutive_failures = 0
```

**默认值：**
- `max_consecutive_failures` = 5 次
- `cooling_duration` = 30 分钟

**用户可配置范围：**
- `max_consecutive_failures`: 3 ~ 10 次
- `cooling_duration`: 10 ~ 120 分钟

### 3.4 SpamBot 检测结果处理（系统固定）

| SpamBot 返回 | 目标状态 | 说明 |
|--------------|----------|------|
| 账号正常 | `normal` | 无限制 |
| 双向限制 | `two_way` | 只能与已有联系人通信 |
| 临时冻结 | `frozen` | 记录 `frozen_until` 解冻时间 |
| 永久冻结 | `dead` | 无解冻时间 |

---

## 四、自动恢复机制

### 4.1 恢复规则

| 当前状态 | 恢复条件 | 目标状态 | 执行时机 |
|----------|----------|----------|----------|
| `cooling` | `cooling_until` 到期 | `normal` | 定时任务 (每5分钟) |
| `warning` | 24小时内无新错误 | `normal` | 定时任务 (每10分钟) |

### 4.2 恢复流程

```
定时任务执行:

1. 查询所有 status='cooling' 且 cooling_until < now 的账号
   → 更新 status='normal', cooling_until=NULL

2. 查询所有 status='warning' 且 updated_at < (now - 24h) 的账号
   → 更新 status='normal'
```

---

## 五、数据模型

### 5.1 账号表修改（tg_accounts）

在现有 `tg_accounts` 表添加字段：

```sql
ALTER TABLE tg_accounts 
ADD COLUMN consecutive_failures INT UNSIGNED DEFAULT 0 
    COMMENT '连续失败次数',
ADD COLUMN cooling_until TIMESTAMP NULL 
    COMMENT '冷却结束时间';
```

### 5.2 用户表修改（users）

在现有 `users` 表添加字段：

```sql
ALTER TABLE users 
ADD COLUMN risk_settings JSON DEFAULT NULL 
    COMMENT '用户风控配置';
```

**risk_settings JSON 结构：**

```json
{
  "max_consecutive_failures": 5,
  "cooling_duration_minutes": 30
}
```

### 5.3 Go 模型更新

```go
// TGAccount 添加字段
type TGAccount struct {
    // ... 现有字段 ...
    
    ConsecutiveFailures uint32     `json:"consecutive_failures" gorm:"default:0"`
    CoolingUntil        *time.Time `json:"cooling_until"`
}

// UserRiskSettings 用户风控配置
type UserRiskSettings struct {
    MaxConsecutiveFailures int `json:"max_consecutive_failures"` // 默认5，范围3-10
    CoolingDurationMinutes int `json:"cooling_duration_minutes"` // 默认30，范围10-120
}

// User 添加字段
type User struct {
    // ... 现有字段 ...
    
    RiskSettings *UserRiskSettings `json:"risk_settings" gorm:"type:json"`
}
```

---

## 六、风控服务接口

### 6.1 接口定义

```go
// RiskControlService 风控服务接口
type RiskControlService interface {
    // CanExecuteTask 检查账号是否可以执行任务
    // 返回: allowed-是否允许, reason-拒绝原因
    CanExecuteTask(ctx context.Context, accountID uint64, taskType TaskType) (allowed bool, reason string)
    
    // ReportTaskResult 上报任务执行结果
    // 根据结果更新连续失败计数，触发风控规则
    ReportTaskResult(ctx context.Context, accountID uint64, success bool, taskErr error)
    
    // HandleTelegramError 处理Telegram错误
    // 根据错误类型更新账号状态
    HandleTelegramError(ctx context.Context, accountID uint64, err error)
    
    // ProcessCoolingRecovery 处理冷却恢复（定时任务调用）
    ProcessCoolingRecovery(ctx context.Context) (recoveredCount int)
    
    // ProcessWarningRecovery 处理警告恢复（定时任务调用）
    ProcessWarningRecovery(ctx context.Context) (recoveredCount int)
    
    // GetUserRiskSettings 获取用户风控配置
    GetUserRiskSettings(ctx context.Context, userID uint64) *UserRiskSettings
    
    // UpdateUserRiskSettings 更新用户风控配置
    UpdateUserRiskSettings(ctx context.Context, userID uint64, settings *UserRiskSettings) error
}
```

### 6.2 核心方法实现逻辑

#### CanExecuteTask 检查逻辑

```go
func (s *riskControlService) CanExecuteTask(ctx context.Context, accountID uint64, taskType TaskType) (bool, string) {
    account, err := s.accountRepo.GetByID(accountID)
    if err != nil {
        return false, "账号不存在"
    }
    
    // 检查账号状态
    switch account.Status {
    case AccountStatusDead:
        return false, "账号已死亡，无法执行任务"
    
    case AccountStatusFrozen:
        return false, "账号已冻结，无法执行任务"
    
    case AccountStatusCooling:
        // 检查冷却是否到期
        if account.CoolingUntil != nil && account.CoolingUntil.After(time.Now()) {
            remaining := account.CoolingUntil.Sub(time.Now())
            return false, fmt.Sprintf("账号冷却中，剩余 %v", remaining.Round(time.Minute))
        }
        // 冷却已到期，允许执行（定时任务会恢复状态）
    
    case AccountStatusRestricted, AccountStatusTwoWay:
        // 允许执行，但记录警告日志
        s.logger.Warn("Executing task on restricted/two_way account",
            zap.Uint64("account_id", accountID),
            zap.String("status", string(account.Status)),
            zap.String("task_type", string(taskType)))
        // 继续执行，任务失败会有日志记录
    }
    
    // new, normal, warning, restricted, two_way 都允许执行
    return true, ""
}
```


#### ReportTaskResult 上报逻辑

```go
func (s *riskControlService) ReportTaskResult(ctx context.Context, accountID uint64, success bool, taskErr error) {
    account, err := s.accountRepo.GetByID(accountID)
    if err != nil {
        return
    }
    
    // 获取用户风控配置
    settings := s.GetUserRiskSettings(ctx, account.UserID)
    
    if success {
        // 成功：重置连续失败计数
        if account.ConsecutiveFailures > 0 {
            account.ConsecutiveFailures = 0
            s.accountRepo.Update(account)
        }
        return
    }
    
    // 失败：增加连续失败计数
    account.ConsecutiveFailures++
    
    // 检查是否触发冷却
    if int(account.ConsecutiveFailures) >= settings.MaxConsecutiveFailures {
        // 触发冷却
        account.Status = AccountStatusCooling
        coolingUntil := time.Now().Add(time.Duration(settings.CoolingDurationMinutes) * time.Minute)
        account.CoolingUntil = &coolingUntil
        account.ConsecutiveFailures = 0 // 重置计数
        
        s.logger.Warn("Account triggered cooling due to consecutive failures",
            zap.Uint64("account_id", accountID),
            zap.Time("cooling_until", coolingUntil))
    }
    
    s.accountRepo.Update(account)
}
```

#### HandleTelegramError 错误处理逻辑

```go
func (s *riskControlService) HandleTelegramError(ctx context.Context, accountID uint64, err error) {
    if err == nil {
        return
    }
    
    account, getErr := s.accountRepo.GetByID(accountID)
    if getErr != nil {
        return
    }
    
    errorStr := strings.ToUpper(err.Error())
    var newStatus AccountStatus
    var coolingDuration time.Duration
    
    // 致命错误 → Dead
    if strings.Contains(errorStr, "AUTH_KEY_UNREGISTERED") ||
       strings.Contains(errorStr, "USER_DEACTIVATED") ||
       strings.Contains(errorStr, "PHONE_NUMBER_BANNED") ||
       strings.Contains(errorStr, "SESSION_REVOKED") {
        newStatus = AccountStatusDead
    
    // 限流错误 → Cooling
    } else if strings.Contains(errorStr, "FLOOD_WAIT") {
        newStatus = AccountStatusCooling
        // 解析等待时间
        waitSeconds := s.parseFloodWaitSeconds(errorStr)
        coolingDuration = time.Duration(waitSeconds+60) * time.Second
    
    } else if strings.Contains(errorStr, "PEER_FLOOD") {
        newStatus = AccountStatusCooling
        coolingDuration = 1 * time.Hour
    
    } else if strings.Contains(errorStr, "PHONE_NUMBER_FLOOD") {
        newStatus = AccountStatusCooling
        coolingDuration = 24 * time.Hour
    
    } else if strings.Contains(errorStr, "SLOWMODE_WAIT") {
        newStatus = AccountStatusCooling
        coolingDuration = 30 * time.Minute
    
    // 限制错误 → Restricted
    } else if strings.Contains(errorStr, "USER_RESTRICTED") ||
              strings.Contains(errorStr, "CHAT_WRITE_FORBIDDEN") ||
              strings.Contains(errorStr, "CHAT_RESTRICTED") {
        newStatus = AccountStatusRestricted
    
    } else {
        // 其他错误不处理
        return
    }
    
    // 更新状态
    oldStatus := account.Status
    account.Status = newStatus
    
    if coolingDuration > 0 {
        coolingUntil := time.Now().Add(coolingDuration)
        account.CoolingUntil = &coolingUntil
    }
    
    s.accountRepo.Update(account)
    
    s.logger.Warn("Account status changed due to Telegram error",
        zap.Uint64("account_id", accountID),
        zap.String("old_status", string(oldStatus)),
        zap.String("new_status", string(newStatus)),
        zap.String("error", err.Error()))
}
```

---

## 七、用户配置

### 7.1 配置项说明

| 配置项 | 类型 | 默认值 | 范围 | 说明 |
|--------|------|--------|------|------|
| `max_consecutive_failures` | int | 5 | 3-10 | 连续失败多少次触发冷却 |
| `cooling_duration_minutes` | int | 30 | 10-120 | 冷却时长（分钟） |

### 7.2 配置获取逻辑

```go
// GetUserRiskSettings 获取用户风控配置（带默认值和范围限制）
func (s *riskControlService) GetUserRiskSettings(ctx context.Context, userID uint64) *UserRiskSettings {
    // 默认配置
    defaults := &UserRiskSettings{
        MaxConsecutiveFailures: 5,
        CoolingDurationMinutes: 30,
    }
    
    user, err := s.userRepo.GetByID(userID)
    if err != nil || user.RiskSettings == nil {
        return defaults
    }
    
    settings := user.RiskSettings
    
    // 应用范围限制: max_consecutive_failures 3-10
    if settings.MaxConsecutiveFailures < 3 {
        settings.MaxConsecutiveFailures = 3
    } else if settings.MaxConsecutiveFailures > 10 {
        settings.MaxConsecutiveFailures = 10
    }
    
    // 应用范围限制: cooling_duration_minutes 10-120
    if settings.CoolingDurationMinutes < 10 {
        settings.CoolingDurationMinutes = 10
    } else if settings.CoolingDurationMinutes > 120 {
        settings.CoolingDurationMinutes = 120
    }
    
    return settings
}
```

### 7.3 API 接口

#### 获取风控配置

```
GET /api/v1/settings/risk
```

**响应：**
```json
{
  "code": 0,
  "data": {
    "max_consecutive_failures": 5,
    "cooling_duration_minutes": 30
  }
}
```

#### 更新风控配置

```
PUT /api/v1/settings/risk
```

**请求体：**
```json
{
  "max_consecutive_failures": 5,
  "cooling_duration_minutes": 30
}
```

**参数校验：**
- `max_consecutive_failures`: 必须在 3-10 之间
- `cooling_duration_minutes`: 必须在 10-120 之间

---

## 八、定时任务

### 8.1 冷却恢复任务

**执行频率：** 每 5 分钟

```go
func (s *riskControlService) ProcessCoolingRecovery(ctx context.Context) int {
    // 查询所有冷却到期的账号: status='cooling' AND cooling_until < NOW()
    accounts, err := s.accountRepo.GetCoolingExpiredAccounts()
    if err != nil {
        return 0
    }
    
    recoveredCount := 0
    for _, account := range accounts {
        account.Status = AccountStatusNormal
        account.CoolingUntil = nil
        
        if err := s.accountRepo.Update(account); err == nil {
            recoveredCount++
            s.logger.Info("Account recovered from cooling",
                zap.Uint64("account_id", account.ID))
        }
    }
    
    return recoveredCount
}
```

### 8.2 警告恢复任务

**执行频率：** 每 10 分钟

```go
func (s *riskControlService) ProcessWarningRecovery(ctx context.Context) int {
    // 查询所有 warning 状态且 24 小时未更新的账号
    // status='warning' AND updated_at < (NOW() - 24h)
    cutoffTime := time.Now().Add(-24 * time.Hour)
    accounts, err := s.accountRepo.GetWarningAccountsOlderThan(cutoffTime)
    if err != nil {
        return 0
    }
    
    recoveredCount := 0
    for _, account := range accounts {
        account.Status = AccountStatusNormal
        
        if err := s.accountRepo.Update(account); err == nil {
            recoveredCount++
            s.logger.Info("Account recovered from warning",
                zap.Uint64("account_id", account.ID))
        }
    }
    
    return recoveredCount
}
```

---

## 九、前端界面

### 9.1 风控设置页面

```
┌─────────────────────────────────────────────────────────────┐
│                       风控设置                               │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  📊 连续失败保护                                             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                                                     │   │
│  │  连续失败 [  5  ▼] 次后触发冷却      (可选: 3-10)   │   │
│  │                                                     │   │
│  │  冷却时长 [ 30  ▼] 分钟              (可选: 10-120) │   │
│  │                                                     │   │
│  └─────────────────────────────────────────────────────┘   │
│                                                             │
│  ℹ️ 说明：                                                  │
│  • 冷却结束后账号自动恢复为正常状态                         │
│  • 警告状态 24 小时无错误后自动恢复                         │
│  • Telegram 返回的限流错误会自动触发冷却，冷却时长由        │
│    Telegram 决定，不受上述设置影响                          │
│  • 账号被封禁（Dead）或冻结（Frozen）状态无法自动恢复       │
│                                                             │
│                              [恢复默认]    [保存设置]       │
└─────────────────────────────────────────────────────────────┘
```

### 9.2 账号列表中的风控状态展示

```
┌──────────────────────────────────────────────────────────────────────┐
│ 手机号          状态      连续失败    冷却剩余    操作               │
├──────────────────────────────────────────────────────────────────────┤
│ +1234567890    🟢 正常    0          -          [检查] [任务]       │
│ +1234567891    🟡 警告    2          -          [检查] [任务]       │
│ +1234567892    🔵 冷却    0          15分钟     [等待中...]         │
│ +1234567893    🟠 受限    0          -          [检查] [任务] ⚠️    │
│ +1234567894    🟡 双向    0          -          [检查] [任务] ⚠️    │
│ +1234567895    🔴 死亡    -          -          [删除]              │
└──────────────────────────────────────────────────────────────────────┘

⚠️ 表示该状态下执行任务可能失败
```

---

## 十、实现清单

### 10.1 数据库变更

- [ ] `tg_accounts` 表添加 `consecutive_failures` 字段
- [ ] `tg_accounts` 表添加 `cooling_until` 字段
- [ ] `users` 表添加 `risk_settings` 字段

### 10.2 后端实现

- [ ] 创建 `RiskControlService` 服务
- [ ] 实现 `CanExecuteTask` 方法
- [ ] 实现 `ReportTaskResult` 方法
- [ ] 实现 `HandleTelegramError` 方法
- [ ] 实现 `ProcessCoolingRecovery` 定时任务
- [ ] 实现 `ProcessWarningRecovery` 定时任务
- [ ] 实现风控配置 API

### 10.3 集成

- [ ] 任务调度器集成 `CanExecuteTask`
- [ ] 任务调度器集成 `ReportTaskResult`
- [ ] 连接池集成 `HandleTelegramError`
- [ ] 定时任务注册恢复任务

### 10.4 前端实现

- [ ] 风控设置页面
- [ ] 账号列表风控状态展示
- [ ] 状态变化通知

---

## 十一、与复杂版对比

| 项目 | 复杂版 | 简化版 |
|------|--------|--------|
| 新增数据表 | 2个 | 0个 |
| 新增字段 | 20+ | 3个 |
| 用户可配置项 | 15+ | 2个 |
| 风控规则数量 | 15+ | 3类 |
| 风险评分系统 | ✅ 有 | ❌ 无 |
| 时间窗口统计 | ✅ 有 | ❌ 无 |
| 风控日志表 | ✅ 有 | ❌ 无 |
| 实现复杂度 | 高 | 低 |
| 开发时间 | 5-7天 | 1-2天 |

**简化版保留的核心能力：**
- ✅ Telegram 错误自动处理
- ✅ 连续失败保护
- ✅ 自动冷却和恢复
- ✅ SpamBot 检测结果处理
- ✅ 用户可调整敏感度

**简化版去掉的功能：**
- ❌ 复杂的风险评分算法
- ❌ 小时/天窗口统计
- ❌ 失败率计算
- ❌ 详细的风控日志记录
- ❌ 风控趋势分析

---

*文档版本: 1.0 (简化版)*
*最后更新: 2024-12-12*
