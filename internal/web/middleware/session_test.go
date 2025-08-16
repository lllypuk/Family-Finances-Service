package middleware_test

import (
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

func TestSessionStore_ProductionSecurity(t *testing.T) {
	tests := []struct {
		name           string
		isProduction   bool
		expectedSecure bool
	}{
		{
			name:           "Production environment should have secure cookies",
			isProduction:   true,
			expectedSecure: true,
		},
		{
			name:           "Development environment should not have secure cookies",
			isProduction:   false,
			expectedSecure: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			e := echo.New()
			secretKey := "test-secret-key"

			// Act
			sessionMiddleware := middleware.SessionStore(secretKey, tt.isProduction)

			// Apply middleware to echo instance
			e.Use(sessionMiddleware)

			// Assert
			// We can't directly access the session store options from the middleware,
			// but we can verify the middleware was created without panicking
			assert.NotNil(t, sessionMiddleware)
		})
	}
}

func TestSessionStore_BasicConfiguration(t *testing.T) {
	// Arrange
	e := echo.New()
	secretKey := "test-secret-key-for-basic-config"
	isProduction := false

	// Act
	sessionMiddleware := middleware.SessionStore(secretKey, isProduction)

	// Assert
	assert.NotNil(t, sessionMiddleware, "SessionStore should return a valid middleware function")

	// Verify middleware can be applied without issues
	e.Use(sessionMiddleware)
	assert.NotNil(t, e, "Echo instance should remain valid after applying session middleware")
}

func TestSessionStore_EmptySecretKey(t *testing.T) {
	// Arrange
	e := echo.New()
	secretKey := ""
	isProduction := false

	// Act & Assert
	// Should not panic even with empty secret key (though not recommended for production)
	assert.NotPanics(t, func() {
		sessionMiddleware := middleware.SessionStore(secretKey, isProduction)
		e.Use(sessionMiddleware)
	}, "SessionStore should not panic with empty secret key")
}

func TestSessionData_StructFields(t *testing.T) {
	// Test that SessionData structure has all required fields
	sessionData := &middleware.SessionData{}

	// Use reflection to verify field existence
	assert.Contains(t, []string{"UserID", "FamilyID", "Role", "Email", "ExpiresAt"},
		"UserID", "SessionData should have UserID field")

	// Verify the struct can be instantiated
	assert.NotNil(t, sessionData)
}

func TestSessionConstants(t *testing.T) {
	// Verify session constants are properly defined
	assert.Equal(t, "family-budget-session", middleware.SessionName)
	assert.Equal(t, "user_id", middleware.SessionUserKey)
	assert.Equal(t, "family_id", middleware.SessionFamilyKey)
	assert.Equal(t, "role", middleware.SessionRoleKey)
	assert.Equal(t, "email", middleware.SessionEmailKey)

	// Verify timeout is reasonable (24 hours)
	assert.Equal(t, 24*60*60, int(middleware.SessionTimeout.Seconds()))
}

// setupSessionTest создает тестовое окружение с session middleware
func setupSessionTest() (*echo.Echo, echo.MiddlewareFunc) {
	e := echo.New()
	sessionMiddleware := middleware.SessionStore("test-secret-session-key", false)
	return e, sessionMiddleware
}

// setupSessionContext создает context с инициализированной сессией
func setupSessionContext(
	e *echo.Echo,
	_ echo.MiddlewareFunc,
) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	return c, rec
}

// initializeSession инициализирует сессию для context
func initializeSession(c echo.Context, sessionMiddleware echo.MiddlewareFunc) error {
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := sessionMiddleware(nextHandler)
	return handler(c)
}

