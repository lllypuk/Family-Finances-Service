# Multi-stage build для оптимизации размера образа
FROM golang:1.25-alpine AS builder

# Установка необходимых инструментов
RUN apk add --no-cache git ca-certificates tzdata

# Создание рабочего каталога
WORKDIR /app

# Копирование go.mod и go.sum для кэширования зависимостей
COPY go.mod go.sum ./

# Загрузка зависимостей
RUN go mod download

# Копирование исходного кода
COPY . .

# Сборка приложения с оптимизацией
RUN CGO_ENABLED=0 GOOS=linux go build \
    -a -installsuffix cgo \
    -ldflags='-w -s -extldflags "-static"' \
    -o family-budget-service \
    ./cmd/server

# Финальный образ
FROM scratch

# Копирование сертификатов и часовых поясов из builder образа
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /usr/share/zoneinfo /usr/share/zoneinfo

# Копирование скомпилированного приложения
COPY --from=builder /app/family-budget-service /family-budget-service

# Создание пользователя без root привилегий
USER 65534:65534

# Открытие порта
EXPOSE 8080

# Healthcheck для проверки состояния приложения
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD ["/family-budget-service", "-health-check"] || exit 1

# Точка входа
ENTRYPOINT ["/family-budget-service"]