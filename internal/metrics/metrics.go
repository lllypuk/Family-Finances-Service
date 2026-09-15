// Package metrics держит собственный реестр Prometheus и инструменты к нему.
// Реестр не глобальный: экземпляр создаётся в NewApplication и едет параметром,
// поэтому два New() в одном процессе (тесты) не конфликтуют.
package metrics

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"runtime"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// namespace — префикс всех собственных метрик сервиса.
const namespace = "ffs"

// Metrics — реестр и наблюдатели поверх него.
type Metrics struct {
	registry *prometheus.Registry
	logger   *slog.Logger
	http     *HTTPObserver
	login    *LoginObserver
	backup   *BackupObserver
}

// New создаёт реестр со стандартными Go/process-коллекторами и метриками сервиса.
func New(version string, logger *slog.Logger) *Metrics {
	registry := prometheus.NewRegistry()
	factory := promauto.With(registry)

	registry.MustRegister(
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}),
	)

	buildInfo := factory.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: namespace,
		Name:      "build_info",
		Help:      "Версия сборки сервиса; значение всегда 1.",
	}, []string{"version", "go"})
	buildInfo.WithLabelValues(version, runtime.Version()).Set(1)

	return &Metrics{
		registry: registry,
		logger:   logger,
		http:     newHTTPObserver(factory),
		login:    newLoginObserver(factory),
		backup:   newBackupObserver(factory),
	}
}

// Gatherer открывает реестр на чтение — для тестов и promhttp.
func (m *Metrics) Gatherer() prometheus.Gatherer {
	return m.registry
}

// Register добавляет коллектор скрейпа в реестр.
func (m *Metrics) Register(collector prometheus.Collector) error {
	return m.registry.Register(collector)
}

// NewDBCollector — стандартный коллектор пула (go_sql_*) с именем базы budget;
// db_name — часть контракта с observability, поэтому имя задаётся здесь, а не у вызывающего.
func NewDBCollector(db *sql.DB) prometheus.Collector {
	return collectors.NewDBStatsCollector(db, "budget")
}

// Handler отдаёт /metrics. ContinueOnError: одна сломавшаяся метрика не должна
// превращать весь ответ в 500.
func (m *Metrics) Handler() http.Handler {
	return promhttp.HandlerFor(m.registry, promhttp.HandlerOpts{
		ErrorHandling: promhttp.ContinueOnError,
		ErrorLog:      promLogger{logger: m.logger},
	})
}

// HTTP возвращает наблюдателя для Echo-middleware.
func (m *Metrics) HTTP() *HTTPObserver {
	return m.http
}

// Login возвращает наблюдателя попыток входа.
func (m *Metrics) Login() *LoginObserver {
	return m.login
}

// Backup возвращает наблюдателя резервных копий.
func (m *Metrics) Backup() *BackupObserver {
	return m.backup
}

// promLogger — адаптер slog под promhttp.Logger (пакет log запрещён depguard).
type promLogger struct {
	logger *slog.Logger
}

func (l promLogger) Println(v ...any) {
	l.logger.ErrorContext(context.Background(), "metrics gathering failed",
		slog.String("error", fmt.Sprint(v...)))
}
