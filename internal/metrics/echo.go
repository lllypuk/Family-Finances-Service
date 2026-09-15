package metrics

import (
	"net/http"
	"time"

	"github.com/labstack/echo/v4"

	"family-budget-service/internal/observability"
)

// methodOther — метка для нестандартного глагола: корневой catch-all матчит любой
// метод, и без этого списка кардинальность method×route ничем не ограничена.
const methodOther = "OTHER"

// EchoMiddleware считает запросы, их длительность и число в обработке.
// Ставится первым, до Recover: паника уходит дальше уже как записанный 500.
func EchoMiddleware(observer *HTTPObserver) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			start := time.Now()
			observer.IncInFlight()
			defer observer.DecInFlight()

			err := next(c)

			observer.ObserveRequest(
				normalizeMethod(c.Request().Method),
				c.Path(),
				observability.ResponseStatus(c, err),
				time.Since(start),
			)

			return err
		}
	}
}

func normalizeMethod(method string) string {
	switch method {
	case http.MethodGet, http.MethodHead, http.MethodPost, http.MethodPut, http.MethodPatch,
		http.MethodDelete, http.MethodConnect, http.MethodOptions, http.MethodTrace:
		return method
	default:
		return methodOther
	}
}
