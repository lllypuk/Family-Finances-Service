# Family Finances Service

**Production-ready family budget management system** with comprehensive web interface, REST API, and advanced security features.

## 🎯 Project Status: PRODUCTION READY ✅

This project is **fully implemented and ready for production deployment** with:
- ✅ Complete web interface (HTMX + PicoCSS)
- ✅ Full REST API with comprehensive endpoints
- ✅ Advanced security (authentication, authorization, CSRF protection)
- ✅ **59.5%+ test coverage** with 450+ tests
- ✅ CI/CD pipelines with GitHub Actions
- ✅ Observability stack (Prometheus, Grafana, Jaeger)
- ✅ Multi-platform Docker builds

## 🚀 Features

- 📊 **Complete Web Interface**: Modern HTMX-powered UI with responsive design
- 👥 **Role-Based Access Control**: Family Admin, Member, Child with different permissions
- 💰 **Advanced Budget Management**: Category limits, period tracking, overspend alerts
- 📈 **Real-Time Analytics**: Interactive dashboards with live updates
- 🎯 **Financial Goals Tracking**: Savings targets with progress visualization
- 🔐 **Enterprise Security**: Session management, CSRF protection, input validation
- 📊 **Comprehensive Reporting**: Export capabilities, trend analysis
- 🌐 **Multi-Platform Ready**: REST API, Web interface, mobile-ready design

## 🏗️ Architecture & Tech Stack

### Backend (Production-Ready)
- **Go 1.24+** with Echo v4.13.4+ framework
- **MongoDB 7.0+** with official Go driver v1.17.4+
- **Clean Architecture** with domain-driven design
- **Repository pattern** with comprehensive error handling
- **Structured logging** with slog + observability

### Frontend (Modern Web Interface)
- **HTMX v1.9+** for dynamic updates without complex JavaScript
- **PicoCSS v1.5+** minimalist CSS framework
- **Go Templates** with layout system and components
- **Progressive Web App** capabilities
- **Responsive design** for mobile and desktop

### DevOps & Quality
- **Docker & Docker Compose** for containerization
- **GitHub Actions** CI/CD with security scanning
- **Multi-platform builds** (linux/amd64, linux/arm64)
- **Comprehensive testing** with testcontainers-go
- **Security scanning** (CodeQL, Semgrep, TruffleHog)

### Observability Stack
- **Prometheus** metrics collection
- **Grafana** dashboards and visualization
- **Jaeger** distributed tracing
- **Health checks** (liveness/readiness probes)
- **Structured logging** with multiple output formats

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
   make run  # Runs on localhost:8080
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
make run              # Run with development config
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

## 🏛️ Architecture

This project follows **Clean Architecture** principles with production-ready implementations:

### Layer Structure
- **Domain Layer** (`internal/domain/`): Business entities with comprehensive validation
- **Application Layer** (`internal/application/`): HTTP server and handler orchestration
- **Web Layer** (`internal/web/`): HTMX templates, middleware, authentication
- **Infrastructure Layer** (`internal/infrastructure/`): MongoDB repositories and data persistence
- **Observability Layer** (`internal/observability/`): Metrics, logging, tracing, health checks

### Project Structure
```
├── cmd/server/              # Application entry point with health checks
├── internal/
│   ├── domain/              # Business entities (User, Family, Transaction, Budget, Report)
│   ├── application/         # HTTP server, handlers, repository interfaces
│   ├── web/                 # Complete web interface
│   │   ├── handlers/        # Authentication, dashboard, HTMX endpoints
│   │   ├── middleware/      # Session, CSRF, auth guards
│   │   ├── templates/       # HTML templates with layouts
│   │   ├── static/          # CSS, JS, images
│   │   └── models/          # Form validation structures
│   ├── infrastructure/      # MongoDB repositories and connection management
│   ├── observability/       # Production monitoring and logging
│   └── testhelpers/         # Testing utilities and factories
├── tests/                   # E2E and integration tests
│   ├── e2e/                # End-to-end workflow tests
│   ├── integration/        # Cross-component integration tests
│   └── performance/        # Load testing and benchmarks
├── .memory_bank/           # Comprehensive project documentation
├── monitoring/             # Grafana dashboards, Prometheus config, alerting
└── .github/workflows/      # CI/CD pipelines (ci, docker, security, release)
```

### Production Components
- **Authentication & Authorization**: Role-based access with session management
- **Data Validation**: Comprehensive input validation with go-playground/validator
- **Error Handling**: Structured error responses with proper HTTP status codes
- **Security**: CSRF protection, password hashing, input sanitization
- **Testing**: 450+ tests with 59.5% coverage across all layers
- **Observability**: Prometheus metrics, structured logging, distributed tracing
- **Deployment**: Multi-platform Docker builds with GitHub Actions CI/CD

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

## 🧪 Testing & Quality

This project maintains **high quality standards** with comprehensive testing:

### Test Coverage: 59.5%
- **450+ tests** across all application layers
- **Unit tests**: Domain models, repositories, handlers with mocking
- **Integration tests**: End-to-end workflows with testcontainers
- **Performance tests**: Load testing, memory profiling, benchmark testing
- **Security tests**: Authentication, authorization, CSRF, input validation
- **E2E tests**: Complete user journeys from registration to reporting

### Quality Assurance
- **golangci-lint** with 50+ linters for code quality
- **Comprehensive CI/CD** with GitHub Actions
- **Security scanning** (CodeQL, Semgrep, TruffleHog, OSV Scanner)
- **Dependency management** with Dependabot automated updates
- **Multi-platform testing** (linux/amd64, linux/arm64)

```bash
# Run comprehensive test suite
make test              # All tests
make test-coverage    # With coverage report
make lint             # Code quality checks
```

## 📊 Production Readiness

### Deployment Ready
- ✅ **Multi-platform Docker images** published to GitHub Container Registry
- ✅ **Kubernetes ready** with health checks and graceful shutdown
- ✅ **Environment configuration** with validation and defaults
- ✅ **Database migrations** and connection management
- ✅ **Observability stack** with metrics, logs, and traces

### Monitoring & Observability
- ✅ **Prometheus metrics** - HTTP, database, business metrics
- ✅ **Grafana dashboards** - Application overview, business metrics, SLI/SLO
- ✅ **Jaeger tracing** - Request flow and performance analysis
- ✅ **Health checks** - Liveness and readiness probes
- ✅ **Structured logging** - JSON format with correlation IDs

### Security Features
- ✅ **Role-based access control** with family isolation
- ✅ **Session security** with HTTP-only cookies and CSRF protection
- ✅ **Input validation** and sanitization for all endpoints
- ✅ **Password security** with bcrypt hashing
- ✅ **Security headers** and modern security practices

## 📚 Documentation

### Developer Resources
- **[CLAUDE.md](CLAUDE.md)** - Comprehensive development guidelines and architecture
- **[.memory_bank](.memory_bank)** - Detailed project documentation including:
  - Product brief and business context
  - Technical architecture and design decisions
  - Testing strategy and implementation details
  - Current project status and roadmap

### API Documentation
- **REST API** with comprehensive endpoint coverage
- **OpenAPI 3.0** specification (available via `/api/docs`)
- **Postman collection** for API testing and integration

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
