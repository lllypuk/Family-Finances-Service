package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/category"
	"family-budget-service/internal/domain/transaction"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/recognize"
	"family-budget-service/internal/testhelpers"
)

const recognizePath = "/api/v1/transactions/recognize"

// stubRecognizer — подменный движок: отдаёт заданный результат или ошибку.
type stubRecognizer struct {
	result recognize.Result
	err    error
	calls  int
}

func (s *stubRecognizer) Recognize(_ context.Context, _ recognize.Input) (recognize.Result, error) {
	s.calls++
	return s.result, s.err
}

func (s *stubRecognizer) Budget() time.Duration { return time.Second }

func postRecognize(
	t *testing.T,
	ts *testhelpers.TestServer,
	sess *testhelpers.AuthSession,
	body io.Reader,
	contentType string,
) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, recognizePath, body)
	req.Header.Set(echo.HeaderContentType, contentType)
	if sess != nil {
		sess.Apply(req)
	}

	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)

	return rec
}

func onePNG(t *testing.T) ([]byte, string) {
	t.Helper()
	return testhelpers.MultipartBody(t, testhelpers.FilePart{
		Field: "images", Name: "shot.png", Data: testhelpers.PNGImage(t, 8, 8),
	})
}

func TestRecognizeAPI_SuccessWithSimilar(t *testing.T) {
	engine := &stubRecognizer{}
	ts := testhelpers.SetupHTTPServer(t, testhelpers.WithRecognizer(engine))
	sess := ts.Auth(t)

	cat := testhelpers.CreateTestCategory(ts.AuthFamily.ID, category.TypeExpense)
	require.NoError(t, ts.Repos.Category.Create(t.Context(), cat))
	existing := testhelpers.CreateTestTransaction(ts.AuthFamily.ID, ts.AuthUser.ID, cat.ID, transaction.TypeExpense)
	require.NoError(t, ts.Repos.Transaction.Create(t.Context(), existing))

	source := 0
	engine.result = recognize.Result{
		Items: []recognize.Item{{
			Source:      &source,
			AmountMinor: existing.AmountMinor,
			Type:        transaction.TypeExpense,
			Date:        &existing.Date,
			Description: "Шавуха",
		}},
		Model: "stub",
	}

	body, contentType := onePNG(t)
	rec := postRecognize(t, ts, sess, bytes.NewReader(body), contentType)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	var resp handlers.APIResponse[recognize.Result]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Data.Items, 1)
	require.Len(t, resp.Data.Items[0].Similar, 1)
	assert.Equal(t, existing.ID, resp.Data.Items[0].Similar[0].ID)
	assert.Equal(t, existing.Description, resp.Data.Items[0].Similar[0].Description)
	assert.Equal(t, "stub", resp.Data.Model)
}

func TestRecognizeAPI_Refusals(t *testing.T) {
	tests := []struct {
		name       string
		engine     *stubRecognizer
		status     int
		code       string
		retryAfter string
	}{
		{name: "engine disabled", status: http.StatusServiceUnavailable, code: handlers.ErrCodeRecognitionUnavailable},
		{
			name:       "engine unavailable with Retry-After",
			engine:     &stubRecognizer{err: &recognize.UnavailableError{RetryAfter: 7 * time.Second}},
			status:     http.StatusServiceUnavailable,
			code:       handlers.ErrCodeRecognitionUnavailable,
			retryAfter: "7",
		},
		{
			name:   "bad model answer",
			engine: &stubRecognizer{err: recognize.ErrBadAnswer},
			status: http.StatusBadGateway,
			code:   handlers.ErrCodeRecognitionFailed,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []testhelpers.ServerOption
			if tt.engine != nil {
				opts = append(opts, testhelpers.WithRecognizer(tt.engine))
			}
			ts := testhelpers.SetupHTTPServer(t, opts...)

			body, contentType := onePNG(t)
			rec := postRecognize(t, ts, ts.Auth(t), bytes.NewReader(body), contentType)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, tt.code, errorCode(t, rec))
			assert.Equal(t, tt.retryAfter, rec.Header().Get(echo.HeaderRetryAfter))
		})
	}
}

