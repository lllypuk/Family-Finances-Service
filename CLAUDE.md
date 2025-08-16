# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
- `make build` - Build the application (outputs to ./build/family-budget-service)
- `make run` - Run the application directly with go run
- `make run-local` - Run with local development environment variables (requires `make dev-up` first)

#### Local Development Setup
**Prerequisites**: Before running `make run-local`, you must start the required services:
```bash
make dev-up  # Starts MongoDB and Redis containers
make run-local  # Runs the application on localhost:8083
```

The `run-local` command sets up the following environment:
- **Server**: localhost:8083 (port 8083 to avoid conflicts)
- **MongoDB**: mongodb://admin:password123@localhost:27017 with authentication
- **Database**: family_budget_local (separate from production)
- **Redis**: redis://:redis123@localhost:6379 (optional caching)
- **Logging**: DEBUG level for development
- **Session Secret**: Development-specific secret key

### Testing and Code Quality
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report (generates coverage.html)
- `make lint` - Run golangci-lint for comprehensive code quality checks
- `make lint-fix` - Run golangci-lint with automatic fixing of issues
- `make fmt` - Format code with go fmt

#### Code Quality Tools
The project uses **golangci-lint** with a comprehensive configuration (`.golangci.yml`) that includes:
- **Static analysis**: errcheck, govet, staticcheck, ineffassign
- **Code style**: gofmt, revive, whitespace alignment
- **Complexity checks**: gocognit, funlen, cyclop
- **Security**: gosec for security vulnerabilities
- **Best practices**: gocritic, unconvert, unused
- **Testing**: testifylint for proper test assertions

Run `make test` and `make lint` before committing to ensure code quality standards.

### Dependencies and Maintenance
- `make deps` - Download and tidy Go modules
- `make clean` - Remove build artifacts and coverage reports
- `make generate` - Generate OpenAPI code

### Docker Environment
- `make dev-up` - Start development environment (MongoDB + Redis + Mongo Express)
- `make docker-up` - Start basic Docker containers (MongoDB, Redis)
- `make docker-down` - Stop Docker containers
- `make docker-logs` - View Docker container logs
- `make observability-up` - Start observability stack (Prometheus, Grafana, Jaeger)
- `make full-up` - Start complete stack (app + observability)

The Docker environment includes:
- **MongoDB** (port 27017) with admin/password123 credentials
- **Mongo Express** web UI (port 8081) for database administration
- **Redis** (port 6379) for caching with password "redis123"
- **Observability Stack**: Prometheus (9090), Grafana (3000), Jaeger (16686)

## Architecture Overview

This is a family budget management service built with Go, following Clean Architecture principles:

### Domain Structure
The application is organized into domain modules in `internal/domain/`:
- **User/Family**: User management with role-based access (admin, member, child)
- **Category**: Income and expense category management
- **Transaction**: Financial transaction tracking
- **Budget**: Budget planning and monitoring
- **Report**: Financial reporting and analytics

### Layer Architecture
- `cmd/server/main.go` - Application entry point
- `internal/run.go` - Application bootstrap and lifecycle management
- `internal/application/` - Application layer with service interfaces
- `internal/domain/` - Domain entities and business logic
- `internal/infrastructure/` - Data persistence and external services (planned)

### Key Technologies
- **Echo v4** - HTTP web framework
- **MongoDB** - Primary database with official Go driver
- **UUID** - For entity identification
- **Docker Compose** - Local development environment

### Configuration
Environment variables are managed in `internal/config.go`. Key variables:
- `SERVER_PORT` / `SERVER_HOST` - Server configuration (default: localhost:8080)
- `MONGODB_URI` / `MONGODB_DATABASE` - Database connection 
- `SESSION_SECRET` - Secret key for session management
- `REDIS_URL` - Cache configuration (optional)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `ENVIRONMENT` - Application environment (development, production)

### Repository Pattern
All data access is abstracted through repository interfaces defined in `internal/application/interfaces.go`. Repository implementations are planned for the `internal/infrastructure/` directory.

### Multi-tenancy
The system supports family-based multi-tenancy where users belong to families and data is isolated by family ID.

### Documentation
Current documentation is available in the `.memory_bank/` directory

## CI/CD Pipeline

The project uses GitHub Actions for continuous integration and deployment with the following workflows:

### CI Pipeline (`.github/workflows/ci.yml`)
Runs on every push and pull request to main/develop branches:
- **Environment Setup**: Go 1.24, MongoDB service for integration tests
- **Quality Checks**: 
  - Code formatting verification with `make fmt`
  - Comprehensive linting with golangci-lint (50+ rules)
  - Security scanning with `govulncheck`
- **Testing**: Full test suite with coverage reporting to Codecov
- **Build Verification**: Compile application and Docker image test

### Docker Build Pipeline (`.github/workflows/docker.yml`)
Triggered on version tags and releases:
- **Multi-platform**: Builds for linux/amd64 and linux/arm64
- **Container Registry**: Publishes to GitHub Container Registry
- **Security**: Trivy vulnerability scanning of images
- **Size Optimization**: Enforces 50MB image size limit
- **Versioning**: Semantic versioning with appropriate tags

### Security Pipeline (`.github/workflows/security.yml`)
Comprehensive security scanning (runs on schedule and PR):
- **CodeQL**: GitHub's semantic code analysis for Go
- **Dependency Review**: License and vulnerability checking
- **Secret Scanning**: TruffleHog for credential detection
- **SAST**: Semgrep static analysis security testing
- **OSV Scanner**: Open source vulnerability scanning
- **OSSF Scorecard**: Security posture scoring

### Release Pipeline (`.github/workflows/release.yml`)
Automated releases on version tags:
- **Multi-platform Binaries**: Linux, macOS, Windows (amd64/arm64)
- **Docker Images**: Automated publishing with semantic versioning
- **Release Notes**: Auto-generated from commit history
- **Checksums**: SHA256 validation for all binaries
- **Notifications**: Success/failure reporting

### Dependency Management
- **Dependabot**: Automated dependency updates (Go modules, GitHub Actions, Docker)
- **Security**: Automatic vulnerability detection and patching
- **Schedule**: Weekly updates on different days to manage load

### Code Review Process
- **CODEOWNERS**: Automatic reviewer assignment for critical files
- **Branch Protection**: Required status checks and reviews
- **Quality Gates**: All CI checks must pass before merge

### Monitoring and Observability
- **Coverage Reports**: Integrated with Codecov for test coverage tracking
- **Security Dashboard**: Centralized vulnerability and compliance reporting
- **Build Metrics**: Automated tracking of build times and success rates

Use `make lint` and `make test` locally to ensure CI pipeline success before pushing changes.
