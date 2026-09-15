package metrics

import (
	"net/http"
	"time"
)

// readHeaderTimeout закрывает Slowloris на служебном порту (gosec G112).
const readHeaderTimeout = 10 * time.Second

// NewServer собирает служебный слушатель с единственным маршрутом GET /metrics.
// Запуском и остановкой распоряжается вызывающий.
func NewServer(addr string, handler http.Handler) *http.Server {
	mux := http.NewServeMux()
	mux.Handle("/metrics", handler)

	return &http.Server{
		Addr:              addr,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
}
