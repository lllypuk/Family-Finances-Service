# Family Finances Service

A comprehensive family budget management system with income/expense tracking, budgeting, and analytics capabilities.

## Features

- 📊 **Family Budget Management**: Track income and expenses for all family members
- 👥 **Multi-User Support**: Role-based access (Admin, Member, Child, View-Only)
- 💰 **Budget Planning**: Set limits by categories and periods
- 📈 **Analytics & Reports**: Detailed insights into spending patterns
- 🎯 **Financial Goals**: Set and track savings targets
- 🔄 **Multi-Platform**: REST API, Web interface (HTMX + PicoCSS), Android app

## Tech Stack

- **Backend**: Go 1.24+ with Echo v4 framework
- **Database**: MongoDB 7.0+
- **Web UI**: HTMX + PicoCSS
- **Mobile**: Android (Kotlin)
- **Containerization**: Docker & Docker Compose

## Quick Start

### Prerequisites
- Go 1.24+
- Docker & Docker Compose
- Make

### Local Development Setup

1. **Start required services**:
   ```bash
   make dev-up  # Starts MongoDB, Redis, and Mongo Express
   ```

2. **Run the application**:
   ```bash
   make run-local  # Runs on localhost:8080
   ```

3. **Access the services**:
   - **Application**: http://localhost:8080
   - **Mongo Express** (DB Admin): http://localhost:8081 (admin/admin)
   - **MongoDB**: localhost:27017 (admin/password123)
   - **Redis**: localhost:6379 (password: redis123)

### Development Commands

```bash
# Development environment
make dev-up           # Start MongoDB, Redis, Mongo Express
make run-local        # Run with development config
make docker-down      # Stop all containers

# Testing and Quality
make test             # Run tests
make test-coverage    # Run tests with coverage report
make lint             # Run linter
make fmt              # Format code

# Building
make build            # Build binary
make clean            # Clean build artifacts

# Observability (optional)
make observability-up # Start Prometheus, Grafana, Jaeger
```

## Architecture

This project follows **Clean Architecture** principles:

- **Domain Layer**: Business entities and rules (`internal/domain/`)
- **Application Layer**: Use cases and interfaces (`internal/application/`)
- **Infrastructure Layer**: External services implementation (`internal/infrastructure/`)
- **Web Layer**: HTTP handlers and middleware (`internal/web/`)

### Project Structure
```
├── cmd/server/          # Application entry point
├── internal/
│   ├── domain/          # Business entities (User, Family, Transaction, etc.)
│   ├── application/     # Use cases and repository interfaces
│   ├── web/             # HTTP handlers, middleware, templates
│   └── infrastructure/  # Database implementations (planned)
├── .memory_bank/        # Project documentation
└── monitoring/          # Observability configuration
```

## Configuration

The application uses environment variables for configuration. Key variables:

| Variable           | Default                   | Description                              |
|--------------------|---------------------------|------------------------------------------|
| `SERVER_PORT`      | 8080                      | HTTP server port                         |
| `SERVER_HOST`      | localhost                 | HTTP server host                         |
| `MONGODB_URI`      | mongodb://localhost:27017 | MongoDB connection string                |
| `MONGODB_DATABASE` | family_budget             | Database name                            |
| `SESSION_SECRET`   | (required)                | Session encryption key                   |
| `REDIS_URL`        | (optional)                | Redis connection string                  |
| `LOG_LEVEL`        | info                      | Logging level (debug, info, warn, error) |
| `ENVIRONMENT`      | production                | Application environment                  |

## Запуск с Docker

```bash
# Сборка и запуск всех сервисов
docker-compose -f docker/docker-compose.yml up --build

# Запуск в фоне
docker-compose -f docker/docker-compose.yml up -d

# Остановка сервисов
docker-compose -f docker/docker-compose.yml down
```

## Разработка

```bash
# Только база данных для разработки
docker-compose -f docker/docker-compose.yml up postgres -d
```

## Development

See [CLAUDE.md](CLAUDE.md) for comprehensive development guidelines and architecture details.

For detailed project documentation, check the [.memory_bank](.memory_bank) directory.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
