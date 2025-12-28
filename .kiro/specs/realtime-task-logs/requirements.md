# Requirements Document

## Introduction

本功能旨在构建完整的实时通知和日志推送系统，将系统中的各类状态变更和日志从一次性拉取模式升级为实时推送模式。当前系统中任务执行日志只能在任务完成后一次性获取，任务状态、账号状态、代理状态等变更也无法实时通知用户。本功能将通过 WebSocket 实现全面的实时推送能力，包括：
- 任务日志实时推送
- 任务状态变更实时推送
- 账号状态变更实时推送
- 代理状态变更实时推送
- 系统告警实时推送
- 实时统计数据推送

## Glossary

- **Task_Log_Service**: 任务日志服务，负责日志的创建、存储、查询和实时推送
- **WebSocket_Hub**: WebSocket 连接管理中心，负责管理用户连接和消息分发
- **Log_Entry**: 单条日志记录，包含时间戳、级别、消息内容等信息
- **Log_Stream**: 实时日志流，通过 WebSocket 推送给订阅的客户端
- **Log_Level**: 日志级别，包括 info、warn、error、debug
- **Notification_Service**: 现有的通知服务，将被扩展以支持日志推送和各类状态变更推送
- **Task_Scheduler**: 任务调度器，负责任务执行和状态管理
- **Account_Service**: 账号服务，负责账号管理和状态变更
- **Proxy_Service**: 代理服务，负责代理管理和状态监控

## Requirements

### Requirement 1: 实时日志推送

**User Story:** As a user, I want to see task execution logs in real-time, so that I can monitor task progress and quickly identify issues.

#### Acceptance Criteria

1. WHEN a task starts executing, THE Task_Log_Service SHALL establish a log stream for that task
2. WHEN a new log entry is created for a running task, THE Task_Log_Service SHALL push the log entry to all subscribed clients within 500ms
3. WHEN a user opens the task log dialog, THE System SHALL automatically subscribe to the log stream for that task
4. WHEN a user closes the task log dialog, THE System SHALL unsubscribe from the log stream
5. WHEN a task completes or fails, THE Task_Log_Service SHALL send a final log entry indicating completion status
6. IF the WebSocket connection is lost, THEN THE System SHALL attempt to reconnect and resume the log stream

### Requirement 2: 日志订阅管理

**User Story:** As a user, I want to subscribe to specific task logs, so that I only receive relevant log updates.

#### Acceptance Criteria

1. WHEN a client sends a subscribe message with task_id, THE WebSocket_Hub SHALL add the client to the task's subscriber list
2. WHEN a client sends an unsubscribe message, THE WebSocket_Hub SHALL remove the client from the task's subscriber list
3. WHEN a client disconnects, THE WebSocket_Hub SHALL automatically remove the client from all subscriber lists
4. THE System SHALL support subscribing to multiple task logs simultaneously
5. WHEN subscribing to a task log, THE System SHALL return the most recent 50 log entries as initial data

### Requirement 3: 历史日志查询

**User Story:** As a user, I want to query historical task logs with filters, so that I can analyze past task executions.

#### Acceptance Criteria

1. THE Task_Log_Service SHALL support pagination for log queries with configurable page size (default 50, max 200)
2. WHEN querying logs, THE System SHALL support filtering by time range (start_time, end_time)
3. WHEN querying logs, THE System SHALL support filtering by log level (info, warn, error, debug)
4. WHEN querying logs, THE System SHALL support filtering by account_id
5. THE System SHALL return logs in chronological order (oldest first) by default, with option for reverse order
6. WHEN a task is deleted, THE System SHALL retain logs for a configurable retention period (default 30 days)

### Requirement 4: 日志数据结构增强

**User Story:** As a developer, I want structured log entries, so that I can easily parse and display log information.

#### Acceptance Criteria

1. THE Log_Entry SHALL contain: id, task_id, account_id (optional), level, action, message, extra_data, created_at
2. THE Log_Entry level field SHALL be one of: info, warn, error, debug
3. THE Log_Entry extra_data field SHALL support JSON format for structured metadata
4. WHEN creating a log entry, THE Task_Log_Service SHALL validate the log level value
5. THE System SHALL generate unique sequential IDs for log entries within each task

### Requirement 5: 前端日志展示

**User Story:** As a user, I want a clear and responsive log display interface, so that I can easily read and understand task logs.

#### Acceptance Criteria

