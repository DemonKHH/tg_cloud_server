# 多账号任务执行问题修复

## 问题描述

使用4个账号执行私信任务时，只有第一个账号实际执行了，然后任务就显示完成。

## 问题根源

在 `executeTask` 方法中，每个账号都使用同一个 `task` 对象执行任务。任务执行器（如 `PrivateMessageTask`）会直接修改 `task.Result`，导致：

1. **结果被覆盖**: 每个账号的执行结果会覆盖前一个账号的结果
2. **account_results 丢失**: 任务执行器不知道 `account_results` 的存在，会覆盖整个 `task.Result`

### 问题代码

```go
// 初始化结果记录
task.Result["account_results"] = make(map[string]interface{})

// 执行任务
for i, accountID := range accountIDs {
    taskExecutor := createTaskExecutor(task)
    err := connectionPool.ExecuteTask(accountIDStr, taskExecutor)
    
    // 任务执行器会修改 task.Result，覆盖 account_results
    // 导致下一次循环时 account_results 丢失
}
```

## 解决方案

在每个账号执行完成后：
1. 立即保存该账号的执行结果
2. 从 `task.Result` 中提取任务执行器写入的数据
3. 恢复 `account_results` 到 `task.Result` 中

### 修复代码

```go
// 执行任务
accountStartTime := time.Now()
err = ts.connectionPool.ExecuteTask(accountIDStr, taskExecutor)
accountDuration := time.Since(accountStartTime)

// 保存该账号的执行结果（从 task.Result 中提取）
accountResult := make(map[string]interface{})
accountResult["duration"] = accountDuration.String()

// 复制任务执行器写入的结果
for key, value := range task.Result {
    if key != "account_results" && key != "success_count" && 
       key != "fail_count" && key != "total_accounts" {
        accountResult[key] = value
    }
}

if err != nil {
    accountResult["status"] = "failed"
    accountResult["error"] = err.Error()
    failCount++
} else {
    accountResult["status"] = "success"
    successCount++
}

// 保存该账号的结果
accountResults[accountIDStr] = accountResult

// 恢复 account_results（防止被任务执行器覆盖）
task.Result["account_results"] = accountResults
```

## 执行流程

1. 初始化 `task.Result["account_results"]`
2. 循环每个账号：
   - 创建任务执行器
   - 执行任务（任务执行器会修改 `task.Result`）
   - 提取执行结果
   - 保存到 `accountResults[accountID]`
   - 恢复 `task.Result["account_results"]`
3. 汇总所有账号的结果

## 结果示例

```json
{
  "account_results": {
    "1": {
      "status": "success",
      "duration": "2.5s",
      "sent_count": 5,
      "failed_count": 0,
      "total_targets": 5,
      "success_rate": 1.0
    },
    "2": {
      "status": "success",
      "duration": "3.1s",
      "sent_count": 5,
      "failed_count": 0,
      "total_targets": 5,
      "success_rate": 1.0
    },
    "3": {
      "status": "failed",
      "duration": "30s",
      "error": "connection timeout"
    },
    "4": {
      "status": "success",
      "duration": "2.8s",
      "sent_count": 5,
      "failed_count": 0,
      "total_targets": 5,
      "success_rate": 1.0
    }
  },
  "success_count": 3,
  "fail_count": 1,
  "total_accounts": 4
}
```

## 测试建议

1. 创建多账号私信任务
2. 检查日志，确认每个账号都执行了
3. 查看任务结果，验证每个账号的执行情况
4. 测试部分账号失败的情况

## 日期

2025-11-23
