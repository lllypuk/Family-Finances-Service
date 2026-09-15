package infrastructure_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/money"
	"family-budget-service/internal/infrastructure"
	authrepo "family-budget-service/internal/infrastructure/auth"
	testutils "family-budget-service/internal/testhelpers"
)

func TestMetricsReader_EmptyDatabase(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	reader := infrastructure.NewMetricsReader(container.GetTestDatabase(t))
	ctx := context.Background()

	sessions, err := reader.ActiveSessions(ctx, time.Now())
	require.NoError(t, err)
	assert.Equal(t, 0, sessions)

	active, inactive, err := reader.UsersByActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, active)
	assert.Equal(t, 0, inactive)

	transactions, err := reader.Transactions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 0, transactions)

	complete, err := reader.SetupComplete(ctx)
	require.NoError(t, err)
	assert.False(t, complete)
}

func TestMetricsReader_ActiveSessions_SkipsExpired(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	db := container.GetTestDatabase(t)
	helper := testutils.NewTestDataHelper(db)
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Second)

	familyID, err := helper.CreateTestFamily(ctx, "Test Family", "RUB")
	require.NoError(t, err)
	userID, err := helper.CreateTestUser(ctx, "anna@example.com", "Anna", "Ivanova", "admin", familyID)
	require.NoError(t, err)

	sessions := authrepo.NewSessionSQLiteRepository(db)
	live := auth.NewSession(uuid.MustParse(userID), "hash-live", "Pixel", now)
	require.NoError(t, sessions.Create(ctx, live))
	expired := auth.NewSession(uuid.MustParse(userID), "hash-expired", "Pixel", now.Add(-2*auth.IdleTTL))
	require.NoError(t, sessions.Create(ctx, expired))

	count, err := infrastructure.NewMetricsReader(db).ActiveSessions(ctx, now)
	require.NoError(t, err)
	assert.Equal(t, 1, count)
}

func TestMetricsReader_CountsUsersTransactionsAndSetup(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	db := container.GetTestDatabase(t)
	helper := testutils.NewTestDataHelper(db)
	ctx := context.Background()

	familyID, err := helper.CreateTestFamily(ctx, "Test Family", "RUB")
	require.NoError(t, err)
	userID, err := helper.CreateTestUser(ctx, "anna@example.com", "Anna", "Ivanova", "admin", familyID)
	require.NoError(t, err)
	offID, err := helper.CreateTestUser(ctx, "bob@example.com", "Bob", "Petrov", "member", familyID)
	require.NoError(t, err)
	_, err = db.ExecContext(ctx, `UPDATE users SET is_active = 0 WHERE id = ?`, offID)
	require.NoError(t, err)

	categoryID, err := helper.CreateTestCategory(ctx, "Food", "expense", familyID, nil)
	require.NoError(t, err)
	_, err = helper.CreateTestTransaction(ctx, money.Minor(1000), "lunch", "expense", categoryID, userID, familyID)
	require.NoError(t, err)

	active, inactive, err := infrastructure.NewMetricsReader(db).UsersByActive(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, active)
	assert.Equal(t, 1, inactive)

	transactions, err := infrastructure.NewMetricsReader(db).Transactions(ctx)
	require.NoError(t, err)
	assert.Equal(t, 1, transactions)

	complete, err := infrastructure.NewMetricsReader(db).SetupComplete(ctx)
	require.NoError(t, err)
	assert.True(t, complete)
}
