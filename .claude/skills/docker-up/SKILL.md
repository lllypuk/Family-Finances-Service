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

## What Gets Started

- **Application**: Go web service on configured port (default: 8080)
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
SESSION_SECRET=your-secret-key-here
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
- **Timeout**: 10 seconds
- **Retries**: 3

Check health status:
```bash
curl http://localhost:8080/health
```

Expected response:
```json
{"status":"healthy","timestamp":"2026-01-30T10:45:00Z"}
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
