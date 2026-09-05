---
name: docker-up
description: Start the application in Docker with SQLite database
disable-model-invocation: true
allowed-tools: Bash(make docker-*), Bash(docker *)
---

# Docker Environment Management

Start and manage the application in Docker with SQLite database.

## Quick Start

### Start in foreground (with logs)
```bash
make docker-up
```

### Start in background (detached)
```bash
make docker-up-d
```

### View logs
```bash
make docker-logs
```

### Stop containers
```bash
make docker-down
```

### Create the family (first start only)

The container serves only the JSON API; until the family exists `/health` reports `setup_complete:false`
and `POST /api/v1/auth/login` answers `409 SETUP_REQUIRED`. Bootstrap it inside the container
(`-T`: the password comes through stdin, no TTY):

```bash
printf 'Admin1234!\n' | docker compose --project-directory . -f docker/docker-compose.yml \
  exec -T family-budget /app/family-budget-service setup \
  --family 'Test Family' --currency RUB --timezone Europe/Moscow \
  --email admin@test.com --first-name Admin --last-name Test --password-stdin
```

## What Gets Started

- **Application**: Go JSON API on configured port (default: 8080)
- **Database**: SQLite at `./data/budget.db` (persisted in Docker volume)
- **Health check**: Automatic container health monitoring via `/health` endpoint
- **Auto-migrations**: Database schema applied on startup

## Docker Image Details

- **Base**: Alpine Linux (~50MB total size)
- **Platform**: Multi-arch (linux/amd64, linux/arm64)
- **Registry**: GitHub Container Registry
- **Security**: Trivy vulnerability scanning enabled

## Environment Configuration

Create `.env` file in project root to override defaults:

```bash
SERVER_PORT=8080
SERVER_HOST=0.0.0.0
DATABASE_PATH=/data/budget.db
TRUSTED_PROXIES=
LOG_LEVEL=info
ENVIRONMENT=production
```

## Data Persistence

Database is persisted in Docker volume at `./data/`:
- **Development**: `./data/budget.db`
- **Backups**: `./backups/` directory

Data survives container restarts and rebuilds.

## Common Tasks

### Rebuild after code changes
```bash
make docker-build
make docker-up
```

### Check container status
```bash
docker ps
```

### Execute command in container
```bash
docker exec -it family-budget-service sh
```

### View container logs (live)
```bash
make docker-logs -f
```

## Health Checks

The container includes automatic health monitoring:
- **Endpoint**: `/health`
- **Interval**: 30 seconds
- **Timeout**: 3 seconds
- **Retries**: 3

Check health status:
```bash
curl http://localhost:8080/health
```

Expected response (`503` only when a check fails; a missing family keeps it `200`):
```json
{"status":"healthy","version":"v0.1.0","setup_complete":true,"timestamp":"2026-09-05T10:45:00Z","checks":{"sqlite":{"status":"healthy"},"setup":{"status":"healthy"}}}
```

## Troubleshooting

### Container won't start
1. Check logs: `make docker-logs`
2. Verify port not in use: `lsof -i :8080`
3. Check Docker daemon: `docker info`

### Database issues
1. Verify volume exists: `docker volume ls`
2. Check permissions: `ls -la ./data/`
3. Restore from backup: `/db-backup`

### Build failures
1. Clean old images: `docker system prune`
2. Rebuild: `make docker-build`
3. Check disk space: `df -h`

## See Also

- `make run-local` - Run without Docker for development
- `make docker-build` - Build Docker image only
- `/db-backup` - Backup database before Docker operations
