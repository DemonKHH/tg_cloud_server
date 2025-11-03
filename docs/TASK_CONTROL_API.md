# 🎮 任务控制系统 API 指南

## 🎯 概述

现在任务系统支持完整的执行控制，您可以精确控制任务何时启动、暂停、恢复或停止，而不是创建后自动执行。

## 🆕 新增功能

### ✅ 任务状态
- **pending**: 待执行
- **queued**: 已排队
- **running**: 执行中
- **paused**: 已暂停 ⭐ **新增**
- **completed**: 已完成
- **failed**: 失败
- **cancelled**: 已取消

### ✅ 控制能力
- ⭐ **auto_start**: 创建时控制是否自动执行
- 🎮 **手动启动**: 按需启动任务
- ⏸️ **暂停/恢复**: 灵活控制执行
- 🛑 **停止**: 随时终止任务
- 📦 **批量控制**: 同时控制多个任务

## 📋 API 接口详解

### 1. 创建任务（支持执行控制）

**POST** `/api/v1/tasks`

```json
{
  "task_type": "private_message",
  "account_id": 123,
  "priority": 5,
  "auto_start": false,  // ⭐ 新增：是否自动启动
  "task_config": {
    "targets": ["@user1", "@user2"],
    "message": "Hello",
    "timeout_seconds": 60
  }
}
```

**响应**:
```json
{
  "code": 200,
  "message": "success", 
  "data": {
    "id": 456,
    "status": "pending",  // auto_start=false时保持pending
    "task_type": "private_message",
    "account_id": 123,
    "created_at": "2024-01-15T10:30:45Z"
  }
}
```

### 2. 单个任务控制

**POST** `/api/v1/tasks/{id}/control`

```json
{
  "action": "start"  // start|pause|stop|resume
}
```

**支持的操作**:
- `start`: 启动pending或paused状态的任务
- `pause`: 暂停queued或running状态的任务  
- `stop`: 停止任务（等同于取消）
- `resume`: 恢复paused状态的任务

**响应**:
```json
{
  "code": 200,
  "message": "任务启动成功",
  "data": {
    "task_id": 456,
    "action": "start"
  }
}
```

### 3. 批量任务控制

**POST** `/api/v1/tasks/batch/control`

```json
{
  "task_ids": [456, 789, 101112],
  "action": "start"  // start|pause|stop|resume|cancel
}
```

**响应**:
```json
{
  "code": 200,
  "message": "批量启动完成",
  "data": {
    "total_tasks": 3,
    "success_count": 3,
    "failed_count": 0,
    "action": "start"
  }
}
```

## 🔄 任务状态流转

```
创建任务
    ↓
[auto_start=true]  → pending → queued → running → completed
    ↓                  ↑           ↓         ↓        
[auto_start=false] → pending    paused ←── ↓     failed
    ↓                  ↓           ↓              
手动start ────────────┘         stop        cancelled
    ↓
resume ──→ paused
```

## 📝 使用示例

### 场景1：创建任务但不立即执行

```bash
# 1. 创建任务（不自动启动）
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "task_type": "check",
    "account_id": 123,
    "auto_start": false,
    "task_config": {
      "timeout_seconds": 30
    }
  }'

# 响应: {"data": {"id": 456, "status": "pending"}}

# 2. 稍后手动启动
curl -X POST http://localhost:8080/api/v1/tasks/456/control \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "start"}'
```

### 场景2：暂停和恢复任务

```bash
# 1. 暂停正在执行的任务
curl -X POST http://localhost:8080/api/v1/tasks/456/control \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "pause"}'

# 2. 稍后恢复任务
curl -X POST http://localhost:8080/api/v1/tasks/456/control \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"action": "resume"}'
```

### 场景3：批量控制任务

```bash
# 批量启动多个任务
curl -X POST http://localhost:8080/api/v1/tasks/batch/control \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "task_ids": [456, 789, 101112],
    "action": "start"
  }'

# 批量暂停多个任务
curl -X POST http://localhost:8080/api/v1/tasks/batch/control \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "task_ids": [456, 789, 101112], 
    "action": "pause"
  }'
```

## 🎯 最佳实践

### 1. 任务创建策略
```javascript
// 推荐：默认不自动启动，给用户控制权
const task = {
  task_type: "private_message",
  account_id: accountId,
  auto_start: false,  // 默认false
  task_config: config
}

// 创建后可以选择性启动
if (shouldStartNow) {
  await startTask(taskId)
}
```

### 2. 错误处理
```javascript
try {
  await controlTask(taskId, 'start')
} catch (error) {
  if (error.status === 400) {
    // 任务状态不允许此操作
    console.log('任务当前状态不支持启动操作')
  } else if (error.status === 404) {
    // 任务不存在
    console.log('任务不存在')
  }
}
```

### 3. 状态检查
```javascript
// 启动前检查任务状态
const task = await getTask(taskId)
if (task.status === 'pending' || task.status === 'paused') {
  await startTask(taskId)
} else {
  console.log(`任务状态为${task.status}，无法启动`)
}
```

## ⚡ 性能优化建议

### 1. 批量操作
- 对多个任务使用批量控制接口，而不是逐个调用
- 批量操作有更好的性能和事务一致性

### 2. 状态监控
- 使用任务列表API定期检查状态，而不是频繁查询单个任务
- 合理设置轮询间隔，避免过度查询

### 3. 优先级管理
- 重要任务设置高优先级
- 批量启动时考虑任务优先级顺序

## 🚨 注意事项

### 1. 状态限制
- `start`: 只能启动 pending 或 paused 状态的任务
- `pause`: 只能暂停 queued 或 running 状态的任务
- `resume`: 只能恢复 paused 状态的任务
- `stop`: 可以停止任何未完成的任务

### 2. 权限要求
- 单个任务控制：需要任务所有权
- 批量控制：需要高级用户权限

### 3. 并发控制
- 暂停操作对正在运行的任务可能有延迟
- 批量操作中部分任务可能失败，检查返回的成功计数

## 📊 监控和日志

所有任务控制操作都会记录详细日志：

```bash
# 查看任务控制日志
Get-Content logs/task.log | Select-String "Task.*control"

# 查看特定任务的操作历史
Get-Content logs/task.log | Select-String "task_id.*456"
```

日志包含：
- 操作类型（start/pause/stop/resume）
- 任务ID和类型
- 操作结果（成功/失败）
- 操作时间和用户

现在您拥有完全的任务执行控制能力！🎉

可以根据业务需要灵活控制任务的启动时机，实现更精细的任务管理。
