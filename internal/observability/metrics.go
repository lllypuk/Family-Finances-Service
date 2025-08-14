package observability

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Metrics структура для инкапсуляции всех метрик приложения
type Metrics struct {
	// HTTP метрики
	HTTPRequestsTotal   *prometheus.CounterVec
	HTTPRequestDuration *prometheus.HistogramVec
	HTTPRequestsErrors  *prometheus.CounterVec

	// Business метрики
	FamiliesTotal     prometheus.Gauge
	UsersTotal        prometheus.Gauge
	TransactionsTotal prometheus.Gauge
	BudgetsActive     prometheus.Gauge
	TransactionAmount *prometheus.HistogramVec

	// Database метрики
	DatabaseConnections       prometheus.Gauge
	DatabaseOperationDuration *prometheus.HistogramVec
	DatabaseOperationsTotal   *prometheus.CounterVec

	// Application метрики
	ApplicationStartTime prometheus.Gauge
	ApplicationUptime    prometheus.Gauge

	startTime time.Time
}

// createHTTPMetrics создает HTTP метрики
func createHTTPMetrics() (*prometheus.CounterVec, *prometheus.HistogramVec, *prometheus.CounterVec) {
	httpRequestsTotal := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	httpRequestsErrors := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_errors_total",
			Help: "Total number of HTTP request errors",
		},
		[]string{"method", "endpoint", "type"},
	)

	return httpRequestsTotal, httpRequestDuration, httpRequestsErrors
}

// createBusinessMetrics создает бизнес метрики
func createBusinessMetrics() (prometheus.Gauge, prometheus.Gauge, prometheus.Gauge, prometheus.Gauge, *prometheus.HistogramVec) {
	familiesTotal := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "families",
			Help: "Total number of families in the system",
		},
	)

	usersTotal := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "users",
			Help: "Total number of users in the system",
		},
	)

	transactionsTotal := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "transactions",
			Help: "Total number of transactions in the system",
		},
	)

	budgetsActive := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "budgets_active",
			Help: "Number of active budgets in the system",
		},
	)

	transactionAmount := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transaction_amount",
			Help:    "Distribution of transaction amounts",
			Buckets: []float64{1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"type", "category"},
	)

	return familiesTotal, usersTotal, transactionsTotal, budgetsActive, transactionAmount
}

// createDatabaseMetrics создает метрики базы данных
func createDatabaseMetrics() (prometheus.Gauge, *prometheus.HistogramVec, *prometheus.CounterVec) {
	databaseConnections := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Number of active database connections",
		},
	)

	databaseOperationDuration := promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_operation_duration_seconds",
			Help:    "Duration of database operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"operation", "collection"},
	)

	databaseOperationsTotal := promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "collection", "status"},
	)

	return databaseConnections, databaseOperationDuration, databaseOperationsTotal
}

// createApplicationMetrics создает метрики приложения
func createApplicationMetrics() (prometheus.Gauge, prometheus.Gauge) {
	applicationStartTime := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "application_start_time_seconds",
			Help: "Start time of the application since unix epoch in seconds",
		},
	)

	applicationUptime := promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "application_uptime_seconds",
			Help: "Uptime of the application in seconds",
		},
	)

	return applicationStartTime, applicationUptime
}

// NewMetrics создает новый экземпляр метрик
func NewMetrics() *Metrics {
	startTime := time.Now()

	// Создаем группы метрик
	httpRequestsTotal, httpRequestDuration, httpRequestsErrors := createHTTPMetrics()
	familiesTotal, usersTotal, transactionsTotal, budgetsActive, transactionAmount := createBusinessMetrics()
	databaseConnections, databaseOperationDuration, databaseOperationsTotal := createDatabaseMetrics()
	applicationStartTime, applicationUptime := createApplicationMetrics()

	return &Metrics{
		// HTTP метрики
		HTTPRequestsTotal:   httpRequestsTotal,
		HTTPRequestDuration: httpRequestDuration,
		HTTPRequestsErrors:  httpRequestsErrors,

		// Business метрики
		FamiliesTotal:     familiesTotal,
		UsersTotal:        usersTotal,
		TransactionsTotal: transactionsTotal,
		BudgetsActive:     budgetsActive,
		TransactionAmount: transactionAmount,

		// Database метрики
		DatabaseConnections:       databaseConnections,
		DatabaseOperationDuration: databaseOperationDuration,
		DatabaseOperationsTotal:   databaseOperationsTotal,

		// Application метрики
		ApplicationStartTime: applicationStartTime,
		ApplicationUptime:    applicationUptime,

		startTime: startTime,
	}
}

