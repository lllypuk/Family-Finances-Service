package integration_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		members, err := testServer.Repos.User.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)

		assert.Len(t, members, 2)

		userEmails := []string{members[0].Email, members[1].Email}
		assert.Contains(t, userEmails, "member1@integration.com")
		assert.Contains(t, userEmails, "member2@integration.com")
	})
}
