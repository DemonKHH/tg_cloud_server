# 前端任务控制系统更新

本文档记录了为支持新的任务控制系统而进行的前端更新。

## 概述

前端已更新以支持以下功能：
- 任务创建时的自动执行选项
- 新的 `paused` 任务状态显示
- 任务控制操作（启动、暂停、恢复、停止）
- 批量任务控制 API 支持

## 文件更新

### 1. `web/app/tasks/page.tsx`

#### 新增图标导入
```typescript
import { Pause, Play, Square } from "lucide-react"
```

#### 任务状态支持
- **新增 `paused` 状态**：
  - 图标：`Pause` 橙色
  - 颜色：橙色主题
  - 文本：已暂停
  - 在状态过滤器中添加选项

#### 创建任务表单更新
- **新增 `auto_start` 字段**：表单状态中添加布尔字段
- **自动执行开关**：使用 Switch 组件，允许用户选择是否立即执行任务
- **请求数据**：创建任务时包含 `auto_start` 参数

#### 任务控制功能
新增任务控制处理函数：
- `handleStartTask()` - 启动任务
- `handlePauseTask()` - 暂停任务  
- `handleResumeTask()` - 恢复任务
- `handleStopTask()` - 停止任务

#### 操作菜单更新
根据任务状态显示相应的操作：

| 任务状态 | 可用操作 |
|---------|---------|
| `pending` | 启动、取消 |
| `running` | 暂停、停止 |
| `paused` | 恢复、停止 |
| `queued` | 停止、取消 |
| `failed`/`cancelled` | 重试 |

### 2. `web/lib/api.ts`

#### 任务控制 API
```typescript
// 单个任务控制
control: (id: string, action: 'start' | 'pause' | 'stop' | 'resume') =>
  apiClient.post(`/tasks/${id}/control`, { action })

// 批量任务控制  
batchControl: (ids: string[], action: 'start' | 'pause' | 'stop' | 'resume' | 'cancel') =>
  apiClient.post('/tasks/batch/control', { task_ids: ids, action })
```

## UI/UX 改进

### 创建任务表单
```typescript
<div className="flex items-center justify-between">
  <div className="space-y-0.5">
    <Label htmlFor="auto-start">自动执行</Label>
    <p className="text-sm text-muted-foreground">
      创建任务后立即开始执行
    </p>
  </div>
  <Switch
    id="auto-start"
    checked={createForm.auto_start}
    onCheckedChange={(checked) => setCreateForm({ ...createForm, auto_start: checked })}
  />
</div>
```

### 任务状态显示
- **暂停状态**：橙色的暂停图标和"已暂停"文本
- **状态过滤器**：可按暂停状态筛选任务

### 任务操作菜单
- **智能操作**：根据任务状态动态显示可用操作
- **清晰图标**：使用直观的图标（播放、暂停、停止）
- **操作分组**：控制操作、管理操作（取消/重试）分别显示

## 后端 API 对接

前端现在完全支持后端的任务控制 API：

### 创建任务
```typescript
POST /api/v1/tasks
{
  "account_id": 123,
  "task_type": "private_message", 
  "auto_start": true,  // 新增字段
  "task_config": { ... }
}
```

### 控制任务
```typescript
POST /api/v1/tasks/:id/control
{
  "action": "start" | "pause" | "resume" | "stop"
}
```

### 批量控制
```typescript
POST /api/v1/tasks/batch/control  
{
  "task_ids": [1, 2, 3],
  "action": "pause"
}
```

## 用户体验

### 工作流程改进
1. **灵活创建**：用户可选择立即执行或稍后手动启动
2. **精确控制**：支持暂停/恢复，而不只是取消/重试
3. **批量操作**：未来可扩展批量控制功能
4. **状态可视化**：清晰的状态指示和操作按钮

### 错误处理
- 所有控制操作都有错误处理和用户反馈
- Toast 消息提供操作结果反馈
- 操作失败时显示详细错误信息

## 兼容性

- 向后兼容：不影响现有任务和功能
- 渐进式增强：新功能不干扰原有工作流
- 状态管理：正确处理所有任务状态转换

## 总结

前端已全面支持新的任务控制系统，为用户提供了更精细的任务管理能力。通过自动执行开关、状态可视化和智能操作菜单，用户可以更好地控制任务的生命周期。