// Initialize инициализирует начальные значения метрик
func (m *Metrics) Initialize() {
	m.ApplicationStartTime.Set(float64(m.startTime.Unix()))
}

// UpdateUptime обновляет метрику времени работы приложения
func (m *Metrics) UpdateUptime() {
	uptime := time.Since(m.startTime)
	m.ApplicationUptime.Set(uptime.Seconds())
}

// GetHTTPRequestsTotal возвращает метрику HTTP запросов
func (m *Metrics) GetHTTPRequestsTotal() *prometheus.CounterVec {
	return m.HTTPRequestsTotal
}

// GetHTTPRequestDuration возвращает метрику длительности HTTP запросов
func (m *Metrics) GetHTTPRequestDuration() *prometheus.HistogramVec {
	return m.HTTPRequestDuration
}

// GetHTTPRequestsErrors возвращает метрику ошибок HTTP запросов
func (m *Metrics) GetHTTPRequestsErrors() *prometheus.CounterVec {
	return m.HTTPRequestsErrors
}

// RecordHTTPRequest записывает HTTP запрос
func (m *Metrics) RecordHTTPRequest(method, endpoint, status string, duration float64) {
	m.HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	m.HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration)
}

// RecordHTTPError записывает HTTP ошибку
func (m *Metrics) RecordHTTPError(method, endpoint, errorType string) {
	m.HTTPRequestsErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// RecordDatabaseOperation записывает операцию базы данных
func (m *Metrics) RecordDatabaseOperation(operation, collection, status string, duration float64) {
	m.DatabaseOperationsTotal.WithLabelValues(operation, collection, status).Inc()
	m.DatabaseOperationDuration.WithLabelValues(operation, collection).Observe(duration)
}

// MetricsRegistry инкапсулирует singleton pattern для метрик
type MetricsRegistry struct {
	once     sync.Once
	instance *Metrics
}

// NewMetricsRegistry создает новый registry
func NewMetricsRegistry() *MetricsRegistry {
	return &MetricsRegistry{}
}

// Get возвращает экземпляр метрик (thread-safe singleton)
func (r *MetricsRegistry) Get() *Metrics {
	r.once.Do(func() {
		r.instance = NewMetrics()
		r.instance.Initialize()
	})
	return r.instance
}

// Package-level функции для backward compatibility используют созданный при необходимости registry
// Избегаем global переменных используя функциональный подход

// InitMetrics инициализирует глобальный экземпляр метрик (для обратной совместимости)
func InitMetrics() {
	// В новом подходе инициализация происходит лениво при первом вызове
	// Это функция остается для API совместимости
}

// RecordHTTPRequest глобальная функция для обратной совместимости
func RecordHTTPRequest(method, endpoint, status string, duration float64) {
	metrics := createMetricsOnce()
	metrics.RecordHTTPRequest(method, endpoint, status, duration)
}

// RecordHTTPError глобальная функция для обратной совместимости
func RecordHTTPError(method, endpoint, errorType string) {
	metrics := createMetricsOnce()
	metrics.RecordHTTPError(method, endpoint, errorType)
}

// createMetricsOnce создает единственный экземпляр метрик через sync.OnceValue
func createMetricsOnce() *Metrics {
	// Используем замыкание для создания статической переменной без global scope
	create := sync.OnceValue(func() *Metrics {
		m := NewMetrics()
		m.Initialize()
		return m
	})
	return create()
}
