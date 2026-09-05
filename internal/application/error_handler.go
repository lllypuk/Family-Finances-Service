package application

import (
	"errors"
	"net/http"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/application/handlers"
)

// apiErrorHandler отдаёт любую ошибку вне хендлера (404 роутера, 405, отказ middleware,
// паника) в общем JSON-envelope. Текст произвольной ошибки наружу не уходит: в нём
// обёрнутые ошибки репозиториев и введённые пользователем значения — только в лог.
func apiErrorHandler(err error, c echo.Context) {
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
		// %q, а не %s: URL.Path приходит декодированным, и %0a в запросе
		// давал бы настоящий перевод строки в логе.
		c.Logger().Errorf("unhandled error on %s %q: %q",
			c.Request().Method, c.Request().URL.Path, err.Error())
	}

	if c.Response().Committed {
		return
	}

	_ = handlers.RespondAPIError(c, code, "", message)
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
