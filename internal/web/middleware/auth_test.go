package middleware_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/middleware"
)

func TestRequireAuth_AuthenticatedUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware
	authMiddleware := middleware.RequireAuth()

	// Создаем следующий handler для тестирования
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "protected content")
	}

	// Мокируем аутентифицированного пользователя
	sessionData := &middleware.SessionData{
		UserID: user.NewUser(
			"test@example.com",
			"Test",
			"User",
			user.RoleMember,
		).ID,
		Role:  user.RoleMember,
		Email: "test@example.com",
	}

	// Мокируем успешное получение сессии
	c.Set("mock_session_data", sessionData)

	// Создаем handler с middleware
	handler := authMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Проверяем результат
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "protected content", rec.Body.String())

	// Проверяем, что пользователь был сохранен в контексте
	userData, exists := c.Get("user").(*middleware.SessionData)
	assert.True(t, exists)
	assert.Equal(t, sessionData.Email, userData.Email)
}

func TestRequireAuth_UnauthenticatedUser_RegularRequest(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware
	authMiddleware := middleware.RequireAuth()

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "protected content")
	}

	// Мокируем отсутствие сессии (ошибка получения)
	c.Set("mock_session_error", "no session")

	// Создаем handler с middleware
	handler := authMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Проверяем результат - должен быть редирект
	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusFound, httpErr.Code)
	assert.Equal(t, http.StatusFound, rec.Code)
}

func TestRequireAuth_UnauthenticatedUser_HTMXRequest(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware
	authMiddleware := middleware.RequireAuth()

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "protected content")
	}

	// Мокируем отсутствие сессии
	c.Set("mock_session_error", "no session")

	// Создаем handler с middleware
	handler := authMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Проверяем результат - должен быть HTMX редирект
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Hx-Redirect"))
}

// --- RequireAPIAuth (задача 3 плана docs/plans/20260816-deployment-blockers.md) ---

// apiErrorBody повторяет формат ответа API-хендлеров
// (internal/application/handlers: ErrorResponse/ErrorDetail/ResponseMeta).
// Тест разбирает ответ в эту структуру, чтобы поймать расхождение формата.
type apiErrorBody struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
	Meta struct {
		RequestID string    `json:"request_id"`
		Timestamp time.Time `json:"timestamp"`
		Version   string    `json:"version"`
	} `json:"meta"`
}

func TestRequireAPIAuth_NoSession_Returns401JSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	handler := middleware.RequireAPIAuth()(func(_ echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "api content")
	})

	// Хранилище сессий не подключено — GetSessionData вернёт ошибку,
	// как это происходит с анонимным запросом к /api/v1.
	err := handler(c)

	require.NoError(t, err)
	assert.False(t, nextCalled, "хендлер не должен вызываться без сессии")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body apiErrorBody
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "UNAUTHORIZED", body.Error.Code)
	assert.Equal(t, "Authentication required", body.Error.Message)
	assert.Equal(t, "v1", body.Meta.Version)
	assert.False(t, body.Meta.Timestamp.IsZero(), "meta.timestamp обязан быть заполнен")
}

func TestRequireAPIAuth_NoSession_DoesNotRedirect(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/budgets", nil)
	// Даже с заголовком HTMX API обязан отвечать 401 JSON, а не редиректом.
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware.RequireAPIAuth()(func(c echo.Context) error {
		return c.String(http.StatusOK, "api content")
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Header().Get("Location"), "API не должен редиректить на /login")
	assert.Empty(t, rec.Header().Get("Hx-Redirect"), "API не должен отдавать Hx-Redirect")
}

func TestRequireAPIAuth_NoSession_ContentTypeIsJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	handler := middleware.RequireAPIAuth()(func(c echo.Context) error {
		return c.String(http.StatusOK, "api content")
	})

	require.NoError(t, handler(c))
	contentType := rec.Header().Get(echo.HeaderContentType)
	assert.Contains(t, contentType, echo.MIMEApplicationJSON)
	assert.NotContains(t, contentType, "text/html")
	assert.NotContains(t, rec.Body.String(), "<html")
}

func TestRequireAPIAuth_ValidSession_CallsNext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sessionData := &middleware.SessionData{
		UserID: uuid.New(),
		Role:   user.RoleMember,
		Email:  "member@example.com",
	}
	c.Set("mock_session_data", sessionData)

	nextCalled := false
	handler := middleware.RequireAPIAuth()(func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "api content")
	})

	require.NoError(t, handler(c))
	assert.True(t, nextCalled)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "api content", rec.Body.String())

	// SessionData обязан лежать под тем же ключом "user", что и у RequireAuth,
	// иначе GetUserFromContext в API-хендлерах не найдёт пользователя.
	userData, err := middleware.GetUserFromContext(c)
	require.NoError(t, err)
	assert.Equal(t, sessionData.UserID, userData.UserID)
	assert.Equal(t, sessionData.Email, userData.Email)
}