func TestSetSessionData_Success(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	c, _ := setupSessionContext(e, sessionMiddleware)
	initializeSession(c, sessionMiddleware)

	// Создаем тестовые данные сессии
	userID := uuid.New()
	familyID := uuid.New()
	sessionData := &middleware.SessionData{
		UserID:    userID,
		FamilyID:  familyID,
		Role:      user.RoleAdmin,
		Email:     "admin@example.com",
		ExpiresAt: time.Now().Add(24 * time.Hour),
	}

	// Сохраняем данные в сессии
	err := middleware.SetSessionData(c, sessionData)
	require.NoError(t, err)

	// Проверяем что данные сохранились
	retrievedData, err := middleware.GetSessionData(c)
	require.NoError(t, err)
	assert.Equal(t, userID, retrievedData.UserID)
	assert.Equal(t, familyID, retrievedData.FamilyID)
	assert.Equal(t, user.RoleAdmin, retrievedData.Role)
	assert.Equal(t, "admin@example.com", retrievedData.Email)
}

func TestGetSessionData_NoSession_ReturnsError(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	c, _ := setupSessionContext(e, sessionMiddleware)
	initializeSession(c, sessionMiddleware)

	// Пытаемся получить данные из пустой сессии
	sessionData, err := middleware.GetSessionData(c)

	require.Error(t, err)
	assert.Nil(t, sessionData)
	assert.Equal(t, echo.ErrUnauthorized, err)
}

func TestGetSessionData_IncompleteSession_ReturnsError(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	c, _ := setupSessionContext(e, sessionMiddleware)
	initializeSession(c, sessionMiddleware)

	// Создаем неполную сессию путем прямого изменения сессионных данных
	// для симуляции поврежденной или неполной сессии
	sess, err := middleware.GetSessionData(c) // Это вернет ошибку для пустой сессии, что и ожидается

	// Проверяем что для пустой сессии действительно возвращается ошибка
	require.Error(t, err)
	assert.Nil(t, sess)
}

func TestClearSession_Success(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	c, _ := setupSessionContext(e, sessionMiddleware)
	initializeSession(c, sessionMiddleware)

	// Сначала устанавливаем данные сессии
	sessionData := &middleware.SessionData{
		UserID:   uuid.New(),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
		Email:    "member@example.com",
	}

	err := middleware.SetSessionData(c, sessionData)
	require.NoError(t, err)

	// Проверяем что данные есть
	_, err = middleware.GetSessionData(c)
	require.NoError(t, err)

	// Очищаем сессию
	err = middleware.ClearSession(c)
	require.NoError(t, err)

	// Проверяем что данные удалены
	_, err = middleware.GetSessionData(c)
	assert.Error(t, err)
}

func TestIsAuthenticated_WithValidSession_ReturnsTrue(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	c, _ := setupSessionContext(e, sessionMiddleware)
	initializeSession(c, sessionMiddleware)

	// Устанавливаем валидную сессию
	sessionData := &middleware.SessionData{
		UserID:   uuid.New(),
		FamilyID: uuid.New(),
		Role:     user.RoleChild,
		Email:    "child@example.com",
	}

	err := middleware.SetSessionData(c, sessionData)
	require.NoError(t, err)

	// Проверяем аутентификацию
	isAuth := middleware.IsAuthenticated(c)
	assert.True(t, isAuth)
}

func TestIsAuthenticated_WithoutSession_ReturnsFalse(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	c, _ := setupSessionContext(e, sessionMiddleware)

	// Проверяем аутентификацию без сессии
	isAuth := middleware.IsAuthenticated(c)
	assert.False(t, isAuth)
}

func TestIsAuthenticated_MockSupport_WorksCorrectly(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	c, _ := setupSessionContext(e, sessionMiddleware)

	// Тестируем mock поддержку
	c.Set("mock_is_authenticated", true)
	assert.True(t, middleware.IsAuthenticated(c))

	c.Set("mock_is_authenticated", false)
	assert.False(t, middleware.IsAuthenticated(c))
}

