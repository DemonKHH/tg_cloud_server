# 账号信息同步问题修复

## 问题描述

在使用两个不同的账号执行同一个任务时，发现：
1. 两个账号的 Telegram 用户信息都被更新为同一个用户的信息
2. 任务实际上只有一个账号执行了

## 问题根源

在 `internal/telegram/connection_pool.go` 的 `updateAccountInfoFromTelegram` 函数中存在严重的并发问题：

### 原有实现的问题

```go
func (cp *ConnectionPool) updateAccountInfoFromTelegram(accountID string, conn *ManagedConnection) {
    // ...
    err = conn.client.Run(context.Background(), func(ctx context.Context) error {
        api := conn.client.API()
        // 获取用户信息并更新数据库
        // ...
    })
}
```

**问题点**：
1. `maintainConnection` 已经调用了 `conn.client.Run()` 来保持连接
2. 在 `Run()` 的回调中，又启动了一个 goroutine 调用 `updateAccountInfoFromTelegram`
3. `updateAccountInfoFromTelegram` 又调用了 `conn.client.Run()`，导致同一个 client 被多次 Run
4. 这会导致连接状态混乱，可能导致多个账号共享同一个连接上下文

## 修复方案

### 1. 修改 `updateAccountInfoFromTelegram` 函数签名

添加 `ctx context.Context` 参数，使用从 `maintainConnection` 传入的上下文：

```go
func (cp *ConnectionPool) updateAccountInfoFromTelegram(accountID string, conn *ManagedConnection, ctx context.Context)
```

### 2. 直接使用已建立的连接

不再调用 `conn.client.Run()`，而是直接使用 `conn.client.API()`：

```go
// 直接使用已建立的连接和 API 客户端，不再调用 Run()
api := conn.client.API()

// 获取当前用户信息
users, err := api.UsersGetUsers(ctx, []tg.InputUserClass{&tg.InputUserSelf{}})
```

### 3. 添加账号 ID 验证

在更新数据库前验证账号 ID 是否匹配，防止更新错误的账号：

```go
// 验证账号 ID 匹配，防止更新错误的账号
if account.ID != accountIDNum {
    cp.logger.Error("Account ID mismatch! This should never happen!",
        zap.String("expected_account_id", accountID),
        zap.Uint64("actual_account_id", account.ID))
    return
}
```

### 4. 增强日志记录

添加更详细的日志，便于追踪问题：

```go
// 记录更新前的信息用于调试
cp.logger.Info("Updating account info",
    zap.String("account_id", accountID),
    zap.String("phone", account.Phone),
    zap.Any("new_tg_user_id", info.TgUserID),
    zap.Any("new_username", info.Username),
    zap.Any("new_first_name", info.FirstName))
```

### 5. 更新调用方式

在 `maintainConnection` 中传递 context：

```go
// 连接成功后，获取并更新账号信息（在同一个 Run 上下文中）
go cp.updateAccountInfoFromTelegram(accountID, conn, ctx)
```

## 修复效果

1. ✅ 每个账号使用独立的连接上下文
2. ✅ 避免了多次调用 `Run()` 导致的冲突
3. ✅ 确保获取的用户信息正确更新到对应的账号
4. ✅ 增加了账号 ID 验证，防止数据错乱
5. ✅ 增强了日志记录，便于问题追踪

## 测试建议

1. 使用两个不同的账号同时执行任务
2. 检查日志，确认每个账号都使用了正确的 account_id
3. 验证数据库中的账号信息是否正确更新
4. 确认两个任务都成功执行

## 相关文件

- `internal/telegram/connection_pool.go` - 连接池管理
- `internal/scheduler/task_scheduler.go` - 任务调度器
- `web/app/accounts/page.tsx` - 前端账号页面（已添加 Telegram 信息列）

## 日期

2025-11-23
