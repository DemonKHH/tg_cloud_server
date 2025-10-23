# TG Cloud Server 项目代码架构

## 📁 项目根目录结构

```
tg_cloud_server/
├── cmd/                          # 应用程序入口
│   ├── web-api/                 # Web API服务
│   │   └── main.go
│   ├── tg-manager/              # TG Manager服务  
│   │   └── main.go
│   ├── task-scheduler/          # Task Scheduler服务
│   │   └── main.go
│   ├── ai-service/              # AI Service服务
│   │   └── main.go
│   └── migrate/                 # 数据库迁移工具
│       └── main.go
├── internal/                     # 内部包（不对外暴露）
│   ├── common/                  # 公共组件
│   │   ├── config/              # 配置管理
│   │   ├── database/            # 数据库连接
│   │   ├── redis/               # Redis连接
│   │   ├── logger/              # 日志组件
│   │   ├── middleware/          # 中间件
│   │   ├── response/            # HTTP响应格式
│   │   └── utils/               # 工具函数
│   ├── models/                  # 数据模型
│   │   ├── user.go
│   │   ├── account.go
│   │   ├── task.go
│   │   ├── proxy.go
│   │   └── risk_log.go
│   ├── repository/              # 数据访问层
│   │   ├── user_repo.go
│   │   ├── account_repo.go
│   │   ├── task_repo.go
│   │   └── proxy_repo.go
│   ├── services/                # 业务逻辑层
│   │   ├── auth_service.go
│   │   ├── account_service.go
│   │   ├── task_service.go
│   │   ├── proxy_service.go
│   │   └── risk_service.go
│   ├── modules/                 # 五大核心模块
│   │   ├── checker/             # 账号检查模块
│   │   │   ├── checker.go
│   │   │   ├── health.go
│   │   │   └── batch.go
│   │   ├── private/             # 私信模块
│   │   │   ├── sender.go
│   │   │   ├── message.go
│   │   │   └── target.go
│   │   ├── broadcast/           # 群发模块
│   │   │   ├── broadcaster.go
│   │   │   ├── template.go
│   │   │   └── scheduler.go
│   │   ├── verify/              # 验证码接收模块
│   │   │   ├── receiver.go
│   │   │   ├── parser.go
│   │   │   └── handler.go
│   │   └── groupchat/           # AI炒群模块
│   │       ├── chatbot.go
│   │       ├── ai_engine.go
│   │       └── behavior.go
│   ├── telegram/                # Telegram客户端管理
│   │   ├── connection_pool.go   # 统一连接池
│   │   ├── client_manager.go    # 客户端管理器
│   │   ├── session_storage.go   # Session存储
│   │   └── proxy_dialer.go      # 代理拨号器
│   ├── scheduler/               # 任务调度器
│   │   ├── task_scheduler.go    # 主调度器
│   │   ├── queue_manager.go     # 队列管理
│   │   ├── account_validator.go # 账号验证
│   │   └── risk_controller.go   # 风控处理
│   └── handlers/                # HTTP处理器
│       ├── auth_handler.go
│       ├── account_handler.go
│       ├── task_handler.go
│       ├── module_handler.go
│       └── proxy_handler.go
├── pkg/                         # 可外部使用的包
│   ├── api/                     # API客户端
│   │   └── client.go
│   ├── types/                   # 类型定义
│   │   ├── account.go
│   │   ├── task.go
│   │   └── response.go
│   └── errors/                  # 错误定义
│       └── errors.go
├── web/                         # 前端资源（如果需要）
│   ├── static/
│   └── templates/
├── configs/                     # 配置文件
│   ├── config.yaml              # 主配置文件
│   ├── config.dev.yaml          # 开发环境配置
│   ├── config.prod.yaml         # 生产环境配置
│   └── docker/                  # Docker相关配置
│       ├── Dockerfile.web-api
│       ├── Dockerfile.tg-manager
│       ├── Dockerfile.task-scheduler
│       ├── Dockerfile.ai-service
│       └── docker-compose.yml
├── migrations/                  # 数据库迁移文件
│   ├── 001_create_users_table.up.sql
│   ├── 001_create_users_table.down.sql
│   ├── 002_create_accounts_table.up.sql
│   ├── 002_create_accounts_table.down.sql
│   ├── 003_create_tasks_table.up.sql
│   ├── 003_create_tasks_table.down.sql
│   ├── 004_create_proxy_ips_table.up.sql
│   ├── 004_create_proxy_ips_table.down.sql
│   ├── 005_create_task_logs_table.up.sql
│   ├── 005_create_task_logs_table.down.sql
│   ├── 006_create_risk_logs_table.up.sql
│   └── 006_create_risk_logs_table.down.sql
├── scripts/                     # 脚本文件
│   ├── build.sh                 # 构建脚本
│   ├── deploy.sh                # 部署脚本
│   ├── test.sh                  # 测试脚本
│   └── setup.sh                 # 环境设置脚本
├── docs/                        # 文档目录
│   ├── api/                     # API文档
│   │   ├── auth.md
│   │   ├── accounts.md
│   │   ├── tasks.md
│   │   └── modules.md
│   ├── deployment.md            # 部署文档
│   └── development.md           # 开发文档
├── tests/                       # 测试文件
│   ├── integration/             # 集成测试
│   ├── unit/                    # 单元测试
│   └── fixtures/                # 测试数据
├── tools/                       # 开发工具
│   ├── gen/                     # 代码生成工具
│   └── mock/                    # Mock工具
├── .gitignore                   # Git忽略文件
├── .env.example                 # 环境变量示例
├── go.mod                       # Go模块文件
├── go.sum                       # Go依赖校验
├── Makefile                     # Make构建文件
├── README.md                    # 项目说明
└── LICENSE                      # 许可证文件
```

