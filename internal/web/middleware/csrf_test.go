package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/web/middleware"
)

func setupCSRFTest() (*echo.Echo, echo.MiddlewareFunc) {
	e := echo.New()

	// Настраиваем session store для тестов
	sessionMiddleware := middleware.SessionStore("test-secret-key-for-csrf-tests", false)
	csrfMiddleware := middleware.CSRFProtection()

	// Создаем комбинированный middleware с обоими middleware
	combinedMiddleware := func(next echo.HandlerFunc) echo.HandlerFunc {
		// Применяем session middleware, затем CSRF middleware
		return sessionMiddleware(csrfMiddleware(next))
	}

	return e, combinedMiddleware
}

func TestCSRFProtection_GET_Request_GeneratesToken(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	// Проверяем что токен был сгенерирован
	token, err := middleware.GetCSRFToken(c)
	require.NoError(t, err)
	assert.NotEmpty(t, token)
}

func TestCSRFProtection_POST_WithValidToken_Success(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	// Сначала делаем GET запрос для генерации токена
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	getCtx := e.NewContext(getReq, getRec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(getCtx)
	require.NoError(t, err)

	// Получаем токен
	token, err := middleware.GetCSRFToken(getCtx)
	require.NoError(t, err)
	require.NotEmpty(t, token)

	// Теперь делаем POST запрос с токеном
	form := url.Values{}
	form.Add("_token", token)

	postReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Копируем cookies из GET запроса в POST
	for _, cookie := range getRec.Result().Cookies() {
		postReq.AddCookie(cookie)
	}

	postRec := httptest.NewRecorder()
	postCtx := e.NewContext(postReq, postRec)

	err = handler(postCtx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, postRec.Code)
}

func TestCSRFProtection_POST_WithValidToken_InHeader_Success(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	// Генерируем токен
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	getCtx := e.NewContext(getReq, getRec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(getCtx)
	require.NoError(t, err)

	token, err := middleware.GetCSRFToken(getCtx)
	require.NoError(t, err)

	// POST запрос с токеном в header (HTMX стиль)
	postReq := httptest.NewRequest(http.MethodPost, "/", nil)
	postReq.Header.Set("X-Csrf-Token", token)
	postReq.Header.Set("Hx-Request", "true")

	// Копируем cookies
	for _, cookie := range getRec.Result().Cookies() {
		postReq.AddCookie(cookie)
	}

	postRec := httptest.NewRecorder()
	postCtx := e.NewContext(postReq, postRec)

	err = handler(postCtx)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, postRec.Code)
}

func TestCSRFProtection_POST_WithoutToken_Forbidden(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)

	if err != nil {
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
		assert.Contains(t, httpErr.Message, "CSRF token not provided")
	} else {
		assert.Equal(t, http.StatusForbidden, rec.Code)
	}
}

func TestCSRFProtection_POST_WithInvalidToken_Forbidden(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	// Генерируем валидный токен
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	getCtx := e.NewContext(getReq, getRec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(getCtx)
	require.NoError(t, err)

	// POST запрос с неправильным токеном
	form := url.Values{}
	form.Add("_token", "invalid-token")

	postReq := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
	postReq.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Копируем cookies
	for _, cookie := range getRec.Result().Cookies() {
		postReq.AddCookie(cookie)
	}

	postRec := httptest.NewRecorder()
	postCtx := e.NewContext(postReq, postRec)

	err = handler(postCtx)
	if err != nil {
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
		assert.Contains(t, httpErr.Message, "CSRF token mismatch")
	} else {
		assert.Equal(t, http.StatusForbidden, postRec.Code)
	}
}

func TestCSRFProtection_POST_HTMX_Request_ReturnsJSON(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodPost, "/", nil)
	req.Header.Set("Hx-Request", "true")
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)

	require.NoError(t, err) // HTMX ошибки возвращаются как JSON response, не как error
	assert.Equal(t, http.StatusForbidden, rec.Code)
	assert.Contains(t, rec.Body.String(), "CSRF token validation failed")
	assert.Contains(t, rec.Header().Get("Content-Type"), "application/json")
}

func TestCSRFProtection_PUT_Request_RequiresToken(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodPut, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)

	if err != nil {
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
	} else {
		assert.Equal(t, http.StatusForbidden, rec.Code)
	}
}

