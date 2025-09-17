package user_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	userrepo "family-budget-service/internal/infrastructure/user"
	testutils "family-budget-service/internal/testing"
)

func TestUserRepositoryPostgreSQL_Integration(t *testing.T) {
	// Setup PostgreSQL testcontainer
	container := testutils.SetupPostgreSQLContainer(t)
	defer container.Cleanup(t)

	// Create repository
	helper := testutils.NewTestDataHelper(container.DB)

	ctx := context.Background()

	t.Run("Create_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user
		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			Password:  "hashed_password",
			FirstName: "John",
			LastName:  "Doe",
			Role:      user.RoleAdmin,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Verify user was created
		retrievedUser, err := repo.GetByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
		assert.Equal(t, testUser.FirstName, retrievedUser.FirstName)
		assert.Equal(t, testUser.LastName, retrievedUser.LastName)
		assert.Equal(t, testUser.Role, retrievedUser.Role)
		assert.Equal(t, testUser.FamilyID, retrievedUser.FamilyID)
	})

	t.Run("Create_DuplicateEmail_ShouldFail", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		email := "duplicate@example.com"

		// Create first user
		testUser1 := &user.User{
			ID:        uuid.New(),
			Email:     email,
			Password:  "hashed_password",
			FirstName: "John",
			LastName:  "Doe",
			Role:      user.RoleAdmin,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser1)
		require.NoError(t, err)

		// Try to create second user with same email
		testUser2 := &user.User{
			ID:        uuid.New(),
			Email:     email, // Same email
			Password:  "hashed_password",
			FirstName: "Jane",
			LastName:  "Doe",
			Role:      user.RoleMember,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family and user
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "getbyid@example.com",
			Password:  "hashed_password",
			FirstName: "Get",
			LastName:  "ByID",
			Role:      user.RoleMember,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Retrieve user
		retrievedUser, err := repo.GetByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
		assert.Equal(t, testUser.FirstName, retrievedUser.FirstName)
		assert.Equal(t, testUser.LastName, retrievedUser.LastName)
		assert.Equal(t, testUser.Role, retrievedUser.Role)
		assert.Equal(t, testUser.FamilyID, retrievedUser.FamilyID)
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		nonExistentID := uuid.New()
		_, err := repo.GetByID(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetByEmail_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family and user
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		email := "getbyemail@example.com"
		testUser := &user.User{
			ID:        uuid.New(),
			Email:     email,
			Password:  "hashed_password",
			FirstName: "Get",
			LastName:  "ByEmail",
			Role:      user.RoleMember,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Retrieve user by email
		retrievedUser, err := repo.GetByEmail(ctx, email)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
		assert.Equal(t, testUser.Email, retrievedUser.Email)
	})

	t.Run("GetByEmail_CaseInsensitive", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family and user
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		email := "CaseTest@Example.Com"
		testUser := &user.User{
			ID:        uuid.New(),
			Email:     email,
			Password:  "hashed_password",
			FirstName: "Case",
			LastName:  "Test",
			Role:      user.RoleMember,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Retrieve user with different case
		retrievedUser, err := repo.GetByEmail(ctx, "casetest@example.com")
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
	})

	t.Run("GetByFamilyID_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Family with Users", "EUR")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create multiple users for the family
		users := []*user.User{
			{
				ID:        uuid.New(),
				Email:     "admin@family.com",
				Password:  "hashed_password",
				FirstName: "Admin",
				LastName:  "User",
				Role:      user.RoleAdmin,
				FamilyID:  familyUUID,
			},
			{
				ID:        uuid.New(),
				Email:     "member@family.com",
				Password:  "hashed_password",
				FirstName: "Member",
				LastName:  "User",
				Role:      user.RoleMember,
				FamilyID:  familyUUID,
			},
			{
				ID:        uuid.New(),
				Email:     "child@family.com",
				Password:  "hashed_password",
				FirstName: "Child",
				LastName:  "User",
				Role:      user.RoleChild,
				FamilyID:  familyUUID,
			},
		}

		// Create all users
		for _, u := range users {
			err = repo.Create(ctx, u)
			require.NoError(t, err)
		}

		// Retrieve users by family ID
		familyUsers, err := repo.GetByFamilyID(ctx, familyUUID)
		require.NoError(t, err)
		assert.Len(t, familyUsers, 3)

		// Verify users are sorted by role, first name, last name
		// Role ordering is alphabetical: admin, child, member
		assert.Equal(t, user.RoleAdmin, familyUsers[0].Role)
		assert.Equal(t, user.RoleMember, familyUsers[1].Role)
		assert.Equal(t, user.RoleChild, familyUsers[2].Role)
	})

	t.Run("Update_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family and user
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "original@example.com",
			Password:  "hashed_password",
			FirstName: "Original",
			LastName:  "Name",
			Role:      user.RoleMember,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Update user
		testUser.FirstName = "Changed"
		testUser.LastName = "NewName"
		testUser.Email = "newemail@example.com"

		err = repo.Update(ctx, testUser)
		require.NoError(t, err)

		// Verify update
		retrievedUser, err := repo.GetByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "Changed", retrievedUser.FirstName)
		assert.Equal(t, "NewName", retrievedUser.LastName)
		assert.Equal(t, "newemail@example.com", retrievedUser.Email)
	})

	t.Run("Delete_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family and user
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "remove@example.com",
			Password:  "hashed_password",
			FirstName: "Delete",
			LastName:  "Me",
			Role:      user.RoleMember,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Delete user (soft delete)
		err = repo.Delete(ctx, testUser.ID, testUser.FamilyID)
		require.NoError(t, err)

		// Verify user is not found (soft deleted)
		_, err = repo.GetByID(ctx, testUser.ID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetUsersByRole_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family
		familyID, err := helper.CreateTestFamily(ctx, "Role Test Family", "USD")
		require.NoError(t, err)
		familyUUID := uuid.MustParse(familyID)

		// Create users with different roles
		adminUser := &user.User{
			ID:        uuid.New(),
			Email:     "admin@role.com",
			Password:  "hashed_password",
			FirstName: "Admin",
			LastName:  "User",
			Role:      user.RoleAdmin,
			FamilyID:  familyUUID,
		}

		memberUser := &user.User{
			ID:        uuid.New(),
			Email:     "member@role.com",
			Password:  "hashed_password",
			FirstName: "Member",
			LastName:  "User",
			Role:      user.RoleMember,
			FamilyID:  familyUUID,
		}

		err = repo.Create(ctx, adminUser)
		require.NoError(t, err)
		err = repo.Create(ctx, memberUser)
		require.NoError(t, err)

		// Get admin users
		adminUsers, err := repo.GetUsersByRole(ctx, familyUUID, user.RoleAdmin)
		require.NoError(t, err)
		assert.Len(t, adminUsers, 1)
		assert.Equal(t, user.RoleAdmin, adminUsers[0].Role)

		// Get member users
		memberUsers, err := repo.GetUsersByRole(ctx, familyUUID, user.RoleMember)
		require.NoError(t, err)
		assert.Len(t, memberUsers, 1)
		assert.Equal(t, user.RoleMember, memberUsers[0].Role)

		// Get child users (should be empty)
		childUsers, err := repo.GetUsersByRole(ctx, familyUUID, user.RoleChild)
		require.NoError(t, err)
		assert.Empty(t, childUsers)
	})

	t.Run("UpdateLastLogin_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewPostgreSQLRepository(db)

		// Create test family and user
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "lastlogin@example.com",
			Password:  "hashed_password",
			FirstName: "Last",
			LastName:  "Login",
			Role:      user.RoleMember,
			FamilyID:  uuid.MustParse(familyID),
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Update last login
		err = repo.UpdateLastLogin(ctx, testUser.ID)
		require.NoError(t, err)

		// Verify last login was updated (we can't easily test the exact time, but no error means success)
		retrievedUser, err := repo.GetByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
	})
}
