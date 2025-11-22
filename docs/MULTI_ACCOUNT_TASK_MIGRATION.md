# 多账号任务功能 - 数据库迁移

## 概述

添加对单个任务使用多个账号执行的支持。

## 数据库变更

### 1. 添加 `account_ids` 字段到 `tasks` 表

```sql
ALTER TABLE tasks ADD COLUMN account_ids TEXT DEFAULT '' COMMENT '多账号ID列表（逗号分隔）';
```

### 说明

- `account_id`: 保留原有字段，作为主执行账号（用于兼容和索引）
- `account_ids`: 新增字段，存储多个账号ID的逗号分隔字符串（如 "1,2,3"）
- 如果只有一个账号，`account_ids` 为空字符串，只使用 `account_id`
- 如果有多个账号，`account_id` 为第一个账号ID，`account_ids` 包含所有账号ID

## 功能说明

### 前端变更

- 选择多个账号创建任务时，创建一个任务而不是多个任务
- 任务详情页显示所有关联的账号
- 任务执行结果显示每个账号的执行情况

### 后端变更

1. **任务模型** (`internal/models/task.go`)
   - 添加 `AccountIDs` 字段
   - 添加 `GetAccountIDList()` 方法获取账号列表
   - 添加 `SetAccountIDList()` 方法设置账号列表

2. **创建任务请求** (`internal/models/task.go`)
   - 添加 `AccountIDs []uint64` 字段支持多账号
   - 保留 `AccountID uint64` 字段向后兼容
   - 添加 `GetAccountIDs()` 方法兼容两种方式
   - 添加 `Validate()` 方法验证请求

3. **任务服务** (`internal/services/task_service.go`)
   - 修改 `CreateTask` 方法支持多账号验证
   - 验证所有账号都属于用户且可用

4. **任务调度器** (`internal/scheduler/task_scheduler.go`)
   - 修改 `executeTask` 方法支持多账号轮换执行
   - 依次使用每个账号执行任务
   - 记录每个账号的执行结果
   - 任务结果包含：
     - `account_results`: 每个账号的执行结果
     - `success_count`: 成功的账号数
     - `fail_count`: 失败的账号数
     - `total_accounts`: 总账号数

## 执行逻辑

1. 用户选择多个账号创建任务
2. 系统创建一个任务，包含所有账号ID
3. 任务调度器获取任务后：
   - 依次使用每个账号执行任务
   - 对每个账号进行风控检查
   - 记录每个账号的执行结果
4. 所有账号执行完成后：
   - 如果全部失败，任务状态为失败
   - 如果部分成功，任务状态为成功（带警告）
   - 如果全部成功，任务状态为成功

## 优势

1. **简化管理**: 一个任务对应一个业务目标，而不是多个重复任务
2. **统一结果**: 所有账号的执行结果集中在一个任务中
3. **更好的追踪**: 可以清楚地看到哪些账号成功，哪些失败
4. **向后兼容**: 单账号任务仍然正常工作

## 示例

### 创建任务请求

```json
{
  "account_ids": [1, 2, 3],
  "task_type": "private_message",
  "priority": 5,
  "auto_start": true,
  "task_config": {
    "targets": ["@user1", "@user2"],
    "message": "Hello!"
  }
}
```

### 任务执行结果

```json
{
  "account_results": {
    "1": {
      "status": "success",
      "duration": "2.5s"
    },
    "2": {
      "status": "failed",
      "error": "connection timeout",
      "duration": "30s"
    },
    "3": {
      "status": "success",
      "duration": "3.1s"
    }
  },
  "success_count": 2,
  "fail_count": 1,
  "total_accounts": 3
}
```

## 迁移步骤

1. 执行 SQL 迁移添加 `account_ids` 字段
2. 重启后端服务
3. 清除前端缓存
4. 测试创建多账号任务

## 日期

2025-11-23
