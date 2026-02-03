package integration_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/services/dto"
	"family-budget-service/internal/testhelpers"
)

func TestInviteFlow_FullCycle(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Step 1: Create family and admin user (via setup)
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	family, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)
	require.NotNil(t, family)

	// Get the admin user
	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)
	require.NotNil(t, admin)
	assert.Equal(t, user.RoleAdmin, admin.Role)

	// Step 2: Admin creates invite
	createDTO := dto.CreateInviteDTO{
		Email: "newuser@test.com",
		Role:  "member",
	}
	invite, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, createDTO)
	require.NoError(t, err)
	assert.Equal(t, user.InviteStatusPending, invite.Status)
	assert.NotEmpty(t, invite.Token)
	assert.Equal(t, "newuser@test.com", invite.Email)
	assert.Equal(t, user.RoleMember, invite.Role)

	// Step 3: Get invite by token
	fetchedInvite, err := testServer.Services.Invite.GetInviteByToken(ctx, invite.Token)
	require.NoError(t, err)
	assert.Equal(t, "newuser@test.com", fetchedInvite.Email)
	assert.Equal(t, invite.ID, fetchedInvite.ID)

	// Step 4: Accept invite
	acceptDTO := dto.AcceptInviteDTO{
		Email:    "newuser@test.com",
		Name:     "New User",
		Password: "newpassword123",
	}
	newUser, err := testServer.Services.Invite.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.NoError(t, err)
	assert.Equal(t, "newuser@test.com", newUser.Email)
	assert.Equal(t, user.RoleMember, newUser.Role)
	assert.NotEmpty(t, newUser.ID)

	// Step 5: Verify invite is now accepted and cannot be reused
	// Try to use the same token again - should fail
	_, err = testServer.Services.Invite.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.Error(t, err) // Should fail — invite already used

	// Step 6: Verify user can be found
	foundUser, err := testServer.Repos.User.GetByEmail(ctx, "newuser@test.com")
	require.NoError(t, err)
	assert.Equal(t, newUser.ID, foundUser.ID)
	assert.Equal(t, newUser.Email, foundUser.Email)
}

func TestInviteFlow_ExpiredInvite(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Setup family and admin
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	_, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)

	// Create invite
	createDTO := dto.CreateInviteDTO{
		Email: "expired@test.com",
		Role:  "member",
	}
	invite, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, createDTO)
	require.NoError(t, err)

	// Manually set expiration to the past using direct SQL
	pastTime := time.Now().Add(-24 * time.Hour)
	query := "UPDATE invites SET expires_at = ? WHERE id = ?"
	_, err = testServer.Container.DB.ExecContext(ctx, query, pastTime.Format(time.RFC3339), invite.ID.String())
	require.NoError(t, err)

	// Attempt to accept — should get ErrInviteExpired
	acceptDTO := dto.AcceptInviteDTO{
		Email:    "expired@test.com",
		Name:     "Expired User",
		Password: "password123",
	}
	_, err = testServer.Services.Invite.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "expired")
}

func TestInviteFlow_RevokedInvite(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Setup family and admin
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	_, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)

	// Create invite
	createDTO := dto.CreateInviteDTO{
		Email: "revoked@test.com",
		Role:  "member",
	}
	invite, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, createDTO)
	require.NoError(t, err)

	// Revoke the invite
	err = testServer.Services.Invite.RevokeInvite(ctx, invite.ID, admin.ID)
	require.NoError(t, err)

	// Attempt to accept — should get ErrInviteRevoked
	acceptDTO := dto.AcceptInviteDTO{
		Email:    "revoked@test.com",
		Name:     "Revoked User",
		Password: "password123",
	}
	_, err = testServer.Services.Invite.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "revoked")
}

func TestInviteFlow_DuplicateEmail(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Setup family and admin with existing user
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	_, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)

	// Try to create invite with existing admin email
	createDTO := dto.CreateInviteDTO{
		Email: "admin@test.com", // Email already exists
		Role:  "member",
	}
	_, err = testServer.Services.Invite.CreateInvite(ctx, admin.ID, createDTO)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInviteFlow_EmailMismatch(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Setup family and admin
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	_, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)

	// Create invite for user@a.com
	createDTO := dto.CreateInviteDTO{
		Email: "user@a.com",
		Role:  "member",
	}
	invite, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, createDTO)
	require.NoError(t, err)

	// Try to accept with user@b.com
	acceptDTO := dto.AcceptInviteDTO{
		Email:    "user@b.com", // Different email
		Name:     "Mismatched User",
		Password: "password123",
	}
	_, err = testServer.Services.Invite.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "does not match")
}

