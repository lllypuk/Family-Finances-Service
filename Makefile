.PHONY: build run test test-all test-unit test-integration test-e2e test-performance clean docker-up docker-down deps lint fmt pre-commit lint-fix observability-up observability-down observability-logs

# Переменные
APP_NAME=family-budget-service
BUILD_DIR=./build
DOCKER_COMPOSE_FILE=docker/docker-compose.yml

# Сборка приложения
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@go build -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

# Запуск приложения
run:
	@echo "Running $(APP_NAME)..."
	@go run ./cmd/server/main.go

# Запуск с локальными переменными окружения
run-local:
	@echo "Running $(APP_NAME) with local config..."
	@SERVER_PORT=8080 \
	 SERVER_HOST=localhost \
	 MONGODB_URI=mongodb://admin:password123@localhost:27017/family_budget_local?authSource=admin \
	 MONGODB_DATABASE=family_budget_local \
	 SESSION_SECRET=your-super-secret-session-key-for-local-dev \
	 REDIS_URL=redis://:redis123@localhost:6379 \
	 LOG_LEVEL=debug \
	 ENVIRONMENT=development \
	 go run ./cmd/server/main.go

# Основные тесты (исключая производительность)
test:
	@echo "Running tests (excluding performance)..."
	@go test -v $$(go list ./... | grep -v '/tests/performance')

# Юнит тесты (internal/)
test-unit:
	@echo "Running unit tests..."
	@go test -v ./internal/...

# Инgo test -v $$(go list ./... | grep -v '/tests/performance')теграционные тесты
test-integration:
	@echo "Running integration tests..."
	@go test -v ./tests/integration/...

# E2E тесты
test-e2e:
	@echo "Running e2e tests..."
	@go test -v ./tests/e2e/...

# Тесты производительности
test-performance:
	@echo "Running performance tests..."
	@go test -v ./tests/performance/...

# Все тесты включая производительность
test-all:
	@echo "Running all tests including performance..."
	@go test -v ./...

# Тесты с покрытием (исключая производительность)
test-coverage:
	@echo "Running tests with coverage (excluding performance)..."
	@go test -v -coverprofile=coverage.out $$(go list ./... | grep -v '/tests/performance')
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# Установка зависимостей
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Линтер
lint:
	@echo "Running linter..."
	@golangci-lint run

# Автоматическое исправление ошибок линтера
lint-fix:
	@echo "Running linter with auto-fix..."
	@golangci-lint run --fix

# Форматирование кода
fmt:
	@echo "Formatting code..."
	@go fmt ./...

# Проверка перед коммитом
pre-commit:
	@echo "Running pre-commit checks..."
	@go fmt ./...
	@go test -v $$(go list ./... | grep -v '/tests/performance')
	@golangci-lint run --fix

# Очистка
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Docker команды
docker-build:
	@echo "Building Docker images..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) build

docker-up:
	@echo "Starting Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up --build

docker-up-d:
	@echo "Starting Docker containers in detached mode..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down

docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

# Observability команды
observability-up:
	@echo "Starting observability stack..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability up -d

observability-down:
	@echo "Stopping observability stack..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability down

observability-logs:
	@echo "Showing observability logs..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability logs -f

# Комбинированные команды
dev-up:
	@echo "Starting development environment..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d mongodb mongo-express redis

full-up:
	@echo "Starting full stack (app + observability)..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile production --profile observability up -d

# Генерация OpenAPI кода
generate:
	@echo "Generating OpenAPI code..."
	@go generate ./...

# Справка
help:
	@echo "Available commands:"
	@echo "  build            - Build the application"
	@echo "  run              - Run the application"
	@echo "  run-local        - Run with local environment variables"
	@echo "  test             - Run unit tests (internal/ only)"
	@echo "  test-all         - Run all tests"
	@echo "  test-unit        - Run unit tests (internal/)"
	@echo "  test-integration - Run integration tests (tests/integration/)"
	@echo "  test-e2e         - Run e2e tests (tests/e2e/)"
	@echo "  test-performance - Run performance tests (tests/performance/)"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Run linter"
	@echo "  lint-fix         - Run linter with auto-fix"
	@echo "  fmt              - Format code"
	@echo "  clean            - Clean build artifacts"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-build      - Build Docker images"
	@echo "  docker-up        - Start Docker containers"
	@echo "  docker-up-d      - Start Docker containers in detached mode"
	@echo "  docker-down      - Stop Docker containers"
	@echo "  docker-logs      - Show Docker logs"
	@echo "  dev-up           - Start development environment"
	@echo "  full-up          - Start full stack (app + observability)"
	@echo ""
	@echo "Observability commands:"
	@echo "  observability-up - Start observability stack (Prometheus, Grafana, etc.)"
	@echo "  observability-down - Stop observability stack"
	@echo "  observability-logs - Show observability logs"
	@echo ""
	@echo "Other commands:"
	@echo "  generate         - Generate OpenAPI code"
	@echo "  help             - Show this help"
