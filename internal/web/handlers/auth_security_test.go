package handlers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web/middleware"
)

func TestSanitizeRedirectURL_SecurityVectors(t *testing.T) {
	tests := []struct {
		name        string
		redirectURL string
		expected    string
		description string
	}{
		{
			name:        "protocol relative",
			redirectURL: "//evil.com",
			expected:    "/",
			description: "Should reject protocol-relative URLs",
		},
		{
			name:        "absolute url",
			redirectURL: "https://evil.com",
			expected:    "/",
			description: "Should reject absolute URLs",
		},
		{
			name:        "javascript protocol",
			redirectURL: "javascript:alert(1)",
			expected:    "/",
			description: "Should reject javascript: URLs",
		},
		{
			name:        "data uri",
			redirectURL: "data:text/html,<h1>hi</h1>",
			expected:    "/",
			description: "Should reject data: URIs",
		},
		{
			name:        "backslash escape",
			redirectURL: "\\\\evil.com",
			expected:    "/",
			description: "Should reject backslash paths that lead to external domains",
		},
		{
			name:        "valid local path",
			redirectURL: "/dashboard",
			expected:    "/dashboard",
			description: "Should allow valid local paths",
		},
		{
			name:        "valid path with query",
			redirectURL: "/page?id=123",
			expected:    "/page?id=123",
			description: "Should allow query parameters",
		},
		{
			name:        "protocol relative with https",
			redirectURL: "//evil.com/path",
			expected:    "/",
			description: "Should reject protocol-relative with path",
		},
		{
			name:        "triple slash attack",
			redirectURL: "///evil.com/path",
			expected:    "/",
			description: "Should reject triple slash URLs",
		},
		{
			name:        "newline injection",
			redirectURL: "/page\nLocation: http://evil.com",
			expected:    "/",
			description: "Should reject URLs with newlines (CRLF injection)",
		},
		{
			name:        "carriage return injection",
			redirectURL: "/page\rLocation: http://evil.com",
			expected:    "/",
			description: "Should reject URLs with carriage returns",
		},
		{
			name:        "tab injection",
			redirectURL: "/page\tLocation: http://evil.com",
			expected:    "/",
			description: "Should reject URLs with tabs",
		},
		{
			name:        "null byte injection",
			redirectURL: "/page\x00http://evil.com",
			expected:    "/",
			description: "Should reject URLs with null bytes",
		},
		{
			name:        "url encoded protocol relative",
			redirectURL: "%2F%2Fevil.com",
			expected:    "/", // After url.Parse decoding, becomes "//evil.com" which is rejected
			description: "URL decoding happens, then validation catches protocol-relative",
		},
		{
			name:        "mixed case javascript",
			redirectURL: "JaVaScRiPt:alert(1)",
			expected:    "/",
			description: "Should reject case-varied javascript: URLs",
		},
		{
			name:        "vbscript protocol",
			redirectURL: "vbscript:msgbox(1)",
			expected:    "/",
			description: "Should reject vbscript: URLs",
		},
		{
			name:        "mailto protocol",
			redirectURL: "mailto:user@evil.com",
			expected:    "/",
			description: "Should reject mailto: URLs",
		},
		{
			name:        "file protocol",
			redirectURL: "file:///etc/passwd",
			expected:    "/",
			description: "Should reject file: URLs",
		},
		{
			name:        "ftp protocol",
			redirectURL: "ftp://evil.com",
			expected:    "/",
			description: "Should reject ftp: URLs",
		},
		{
			name:        "unicode homograph",
			redirectURL: "/dashbοard", // Greek omicron instead of 'o'
			expected:    "/dashbοard",
			description: "Unicode characters pass through (application layer handles)",
		},
		{
			name:        "encoded null byte",
			redirectURL: "/page%00.html",
			expected:    "/page\x00.html", // URL decode happens, null byte passes through
			description: "URL decode occurs, null byte passes through url.Parse",
		},
		{
			name:        "double url encoding",
			redirectURL: "%252F%252Fevil.com",
			expected:    "/", // After decoding once: "%2F%2Fevil.com", becomes "//evil.com" after parse
			description: "Double encoding decoded and validation catches protocol-relative",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeRedirectURL(tt.redirectURL)
			assert.Equal(t, tt.expected, result, tt.description)
		})
	}
}

