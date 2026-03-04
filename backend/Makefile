.PHONY: build build-api build-worker build-metrics-worker run run-worker run-metrics-worker test clean dev dev-worker dev-metrics-worker docker-build docker-run migrate

# 变量
APP_NAME=nexus-api
BUILD_DIR=./build
API_MAIN=./cmd/api-server/main.go
WORKER_MAIN=./cmd/job-worker/main.go
METRICS_WORKER_MAIN=./cmd/metrics-worker/main.go
API_BIN=$(BUILD_DIR)/api-server
WORKER_BIN=$(BUILD_DIR)/job-worker
METRICS_WORKER_BIN=$(BUILD_DIR)/metrics-worker

# 构建 API
build-api:
	@echo "Building $(APP_NAME)..."
	@go build -o $(API_BIN) $(API_MAIN)
	@echo "Build complete: $(API_BIN)"

# 构建 Worker
build-worker:
	@echo "Building $(APP_NAME) worker..."
	@go build -o $(WORKER_BIN) $(WORKER_MAIN)
	@echo "Build complete: $(WORKER_BIN)"

# 构建 Metrics Worker
build-metrics-worker:
	@echo "Building $(APP_NAME) metrics worker..."
	@go build -o $(METRICS_WORKER_BIN) $(METRICS_WORKER_MAIN)
	@echo "Build complete: $(METRICS_WORKER_BIN)"

# 构建（API + Worker + Metrics Worker）
build: build-api build-worker build-metrics-worker

# 运行 API
run: build-api
	@echo "Running $(APP_NAME)..."
	@$(API_BIN)

# 运行 Worker
run-worker: build-worker
	@echo "Running $(APP_NAME) worker..."
	@$(WORKER_BIN)

# 运行 Metrics Worker
run-metrics-worker: build-metrics-worker
	@echo "Running $(APP_NAME) metrics worker..."
	@$(METRICS_WORKER_BIN)

# 开发模式（热重载）
dev:
	@echo "Starting development server..."
	@go run $(API_MAIN)

# 开发模式运行 Worker
dev-worker:
	@echo "Starting job worker..."
	@go run $(WORKER_MAIN)

# 开发模式运行 Metrics Worker
dev-metrics-worker:
	@echo "Starting metrics worker..."
	@go run $(METRICS_WORKER_MAIN)

# 测试
test:
	@echo "Running tests..."
	@go test -v ./...

# 测试覆盖率
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# 清理
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html
	@echo "Clean complete"

# 依赖
deps:
	@echo "Downloading dependencies..."
	@go mod download
	@go mod tidy
	@echo "Dependencies updated"

# 代码检查
lint:
	@echo "Running linter..."
	@golangci-lint run ./...

# 格式化
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Docker 构建
docker-build:
	@echo "Building Docker image..."
	@docker build -t $(APP_NAME):latest .

# Docker 运行
docker-run:
	@echo "Running Docker container..."
	@docker run -p 8080:8080 --env-file .env $(APP_NAME):latest

# 生成 Swagger 文档
swagger:
	@echo "Generating Swagger documentation..."
	@swag init -g $(API_MAIN) -o ./docs

# 数据库迁移
migrate:
	@echo "Running database migrations..."
	@go run $(API_MAIN) migrate

# 帮助
help:
	@echo "Available commands:"
	@echo "  make build         - Build API + worker"
	@echo "  make build-api     - Build API server"
	@echo "  make build-worker  - Build job worker"
	@echo "  make build-metrics-worker - Build metrics worker"
	@echo "  make run           - Build and run API server"
	@echo "  make run-worker    - Build and run job worker"
	@echo "  make run-metrics-worker - Build and run metrics worker"
	@echo "  make dev           - Run API server in development mode"
	@echo "  make dev-worker    - Run job worker in development mode"
	@echo "  make dev-metrics-worker - Run metrics worker in development mode"
	@echo "  make test          - Run tests"
	@echo "  make test-coverage - Run tests with coverage report"
	@echo "  make clean         - Clean build artifacts"
	@echo "  make deps          - Download and tidy dependencies"
	@echo "  make lint          - Run linter"
	@echo "  make fmt           - Format code"
	@echo "  make docker-build  - Build Docker image"
	@echo "  make docker-run    - Run Docker container"
	@echo "  make swagger       - Generate Swagger documentation"
	@echo "  make migrate       - Run database migrations"
