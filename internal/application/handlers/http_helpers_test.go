package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/date"
)

type decodeTarget struct {
	Name string    `json:"name"`
	When date.Date `json:"when"`
}

func jsonRequest(body string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	return req
}

func TestDecodeJSON_NotJSONContentType_Rejected(t *testing.T) {
	for _, ct := range []string{"", "text/plain", "application/jsonp", "Application/JSON", "application/jsonx"} {
		req := jsonRequest(`{"name":"a"}`)
		req.Header.Del("Content-Type")
		if ct != "" {
			req.Header.Set("Content-Type", ct)
		}

		require.Error(t, handlers.DecodeJSON(req, &decodeTarget{}), ct)
	}
}

func TestDecodeJSON_ContentTypeParams_Accepted(t *testing.T) {
	for _, ct := range []string{"application/json; charset=utf-8", " application/json ;x=1"} {
		req := jsonRequest(`{"name":"a"}`)
		req.Header.Set("Content-Type", ct)

		require.NoError(t, handlers.DecodeJSON(req, &decodeTarget{}), ct)
	}
}

func TestDecodeJSON_EmptyBody_Skipped(t *testing.T) {
	var v decodeTarget

	require.NoError(t, handlers.DecodeJSON(jsonRequest(""), &v))
	assert.Empty(t, v.Name)
}

func TestDecodeJSON_UnknownFieldsAndTrailingGarbage_Allowed(t *testing.T) {
	var v decodeTarget

	require.NoError(t, handlers.DecodeJSON(jsonRequest(`{"name":"a","zzz":1}garbage`), &v))
	assert.Equal(t, "a", v.Name)
}

func TestDecodeJSON_UnknownLength_EmptyBodyFails(t *testing.T) {
	req := jsonRequest("")
	req.ContentLength = -1

	require.Error(t, handlers.DecodeJSON(req, &decodeTarget{}))
}

func TestDecodeJSON_UnknownLength_DecodesBody(t *testing.T) {
	req := jsonRequest(`{"name":"b"}`)
	req.ContentLength = -1
	var v decodeTarget

	require.NoError(t, handlers.DecodeJSON(req, &v))
	assert.Equal(t, "b", v.Name)
}

func TestDecodeJSON_Null_LeavesTargetZero(t *testing.T) {
	var v decodeTarget

	require.NoError(t, handlers.DecodeJSON(jsonRequest("null"), &v))
	assert.Empty(t, v.Name)
}

func TestRespondBindError_Cases(t *testing.T) {
	tests := []struct {
		name, body string
		status     int
		code       string
		field      string
	}{
		{"Garbage", "{{{", http.StatusBadRequest, handlers.ErrCodeInvalidRequest, "body"},
		{"BadDate", `{"when":"2026-13-45"}`, http.StatusUnprocessableEntity, handlers.ErrCodeValidationError, "when"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := handlers.DecodeJSON(jsonRequest(tt.body), &decodeTarget{})
			require.Error(t, err)
			rec := httptest.NewRecorder()

			require.NoError(t, handlers.RespondBindError(rec, err))

			assert.Equal(t, tt.status, rec.Code)
			var resp handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			assert.Equal(t, tt.code, resp.Error.Code)
			require.Len(t, resp.Error.Details, 1)
			assert.Equal(t, tt.field, resp.Error.Details[0].Field)
		})
	}
}

func TestRespondAPIHTTP_WrapsDataWithRequestID(t *testing.T) {
	rec := httptest.NewRecorder()
	rec.Header().Set("X-Request-ID", "rid-1")

	require.NoError(t, handlers.RespondAPI(rec, http.StatusCreated, map[string]string{"a": "b"}))

	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var resp handlers.APIResponse[map[string]string]
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, "b", resp.Data["a"])
	assert.Equal(t, "rid-1", resp.Meta.RequestID)
	assert.Equal(t, "v1", resp.Meta.Version)
}

func TestParsePaginationHTTP_Cases(t *testing.T) {
	tests := []struct {
		name, query string
		limit, off  int
		invalid     bool
	}{
		{"Defaults", "", 50, 0, false},
		{"Explicit", "limit=10&offset=5", 10, 5, false},
		{"LimitZero", "limit=0", 0, 0, true},
		{"LimitOverMax", "limit=201", 0, 0, true},
		{"OffsetNegative", "offset=-1", 0, 0, true},
		{"OffsetGarbage", "offset=x", 0, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/?"+tt.query, nil)

			page, err := handlers.ParsePagination(rec, req)

			if tt.invalid {
				require.ErrorIs(t, err, handlers.ErrResponseWritten())
				assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.limit, page.Limit)
			assert.Equal(t, tt.off, page.Offset)
		})
	}
}

func TestRespondListHTTP_NilItemsIsEmptyArray(t *testing.T) {
	rec := httptest.NewRecorder()

	require.NoError(t, handlers.RespondList(rec, nil, handlers.PageParams{Limit: 50}, 0))

	assert.Contains(t, rec.Body.String(), `"data":[]`)
	assert.Contains(t, rec.Body.String(), `"pagination":{"limit":50,"offset":0,"total":0}`)
}

func TestRespondAPIErrorHTTP_CodeFromStatus(t *testing.T) {
	rec := httptest.NewRecorder()

	require.NoError(t, handlers.RespondAPIError(rec, http.StatusNotFound, "", "Not Found"))

	var resp handlers.ErrorResponse
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
	assert.Equal(t, handlers.ErrCodeNotFound, resp.Error.Code)
}