func TestCSRFProtection_DELETE_Request_RequiresToken(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodDelete, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "should not reach")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)

	if err != nil {
		var httpErr *echo.HTTPError
		require.ErrorAs(t, err, &httpErr)
		assert.Equal(t, http.StatusForbidden, httpErr.Code)
	} else {
		assert.Equal(t, http.StatusForbidden, rec.Code)
	}
}

func TestCSRFProtection_HEAD_Request_Allowed(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodHead, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestCSRFProtection_OPTIONS_Request_Allowed(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodOptions, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.NoContent(http.StatusOK)
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)

	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestGetCSRFToken_GeneratesNewToken_WhenNoneExists(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	// Сначала инициализируем сессию через middleware
	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)
	require.NoError(t, err)

	token, err := middleware.GetCSRFToken(c)

	require.NoError(t, err)
	assert.NotEmpty(t, token)
	assert.Greater(t, len(token), 30) // Base64 encoded token should be sufficiently long
}

func TestGetCSRFToken_ReturnsExistingToken(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)
	err := handler(c)
	require.NoError(t, err)

	// Получаем токен первый раз
	token1, err := middleware.GetCSRFToken(c)
	require.NoError(t, err)

	// Получаем токен второй раз - должен быть тот же
	token2, err := middleware.GetCSRFToken(c)
	require.NoError(t, err)

	assert.Equal(t, token1, token2)
}

func TestCSRFToken_Security_Properties(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	// Генерируем несколько токенов и проверяем их уникальность
	tokens := make(map[string]bool)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	for range 10 { // Уменьшим количество для быстроты
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		// Инициализируем сессию
		handler := csrfMiddleware(nextHandler)
		err := handler(c)
		require.NoError(t, err)

		token, err := middleware.GetCSRFToken(c)
		require.NoError(t, err)
		require.NotEmpty(t, token)

		// Проверяем уникальность
		assert.False(t, tokens[token], "Token should be unique")
		tokens[token] = true

		// Проверяем длину и формат
		assert.Greater(t, len(token), 30, "Token should be sufficiently long")
		assert.NotContains(t, token, " ", "Token should not contain spaces")
	}
}

// TestRegenerateCSRFToken_IssuesDifferentToken — S-02: перевыпуск обязан выдать
// другой токен и положить в сессию именно его.
func TestRegenerateCSRFToken_IssuesDifferentToken(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	var oldToken, newToken string
	handler := csrfMiddleware(func(c echo.Context) error {
		var err error
		oldToken, err = middleware.GetCSRFToken(c)
		require.NoError(t, err)

		newToken, err = middleware.RegenerateCSRFToken(c)
		require.NoError(t, err)

		return c.String(http.StatusOK, "ok")
	})

	require.NoError(t, handler(c))

	require.NotEmpty(t, oldToken)
	require.NotEmpty(t, newToken)
	assert.NotEqual(t, oldToken, newToken, "перевыпущенный токен обязан отличаться от прежнего")

	// В сессии лежит именно новый токен.
	current, err := middleware.GetCSRFToken(c)
	require.NoError(t, err)
	assert.Equal(t, newToken, current)
}

// TestRegenerateCSRFToken_OldTokenRejected — старый токен после перевыпуска
// перестаёт подходить к сессии, новый принимается.
func TestRegenerateCSRFToken_OldTokenRejected(t *testing.T) {
	e, csrfMiddleware := setupCSRFTest()

	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	getCtx := e.NewContext(getReq, getRec)

	var oldToken, newToken string
	genHandler := csrfMiddleware(func(c echo.Context) error {
		var err error
		oldToken, err = middleware.GetCSRFToken(c)
		require.NoError(t, err)

		newToken, err = middleware.RegenerateCSRFToken(c)
		require.NoError(t, err)

		return c.String(http.StatusOK, "ok")
	})
	require.NoError(t, genHandler(getCtx))

	// Перевыпуск добавляет второй Set-Cookie; клиент применяет их по порядку,
	// поэтому актуальна последняя cookie сессии.
	cookies := getRec.Result().Cookies()
	require.NotEmpty(t, cookies, "сессионная cookie не выдана")
	sessionCookie := cookies[len(cookies)-1]

	postWithToken := func(token string) int {
		postReq := httptest.NewRequest(http.MethodPost, "/", nil)
		postReq.Header.Set(middleware.CSRFHeaderKey, token)
		postReq.AddCookie(sessionCookie)

		postRec := httptest.NewRecorder()
		postCtx := e.NewContext(postReq, postRec)

		handler := csrfMiddleware(func(c echo.Context) error {
			return c.String(http.StatusOK, "ok")
		})
		require.NoError(t, handler(postCtx))

		return postRec.Code
	}

	assert.Equal(t, http.StatusForbidden, postWithToken(oldToken),
		"токен, выданный до перевыпуска, обязан отвергаться")
	assert.Equal(t, http.StatusOK, postWithToken(newToken),
		"перевыпущенный токен обязан приниматься")
}

