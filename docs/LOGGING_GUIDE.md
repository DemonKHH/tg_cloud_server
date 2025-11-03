# 📝 日志系统使用指南

## 🎯 概述

本项目实现了一个完整的分级日志系统，支持文件轮转、自动清理、分类存储等功能，便于问题排查和系统监控。

## 🏗️ 系统架构

### 📂 日志文件结构
```
logs/
├── app.log         # 主日志文件（所有级别）
├── error.log       # 错误日志（ERROR级别）
├── warn.log        # 警告日志（WARN级别）
├── info.log        # 信息日志（INFO级别）
├── debug.log       # 调试日志（DEBUG级别）
├── task.log        # 任务专用日志
└── api.log         # API专用日志
```

### 🎛️ 日志级别
- **DEBUG**: 详细的调试信息
- **INFO**: 一般信息，如操作成功
- **WARN**: 警告信息，可能的问题
- **ERROR**: 错误信息，需要关注

## ⚙️ 配置说明

### 📋 配置文件（config.yaml）
```yaml
logging:
  level: "info"                    # 日志级别: debug, info, warn, error
  format: "json"                   # 日志格式: json, console  
  output: "file"                   # 输出方式: stdout, file
  filename: "logs/app.log"         # 主日志文件
  max_size: 100                    # 单个文件最大大小(MB)
  max_backups: 7                   # 保留的历史文件数量
  max_age: 30                      # 文件最大保存天数
  compress: true                   # 是否压缩历史文件
  files:
    error_log: "logs/error.log"    # 错误日志文件
    warn_log: "logs/warn.log"      # 警告日志文件
    info_log: "logs/info.log"      # 信息日志文件
    debug_log: "logs/debug.log"    # 调试日志文件
    task_log: "logs/task.log"      # 任务专用日志文件
    api_log: "logs/api.log"        # API专用日志文件
```

## 🚀 使用方法

### 1. 基本日志记录
```go
import (
    "go.uber.org/zap"
    "tg_cloud_server/internal/common/logger"
)

// 获取主日志器
log := logger.Get()

// 记录不同级别的日志
log.Debug("调试信息", zap.String("component", "auth"))
log.Info("操作成功", zap.String("action", "login"), zap.String("user", "admin"))
log.Warn("警告信息", zap.String("issue", "rate_limit"))
log.Error("错误信息", zap.Error(err), zap.String("operation", "database"))
```

### 2. 专用日志器
```go
// 使用任务专用日志器
logger.LogTask(zapcore.InfoLevel, "任务创建成功",
    zap.Uint64("task_id", taskID),
    zap.String("task_type", "private_message"),
    zap.Uint64("account_id", accountID))

// 使用API专用日志器  
logger.LogAPI(zapcore.InfoLevel, "API请求处理",
    zap.String("method", "POST"),
    zap.String("path", "/api/v1/tasks"),
    zap.Int("status_code", 200),
    zap.Duration("latency", time.Since(start)))
```

### 3. 分级日志器
```go
// 直接使用特定级别的日志器
errorLogger := logger.GetError()
errorLogger.Error("数据库连接失败", zap.String("database", "mysql"))

warnLogger := logger.GetWarn()  
warnLogger.Warn("性能警告", zap.Duration("response_time", 2*time.Second))

infoLogger := logger.GetInfo()
infoLogger.Info("系统状态", zap.String("status", "healthy"))

debugLogger := logger.GetDebug()
debugLogger.Debug("内存使用情况", zap.Any("stats", memStats))
```

### 4. 带预设字段的日志
```go
// 任务日志带预设字段
logger.LogTaskWithFields(zapcore.InfoLevel, "任务状态更新", 
    taskID, taskType,
    zap.String("status", "running"),
    zap.Time("updated_at", time.Now()))

// API日志带预设字段
logger.LogAPIWithFields(zapcore.ErrorLevel, "API错误", 
    "POST", "/api/v1/tasks", 500,
    zap.Error(err),
    zap.String("user_id", userID))
```

## 🛠️ 日志管理

### 📊 日志统计
```go
logManager := logger.NewLogManager(config)

// 获取日志文件信息
files := logManager.GetLogFiles()
for _, file := range files {
    fmt.Printf("文件: %s, 类型: %s, 大小: %d字节\n", 
        file.Path, file.Type, file.Size)
}

// 获取统计信息
stats := logManager.GetLogStats()
fmt.Printf("总文件数: %d, 总大小: %d字节\n", 
    stats.TotalFiles, stats.TotalSize)
```

