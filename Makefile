# TG Cloud Server Makefile

# Go相关配置
GO_VERSION := 1.21
GOCMD := go
GOBUILD := $(GOCMD) build
GOCLEAN := $(GOCMD) clean
GOTEST := $(GOCMD) test
GOGET := $(GOCMD) get
GOMOD := $(GOCMD) mod

# 项目信息
PROJECT_NAME := tg_cloud_server
VERSION := $(shell git describe --tags --always --dirty)
BUILD_TIME := $(shell date +%Y-%m-%d_%H:%M:%S)
GIT_COMMIT := $(shell git rev-parse HEAD)

# 构建目录
BUILD_DIR := build
BIN_DIR := $(BUILD_DIR)/bin

# 服务列表
SERVICES := web-api tg-manager task-scheduler ai-service migrate

# 构建标志
LDFLAGS := -X main.Version=$(VERSION) \
           -X main.BuildTime=$(BUILD_TIME) \
           -X main.GitCommit=$(GIT_COMMIT)

# Docker相关
DOCKER_REGISTRY := your-registry.com
DOCKER_NAMESPACE := tg-cloud

.PHONY: all build clean test deps docker-build docker-push help

# 默认目标
all: clean deps build

# 构建所有服务
build: $(SERVICES)

# 构建单个服务
$(SERVICES):
	@echo "Building $@..."
	@mkdir -p $(BIN_DIR)
	$(GOBUILD) -ldflags "$(LDFLAGS)" -o $(BIN_DIR)/$@ ./cmd/$@

# 清理构建文件
clean:
	@echo "Cleaning..."
	$(GOCLEAN)
	rm -rf $(BUILD_DIR)

# 运行测试
test:
	@echo "Running tests..."
	$(GOTEST) -v -race -coverprofile=coverage.out ./...

# 运行测试并生成覆盖率报告
test-coverage: test
	@echo "Generating coverage report..."
	$(GOCMD) tool cover -html=coverage.out -o coverage.html

# 下载依赖
deps:
	@echo "Downloading dependencies..."
	$(GOMOD) download
	$(GOMOD) tidy

# 更新依赖
deps-update:
	@echo "Updating dependencies..."
	$(GOGET) -u ./...
	$(GOMOD) tidy

# 代码格式化
fmt:
	@echo "Formatting code..."
	$(GOCMD) fmt ./...

# 代码静态检查
lint:
	@echo "Running linter..."
	golangci-lint run

# 生成代码
generate:
	@echo "Generating code..."
	$(GOCMD) generate ./...

# Docker构建
docker-build:
	@echo "Building Docker images..."
	@for service in $(SERVICES); do \
		if [ "$$service" != "migrate" ]; then \
			echo "Building Docker image for $$service..."; \
			docker build -f configs/docker/Dockerfile.$$service -t $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$$service:$(VERSION) .; \
			docker tag $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$$service:$(VERSION) $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$$service:latest; \
		fi \
	done

# Docker推送
docker-push:
	@echo "Pushing Docker images..."
	@for service in $(SERVICES); do \
		if [ "$$service" != "migrate" ]; then \
			echo "Pushing Docker image for $$service..."; \
			docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$$service:$(VERSION); \
			docker push $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)/$$service:latest; \
		fi \
	done

# 启动开发环境
dev-up:
	@echo "Starting development environment..."
	docker-compose -f configs/docker/docker-compose.yml -f configs/docker/docker-compose.dev.yml up -d

# 停止开发环境
dev-down:
	@echo "Stopping development environment..."
	docker-compose -f configs/docker/docker-compose.yml -f configs/docker/docker-compose.dev.yml down

# 查看开发环境日志
dev-logs:
	docker-compose -f configs/docker/docker-compose.yml -f configs/docker/docker-compose.dev.yml logs -f

