# 接口控制中间件使用指南

## 📋 已实现的接口控制功能

### ✅ 1. 增强的认证中间件 (`auth.go`)
- **获取用户完整信息**：从数据库加载用户资料（包括角色和权限）
- **用户状态检查**：自动检查用户是否被禁用
- **上下文存储**：将用户ID、角色和用户资料存储到请求上下文

**使用方式**：
```go
api.Use(middleware.JWTAuthMiddleware(authService))
```

**上下文中的可用数据**：
- `c.Get("user_id")` - 用户ID (uint64)
- `c.Get("user_role")` - 用户角色 (models.UserRole)
- `c.Get("user_profile")` - 用户完整资料 (*models.UserProfile)

---

### ✅ 2. 基于角色的权限控制 (`permission.go`)

#### RequireRole - 要求指定角色
```go
// 只允许管理员访问
adminRoutes.Use(middleware.RequireAdmin())

// 只允许高级用户或管理员
premiumRoutes.Use(middleware.RequirePremium())

// 要求多个角色中的任意一个
routes.Use(middleware.RequireRole(models.RoleAdmin, models.RolePremium))
```

#### RequirePermission - 要求指定权限
```go
// 要求基础功能权限
routes.Use(middleware.RequirePermission("basic_features"))

// 要求高级功能权限
routes.Use(middleware.RequirePermission("advanced_features"))

// 要求管理用户权限（仅管理员）
routes.Use(middleware.RequirePermission("manage_users"))
```

#### RequireAnyPermission - 要求任意一个权限
```go
// 要求拥有任意一个权限
routes.Use(middleware.RequireAnyPermission(
    "unlimited_accounts",
    "advanced_features",
))
```

**可用权限列表**：
- `"basic_features"` - 基础功能（所有活跃用户）
- `"advanced_features"` - 高级功能（Premium/Admin）
- `"unlimited_accounts"` - 无限制账号数（Premium/Admin）
- `"manage_users"` - 管理用户（仅Admin）

---

### ✅ 3. 基于用户的限流 (`user_ratelimit.go`)

#### UserRateLimit - 基于用户的限流
```go
// 每个用户每分钟最多100个请求
api.Use(middleware.UserRateLimit(redisClient, 100, time.Minute))

// 每个用户每小时最多1000个请求
api.Use(middleware.UserRateLimit(redisClient, 1000, time.Hour))
```

**特点**：
- 基于用户ID限流（已登录用户）
- 未登录用户自动降级为IP限流
- 返回标准的RateLimit响应头
- 支持自定义限制数量和时间窗口

#### APIEndpointRateLimit - 接口级别限流
```go
// 配置不同接口的不同限流策略
endpointLimits := map[string]middleware.EndpointLimit{
    "POST:/api/v1/modules/broadcast": {
        Limit:  10,           // 每分钟10次
        Window: time.Minute,
    },
    "POST:/api/v1/modules/private": {
        Limit:  30,           // 每分钟30次
        Window: time.Minute,
    },
    "GET:/api/v1/stats/overview": {
        Limit:  60,           // 每分钟60次
        Window: time.Minute,
    },
}

api.Use(middleware.APIEndpointRateLimit(redisClient, endpointLimits))
```

---

### ✅ 4. 接口访问日志和统计 (`access_log.go`)

#### AccessLogMiddleware - 访问日志和统计
```go
// 自动记录所有API访问并统计
router.Use(middleware.AccessLogMiddleware(redisClient))
```

**功能**：
- 记录每次API访问的详细信息
- 统计接口调用次数（总计、每小时、每天）
- 统计成功/失败次数
- 统计平均响应时间
- 按用户统计调用次数
- 数据存储在Redis中，支持查询

#### GetAPIStats - 获取接口统计信息
```go
// 获取特定接口的统计信息
stats, err := middleware.GetAPIStats(redisClient, "POST", "/api/v1/modules/broadcast")
if err == nil {
    // stats包含：
    // - total_calls: 总调用次数
    // - success_calls: 成功次数
    // - error_calls: 错误次数
    // - avg_response_time_ms: 平均响应时间(毫秒)
}
```

---

## 📝 路由配置示例