### 🧹 日志清理
```go
logManager := logger.NewLogManager(config)

// 手动清理过期日志
if err := logManager.CleanupOldLogs(); err != nil {
    log.Error("日志清理失败", zap.Error(err))
}

// 手动轮转日志
if err := logManager.RotateLogs(); err != nil {
    log.Error("日志轮转失败", zap.Error(err))
}
```

## 🔄 自动化功能

### ⏰ 定时清理
系统会自动在每天凌晨2点清理过期的日志文件，根据配置的保留天数。

### 📦 文件轮转
当日志文件达到设置的最大大小时，会自动创建新文件，并可选择压缩旧文件。

### 🎯 API中间件
系统自动记录所有API请求的详细信息：
- 请求方法和路径
- 响应状态码和处理时间
- 用户信息（如果已认证）
- 错误信息（如果有）

## 📋 日志格式示例

### JSON格式（默认）
```json
{
  "level": "info",
  "timestamp": "2024-01-15T10:30:45+08:00",
  "caller": "services/task_service.go:95",
  "msg": "任务创建成功",
  "task_id": 12345,
  "task_type": "private_message",
  "account_id": 67890,
  "user_id": 1
}
```

### Console格式（开发环境）
```
2024-01-15T10:30:45+08:00  INFO  services/task_service.go:95  任务创建成功  
{"task_id": 12345, "task_type": "private_message", "account_id": 67890}
```

## 🔍 日志查看

### 实时查看
```bash
# Windows PowerShell
Get-Content logs/app.log -Tail 50 -Wait

# Linux/macOS
tail -f logs/app.log
```

### 按类型查看
```bash
# 查看错误日志
Get-Content logs/error.log -Tail 20

# 查看任务日志
Get-Content logs/task.log -Tail 30

# 查看API日志
Get-Content logs/api.log -Tail 40
```

### 日志分析
```bash
# 统计错误数量
Get-Content logs/error.log | Measure-Object -Line

# 查找特定任务的日志
Get-Content logs/task.log | Select-String "task_id.*12345"

# 查找API错误
Get-Content logs/api.log | Select-String "status_code.*[45][0-9][0-9]"
```

## 🎮 测试和验证

### 运行日志测试
```bash
# 基本功能测试
go run scripts/test_logging.go

# 包含清理功能测试  
go run scripts/test_logging.go --test-cleanup
```

### 测试输出示例
```
🚀 开始测试日志系统...
✅ 日志系统初始化成功
📝 测试各种日志级别...
🔧 测试专用日志器...
📊 测试各级别专用日志器...
🛠️ 测试日志管理器功能...
📁 发现日志文件 7 个
📈 日志统计: 文件总数=7, 总大小=35.65KB
🔄 测试日志轮转功能...
💾 日志缓冲区已同步
✅ 验证日志文件创建...
🎉 日志系统测试完成!
```

## 🚨 故障排查

### 常见问题

1. **日志文件未创建**
   - 检查logs目录权限
   - 确认配置文件路径正确
   - 验证磁盘空间充足

2. **日志轮转不工作**
   - 检查max_size配置
   - 确认文件写入权限
   - 查看系统错误日志

3. **性能问题**
   - 降低日志级别（info->warn->error）
   - 增加缓冲区大小
   - 使用异步写入

4. **磁盘空间不足**
   - 减少max_age和max_backups
   - 启用压缩功能
   - 定期清理日志

### 调试步骤

1. 运行测试脚本验证功能
2. 检查配置文件语法
3. 查看系统日志获取错误信息
4. 验证文件权限和磁盘空间
5. 重启应用重新初始化日志系统

## 📈 性能优化

### 建议配置

**生产环境**:
```yaml
logging:
  level: "info"          # 避免debug级别
  format: "json"         # 便于日志分析
  output: "file"         # 文件输出
  max_size: 100          # 100MB轮转
  max_backups: 7         # 保留7个备份
  max_age: 30            # 30天清理
  compress: true         # 启用压缩
```

**开发环境**:
```yaml
logging:
  level: "debug"         # 详细调试信息
  format: "console"      # 易读格式
  output: "stdout"       # 控制台输出
  max_size: 50           # 小文件便于查看
  max_backups: 3         # 少量备份
  max_age: 7             # 快速清理
  compress: false        # 不压缩便于查看
```

## 🔗 相关文件

- `internal/common/logger/logger.go` - 核心日志系统
- `internal/common/logger/log_manager.go` - 日志管理工具
- `internal/common/middleware/api_logger.go` - API日志中间件
- `internal/cron/cron.go` - 定时清理任务
- `configs/config.yaml` - 日志配置
- `scripts/test_logging.go` - 测试脚本

现在您的日志系统已经完全配置好了，所有日志都会写入到文件中，便于问题排查和系统监控！🎉
