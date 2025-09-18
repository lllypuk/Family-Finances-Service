package observability

import (
	"context"
	"net/http"
	"time"

	"github.com/labstack/echo/v4"
)

const (
	// HealthCheckTimeout timeout for individual health checks
	HealthCheckTimeout = 5 * time.Second
	// HTTPHealthCheckTimeout timeout for HTTP health check endpoint
	HTTPHealthCheckTimeout = 3 * time.Second
	// HealthStatusHealthy represents healthy status
	HealthStatusHealthy = "healthy"
	// HealthStatusUnhealthy represents unhealthy status
	HealthStatusUnhealthy = "unhealthy"
)

// HealthStatus представляет статус health check
type HealthStatus struct {
	Status    string                 `json:"status"`
	Timestamp time.Time              `json:"timestamp"`
	Version   string                 `json:"version"`
	Checks    map[string]CheckResult `json:"checks"`
	Uptime    time.Duration          `json:"uptime"`
}

// CheckResult результат индивидуальной проверки
type CheckResult struct {
	Status    string        `json:"status"`
	Message   string        `json:"message,omitempty"`
	Duration  time.Duration `json:"duration"`
	Timestamp time.Time     `json:"timestamp"`
}

// HealthChecker интерфейс для health checks
type HealthChecker interface {
	CheckHealth(ctx context.Context) CheckResult
	Name() string
}

// HealthService управляет health checks
type HealthService struct {
	checkers  []HealthChecker
	version   string
	startTime time.Time
}

// NewHealthService создает новый HealthService
func NewHealthService(version string) *HealthService {
	return &HealthService{
		checkers:  make([]HealthChecker, 0),
		version:   version,
		startTime: time.Now(),
	}
}

// AddChecker добавляет новый checker
func (hs *HealthService) AddChecker(checker HealthChecker) {
	hs.checkers = append(hs.checkers, checker)
}

// CheckHealth выполняет все проверки
func (hs *HealthService) CheckHealth(ctx context.Context) HealthStatus {
	checks := make(map[string]CheckResult)
	overallStatus := HealthStatusHealthy

	// Выполняем все проверки
	for _, checker := range hs.checkers {
		result := checker.CheckHealth(ctx)
		checks[checker.Name()] = result

		if result.Status != HealthStatusHealthy {
			overallStatus = HealthStatusUnhealthy
		}
	}

	return HealthStatus{
		Status:    overallStatus,
		Timestamp: time.Now(),
		Version:   hs.version,
		Checks:    checks,
		Uptime:    time.Since(hs.startTime),
	}
}

// HealthHandler создает HTTP handler для health check
func (hs *HealthService) HealthHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		health := hs.CheckHealth(c.Request().Context())

		statusCode := http.StatusOK
		if health.Status != HealthStatusHealthy {
			statusCode = http.StatusServiceUnavailable
		}

		return c.JSON(statusCode, health)
	}
}

// ReadinessHandler создает HTTP handler для readiness probe
func (hs *HealthService) ReadinessHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Быстрая проверка готовности - только критичные зависимости
		ctx, cancel := context.WithTimeout(c.Request().Context(), HTTPHealthCheckTimeout)
		defer cancel()

		health := hs.CheckHealth(ctx)

		// Для readiness проверяем только критичные компоненты
		ready := true
		for name, check := range health.Checks {
			// PostgreSQL критичен для готовности
			if name == "postgresql" && check.Status != HealthStatusHealthy {
				ready = false
				break
			}
		}

		response := map[string]any{
			"ready":     ready,
			"timestamp": time.Now(),
		}

		statusCode := http.StatusOK
		if !ready {
			statusCode = http.StatusServiceUnavailable
		}

		return c.JSON(statusCode, response)
	}
}

// LivenessHandler создает HTTP handler для liveness probe
func (hs *HealthService) LivenessHandler() echo.HandlerFunc {
	return func(c echo.Context) error {
		// Простая проверка жизнеспособности - сервис отвечает
		response := map[string]any{
			"alive":     true,
			"timestamp": time.Now(),
			"uptime":    time.Since(hs.startTime).Seconds(),
		}

		return c.JSON(http.StatusOK, response)
	}
}

// CustomHealthChecker для пользовательских проверок
type CustomHealthChecker struct {
	name      string
	checkFunc func(ctx context.Context) error
}

// NewCustomHealthChecker создает пользовательский checker
func NewCustomHealthChecker(name string, checkFunc func(ctx context.Context) error) *CustomHealthChecker {
	return &CustomHealthChecker{
		name:      name,
		checkFunc: checkFunc,
	}
}

// Name возвращает имя checker'а
func (c *CustomHealthChecker) Name() string {
	return c.name
}

// CheckHealth выполняет пользовательскую проверку
func (c *CustomHealthChecker) CheckHealth(ctx context.Context) CheckResult {
	start := time.Now()

	err := c.checkFunc(ctx)
	duration := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:    "unhealthy",
			Message:   err.Error(),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return CheckResult{
		Status:    HealthStatusHealthy,
		Duration:  duration,
		Timestamp: time.Now(),
	}
}

// PostgreSQLChecker checker для PostgreSQL
type PostgreSQLChecker struct {
	checker PostgreSQLHealthChecker
}

// NewPostgreSQLHealthChecker создает новый checker для PostgreSQL
func NewPostgreSQLHealthChecker(checker PostgreSQLHealthChecker) *PostgreSQLChecker {
	return &PostgreSQLChecker{checker: checker}
}

// Name возвращает имя checker'а
func (p *PostgreSQLChecker) Name() string {
	return "postgresql"
}

// CheckHealth проверяет состояние PostgreSQL
func (p *PostgreSQLChecker) CheckHealth(ctx context.Context) CheckResult {
	start := time.Now()

	// Создаем контекст с таймаутом для проверки
	checkCtx, cancel := context.WithTimeout(ctx, HealthCheckTimeout)
	defer cancel()

	// Проверяем PostgreSQL через интерфейс
	err := p.checker.HealthCheck(checkCtx)
	duration := time.Since(start)

	if err != nil {
		return CheckResult{
			Status:    "unhealthy",
			Message:   err.Error(),
			Duration:  duration,
			Timestamp: time.Now(),
		}
	}

	return CheckResult{
		Status:    HealthStatusHealthy,
		Duration:  duration,
		Timestamp: time.Now(),
	}
}
