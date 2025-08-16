package middleware_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"

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
			user.NewFamily("Test Family", "USD").ID,
			user.RoleMember,
		).ID,
		FamilyID: user.NewFamily("Test Family", "USD").ID,
		Role:     user.RoleMember,
		Email:    "test@example.com",
	}

	// Мокируем успешное получение сессии
	c.Set("mock_session_data", sessionData)

	// Создаем handler с middleware
	handler := authMiddleware(nextHandler)

	// Выполняем запрос
	err := handler(c)

	// Проверяем результат
	assert.NoError(t, err)
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
	assert.IsType(t, &echo.HTTPError{}, err)
	httpErr := func() *echo.HTTPError {
		target := &echo.HTTPError{}
		_ = errors.As(err, &target)
		return target
	}()
	assert.Equal(t, http.StatusFound, httpErr.Code)
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
	assert.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Equal(t, "/login", rec.Header().Get("Hx-Redirect"))
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
					user.NewFamily("Test Family", "USD").ID,
					tt.userRole,
				).ID,
				FamilyID: user.NewFamily("Test Family", "USD").ID,
				Role:     tt.userRole,
				Email:    "test@example.com",
			}
			c.Set("user", sessionData)

			// Создаем handler с middleware
			handler := roleMiddleware(nextHandler)

			// Выполняем запрос
			err := handler(c)

			if tt.shouldPass {
				assert.NoError(t, err)
				assert.Equal(t, http.StatusOK, rec.Code)
				assert.Equal(t, "role protected content", rec.Body.String())
			} else {
				// Проверяем что доступ был отклонен
				if err != nil {
					// Если вернулась ошибка, проверяем ее тип
					httpErr := func() *echo.HTTPError {
						target := &echo.HTTPError{}
						_ = errors.As(err, &target)
						return target
					}()
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

	assert.NoError(t, err)
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

	// Должен быть редирект на логин или ошибка авторизации
	if err != nil {
		assert.IsType(t, &echo.HTTPError{}, err)
		httpErr := func() *echo.HTTPError {
			target := &echo.HTTPError{}
			_ = errors.As(err, &target)
			return target
		}()
		assert.Equal(t, http.StatusFound, httpErr.Code)
	} else {
		// Если ошибки нет, проверяем код ответа
		assert.Equal(t, http.StatusFound, rec.Code)
	}
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
	assert.NoError(t, err)
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

	assert.NoError(t, err)
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

	assert.NoError(t, err)
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

	assert.NoError(t, err)
	assert.Equal(t, expectedUser, userData)
}

func TestGetUserFromContext_NoUser(t *testing.T) {
	e := echo.New()
	c := e.NewContext(nil, nil)

	// НЕ устанавливаем пользователя в контексте

	// Пытаемся получить пользователя
	userData, err := middleware.GetUserFromContext(c)

	assert.Error(t, err)
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
		assert.IsType(t, &echo.HTTPError{}, err)
		httpErr := func() *echo.HTTPError {
			target := &echo.HTTPError{}
			_ = errors.As(err, &target)
			return target
		}()
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
	assert.NoError(t, err)
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

	for range b.N {
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

	for range b.N {
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