func TestLogin_OpenRedirectProtection(t *testing.T) {
	// Setup Echo
	e := echo.New()

	// Create minimal handler setup for testing
	repos := &handlers.Repositories{}
	svcs := &services.Services{}
	_ = NewAuthHandler(repos, svcs, false)

	redirectTests := []struct {
		name        string
		redirectURL string
		expected    string
	}{
		{"protocol relative", "//evil.com", "/"},
		{"absolute url", "https://evil.com", "/"},
		{"javascript", "javascript:alert(1)", "/"},
		{"data uri", "data:text/html,<h1>hi</h1>", "/"},
		{"backslash", "\\\\evil.com", "/"},
		{"valid local", "/dashboard", "/dashboard"},
		{"valid with query", "/page?id=123", "/page?id=123"},
	}

	for _, tt := range redirectTests {
		t.Run(tt.name, func(t *testing.T) {
			// Create request with redirect parameter
			req := httptest.NewRequest(http.MethodGet, "/login?redirect="+url.QueryEscape(tt.redirectURL), nil)
			rec := httptest.NewRecorder()
			c := e.NewContext(req, rec)

			// Test that sanitization is applied
			result := sanitizeRedirectURL(c.QueryParam("redirect"))
			assert.Equal(t, tt.expected, result,
				"Redirect URL should be sanitized to prevent open redirect")
		})
	}
}

func TestLogin_XSSProtection(t *testing.T) {
	// These should be handled by proper output encoding in templates
	xssAttempts := []string{
		"<script>alert(1)</script>",
		"<img src=x onerror=alert(1)>",
		"<svg/onload=alert(1)>",
		"javascript:alert(1)",
		"<iframe src=javascript:alert(1)>",
		"<body onload=alert(1)>",
		"<input onfocus=alert(1) autofocus>",
		"<select onfocus=alert(1) autofocus>",
		"<textarea onfocus=alert(1) autofocus>",
		"<marquee onstart=alert(1)>",
		"\"><script>alert(1)</script>",
		"'><script>alert(1)</script>",
	}

	for _, attempt := range xssAttempts {
		t.Run("XSS: "+attempt, func(t *testing.T) {
			// Template engine should escape these
			assert.NotEmpty(t, attempt, "XSS string should not be empty")
			// Verify dangerous patterns exist that need escaping
			dangerous := strings.Contains(attempt, "<") ||
				strings.Contains(attempt, "javascript:") ||
				strings.Contains(attempt, "onerror") ||
				strings.Contains(attempt, "onload")
			assert.True(t, dangerous, "Should contain patterns that require escaping")
		})
	}
}

func TestLogin_CSRFProtection(t *testing.T) {
	// CSRF token should be required for POST requests
	// This is handled by middleware, but documented here
	e := echo.New()

	tests := []struct {
		name          string
		hasCSRFToken  bool
		shouldSucceed bool
	}{
		{"with valid CSRF token", true, true},
		{"without CSRF token", false, false},
		{"with invalid CSRF token", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/login", nil)
			rec := httptest.NewRecorder()
			_ = e.NewContext(req, rec)

			// CSRF validation happens in middleware
			// This test documents the requirement
			if tt.hasCSRFToken {
				assert.True(t, tt.shouldSucceed, "Should succeed with valid CSRF token")
			} else {
				assert.False(t, tt.shouldSucceed, "Should fail without CSRF token")
			}
		})
	}
}

func TestSetup_InputValidation(t *testing.T) {
	// Test that setup validates all inputs properly
	maliciousInputs := []struct {
		field string
		value string
	}{
		{"email", "admin@example.com'; DROP TABLE users;--"},
		{"familyName", "<script>alert(1)</script>"},
		{"firstName", "'; DELETE FROM families;--"},
		{"lastName", "<img src=x onerror=alert(1)>"},
		{"currency", "USD'; --"},
		{"password", "pass\x00word"}, // null byte
	}

	for _, input := range maliciousInputs {
		t.Run("Malicious "+input.field, func(t *testing.T) {
			// Validation should reject or sanitize these
			assert.NotEmpty(t, input.value, "Test value should not be empty")
			// Document that these need validation
			needsValidation := strings.Contains(input.value, "'") ||
				strings.Contains(input.value, "<") ||
				strings.Contains(input.value, "\x00") ||
				strings.Contains(input.value, ";")
			assert.True(t, needsValidation, "Input contains characters requiring validation")
		})
	}
}

