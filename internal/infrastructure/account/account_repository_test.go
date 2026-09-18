package account_test

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/account"
	accountrepo "family-budget-service/internal/infrastructure/account"
	"family-budget-service/internal/testhelpers"
)

func setupRepo(t *testing.T) (*accountrepo.SQLiteRepository, *testhelpers.SQLiteTestDB, string) {
	t.Helper()

	container := testhelpers.SetupSQLiteTestDB(t)
	familyID, err := testhelpers.NewTestDataHelper(container.DB).CreateTestFamily(t.Context(), "Family", "RUB")
	require.NoError(t, err)

	return accountrepo.NewSQLiteRepository(container.DB), container, familyID
}

func create(t *testing.T, repo *accountrepo.SQLiteRepository, name string) *account.Account {
	t.Helper()

	a := &account.Account{ID: uuid.New(), Name: name}
	require.NoError(t, repo.Create(t.Context(), a))

	return a
}

func archive(t *testing.T, repo *accountrepo.SQLiteRepository, id uuid.UUID) {
	t.Helper()

	archived := true
	_, err := repo.Update(t.Context(), id, nil, &archived)
	require.NoError(t, err)
}

func TestAccountRepository_Create_GetByID(t *testing.T) {
	repo, _, _ := setupRepo(t)
	created := create(t, repo, "Тинькофф")

	got, err := repo.GetByID(t.Context(), created.ID)
	require.NoError(t, err)
	assert.Equal(t, "Тинькофф", got.Name)
	assert.False(t, got.IsArchived)
	assert.False(t, got.CreatedAt.IsZero())

	_, err = repo.GetByID(t.Context(), uuid.New())
	require.ErrorIs(t, err, account.ErrNotFound)
}

func TestAccountRepository_Create_NameKeyUnique(t *testing.T) {
	repo, _, _ := setupRepo(t)
	archived := create(t, repo, "Карта")
	archive(t, repo, archived.ID)

	err := repo.Create(t.Context(), &account.Account{ID: uuid.New(), Name: " карта "})
	require.ErrorIs(t, err, account.ErrNameExists, "архивный счёт имя занимает")
}

func TestAccountRepository_Create_DuplicateIDIsNotNameConflict(t *testing.T) {
	repo, _, _ := setupRepo(t)
	first := create(t, repo, "Сбер")

	err := repo.Create(t.Context(), &account.Account{ID: first.ID, Name: "Альфа"})
	require.Error(t, err)
	assert.NotErrorIs(t, err, account.ErrNameExists)
}

func TestAccountRepository_List(t *testing.T) {
	repo, _, _ := setupRepo(t)
	create(t, repo, "сбер")
	create(t, repo, "Альфа")
	old := create(t, repo, "Архив")
	archive(t, repo, old.ID)

	active, err := repo.List(t.Context(), false)
	require.NoError(t, err)
	require.Len(t, active, 2)
	assert.Equal(t, "Альфа", active[0].Name)
	assert.Equal(t, "сбер", active[1].Name)

	all, err := repo.List(t.Context(), true)
	require.NoError(t, err)
	assert.Len(t, all, 3)
}

func TestAccountRepository_Update(t *testing.T) {
	repo, _, _ := setupRepo(t)
	a := create(t, repo, "Сбер")
	other := create(t, repo, "Альфа")

	name := "Сбер 2"
	updated, err := repo.Update(t.Context(), a.ID, &name, nil)
	require.NoError(t, err)
	assert.Equal(t, "Сбер 2", updated.Name)
	got, err := repo.GetByID(t.Context(), a.ID)
	require.NoError(t, err)
	assert.Equal(t, "Сбер 2", got.Name)

	clash := "СБЕР 2"
	_, err = repo.Update(t.Context(), other.ID, &clash, nil)
	require.ErrorIs(t, err, account.ErrNameExists)

	_, err = repo.Update(t.Context(), uuid.New(), &name, nil)
	require.ErrorIs(t, err, account.ErrNotFound)
}

func TestAccountRepository_Update_KeepsFieldsNotGiven(t *testing.T) {
	repo, _, _ := setupRepo(t)
	a := create(t, repo, "Сбер")

	archive(t, repo, a.ID)
	name := "Сбер старый"
	updated, err := repo.Update(t.Context(), a.ID, &name, nil)
	require.NoError(t, err)
	assert.True(t, updated.IsArchived, "переименование не снимает архив")

	archived := false
	updated, err = repo.Update(t.Context(), a.ID, nil, &archived)
	require.NoError(t, err)
	assert.Equal(t, "Сбер старый", updated.Name)

	create(t, repo, "Карта")
	key := "карта"
	_, err = repo.Update(t.Context(), a.ID, &key, nil)
	require.ErrorIs(t, err, account.ErrNameExists)
}

func TestAccountRepository_Delete(t *testing.T) {
	repo, container, familyID := setupRepo(t)
	free := create(t, repo, "Свободный")
	used := create(t, repo, "Занятый")
	reconciled := create(t, repo, "Сверенный")

	helper := testhelpers.NewTestDataHelper(container.DB)
	userID, err := helper.CreateTestUser(t.Context(), "a@example.com", "A", "B", "admin", familyID)
	require.NoError(t, err)
	categoryID, err := helper.CreateTestCategory(t.Context(), "Еда", "expense", familyID, nil)
	require.NoError(t, err)
	txID, err := helper.CreateTestTransaction(t.Context(), 100, "x", "expense", categoryID, userID, familyID)
	require.NoError(t, err)
	_, err = container.DB.ExecContext(t.Context(),
		`UPDATE transactions SET account_id = ? WHERE id = ?`, used.ID.String(), txID)
	require.NoError(t, err)
	_, err = container.DB.ExecContext(t.Context(),
		`INSERT INTO account_reconciliations (account_id, month, bank_expense_minor) VALUES (?, '2026-09', 0)`,
		reconciled.ID.String())
	require.NoError(t, err)

	require.NoError(t, repo.Delete(t.Context(), free.ID))
	require.ErrorIs(t, repo.Delete(t.Context(), free.ID), account.ErrNotFound)
	require.ErrorIs(t, repo.Delete(t.Context(), used.ID), account.ErrInUse)
	require.ErrorIs(t, repo.Delete(t.Context(), reconciled.ID), account.ErrInUse)
}
