package integration_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/testhelpers"
)

func TestFamilyRepository_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)

	t.Run("CreateAndGetFamily", func(t *testing.T) {
		// Create family directly via repository
		family := testhelpers.CreateTestFamily()
		err := testServer.Repos.Family.Create(context.Background(), family)
		require.NoError(t, err)

		// Get the single family
		foundFamily, err := testServer.Repos.Family.Get(context.Background())
		require.NoError(t, err)

		assert.Equal(t, family.ID, foundFamily.ID)
		assert.Equal(t, family.Name, foundFamily.Name)
		assert.Equal(t, family.Currency, foundFamily.Currency)
	})

	t.Run("FamilyExists", func(t *testing.T) {
		exists, err := testServer.Repos.Family.Exists(context.Background())
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("GetFamilyMembers", func(t *testing.T) {
		// Get the existing family
		family, err := testServer.Repos.Family.Get(context.Background())
		require.NoError(t, err)

		// Create users in the family
		user1 := testhelpers.CreateTestUser(family.ID)
		user1.Email = "member1@integration.com"
		user2 := testhelpers.CreateTestUser(family.ID)
		user2.Email = "member2@integration.com"

		err = testServer.Repos.User.Create(context.Background(), user1)
		require.NoError(t, err)
		err = testServer.Repos.User.Create(context.Background(), user2)
		require.NoError(t, err)

		// Get family members
		members, err := testServer.Repos.User.GetAll(context.Background())
		require.NoError(t, err)

		assert.Len(t, members, 2)

		userEmails := []string{members[0].Email, members[1].Email}
		assert.Contains(t, userEmails, "member1@integration.com")
		assert.Contains(t, userEmails, "member2@integration.com")
	})
}

// TestFamilyAPI_Integration покрывает роуты /api/v1/family: чтение доступно
// любой роли, изменение — только админу.
func TestFamilyAPI_Integration(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	adminAuth := testServer.Auth(t)
	_, memberAuth := testServer.AuthAs(t, user.RoleMember)

	do := func(t *testing.T, method, body string, auth *testhelpers.AuthSession) *httptest.ResponseRecorder {
		t.Helper()

		var reader io.Reader
		if body != "" {
			reader = bytes.NewBufferString(body)
		}
		req := httptest.NewRequest(method, "/api/v1/family", reader)
		req.Header.Set("Content-Type", "application/json")
		auth.Apply(req)
		rec := httptest.NewRecorder()
		testServer.Server.Echo().ServeHTTP(rec, req)

		return rec
	}

	t.Run("GetFamily_MemberAllowed", func(t *testing.T) {
		rec := do(t, http.MethodGet, "", memberAuth)
		require.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())

		var response handlers.APIResponse[handlers.FamilyResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, testServer.AuthFamily.ID, response.Data.ID)
	})

	t.Run("UpdateFamily_MemberForbidden", func(t *testing.T) {
		rec := do(t, http.MethodPut, `{"name":"Hijacked"}`, memberAuth)
		assert.Equal(t, http.StatusForbidden, rec.Code, "тело: %s", rec.Body.String())
	})

	t.Run("UpdateFamily_AdminRenames", func(t *testing.T) {
		rec := do(t, http.MethodPut, `{"name":"Renamed Family","currency":"USD"}`, adminAuth)
		require.Equal(t, http.StatusOK, rec.Code, "тело: %s", rec.Body.String())

		var response handlers.APIResponse[handlers.FamilyResponse]
		require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &response))
		assert.Equal(t, "Renamed Family", response.Data.Name)
		assert.Equal(t, "USD", response.Data.Currency)

		stored, err := testServer.Repos.Family.Get(context.Background())
		require.NoError(t, err)
		assert.Equal(t, "Renamed Family", stored.Name)
	})

	t.Run("UpdateFamily_ValidationError", func(t *testing.T) {
		rec := do(t, http.MethodPut, `{"currency":"EURO"}`, adminAuth)
		assert.Equal(t, http.StatusBadRequest, rec.Code, "тело: %s", rec.Body.String())
	})
}
