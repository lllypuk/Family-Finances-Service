package integration_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/date"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
)

func TestNetWorthAPI_Series(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	_, member := ts.AuthAs(t, user.RoleMember)

	today := date.Today(ts.AuthFamily.Location())
	monthStart, _ := today.MonthBounds()
	from := monthStart.AddMonths(-2)

	flat := decodeHolding(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Квартира","side":"asset","kind":"property"}`))
	loan := decodeHolding(t, doAccountRequest(t, ts, admin, http.MethodPost, "/api/v1/holdings",
		`{"name":"Кредит","side":"liability","kind":"loan"}`))
	for path, body := range map[string]string{
		holdingValuesPath(flat.ID, from.AddDays(-1).String()):  `{"value_minor":1000}`,
		holdingValuesPath(loan.ID, from.AddMonths(1).String()): `{"value_minor":300}`,
		holdingValuesPath(flat.ID, today.String()):             `{"value_minor":1200}`,
	} {
		rec := doAccountRequest(t, ts, admin, http.MethodPut, path, body)
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())
	}

	for _, session := range []*testhelpers.AuthSession{admin, member} {
		rec := doGET(t, ts, session, "/api/v1/stats/net-worth?from="+from.String())
		require.Equal(t, http.StatusOK, rec.Code, rec.Body.String())

		var resp handlers.APIResponse[dto.StatsNetWorth]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
		assert.Equal(t, today, resp.Data.To, "to по умолчанию — сегодня")
		require.Len(t, resp.Data.Months, 3)
		assert.Equal(t, dto.NetWorthMonth{
			Month: from.MonthKey(), AssetsMinor: 1000, LiabilitiesMinor: 0, NetMinor: 1000,
		}, resp.Data.Months[0], "начальное состояние до from")
		assert.Equal(t, money.Minor(700), resp.Data.Months[1].NetMinor)
		assert.Equal(t, money.Minor(900), resp.Data.Months[2].NetMinor)
	}
}

func TestNetWorthAPI_Unauthorized(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/stats/net-worth", nil)
	rec := httptest.NewRecorder()
	ts.Server.Echo().ServeHTTP(rec, req)

	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestNetWorthAPI_Validation(t *testing.T) {
	ts := testhelpers.SetupHTTPServer(t)
	admin := ts.Auth(t)
	today := date.Today(ts.AuthFamily.Location())
	monthStart, _ := today.MonthBounds()

	tests := []struct {
		name  string
		query string
		field string
	}{
		{name: "121 months", query: "?from=" + monthStart.AddMonths(-120).String(), field: "from"},
		{name: "to tomorrow", query: "?to=" + today.AddDays(1).String(), field: "to"},
		{name: "far future beats the cap", query: "?from=0001-01-01&to=9999-12-31", field: "to"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := doGET(t, ts, admin, "/api/v1/stats/net-worth"+tt.query)
			require.Equal(t, http.StatusUnprocessableEntity, rec.Code, rec.Body.String())

			var resp handlers.ErrorResponse
			require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &resp))
			require.Len(t, resp.Error.Details, 1)
			assert.Equal(t, tt.field, resp.Error.Details[0].Field)
		})
	}
}
