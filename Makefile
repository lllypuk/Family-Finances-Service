.PHONY: build run test clean docker-up docker-down deps lint fmt lint-fix

# Переменные
APP_NAME=family-budget-service
BUILD_DIR=./build
DOCKER_COMPOSE_FILE=docker-compose.yml

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
	 MONGODB_URI=mongodb://localhost:27017 \
	 MONGODB_DATABASE=family_budget_local \
	 go run ./cmd/server/main.go

# Тестирование
test:
	@echo "Running tests..."
	@go test -v ./...

# Тестирование с покрытием
test-coverage:
	@echo "Running tests with coverage..."
	@go test -v -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html

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

# Очистка
clean:
	@echo "Cleaning..."
	@rm -rf $(BUILD_DIR)
	@rm -f coverage.out coverage.html

# Docker команды
docker-up:
	@echo "Starting Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down

docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

# Генерация OpenAPI кода
generate:
	@echo "Generating OpenAPI code..."
	@go generate ./...

# Миграции базы данных (для будущего использования)
migrate-up:
	@echo "Running database migrations..."
	# TODO: Добавить команды миграций

migrate-down:
	@echo "Rolling back database migrations..."
	# TODO: Добавить команды отката миграций

# Справка
help:
	@echo "Available commands:"
	@echo "  build        - Build the application"
	@echo "  run          - Run the application"
	@echo "  run-local    - Run with local environment variables"
	@echo "  test         - Run tests"
	@echo "  test-coverage- Run tests with coverage report"
	@echo "  deps         - Install dependencies"
	@echo "  lint         - Run linter"
	@echo "  lint-fix     - Run linter with auto-fix"
	@echo "  fmt          - Format code"
	@echo "  clean        - Clean build artifacts"
	@echo "  docker-up    - Start Docker containers"
	@echo "  docker-down  - Stop Docker containers"
	@echo "  docker-logs  - Show Docker logs"
	@echo "  generate     - Generate OpenAPI code"
	@echo "  help         - Show this help"