## 🏗️ 各服务详细架构

### 1. Web API 服务架构
```
internal/handlers/
├── auth_handler.go              # 用户认证处理
├── account_handler.go           # 账号管理处理
├── task_handler.go              # 任务管理处理
├── module_handler.go            # 模块功能处理
├── proxy_handler.go             # 代理管理处理
├── stats_handler.go             # 统计数据处理
└── websocket_handler.go         # WebSocket处理

internal/middleware/
├── auth.go                      # JWT认证中间件
├── cors.go                      # CORS中间件
├── ratelimit.go                 # 限流中间件
├── logging.go                   # 日志中间件
└── recovery.go                  # 恢复中间件

internal/routes/
├── auth.go                      # 认证路由
├── api.go                       # API路由
└── websocket.go                 # WebSocket路由
```

### 2. TG Manager 服务架构
```
internal/telegram/
├── connection_pool.go           # 连接池管理
│   ├── type ConnectionPool struct
│   ├── func GetOrCreateConnection()
│   ├── func ExecuteTask()
│   └── func cleanupIdleConnections()
├── client_manager.go            # 客户端管理
│   ├── type TGManager struct
│   ├── func CreateClient()
│   ├── func HealthCheck()
│   └── func HandleUpdates()
├── session_storage.go           # Session存储
│   ├── type DatabaseSessionStorage struct
│   ├── func LoadSession()
│   └── func StoreSession()
├── proxy_dialer.go              # 代理拨号
│   ├── func createProxyDialer()
│   └── func testProxyConnection()
└── task_executor.go             # 任务执行器
    ├── type TaskExecutor struct
    ├── func ExecuteWithRiskControl()
    └── func HandleTaskResult()
```

### 3. Task Scheduler 服务架构
```
internal/scheduler/
├── task_scheduler.go            # 主调度器
│   ├── type TaskScheduler struct
│   ├── func SubmitTask()
│   ├── func ValidateAccount()
│   └── func GetAccountAvailability()
├── queue_manager.go             # 队列管理
│   ├── type QueueManager struct
│   ├── func AddToQueue()
│   ├── func GetNextTask()
│   └── func RemoveFromQueue()
├── account_validator.go         # 账号验证
│   ├── type AccountValidator struct
│   ├── func ValidateAccountForTask()
│   └── func GetValidationResult()
└── risk_controller.go           # 风控处理
    ├── type RiskController struct
    ├── func EvaluateRisk()
    ├── func ApplyRiskAction()
    └── func LogRiskEvent()
```

### 4. AI Service 服务架构
```
internal/ai/
├── ai_service.go                # AI服务主体
│   ├── type AIService struct
│   ├── func GenerateMessage()
│   ├── func AnalyzeSentiment()
│   └── func PredictRisk()
├── openai_client.go             # OpenAI客户端
│   ├── type OpenAIClient struct
│   ├── func ChatCompletion()
│   └── func HandleResponse()
├── nlp_engine.go                # 自然语言处理
│   ├── type NLPEngine struct
│   ├── func ExtractKeywords()
│   ├── func AnalyzeContext()
│   └── func GenerateReply()
└── analytics_engine.go          # 分析引擎
    ├── type AnalyticsEngine struct
    ├── func AnalyzePattern()
    └── func PredictOutcome()
```

## 📦 核心模块详细结构

