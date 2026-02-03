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

		err = repo.Create(ctx, invite)
		require.NoError(t, err)

		// Verify invite was created
		retrievedInvite, err := repo.GetByID(ctx, invite.ID)
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

		err = repo.Create(ctx, invite)
		require.NoError(t, err)

		// Retrieve by token
		retrievedInvite, err := repo.GetByToken(ctx, invite.Token)
		require.NoError(t, err)
		assert.Equal(t, invite.ID, retrievedInvite.ID)
		assert.Equal(t, invite.Token, retrievedInvite.Token)
	})

	t.Run("GetByToken_NotFound", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Try to get non-existent token
		_, err := repo.GetByToken(ctx, "nonexistent-token")
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
		err = repo.Create(ctx, invite1)
		require.NoError(t, err)

		invite2, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"user2@example.com",
			user.RoleChild,
		)
		require.NoError(t, err)
		err = repo.Create(ctx, invite2)
		require.NoError(t, err)

		// Retrieve all family invites
		invites, err := repo.GetByFamily(ctx, uuid.MustParse(familyID))
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
		err = repo.Create(ctx, invite)
		require.NoError(t, err)

		// Retrieve pending invites
		invites, err := repo.GetPendingByEmail(ctx, email)
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
		err = repo.Create(ctx, invite)
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
		err = repo.Update(ctx, invite)
		require.NoError(t, err)

		// Verify update
		retrievedInvite, err := repo.GetByID(ctx, invite.ID)
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
		err = repo.Create(ctx, invite)
		require.NoError(t, err)

		// Delete invite
		err = repo.Delete(ctx, invite.ID)
		require.NoError(t, err)

		// Verify deletion
		_, err = repo.GetByID(ctx, invite.ID)
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
		err = repo.Create(ctx, expiredInvite)
		require.NoError(t, err)

		// Create valid invite
		validInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"valid@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(ctx, validInvite)
		require.NoError(t, err)

		// Delete expired invites
		err = repo.DeleteExpired(ctx)
		require.NoError(t, err)

		// Verify expired invite is deleted
		_, err = repo.GetByID(ctx, expiredInvite.ID)
		require.Error(t, err)

		// Verify valid invite still exists
		_, err = repo.GetByID(ctx, validInvite.ID)
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
		err = repo.Create(ctx, invite)
		require.NoError(t, err)

		// Revoke the invite
		invite.Revoke()

		// Update invite
		err = repo.Update(ctx, invite)
		require.NoError(t, err)

		// Verify revocation
		retrievedInvite, err := repo.GetByID(ctx, invite.ID)
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
		invites, err := repo.GetByFamily(ctx, uuid.MustParse(familyID))
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
		err = repo.Create(ctx, pendingInvite)
		require.NoError(t, err)

		// Create accepted invite with same email
		acceptedInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			email,
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(ctx, acceptedInvite)
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
		err = repo.Update(ctx, acceptedInvite)
		require.NoError(t, err)

		// Get pending invites
		invites, err := repo.GetPendingByEmail(ctx, email)
		require.NoError(t, err)
		assert.Len(t, invites, 1)
		assert.Equal(t, user.InviteStatusPending, invites[0].Status)
		assert.Equal(t, pendingInvite.ID, invites[0].ID)
	})

	t.Run("MarkExpiredBulk_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewInviteSQLiteRepository(db)

		// Create test family first
		familyID, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user (creator)
		creatorID, err := helper.CreateTestUser(ctx, "admin@example.com", "Admin", "User", "admin", familyID)
		require.NoError(t, err)

		// Create multiple expired invites
		expiredInvite1, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"expired1@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		expiredInvite1.ExpiresAt = time.Now().Add(-24 * time.Hour)
		err = repo.Create(ctx, expiredInvite1)
		require.NoError(t, err)

		expiredInvite2, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"expired2@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		expiredInvite2.ExpiresAt = time.Now().Add(-48 * time.Hour)
		err = repo.Create(ctx, expiredInvite2)
		require.NoError(t, err)

		// Create valid invite
		validInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"valid@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		err = repo.Create(ctx, validInvite)
		require.NoError(t, err)

		// Create already expired invite (status already set)
		alreadyExpiredInvite, err := user.NewInvite(
			uuid.MustParse(familyID),
			uuid.MustParse(creatorID),
			"already-expired@example.com",
			user.RoleMember,
		)
		require.NoError(t, err)
		alreadyExpiredInvite.ExpiresAt = time.Now().Add(-72 * time.Hour)
		alreadyExpiredInvite.MarkExpired()
		err = repo.Create(ctx, alreadyExpiredInvite)
		require.NoError(t, err)

		// Mark expired invites in bulk
		rowsAffected, err := repo.MarkExpiredBulk(ctx)
		require.NoError(t, err)
		assert.Equal(t, int64(2), rowsAffected) // Only 2 pending expired invites should be affected

		// Verify expired invites are now marked as expired
		retrievedInvite1, err := repo.GetByID(ctx, expiredInvite1.ID)
		require.NoError(t, err)
		assert.Equal(t, user.InviteStatusExpired, retrievedInvite1.Status)

		retrievedInvite2, err := repo.GetByID(ctx, expiredInvite2.ID)
		require.NoError(t, err)
		assert.Equal(t, user.InviteStatusExpired, retrievedInvite2.Status)

		// Verify valid invite is still pending
		retrievedValidInvite, err := repo.GetByID(ctx, validInvite.ID)
		require.NoError(t, err)
		assert.Equal(t, user.InviteStatusPending, retrievedValidInvite.Status)

		// Verify already expired invite is still expired (not counted twice)
		retrievedAlreadyExpired, err := repo.GetByID(ctx, alreadyExpiredInvite.ID)
		require.NoError(t, err)
		assert.Equal(t, user.InviteStatusExpired, retrievedAlreadyExpired.Status)
	})
}
