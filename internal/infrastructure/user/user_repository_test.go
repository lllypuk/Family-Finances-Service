package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userrepo "family-budget-service/internal/infrastructure/user"
	"family-budget-service/internal/testhelpers"
)

func TestUserRepository_Integration(t *testing.T) {
	mongoContainer := testhelpers.SetupMongoDB(t)
	repo := userrepo.NewRepository(mongoContainer.Database)

	t.Run("Create_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testUser := testhelpers.CreateTestUser(family.ID)

		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)
	})

	t.Run("Create_DuplicateEmail", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testUser1 := testhelpers.CreateTestUser(family.ID)
		testUser2 := testhelpers.CreateTestUser(family.ID)
		testUser2.Email = testUser1.Email // Same email

		err := repo.Create(context.Background(), testUser1)
		require.NoError(t, err)

		err = repo.Create(context.Background(), testUser2)
		// Without unique index, this will succeed
		require.NoError(t, err)
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testUser := testhelpers.CreateTestUser(family.ID)

		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)

		retrievedUser, err := repo.GetByID(context.Background(), testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
		assert.Equal(t, testUser.FirstName, retrievedUser.FirstName)
		assert.Equal(t, testUser.LastName, retrievedUser.LastName)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.GetByID(context.Background(), nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetByEmail_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testUser := testhelpers.CreateTestUser(family.ID)

		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)

		retrievedUser, err := repo.GetByEmail(context.Background(), testUser.Email)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
	})

	t.Run("GetByEmail_NotFound", func(t *testing.T) {
		_, err := repo.GetByEmail(context.Background(), "nonexistent@example.com")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetByFamilyID_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testUser1 := testhelpers.CreateTestUser(family.ID)
		testUser1.Email = "user1@example.com"
		testUser2 := testhelpers.CreateTestUser(family.ID)
		testUser2.Email = "user2@example.com"

		err := repo.Create(context.Background(), testUser1)
		require.NoError(t, err)
		err = repo.Create(context.Background(), testUser2)
		require.NoError(t, err)

		users, err := repo.GetByFamilyID(context.Background(), family.ID)
		require.NoError(t, err)
		assert.Len(t, users, 2)

		userIDs := make([]uuid.UUID, len(users))
		for i, u := range users {
			userIDs[i] = u.ID
		}
		assert.Contains(t, userIDs, testUser1.ID)
		assert.Contains(t, userIDs, testUser2.ID)
	})

	t.Run("GetByFamilyID_NoUsers", func(t *testing.T) {
		nonExistentFamilyID := uuid.New()
		users, err := repo.GetByFamilyID(context.Background(), nonExistentFamilyID)
		require.NoError(t, err)
		assert.Len(t, users, 0)
	})

	t.Run("Update_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testUser := testhelpers.CreateTestUser(family.ID)

		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)

		testUser.FirstName = "Jane"
		testUser.LastName = "Smith"
		err = repo.Update(context.Background(), testUser)
		require.NoError(t, err)

		retrievedUser, err := repo.GetByID(context.Background(), testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "Jane", retrievedUser.FirstName)
		assert.Equal(t, "Smith", retrievedUser.LastName)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		nonExistentUser := testhelpers.CreateTestUser(family.ID)

		err := repo.Update(context.Background(), nonExistentUser)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_Success", func(t *testing.T) {
		family := testhelpers.CreateTestFamily()
		testUser := testhelpers.CreateTestUser(family.ID)

		err := repo.Create(context.Background(), testUser)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), testUser.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(context.Background(), testUser.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(context.Background(), nonExistentID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
