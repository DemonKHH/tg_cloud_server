# 账号搜索功能修复

## 问题描述

前端账号页面的搜索框无法正常工作，因为后端没有实现搜索逻辑。

## 修复内容

### 1. Repository 层 (internal/repository/account_repo.go)

**修改 `GetAccountSummaries` 方法**：
- 添加 `search` 参数
- 实现搜索逻辑：支持按手机号搜索
- 使用 SQL LIKE 查询：`phone LIKE ?`

```go
func (r *accountRepository) GetAccountSummaries(userID uint64, page, limit int, search string) ([]*models.AccountSummary, int64, error) {
    // 添加搜索条件（仅搜索手机号）
    if search != "" {
        query = query.Where("phone LIKE ?", "%"+search+"%")
    }
    // ...
}
```

### 2. Service 层 (internal/services/account_service.go)

**更新 `AccountFilter` 结构**：
- 添加 `Search` 字段

**更新 `GetAccounts` 方法**：
- 传递 search 参数到 repository 层

### 3. Handler 层 (internal/handlers/account_handler.go)

**更新 `GetAccounts` 方法**：
- 从查询参数中获取 `search` 参数
- 添加到过滤器中

```go
search := c.Query("search")
filter := &services.AccountFilter{
    Search: search,
    // ...
}
```

### 4. Model 层 (internal/models/account.go)

**更新 `AccountSummary` 结构**：
- 添加 `ProxyID` 字段（用于显示代理绑定状态）
- 添加 `LastUsedAt` 字段（用于显示最后使用时间）
- 添加 `CreatedAt` 字段（用于排序）

注意：由于 TGAccount 模型中没有 note 字段，AccountSummary 也不包含该字段。

## 搜索功能

现在支持以下搜索：
- **手机号搜索**：输入手机号的任意部分
- **模糊匹配**：使用 SQL LIKE 实现模糊搜索

注意：由于数据库表中没有 note 字段，暂时只支持手机号搜索。

## API 使用示例

```bash
# 获取所有账号（不搜索）
GET /api/v1/accounts?page=1&limit=50

# 搜索手机号包含 "123" 的账号
GET /api/v1/accounts?search=123&page=1&limit=50

# 搜索手机号包含 "+86" 的账号
GET /api/v1/accounts?search=+86&page=1&limit=50

# 组合搜索和状态过滤
GET /api/v1/accounts?search=123&status=normal&page=1&limit=50
```

## 参数说明

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 20 | 每页数量 |
| status | string | 否 | - | 账号状态过滤 |
| search | string | 否 | - | 搜索关键词（手机号） |

## 前端集成

前端已经实现了搜索框 UI，现在可以正常工作：
- 输入搜索关键词
- 自动防抖（500ms）
- 搜索时自动重置到第一页
- 显示清除按钮

## 测试

编译测试通过：
```bash
go build -o nul ./...
```

## 后续优化建议

1. 添加更多搜索字段（如状态、健康度范围等）
2. 支持高级搜索（多条件组合）
3. 添加搜索历史记录
4. 优化搜索性能（添加数据库索引）
