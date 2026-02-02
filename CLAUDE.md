# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Development Commands

### Building and Running
- `make build` - Build the application (outputs to ./build/family-budget-service)
- `make run` - Run the application directly with go run
- `make run-local` - Run with local development environment variables (requires `make dev-up` first)

#### Local Development Setup
**No prerequisites** - just run the application:
```bash
make run-local  # Runs the application on localhost:8080
```

The `run-local` command sets up the following environment:
- **Server**: localhost:8080 (default port)
- **Database**: SQLite at `./data/budget.db` (created automatically)
- **Logging**: DEBUG level for development
- **Session Secret**: Development-specific secret key
- **Auto-migrations**: Database schema created on first run

#### Testing Local Interface
When testing the web interface locally, use the `--noproxy` flag to bypass proxy settings:
```bash
# Test health endpoint
curl -s --noproxy '*' 127.0.0.1:8080/health

# Test main page
curl -s --noproxy '*' 127.0.0.1:8080/

# Test login page
curl -s --noproxy '*' 127.0.0.1:8080/login
```

### Testing and Code Quality
- `make test` - Run tests with in-memory SQLite (⚡ instant)
- `make test-unit` - Unit tests only
- `make test-integration` - Integration tests only
- `make test-coverage` - Run tests with coverage report
- `make lint` - Run golangci-lint for comprehensive code quality checks
- `make fmt` - Format code with go fmt
- `make pre-commit` - Run pre-commit checks (format, test, lint)

#### 🚀 Performance Optimization for Tests
The project uses **in-memory SQLite** for ultra-fast testing:

**Performance benefits:**
- **No Docker** - Tests start instantly
- **In-memory database** - Each test gets a fresh database in milliseconds
- **Automatic cleanup** - Database destroyed after each test
- **Parallel execution** - Tests can run in parallel safely

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

### Frontend Testing with Agent-Browser
The project includes **agent-browser** for interactive frontend testing and browser automation.

**What it is:**
- Headless browser automation CLI optimized for AI agents
- Provides accessibility tree snapshots with element references
- Enables testing of HTMX dynamic updates and web interactions

**Quick usage:**
```bash
# Start local server first
make run-local

# In another terminal, test the frontend
agent-browser open "http://127.0.0.1:8080"
agent-browser snapshot  # Get page structure with element refs
agent-browser click @e5  # Interact with elements
agent-browser screenshot /tmp/page.png  # Capture visuals
agent-browser close
```

**Common testing scenarios:**
- Login flow validation
- HTMX dynamic update verification
- Form submission and validation
- Session management testing
- Responsive design checks
- Screenshot generation for documentation

**Available as skill:** Use `/test-frontend` to launch interactive frontend testing with step-by-step guidance.

**Documentation:** Full command reference at https://github.com/vercel-labs/agent-browser

### Dependencies and Maintenance
- `make deps` - Download and tidy Go modules
- `make clean` - Remove build artifacts and coverage reports
- `make generate` - Generate OpenAPI code

### Docker Environment
- `make docker-up` - Start application in Docker
- `make docker-up-d` - Start in detached mode (background)
- `make docker-down` - Stop Docker containers
- `make docker-logs` - View application logs
- `make docker-build` - Build Docker image

The Docker container includes:
- **Single container** deployment (~50MB Alpine-based image)
- **SQLite database** persisted in Docker volume (`./data/`)
- **Health check** endpoint for container orchestration
- **Automatic migrations** applied on startup

### SQLite Database Commands
- `make sqlite-backup` - Create database backup
- `make sqlite-restore BACKUP_FILE=./backups/file.db` - Restore from backup
- `make sqlite-shell` - Open SQLite interactive shell
- `make sqlite-stats` - Show database statistics

### Database Migrations
- `make migrate-create` - Show guide for adding new migrations
- **Migration Structure**: All migrations are consolidated in two files:
  - `migrations/001_consolidated.up.sql` - All schema changes (9 tables, 19+ indexes, triggers)
  - `migrations/001_consolidated.down.sql` - All rollback statements
- **Tables**: families, users, categories, transactions, budgets, budget_alerts, reports, user_sessions, invites
- **Adding New Migrations**: Edit the consolidated files directly, adding new changes at the end
- **Documentation**: See `migrations/README.md` for detailed guide and `migrations/CHANGELOG.md` for history
- **Note**: Migrations run automatically on application startup

## Architecture Overview

This is a production-ready family budget management service built with Go, following Clean Architecture principles and comprehensive testing practices.

### Current Status: DEVELOPMENT READY
- ✅ **Complete web interface** with HTMX 2.0.4 + PicoCSS 2.1.1
- ✅ **Full API implementation** with REST endpoints
- ✅ **Comprehensive security** with authentication & authorization
- ✅ **Invite system** for user onboarding via secure token links
- ✅ **Backup management** with create, download, restore, and auto-cleanup
- ✅ **Admin panel** for user and invite management
- ✅ **CI/CD pipelines** with GitHub Actions

