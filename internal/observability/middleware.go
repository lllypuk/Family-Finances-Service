package observability

import (
	"errors"
	"log/slog"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// PrometheusMiddleware создает middleware для сбора Prometheus метрик
func PrometheusMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Выполняем следующий handler
			err := next(c)

			// Записываем метрики
			duration := time.Since(start)
			method := c.Request().Method
			path := c.Path()
			status := strconv.Itoa(c.Response().Status)

			// Нормализуем path для метрик (убираем параметры)
			if path == "" {
				path = c.Request().URL.Path
			}

			// Записываем основные HTTP метрики
			RecordHTTPRequest(method, path, status, duration.Seconds())

			// Записываем ошибки если есть
			if err != nil {
				errorType := "unknown"
				he := &echo.HTTPError{}
				if errors.As(err, &he) {
					errorType = strconv.Itoa(he.Code)
				}
				RecordHTTPError(method, path, errorType)
			}

			return err
		}
	}
}

// LoggingMiddleware создает middleware для structured logging
func LoggingMiddleware(logger *slog.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()

			// Генерируем request ID если его нет
			requestID := c.Request().Header.Get("X-Request-ID")
			if requestID == "" {
				requestID = generateRequestID()
				c.Request().Header.Set("X-Request-ID", requestID)
			}
			c.Response().Header().Set("X-Request-ID", requestID)

			// Выполняем следующий handler
			err := next(c)

			// Логируем результат
			duration := time.Since(start)

			logArgs := []any{
				slog.String("request_id", requestID),
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.String("remote_addr", c.RealIP()),
				slog.String("user_agent", c.Request().UserAgent()),
				slog.Int("status", c.Response().Status),
				slog.Duration("duration", duration),
				slog.Int64("bytes_in", c.Request().ContentLength),
				slog.Int64("bytes_out", c.Response().Size),
			}

			if err != nil {
				logArgs = append(logArgs, slog.String("error", err.Error()))
				logger.ErrorContext(c.Request().Context(), "HTTP request failed", logArgs...)
			} else {
				logger.InfoContext(c.Request().Context(), "HTTP request completed", logArgs...)
			}

			return err
		}
	}
}

// MetricsHandler возвращает handler для эндпоинта /metrics
func MetricsHandler() echo.HandlerFunc {
	h := promhttp.Handler()
	return echo.WrapHandler(h)
}

// generateRequestID генерирует уникальный ID для запроса
func generateRequestID() string {
	// Простая генерация ID на основе времени
	// В production лучше использовать UUID или более надежный метод
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}

// HealthCheckMiddleware добавляет базовые health check endpoints
func HealthCheckMiddleware() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Пропускаем health check endpoints от метрик
			path := c.Request().URL.Path
			if path == "/health" || path == "/ready" || path == "/metrics" {
				return next(c)
			}

			return next(c)
		}
	}
}
