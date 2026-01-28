# Multi-stage build для оптимизации размера образа
FROM golang:1.25-alpine AS builder

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

# Сборка приложения с оптимизацией (CGO_ENABLED=0 для pure Go + SQLite)
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o family-budget-service \
    ./cmd/server

# Финальный образ - используем alpine вместо scratch для лучшей совместимости
FROM alpine:3.20

# Установка необходимых runtime зависимостей
# Note: User 65534 (nobody) already exists in Alpine
RUN apk add --no-cache ca-certificates tzdata wget

# Создание рабочего каталога
WORKDIR /app

# Копирование скомпилированного приложения
COPY --from=builder /app/family-budget-service /app/family-budget-service

# Копирование миграций базы данных
COPY --from=builder /app/migrations /app/migrations

# Копирование web templates и static файлов
COPY --from=builder /app/internal/web/templates /app/internal/web/templates
COPY --from=builder /app/internal/web/static /app/internal/web/static

# Создание директории для данных SQLite
RUN mkdir -p /data && \
    chown -R nobody:nobody /data && \
    chown -R nobody:nobody /app

# Переключение на непривилегированного пользователя (nobody = 65534)
USER nobody

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

# Healthcheck для проверки состояния приложения
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget -q --spider http://localhost:8080/health || exit 1

# Точка входа
ENTRYPOINT ["/app/family-budget-service"]
