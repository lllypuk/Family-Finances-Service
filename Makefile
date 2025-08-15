.PHONY: build run test clean docker-up docker-down deps lint fmt lint-fix observability-up observability-down observability-logs

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
	@echo "  test             - Run tests"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  deps             - Install dependencies"
	@echo "  lint             - Run linter"
	@echo "  lint-fix         - Run linter with auto-fix"
	@echo "  fmt              - Format code"
	@echo "  clean            - Clean build artifacts"
	@echo ""
	@echo "Docker commands:"
	@echo "  docker-up        - Start basic Docker containers (MongoDB, Redis)"
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
