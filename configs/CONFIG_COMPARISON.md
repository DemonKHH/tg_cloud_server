# Docker Compose 与 config.yaml 配置对应表

## 🔍 配置对应关系检查

### MySQL 数据库配置

| Docker Compose | config.yaml (本地) | config.docker.yaml (Docker) | 状态 |
|---------------|-------------------|---------------------------|------|
| 服务名: `mysql` | `host: "localhost"` | `host: "mysql"` | ✅ 已修复 |
| `MYSQL_DATABASE=tg_manager` | `database: "tg_manager"` | `database: "tg_manager"` | ✅ 一致 |
| `MYSQL_USER=tg_user` | `username: "tg_user"` | `username: "tg_user"` | ✅ 一致 |
| `MYSQL_PASSWORD=tg_pass123` | `password: "your_password"` | `password: "tg_pass123"` | ✅ 已同步 |
| 端口 `3306` | `port: 3306` | `port: 3306` | ✅ 一致 |

### Redis 缓存配置

| Docker Compose | config.yaml (本地) | config.docker.yaml (Docker) | 状态 |
|---------------|-------------------|---------------------------|------|
| 服务名: `redis` | `host: "localhost"` | `host: "redis"` | ✅ 已修复 |
| 端口 `6379` | `port: 6379` | `port: 6379` | ✅ 一致 |
| 密码: 空 | `password: ""` | `password: ""` | ✅ 一致 |
| 数据库: 默认0 | `database: 6` | `database: 0` | ⚠️ 需确认 |
| `pool_size: 10` | `pool_size: 10` | `pool_size: 10` | ✅ 一致 |

### Web API 服务配置

| Docker Compose | config.yaml | 状态 |
|---------------|-------------|------|
| `WEB_API_PORT=8080` | `port: 8080` | ✅ 一致 |
| `host: "0.0.0.0"` | `host: "0.0.0.0"` | ✅ 一致 |

## ⚠️ 发现的问题

### 1. 数据库 Host 不匹配
- **问题**：Docker 环境需要使用服务名 `mysql` 和 `redis`，而不是 `localhost`
- **解决**：已创建 `config.docker.yaml`，使用正确的服务名
- **docker-compose.yml**：已更新为使用 `config.docker.yaml`

### 2. Redis Database 不一致
- **config.yaml**: `database: 6`
- **config.docker.yaml**: `database: 0` (默认值)
- **建议**：统一为 `database: 0`

### 3. MySQL 密码需要同步
- **docker-compose.yml 默认**: `tg_pass123`
- **config.yaml**: `your_password`
- **已修复**: `config.docker.yaml` 使用 `tg_pass123`

## ✅ 修复方案

### 方案 1：使用 Docker 专用配置文件（已实现）

1. **本地开发**：使用 `config.yaml`（host = localhost）
2. **Docker 环境**：使用 `config.docker.yaml`（host = 服务名）
3. **docker-compose.yml**：已配置使用 `config.docker.yaml`

### 方案 2：统一 Redis Database

修复 `config.yaml` 中的 Redis database：

```yaml
redis:
  database: 0  # 改为 0（与 Docker 默认一致）
```

## 📝 使用说明

### 本地开发
```bash
# 使用 config.yaml（默认）
go run cmd/web-api/main.go
```

### Docker 部署
```bash
# 自动使用 config.docker.yaml
cd configs/docker
docker-compose up
```

### 手动指定配置
```bash
# 环境变量指定配置文件
CONFIG_PATH=configs/config.docker.yaml go run cmd/web-api/main.go
```