func TestInviteFlow_DoubleAccept(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Setup family and admin
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	_, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)

	// Create invite
	createDTO := dto.CreateInviteDTO{
		Email: "doubleaccept@test.com",
		Role:  "member",
	}
	invite, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, createDTO)
	require.NoError(t, err)

	// Accept invite first time
	acceptDTO := dto.AcceptInviteDTO{
		Email:    "doubleaccept@test.com",
		Name:     "Double Accept User",
		Password: "password123",
	}
	_, err = testServer.Services.Invite.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.NoError(t, err)

	// Try to accept again
	_, err = testServer.Services.Invite.AcceptInvite(ctx, invite.Token, acceptDTO)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already")
}

func TestInviteFlow_ListFamilyInvites(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Setup family and admin
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	family, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)

	// Create multiple invites
	invite1DTO := dto.CreateInviteDTO{
		Email: "user1@test.com",
		Role:  "member",
	}
	invite1, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, invite1DTO)
	require.NoError(t, err)

	invite2DTO := dto.CreateInviteDTO{
		Email: "user2@test.com",
		Role:  "child",
	}
	invite2, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, invite2DTO)
	require.NoError(t, err)

	// List all invites for family
	invites, err := testServer.Services.Invite.ListFamilyInvites(ctx, family.ID)
	require.NoError(t, err)
	assert.Len(t, invites, 2)

	// Verify invites are in the list
	inviteIDs := []string{invites[0].ID.String(), invites[1].ID.String()}
	assert.Contains(t, inviteIDs, invite1.ID.String())
	assert.Contains(t, inviteIDs, invite2.ID.String())
}

func TestInviteFlow_DeleteExpiredInvites(t *testing.T) {
	testServer := testhelpers.SetupHTTPServer(t)
	ctx := context.Background()

	// Setup family and admin
	setupDTO := dto.SetupFamilyDTO{
		FamilyName: "Test Family",
		Currency:   "USD",
		Email:      "admin@test.com",
		FirstName:  "Admin",
		LastName:   "User",
		Password:   "securepassword123",
	}
	family, err := testServer.Services.Family.SetupFamily(ctx, setupDTO)
	require.NoError(t, err)

	admin, err := testServer.Repos.User.GetByEmail(ctx, "admin@test.com")
	require.NoError(t, err)

	// Create valid invite
	validDTO := dto.CreateInviteDTO{
		Email: "valid@test.com",
		Role:  "member",
	}
	validInvite, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, validDTO)
	require.NoError(t, err)

	// Create expired invite
	expiredDTO := dto.CreateInviteDTO{
		Email: "expired@test.com",
		Role:  "member",
	}
	expiredInvite, err := testServer.Services.Invite.CreateInvite(ctx, admin.ID, expiredDTO)
	require.NoError(t, err)

	// Set expiration to the past and mark as expired using direct SQL
	pastTime := time.Now().Add(-24 * time.Hour)
	expiredInvite.MarkExpired()
	query := "UPDATE invites SET expires_at = ?, status = ?, updated_at = ? WHERE id = ?"
	_, err = testServer.Container.DB.ExecContext(ctx, query,
		pastTime.Format(time.RFC3339),
		string(expiredInvite.Status),
		time.Now().Format(time.RFC3339),
		expiredInvite.ID.String())
	require.NoError(t, err)

	// Delete expired invites
	err = testServer.Services.Invite.DeleteExpiredInvites(ctx)
	require.NoError(t, err)

	// Verify valid invite still exists
	invites, err := testServer.Services.Invite.ListFamilyInvites(ctx, family.ID)
	require.NoError(t, err)

	// Should have at least the valid invite
	var foundValid bool
	for _, inv := range invites {
		if inv.ID == validInvite.ID {
			foundValid = true
		}
	}
	assert.True(t, foundValid, "Valid invite should still exist")
}