### 五大核心模块架构
```
internal/modules/

├── checker/                     # 账号检查模块
│   ├── checker.go              # 主检查器
│   │   ├── type AccountChecker struct
│   │   ├── func CheckAccount()
│   │   ├── func BatchCheck()
│   │   └── func GenerateReport()
│   ├── health.go               # 健康度评估
│   │   ├── func CalculateHealthScore()
│   │   ├── func CheckConnection()
│   │   └── func ValidateSession()
│   └── batch.go                # 批量处理
│       ├── func BatchAccountCheck()
│       └── func ProcessResults()

├── private/                     # 私信模块
│   ├── sender.go               # 私信发送器
│   │   ├── type PrivateSender struct
│   │   ├── func SendMessage()
│   │   └── func BatchSend()
│   ├── message.go              # 消息处理
│   │   ├── type Message struct
│   │   ├── func FormatMessage()
│   │   └── func ValidateContent()
│   └── target.go               # 目标用户管理
│       ├── func GetTargetUsers()
│       ├── func FilterUsers()
│       └── func ValidateTargets()

├── broadcast/                   # 群发模块
│   ├── broadcaster.go          # 群发器
│   │   ├── type Broadcaster struct
│   │   ├── func BroadcastToGroups()
│   │   └── func BroadcastToChannels()
│   ├── template.go             # 模板管理
│   │   ├── type MessageTemplate struct
│   │   ├── func ApplyTemplate()
│   │   └── func ReplaceVariables()
│   └── scheduler.go            # 发送调度
│       ├── func ScheduleBroadcast()
│       └── func OptimizeTiming()

├── verify/                      # 验证码接收模块
│   ├── receiver.go             # 验证码接收器
│   │   ├── type CodeReceiver struct
│   │   ├── func StartListening()
│   │   └── func HandleCode()
│   ├── parser.go               # 验证码解析
│   │   ├── func ParseSMSCode()
│   │   ├── func ParseTelegramCode()
│   │   └── func ExtractCode()
│   └── handler.go              # 处理器
│       ├── func ProcessCode()
│       └── func NotifyResult()

└── groupchat/                   # AI炒群模块
    ├── chatbot.go              # 聊天机器人
    │   ├── type GroupChatBot struct
    │   ├── func JoinGroup()
    │   ├── func SendMessage()
    │   └── func InteractNaturally()
    ├── ai_engine.go            # AI引擎
    │   ├── type AIEngine struct
    │   ├── func GenerateResponse()
    │   ├── func AnalyzeContext()
    │   └── func SelectReplyTime()
    └── behavior.go             # 行为模拟
        ├── func SimulateTyping()
        ├── func SimulateReactions()
        └── func FollowConversation()
```

## 🗄️ 数据模型结构

### Models 包结构
```
internal/models/
├── user.go                     # 用户模型
│   └── type User struct
├── account.go                  # 账号模型
│   ├── type TGAccount struct
│   ├── type AccountStatus enum
│   └── type ConnectionStatus enum
├── task.go                     # 任务模型
│   ├── type Task struct
│   ├── type TaskType enum
│   ├── type TaskStatus enum
│   └── type TaskConfig interface
├── proxy.go                    # 代理模型
│   ├── type ProxyIP struct
│   └── type ProxyProtocol enum
├── risk_log.go                 # 风控日志模型
│   ├── type RiskLog struct
│   └── type RiskLevel enum
└── common.go                   # 通用模型
    ├── type BaseModel struct
    ├── type PaginationRequest struct
    └── type PaginationResponse struct
```

## 🔧 配置文件结构

### 主配置文件 (configs/config.yaml)
```yaml
# 服务配置
server:
  web_api:
    host: "0.0.0.0"
    port: 8080
  tg_manager:
    host: "0.0.0.0"
    port: 8081
  task_scheduler:
    host: "0.0.0.0"
    port: 8082
  ai_service:
    host: "0.0.0.0"
    port: 8083

# 数据库配置
database:
  mysql:
    host: "localhost"
    port: 3306
    username: "root"
    password: ""
    database: "tg_manager"
    max_open_conns: 100
    max_idle_conns: 10
  redis:
    host: "localhost"
    port: 6379
    password: ""
    database: 0

# Telegram配置
telegram:
  api_id: 12345
  api_hash: "your_api_hash"
  connection_pool:
    max_connections: 1000
    idle_timeout: "30m"
    cleanup_interval: "5m"

# AI配置
ai:
  openai:
    api_key: "your_openai_key"
    model: "gpt-3.5-turbo"
    max_tokens: 1000

# 风控配置
risk_control:
  enabled: true
  check_interval: "1m"
  max_failures: 3
  cooldown_duration: "30m"

# 日志配置
logging:
  level: "info"
  format: "json"
  output: "stdout"
```

这个项目架构完全基于我们之前讨论的技术方案，采用标准的Go项目结构，支持微服务部署，具备高可扩展性和维护性。
