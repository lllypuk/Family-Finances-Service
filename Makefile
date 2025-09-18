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

# Запуск с локальными переменными окружения для PostgreSQL
.PHONY: run-local
run-local:
	@echo "Running $(APP_NAME) with local PostgreSQL config..."
	@SERVER_PORT=8080 \
	 SERVER_HOST=localhost \
	 POSTGRESQL_URI=postgres://postgres:postgres123@localhost:5432/family_budget?sslmode=disable \
	 POSTGRESQL_DATABASE=family_budget \
	 POSTGRESQL_SCHEMA=family_budget \
	 SESSION_SECRET=your-super-secret-session-key-for-local-dev \
	 CSRF_SECRET=your-csrf-secret-key-for-local-dev \
	 LOG_LEVEL=debug \
	 ENVIRONMENT=development \
	 go run ./cmd/server/main.go

# Быстрые тесты с переиспользованием PostgreSQL контейнера
.PHONY: test
test:
	@echo "Running fast tests with shared PostgreSQL container..."
	@REUSE_POSTGRES_CONTAINER=true go test -v ./...

# Юнит тесты с быстрыми контейнерами
.PHONY: test-unit
test-unit:
	@echo "Running unit tests with fast containers..."
	@REUSE_POSTGRES_CONTAINER=true go test -v ./internal/...

# Интеграционные тесты с переиспользованием контейнера
.PHONY: test-integration
test-integration:
	@echo "Running integration tests with shared container..."
	@REUSE_POSTGRES_CONTAINER=true go test -v ./tests/...

# Быстрые тесты с покрытием и переиспользованием контейнера
.PHONY: test-coverage
test-coverage:
	@echo "Running fast tests with coverage and shared PostgreSQL container..."
	@REUSE_POSTGRES_CONTAINER=true go test -coverprofile=coverage.out ./...
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
	@REUSE_POSTGRES_CONTAINER=true go test -v ./...
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

# PostgreSQL специфичные команды
.PHONY: postgres-up
postgres-up:
	@echo "Starting PostgreSQL and related services..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) up -d postgresql

.PHONY: postgres-down
postgres-down:
	@echo "Stopping PostgreSQL and related services..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) down postgresql

.PHONY: postgres-logs
postgres-logs:
	@echo "Showing PostgreSQL logs..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) logs -f postgresql

.PHONY: postgres-shell
postgres-shell:
	@echo "Connecting to PostgreSQL shell..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgresql psql -U postgres -d family_budget

.PHONY: postgres-backup
postgres-backup:
	@echo "Creating PostgreSQL backup..."
	@mkdir -p ./backups
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgresql pg_dump -U postgres -d family_budget > ./backups/family_budget_$(shell date +%Y%m%d_%H%M%S).sql
	@echo "Backup created in ./backups/"

.PHONY: postgres-restore
postgres-restore:
	@echo "Restoring PostgreSQL from backup..."
	@echo "Usage: make postgres-restore BACKUP_FILE=./backups/family_budget_YYYYMMDD_HHMMSS.sql"
	@if [ -z "$(BACKUP_FILE)" ]; then \
		echo "Error: BACKUP_FILE is required"; \
		exit 1; \
	fi
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec -T postgresql psql -U postgres -d family_budget < $(BACKUP_FILE)

# Миграции базы данных
.PHONY: migrate-up
migrate-up:
	@echo "Running database migrations (docker)..."
	@docker run --rm \
	 -v "$(shell pwd)/migrations:/migrations" \
	 --network host \
	 migrate/migrate:latest \
	 -path=/migrations -database "postgres://postgres:postgres123@localhost:5432/family_budget?sslmode=disable" up

.PHONY: migrate-down
migrate-down:
	@echo "Rolling back database migrations (docker)..."
	@docker run --rm \
	 -v "$(shell pwd)/migrations:/migrations" \
	 --network host \
	 migrate/migrate:latest \
	 -path=/migrations -database "postgres://postgres:postgres123@localhost:5432/family_budget?sslmode=disable" down

.PHONY: migrate-create
migrate-create:
	@echo "Creating new migration..."
	@if [ -z "$(NAME)" ]; then \
		echo "Usage: make migrate-create NAME=migration_name"; \
		exit 1; \
	fi
	@docker run --rm \
	 -v "$(shell pwd)/migrations:/migrations" \
	 migrate/migrate:latest \
	 create -ext sql -dir /migrations $(NAME)

