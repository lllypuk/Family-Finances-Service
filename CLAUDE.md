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
- **Server**: localhost:8080 (port 8080 to avoid conflicts)
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

**IMPORTANT**: Always run `make lint` after completing any development work to ensure code quality standards. All linter errors must be fixed before committing changes.

**Workflow for development:**
1. Make your changes
2. Run `make fmt` to format code
3. Run `make test` to ensure tests pass
4. **Run `make lint` and fix ALL errors** - this is mandatory
5. Commit changes only after lint passes with 0 issues

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

This is a production-ready family budget management service built with Go, following Clean Architecture principles and comprehensive testing practices.

### Current Status: PRODUCTION READY
- ✅ **Complete web interface** with HTMX + PicoCSS
- ✅ **Full API implementation** with REST endpoints
- ✅ **Comprehensive security** with authentication & authorization
- ✅ **59.5%+ test coverage** with 450+ tests
- ✅ **CI/CD pipelines** with GitHub Actions
- ✅ **Observability stack** (Prometheus, Grafana, Jaeger)
- ✅ **Production deployment** ready

### Domain Structure
The application is organized into domain modules in `internal/domain/`:
- **User/Family**: User management with role-based access (admin, member, child)
- **Category**: Income and expense category management  
- **Transaction**: Financial transaction tracking
- **Budget**: Budget planning and monitoring
- **Report**: Financial reporting and analytics

### Layer Architecture (Clean Architecture)
- `cmd/server/main.go` - Application entry point with health checks
- `internal/run.go` - Application bootstrap and graceful shutdown
- `internal/application/` - HTTP server and handler layer
- `internal/web/` - Web interface (HTMX templates, middleware, static files)
- `internal/domain/` - Domain entities and business logic
- `internal/infrastructure/` - MongoDB repositories and data persistence
- `internal/observability/` - Metrics, logging, tracing, health checks

### Key Technologies (Production Stack)
- **Echo v4.13.4+** - HTTP web framework with middleware
- **MongoDB v1.17.4+** - Primary database with official Go driver
- **HTMX + PicoCSS** - Modern web interface without complex JavaScript
- **Prometheus + Grafana** - Metrics and monitoring
- **OpenTelemetry + Jaeger** - Distributed tracing
- **Docker + GitHub Actions** - Multi-platform CI/CD
- **testcontainers-go** - Integration testing with real dependencies

### Configuration
Environment variables are managed in `internal/config.go`. Key variables:
- `SERVER_PORT` / `SERVER_HOST` - Server configuration (default: localhost:8080)
- `MONGODB_URI` / `MONGODB_DATABASE` - Database connection 
- `SESSION_SECRET` - Secret key for session management (required for web interface)
- `REDIS_URL` - Cache configuration (optional)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `ENVIRONMENT` - Application environment (development, production)

### Repository Pattern
All data access is abstracted through repository interfaces in `internal/application/handlers/repositories.go`. 
Full implementations are available in `internal/infrastructure/` with comprehensive error handling.

### Multi-tenancy & Security
- **Family-based isolation**: Data is strictly isolated by family ID
- **Role-based access control**: Admin, Member, Child roles with different permissions
- **Session management**: Secure HTTP-only cookies with CSRF protection
- **Input validation**: Comprehensive validation with go-playground/validator
- **Password security**: bcrypt hashing with proper salt rounds

### Web Interface Architecture
The project includes a complete web interface built with modern technologies:

**Frontend Stack:**
- **HTMX** - Dynamic updates without complex JavaScript
- **PicoCSS** - Minimalist CSS framework for clean UI
- **Go Templates** - Server-side rendering with layout system
- **Progressive Web App** - Installable with offline capabilities

**Web Components:**
- `internal/web/handlers/` - Authentication, dashboard, and HTMX endpoints  
- `internal/web/middleware/` - Session management, CSRF protection, auth guards
- `internal/web/templates/` - HTML templates with layouts and components
- `internal/web/static/` - CSS, JS, and image assets
- `internal/web/models/` - Form validation and web-specific data structures

**User Experience:**
- Responsive design works on mobile and desktop
- Real-time updates via HTMX without page refreshes
- Form validation with immediate feedback
- Accessible interface following modern UX principles

### Testing Strategy (59.5% Coverage)
The project has comprehensive testing across all layers:

**Unit Tests (200+ tests):**
- Domain models with business logic validation
- Repository implementations with mocking  
- HTTP handlers with table-driven tests
- Middleware components with edge cases
- Web form validation and error handling

**Integration Tests (100+ tests):**
- End-to-end API workflows with testcontainers
- Database operations with real MongoDB instances
- Authentication flows with session management
- Multi-family data isolation validation

**Performance Tests (50+ tests):**
- Load testing for API endpoints
- Concurrent access scenarios  
- Memory leak detection
- Database query optimization benchmarks

**Security Tests (100+ tests):**
- Authentication bypass attempts
- Authorization privilege escalation
- CSRF attack prevention
- Input validation and sanitization
- NoSQL injection protection

### Documentation
Comprehensive documentation is available in the `.memory_bank/` directory:
- **Product Brief** - Business context and goals
- **Tech Stack** - Architecture and technology choices
- **Testing Plan** - Detailed testing strategy and coverage
- **Current Tasks** - Project status and next steps

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

## 🚨 CRITICAL: Code Quality Requirements

**MANDATORY LINTER COMPLIANCE**: This project enforces strict code quality standards through golangci-lint. 

### For ALL Development Work:

**After completing ANY task that modifies code, you MUST:**

1. **Run `make fmt`** - Format code according to project standards
2. **Run `make test`** - Ensure all tests pass 
3. **Run `make lint`** - **MANDATORY** - Must result in 0 issues
4. **Fix ALL linter errors** - No exceptions, no compromises

### What the Linter Catches:
- Type errors and unused variables/imports
- Security vulnerabilities (gosec)
- Code style violations (revive, gofmt)
- Test assertion problems (testifylint)  
- Performance issues (ineffassign)
- Complexity violations (funlen, cyclop)
- Best practices violations (gocritic)

### Zero Tolerance Policy:
- **0 linter issues** required before task completion
- **No bypassing** linter rules without explicit approval
- **All warnings must be addressed** 
- If unable to resolve, escalate with specific error details

### Quick Fix Commands:
```bash
make fmt          # Auto-format code
make lint-fix     # Auto-fix simple issues  
make lint         # Check for remaining issues
```

**Remember: Clean code is not optional - it's mandatory for project integrity.**

- add to memory .memory_bank/