func TestRecognizeAPI_PayloadTooLarge(t *testing.T) {
	engine := &stubRecognizer{}
	ts := testhelpers.SetupHTTPServer(t, testhelpers.WithRecognizer(engine))
	sess := ts.Auth(t)

	parts := make([]testhelpers.FilePart, 0, recognize.MaxImages+1)
	for range recognize.MaxImages + 1 {
		parts = append(parts, testhelpers.FilePart{Field: "images", Name: "big.png", Data: make([]byte, 2_000_000)})
	}
	body, contentType := testhelpers.MultipartBody(t, parts...)

	t.Run("by Content-Length", func(t *testing.T) {
		rec := postRecognize(t, ts, sess, bytes.NewReader(body), contentType)
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
		assert.Equal(t, handlers.ErrCodePayloadTooLarge, errorCode(t, rec))
	})

	t.Run("while streaming", func(t *testing.T) {
		// Без Content-Length лимитер срабатывает на чтении: преамбулу больше лимита
		// multipart-парсер пролистывает сам, и ошибка лимитера приходит из NextPart.
		png, pngType := onePNG(t)
		preamble := bytes.Repeat([]byte("junk\r\n"), 2_000_000)
		stream := io.MultiReader(bytes.NewReader(preamble), bytes.NewReader(png))
		rec := postRecognize(t, ts, sess, stream, pngType)
		require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
		assert.Equal(t, handlers.ErrCodePayloadTooLarge, errorCode(t, rec))
	})

	assert.Zero(t, engine.calls)
}

func TestRecognizeAPI_NotAnImage(t *testing.T) {
	engine := &stubRecognizer{}
	ts := testhelpers.SetupHTTPServer(t, testhelpers.WithRecognizer(engine))

	body, contentType := testhelpers.MultipartBody(t, testhelpers.FilePart{
		Field: "images", Name: "note.txt", Data: []byte("not an image"),
	})
	rec := postRecognize(t, ts, ts.Auth(t), bytes.NewReader(body), contentType)
	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

	var resp handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	require.Len(t, resp.Error.Details, 1)
	assert.Equal(t, "images[0]", resp.Error.Details[0].Field)
	assert.Zero(t, engine.calls)
}

func TestRecognizeAPI_Access(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t, testhelpers.WithRecognizer(&stubRecognizer{}))
	body, contentType := onePNG(t)

	_, member := ts.AuthAs(t, user.RoleMember)
	rec := postRecognize(t, ts, member, bytes.NewReader(body), contentType)
	assert.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	rec = postRecognize(t, ts, nil, bytes.NewReader(body), contentType)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestRecognizeAPI_Metrics(t *testing.T) {
	engine := &stubRecognizer{result: recognize.Result{Items: []recognize.Item{{
		AmountMinor: 30000, Type: transaction.TypeExpense, Description: "Шавуха",
	}}}}
	ts := testhelpers.SetupHTTPServer(t, testhelpers.WithRecognizer(engine))
	sess := ts.Auth(t)

	body, contentType := onePNG(t)
	require.Equal(t, http.StatusOK, postRecognize(t, ts, sess, bytes.NewReader(body), contentType).Code)

	engine.result = recognize.Result{}
	require.Equal(t, http.StatusOK, postRecognize(t, ts, sess, bytes.NewReader(body), contentType).Code)

	engine.err = recognize.ErrBadAnswer
	require.Equal(t, http.StatusBadGateway, postRecognize(t, ts, sess, bytes.NewReader(body), contentType).Code)

	engine.err = &recognize.UnavailableError{RetryAfter: time.Second}
	require.Equal(t, http.StatusServiceUnavailable,
		postRecognize(t, ts, sess, bytes.NewReader(body), contentType).Code)

	metrics := scrape(t, ts)
	for _, outcome := range []string{"ok", "empty", "failed", "unavailable"} {
		assert.Contains(t, metrics, `ffs_recognitions_total{outcome="`+outcome+`"} 1`)
	}
	assert.Contains(t, metrics, "ffs_recognition_duration_seconds_count 4")
}
