# API 响应格式统一更新指南

## 已完成的工作

1. **统一响应格式结构** (`internal/common/response/response.go`)
   - 新的响应格式：`{code: 0, msg: "xx", data: xx}`
   - code: 0 表示成功，非0表示失败
   - 定义了统一的错误码体系

2. **路由方法统一**
   - 所有接口统一为 GET 和 POST 方法
   - PUT 改为 POST + `/update` 后缀
   - DELETE 改为 POST + `/delete` 后缀

3. **Handler 更新**
   - ✅ `auth_handler.go` - 已完成
   - ✅ `task_handler.go` - 已完成
   - ⏳ 其他 handler 文件需要更新

## 待更新的 Handler 文件

以下文件需要按照相同模式更新响应格式：

- `internal/handlers/account_handler.go`
- `internal/handlers/proxy_handler.go`
- `internal/handlers/template_handler.go`
- `internal/handlers/file_handler.go`
- `internal/handlers/module_handler.go`
- `internal/handlers/ai_handler.go`
- `internal/handlers/stats_handler.go`

## 更新步骤

### 1. 更新 Import

在每个 handler 文件中添加：
```go
import (
    // ... 其他导入
    "tg_cloud_server/internal/common/response"
)
```

删除：
```go
import "net/http"  // 如果只用于响应状态码，可以删除
```

### 2. 替换响应格式

#### 成功响应
```go
// 旧格式
c.JSON(http.StatusOK, data)
c.JSON(http.StatusCreated, gin.H{"message": "...", "data": data})

// 新格式
response.Success(c, data)
response.SuccessWithMessage(c, "操作成功", data)
```

#### 错误响应
```go
// 未授权
c.JSON(http.StatusUnauthorized, gin.H{"error": "..."})
→ response.Unauthorized(c, "错误信息")

// 参数错误
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
→ response.InvalidParam(c, err.Error())

// 资源不存在
c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
→ response.NotFound(c, "资源不存在")
// 或特定资源
→ response.AccountNotFound(c)
→ response.TaskNotFound(c)
→ response.ProxyNotFound(c)

// 内部错误
c.JSON(http.StatusInternalServerError, gin.H{"error": "..."})
→ response.InternalError(c, "错误信息")

// 冲突
c.JSON(http.StatusConflict, gin.H{"error": "..."})
→ response.Conflict(c, "错误信息")
```

### 3. 分页响应

```go
// 旧格式
c.JSON(http.StatusOK, gin.H{
    "data": gin.H{
        "items": items,
        "pagination": gin.H{...}
    }
})

// 新格式
response.Paginated(c, items, page, limit, total)
```

## 错误码定义

所有错误码定义在 `internal/common/response/response.go`：

- `CodeSuccess = 0` - 成功
- `CodeInvalidParam = 1001` - 参数错误
- `CodeUnauthorized = 1002` - 未授权
- `CodeForbidden = 1003` - 禁止访问
- `CodeNotFound = 1004` - 资源不存在
- `CodeInternalError = 1005` - 服务器内部错误
- `CodeRateLimit = 1006` - 请求过于频繁
- `CodeConflict = 1007` - 资源冲突
- `CodeUserExists = 2001` - 用户已存在
- `CodeInvalidCredentials = 2002` - 凭证无效
- `CodeAccountNotFound = 2003` - 账号不存在
- `CodeTaskNotFound = 2004` - 任务不存在
- `CodeProxyNotFound = 2005` - 代理不存在
- `CodeAccountBusy = 2006` - 账号忙碌
- `CodeConnectionFailed = 2007` - 连接失败

## 路由变更清单

### 更新操作（原 PUT）
- `/api/v1/auth/profile` PUT → POST
- `/api/v1/accounts/:id` PUT → POST `/accounts/:id/update`
- `/api/v1/templates/:id` PUT → POST `/templates/:id/update`
- `/api/v1/proxies/:id` PUT → POST `/proxies/:id/update`
- `/api/v1/tasks/:id` PUT → POST `/tasks/:id/update`

### 删除操作（原 DELETE）
- `/api/v1/accounts/:id` DELETE → POST `/accounts/:id/delete`
- `/api/v1/templates/:id` DELETE → POST `/templates/:id/delete`
- `/api/v1/files/:id` DELETE → POST `/files/:id/delete`
- `/api/v1/proxies/:id` DELETE → POST `/proxies/:id/delete`
- `/api/v1/tasks/:id` DELETE → POST `/tasks/:id/cancel`

## 验证

更新完成后，运行：
```bash
go build ./...
```

确保所有代码编译通过，然后测试各个 API 接口，验证响应格式是否符合要求。