### 示例1：基础路由（仅认证）
```go
// 所有用户都可以访问
api := router.Group("/api/v1")
api.Use(middleware.JWTAuthMiddleware(authService))
{
    api.GET("/profile", handler.GetProfile)
    api.PUT("/profile", handler.UpdateProfile)
}
```

### 示例2：需要特定权限的路由
```go
// 需要基础功能权限
modules := api.Group("/modules")
modules.Use(middleware.RequirePermission("basic_features"))
{
    modules.POST("/check", handler.CheckAccount)
    modules.POST("/private", handler.SendPrivateMessage)
}

// 需要高级功能权限
advanced := api.Group("/advanced")
advanced.Use(middleware.RequirePermission("advanced_features"))
{
    advanced.POST("/batch", handler.BatchOperation)
}
```

### 示例3：管理员专用路由
```go
// 仅管理员可以访问
admin := api.Group("/admin")
admin.Use(middleware.RequireAdmin())
{
    admin.GET("/users", handler.ListUsers)
    admin.POST("/users", handler.CreateUser)
    admin.DELETE("/users/:id", handler.DeleteUser)
}
```

### 示例4：混合权限控制
```go
accounts := api.Group("/accounts")
{
    // 所有认证用户都可以查看
    accounts.GET("", handler.GetAccounts)
    accounts.GET("/:id", handler.GetAccount)
    
    // 创建账号需要基础权限
    accounts.POST("", 
        middleware.RequirePermission("basic_features"),
        handler.CreateAccount)
    
    // 删除账号需要高级权限
    accounts.DELETE("/:id",
        middleware.RequirePermission("advanced_features"),
        handler.DeleteAccount)
    
    // 批量操作需要高级用户
    accounts.POST("/batch/bind-proxy",
        middleware.RequirePremium(),
        handler.BatchBindProxy)
}
```

### 示例5：接口级别限流
```go
// 为敏感接口设置更严格的限流
sensitiveLimits := map[string]middleware.EndpointLimit{
    "POST:/api/v1/modules/broadcast": {
        Limit:  5,            // 每分钟5次
        Window: time.Minute,
    },
    "POST:/api/v1/modules/groupchat": {
        Limit:  10,           // 每分钟10次
        Window: time.Minute,
    },
}

api.Use(middleware.APIEndpointRateLimit(redisClient, sensitiveLimits))
```

---

## 🔒 权限映射表

| 权限名称 | 标准用户 | 高级用户 | 管理员 |
|---------|---------|---------|--------|
| `basic_features` | ✅ | ✅ | ✅ |
| `advanced_features` | ❌ | ✅ | ✅ |
| `unlimited_accounts` | ❌ | ✅ | ✅ |
| `manage_users` | ❌ | ❌ | ✅ |

---

## 📊 响应头说明

### RateLimit响应头
所有限流中间件都会设置以下响应头：
- `X-RateLimit-Limit`: 限制数量
- `X-RateLimit-Remaining`: 剩余请求数
- `X-RateLimit-Reset`: 重置时间戳（Unix时间）

---

## 🚨 错误响应

### 认证失败 (401)
```json
{
  "error": "unauthorized",
  "message": "缺少认证令牌"
}
```

### 权限不足 (403)
```json
{
  "error": "forbidden",
  "message": "权限不足",
  "required_permission": "advanced_features"
}
```

### 限流超限 (429)
```json
{
  "error": "rate_limit_exceeded",
  "message": "请求过于频繁，请稍后重试",
  "retry_after": 60
}
```

---

## 💡 最佳实践

1. **认证中间件放在最前面**：确保后续中间件可以获取用户信息
2. **权限检查放在路由组级别**：避免重复代码
3. **敏感接口设置更严格的限流**：使用`APIEndpointRateLimit`
4. **记录访问日志**：使用`AccessLogMiddleware`监控API使用情况
5. **合理设置限流策略**：平衡用户体验和系统安全

---

## 📈 监控和统计

所有API访问都会被记录到Redis中，可以通过以下键查询：
- `api:stats:POST:/api/v1/modules/broadcast` - 总调用次数
- `api:stats:hourly:POST:/api/v1/modules/broadcast:2024-12-19-14` - 每小时统计
- `api:stats:daily:POST:/api/v1/modules/broadcast:2024-12-19` - 每天统计
- `api:stats:user:123:POST:/api/v1/modules/broadcast` - 用户调用统计

