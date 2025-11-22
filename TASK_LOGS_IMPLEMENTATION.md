# 任务日志功能实现

## 概述

为多账号任务执行添加了详细的日志记录功能，用户可以在前端查看任务执行的每个步骤。

## 后端实现

### 1. 日志记录点

在 `internal/scheduler/task_scheduler.go` 的 `executeTask` 方法中添加了以下日志记录点：

#### 任务级别日志
- **task_started**: 任务开始执行
- **task_completed**: 任务全部成功完成
- **task_partial_success**: 任务部分成功
- **task_failed**: 任务全部失败

#### 账号级别日志
- **account_started**: 开始使用某个账号执行
- **risk_check_passed**: 风控检查通过
- **risk_check_failed**: 风控检查失败
- **executor_creation_failed**: 创建任务执行器失败
- **execution_success**: 账号执行成功
- **execution_failed**: 账号执行失败

### 2. createTaskLog 方法

```go
func (ts *TaskScheduler) createTaskLog(taskID uint64, accountID *uint64, action, message string, extraData interface{}) {
    var extraDataJSON []byte
    if extraData != nil {
        extraDataJSON, _ = json.Marshal(extraData)
    } else {
        extraDataJSON = []byte("{}")
    }

    taskLog := &models.TaskLog{
        TaskID:    taskID,
        AccountID: accountID,
        Action:    action,
        Message:   message,
        ExtraData: extraDataJSON,
        CreatedAt: time.Now(),
    }

    ts.taskRepo.CreateTaskLog(taskLog)
}
```

### 3. 日志内容示例

```json
[
  {
    "id": 1,
    "task_id": 123,
    "account_id": null,
    "action": "task_started",
    "message": "开始执行任务，共 4 个账号",
    "created_at": "2025-11-23T10:00:00Z"
  },
  {
    "id": 2,
    "task_id": 123,
    "account_id": 1,
    "action": "account_started",
    "message": "开始使用账号 1 执行任务 (1/4)",
    "created_at": "2025-11-23T10:00:01Z"
  },
  {
    "id": 3,
    "task_id": 123,
    "account_id": 1,
    "action": "risk_check_passed",
    "message": "账号 1 风控检查通过",
    "created_at": "2025-11-23T10:00:02Z"
  },
  {
    "id": 4,
    "task_id": 123,
    "account_id": 1,
    "action": "execution_success",
    "message": "账号 1 执行成功 (耗时: 2.5s)",
    "extra_data": {
      "status": "success",
      "duration": "2.5s",
      "sent_count": 5,
      "total_targets": 5
    },
    "created_at": "2025-11-23T10:00:04Z"
  },
  {
    "id": 5,
    "task_id": 123,
    "account_id": null,
    "action": "task_completed",
    "message": "任务执行成功，所有 4 个账号都成功了 (总耗时: 10.2s)",
    "extra_data": {
      "account_results": {...},
      "success_count": 4,
      "fail_count": 0,
      "total_accounts": 4
    },
    "created_at": "2025-11-23T10:00:10Z"
  }
]
```

## 前端实现

### 1. 日志查看对话框

位置：`web/app/tasks/page.tsx`

功能：
- 点击任务的"查看日志"按钮打开对话框
- 显示任务的所有执行日志
- 按时间顺序排列
- 显示操作类型、消息和时间

### 2. UI 组件

```tsx
<Dialog open={logsDialogOpen} onOpenChange={setLogsDialogOpen}>
  <DialogContent className="sm:max-w-[800px] max-h-[600px] overflow-y-auto">
    <DialogHeader>
      <DialogTitle>任务日志 - #{viewingTask?.id}</DialogTitle>
      <DialogDescription>
        {viewingTask && `${getTaskTypeText(viewingTask.task_type)} - ${getStatusText(viewingTask.status)}`}
      </DialogDescription>
    </DialogHeader>
    <div className="space-y-2 py-4">
      {logs.map((log: any, index: number) => (
        <div key={index} className="border-l-2 border-muted pl-4 py-2">
          <div className="flex items-center justify-between mb-1">
            <span className="text-sm font-medium">{log.action}</span>
            <span className="text-xs text-muted-foreground">
              {new Date(log.created_at).toLocaleString()}
            </span>
          </div>
          <div className="text-sm text-muted-foreground">{log.message}</div>
        </div>
      ))}
    </div>
  </DialogContent>
</Dialog>
```

### 3. API 调用

```typescript
const loadLogs = async (taskId: string) => {
  try {
    setLoadingLogs(true)
    const response = await taskAPI.getLogs(taskId)
    if (response.data) {
      const logsData = Array.isArray(response.data) ? response.data : []
      setLogs(logsData)
    }
  } catch (error: any) {
    toast.error("加载日志失败")
  } finally {
    setLoadingLogs(false)
  }
}
```

## API 端点

### GET /api/v1/tasks/:id/logs

获取指定任务的所有日志

**响应示例**：
```json
{
  "code": 0,
  "msg": "success",
  "data": [
    {
      "id": 1,
      "task_id": 123,
      "account_id": 1,
      "action": "account_started",
      "message": "开始使用账号 1 执行任务 (1/4)",
      "extra_data": {},
      "created_at": "2025-11-23T10:00:01Z"
    }
  ]
}
```

## 使用方法

1. 在任务列表页面，点击任务行的"查看日志"按钮
2. 弹出日志对话框，显示该任务的所有执行日志
3. 日志按时间顺序排列，最新的在下面
4. 可以看到每个账号的执行情况和详细信息

## 调试建议

如果任务只有一个账号执行了，可以通过日志查看：
1. 是否所有账号都开始执行了（查找 `account_started` 日志）
2. 哪些账号的风控检查失败了（查找 `risk_check_failed` 日志）
3. 哪些账号执行失败了（查找 `execution_failed` 日志）
4. 每个账号的执行结果（查找 `execution_success` 或 `execution_failed` 日志）

## 日期

2025-11-23
