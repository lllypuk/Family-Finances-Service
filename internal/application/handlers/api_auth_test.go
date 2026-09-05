package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/web/middleware"
)

const stubBearerToken = "stub-bearer-token"

// stubAuthenticator принимает только stubBearerToken; failErr имитирует сбой хранилища.
type stubAuthenticator struct {
	principal *auth.Principal
	failErr   error
}

func newStubAuthenticator() *stubAuthenticator {
	return &stubAuthenticator{principal: &auth.Principal{
		SessionID: uuid.New(),
		UserID:    uuid.New(),
		Email:     "bearer@example.com",
		Role:      user.RoleMember,
	}}
}

func (s *stubAuthenticator) Authenticate(_ context.Context, token string) (*auth.Principal, error) {
	if s.failErr != nil {
		return nil, s.failErr
	}
	if token != stubBearerToken {
		return nil, auth.ErrUnauthorized
	}
	return s.principal, nil
}

// --- RequireAPIAuth / RequireAPIRole: авторизация группы /api/v1.
// Раньше эти middleware жили в internal/web/middleware и несли собственную
// копию формата ошибок API; теперь они лежат рядом с хендлерами и используют
// общий ErrorResponse, а тесты сверяют именно его.

func TestRequireAPIAuth_NoSession_Returns401JSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	handler := handlers.RequireAPIAuth(newStubAuthenticator())(func(c echo.Context) error {
		nextCalled = true
		return c.String(http.StatusOK, "api content")
	})

	// Хранилище сессий не подключено — GetSessionData вернёт ошибку,
	// как это происходит с анонимным запросом к /api/v1.
	err := handler(c)

	require.NoError(t, err)
	assert.False(t, nextCalled, "хендлер не должен вызываться без сессии")
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body handlers.ErrorResponse
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

	handler := handlers.RequireAPIAuth(newStubAuthenticator())(func(c echo.Context) error {
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

	handler := handlers.RequireAPIAuth(newStubAuthenticator())(func(c echo.Context) error {
		return c.String(http.StatusOK, "api content")
	})

	require.NoError(t, handler(c))
	contentType := rec.Header().Get(echo.HeaderContentType)
	assert.Contains(t, contentType, echo.MIMEApplicationJSON)
	assert.NotContains(t, contentType, "text/html")
	assert.NotContains(t, rec.Body.String(), "<html")
}

// TestRequireAPIAuth_ValidSession_CallsNext проверяет полный путь через
// настоящее cookie-хранилище: сначала запрос кладёт данные в сессию, затем
// выданная cookie переиспользуется для защищённого маршрута. SessionData
// обязан лечь под тем же ключом, что и у RequireAuth, иначе
// middleware.GetUserFromContext в API-хендлерах не найдёт пользователя.
func TestRequireAPIAuth_ValidSession_CallsNext(t *testing.T) {
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

	var seen *middleware.SessionData
	e.GET("/api/v1/protected", func(c echo.Context) error {
		userData, err := middleware.GetUserFromContext(c)
		if err != nil {
			return err
		}
		seen = userData
		return c.String(http.StatusOK, userData.Email)
	}, handlers.RequireAPIAuth(newStubAuthenticator()))

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
	require.NotNil(t, seen)
	assert.Equal(t, expectedUser.UserID, seen.UserID)
	assert.Equal(t, expectedUser.Role, seen.Role)
}

// --- Bearer-путь RequireAPIAuth (план 03): до удаления веб-слоя оба способа живут рядом.

func TestRequireAPIAuth_ValidBearer_PutsPrincipalAndSessionData(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+stubBearerToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	stub := newStubAuthenticator()
	var seenSession *middleware.SessionData
	var seenPrincipal *auth.Principal
	handler := handlers.RequireAPIAuth(stub)(func(c echo.Context) error {
		sessionData, err := middleware.GetUserFromContext(c)
		if err != nil {
			return err
		}
		principal, err := auth.FromContext(c)
		if err != nil {
			return err
		}
		seenSession, seenPrincipal = sessionData, principal
		return c.NoContent(http.StatusOK)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, stub.principal, seenPrincipal)
	require.NotNil(t, seenSession)
	assert.Equal(t, stub.principal.UserID, seenSession.UserID)
	assert.Equal(t, stub.principal.Role, seenSession.Role)
	assert.Equal(t, stub.principal.Email, seenSession.Email)
}

func TestRequireAPIAuth_InvalidBearer_Returns401JSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer garbage")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	handler := handlers.RequireAPIAuth(newStubAuthenticator())(func(echo.Context) error {
		nextCalled = true
		return nil
	})

	require.NoError(t, handler(c))
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)

	var body handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "UNAUTHORIZED", body.Error.Code)
}

// TestRequireAPIAuth_InvalidBearer_NoCookieFallback — невалидный bearer не откатывается на
// cookie: клиент должен узнать, что его токен отозван, а не жить на старой сессии.
func TestRequireAPIAuth_InvalidBearer_NoCookieFallback(t *testing.T) {
	e := echo.New()
	e.Use(middleware.SessionStore("test-secret-key-for-api-auth", false))
	e.GET("/sign-in", func(c echo.Context) error {
		if err := middleware.SetSessionData(
			c,
			&middleware.SessionData{UserID: uuid.New(), Role: user.RoleAdmin},
		); err != nil {
			return err
		}
		return c.NoContent(http.StatusOK)
	})
	e.GET("/api/v1/protected", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}, handlers.RequireAPIAuth(newStubAuthenticator()))

	loginRec := httptest.NewRecorder()
	e.ServeHTTP(loginRec, httptest.NewRequest(http.MethodGet, "/sign-in", nil))
	cookies := (&http.Response{Header: loginRec.Header()}).Cookies()
	require.NotEmpty(t, cookies)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/protected", nil)
	for _, cookie := range cookies {
		req.AddCookie(cookie)
	}
	req.Header.Set(echo.HeaderAuthorization, "Bearer revoked")
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRequireAPIAuth_BearerStorageFailure_Returns500(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+stubBearerToken)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	stub := newStubAuthenticator()
	stub.failErr = errors.New("database is locked")
	handler := handlers.RequireAPIAuth(stub)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "database is locked")

	var body handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "INTERNAL_ERROR", body.Error.Code)
}