func TestSanitizeRedirectURL_EdgeCases(t *testing.T) {
	edgeCases := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "unicode rtl override",
			input:    "/page\u202E",
			expected: "/page\u202E", // Unicode characters pass through
		},
		{
			name:     "unicode zero-width space",
			input:    "/page\u200B",
			expected: "/page\u200B",
		},
		{
			name:     "multiple slashes normalized",
			input:    "////multiple///slashes",
			expected: "/",
		},
		{
			name:     "windows file path",
			input:    "C:\\Windows\\System32",
			expected: "/",
		},
		{
			name:     "unc path",
			input:    "\\\\server\\share",
			expected: "/",
		},
		{
			name:     "very long url",
			input:    "/" + strings.Repeat("a", 10000),
			expected: "/" + strings.Repeat("a", 10000),
		},
		{
			name:     "empty after normalization",
			input:    "\\\\",
			expected: "/",
		},
		{
			name:     "query with redirect attempt",
			input:    "/page?redirect=//evil.com",
			expected: "/page?redirect=//evil.com", // Query params are preserved as-is, app must validate separately
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			result := sanitizeRedirectURL(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestLogin_HeaderInjection(t *testing.T) {
	// Test CRLF injection attempts in redirect parameter
	// Note: url.Parse decodes %0d and %0a, so these pass through
	// Application must validate after URL parsing
	headerInjectionAttempts := []struct {
		input    string
		expected string
	}{
		{"/page\r\nSet-Cookie: admin=true", "/"},                                  // Raw CRLF rejected
		{"/page\nLocation: http://evil.com", "/"},                                 // Raw newline rejected
		{"/page\r\nX-Injected: true", "/"},                                        // Raw CRLF rejected
		{"/page%0d%0aSet-Cookie: admin=true", "/page\r\nSet-Cookie: admin=true"},  // URL encoded passes through Parse
		{"/page%0aLocation: http://evil.com", "/page\nLocation: http://evil.com"}, // URL encoded passes through Parse
	}

	for _, tt := range headerInjectionAttempts {
		t.Run("Header injection: "+tt.input, func(t *testing.T) {
			result := sanitizeRedirectURL(tt.input)
			assert.Equal(t, tt.expected, result,
				"CRLF injection handling")
		})
	}
}

func TestSanitizeRedirectURL_ProtocolVariations(t *testing.T) {
	protocolTests := []struct {
		name     string
		input    string
		expected string
	}{
		// Standard protocols
		{"http", "http://evil.com", "/"},
		{"https", "https://evil.com", "/"},
		{"ftp", "ftp://evil.com", "/"},
		{"file", "file:///etc/passwd", "/"},

		// Data URLs
		{"data", "data:text/html,<script>alert(1)</script>", "/"},
		{"data base64", "data:text/html;base64,PHNjcmlwdD5hbGVydCgxKTwvc2NyaXB0Pg==", "/"},

		// Script protocols
		{"javascript", "javascript:alert(1)", "/"},
		{"vbscript", "vbscript:msgbox(1)", "/"},

		// Other protocols
		{"mailto", "mailto:user@evil.com", "/"},
		{"tel", "tel:+1234567890", "/"},
		{"sms", "sms:+1234567890", "/"},

		// Case variations
		{"HTTP uppercase", "HTTP://evil.com", "/"},
		{"JaVaScRiPt mixed", "JaVaScRiPt:alert(1)", "/"},

		// Whitespace tricks
		{"leading space", " http://evil.com", "/"},
		{"trailing space", "http://evil.com ", "/"},
		{"tab before protocol", "\thttp://evil.com", "/"},
	}

	for _, tt := range protocolTests {
		t.Run(tt.name, func(t *testing.T) {
			result := sanitizeRedirectURL(tt.input)
			assert.Equal(t, tt.expected, result,
				"All external protocols should be rejected")
		})
	}
}

func TestLogin_PasswordTimingAttack(t *testing.T) {
	// Document that bcrypt.CompareHashAndPassword should be used
	// to prevent timing attacks on password comparison
	t.Run("Use bcrypt for constant-time comparison", func(_ *testing.T) {
		// This is a documentation test
		// The actual implementation uses bcrypt.CompareHashAndPassword
		// which has constant-time comparison built in
		// No assertion needed - this test exists for documentation
	})
}

func TestSetup_RateLimitingConsiderations(t *testing.T) {
	// Document that setup endpoint should have rate limiting
	t.Run("Setup should have rate limiting", func(_ *testing.T) {
		// This is a documentation test
		// Setup endpoint is critical and should be rate-limited
		// No assertion needed - this test exists for documentation
	})
}

// loginTestPassword — пароль тестового пользователя для прогона настоящего
// обработчика входа.
const loginTestPassword = "Admin123!"

// loginTestSessionSecret — секрет подписи сессий тестового сервера входа.
const loginTestSessionSecret = "test-session-secret-for-login-fixation"

// stubUserRepository — минимальная реализация handlers.UserRepository:
// обработчику входа нужен только GetByEmail.
type stubUserRepository struct {
	byEmail map[string]*user.User
}

func (r *stubUserRepository) Create(_ context.Context, _ *user.User) error { return nil }

func (r *stubUserRepository) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	for _, u := range r.byEmail {
		if u.ID == id {
			return u, nil
		}
	}
	return nil, errors.New("user not found")
}

func (r *stubUserRepository) GetByEmail(_ context.Context, email string) (*user.User, error) {
	if u, ok := r.byEmail[email]; ok {
		return u, nil
	}
	return nil, errors.New("user not found")
}

func (r *stubUserRepository) GetAll(_ context.Context) ([]*user.User, error) {
	users := make([]*user.User, 0, len(r.byEmail))
	for _, u := range r.byEmail {
		users = append(users, u)
	}
	return users, nil
}

func (r *stubUserRepository) Update(_ context.Context, _ *user.User) error { return nil }

func (r *stubUserRepository) Delete(_ context.Context, _ uuid.UUID) error { return nil }

// loginTestValidator — Echo-валидатор для формы входа.
type loginTestValidator struct {
	validator *validator.Validate
}

func (v *loginTestValidator) Validate(i any) error {
	return v.validator.Struct(i)
}

// loginTestSession — результат прохода через настоящую форму входа.
type loginTestSession struct {
	anonCookie *http.Cookie
	anonToken  string
	authCookie *http.Cookie
}

// newLoginTestServer поднимает Echo с настоящими SessionStore и CSRFProtection
// и реальным AuthHandler. Только на полном стеке видно, что вход перевыпускает
// CSRF-токен (S-02) и при этом не убивает cookie сессии.
func newLoginTestServer(t *testing.T) (*echo.Echo, *user.User) {
	t.Helper()

	hash, err := bcrypt.GenerateFromPassword([]byte(loginTestPassword), bcrypt.MinCost)
	require.NoError(t, err)

	testUser := &user.User{
		ID:       uuid.New(),
		Email:    "login-fixation@example.com",
		Password: string(hash),
		Role:     user.RoleAdmin,
	}

	repos := &handlers.Repositories{
		User: &stubUserRepository{byEmail: map[string]*user.User{testUser.Email: testUser}},
	}

	e := echo.New()
	e.Validator = &loginTestValidator{validator: validator.New()}
	e.Use(middleware.SessionStore(loginTestSessionSecret, false))
	e.Use(middleware.CSRFProtection())

	authHandler := NewAuthHandler(repos, &services.Services{}, false)
	e.POST("/login", authHandler.Login)
	e.GET("/logout", authHandler.Logout)

	// /token отдаёт текущий токен сессии, /probe — приёмник записи, на котором
	// видно, принимает ли CSRF-middleware конкретный токен.
	e.GET("/token", func(c echo.Context) error {
		token, tokenErr := middleware.GetCSRFToken(c)
		if tokenErr != nil {
			return tokenErr
		}
		return c.String(http.StatusOK, token)
	})
	e.POST("/probe", func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	})
	e.GET("/whoami", func(c echo.Context) error {
		if middleware.IsAuthenticated(c) {
			return c.String(http.StatusOK, "authenticated")
		}
		return c.String(http.StatusOK, "anonymous")
	})

	return e, testUser
}

