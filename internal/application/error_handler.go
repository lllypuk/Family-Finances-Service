package application

import (
	"errors"
	"log/slog"
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
)

// newAPIErrorHandler отдаёт любую ошибку вне хендлера (404 роутера, 405, отказ middleware,
// паника) в общем JSON-envelope. Текст произвольной ошибки наружу не уходит: в нём
// обёрнутые ошибки репозиториев и введённые пользователем значения — только в лог.
func newAPIErrorHandler(logger *slog.Logger) echo.HTTPErrorHandler {
	return func(err error, c echo.Context) {
		code := http.StatusInternalServerError
		message := statusMessage(code)

		var he *echo.HTTPError
		if errors.As(err, &he) {
			code = he.Code
			message = statusMessage(code)
			// echo.NewHTTPError(code) кладёт в Message http.StatusText — это не текст автора ошибки.
			if text, ok := he.Message.(string); ok && text != "" && text != http.StatusText(code) {
				message = text
			}
		} else {
			logger.ErrorContext(c.Request().Context(), "unhandled error",
				slog.String("method", c.Request().Method),
				slog.String("path", c.Request().URL.Path),
				slog.String("error", err.Error()))
		}

		if c.Response().Committed {
			return
		}

		_ = handlers.RespondAPIError(c, code, "", message)
	}
}

// statusMessage — текст ошибки middleware в тех же словах, что и у хендлеров.
func statusMessage(code int) string {
	switch code {
	case http.StatusUnauthorized:
		return handlers.ErrMessageUnauthorized
	case http.StatusForbidden:
		return handlers.ErrMessageForbidden
	default:
		return http.StatusText(code)
	}
}
