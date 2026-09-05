package middleware

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
)

// apiPathPrefix — префикс JSON-роутов; ответы под ним всегда в API-envelope.
const apiPathPrefix = "/api/"

// ErrCSRFTokenInvalid — маркер отбитого CSRF-запроса к API: по нему обработчик
// ошибок выбирает error.code в JSON-envelope.
var ErrCSRFTokenInvalid = errors.New("csrf token validation failed")

// IsAPIPath сообщает, обслуживается ли путь JSON-API.
func IsAPIPath(path string) bool {
	return strings.HasPrefix(path, apiPathPrefix)
}

const (
	CSRFTokenKey    = "csrf_token"
	CSRFFormKey     = "_token"
	CSRFHeaderKey   = "X-Csrf-Token"
	CSRFTokenLength = 32
)

// CSRFProtection middleware защищает от CSRF атак
func CSRFProtection() echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			// Пропускаем GET, HEAD, OPTIONS запросы
			method := c.Request().Method
			if method == "GET" || method == "HEAD" || method == "OPTIONS" {
				// Генерируем токен для новых сессий
				if err := ensureCSRFToken(c); err != nil {
					return err
				}
				return next(c)
			}

			// Для POST, PUT, DELETE запросов проверяем токен
			if err := validateCSRFToken(c); err != nil {
				// API отвечает общим envelope, а не текстом: сгенерированный
				// клиент читает только {"error":…} и по коду понимает,
				// что токен пора обновить.
				if IsAPIPath(c.Request().URL.Path) {
					return echo.NewHTTPError(http.StatusForbidden, "CSRF token validation failed").
						SetInternal(ErrCSRFTokenInvalid)
				}
				if IsHTMXRequest(c) {
					return c.JSON(http.StatusForbidden, map[string]string{
						"error": "CSRF token validation failed",
					})
				}
				return c.String(http.StatusForbidden, "CSRF token validation failed")
			}

			return next(c)
		}
	}
}

// ensureCSRFToken гарантирует наличие CSRF токена в сессии.
//
// gorilla CookieStore.New возвращает НЕнулевую ошибку вместе с пригодной новой
// сессией, когда пришедшую cookie не удалось расшифровать (чужой или
// провёрнутый SESSION_SECRET). Раньше эта ошибка пробрасывалась наружу, и
// браузер со «старой» cookie получал 500 на каждый GET, причём сама cookie не
// перезаписывалась — выйти из этого состояния можно было только руками.
// Поэтому нечитаемая сессия трактуется как новая: ниже в неё кладётся свежий
// токен и sess.Save перезаписывает cookie.
func ensureCSRFToken(c echo.Context) error {
	sess, err := sessionOrNew(c)
	if sess == nil {
		return err
	}

	// Проверяем, есть ли уже токен
	if token, exists := sess.Values[CSRFTokenKey]; exists && token != nil {
		return nil
	}

	_, err = setFreshCSRFToken(c, sess)
	return err
}

// setFreshCSRFToken выпускает новый токен, кладёт его в сессию и сохраняет
// cookie. Единственное место, где эти три шага идут вместе: ensureCSRFToken,
// RegenerateCSRFToken и запасная ветка GetCSRFToken делают ровно это.
func setFreshCSRFToken(c echo.Context, sess *sessions.Session) (string, error) {
	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}

	sess.Values[CSRFTokenKey] = token
	if saveErr := sess.Save(c.Request(), c.Response()); saveErr != nil {
		return "", saveErr
	}

	return token, nil
}

// validateCSRFToken проверяет CSRF токен.
//
// Нерасшифровываемая cookie (см. ensureCSRFToken) — это не 500, а обычный
// отказ: сессии нет, значит и токена в ней нет.
func validateCSRFToken(c echo.Context) error {
	sess, err := sessionOrNew(c)
	if sess == nil {
		return err
	}

	// Получаем токен из сессии
	sessionToken, exists := sess.Values[CSRFTokenKey]
	if !exists || sessionToken == nil {
		return echo.NewHTTPError(http.StatusForbidden, "CSRF token not found in session")
	}

	sessionTokenStr, ok := sessionToken.(string)
	if !ok {
		return echo.NewHTTPError(http.StatusForbidden, "Invalid CSRF token in session")
	}

	// Получаем токен из запроса (form или header)
	var requestToken string

	// Сначала проверяем header (для HTMX запросов)
	requestToken = c.Request().Header.Get(CSRFHeaderKey)

	// Если нет в header, проверяем форму
	if requestToken == "" {
		requestToken = c.FormValue(CSRFFormKey)
	}

	if requestToken == "" {
		return echo.NewHTTPError(http.StatusForbidden, "CSRF token not provided")
	}

	// Сравниваем токены за постоянное время — обычное сравнение строк
	// завершается на первом несовпавшем байте и позволяет подбирать токен
	// по времени ответа.
	if subtle.ConstantTimeCompare([]byte(sessionTokenStr), []byte(requestToken)) != 1 {
		return echo.NewHTTPError(http.StatusForbidden, "CSRF token mismatch")
	}

	return nil
}

// RegenerateCSRFToken выпускает новый CSRF-токен и кладёт его в сессию,
// делая ранее выданный токен недействительным.
//
// Хранилище сессий cookie-based (session.go:38), серверного session ID не
// существует, поэтому от фиксации сессии (S-02) защищает именно перевыпуск
// токена: полученный до входа токен перестаёт подходить к сессии после входа.
func RegenerateCSRFToken(c echo.Context) (string, error) {
	sess, err := sessionOrNew(c)
	if sess == nil {
		return "", err
	}

	return setFreshCSRFToken(c, sess)
}

// GetCSRFToken возвращает CSRF токен для использования в шаблонах
func GetCSRFToken(c echo.Context) (string, error) {
	// Нечитаемая cookie (чужой/провёрнутый SESSION_SECRET) — не ошибка
	// рендеринга: gorilla отдаёт вместе с ней пригодную новую сессию,
	// в которую ниже кладётся свежий токен. См. ensureCSRFToken.
	sess, err := sessionOrNew(c)
	if sess == nil {
		return "", err
	}

	token, exists := sess.Values[CSRFTokenKey]
	if !exists || token == nil {
		// Генерируем новый токен если его нет
		return setFreshCSRFToken(c, sess)
	}

	tokenStr, ok := token.(string)
	if !ok {
		return "", echo.NewHTTPError(http.StatusInternalServerError, "Invalid CSRF token type")
	}

	return tokenStr, nil
}

// generateCSRFToken генерирует криптографически безопасный токен
func generateCSRFToken() (string, error) {
	bytes := make([]byte, CSRFTokenLength)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}
