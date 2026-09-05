package integration_test

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/testhelpers"
)

// TestLogging_RequestPathIsEscapedInLogMessage закрывает находку CodeQL
// "Log entries created from user input".
//
// echo отдаёт уже декодированный r.URL.Path, поэтому %0a в запросе превращается
// в настоящий перевод строки. Обработчик ошибок пишет путь отдельным атрибутом slog,
// и текстовый handler экранирует его через strconv.Quote — запись остаётся одной
// физической строкой даже без JSON. Тест проверяет именно текстовый вывод.
func TestLogging_RequestPathIsEscapedInLogMessage(t *testing.T) {
	const injected = "2026-01-01 INFO admin login succeeded"

	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previous) })

	// Без observability сервер берёт slog.Default() при создании — после подмены.
	testServer := testhelpers.SetupHTTPServer(t)
	auth := testServer.Auth(t)
	e := testServer.Server.Echo()

	// Маршрут, падающий необработанной (не echo.HTTPError) ошибкой — именно
	// эта ветка обработчика ошибок пишет путь запроса в лог.
	e.GET("/log-injection-probe/*", func(_ echo.Context) error {
		return errors.New("boom")
	})

	target := "/log-injection-probe/x%0a" + strings.ReplaceAll(injected, " ", "%20")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	auth.Apply(req)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code,
		"проба должна доходить до обработчика необработанных ошибок")

	lines := strings.Split(strings.TrimSpace(logs.String()), "\n")
	var entry string
	for _, line := range lines {
		assert.True(t, strings.HasPrefix(line, "time="),
			"каждая физическая строка лога должна быть отдельной записью, получено:\n%s", logs.String())
		if strings.Contains(line, "msg=\"unhandled error\"") {
			entry = line
		}
	}
	require.NotEmpty(t, entry, "обработчик ошибок не попал в лог, тест ничего не проверяет:\n%s", logs.String())

	assert.Contains(t, entry, `path="/log-injection-probe/x\n2026-01-01 INFO admin login succeeded"`,
		"путь должен попадать в запись закавыченным и с экранированным переводом строки:\n%s", entry)
}
