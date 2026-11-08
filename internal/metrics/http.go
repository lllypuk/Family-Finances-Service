package metrics

import (
	"net/http"
	"time"

	"family-budget-service/internal/application/router"
	"family-budget-service/internal/observability"
)

// methodOther — метка для нестандартного глагола: корневой catch-all матчит любой
// метод, и без этого списка кардинальность method×route ничем не ограничена.
const methodOther = "OTHER"

// HTTPMiddleware считает запросы, их длительность и число в обработке; ставится снаружи Router и Recover.
func HTTPMiddleware(observer *HTTPObserver) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			observer.IncInFlight()
			defer observer.DecInFlight()

			ctx, holder := router.WithLabelHolder(r.Context())
			sw := observability.AsStatusWriter(w)
			defer func() {
				observer.ObserveRequest(normalizeMethod(r.Method), holder.Route(), sw.Status(), time.Since(start))
			}()
			next.ServeHTTP(sw, r.WithContext(ctx))
		})
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
