package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// HTTP метрики
var (
	// HTTPRequestsTotal - общее количество HTTP запросов
	HTTPRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	// HTTPRequestDuration - длительность HTTP запросов
	HTTPRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		},
		[]string{"method", "endpoint"},
	)

	// HTTPRequestsErrors - количество ошибок HTTP запросов
	HTTPRequestsErrors = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_errors_total",
			Help: "Total number of HTTP request errors",
		},
		[]string{"method", "endpoint", "type"},
	)
)

// Business метрики
var (
	// FamiliesTotal - общее количество семей
	FamiliesTotal = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "families_total",
			Help: "Total number of families in the system",
		},
	)

	// UsersTotal - количество пользователей по ролям
	UsersTotal = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "users_total",
			Help: "Total number of users by role",
		},
		[]string{"role"},
	)

	// TransactionsTotal - количество транзакций
	TransactionsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "transactions_total",
			Help: "Total number of transactions",
		},
		[]string{"type", "family_id"},
	)

	// BudgetsActive - количество активных бюджетов
	BudgetsActive = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "budgets_active",
			Help: "Number of active budgets",
		},
		[]string{"family_id"},
	)

	// TransactionAmount - сумма транзакций
	TransactionAmount = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "transaction_amount",
			Help:    "Amount of transactions",
			Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000, 2500, 5000, 10000},
		},
		[]string{"type", "currency"},
	)
)

// Database метрики
var (
	// DatabaseConnections - количество подключений к базе данных
	DatabaseConnections = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "database_connections",
			Help: "Number of database connections",
		},
		[]string{"database", "state"},
	)

	// DatabaseOperationDuration - длительность операций с БД
	DatabaseOperationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "database_operation_duration_seconds",
			Help:    "Duration of database operations in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2, 5},
		},
		[]string{"operation", "collection", "status"},
	)

	// DatabaseOperationsTotal - общее количество операций с БД
	DatabaseOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "database_operations_total",
			Help: "Total number of database operations",
		},
		[]string{"operation", "collection", "status"},
	)
)

// Application метрики
var (
	// ApplicationStartTime - время запуска приложения
	ApplicationStartTime = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "application_start_time_seconds",
			Help: "Application start time in unix timestamp",
		},
	)

	// ApplicationUptime - время работы приложения
	ApplicationUptime = promauto.NewGaugeFunc(
		prometheus.GaugeOpts{
			Name: "application_uptime_seconds",
			Help: "Application uptime in seconds",
		},
		func() float64 {
			return time.Since(startTime).Seconds()
		},
	)
)

var startTime time.Time

// InitMetrics инициализирует метрики
func InitMetrics() {
	startTime = time.Now()
	ApplicationStartTime.Set(float64(startTime.Unix()))
}

// RecordHTTPRequest записывает метрики HTTP запроса
func RecordHTTPRequest(method, endpoint, status string, duration time.Duration) {
	HTTPRequestsTotal.WithLabelValues(method, endpoint, status).Inc()
	HTTPRequestDuration.WithLabelValues(method, endpoint).Observe(duration.Seconds())
}

// RecordHTTPError записывает ошибку HTTP запроса
func RecordHTTPError(method, endpoint, errorType string) {
	HTTPRequestsErrors.WithLabelValues(method, endpoint, errorType).Inc()
}

// RecordDatabaseOperation записывает метрики операции с БД
func RecordDatabaseOperation(operation, collection, status string, duration time.Duration) {
	DatabaseOperationsTotal.WithLabelValues(operation, collection, status).Inc()
	DatabaseOperationDuration.WithLabelValues(operation, collection, status).Observe(duration.Seconds())
}

// UpdateFamiliesCount обновляет количество семей
func UpdateFamiliesCount(count float64) {
	FamiliesTotal.Set(count)
}

// UpdateUsersCount обновляет количество пользователей по ролям
func UpdateUsersCount(role string, count float64) {
	UsersTotal.WithLabelValues(role).Set(count)
}

// RecordTransaction записывает метрики транзакции
func RecordTransaction(transactionType, familyID string, amount float64, currency string) {
	TransactionsTotal.WithLabelValues(transactionType, familyID).Inc()
	TransactionAmount.WithLabelValues(transactionType, currency).Observe(amount)
}

// UpdateActiveBudgets обновляет количество активных бюджетов
func UpdateActiveBudgets(familyID string, count float64) {
	BudgetsActive.WithLabelValues(familyID).Set(count)
}

// UpdateDatabaseConnections обновляет метрики подключений к БД
func UpdateDatabaseConnections(database, state string, count float64) {
	DatabaseConnections.WithLabelValues(database, state).Set(count)
}