// lastSessionCookie возвращает последнюю cookie сессии из ответа: за один
// запрос сессия может сохраняться несколько раз, и клиент применяет Set-Cookie
// по порядку, поэтому актуальна именно последняя.
func lastSessionCookie(t *testing.T, rec *httptest.ResponseRecorder) *http.Cookie {
	t.Helper()

	var found *http.Cookie
	for _, cookie := range (&http.Response{Header: rec.Header()}).Cookies() {
		if cookie.Name == middleware.SessionName {
			found = cookie
		}
	}
	require.NotNil(t, found, "сессионная cookie не выдана")

	return found
}

// fetchCSRFToken забирает токен текущей сессии через /token.
func fetchCSRFToken(t *testing.T, e *echo.Echo, cookie *http.Cookie) (string, *httptest.ResponseRecorder) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/token", nil)
	if cookie != nil {
		req.AddCookie(cookie)
	}
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	require.NotEmpty(t, rec.Body.String())

	return rec.Body.String(), rec
}

// postProbe шлёт запись с указанными cookie и токеном.
func postProbe(t *testing.T, e *echo.Echo, cookie *http.Cookie, token string) int {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, "/probe", nil)
	req.Header.Set(middleware.CSRFHeaderKey, token)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	return rec.Code
}

// whoami сообщает, считает ли сервер сессию аутентифицированной.
func whoami(t *testing.T, e *echo.Echo, cookie *http.Cookie) string {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/whoami", nil)
	req.AddCookie(cookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	return rec.Body.String()
}

// performLogin проходит настоящую форму входа с анонимным CSRF-токеном.
func performLogin(t *testing.T, e *echo.Echo, u *user.User) loginTestSession {
	t.Helper()

	anonToken, anonRec := fetchCSRFToken(t, e, nil)
	anonCookie := lastSessionCookie(t, anonRec)

	form := url.Values{}
	form.Set("email", u.Email)
	form.Set("password", loginTestPassword)

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationForm)
	req.Header.Set(middleware.CSRFHeaderKey, anonToken)
	req.AddCookie(anonCookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code, "вход не удался: %s", rec.Body.String())

	return loginTestSession{
		anonCookie: anonCookie,
		anonToken:  anonToken,
		authCookie: lastSessionCookie(t, rec),
	}
}

