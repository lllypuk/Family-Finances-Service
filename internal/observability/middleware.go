package observability

import (
	"log/slog"
	"strconv"
	"time"

	"github.com/labstack/echo/v4"
)

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

// generateRequestID генерирует уникальный ID для запроса
func generateRequestID() string {
	// Простая генерация ID на основе времени
	// В production лучше использовать UUID или более надежный метод
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
