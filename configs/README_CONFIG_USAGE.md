# 配置文件使用说明

## 📋 配置文件说明

项目中有两个配置文件，用于不同的运行场景：

### 1. `config.yaml` - 本地开发配置
- **用途**：本地运行应用，连接 Docker 容器中的数据库
- **数据库连接**：使用 `localhost` 连接（Docker 映射的端口）
- **适用场景**：开发调试，热重载

### 2. `config.docker.yaml` - Docker 环境配置
- **用途**：Docker 容器内运行应用
- **数据库连接**：使用 Docker 服务名 `mysql`、`redis`
- **适用场景**：Docker Compose 部署

## 🚀 使用场景

### 场景 1：本地运行应用 + Docker 数据库（推荐开发）

**步骤：**

1. **启动 Docker 数据库服务**：
   ```bash
   cd configs/docker
   # 只启动数据库服务，不启动 web-api
   docker-compose up mysql redis -d
   ```

2. **本地运行应用**：
   ```bash
   # 使用 config.yaml（默认配置，host=localhost）
   go run cmd/web-api/main.go
   ```

**配置对应：**
- `config.yaml` → MySQL: `localhost:3306`, Redis: `localhost:6379`
- Docker 端口映射：`3306:3306`, `6379:6379`

### 场景 2：完全 Docker 部署

**步骤：**

1. **启动所有服务**：
   ```bash
   cd configs/docker
   docker-compose up -d
   ```

2. **应用自动使用 `config.docker.yaml`**

**配置对应：**
- `config.docker.yaml` → MySQL: `mysql:3306`, Redis: `redis:6379`
- Docker Compose 自动配置 `CONFIG_PATH`

## ⚙️ 配置文件对比

| 配置项 | config.yaml (本地) | config.docker.yaml (Docker) |
|--------|-------------------|---------------------------|
| MySQL host | `localhost` | `mysql` (服务名) |
| Redis host | `localhost` | `redis` (服务名) |
| 使用场景 | 本地开发 | Docker 部署 |

## 🔧 Docker Compose 只启动数据库

如果您想本地运行应用但使用 Docker 的数据库，可以这样操作：

```bash
cd configs/docker

# 方法 1：只启动数据库服务
docker-compose up mysql redis -d

# 方法 2：启动所有服务但排除 web-api（注释掉 web-api 服务）

# 然后在项目根目录运行应用
go run cmd/web-api/main.go
```

## 📝 注意事项

1. **端口占用**：确保本地 3306 和 6379 端口未被占用
2. **配置文件**：本地开发使用 `config.yaml`，Docker 使用 `config.docker.yaml`
3. **数据库密码**：两个配置文件中的密码应该与 `docker-compose.yml` 保持一致

## ✅ 快速验证

```bash
# 1. 启动数据库
cd configs/docker && docker-compose up mysql redis -d

# 2. 检查数据库是否启动
docker ps

# 3. 本地运行应用
go run cmd/web-api/main.go

# 4. 测试连接
curl http://localhost:8080/health
```

