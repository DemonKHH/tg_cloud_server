# 健康度功能移除总结

## 概述
已完全移除账号健康度（health_score）功能，改为在连接和执行任务时自动检测并更新账号状态。

## 后端修改

### 1. 数据模型 (internal/models/)
- ✅ `account.go`: 移除 `AccountAvailability` 和 `ValidationResult` 中的 `HealthScore` 字段
- ✅ `stats.go`: 移除以下字段：
  - `DashboardQuickStats.AvgHealthScore`
  - `DashboardMetrics.HealthScoreTrend`
  - `AccountActivityStats.AvgHealthScore`
  - `AccountStatistics.HealthDistribution`

### 2. 数据访问层 (internal/repository/)
- ✅ `account_repo.go`: 移除以下方法和字段：
  - `UpdateHealthScore()` 方法
  - `GetAccountsNeedingHealthCheck()` 方法
  - `GetAccountsWithFilters()` 中的 `health_score_min/max` 过滤
  - `GetAccountSummaries()` 查询中的 `health_score` 字段

### 3. 服务层 (internal/services/)
- ✅ `account_service.go`: 移除 `CreateAccountsFromUploadData()` 中的 `HealthScore: 1.0` 初始化
- ✅ `batch_service.go`: CSV导出改为包含 `Last Check At` 和 `Last Used At` 而非健康度
- ✅ `stats_service.go`: 移除以下内容：
  - `getHealthScoreTrend()` 方法
  - `getAccountHealthDistribution()` 方法
  - 统计数据中的健康度相关字段

### 4. 任务调度 (internal/scheduler/)
- ✅ `task_scheduler.go`: 
  - 移除 `performRiskControlCheck()` 中的健康度检查
  - 移除 `ValidateAccountForTask()` 中的健康度警告
  - 更新 `generateRecommendations()` 改为基于账号状态生成建议

### 5. 连接池 (internal/telegram/)
- ✅ `connection_pool.go`: **新增自动状态检测功能**
  - `updateAccountStatusOnSuccess()`: 连接/任务成功时更新状态为 normal
  - `updateAccountStatusOnError()`: 连接失败时根据错误类型更新状态
  - `updateAccountStatusOnTaskError()`: 任务失败时根据错误类型更新状态
  - **在 `maintainConnection()` 中**：
    - 连接成功时调用 `updateAccountStatusOnSuccess()`
    - 连接错误时调用 `updateAccountStatusOnError()`
  - **在 `ExecuteTask()` 中**：
    - 任务执行前连接成功时调用 `updateAccountStatusOnSuccess()`
    - 任务执行失败时调用 `updateAccountStatusOnTaskError()`
  - 错误类型映射：
    - `AUTH_KEY_UNREGISTERED`, `USER_DEACTIVATED`, `PHONE_NUMBER_BANNED` → `dead`
    - `FLOOD_WAIT`, `SLOWMODE_WAIT`, `PEER_FLOOD` → `cooling`
    - `CHAT_WRITE_FORBIDDEN`, `USER_RESTRICTED`, `CHAT_RESTRICTED` → `restricted`
    - 其他连接错误 → `warning`

### 6. 任务执行器 (internal/telegram/)
- ✅ `task_executors.go`: 将 `health_score` 改名为 `check_score`（仅用于任务结果展示）

### 7. 定时任务 (internal/cron/)
- ✅ `cron.go`: 重写 `updateAccountStatuses()` 方法
  - 移除基于健康度的状态更新
  - 新增状态自动恢复逻辑：
    - `cooling` 状态超过1小时 → 恢复为 `normal`
    - `warning` 状态超过24小时 → 恢复为 `normal`

### 8. 监控指标 (internal/common/metrics/)
- ✅ `metrics.go`: 移除 `AccountHealthScore` 指标和相关方法

## 前端修改

### 1. 账号管理页面 (web/app/accounts/page.tsx)
- ✅ 健康检查结果提示：从显示健康度改为显示状态和问题数量
- ✅ 统计卡片：将"平均健康度"改为"正常账号"数量
- ✅ 表格列：将"健康度"列改为"连接状态"列
- ✅ 状态显示：用彩色圆点和文字显示连接状态

### 2. 首页 (web/app/page.tsx)
- ✅ 功能介绍：将"健康度评估"改为"状态监控"

### 3. 任务页面 (web/app/tasks/page.tsx)
- ✅ 任务说明：将"检查账号的状态和健康度"改为"检查账号的连接状态和可用性"

### 4. 验证码页面 (web/app/verify-codes/page.tsx)
- ✅ 类型定义：移除 `health_score` 字段

## 数据库迁移

创建了迁移脚本 `migrations/remove_health_score.sql`:
```sql
ALTER TABLE tg_accounts DROP COLUMN IF EXISTS health_score;
```

## 新的状态管理机制

### 自动状态更新规则

1. **连接建立成功时** (`maintainConnection`)
   - 如果账号状态为 `warning` 或 `new` → 更新为 `normal`
   - 更新 `last_check_at` 和 `last_used_at`

2. **连接断开/失败时** (`maintainConnection`)
   - 严重错误（AUTH_KEY_UNREGISTERED等） → `dead`
   - 限流错误（FLOOD_WAIT等） → `cooling`
   - 其他错误且当前为 `normal`/`new` → `warning`
   - 更新 `last_check_at`

3. **任务执行成功时** (`ExecuteTask`)
   - 如果账号状态为 `warning` 或 `new` → 更新为 `normal`
   - 更新 `last_check_at` 和 `last_used_at`

4. **任务执行失败时** (`ExecuteTask`)
   - 严重错误 → `dead`
   - 限流错误 → `cooling`
   - 权限错误 → `restricted`
   - 更新 `last_check_at`

5. **定时恢复** (定时任务)
   - `cooling` 状态超过1小时 → `normal`
   - `warning` 状态超过24小时 → `normal`

### 状态说明

- `new`: 新建账号，未使用
- `normal`: 正常可用
- `warning`: 出现异常，但可能恢复
- `cooling`: 触发限流，需要冷却
- `restricted`: 受到限制
- `dead`: 账号已死亡
- `maintenance`: 维护中

## 优势

1. **实时性更强**: 状态在每次连接和任务执行时实时更新
2. **更准确**: 基于实际的Telegram API错误响应，而非计算得分
3. **更简单**: 移除了复杂的健康度计算逻辑
4. **自动恢复**: 系统会自动尝试恢复异常状态的账号
5. **更直观**: 用户可以直接看到账号的实际状态

## 测试建议

1. 测试账号连接成功时状态更新
2. 测试各种错误类型的状态映射
3. 测试定时任务的状态恢复功能
4. 测试前端状态显示是否正确
5. 执行数据库迁移并验证

## 注意事项

- 需要执行数据库迁移脚本移除 `health_score` 字段
- 前端需要重新构建（删除 `.next` 目录）
- 现有的健康度数据将丢失，但会被新的状态系统替代
