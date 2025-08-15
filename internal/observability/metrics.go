package observability

import (
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
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

// createHTTPMetrics создает HTTP метрики с безопасной регистрацией
func createHTTPMetrics() (*prometheus.CounterVec, *prometheus.HistogramVec, *prometheus.CounterVec) {
	httpRequestsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	httpRequestsErrors := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_errors_total",
			Help: "Total number of HTTP request errors",
		},
		[]string{"method", "endpoint", "type"},
	)

	// Безопасная регистрация метрик - игнорируем ошибки дублирования
	prometheus.Register(httpRequestsTotal)
	prometheus.Register(httpRequestDuration)
	prometheus.Register(httpRequestsErrors)

	return httpRequestsTotal, httpRequestDuration, httpRequestsErrors
}

// createBusinessMetrics создает бизнес метрики с безопасной регистрацией
func createBusinessMetrics() (prometheus.Gauge, prometheus.Gauge, prometheus.Gauge, prometheus.Gauge, *prometheus.HistogramVec) {
	familiesTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "families",
			Help: "Total number of families in the system",
		},
	)

	usersTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "users",
			Help: "Total number of users in the system",
		},
	)

	transactionsTotal := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "transactions",
			Help: "Total number of transactions in the system",
		},
	)

	budgetsActive := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "budgets_active",
			Help: "Number of active budgets in the system",
		},
	)

	transactionAmount := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transaction_amount",
			Help:    "Distribution of transaction amounts",
			Buckets: []float64{1, 10, 50, 100, 500, 1000, 5000, 10000, 50000, 100000},
		},
		[]string{"type", "category"},
	)

	// Безопасная регистрация метрик - игнорируем ошибки дублирования
	prometheus.Register(familiesTotal)
	prometheus.Register(usersTotal)
	prometheus.Register(transactionsTotal)
	prometheus.Register(budgetsActive)
	prometheus.Register(transactionAmount)

	return familiesTotal, usersTotal, transactionsTotal, budgetsActive, transactionAmount
}

// createDatabaseMetrics создает метрики базы данных с безопасной регистрацией
func createDatabaseMetrics() (prometheus.Gauge, *prometheus.HistogramVec, *prometheus.CounterVec) {
	databaseConnections := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Number of active database connections",
		},
	)

	databaseOperationDuration := prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_operation_duration_seconds",
			Help:    "Duration of database operations",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5},
		},
		[]string{"operation", "collection"},
	)

	databaseOperationsTotal := prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "collection", "status"},
	)

	// Безопасная регистрация метрик - игнорируем ошибки дублирования
	prometheus.Register(databaseConnections)
	prometheus.Register(databaseOperationDuration)
	prometheus.Register(databaseOperationsTotal)

	return databaseConnections, databaseOperationDuration, databaseOperationsTotal
}

// createApplicationMetrics создает метрики приложения с безопасной регистрацией
func createApplicationMetrics() (prometheus.Gauge, prometheus.Gauge) {
	applicationStartTime := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "application_start_time_seconds",
			Help: "Start time of the application since unix epoch in seconds",
		},
	)

	applicationUptime := prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "application_uptime_seconds",
			Help: "Uptime of the application in seconds",
		},
	)

	// Безопасная регистрация метрик - игнорируем ошибки дублирования
	prometheus.Register(applicationStartTime)
	prometheus.Register(applicationUptime)

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

// getDefaultMetrics возвращает единственный экземпляр метрик, используя sync.OnceValue
func getDefaultMetrics() *Metrics {
	// Используем sync.OnceValue для ленивой инициализации без глобальных переменных
	return sync.OnceValue(func() *Metrics {
		m := NewMetrics()
		m.Initialize()
		return m
	})()
}

// InitMetrics инициализирует экземпляр метрик (для обратной совместимости)
// В текущей реализации инициализация происходит лениво при первом обращении
func InitMetrics() {
	// Вызываем getDefaultMetrics для принудительной инициализации
	_ = getDefaultMetrics()
}

// RecordHTTPRequest глобальная функция для обратной совместимости
func RecordHTTPRequest(method, endpoint, status string, duration float64) {
	metrics := getDefaultMetrics()
	metrics.RecordHTTPRequest(method, endpoint, status, duration)
}

// RecordHTTPError глобальная функция для обратной совместимости
func RecordHTTPError(method, endpoint, errorType string) {
	metrics := getDefaultMetrics()
	metrics.RecordHTTPError(method, endpoint, errorType)
}