// TestRequireAPIAuth_RealSessionStore проверяет полный путь через настоящее
// cookie-хранилище: сначала запрос кладёт данные в сессию, затем выданная
// cookie переиспользуется для защищённого маршрута.
func TestRequireAPIAuth_RealSessionStore(t *testing.T) {
	e := echo.New()
	e.Use(middleware.SessionStore("test-secret-key-for-api-auth", false))

	expectedUser := &middleware.SessionData{
		UserID: uuid.New(),
		Role:   user.RoleAdmin,
		Email:  "admin@example.com",
	}

	e.GET("/sign-in", func(c echo.Context) error {
		if err := middleware.SetSessionData(c, expectedUser); err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})

	e.GET("/api/v1/protected", func(c echo.Context) error {
		userData, err := middleware.GetUserFromContext(c)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, userData.Email)
	}, middleware.RequireAPIAuth())

	// Без cookie — 401 JSON.
	anonRec := httptest.NewRecorder()
	e.ServeHTTP(anonRec, httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil))
	require.Equal(t, http.StatusUnauthorized, anonRec.Code)
	assert.Contains(t, anonRec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)

	// Забираем cookie сессии.
	loginRec := httptest.NewRecorder()
	e.ServeHTTP(loginRec, httptest.NewRequest(http.MethodGet, "/sign-in", nil))
	require.Equal(t, http.StatusOK, loginRec.Code)
	cookies := (&http.Response{Header: loginRec.Header()}).Cookies()
	require.NotEmpty(t, cookies, "сессия не выдала cookie")

	// С cookie — запрос проходит.
	authReq := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	for _, cookie := range cookies {
		authReq.AddCookie(cookie)
	}
	authRec := httptest.NewRecorder()
	e.ServeHTTP(authRec, authReq)

	assert.Equal(t, http.StatusOK, authRec.Code)
	assert.Equal(t, expectedUser.Email, authRec.Body.String())
}

func TestRequireRole_ValidRole(t *testing.T) {
	tests := []struct {
		name         string
		userRole     user.Role
		requiredRole user.Role
		shouldPass   bool
	}{
		{
			name:         "Admin accessing admin endpoint",
			userRole:     user.RoleAdmin,
			requiredRole: user.RoleAdmin,
			shouldPass:   true,
		},
		{
			name:         "Member accessing member endpoint",
			userRole:     user.RoleMember,
			requiredRole: user.RoleMember,
			shouldPass:   true,
		},
		{
			name:         "Child accessing child endpoint",
			userRole:     user.RoleChild,
			requiredRole: user.RoleChild,
			shouldPass:   true,
		},
		{
			name:         "Member trying to access admin endpoint",
			userRole:     user.RoleMember,
			requiredRole: user.RoleAdmin,
			shouldPass:   false,
		},
		{
			name:         "Child trying to access member endpoint",
			userRole:     user.RoleChild,
			requiredRole: user.RoleMember,
			shouldPass:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodGet, "/protected", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Создаем middleware с требуемой ролью
			roleMiddleware := middleware.RequireRole(tt.requiredRole)

			// Создаем следующий handler
			nextHandler := func(c echo.Context) error {
				return c.String(http.StatusOK, "role protected content")
			}

			// Мокируем пользователя с определенной ролью
			sessionData := &middleware.SessionData{
				UserID: user.NewUser(
					"test@example.com",
					"Test",
					"User",
					tt.userRole,
				).ID,
				Role:  tt.userRole,
				Email: "test@example.com",
			}
			c.Set("user", sessionData)

			// Создаем handler с middleware
			handler := roleMiddleware(nextHandler)

			// Выполняем запрос
			err := handler(c)

			if tt.shouldPass {
				require.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, "role protected content", rec.Body.String())
			} else {
				// Проверяем что доступ был отклонен
				// Если вернулась ошибка, проверяем ее тип
				if err != nil {
					// Если вернулась ошибка, проверяем ее тип
					var httpErr *echo.HTTPError
					require.ErrorAs(t, err, &httpErr)
					assert.Equal(t, http.StatusForbidden, httpErr.Code)
				} else {
					// Если ошибки нет, проверяем статус код ответа
					assert.Equal(t, http.StatusForbidden, rec.Code)
				}
			}
		})
	}
}

