package user_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	userrepo "family-budget-service/internal/infrastructure/user"
	testutils "family-budget-service/internal/testhelpers"
)

func TestInviteRepositorySQLite_Integration(t *testing.T) {
	// Setup SQLite in-memory database
	container := testutils.SetupSQLiteTestDB(t)

	// Create repository
	helper := testutils.NewTestDataHelper(container.DB)

	ctx := context.Background()

	t.Run("Create_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create test invite
		invite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"test@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)

		err = repo.Create(invite)
		require.NoError(t, err)

		// Verify invite was created
		retrievedInvite, err := repo.GetByID(invite.ID)
		require.NoError(t, err)
		assert.Equal(t, invite.ID, retrievedInvite.ID)
		assert.Equal(t, invite.Email, retrievedInvite.Email)
		assert.Equal(t, invite.Token, retrievedInvite.Token)
		assert.Equal(t, invite.Status, retrievedInvite.Status)
		assert.Equal(t, invite.Role, retrievedInvite.Role)
	})

	t.Run("GetByToken_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create test invite
		invite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"token@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)

		err = repo.Create(invite)
		require.NoError(t, err)

		// Retrieve by token
		retrievedInvite, err := repo.GetByToken(invite.Token)
		require.NoError(t, err)
		assert.Equal(t, invite.ID, retrievedInvite.ID)
		assert.Equal(t, invite.Token, retrievedInvite.Token)
	})

	t.Run("GetByToken_NotFound", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Try to get non-existent token
		_, err := repo.GetByToken("nonexistent-token")
		require.Error(t, err)
	})

	t.Run("GetByFamily_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create multiple invites
		invite1, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"user1@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(invite1)
		require.NoError(t, err)

		invite2, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"user2@example.com",
			user.RoleChild,
		)
		require.NoError(t, err)
		err = repo.Create(invite2)
		require.NoError(t, err)

		// Retrieve all family invites
		invites, err := repo.GetByFamily(uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Len(t, invites, 2)
	})

	t.Run("GetPendingByEmail_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		email := "pending@example.com"

		// Create pending invite
		invite, err := user.NewInvite(uuid.MustParse(familyID), uuid.MustParse(creatorID), email, user.RoleMember)
		require.NoError(t, err)
		err = repo.Create(invite)
		require.NoError(t, err)

		// Retrieve pending invites
		invites, err := repo.GetPendingByEmail(email)
		require.NoError(t, err)
		assert.Len(t, invites, 1)
		assert.Equal(t, user.InviteStatusPending, invites[0].Status)
	})

	t.Run("Update_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create test invite
		invite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"update@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(invite)
		require.NoError(t, err)

		// Create accepting user
		acceptingUserID, err := helper.CreateTestUser(
			ctx,
			"accepting@example.com",
			"Accept",
			"User",
			"member",
			familyID,
		)
		require.NoError(t, err)

		// Accept the invite
		invite.Accept(uuid.MustParse(acceptingUserID))

		// Update invite
		err = repo.Update(invite)
		require.NoError(t, err)

		// Verify update
		retrievedInvite, err := repo.GetByID(invite.ID)
		require.NoError(t, err)
		assert.Equal(t, user.InviteStatusAccepted, retrievedInvite.Status)
		assert.NotNil(t, retrievedInvite.AcceptedBy)
		assert.Equal(t, uuid.MustParse(acceptingUserID), *retrievedInvite.AcceptedBy)
		assert.NotNil(t, retrievedInvite.AcceptedAt)
	})

	t.Run("Delete_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create test invite
		invite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"delete@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(invite)
		require.NoError(t, err)

		// Delete invite
		err = repo.Delete(invite.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetByID(invite.ID)
		require.Error(t, err)
	})

	t.Run("DeleteExpired_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create expired invite
		expiredInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"expired@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		// Set expiration to past
		expiredInvite.ExpiresAt = time.Now().Add(-24 * time.Hour)
		err = repo.Create(expiredInvite)
		require.NoError(t, err)

		// Create valid invite
		validInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"valid@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(validInvite)
		require.NoError(t, err)

		// Delete expired invites
		err = repo.DeleteExpired()
		require.NoError(t, err)

		// Verify expired invite is deleted
		_, err = repo.GetByID(expiredInvite.ID)
		require.Error(t, err)

		// Verify valid invite still exists
		_, err = repo.GetByID(validInvite.ID)
		require.NoError(t, err)
	})

	t.Run("Revoke_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create test invite
		invite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"revoke@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(invite)
		require.NoError(t, err)

		// Revoke the invite
		invite.Revoke()

		// Update invite
		err = repo.Update(invite)
		require.NoError(t, err)

		// Verify revocation
		retrievedInvite, err := repo.GetByID(invite.ID)
		require.NoError(t, err)
		assert.Equal(t, user.InviteStatusRevoked, retrievedInvite.Status)
	})

	t.Run("GetByFamily_EmptyResult", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Get invites for family with no invites
		invites, err := repo.GetByFamily(uuid.MustParse(familyID))
		require.NoError(t, err)
		assert.Empty(t, invites)
	})

	t.Run("GetPendingByEmail_OnlyPending", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		email := "test@example.com"

		// Create pending invite
		pendingInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			email,
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(pendingInvite)
		require.NoError(t, err)

		// Create accepted invite with same email
		acceptedInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			email,
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(acceptedInvite)
		require.NoError(t, err)

		// Create accepting user
		acceptingUserID, err := helper.CreateTestUser(
			ctx,
			"accepting@example.com",
			"Accept",
			"User",
			"member",
			familyID,
		)
		require.NoError(t, err)

		acceptedInvite.Accept(uuid.MustParse(acceptingUserID))
		err = repo.Update(acceptedInvite)
		require.NoError(t, err)

		// Get pending invites
		invites, err := repo.GetPendingByEmail(email)
		require.NoError(t, err)
		assert.Len(t, invites, 1)
		assert.Equal(t, user.InviteStatusPending, invites[0].Status)
		assert.Equal(t, pendingInvite.ID, invites[0].ID)
	})
}
