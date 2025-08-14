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

func TestFamilyRepository_Integration(t *testing.T) {
	mongoContainer := testhelpers.SetupMongoDB(t)
	repo := userrepo.NewFamilyRepository(mongoContainer.Database)

	t.Run("Create_Success", func(t *testing.T) {
		testFamily := testhelpers.CreateTestFamily()

		err := repo.Create(context.Background(), testFamily)
		require.NoError(t, err)
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		testFamily := testhelpers.CreateTestFamily()

		err := repo.Create(context.Background(), testFamily)
		require.NoError(t, err)

		retrievedFamily, err := repo.GetByID(context.Background(), testFamily.ID)
		require.NoError(t, err)
		assert.Equal(t, testFamily.ID, retrievedFamily.ID)
		assert.Equal(t, testFamily.Name, retrievedFamily.Name)
		// Family structure doesn't have CreatedBy field
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := repo.GetByID(context.Background(), nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Update_Success", func(t *testing.T) {
		testFamily := testhelpers.CreateTestFamily()

		err := repo.Create(context.Background(), testFamily)
		require.NoError(t, err)

		testFamily.Name = "Updated Family Name"
		err = repo.Update(context.Background(), testFamily)
		require.NoError(t, err)

		retrievedFamily, err := repo.GetByID(context.Background(), testFamily.ID)
		require.NoError(t, err)
		assert.Equal(t, "Updated Family Name", retrievedFamily.Name)
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		nonExistentFamily := testhelpers.CreateTestFamily()

		err := repo.Update(context.Background(), nonExistentFamily)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_Success", func(t *testing.T) {
		testFamily := testhelpers.CreateTestFamily()

		err := repo.Create(context.Background(), testFamily)
		require.NoError(t, err)

		err = repo.Delete(context.Background(), testFamily.ID)
		require.NoError(t, err)

		_, err = repo.GetByID(context.Background(), testFamily.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("Delete_NotFound", func(t *testing.T) {
		nonExistentID := uuid.New()
		err := repo.Delete(context.Background(), nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}
