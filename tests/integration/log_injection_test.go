package integration_test

import (
	"bytes"
	"encoding/json"
	"errors"
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
// в настоящий перевод строки. Подделать запись в журнале это не даёт: логгер
// echo сериализует сообщение в JSON и сам экранирует перевод строки, так что
// запись остаётся одной физической строкой. Но полагаться на энкодер логгера
// не стоит — при переходе на текстовый вывод дыра открылась бы. Поэтому
// обработчики форматируют путь и текст ошибки через %q.
//
// Тест проверяет именно это: в самом сообщении (после разбора JSON) путь
// закавычен, а перевод строки представлен escape-последовательностью.
func TestLogging_RequestPathIsEscapedInLogMessage(t *testing.T) {
	const injected = "2026-01-01 INFO admin login succeeded"

	testServer := testhelpers.SetupHTTPServer(t)
	// Без семьи RequireSetup редиректит любой путь на /setup, и до обработчика
	// ошибок запрос не доходит.
	auth := testServer.Auth(t)
	e := testServer.Server.Echo()

	// Маршрут, падающий необработанной (не echo.HTTPError) ошибкой — именно
	// эта ветка customHTTPErrorHandler пишет путь запроса в лог.
	e.GET("/log-injection-probe/*", func(_ echo.Context) error {
		return errors.New("boom")
	})

	var logs bytes.Buffer
	e.Logger.SetOutput(&logs)

	target := "/log-injection-probe/x%0a" + strings.ReplaceAll(injected, " ", "%20")

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, target, nil)
	auth.Apply(req)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusInternalServerError, rec.Code,
		"проба должна доходить до обработчика необработанных ошибок")

	var entry struct {
		Message string `json:"message"`
	}
	require.NoError(t, json.Unmarshal([]byte(strings.TrimSpace(logs.String())), &entry),
		"лог echo ожидается в формате JSON, получено: %s", logs.String())
	require.Contains(t, entry.Message, "unhandled error",
		"обработчик ошибок не попал в лог, тест ничего не проверяет")

	assert.NotContains(t, entry.Message, "\n",
		"путь из запроса принёс в сообщение сырой перевод строки — при текстовом логгере это позволило бы "+
			"подделать запись; форматируйте путь через %%q:\n%s", entry.Message)
	assert.Contains(t, entry.Message, `"/log-injection-probe/x\n`,
		"путь должен попадать в сообщение закавыченным и с экранированным переводом строки:\n%s", entry.Message)
}