.PHONY: migrate-force
migrate-force:
	@echo "Forcing migration version..."
	@if [ -z "$(VERSION)" ]; then \
		echo "Usage: make migrate-force VERSION=version_number"; \
		exit 1; \
	fi
	@migrate -path ./migrations -database "postgres://postgres:postgres123@localhost:5432/family_budget?sslmode=disable" force $(VERSION)

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
	@echo "Starting development environment with PostgreSQL and observability..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile observability up -d postgresql jaeger prometheus grafana postgres-exporter

.PHONY: full-up
full-up:
	@echo "Starting full stack (app + observability)..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) --profile production --profile observability up -d

# Мониторинг и производительность
.PHONY: postgres-stats
postgres-stats:
	@echo "Showing PostgreSQL performance statistics..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgresql psql -U postgres -d family_budget -c "\
		SELECT schemaname, tablename, \
		pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size, \
		pg_stat_get_tuples_returned(c.oid) as rows_fetched, \
		pg_stat_get_tuples_inserted(c.oid) as rows_inserted, \
		pg_stat_get_tuples_updated(c.oid) as rows_updated, \
		pg_stat_get_tuples_deleted(c.oid) as rows_deleted \
		FROM pg_tables LEFT JOIN pg_class c ON c.relname = tablename \
		WHERE schemaname = 'family_budget' ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC;"

.PHONY: postgres-indexes
postgres-indexes:
	@echo "Showing PostgreSQL index usage..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgresql psql -U postgres -d family_budget -c "\
		SELECT schemaname, tablename, indexname, idx_tup_read, idx_tup_fetch \
		FROM pg_stat_user_indexes \
		WHERE schemaname = 'family_budget' AND idx_tup_read > 0 \
		ORDER BY idx_tup_read DESC;"

.PHONY: postgres-slow-queries
postgres-slow-queries:
	@echo "Showing slow queries..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgresql psql -U postgres -d family_budget -c "\
		SELECT query, calls, total_exec_time, mean_exec_time, max_exec_time \
		FROM pg_stat_statements \
		WHERE mean_exec_time > 100 \
		ORDER BY mean_exec_time DESC LIMIT 10;"

# Безопасность и валидация
.PHONY: security-check
security-check:
	@echo "Running security checks..."
	@gosec ./...
	@govulncheck ./...

.PHONY: validate-schema
validate-schema:
	@echo "Validating database schema..."
	@docker-compose -f $(DOCKER_COMPOSE_FILE) exec postgresql psql -U postgres -d family_budget -c "\
		SELECT schemaname, tablename, \
		CASE WHEN c.relchecks > 0 THEN 'Has constraints' ELSE 'No constraints' END as constraints \
		FROM pg_tables LEFT JOIN pg_class c ON c.relname = tablename \
		WHERE schemaname = 'family_budget';"

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
	@echo "  run-local        - Run with local development environment variables (requires make dev-up first)"
	@echo ""
	@echo "Testing and Code Quality:"
	@echo "  test             - Run tests with shared PostgreSQL container (fast)"
	@echo "  test-coverage    - Run tests with coverage report"
	@echo "  test-unit        - Unit tests with fast containers"
	@echo "  test-integration - Integration tests with shared container"
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
	@echo "PostgreSQL Environment:"
	@echo "  postgres-up      - Start PostgreSQL"
	@echo "  postgres-down    - Stop PostgreSQL and related services"
	@echo "  postgres-logs    - View PostgreSQL container logs"
	@echo "  postgres-shell   - Connect to PostgreSQL shell"
	@echo "  postgres-backup  - Create PostgreSQL backup"
	@echo "  postgres-restore - Restore from backup (BACKUP_FILE=path required)"
	@echo ""
	@echo "Database Migrations:"
	@echo "  migrate-up       - Run database migrations"
	@echo "  migrate-down     - Rollback database migrations"
	@echo "  migrate-create   - Create new migration (NAME=migration_name required)"
	@echo "  migrate-force    - Force migration version (VERSION=number required)"
	@echo ""
	@echo "Docker Environment:"
	@echo "  dev-up           - Start development environment (PostgreSQL + Observability)"
	@echo "  docker-up        - Start basic Docker containers"
	@echo "  docker-down      - Stop Docker containers"
	@echo "  docker-logs      - View Docker container logs"
	@echo "  observability-up - Start observability stack (Prometheus, Grafana, Jaeger)"
	@echo "  full-up          - Start complete stack (app + observability)"
	@echo ""
	@echo "Monitoring and Performance:"
	@echo "  postgres-stats   - Show PostgreSQL table statistics"
	@echo "  postgres-indexes - Show index usage statistics"
	@echo "  postgres-slow-queries - Show slow query analysis"
	@echo "  validate-schema  - Validate database schema integrity"
	@echo ""
	@echo "Other commands:"
	@echo "  help             - Show this help"