func TestRequireRole_MultipleRoles(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware который принимает админов и обычных пользователей
	roleMiddleware := middleware.RequireRole(user.RoleAdmin, user.RoleMember)

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "multi-role content")
	}

	// Тестируем с пользователем-членом семьи
	sessionData := &middleware.SessionData{
		Role: user.RoleMember,
	}
	c.Set("user", sessionData)

	// Создаем handler с middleware
	handler := roleMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "multi-role content", rec.Body.String())
}

func TestRequireRole_NoUserInContext(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware
	roleMiddleware := middleware.RequireRole(user.RoleAdmin)

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "protected content")
	}

	// НЕ устанавливаем пользователя в контексте

	// Создаем handler с middleware
	handler := roleMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Должен быть ошибка авторизации с редиректом
	var httpErr *echo.HTTPError
	require.ErrorAs(t, err, &httpErr)
	assert.Equal(t, http.StatusFound, httpErr.Code)
	assert.Equal(t, http.StatusFound, rec.Code)
}

func TestRequireRole_HTMXForbidden(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/protected", nil)
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware
	roleMiddleware := middleware.RequireRole(user.RoleAdmin)

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "admin content")
	}

	// Мокируем пользователя с недостаточными правами
	sessionData := &middleware.SessionData{
		Role: user.RoleChild,
	}
	c.Set("user", sessionData)

	// Создаем handler с middleware
	handler := roleMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Для HTMX должен быть JSON response с ошибкой
	require.NoError(t, err)
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "Insufficient permissions")
}

func TestRequireAdmin_Shortcut(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/admin", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем admin middleware
	adminMiddleware := middleware.RequireAdmin()

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "admin content")
	}

	// Мокируем админа
	sessionData := &middleware.SessionData{
		Role: user.RoleAdmin,
	}
	c.Set("user", sessionData)

	// Создаем handler с middleware
	handler := adminMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "admin content", rec.Body.String())
}

func TestRequireAdminOrMember_Shortcut(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/family", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем admin-or-member middleware
	familyMiddleware := middleware.RequireAdminOrMember()

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "family content")
	}

	// Тестируем с обычным пользователем
	sessionData := &middleware.SessionData{
		Role: user.RoleMember,
	}
	c.Set("user", sessionData)

	// Создаем handler с middleware
	handler := familyMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "family content", rec.Body.String())
}

func TestGetUserFromContext_Success(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// Устанавливаем пользователя в контексте
	expectedUser := &middleware.SessionData{
		Role:  user.RoleAdmin,
		Email: "admin@example.com",
	}
	c.Set("user", expectedUser)

	// Получаем пользователя
	userData, err := middleware.GetUserFromContext(c)

	require.NoError(t, err)
	assert.Equal(t, expectedUser, userData)
}

func TestGetUserFromContext_NoUser(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// НЕ устанавливаем пользователя в контексте

	// Пытаемся получить пользователя
	userData, err := middleware.GetUserFromContext(c)

	require.Error(t, err)
	assert.Nil(t, userData)
	assert.Equal(t, echo.ErrUnauthorized, err)
}

func TestRedirectIfAuthenticated_AuthenticatedUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware
	redirectMiddleware := middleware.RedirectIfAuthenticated("/dashboard")

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "login page")
	}

	// Мокируем аутентифицированного пользователя
	c.Set("mock_is_authenticated", true)

	// Создаем handler с middleware
	handler := redirectMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Должен быть редирект на dashboard
	if err != nil {
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusFound, httpErr.Code)
	} else {
		// Проверяем что был установлен редирект заголовок
		assert.Equal(t, http.StatusFound, rec.Code)
	}
}

func TestRedirectIfAuthenticated_UnauthenticatedUser(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/login", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Создаем middleware
	redirectMiddleware := middleware.RedirectIfAuthenticated("/dashboard")

	// Создаем следующий handler
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "login page")
	}

	// Мокируем НЕаутентифицированного пользователя
	c.Set("mock_is_authenticated", false)

	// Создаем handler с middleware
	handler := redirectMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Должен пройти к следующему handler
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "login page", rec.Body.String())
}

// Benchmark тесты для проверки производительности
func BenchmarkRequireAuth(b *testing.B) {
	e := echo.New()
	authMiddleware := middleware.RequireAuth()

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := authMiddleware(nextHandler)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Мокируем успешную аутентификацию
		sessionData := &middleware.SessionData{
			Role: user.RoleMember,
		}
		c.Set("user", sessionData)

		handler(c)
	}
}

func BenchmarkRequireRole(b *testing.B) {
	e := echo.New()
	roleMiddleware := middleware.RequireRole(user.RoleAdmin, user.RoleMember)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := roleMiddleware(nextHandler)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		sessionData := &middleware.SessionData{
			Role: user.RoleMember,
		}
		c.Set("user", sessionData)

		handler(c)
	}
}
