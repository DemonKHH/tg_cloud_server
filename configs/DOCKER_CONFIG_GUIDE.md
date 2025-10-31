# Docker 配置指南

## ⚠️ 重要说明

Docker 环境中的数据库连接配置与本地开发环境不同：

- **本地开发**：使用 `localhost` 连接数据库
- **Docker 环境**：使用 Docker Compose **服务名**连接数据库

## 📋 配置对应关系

### Docker Compose → config.yaml

| Docker Compose 配置 | config.yaml 配置 | 说明 |
|-------------------|-----------------|------|
| `MYSQL_DATABASE=tg_manager` | `database.mysql.database: "tg_manager"` | ✅ 一致 |
| `MYSQL_USER=tg_user` | `database.mysql.username: "tg_user"` | ✅ 一致 |
| `MYSQL_PASSWORD=tg_pass123` | `database.mysql.password` | ⚠️ 需手动同步 |
| 服务名 `mysql` | `database.mysql.host: "mysql"` | ⚠️ Docker 环境需使用服务名 |
| `REDIS_PORT=6379` | `database.redis.port: 6379` | ✅ 一致 |
| 服务名 `redis` | `database.redis.host: "redis"` | ⚠️ Docker 环境需使用服务名 |

## 🔧 解决方案

### 方案 1：使用 Docker 专用配置文件（推荐）

1. 使用 `config.docker.yaml`（已创建）
   - MySQL host: `mysql`（Docker 服务名）
   - Redis host: `redis`（Docker 服务名）

2. Docker Compose 中已配置为使用 `config.docker.yaml`

### 方案 2：使用环境变量覆盖

在 `docker-compose.yml` 中添加环境变量：

```yaml
web-api:
  environment:
    - DB_HOST=mysql
    - REDIS_HOST=redis
    - DB_PASSWORD=${DB_PASSWORD:-tg_pass123}
```

### 方案 3：动态检测环境

修改代码，自动检测 Docker 环境并切换 host。

## 📝 当前配置状态

### config.yaml（本地开发）
- MySQL host: `localhost`
- Redis host: `localhost`
- 适用于本地直接运行

### config.docker.yaml（Docker 环境）
- MySQL host: `mysql`
- Redis host: `redis`
- 适用于 Docker Compose 部署

## ✅ 验证步骤

1. **本地开发**：
   ```bash
   # 使用 config.yaml（默认）
   go run cmd/web-api/main.go
   ```

2. **Docker 环境**：
   ```bash
   # 使用 config.docker.yaml
   docker-compose up
   ```

## 🔍 检查清单

- [x] Docker Compose 服务名：`mysql`, `redis`
- [x] config.docker.yaml 中 host 使用服务名
- [x] docker-compose.yml 挂载正确的配置文件
- [ ] 数据库用户名和密码保持一致

