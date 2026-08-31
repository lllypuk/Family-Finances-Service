# Multi-stage build для оптимизации размера образа
FROM golang:1.27-alpine@sha256:4c9fe60190a2a3350ddc51de80d0224b8a6698d12bdfc999fee45ea9d6c46dbc AS builder

# Установка необходимых инструментов
RUN apk add --no-cache ca-certificates git tzdata

# Создание рабочего каталога
WORKDIR /app

# Копирование go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения с оптимизацией
# CGO_ENABLED=0 правильно для modernc.org/sqlite (pure Go драйвер, не требует CGO)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o family-budget-service \
    ./cmd/server

# Финальный образ - используем alpine вместо scratch для лучшей совместимости
FROM alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b

# Установка необходимых runtime зависимостей
RUN apk add --no-cache ca-certificates tzdata wget

# Непривилегированный пользователь с фиксированным UID/GID 1000.
# Раньше здесь был `nobody` (65534), а deploy/docker-compose.prod.yml запускал
# контейнер от `1000:1000` — на bind-mount `./data` SQLite не мог открыть БД (D-03).
# Теперь UID один и тот же в образе и в compose; install.sh делает
# `chown 1000:1000` на data/backups/logs.
RUN addgroup -g 1000 -S app && \
    adduser -u 1000 -S -G app -H -s /sbin/nologin app

# Создание рабочего каталога
WORKDIR /app

# Копирование скомпилированного приложения
COPY --from=builder /app/family-budget-service /app/family-budget-service

# Копирование миграций базы данных
COPY --from=builder /app/migrations /app/migrations

# Копирование web templates и static файлов
COPY --from=builder /app/internal/web/templates /app/internal/web/templates
COPY --from=builder /app/internal/web/static /app/internal/web/static

# Создание директорий для данных SQLite, бэкапов и логов
# (deploy/docker-compose.*.yml монтируют /backups и /logs с хоста)
RUN mkdir -p /data /backups /logs && \
    chown -R app:app /data /backups /logs /app

# Переключение на непривилегированного пользователя (app = 1000:1000)
USER 1000:1000

# Volume для SQLite данных
VOLUME ["/data"]

# Переменные окружения по умолчанию
ENV DATABASE_PATH=/data/budget.db \
    SERVER_PORT=8080 \
    SERVER_HOST=0.0.0.0 \
    LOG_LEVEL=info \
    ENVIRONMENT=production

# Открытие порта
EXPOSE 8080

# Healthcheck для проверки состояния приложения.
# `--spider` шлёт HEAD, а зарегистрирован только GET /health — HEAD уходит
# в редирект на /login и healthcheck падает. Поэтому обычный GET в /dev/null.
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q -O /dev/null http://localhost:8080/health || exit 1

# Точка входа
ENTRYPOINT ["/app/family-budget-service"]
