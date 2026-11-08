package handlers_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/recognize"
	"family-budget-service/internal/testhelpers"
)

const recognizePath = "/api/v1/transactions/recognize"

type fakeRecognizeService struct {
	result recognize.Result
	err    error
	images []recognize.Image
	calls  int
}

func (f *fakeRecognizeService) Recognize(_ context.Context, images []recognize.Image) (recognize.Result, error) {
	f.calls++
	f.images = images

	return f.result, f.err
}

func (f *fakeRecognizeService) Budget() time.Duration { return time.Minute }

func recognizeRequest(
	t *testing.T,
	svc *fakeRecognizeService,
	uploadTimeout time.Duration,
	parts ...testhelpers.FilePart,
) (*httptest.ResponseRecorder, error) {
	t.Helper()

	body, contentType := testhelpers.MultipartBody(t, parts...)
	principal := &auth.Principal{SessionID: uuid.New(), UserID: uuid.New(), Role: user.RoleMember}
	req, rec := principalRequest(http.MethodPost, recognizePath, string(body), principal)
	req.Header.Set("Content-Type", contentType)

	return rec, handlers.NewRecognizeHandler(svc, uploadTimeout).Recognize(rec, req)
}

func imagePart(data []byte) testhelpers.FilePart {
	return testhelpers.FilePart{Field: "images", Name: "shot.png", Data: data}
}

func assertValidationField(t *testing.T, rec *httptest.ResponseRecorder, field, message string) {
	t.Helper()

	require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())
	body := decodeError(t, rec)
	assert.Equal(t, handlers.ErrCodeValidationError, body.Error.Code)
	require.Len(t, body.Error.Details, 1)
	assert.Equal(t, field, body.Error.Details[0].Field)
	assert.Contains(t, body.Error.Details[0].Message, message)
}

func TestRecognizeHandler_Recognize_Success(t *testing.T) {
	source := 1
	svc := &fakeRecognizeService{result: recognize.Result{
		Items: []recognize.Item{{Source: &source, AmountMinor: 30000, Type: "expense", Description: "Кафе"}},
		Model: "gemma",
	}}

	rec, err := recognizeRequest(t, svc, handlers.RecognizeUploadTimeout,
		imagePart(testhelpers.PNGImage(t, 20, 40)),
		imagePart(testhelpers.JPEGImage(t, 10, 10)))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

	require.Len(t, svc.images, 2)
	assert.Equal(t, recognize.MIMEPNG, svc.images[0].MIME)
	assert.Equal(t, recognize.MIMEJPEG, svc.images[1].MIME)

	var body handlers.APIResponse[recognize.Result]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	require.Len(t, body.Data.Items, 1)
	assert.Equal(t, 1, *body.Data.Items[0].Source)
	assert.Equal(t, "gemma", body.Data.Model)
}

