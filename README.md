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

```bash
# Start MongoDB and supporting services
make docker-up

# Run the application locally
make run-local

# Run tests
make test

# Build the application
make build
```

## Development

See [CLAUDE.md](CLAUDE.md) for comprehensive development guidelines and architecture details.

For detailed project documentation, check the [.memory_bank](.memory_bank/) directory.

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