// Benchmark тесты для производительности CSRF
func BenchmarkCSRFProtection_GET(b *testing.B) {
	e, csrfMiddleware := setupCSRFTest()

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler(c)
	}
}

func BenchmarkCSRFProtection_POST_WithToken(b *testing.B) {
	e, csrfMiddleware := setupCSRFTest()

	// Подготавливаем валидный токен
	getReq := httptest.NewRequest(http.MethodGet, "/", nil)
	getRec := httptest.NewRecorder()
	getCtx := e.NewContext(getReq, getRec)

	nextHandler := func(c echo.Context) error {
		return c.String(http.StatusOK, "ok")
	}

	handler := csrfMiddleware(nextHandler)
	handler(getCtx)

	token, _ := middleware.GetCSRFToken(getCtx)

	for b.Loop() {
		form := url.Values{}
		form.Add("_token", token)

		req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(form.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

		// Копируем cookies
		for _, cookie := range getRec.Result().Cookies() {
			req.AddCookie(cookie)
		}

		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		handler(c)
	}
}

func BenchmarkGetCSRFToken(b *testing.B) {
	e, _ := setupCSRFTest()

	for b.Loop() {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()
		c := e.NewContext(req, rec)

		middleware.GetCSRFToken(c)
	}
}

// Cookie, подписанная другим (провёрнутым) SESSION_SECRET, не расшифровывается.
// gorilla CookieStore.New возвращает в этом случае НЕнулевую ошибку вместе с
// пригодной новой сессией; раньше ensureCSRFToken пробрасывал её наружу, и
// браузер со старой cookie получал 500 на каждый GET, причём сама cookie не
// перезаписывалась — выйти из этого состояния можно было только руками.
func TestCSRFProtection_GET_WithUndecodableCookie_RecoversInsteadOf500(t *testing.T) {
	// Сессия, подписанная чужим секретом.
	foreignStore := sessions.NewCookieStore([]byte("some-completely-different-secret-value"))
	foreignReq := httptest.NewRequest(http.MethodGet, "/", nil)
	foreignSess, err := foreignStore.New(foreignReq, middleware.SessionName)
	require.NoError(t, err)
	foreignSess.Values["csrf_token"] = "stale-token"

	foreignRec := httptest.NewRecorder()
	require.NoError(t, foreignSess.Save(foreignReq, foreignRec))

	staleCookies := (&http.Response{Header: foreignRec.Header()}).Cookies()
	require.NotEmpty(t, staleCookies)

	e, csrfMiddleware := setupCSRFTest()

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(staleCookies[0])
	rec := httptest.NewRecorder()
	c := e.NewContext(req, rec)

	called := false
	handler := csrfMiddleware(func(ctx echo.Context) error {
		called = true
		return ctx.String(http.StatusOK, "ok")
	})

	require.NoError(t, handler(c))
	assert.True(t, called, "обработчик не вызван — нечитаемая cookie отдала ошибку вместо новой сессии")
	assert.Equal(t, http.StatusOK, rec.Code)

	token, tokenErr := middleware.GetCSRFToken(c)
	require.NoError(t, tokenErr)
	assert.NotEmpty(t, token)
	assert.NotEqual(t, "stale-token", token, "токен взят из нечитаемой сессии")

	assert.NotEmpty(t, (&http.Response{Header: rec.Header()}).Cookies(),
		"новая cookie не выдана, браузер продолжит слать нечитаемую")
}
