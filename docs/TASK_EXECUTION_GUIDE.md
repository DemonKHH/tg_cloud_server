# 📋 任务执行系统使用指南

## 🎯 概述

任务执行系统现在已经完全集成，包括：
- ✅ **任务调度器 (TaskScheduler)** - 管理任务队列和执行
- ✅ **连接池 (ConnectionPool)** - 管理Telegram连接
- ✅ **任务服务 (TaskService)** - 处理任务CRUD操作
- ✅ **自动提交** - 创建任务后自动提交给调度器执行
- ✅ **日志记录** - 完整的任务执行日志

## 🚀 快速开始

### 1. 启动服务器

```bash
# 启动主服务器（包含任务调度器）
go run cmd/web-api/main.go
```

服务器启动时会自动：
- 初始化连接池
- 启动任务调度器
- 连接任务服务和调度器
- 加载所有待处理任务

### 2. 创建任务

通过API创建任务，系统会自动提交给调度器执行：

```bash
# 创建账号检查任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "task_type": "check",
    "account_id": 1,
    "priority": 5,
    "config": {
      "timeout_seconds": 30
    }
  }'

# 创建私信任务
curl -X POST http://localhost:8080/api/v1/tasks \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_TOKEN" \
  -d '{
    "task_type": "private_message",
    "account_id": 1,
    "priority": 3,
    "config": {
      "targets": ["@username1", "@username2"],
      "message": "Hello, this is a test message",
      "timeout_seconds": 60
    }
  }'
```

### 3. 监控任务执行

```bash
# 查看任务列表
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/v1/tasks?page=1&limit=10"

# 查看特定任务详情
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/v1/tasks/123"

# 查看任务统计
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/v1/tasks/stats"
```

## 🧪 自动化测试

我们提供了完整的测试脚本：

```bash
# 运行任务执行测试
go run scripts/test_task_execution.go
```

测试脚本会：
1. 检查服务器状态
2. 模拟用户登录
3. 创建测试账号
4. 创建多种类型的测试任务
5. 监控任务执行状态
6. 显示执行统计
7. 检查日志文件

## 📊 支持的任务类型

| 任务类型 | 描述 | 配置参数 |
|---------|------|----------|
| `check` | 账号检查 | `timeout_seconds` |
| `private_message` | 私信发送 | `targets`, `message`, `timeout_seconds` |
| `broadcast` | 群发消息 | `groups`, `message`, `timeout_seconds` |
| `verify_code` | 验证码接收 | `phone_number`, `timeout_seconds` |
| `group_chat` | AI炒群 | `groups`, `ai_config`, `timeout_seconds` |

## 📈 任务状态流转

```
pending → queued → running → completed
                    ↓
                  failed
                    ↓
                cancelled
```

- **pending**: 任务已创建，等待提交
- **queued**: 已提交给调度器，排队中
- **running**: 正在执行
- **completed**: 执行完成
- **failed**: 执行失败
- **cancelled**: 已取消

## 🔍 日志监控

### 实时查看日志

```powershell
# 查看任务执行日志
Get-Content logs/task.log -Tail 20 -Wait

# 查看API请求日志
Get-Content logs/api.log -Tail 20 -Wait

# 查看错误日志
Get-Content logs/error.log -Tail 20 -Wait

# 查看主日志
Get-Content logs/app.log -Tail 20 -Wait
```

### 日志内容示例

```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:45+08:00",
  "caller": "services/task_service.go:125",
  "msg": "Task submitted to scheduler",
  "task_id": 123,
  "task_type": "private_message"
}
```

## 🛠️ 故障排查

### 常见问题

1. **任务创建后不执行**
   - 检查服务器启动日志，确认任务调度器已启动
   - 查看 `logs/task.log` 是否有任务提交记录
   - 检查账号状态是否正常

2. **任务执行失败**
   - 查看错误日志 `logs/error.log`
   - 检查Telegram连接配置
   - 验证任务配置参数是否正确

3. **任务长时间pending**
   - 检查账号是否有效
   - 查看连接池状态
   - 确认任务调度器运行正常

### 调试命令

```bash
# 检查服务器健康状态
curl http://localhost:8080/health

# 查看系统信息
curl http://localhost:8080/info

# 获取账号队列信息
curl -H "Authorization: Bearer YOUR_TOKEN" \
  "http://localhost:8080/api/v1/tasks/queue/account/1"
```

## 📝 API端点

### 任务管理
- `POST /api/v1/tasks` - 创建任务
- `GET /api/v1/tasks` - 获取任务列表
- `GET /api/v1/tasks/{id}` - 获取任务详情
- `PUT /api/v1/tasks/{id}` - 更新任务
- `DELETE /api/v1/tasks/{id}` - 取消任务

### 任务监控
- `GET /api/v1/tasks/stats` - 获取任务统计
- `GET /api/v1/tasks/{id}/logs` - 获取任务日志
- `GET /api/v1/tasks/queue/account/{id}` - 获取账号队列信息

### 批量操作
- `POST /api/v1/tasks/batch/cancel` - 批量取消任务
- `POST /api/v1/tasks/batch/retry` - 批量重试任务

## 🎯 最佳实践

### 任务设计
1. **合理设置优先级** - 重要任务设置高优先级
2. **配置超时时间** - 避免任务长时间占用资源
3. **批量操作** - 对大量相似任务使用批量接口

### 性能优化
1. **账号负载均衡** - 将任务分散到多个账号
2. **监控队列长度** - 避免单个账号队列过长
3. **定期清理** - 清理过期的已完成任务

### 错误处理
1. **设置重试策略** - 对暂时性错误进行重试
2. **监控失败率** - 及时发现系统问题
3. **日志分析** - 通过日志定位问题根因

## 🔗 相关文件

- `internal/services/task_service.go` - 任务服务
- `internal/scheduler/task_scheduler.go` - 任务调度器
- `internal/telegram/connection_pool.go` - 连接池
- `internal/telegram/task_executors.go` - 任务执行器
- `scripts/test_task_execution.go` - 测试脚本
- `docs/LOGGING_GUIDE.md` - 日志系统指南

现在您的任务执行系统已经完全可用！🎉

创建任务后会自动提交给调度器执行，所有过程都有完整的日志记录。
