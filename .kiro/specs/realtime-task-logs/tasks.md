# Implementation Plan: Real-time Task Logs

## Overview

本实现计划将任务日志系统从一次性拉取模式升级为实时推送模式。核心改造点是在现有的 `TaskScheduler.createTaskLog()` 方法中增加实时推送能力，并扩展 `NotificationService` 支持任务日志订阅。

## Tasks

- [x] 1. 创建 TaskLogService 服务
  - [x] 1.1 定义 TaskLogService 接口和数据结构
    - 在 `internal/services/` 创建 `task_log_service.go`
    - 定义 `TaskLogService` 接口
    - 定义 `TaskLogEntry`, `LogQueryFilter`, `LogQueryResult` 结构
    - 定义 `LogLevel` 常量 (info, warn, error, debug)
    - _Requirements: 4.1, 4.2_

  - [x] 1.2 实现 TaskLogService 核心方法
    - 实现 `CreateLog()` - 创建日志并推送
    - 实现 `BatchCreateLogs()` - 批量创建日志
    - 实现 `QueryLogs()` - 支持分页和过滤的查询
    - 实现 `GetRecentLogs()` - 获取最近日志
    - _Requirements: 1.1, 1.2, 3.1, 3.2, 3.3, 3.4, 3.5_

  - [x] 1.3 实现日志清理功能
    - 实现 `CleanupExpiredLogs()` - 清理过期日志
    - 实现 `DeleteTaskLogs()` - 删除任务相关日志
    - _Requirements: 6.1, 6.3_

- [x] 2. 扩展 NotificationService 支持日志订阅
  - [x] 2.1 添加任务日志订阅管理
    - 在 `WSConnection` 中添加 `taskLogSubscriptions map[uint64]bool`
    - 添加 `TaskLogSubscription` 内存结构管理订阅关系
    - _Requirements: 2.1, 2.4_

  - [x] 2.2 实现订阅/取消订阅方法
    - 实现 `SubscribeTaskLogs(userID, taskID)` 方法
    - 实现 `UnsubscribeTaskLogs(userID, taskID)` 方法
    - 实现 `GetTaskLogSubscribers(taskID)` 方法
    - _Requirements: 2.1, 2.2_

  - [x] 2.3 实现日志推送方法
    - 实现 `PushTaskLog(userID, taskID, log)` 方法
    - 确保只推送给订阅了该任务的用户
    - _Requirements: 1.2_

  - [x] 2.4 处理 WebSocket 消息
    - 在 `handleWSMessage()` 中添加 `subscribe_task_logs` 处理
    - 在 `handleWSMessage()` 中添加 `unsubscribe_task_logs` 处理
    - 订阅时返回最近 50 条日志
    - _Requirements: 2.1, 2.2, 2.5_

  - [x] 2.5 处理连接断开清理
    - 在连接断开时自动清理该用户的所有任务日志订阅
    - _Requirements: 2.3_

- [x] 3. 改造 TaskScheduler 集成实时推送
  - [x] 3.1 注入 TaskLogService 依赖
    - 在 `TaskScheduler` 结构体中添加 `taskLogService` 字段
    - 在 `NewTaskScheduler()` 中注入依赖
    - _Requirements: 1.1_

  - [x] 3.2 改造 createTaskLog 方法
    - 修改 `createTaskLog()` 调用 `TaskLogService.CreateLog()`
    - 确保日志先持久化再推送
    - _Requirements: 1.2, 7.4_

  - [x] 3.3 添加任务完成日志
    - 在任务完成时发送 `task_completed` 日志
    - 在任务失败时发送 `task_failed` 日志
    - _Requirements: 1.5_

- [x] 4. Checkpoint - 后端核心功能完成
  - 确保所有测试通过，ask the user if questions arise.

- [x] 5. 创建日志查询 API
  - [x] 5.1 添加日志查询 Handler
    - 在 `internal/handlers/task_handler.go` 添加 `GetTaskLogs` 方法
    - 支持分页参数: page, limit
    - 支持过滤参数: level, start_time, end_time, account_id, order
    - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

  - [x] 5.2 注册路由
    - 在路由配置中添加 `GET /api/v1/tasks/:id/logs`
    - _Requirements: 3.1_

- [x] 6. 添加日志清理定时任务
  - [x] 6.1 创建日志清理 Cron Job
    - 在 `internal/cron/` 添加日志清理任务
    - 配置每天 3:00 AM 执行
    - 默认保留 30 天日志
    - _Requirements: 6.1, 6.2_

  - [x] 6.2 添加清理失败重试逻辑
    - 失败时重试 3 次，使用指数退避
    - 记录清理结果日志
    - _Requirements: 6.4, 6.5_

- [x] 7. Checkpoint - 后端功能完成
  - 确保所有测试通过，ask the user if questions arise.

- [x] 8. 前端 WebSocket 客户端
  - [x] 8.1 创建 WebSocket Hook
    - 在 `web/hooks/` 创建 `useTaskLogs.ts`
    - 实现连接管理、自动重连
    - 实现订阅/取消订阅方法
    - _Requirements: 1.3, 1.4, 1.6_

  - [x] 8.2 实现日志状态管理
    - 管理日志列表状态
    - 处理新日志追加
    - 处理初始日志加载
    - _Requirements: 5.3, 5.4_

- [x] 9. 前端日志展示组件
  - [x] 9.1 创建 TaskLogDialog 组件
    - 在 `web/components/` 创建 `TaskLogDialog.tsx`
    - 实现可滚动日志容器
    - 实现自动滚动到最新
    - _Requirements: 5.1_

  - [x] 9.2 实现日志颜色编码
    - info = blue, warn = yellow, error = red, debug = gray
    - _Requirements: 5.2_

  - [x] 9.3 添加连接状态指示器
    - 显示 WebSocket 连接状态
    - 断开时显示重连按钮
    - _Requirements: 5.5, 5.6_

  - [x] 9.4 添加加载状态
    - 获取初始日志时显示加载指示器
    - _Requirements: 5.4_

- [ ] 10. Final Checkpoint - 全部功能完成
  - 确保所有测试通过，ask the user if questions arise.

## Notes

- 核心改造点是 `TaskScheduler.createTaskLog()` 方法，需要在写入数据库后立即推送
- 复用现有的 `NotificationService` 和 `WSHub` 基础设施
- 日志先持久化再推送，确保不丢失数据
- 前端使用 React Hook 管理 WebSocket 连接和日志状态