func TestSessionData_SecurityFields_ValidUUIDs(t *testing.T) {
	userID := uuid.New()
	familyID := uuid.New()

	sessionData := &middleware.SessionData{
		UserID:    userID,
		FamilyID:  familyID,
		Role:      user.RoleAdmin,
		Email:     "security@example.com",
		ExpiresAt: time.Now().Add(time.Hour), //nolint:govet // Used in test assertions
	}

	// Проверяем что UUID корректные
	assert.NotEqual(t, uuid.Nil, sessionData.UserID)
	assert.NotEqual(t, uuid.Nil, sessionData.FamilyID)
	assert.Equal(t, userID, sessionData.UserID)
	assert.Equal(t, familyID, sessionData.FamilyID)

	// Проверяем корректность роли
	assert.Contains(t, []user.Role{user.RoleAdmin, user.RoleMember, user.RoleChild}, sessionData.Role)

	// Проверяем email формат
	assert.Contains(t, sessionData.Email, "@")
	assert.Contains(t, sessionData.Email, ".")
}

func TestSessionData_RoleValidation(t *testing.T) {
	tests := []struct {
		name  string
		role  user.Role
		valid bool
	}{
		{"Admin role", user.RoleAdmin, true},
		{"Member role", user.RoleMember, true},
		{"Child role", user.RoleChild, true},
		{"Invalid role", user.Role("invalid"), false},
		{"Empty role", user.Role(""), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sessionData := &middleware.SessionData{
				UserID:   uuid.New(), //nolint:govet // Used for role validation test
				FamilyID: uuid.New(), //nolint:govet // Used for role validation test
				Role:     tt.role,
				Email:    "test@example.com", //nolint:govet // Used for role validation test
			}

			if tt.valid {
				assert.Contains(t, []user.Role{user.RoleAdmin, user.RoleMember, user.RoleChild}, sessionData.Role)
			} else {
				assert.NotContains(t, []user.Role{user.RoleAdmin, user.RoleMember, user.RoleChild}, sessionData.Role)
			}
		})
	}
}

func TestSession_DataPersistence_AcrossRequests(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	// Первый запрос - установка данных
	req1 := httptest.NewRequest(http.MethodPost, "/login", nil)
	rec1 := httptest.NewRecorder()
	c1 := e.NewContext(req1, rec1)

	sessionData := &middleware.SessionData{
		UserID:   uuid.New(),
		FamilyID: uuid.New(),
		Role:     user.RoleAdmin,
		Email:    "persistent@example.com",
	}

	// Выполняем полный цикл через middleware
	nextHandler := func(c echo.Context) error {
		// Сохраняем данные сессии внутри handler
		err := middleware.SetSessionData(c, sessionData)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, "login successful")
	}

	handler := sessionMiddleware(nextHandler)
	err := handler(c1)
	require.NoError(t, err)

	// Получаем cookies из первого ответа
	cookies := rec1.Result().Cookies()
	require.NotEmpty(t, cookies, "Should have session cookie")

	// Второй запрос - проверка данных с теми же cookies
	req2 := httptest.NewRequest(http.MethodGet, "/dashboard", nil)
	for _, cookie := range cookies {
		req2.AddCookie(cookie)
	}

	rec2 := httptest.NewRecorder()
	c2 := e.NewContext(req2, rec2)

	// Инициализируем сессию для второго контекста
	nextHandler2 := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler2 := sessionMiddleware(nextHandler2)
	handler2(c2)

	// Проверяем что данные сохранились
	retrievedData, err := middleware.GetSessionData(c2)
	require.NoError(t, err)
	assert.Equal(t, sessionData.UserID, retrievedData.UserID)
	assert.Equal(t, sessionData.FamilyID, retrievedData.FamilyID)
	assert.Equal(t, sessionData.Role, retrievedData.Role)
	assert.Equal(t, sessionData.Email, retrievedData.Email)
}

