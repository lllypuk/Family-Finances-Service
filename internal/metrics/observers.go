package metrics

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

const (
	// subsystemHTTP — подсистема метрик Echo-middleware.
	subsystemHTTP = "http"
	labelOutcome  = "outcome"
)

// HTTPObserver — инструменты Echo-middleware.
type HTTPObserver struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
	inFlight prometheus.Gauge
	panics   prometheus.Counter
}

func newHTTPObserver(factory promauto.Factory) *HTTPObserver {
	return &HTTPObserver{
		requests: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystemHTTP,
			Name:      "requests_total",
			Help:      "Число обработанных HTTP-запросов.",
		}, []string{"method", "route", "status"}),
		duration: factory.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: namespace,
			Subsystem: subsystemHTTP,
			Name:      "request_duration_seconds",
			Help:      "Длительность обработки HTTP-запроса.",
			Buckets:   prometheus.DefBuckets,
		}, []string{"method", "route"}),
		inFlight: factory.NewGauge(prometheus.GaugeOpts{
			Namespace: namespace,
			Subsystem: subsystemHTTP,
			Name:      "requests_in_flight",
			Help:      "Число запросов в обработке прямо сейчас.",
		}),
		panics: factory.NewCounter(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: subsystemHTTP,
			Name:      "panics_total",
			Help:      "Число паник, перехваченных Recover-middleware.",
		}),
	}
}

// IncInFlight отмечает начало обработки запроса.
func (o *HTTPObserver) IncInFlight() {
	o.inFlight.Inc()
}

// DecInFlight отмечает конец обработки запроса.
func (o *HTTPObserver) DecInFlight() {
	o.inFlight.Dec()
}

// ObserveRequest пишет счётчик и длительность завершённого запроса.
func (o *HTTPObserver) ObserveRequest(method, route string, status int, d time.Duration) {
	o.requests.WithLabelValues(method, route, strconv.Itoa(status)).Inc()
	o.duration.WithLabelValues(method, route).Observe(d.Seconds())
}

// ObservePanic считает перехваченную панику.
func (o *HTTPObserver) ObservePanic() {
	o.panics.Inc()
}

// LoginObserver считает попытки входа по исходу.
type LoginObserver struct {
	attempts *prometheus.CounterVec
}

func newLoginObserver(factory promauto.Factory) *LoginObserver {
	return &LoginObserver{
		attempts: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Subsystem: "login",
			Name:      "attempts_total",
			Help:      "Попытки входа по исходу.",
		}, []string{labelOutcome}),
	}
}

// ObserveLogin считает попытку входа с данным исходом.
func (o *LoginObserver) ObserveLogin(outcome string) {
	o.attempts.WithLabelValues(outcome).Inc()
}

// BackupObserver считает резервные копии, запущенные через API.
type BackupObserver struct {
	total    *prometheus.CounterVec
	duration prometheus.Histogram
}

func newBackupObserver(factory promauto.Factory) *BackupObserver {
	return &BackupObserver{
		total: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "backups_total",
			Help:      "Резервные копии, снятые через API, по исходу.",
		}, []string{labelOutcome}),
		duration: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "backup_duration_seconds",
			Help:      "Длительность снятия резервной копии через API.",
			Buckets:   []float64{0.1, 0.25, 0.5, 1, 2.5, 5, 10, 30, 60, 120},
		}),
	}
}

// ObserveBackup пишет исход и длительность снятия копии.
func (o *BackupObserver) ObserveBackup(outcome string, d time.Duration) {
	o.total.WithLabelValues(outcome).Inc()
	o.duration.Observe(d.Seconds())
}

// RecognizeObserver считает вызовы распознавания скриншотов.
type RecognizeObserver struct {
	total    *prometheus.CounterVec
	duration prometheus.Histogram
}

func newRecognizeObserver(factory promauto.Factory) *RecognizeObserver {
	return &RecognizeObserver{
		total: factory.NewCounterVec(prometheus.CounterOpts{
			Namespace: namespace,
			Name:      "recognitions_total",
			Help:      "Вызовы распознавания скриншотов по исходу.",
		}, []string{labelOutcome}),
		duration: factory.NewHistogram(prometheus.HistogramOpts{
			Namespace: namespace,
			Name:      "recognition_duration_seconds",
			Help:      "Длительность распознавания скриншотов, включая повторы вызова модели.",
			Buckets:   []float64{0.5, 1, 2.5, 5, 10, 20, 30, 60, 90, 130},
		}),
	}
}

// ObserveRecognition пишет исход и длительность вызова; число строк в метрики не идёт.
func (o *RecognizeObserver) ObserveRecognition(outcome string, d time.Duration, _ int) {
	o.total.WithLabelValues(outcome).Inc()
	o.duration.Observe(d.Seconds())
}
