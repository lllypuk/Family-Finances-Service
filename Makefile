.PHONY: build run test test-all test-unit test-integration test-e2e test-e2e-fast test-e2e-playwright test-e2e-playwright-ui test-e2e-playwright-debug test-performance test-coverage test-ci clean docker-build docker-up docker-up-d docker-down docker-logs deps lint fmt pre-commit dev-up full-up observability-up observability-down observability-logs

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

# Быстрые тесты с переиспользованием MongoDB контейнера
test:
	@echo "Running fast tests with shared MongoDB container..."
	@REUSE_MONGO_CONTAINER=true go test -v $$(go list ./... | grep -v '/tests/performance')

# Юнит тесты без контейнеров (только internal/, исключая integration тесты)
test-unit:
	@echo "Running unit tests without containers..."
	@go test -v $$(go list ./internal/... | grep -v 'infrastructure')

# Интеграционные тесты
test-integration:
	@echo "Running integration tests with shared container..."
	@REUSE_MONGO_CONTAINER=true go test -v ./tests/integration/...

# E2E тесты (Go)
test-e2e:
	@echo "Running e2e tests..."
	@go test -v ./tests/e2e/...

# E2E тесты с переиспользованием контейнера (Go)
test-e2e-fast:
	@echo "Running e2e tests with shared container..."
	@REUSE_MONGO_CONTAINER=true go test -v ./tests/e2e/...

# Playwright E2E тесты
test-e2e-playwright:
	@echo "Running Playwright E2E tests..."
	@npm run test:e2e

# Playwright E2E тесты с UI
test-e2e-playwright-ui:
	@echo "Running Playwright E2E tests with UI..."
	@npm run test:e2e:ui

# Playwright E2E тесты с отладкой
test-e2e-playwright-debug:
	@echo "Running Playwright E2E tests in debug mode..."
	@npm run test:e2e:debug

# Тесты производительности
test-performance:
	@echo "Running performance tests..."
	@go test -v ./tests/performance/...

# Все тесты включая производительность
test-all:
	@echo "Running all tests including performance..."
	@go test -v ./...

# Быстрые тесты с покрытием и переиспользованием контейнера
test-coverage:
	@echo "Running fast tests with coverage and shared MongoDB container..."
	@REUSE_MONGO_CONTAINER=true go test -coverprofile=coverage.out $$(go list ./... | grep -v '/tests/performance')
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

# CI-оптимизированные тесты (параллельные + fast)
test-ci:
	@echo "Running CI-optimized tests..."
	@REUSE_MONGO_CONTAINER=true go test -short -race -coverprofile=coverage.out $$(go list ./... | grep -v '/tests/performance')
	@go tool cover -html=coverage.out -o coverage.html

# Установка зависимостей
deps:
	@echo "Installing dependencies..."
	@go mod download
	@go mod tidy

# Линтер
lint:
	@echo "Running linter..."
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
	@echo "Starting development environment with observability..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability up -d mongodb mongo-express redis jaeger prometheus grafana

full-up:
	@echo "Starting full stack (app + observability)..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile production --profile observability up -d
