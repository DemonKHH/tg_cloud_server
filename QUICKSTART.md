# 🚀 TG Cloud Server 快速启动指南

## 📋 前置要求

- Go 1.21+
- MySQL 8.0+
- Redis 7.0+
- Docker & Docker Compose (可选)
- Telegram API credentials (从 https://my.telegram.org/apps 获取)
- OpenAI API Key (可选，用于AI功能)

## 🛠️ 快速部署

### 方式一：Docker Compose 部署 (推荐)

1. **克隆并进入项目目录**
```bash
git clone <repository-url>
cd tg_cloud_server
```

2. **配置环境变量**
```bash
cp env.example .env
# 编辑 .env 文件，填入实际配置
vim .env
```

3. **启动服务**
```bash
# 启动开发环境
make dev-up

# 或启动生产环境
make prod-up
```

4. **查看服务状态**
```bash
# 查看服务日志
make dev-logs

# 检查健康状态
curl http://localhost:8080/health
```

### 方式二：本地开发部署

1. **安装依赖**
```bash
make deps
make install-tools
```

2. **配置数据库**
```bash
# 启动MySQL和Redis
docker run -d --name mysql -p 3306:3306 -e MYSQL_ROOT_PASSWORD=root123 mysql:8.0
docker run -d --name redis -p 6379:6379 redis:7.0-alpine

# 运行数据库迁移
make migrate-up
```

3. **配置文件**
```bash
cp configs/config.example.yaml configs/config.yaml
# 编辑配置文件
vim configs/config.yaml
```

4. **构建和运行**
```bash
# 构建所有服务
make build

# 分别运行各服务 (需要多个终端)
make run-web-api
make run-tg-manager  
make run-task-scheduler
make run-ai-service
```

## 🔧 核心配置

### 1. Telegram API 配置
```yaml
telegram:
  api_id: 12345  # 你的API ID
  api_hash: "your_api_hash"  # 你的API Hash
```

### 2. 数据库配置
```yaml
database:
  mysql:
    host: "localhost"
    port: 3306
    username: "tg_user"
    password: "your_password"
    database: "tg_manager"
  redis:
    host: "localhost"
    port: 6379
```

### 3. AI服务配置 (可选)
```yaml
ai:
  openai:
    api_key: "your_openai_key"
    model: "gpt-3.5-turbo"
```

## 📚 基本使用

### 1. 用户注册和登录

**注册用户**
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "email": "admin@example.com",
    "password": "password123"
  }'
```

**用户登录**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "password123"
  }'
```

### 2. 添加TG账号

**添加账号**
```bash
curl -X POST http://localhost:8080/api/v1/accounts \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token>" \
  -d '{
    "phone": "+1234567890"
  }'
```

**检查账号状态**
```bash
curl -X GET http://localhost:8080/api/v1/accounts/1/health \
  -H "Authorization: Bearer <your_token>"
```

### 3. 执行任务

**账号检查任务**
```bash
curl -X POST http://localhost:8080/api/v1/modules/check \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token>" \
  -d '{
    "account_id": "1"
  }'
```

**私信发送任务**
```bash
curl -X POST http://localhost:8080/api/v1/modules/private \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token>" \
  -d '{
    "account_id": "1",
    "task_config": {
      "targets": ["username1", "username2"],
      "message": "Hello from TG Cloud Server!"
    }
  }'
```

### 4. 代理管理

**添加代理**
```bash
curl -X POST http://localhost:8080/api/v1/proxies \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token>" \
  -d '{
    "name": "My Proxy",
    "ip": "192.168.1.100",
    "port": 1080,
    "protocol": "socks5",
    "username": "proxy_user",
    "password": "proxy_pass"
  }'
```

**绑定代理到账号**
```bash
curl -X POST http://localhost:8080/api/v1/accounts/1/bind-proxy \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer <your_token>" \
  -d '{
    "account_id": "1",
    "proxy_id": "1"
  }'
```

## 🔍 监控和调试

### 1. 健康检查
```bash
# Web API健康检查
curl http://localhost:8080/health

# 所有服务健康检查
curl http://localhost:8081/health  # TG Manager
curl http://localhost:8082/health  # Task Scheduler  
curl http://localhost:8083/health  # AI Service
```

### 2. 监控指标 (如果启用)
```bash
# Prometheus指标
curl http://localhost:9091/metrics

# Grafana仪表板
open http://localhost:3000
```

### 3. 查看日志
```bash
# Docker环境查看日志
docker-compose logs -f web-api
docker-compose logs -f tg-manager

# 本地环境查看日志文件
tail -f logs/app.log
```

## 🐛 常见问题

### 1. 连接数据库失败
```bash
# 检查MySQL服务状态
docker ps | grep mysql
# 检查配置中的数据库连接信息
```

### 2. Telegram连接失败
```bash
# 检查API ID和API Hash是否正确
# 检查网络连接
# 检查代理配置
```

### 3. 任务执行失败
```bash
# 查看任务日志
curl -X GET http://localhost:8080/api/v1/tasks/1 \
  -H "Authorization: Bearer <your_token>"

# 检查账号状态
curl -X GET http://localhost:8080/api/v1/accounts/1/availability \
  -H "Authorization: Bearer <your_token>"
```

## 📖 开发指南

### 1. 项目结构
```
tg_cloud_server/
├── cmd/                    # 应用程序入口
├── internal/               # 内部包
│   ├── models/            # 数据模型
│   ├── handlers/          # HTTP处理器
│   ├── services/          # 业务逻辑
│   ├── modules/           # 五大核心模块
│   └── telegram/          # TG连接管理
├── configs/               # 配置文件
└── migrations/            # 数据库迁移
```

### 2. 添加新功能
1. 在 `internal/models/` 中定义数据模型
2. 在 `internal/services/` 中实现业务逻辑
3. 在 `internal/handlers/` 中添加HTTP处理器
4. 在路由中注册新的API端点

### 3. 添加新的任务类型
1. 在 `internal/models/task.go` 中添加新的任务类型
2. 在 `internal/modules/` 中创建新的模块目录
3. 实现 `telegram.TaskInterface` 接口
4. 在调度器中注册新的任务执行器

## 🚀 生产部署

### 1. 环境准备
- 配置生产环境的数据库
- 设置安全的JWT密钥
- 配置HTTPS证书
- 设置监控和告警

### 2. 性能优化
- 调整数据库连接池大小
- 配置适当的TG连接池参数
- 启用Redis缓存
- 配置负载均衡

### 3. 安全配置
- 使用强密码和密钥
- 配置防火墙规则
- 启用请求限流
- 定期备份数据

## 📞 支持

如果遇到问题，请：
1. 查看日志文件获取详细错误信息
2. 检查配置文件是否正确
3. 确认所有依赖服务正常运行
4. 参考API文档进行调试

---

🎉 **恭喜！你已经成功启动了TG Cloud Server系统！**

现在你可以开始管理你的Telegram账号，执行各种自动化任务了。
