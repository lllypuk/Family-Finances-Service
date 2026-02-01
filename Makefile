# Переменные
APP_NAME=family-budget-service
BUILD_DIR=./build
DATA_DIR=./data
DOCKER_COMPOSE_FILE=docker/docker-compose.yml

# Сборка приложения
.PHONY: build
build:
	@echo "Building $(APP_NAME)..."
	@mkdir -p $(BUILD_DIR)
	@CGO_ENABLED=0 go build -ldflags="-w -s" -o $(BUILD_DIR)/$(APP_NAME) ./cmd/server

# Запуск приложения
.PHONY: run
run:
	@echo "Running $(APP_NAME)..."
	@go run ./cmd/server/main.go

# Запуск с локальными переменными окружения для SQLite
.PHONY: run-local
run-local:
	@echo "Running $(APP_NAME) with local SQLite config..."
	@mkdir -p $(DATA_DIR)
	@SERVER_PORT=8080 \
	 SERVER_HOST=localhost \
	 DATABASE_PATH=$(DATA_DIR)/budget.db \
	 SESSION_SECRET=your-super-secret-session-key-for-local-dev \
	 LOG_LEVEL=debug \
	 ENVIRONMENT=development \
	 go run ./cmd/server/main.go

# Тесты с SQLite (in-memory)
.PHONY: test
test:
	@echo "Running tests with SQLite in-memory..."
	@go test -v ./...

# Юнит тесты
.PHONY: test-unit
test-unit:
	@echo "Running unit tests..."
	@go test -v ./internal/...

# Интеграционные тесты
.PHONY: test-integration
test-integration:
	@echo "Running integration tests..."
	@go test -v ./tests/...

# Тесты с покрытием
.PHONY: test-coverage
test-coverage:
	@echo "Running tests with coverage..."
	@go test -coverprofile=coverage.out ./...
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
	@go test -v ./...
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
	@echo "Building Docker image..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) build

.PHONY: docker-up
docker-up:
	@echo "Starting Docker container..."
	@mkdir -p $(DATA_DIR)
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up

.PHONY: docker-up-d
docker-up-d:
	@echo "Starting Docker container in detached mode..."
	@mkdir -p $(DATA_DIR)
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d

.PHONY: docker-down
docker-down:
	@echo "Stopping Docker containers..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down

.PHONY: docker-logs
docker-logs:
	@echo "Showing Docker logs..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f

# SQLite специфичные команды
.PHONY: sqlite-backup
sqlite-backup:
	@echo "Creating SQLite backup..."
	@mkdir -p ./backups
	@cp $(DATA_DIR)/budget.db ./backups/budget_$(shell date +%Y%m%d_%H%M%S).db
	@echo "Backup created in ./backups/"

.PHONY: sqlite-restore
sqlite-restore:
	@echo "Restoring SQLite from backup..."
	@echo "Usage: make sqlite-restore BACKUP_FILE=./backups/budget_YYYYMMDD_HHMMSS.db"
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "Error: BACKUP_FILE is required"; \
		exit 1; \
	fi
	@cp $(BACKUP_FILE) $(DATA_DIR)/budget.db
	@echo "Database restored from $(BACKUP_FILE)"

.PHONY: sqlite-shell
sqlite-shell:
	@echo "Opening SQLite shell..."
	@sqlite3 $(DATA_DIR)/budget.db

.PHONY: sqlite-stats
sqlite-stats:
	@echo "Showing SQLite database statistics..."
	@sqlite3 $(DATA_DIR)/budget.db "SELECT name, \
		(SELECT COUNT(*) FROM sqlite_master WHERE type='index' AND tbl_name=m.name) as indexes \
		FROM sqlite_master m WHERE type='table' ORDER BY name;"

# Создание новой миграции
.PHONY: migrate-create
migrate-create:
	@echo "Creating new migration..."
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@TIMESTAMP=$$(date +%s); \
	touch migrations/$${TIMESTAMP}_$(NAME).up.sql; \
	touch migrations/$${TIMESTAMP}_$(NAME).down.sql; \
	echo "Created migrations/$${TIMESTAMP}_$(NAME).up.sql"; \
	echo "Created migrations/$${TIMESTAMP}_$(NAME).down.sql"

# Безопасность и валидация
.PHONY: security-check
security-check:
	@echo "Running security checks..."
	@gosec ./...
	@govulncheck ./...

# Генерация OpenAPI кода
.PHONY: generate
generate:
	@echo "Generating OpenAPI code..."
	@go generate ./...

# Документация
.PHONY: docs
docs:
	@echo "Generating documentation..."
	@godoc -http=:6060
	@echo "Documentation available at http://localhost:6060"

# Справка
.PHONY: help
help:
	@echo "Available commands:"
	@echo ""
	@echo "Building and Running:"
	@echo "  build            - Build the application (outputs to ./build/family-budget-service)"
	@echo "  run              - Run the application directly with go run"
	@echo "  run-local        - Run with local SQLite database (./data/budget.db)"
	@echo ""
	@echo "Testing and Code Quality:"
	@echo "  test             - Run tests with SQLite in-memory"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-unit        - Unit tests"
	@echo "  test-integration - Integration tests"
	@echo "  lint             - Run golangci-lint for comprehensive code quality checks"
	@echo "  fmt              - Format code with go fmt"
	@echo "  pre-commit       - Run pre-commit checks (format, test, lint)"
	@echo "  security-check   - Run security analysis with gosec and govulncheck"
	@echo ""
	@echo "Dependencies and Maintenance:"
	@echo "  deps             - Download and tidy Go modules"
	@echo "  clean            - Remove build artifacts and coverage reports"
	@echo "  generate         - Generate OpenAPI code"
	@echo "  docs             - Start documentation server"
	@echo ""
	@echo "SQLite Database:"
	@echo "  sqlite-backup    - Create SQLite backup"
	@echo "  sqlite-restore   - Restore from backup (BACKUP_FILE=path required)"
	@echo "  sqlite-shell     - Open SQLite interactive shell"
	@echo "  sqlite-stats     - Show database statistics"
	@echo ""
	@echo "Database Migrations:"
	@echo "  migrate-create   - Create new migration (NAME=migration_name required)"
	@echo ""
	@echo "Docker Environment:"
	@echo "  docker-build     - Build Docker image"
	@echo "  docker-up        - Start Docker container"
	@echo "  docker-up-d      - Start Docker container in detached mode"
	@echo "  docker-down      - Stop Docker containers"
	@echo "  docker-logs      - View Docker container logs"
	@echo ""
	@echo "Other commands:"
	@echo "  help             - Show this help"