# 启动生产环境
prod-up:
	@echo "Starting production environment..."
	docker-compose -f configs/docker/docker-compose.yml -f configs/docker/docker-compose.prod.yml up -d

# 停止生产环境
prod-down:
	@echo "Stopping production environment..."
	docker-compose -f configs/docker/docker-compose.yml -f configs/docker/docker-compose.prod.yml down

# 数据库迁移
migrate-up:
	@echo "Running database migrations..."
	$(BIN_DIR)/migrate up

migrate-down:
	@echo "Rolling back database migrations..."
	$(BIN_DIR)/migrate down

migrate-create:
	@echo "Creating new migration..."
	@read -p "Enter migration name: " name; \
	$(BIN_DIR)/migrate create $$name

# 安装开发工具
install-tools:
	@echo "Installing development tools..."
	$(GOGET) github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	$(GOGET) github.com/golang-migrate/migrate/v4/cmd/migrate@latest
	$(GOGET) github.com/swaggo/swag/cmd/swag@latest

# 生成API文档
docs:
	@echo "Generating API documentation..."
	swag init -g cmd/web-api/main.go -o docs/api

# 构建发布版本
release: clean deps test build
	@echo "Building release..."
	@mkdir -p $(BUILD_DIR)/release
	@tar -czf $(BUILD_DIR)/release/$(PROJECT_NAME)-$(VERSION)-linux-amd64.tar.gz -C $(BIN_DIR) .

# 基准测试
benchmark:
	@echo "Running benchmarks..."
	$(GOTEST) -bench=. -benchmem ./...

# 性能分析
profile:
	@echo "Running profiling..."
	$(GOTEST) -cpuprofile=cpu.prof -memprofile=mem.prof -bench=. ./...

# 安全检查
security:
	@echo "Running security checks..."
	gosec ./...

# 检查更新
check-updates:
	@echo "Checking for updates..."
	$(GOCMD) list -u -m all

# 本地运行服务
run-web-api: build
	@echo "Running web-api service..."
	CONFIG_PATH=configs/config.dev.yaml $(BIN_DIR)/web-api

run-tg-manager: build
	@echo "Running tg-manager service..."
	CONFIG_PATH=configs/config.dev.yaml $(BIN_DIR)/tg-manager

run-task-scheduler: build
	@echo "Running task-scheduler service..."
	CONFIG_PATH=configs/config.dev.yaml $(BIN_DIR)/task-scheduler

run-ai-service: build
	@echo "Running ai-service service..."
	CONFIG_PATH=configs/config.dev.yaml $(BIN_DIR)/ai-service

# 帮助信息
help:
	@echo "Available targets:"
	@echo "  build                 - Build all services"
	@echo "  clean                 - Clean build files"
	@echo "  test                  - Run tests"
	@echo "  test-coverage         - Run tests with coverage"
	@echo "  deps                  - Download dependencies"
	@echo "  deps-update          - Update dependencies"
	@echo "  fmt                  - Format code"
	@echo "  lint                 - Run linter"
	@echo "  generate             - Generate code"
	@echo "  docker-build         - Build Docker images"
	@echo "  docker-push          - Push Docker images"
	@echo "  dev-up               - Start development environment"
	@echo "  dev-down             - Stop development environment"
	@echo "  dev-logs             - View development logs"
	@echo "  prod-up              - Start production environment"
	@echo "  prod-down            - Stop production environment"
	@echo "  migrate-up           - Run database migrations"
	@echo "  migrate-down         - Rollback database migrations"
	@echo "  migrate-create       - Create new migration"
	@echo "  install-tools        - Install development tools"
	@echo "  docs                 - Generate API documentation"
	@echo "  release              - Build release package"
	@echo "  benchmark            - Run benchmarks"
	@echo "  profile              - Run profiling"
	@echo "  security             - Run security checks"
	@echo "  check-updates        - Check for dependency updates"
	@echo "  run-<service>        - Run specific service locally"
	@echo "  help                 - Show this help message"