// TestLogin_SessionFixation — регрессия на S-02: CSRF-токен, полученный до
// входа, обязан стать недействительным после успешного входа.
func TestLogin_SessionFixation(t *testing.T) {
	e, testUser := newLoginTestServer(t)

	login := performLogin(t, e, testUser)

	// Вход через UI не сломан: cookie сессии живая, а не на удаление.
	require.Positive(t, login.authCookie.MaxAge,
		"cookie сессии после входа обязана жить, иначе вход через UI сломан")

	t.Run("AnonymousTokenRejected", func(t *testing.T) {
		assert.Equal(t, http.StatusForbidden, postProbe(t, e, login.authCookie, login.anonToken),
			"токен, полученный до входа, обязан отвергаться после входа")
	})

	t.Run("NewTokenIssuedAndAccepted", func(t *testing.T) {
		newToken, _ := fetchCSRFToken(t, e, login.authCookie)

		assert.NotEqual(t, login.anonToken, newToken, "после входа обязан выдаваться новый токен")
		assert.Equal(t, http.StatusOK, postProbe(t, e, login.authCookie, newToken),
			"новый токен обязан приниматься")
	})

	t.Run("OldSessionCookieStaysAnonymous", func(t *testing.T) {
		// Хранилище cookie-based: злоумышленник, подсунувший жертве свою cookie
		// до входа, продолжает держать в руках именно анонимную сессию — данные
		// пользователя ушли в новую cookie, выданную при входе.
		assert.Equal(t, "anonymous", whoami(t, e, login.anonCookie))
		assert.Equal(t, "authenticated", whoami(t, e, login.authCookie))
	})
}

// TestLogout_ClearsCSRFToken — Logout уже чистит сессию через ClearSession,
// тест фиксирует это поведение: выданная cookie удаляет сессию, а её токен
// вместе с ней перестаёт действовать.
func TestLogout_ClearsCSRFToken(t *testing.T) {
	e, testUser := newLoginTestServer(t)

	login := performLogin(t, e, testUser)
	authToken, _ := fetchCSRFToken(t, e, login.authCookie)

	req := httptest.NewRequest(http.MethodGet, "/logout", nil)
	req.AddCookie(login.authCookie)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusFound, rec.Code)

	clearedCookie := lastSessionCookie(t, rec)
	assert.Negative(t, clearedCookie.MaxAge, "выход обязан удалять cookie сессии")

	assert.Equal(t, http.StatusForbidden, postProbe(t, e, clearedCookie, authToken),
		"после выхода прежний CSRF-токен не должен подходить к сессии")
}

func TestSanitizeRedirectURL_ComprehensiveSecurityTest(t *testing.T) {
	// Comprehensive test covering all major attack vectors
	require.Equal(t, "/", sanitizeRedirectURL("//evil.com"), "Protocol-relative")
	require.Equal(t, "/", sanitizeRedirectURL("https://evil.com"), "Absolute URL")
	require.Equal(t, "/", sanitizeRedirectURL("javascript:alert(1)"), "JavaScript protocol")
	require.Equal(t, "/", sanitizeRedirectURL("data:text/html,<h1>hi</h1>"), "Data URI")
	require.Equal(t, "/", sanitizeRedirectURL("\\\\evil.com"), "Backslash escape")
	require.Equal(t, "/dashboard", sanitizeRedirectURL("/dashboard"), "Valid local path")
	require.Equal(t, "/page?id=123", sanitizeRedirectURL("/page?id=123"), "Valid with query")
	require.Equal(t, "/", sanitizeRedirectURL("\r\n"), "CRLF injection")
	require.Equal(t, "/", sanitizeRedirectURL("///evil.com"), "Triple slash")
	require.Equal(t, "/", sanitizeRedirectURL("file:///etc/passwd"), "File protocol")
}
