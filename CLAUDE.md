# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
- `make build` - Build the application (outputs to ./build/family-budget-service)
- `make run` - Run the application directly with go run
- `make run-local` - Run with local development environment variables (PORT=8080, MongoDB on localhost:27017)

### Testing and Code Quality
- `make test` - Run all tests
- `make test-coverage` - Run tests with coverage report (generates coverage.html)
- `make lint` - Run golangci-lint for code quality checks
- `make fmt` - Format code with go fmt

### Dependencies and Maintenance
- `make deps` - Download and tidy Go modules
- `make clean` - Remove build artifacts and coverage reports
- `make generate` - Generate OpenAPI code

### Docker Environment
- `make docker-up` - Start MongoDB and supporting services via Docker Compose
- `make docker-down` - Stop Docker containers
- `make docker-logs` - View Docker container logs

The Docker environment includes:
- MongoDB (port 27017) with admin/password123 credentials
- Mongo Express web UI (port 8081) for database administration
- Redis (port 6379) for caching with password "redis123"

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
- `SERVER_PORT` / `SERVER_HOST` - Server configuration
- `MONGODB_URI` / `MONGODB_DATABASE` - Database connection
- `REDIS_URL` - Cache configuration (optional)

### Repository Pattern
All data access is abstracted through repository interfaces defined in `internal/application/interfaces.go`. Repository implementations are planned for the `internal/infrastructure/` directory.

### Multi-tenancy
The system supports family-based multi-tenancy where users belong to families and data is isolated by family ID.