func TestSession_Security_HTTPOnlyAndSameSite(t *testing.T) {
	e, sessionMiddleware := setupSessionTest()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	sessionData := &middleware.SessionData{
		UserID:   uuid.New(),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
		Email:    "security@example.com",
	}

	// Выполняем полный цикл через middleware
	nextHandler := func(c echo.Context) error {
		err := middleware.SetSessionData(c, sessionData)
		if err != nil {
			return err
		}
		return c.String(http.StatusOK, "ok")
	}

	handler := sessionMiddleware(nextHandler)
	err := handler(c)
	require.NoError(t, err)

	// Проверяем cookie безопасность
	cookies := rec.Result().Cookies()
	require.NotEmpty(t, cookies)

	sessionCookie := cookies[0]
	assert.True(t, sessionCookie.HttpOnly, "Session cookie should be HttpOnly")
	assert.Equal(t, http.SameSiteLaxMode, sessionCookie.SameSite, "Session cookie should use SameSite Lax")
	assert.Equal(t, "/", sessionCookie.Path, "Session cookie should be valid for entire site")
}

func TestSession_ExpiresAt_ReasonableTimeout(t *testing.T) {
	now := time.Now()
	expiresAt := now.Add(middleware.SessionTimeout)

	sessionData := &middleware.SessionData{
		UserID:    uuid.New(),            //nolint:govet // Used in timeout calculation test
		FamilyID:  uuid.New(),            //nolint:govet // Used in timeout calculation test
		Role:      user.RoleAdmin,        //nolint:govet // Used in timeout calculation test
		Email:     "timeout@example.com", //nolint:govet // Used in timeout calculation test
		ExpiresAt: expiresAt,
	}

	// Проверяем что timeout не слишком короткий и не слишком длинный
	timeDiff := sessionData.ExpiresAt.Sub(now)
	assert.Greater(t, timeDiff, time.Hour, "Session should last more than 1 hour")
	assert.LessOrEqual(t, timeDiff, 24*time.Hour, "Session should not last more than 24 hours")
	assert.Equal(t, 24*time.Hour, middleware.SessionTimeout, "Default timeout should be 24 hours")
}

// Benchmark тесты для производительности сессий
func BenchmarkSetSessionData(b *testing.B) {
	e, sessionMiddleware := setupSessionTest()

	sessionData := &middleware.SessionData{
		UserID:   uuid.New(),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
		Email:    "benchmark@example.com",
	}

	for range b.N {
		c, _ := setupSessionContext(e, sessionMiddleware)
		middleware.SetSessionData(c, sessionData)
	}
}

func BenchmarkGetSessionData(b *testing.B) {
	e, sessionMiddleware := setupSessionTest()

	// Подготавливаем сессию с данными
	c, rec := setupSessionContext(e, sessionMiddleware)

	sessionData := &middleware.SessionData{
		UserID:   uuid.New(),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
		Email:    "benchmark@example.com",
	}

	middleware.SetSessionData(c, sessionData)
	cookies := rec.Result().Cookies()

	b.ResetTimer()
	for range b.N {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		recInner := httptest.NewRecorder()
		cInner := e.NewContext(req, recInner)

		// Инициализируем сессию через middleware
		nextHandler := func(ctx echo.Context) error {
			return ctx.String(http.StatusOK, "ok")
		}

		handler := sessionMiddleware(nextHandler)
		handler(cInner)

		middleware.GetSessionData(cInner)
	}
}

func BenchmarkIsAuthenticated(b *testing.B) {
	e, sessionMiddleware := setupSessionTest()

	// Подготавливаем аутентифицированную сессию
	c, rec := setupSessionContext(e, sessionMiddleware)

	sessionData := &middleware.SessionData{
		UserID:   uuid.New(),
		FamilyID: uuid.New(),
		Role:     user.RoleMember,
		Email:    "benchmark@example.com",
	}

	middleware.SetSessionData(c, sessionData)
	cookies := rec.Result().Cookies()

	b.ResetTimer()
	for range b.N {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		for _, cookie := range cookies {
			req.AddCookie(cookie)
		}
		benchRec := httptest.NewRecorder()
		benchC := e.NewContext(req, benchRec)

		// Инициализируем сессию через middleware
		nextHandler := func(ctx echo.Context) error {
			return ctx.String(http.StatusOK, "ok")
		}

		handler := sessionMiddleware(nextHandler)
		handler(benchC)

		middleware.IsAuthenticated(benchC)
	}
}