### Domain Structure
The application is organized into domain modules in `internal/domain/`:
- **User/Family**: User management with role-based access (admin, member, child)
- **Invite**: Invitation system with secure tokens, expiration, and status tracking
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
- `internal/services/` - Business logic services (invite, backup, budget, etc.)
- `internal/infrastructure/` - SQLite repositories and data persistence
- `internal/observability/` - Logging and health checks

### Key Technologies (Production Stack)
- **Go 1.25.6** - Latest Go version with enhanced performance
- **Echo v4.13.4** - HTTP web framework with middleware
- **SQLite** (modernc.org/sqlite) - Embedded database, pure Go, no CGO
- **HTMX v2.0.4** - Modern web interface without complex JavaScript
- **PicoCSS v2.1.1** - Minimalist CSS framework for clean UI
- **Docker + GitHub Actions** - Multi-platform CI/CD
- **In-memory testing** - Ultra-fast tests with SQLite

### Configuration
Environment variables are managed in `internal/config.go`. Key variables:
- `SERVER_PORT` / `SERVER_HOST` - Server configuration (default: localhost:8080)
- `DATABASE_PATH` - Path to SQLite database file (default: `./data/budget.db`)
- `SESSION_SECRET` - Secret key for session management (required for web interface)
- `LOG_LEVEL` - Logging level (debug, info, warn, error)
- `ENVIRONMENT` - Application environment (development, production)

### Repository Pattern
All data access is abstracted through repository interfaces in `internal/application/handlers/repositories.go`.
Full implementations are available in `internal/infrastructure/` with comprehensive error handling.

### Security
- **Role-based access control**: Admin, Member, Child roles with different permissions
- **Session management**: Secure HTTP-only cookies with CSRF protection
- **Input validation**: Comprehensive validation with go-playground/validator
- **Password security**: bcrypt hashing with proper salt rounds
- **Invite tokens**: Cryptographically secure 32-byte tokens with 7-day expiration
- **Backup security**: Path traversal protection, admin-only access, filename validation

### Web Interface Architecture
The project includes a complete web interface built with modern technologies:

**Frontend Stack:**
- **HTMX v2.0.4** - Dynamic updates without complex JavaScript (MANDATORY: use HTMX 2.0.4+ only)
- **PicoCSS v2.1.1** - Minimalist CSS framework for clean UI (MANDATORY: use PicoCSS 2.1.1+ only)
- **Go Templates** - Server-side rendering with layout system
- **Progressive Web App** - Installable with offline capabilities

**🚨 CRITICAL Frontend Development Rules:**
1. **MANDATORY: Use HTMX v2.0.4+** - Never downgrade to HTMX v1.x
2. **MANDATORY: Use PicoCSS v2.1.1+** - Leverage class-less approach and modern CSS variables
3. **AVOID JavaScript** - Use HTMX for dynamic behavior instead of custom JS
4. **Server-Side First** - Prefer server rendering over client-side solutions
5. **HTMX-Only Interactivity** - Use hx-* attributes for all dynamic features

**Web Components:**
- `internal/web/handlers/` - Authentication, dashboard, admin, backup, and HTMX endpoints
- `internal/web/middleware/` - Session management, CSRF protection, auth guards
- `internal/web/templates/` - HTML templates with layouts, components, and admin pages
- `internal/web/static/` - CSS, JS, and image assets
- `internal/web/models/` - Form validation and web-specific data structures

**User Experience:**
- Responsive design works on mobile and desktop
- Real-time updates via HTMX without page refreshes
- Form validation with immediate feedback
- Accessible interface following modern UX principles

### Testing Strategy
The project has comprehensive testing across all layers:

**Unit Tests (42+ test files):**
- Domain models with business logic validation
- Repository implementations with in-memory SQLite
- Service layer tests (invite, backup, budget, category, transaction, report, user)
- HTTP handlers with table-driven tests
- Middleware components with edge cases
- Web form validation and error handling
- Template renderer tests

**Integration Tests:**
- End-to-end API workflows with in-memory database
- Database operations with SQLite
- Authentication flows with session management
- Invite flow integration
- Data integrity validation

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
- **Environment Setup**: Go 1.25.6, SQLite for integration tests
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

### Monitoring
- **Health Check**: `/health` endpoint for container orchestration
- **Structured Logging**: slog-based logging with configurable levels
- **Coverage Reports**: Integrated with Codecov for test coverage tracking

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
make lint         # Check for issues
make pre-commit   # Run full check sequence
```

**Remember: Clean code is not optional - it's mandatory for project integrity.**

## 🚧 Known Issues & TODO

### Test Coverage Improvements Needed
- **Web handlers**: Low coverage - needs test implementation
- **Web models**: Low coverage - needs expanded validation tests
- **Admin/Backup handlers**: New handlers need test coverage
- **Target**: Increase overall test coverage

### Development Priorities
1. Add tests for admin and backup web handlers
2. Improve web models test coverage
3. Add more integration test scenarios for invite flow
4. Performance optimization and benchmarking