1. WHEN viewing task logs, THE UI SHALL display logs in a scrollable container with auto-scroll to latest
2. THE UI SHALL color-code log entries based on level (info=blue, warn=yellow, error=red, debug=gray)
3. WHEN new logs arrive via WebSocket, THE UI SHALL append them to the display without page refresh
4. THE UI SHALL show a loading indicator while fetching initial logs
5. THE UI SHALL display a connection status indicator for the WebSocket connection
6. WHEN the log stream is disconnected, THE UI SHALL show a reconnection button

### Requirement 6: 日志清理和保留

**User Story:** As a system administrator, I want automatic log cleanup, so that the database doesn't grow indefinitely.

#### Acceptance Criteria

1. THE System SHALL automatically delete logs older than the retention period (configurable, default 30 days)
2. THE System SHALL run log cleanup as a scheduled job (daily at 3:00 AM)
3. WHEN a task is manually deleted, THE System SHALL delete associated logs immediately
4. THE System SHALL log the number of deleted log entries after each cleanup operation
5. IF log cleanup fails, THEN THE System SHALL retry up to 3 times with exponential backoff

### Requirement 7: 性能和可靠性

**User Story:** As a user, I want the log system to be fast and reliable, so that I don't miss important information.

#### Acceptance Criteria

1. THE Task_Log_Service SHALL handle at least 1000 log entries per second per task
2. THE WebSocket_Hub SHALL support at least 100 concurrent connections per user
3. WHEN the system is under high load, THE Task_Log_Service SHALL queue log entries rather than dropping them
4. THE System SHALL persist log entries to database before sending via WebSocket
5. IF database write fails, THEN THE System SHALL retry up to 3 times before marking the log as failed

### Requirement 8: 任务状态变更实时推送

**User Story:** As a user, I want to receive real-time notifications when task status changes, so that I can monitor task execution without refreshing the page.

#### Acceptance Criteria

1. WHEN a task status changes (pending → queued → running → completed/failed/cancelled), THE System SHALL push a status change notification to the task owner
2. THE notification SHALL contain: task_id, old_status, new_status, timestamp, and relevant metadata
3. WHEN a task starts executing, THE System SHALL push a notification with estimated completion time if available
4. WHEN a task completes, THE System SHALL push a notification with execution summary (success_count, fail_count, duration)
5. WHEN a task fails, THE System SHALL push a notification with error details and affected accounts

### Requirement 9: 账号状态变更实时推送

**User Story:** As a user, I want to receive real-time notifications when account status changes, so that I can quickly respond to account issues.

#### Acceptance Criteria

1. WHEN an account status changes (normal → restricted/frozen/dead/cooling), THE System SHALL push a status change notification to the account owner
2. THE notification SHALL contain: account_id, phone, old_status, new_status, reason (if available), timestamp
3. WHEN an account encounters an error during task execution, THE System SHALL push an error notification immediately
4. WHEN an account connection status changes (connected → disconnected → error), THE System SHALL push a connection status notification
5. THE System SHALL support configurable notification preferences (which status changes to receive)

### Requirement 10: 代理状态变更实时推送

**User Story:** As a user, I want to receive real-time notifications when proxy status changes, so that I can ensure my accounts have working proxies.

#### Acceptance Criteria

1. WHEN a proxy status changes (active → inactive/error), THE System SHALL push a status change notification to the proxy owner
2. THE notification SHALL contain: proxy_id, proxy_name, old_status, new_status, affected_accounts_count
3. WHEN a proxy health check fails, THE System SHALL push a notification with failure details
4. WHEN multiple accounts are affected by a proxy issue, THE System SHALL aggregate notifications to avoid spam

### Requirement 11: 系统告警实时推送

**User Story:** As a user, I want to receive real-time system alerts, so that I can be aware of system-wide issues affecting my operations.

#### Acceptance Criteria

1. WHEN a system-level issue occurs (rate limiting, maintenance, service degradation), THE System SHALL push an alert to affected users
2. THE alert SHALL contain: alert_level (info/warning/error/critical), message, affected_services, timestamp
3. WHEN scheduled maintenance is planned, THE System SHALL push advance notifications (24h, 1h, 15min before)
4. THE System SHALL support alert acknowledgment to prevent repeated notifications

### Requirement 12: 实时统计数据推送

**User Story:** As a user, I want to receive real-time statistics updates, so that I can monitor my overall system performance.

#### Acceptance Criteria

1. WHEN task queue status changes significantly, THE System SHALL push queue statistics to subscribed users
2. THE statistics SHALL include: pending_tasks, running_tasks, completed_today, failed_today, success_rate
3. WHEN account utilization changes, THE System SHALL push account statistics (active_accounts, busy_accounts, idle_accounts)
4. THE System SHALL support configurable push intervals (minimum 5 seconds) to prevent excessive updates
5. THE System SHALL only push statistics when values have changed to reduce unnecessary traffic