func TestRecognizeHandler_Recognize_EmptyItemsIsArray(t *testing.T) {
	rec, err := recognizeRequest(t, &fakeRecognizeService{}, handlers.RecognizeUploadTimeout,
		imagePart(testhelpers.PNGImage(t, 5, 5)))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"items":[]`)
}

func TestRecognizeHandler_Recognize_PartRefusals(t *testing.T) {
	png := testhelpers.PNGImage(t, 5, 5)
	six := make([]testhelpers.FilePart, 0, recognize.MaxImages+1)
	for range recognize.MaxImages + 1 {
		six = append(six, imagePart(png))
	}

	tests := []struct {
		name    string
		parts   []testhelpers.FilePart
		field   string
		message string
	}{
		{name: "no parts", parts: nil, field: "images", message: "is required"},
		{
			name:    "not an image",
			parts:   []testhelpers.FilePart{imagePart(png), imagePart([]byte("GIF89a"))},
			field:   "images[1]",
			message: "PNG or JPEG",
		},
		{
			name:    "too large",
			parts:   []testhelpers.FilePart{imagePart(make([]byte, recognize.MaxImageBytes+1))},
			field:   "images[0]",
			message: "larger than",
		},
		{name: "sixth part", parts: six, field: "images[5]", message: "at most 5"},
		{
			name:    "foreign field",
			parts:   []testhelpers.FilePart{{Field: "note", Name: "a.png", Data: png}},
			field:   "note",
			message: "unexpected part",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := &fakeRecognizeService{}
			rec, err := recognizeRequest(t, svc, handlers.RecognizeUploadTimeout, tt.parts...)
			require.NoError(t, err)
			assertValidationField(t, rec, tt.field, tt.message)
			assert.Zero(t, svc.calls)
		})
	}
}

func TestRecognizeHandler_Recognize_NotMultipart(t *testing.T) {
	principal := &auth.Principal{SessionID: uuid.New(), UserID: uuid.New(), Role: user.RoleAdmin}
	req, rec := principalRequest(http.MethodPost, recognizePath, `{"images":[]}`, principal)

	require.NoError(t, handlers.NewRecognizeHandler(&fakeRecognizeService{}, time.Minute).Recognize(rec, req))
	require.Equal(t, http.StatusBadRequest, rec.Code)
	assert.Equal(t, handlers.ErrCodeInvalidRequest, decodeError(t, rec).Error.Code)
}

// recognizeTruncatedTail обрезает тело внутри данных картинки: обрыв в заголовках части
// multipart.Reader отдаёт как io.EOF, и ответом был бы 422 «нет картинок».
const recognizeTruncatedTail = 80

func TestRecognizeHandler_Recognize_TruncatedBody(t *testing.T) {
	body, contentType := testhelpers.MultipartBody(t, imagePart(testhelpers.PNGImage(t, 5, 5)))
	principal := &auth.Principal{SessionID: uuid.New(), UserID: uuid.New(), Role: user.RoleMember}
	req, rec := principalRequest(
		http.MethodPost,
		recognizePath,
		string(body[:len(body)-recognizeTruncatedTail]),
		principal,
	)
	req.Header.Set("Content-Type", contentType)
	svc := &fakeRecognizeService{}

	require.NoError(t, handlers.NewRecognizeHandler(svc, time.Minute).Recognize(rec, req))
	require.Equal(t, http.StatusBadRequest, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodeInvalidRequest, decodeError(t, rec).Error.Code)
	assert.Zero(t, svc.calls)
}

func TestRecognizeHandler_Recognize_UploadTimeout(t *testing.T) {
	svc := &fakeRecognizeService{}
	rec, err := recognizeRequest(t, svc, time.Nanosecond, imagePart(testhelpers.PNGImage(t, 5, 5)))
	require.NoError(t, err)
	require.Equal(t, http.StatusRequestTimeout, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodeRequestTimeout, decodeError(t, rec).Error.Code)
	assert.Zero(t, svc.calls)
}

func TestRecognizeHandler_Recognize_ServiceErrors(t *testing.T) {
	tests := []struct {
		name       string
		err        error
		status     int
		code       string
		retryAfter string
	}{
		{name: "disabled", err: recognize.ErrDisabled,
			status: http.StatusServiceUnavailable, code: handlers.ErrCodeRecognitionUnavailable},
		{name: "unavailable without retry", err: &recognize.UnavailableError{Err: errors.New("dial")},
			status: http.StatusServiceUnavailable, code: handlers.ErrCodeRecognitionUnavailable},
		{name: "unavailable with retry", err: &recognize.UnavailableError{RetryAfter: 1500 * time.Millisecond},
			status: http.StatusServiceUnavailable, code: handlers.ErrCodeRecognitionUnavailable, retryAfter: "2"},
		{name: "bad answer", err: recognize.ErrBadAnswer,
			status: http.StatusBadGateway, code: handlers.ErrCodeRecognitionFailed},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec, err := recognizeRequest(t, &fakeRecognizeService{err: tt.err}, handlers.RecognizeUploadTimeout,
				imagePart(testhelpers.PNGImage(t, 5, 5)))
			require.NoError(t, err)
			require.Equal(t, tt.status, rec.Code, rec.Body.String())
			assert.Equal(t, tt.code, decodeError(t, rec).Error.Code)
			assert.Equal(t, tt.retryAfter, rec.Header().Get("Retry-After"))
		})
	}
}

func TestRecognizeHandler_Recognize_UnknownErrorGoesToErrorHandler(t *testing.T) {
	cause := errors.New("database is locked")
	rec, err := recognizeRequest(t, &fakeRecognizeService{err: cause}, handlers.RecognizeUploadTimeout,
		imagePart(testhelpers.PNGImage(t, 5, 5)))
	require.ErrorIs(t, err, cause)
	assert.Empty(t, rec.Body.String())
}

func TestRecognizeHandler_Recognize_TooLarge(t *testing.T) {
	svc := &fakeRecognizeService{}
	principal := &auth.Principal{SessionID: uuid.New(), UserID: uuid.New(), Role: user.RoleMember}
	multipartBody, contentType := testhelpers.MultipartBody(t, imagePart(testhelpers.PNGImage(t, 5, 5)))
	cut := io.MultiReader(
		strings.NewReader(string(multipartBody[:len(multipartBody)-recognizeTruncatedTail])),
		errReader{err: &http.MaxBytesError{Limit: handlers.RecognizeBodyLimit}},
	)
	req := httptest.NewRequest(http.MethodPost, recognizePath, cut)
	req.ContentLength = -1
	req.Header.Set("Content-Type", contentType)
	req = req.WithContext(auth.WithPrincipal(req.Context(), principal))
	rec := httptest.NewRecorder()

	require.NoError(t, handlers.NewRecognizeHandler(svc, time.Minute).Recognize(rec, req))
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Equal(t, handlers.ErrCodePayloadTooLarge, decodeError(t, rec).Error.Code)

	early := httptest.NewRequest(http.MethodPost, recognizePath, strings.NewReader("x"))
	early.ContentLength = handlers.RecognizeBodyLimit + 1
	rec = httptest.NewRecorder()

	require.NoError(t, handlers.NewRecognizeHandler(svc, time.Minute).Recognize(rec, early))
	require.Equal(t, http.StatusRequestEntityTooLarge, rec.Code)
	assert.Zero(t, svc.calls)
}

func TestRecognizeHandler_Recognize_BodyLimitBoundary(t *testing.T) {
	assert.Equal(t, 11_000_000, handlers.RecognizeBodyLimit)

	svc := &fakeRecognizeService{}
	h := handlers.NewRecognizeHandler(svc, time.Minute)

	atLimit := httptest.NewRequest(http.MethodPost, recognizePath, strings.NewReader("x"))
	atLimit.ContentLength = handlers.RecognizeBodyLimit
	rec := httptest.NewRecorder()
	require.NoError(t, h.Recognize(rec, atLimit))
	assert.NotEqual(t, http.StatusRequestEntityTooLarge, rec.Code)

	over := httptest.NewRequest(http.MethodPost, recognizePath, strings.NewReader("x"))
	over.ContentLength = handlers.RecognizeBodyLimit + 1
	rec = httptest.NewRecorder()
	require.NoError(t, h.Recognize(rec, over))
	assert.Equal(t, http.StatusRequestEntityTooLarge, rec.Code, rec.Body.String())
	assert.Zero(t, svc.calls)
}

type errReader struct{ err error }

func (e errReader) Read([]byte) (int, error) { return 0, e.err }
