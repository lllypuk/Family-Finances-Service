# Переменные
APP_NAME=family-budget-service
BUILD_DIR=./build
DOCKER_COMPOSE_FILE=docker/docker-compose.yml

# Сборка приложения
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

# Запуск приложения
.PHONY: run
run:
	@echo "Running $(APP_NAME)..."
	@go run ./cmd/server/main.go

# Запуск с локальными переменными окружения
.PHONY: run-local
run-local:
	@echo "Running $(APP_NAME) with local config..."
	@SERVER_PORT=8080 \
	 SERVER_HOST=localhost \
	 MONGODB_URI=mongodb://admin:password123@localhost:27017/family_budget?authSource=admin \
	 MONGODB_DATABASE=family_budget \
	 SESSION_SECRET=your-super-secret-session-key-for-local-dev \
	 REDIS_URL=redis://:redis123@localhost:6379 \
	 LOG_LEVEL=debug \
	 ENVIRONMENT=development \
	 go run ./cmd/server/main.go

# Быстрые тесты с переиспользованием MongoDB контейнера
.PHONY: test
test:
	@echo "Running fast tests with shared MongoDB container..."
	@REUSE_MONGO_CONTAINER=true go test -v ./...

# Юнит тесты с быстрыми контейнерами
.PHONY: test-unit
test-unit:
	@echo "Running unit tests with fast containers..."
	@REUSE_MONGO_CONTAINER=true go test -v ./internal/...

# Интеграционные тесты с переиспользованием контейнера
.PHONY: test-integration
test-integration:
	@echo "Running integration tests with shared container..."
	@REUSE_MONGO_CONTAINER=true go test -v ./tests/...

# Быстрые тесты с покрытием и переиспользованием контейнера
.PHONY: test-coverage
test-coverage:
	@echo "Running fast tests with coverage and shared MongoDB container..."
	@REUSE_MONGO_CONTAINER=true go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Установка зависимостей
.PHONY: deps
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Линтер
.PHONY: lint
lint:
	@echo "Running linter..."
	@golangci-lint run --fix

# Форматирование кода
.PHONY: fmt
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Проверка перед коммитом
.PHONY: pre-commit
pre-commit:
	@echo "Running pre-commit checks..."
	@go fmt ./...
	@REUSE_MONGO_CONTAINER=true go test -v ./...
	@golangci-lint run --fix

# Очистка
.PHONY: clean
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Docker команды
.PHONY: docker-build
docker-build:
	@echo "Building Docker images..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) build

.PHONY: docker-up
docker-up:
	@echo "Starting Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up --build

.PHONY: docker-up-d
docker-up-d:
	@echo "Starting Docker containers in detached mode..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: docker-down
docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down

.PHONY: docker-logs
docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

# Observability команды
.PHONY: observability-up
observability-up:
	@echo "Starting observability stack..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability up -d

.PHONY: observability-down
observability-down:
	@echo "Stopping observability stack..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability down

.PHONY: observability-logs
observability-logs:
	@echo "Showing observability logs..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability logs -f

# Комбинированные команды
.PHONY: dev-up
dev-up:
	@echo "Starting development environment with observability..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability up -d mongodb mongo-express redis jaeger prometheus grafana

.PHONY: full-up
full-up:
	@echo "Starting full stack (app + observability)..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile production --profile observability up -d

# Генерация OpenAPI кода
.PHONY: generate
generate:
	@echo "Generating OpenAPI code..."
	@go generate ./...

# Справка
.PHONY: help
help:
	@echo "Available commands:"
	@echo "  build            - Build the application (outputs to ./build/family-budget-service)"
	@echo "  run              - Run the application directly with go run"
	@echo "  run-local        - Run with local development environment variables (requires make dev-up first)"
	@echo ""
	@echo "Testing and Code Quality:"
	@echo "  test             - Run tests (excluding performance)"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-unit        - Unit tests without containers"
	@echo "  lint             - Run golangci-lint for comprehensive code quality checks"
	@echo "  fmt              - Format code with go fmt"
	@echo ""
	@echo "Dependencies and Maintenance:"
	@echo "  deps             - Download and tidy Go modules"
	@echo "  clean            - Remove build artifacts and coverage reports"
	@echo "  generate         - Generate OpenAPI code"
	@echo ""
	@echo "Docker Environment:"
	@echo "  dev-up           - Start development environment (MongoDB + Redis + Mongo Express)"
	@echo "  docker-up        - Start basic Docker containers (MongoDB, Redis)"
	@echo "  docker-down      - Stop Docker containers"
	@echo "  docker-logs      - View Docker container logs"
	@echo "  observability-up - Start observability stack (Prometheus, Grafana, Jaeger)"
	@echo "  full-up          - Start complete stack (app + observability)"
	@echo ""
	@echo "Other commands:"
	@echo "  help             - Show this help"