func TestRequireAPIRole_RolesMatrix(t *testing.T) {
	tests := []struct {
		name     string
		role     user.Role
		required []user.Role
		wantCode int
		wantNext bool
	}{
		{
			name:     "admin passes admin-only route",
			role:     user.RoleAdmin,
			required: []user.Role{user.RoleAdmin},
			wantCode: http.StatusOK,
			wantNext: true,
		},
		{
			name:     "member rejected on admin-only route",
			role:     user.RoleMember,
			required: []user.Role{user.RoleAdmin},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "child rejected on admin-only route",
			role:     user.RoleChild,
			required: []user.Role{user.RoleAdmin},
			wantCode: http.StatusForbidden,
		},
		{
			name:     "member passes finance route",
			role:     user.RoleMember,
			required: []user.Role{user.RoleAdmin, user.RoleMember},
			wantCode: http.StatusOK,
			wantNext: true,
		},
		{
			name:     "child rejected on finance route",
			role:     user.RoleChild,
			required: []user.Role{user.RoleAdmin, user.RoleMember},
			wantCode: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := echo.New()
			req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/42", nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			c.Set("user", &middleware.SessionData{
				UserID: uuid.New(),
				Role:   tt.role,
				Email:  "user@example.com",
			})

			nextCalled := false
			handler := handlers.RequireAPIRole(tt.required...)(func(c echo.Context) error {
				nextCalled = true
				return c.NoContent(http.StatusOK)
			})

			require.NoError(t, handler(c))
			assert.Equal(t, tt.wantCode, rec.Code)
			assert.Equal(t, tt.wantNext, nextCalled)
		})
	}
}

func TestRequireAPIRole_Forbidden_ReturnsJSON(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/42", nil)
	// Даже с заголовком HTMX API обязан отвечать JSON, а не HTML "Access denied".
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user", &middleware.SessionData{UserID: uuid.New(), Role: user.RoleChild})

	handler := handlers.RequireAPIRole(user.RoleAdmin)(func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})

	require.NoError(t, handler(c))
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Header().Get(echo.HeaderContentType), echo.MIMEApplicationJSON)
	assert.NotContains(t, rec.Body.String(), "Access denied")

	var body handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "FORBIDDEN", body.Error.Code)
	assert.Equal(t, "Insufficient permissions", body.Error.Message)
	assert.Equal(t, "v1", body.Meta.Version)
	assert.False(t, body.Meta.Timestamp.IsZero())
}

// TestRequireAPIRole_NoSession_Returns401 — порядок middleware может измениться,
// поэтому ролевая проверка без сессии обязана отвечать 401, а не 403 и не
// редиректом на /login, как это делает веб-вариант RequireRole.
func TestRequireAPIRole_NoSession_Returns401(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodDelete, "/api/v1/users/42", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextCalled := false
	handler := handlers.RequireAPIRole(user.RoleAdmin, user.RoleMember)(func(c echo.Context) error {
		nextCalled = true
		return c.NoContent(http.StatusOK)
	})

	require.NoError(t, handler(c))
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Empty(t, rec.Header().Get("Location"))

	var body handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "UNAUTHORIZED", body.Error.Code)
}

// --- RequireAPIActiveUser: перепроверка владельца сессии в БД для /api/v1.

type stubAPISessionLookup struct {
	record *user.User
	err    error
}

func (s *stubAPISessionLookup) GetUserByID(context.Context, uuid.UUID) (*user.User, error) {
	return s.record, s.err
}

// TestRequireAPIActiveUser_TransientDBError_Returns500 — сбой БД это 500, а не
// 401: 401 заставил бы клиента выбросить рабочую сессию и перелогиниться.
func TestRequireAPIActiveUser_TransientDBError_Returns500(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user", &middleware.SessionData{
		UserID: uuid.New(),
		Role:   user.RoleMember,
		Email:  "member@example.com",
	})

	nextCalled := false
	handler := handlers.RequireAPIActiveUser(&stubAPISessionLookup{
		err: errors.New("database is locked"),
	})(func(c echo.Context) error {
		nextCalled = true
		return c.NoContent(http.StatusOK)
	})

	require.NoError(t, handler(c))
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
	assert.NotContains(t, rec.Body.String(), "database is locked", "текст ошибки БД наружу не отдаём")

	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	errBody, ok := body["error"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "INTERNAL_ERROR", errBody["code"])
}

// TestRequireAPIActiveUser_DeletedUser_Returns401 — пользователя нет: 401.
func TestRequireAPIActiveUser_DeletedUser_Returns401(t *testing.T) {
	e := echo.New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/transactions", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	c.Set("user", &middleware.SessionData{
		UserID: uuid.New(),
		Role:   user.RoleMember,
		Email:  "deleted@example.com",
	})

	nextCalled := false
	handler := handlers.RequireAPIActiveUser(&stubAPISessionLookup{
		err: user.ErrNotFound,
	})(func(c echo.Context) error {
		nextCalled = true
		return c.NoContent(http.StatusOK)
	})

	require.NoError(t, handler(c))
	assert.False(t, nextCalled)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}
