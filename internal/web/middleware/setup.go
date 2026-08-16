package middleware

import (
	"context"
	"net/http"
	"strings"
	"sync/atomic"

	"github.com/labstack/echo/v4"
)

const (
	// staticPath — корень статики; Echo регистрирует его как маршрут `/static*`.
	staticPath = "/static"
	// staticPathPrefix — префикс статических файлов, которые не участвуют в проверке настройки.
	staticPathPrefix = staticPath + "/"
	// setupPath — единственная страница, доступная до завершения настройки.
	setupPath = "/setup"
	// healthPath — эндпоинт проверки живости, нужен мониторингу до завершения настройки.
	healthPath = "/health"
	// faviconPath — браузер запрашивает его сам, редирект на /setup здесь бессмысленен.
	faviconPath = "/favicon.ico"
)

// SetupChecker defines an interface for checking setup status
type SetupChecker interface {
	IsSetupComplete(ctx context.Context) (bool, error)
}

// isSetupExempt сообщает, что путь не участвует в проверке завершённости настройки.
// Без этого страница /setup открывается без стилей, а мониторинг не может опросить
// сервис до первого запуска (находка U-01).
func isSetupExempt(path string) bool {
	switch path {
	case healthPath, faviconPath, staticPath:
		return true
	default:
		return strings.HasPrefix(path, staticPathPrefix)
	}
}

// checkSetupCached спрашивает checker только пока настройка не завершена.
// Второе значение false означает «проверить не удалось» — вызывающий код обязан
// пропустить запрос (graceful degradation).
//
// Кэшируется только положительный результат: семью нельзя удалить через UI,
// поэтому обратный переход true → false невозможен. Иначе IsSetupComplete
// означал бы запрос в БД на каждый HTTP-запрос (находка S-05).
func checkSetupCached(ctx context.Context, checker SetupChecker, cache *atomic.Bool) (bool, bool) {
	if cache.Load() {
		return true, true
	}

	complete, err := checker.IsSetupComplete(ctx)
	if err != nil {
		return false, false
	}

	if complete {
		cache.Store(true)
	}

	return complete, true
}

// RequireSetup middleware redirects to /setup if the family has not been created yet.
// If the family already exists and the user requests /setup, redirects to /login.
func RequireSetup(checker SetupChecker) echo.MiddlewareFunc {
	// Кэш живёт в замыкании, а не в пакетной переменной (gochecknoglobals).
	var setupComplete atomic.Bool

	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Именно URL запроса, а не c.Path(): последний возвращает шаблон маршрута
			// (для статики — `/static*`), а для незарегистрированного пути — пустую строку.
			path := c.Request().URL.Path

			if isSetupExempt(path) {
				return next(c)
			}

			complete, checked := checkSetupCached(c.Request().Context(), checker, &setupComplete)
			if !checked {
				// On error, allow access (graceful degradation)
				return next(c)
			}

			if !complete {
				// Family not set up yet — only /setup is accessible
				if path != setupPath {
					return c.Redirect(http.StatusFound, setupPath)
				}
				return next(c)
			}

			// Family exists — /setup is no longer accessible
			if path == setupPath {
				return c.Redirect(http.StatusFound, "/login")
			}

			return next(c)
		}
	}
}
