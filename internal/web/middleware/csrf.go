package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"net/http"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo-contrib/session"
	"github.com/labstack/echo/v4"
)

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
				if c.Request().Header.Get("Hx-Request") == HTMXRequestValue {
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

// ensureCSRFToken гарантирует наличие CSRF токена в сессии
func ensureCSRFToken(c echo.Context) error {
	sess, err := getSession(c)
	if err != nil {
		return err
	}

	// Проверяем, есть ли уже токен
	if token, exists := sess.Values[CSRFTokenKey]; exists && token != nil {
		return nil
	}

	// Генерируем новый токен
	token, err := generateCSRFToken()
	if err != nil {
		return err
	}

	sess.Values[CSRFTokenKey] = token
	return sess.Save(c.Request(), c.Response())
}

// validateCSRFToken проверяет CSRF токен
func validateCSRFToken(c echo.Context) error {
	sess, err := getSession(c)
	if err != nil {
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

	// Сравниваем токены
	if sessionTokenStr != requestToken {
		return echo.NewHTTPError(http.StatusForbidden, "CSRF token mismatch")
	}

	return nil
}

// GetCSRFToken возвращает CSRF токен для использования в шаблонах
func GetCSRFToken(c echo.Context) (string, error) {
	sess, err := getSession(c)
	if err != nil {
		return "", err
	}

	token, exists := sess.Values[CSRFTokenKey]
	if !exists || token == nil {
		// Генерируем новый токен если его нет
		newToken, tokenErr := generateCSRFToken()
		if tokenErr != nil {
			return "", tokenErr
		}

		sess.Values[CSRFTokenKey] = newToken
		if saveErr := sess.Save(c.Request(), c.Response()); saveErr != nil {
			return "", saveErr
		}

		return newToken, nil
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

// getSession - вспомогательная функция для получения сессии
func getSession(c echo.Context) (*sessions.Session, error) {
	return session.Get(SessionName, c)
}
