package application

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/application/router"
)

// statusMessage — текст ошибки middleware в тех же словах, что и у хендлеров.
func statusMessage(code int) string {
	switch code {
	case http.StatusUnauthorized:
		return handlers.ErrMessageUnauthorized
	case http.StatusForbidden:
		return handlers.ErrMessageForbidden
	case http.StatusRequestEntityTooLarge:
		return handlers.ErrMessagePayloadTooLarge
	default:
		return http.StatusText(code)
	}
}

// newAPIErrorHandler отдаёт любую ошибку хендлера, middleware и роутера в общем JSON-envelope.
// Не-HTTPError — обезличенный 500: в её тексте ошибки репозиториев и пользовательский ввод, он идёт только в лог.
func newAPIErrorHandler(logger *slog.Logger) router.ErrorHandler {
	return func(w http.ResponseWriter, r *http.Request, err error) {
		status := http.StatusInternalServerError
		code := ""
		message := statusMessage(status)

		var he *router.HTTPError
		switch {
		case errors.Is(err, context.DeadlineExceeded) && hasDeadline(r):
			status = http.StatusServiceUnavailable
			message = statusMessage(status)
		case errors.As(err, &he):
			status = he.Status
			code = he.Code
			message = statusMessage(status)
			if he.Message != "" {
				message = he.Message
			}
			if status >= http.StatusInternalServerError && he.Err != nil {
				logger.ErrorContext(r.Context(), "server error",
					slog.String("method", r.Method),
					slog.String("path", r.URL.Path),
					slog.Int("status", status),
					slog.String("error", he.Err.Error()))
			}
		default:
			logger.ErrorContext(r.Context(), "unhandled error",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.String("error", err.Error()))
		}

		if router.Committed(w) {
			return
		}

		_ = handlers.RespondAPIError(w, status, code, message)
	}
}

// hasDeadline: срок есть только у запросов под Timeout; маршрут распознавания без срока отдаёт 500, как с echo.
func hasDeadline(r *http.Request) bool {
	_, ok := r.Context().Deadline()

	return ok
}
