package user_test

import (
	"context"
	"database/sql"
	"testing"
	"time"

	testutils "family-budget-service/internal/testhelpers"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/domain/user"
	userrepo "family-budget-service/internal/infrastructure/user"
)

func TestUserRepositorySQLite_Integration(t *testing.T) {
	// Setup SQLite in-memory database
	container := testutils.SetupSQLiteTestDB(t)

	// Create repository
	helper := testutils.NewTestDataHelper(container.DB)

	ctx := context.Background()

	t.Run("Create_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family first
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		// Create test user
		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "test@example.com",
			Password:  "hashed_password",
			FirstName: "John",
			LastName:  "Doe",
			Role:      user.RoleAdmin,
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
	})

	t.Run("Create_DuplicateEmail_ShouldFail", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family first
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
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
		}

		err = repo.Create(ctx, testUser2)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("GetByID_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family and user
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "getbyid@example.com",
			Password:  "hashed_password",
			FirstName: "Get",
			LastName:  "ByID",
			Role:      user.RoleMember,
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
	})

	t.Run("GetByID_NotFound", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		nonExistentID := uuid.New()
		_, err := repo.GetByID(ctx, nonExistentID)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("GetByEmail_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family and user
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		email := "getbyemail@example.com"
		testUser := &user.User{
			ID:        uuid.New(),
			Email:     email,
			Password:  "hashed_password",
			FirstName: "Get",
			LastName:  "ByEmail",
			Role:      user.RoleMember,
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
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family and user
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		email := "CaseTest@Example.Com"
		testUser := &user.User{
			ID:        uuid.New(),
			Email:     email,
			Password:  "hashed_password",
			FirstName: "Case",
			LastName:  "Test",
			Role:      user.RoleMember,
		}

		err = repo.Create(ctx, testUser)
		require.NoError(t, err)

		// Retrieve user with different case
		retrievedUser, err := repo.GetByEmail(ctx, "casetest@example.com")
		require.NoError(t, err)
		assert.Equal(t, testUser.ID, retrievedUser.ID)
	})

	t.Run("GetAll_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family
		_, err := helper.CreateTestFamily(ctx, "Family with Users", "EUR")
		require.NoError(t, err)

		// Create multiple users for the family
		users := []*user.User{
			{
				ID:        uuid.New(),
				Email:     "admin@family.com",
				Password:  "hashed_password",
				FirstName: "Admin",
				LastName:  "User",
				Role:      user.RoleAdmin,
			},
			{
				ID:        uuid.New(),
				Email:     "zoe@family.com",
				Password:  "hashed_password",
				FirstName: "Zoe",
				LastName:  "User",
				Role:      user.RoleMember,
			},
			{
				ID:        uuid.New(),
				Email:     "anna@family.com",
				Password:  "hashed_password",
				FirstName: "Anna",
				LastName:  "User",
				Role:      user.RoleMember,
			},
		}

		// Create all users
		for _, u := range users {
			err = repo.Create(ctx, u)
			require.NoError(t, err)
		}

		// Retrieve all users (single family model)
		allUsers, err := repo.GetAll(ctx)
		require.NoError(t, err)
		assert.Len(t, allUsers, 3)

		// Verify users are sorted by role, first name, last name
		// Role ordering is alphabetical: admin, member
		assert.Equal(t, user.RoleAdmin, allUsers[0].Role)
		assert.Equal(t, "Anna", allUsers[1].FirstName)
		assert.Equal(t, "Zoe", allUsers[2].FirstName)
	})

	t.Run("Update_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family and user
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "original@example.com",
			Password:  "hashed_password",
			FirstName: "Original",
			LastName:  "Name",
			Role:      user.RoleMember,
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

	// Update пишет только профиль: параллельная смена пароля/роли/активности не откатывается
	// значениями, прочитанными до неё.
	t.Run("Update_LeavesPasswordRoleActiveUntouched", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		stale := &user.User{
			ID: uuid.New(), Email: "stale@example.com", Password: "old-hash",
			FirstName: "Old", LastName: "Name", Role: user.RoleAdmin,
		}
		require.NoError(t, repo.Create(ctx, stale))
		other := &user.User{
			ID:        uuid.New(),
			Email:     "other@example.com",
			Password:  "x",
			FirstName: "O",
			LastName:  "T",
			Role:      user.RoleAdmin,
		}
		require.NoError(t, repo.Create(ctx, other))

		require.NoError(t, repo.UpdatePassword(ctx, stale.ID, "new-hash", uuid.Nil))
		require.NoError(t, repo.UpdateRole(ctx, stale.ID, user.RoleMember))
		require.NoError(t, repo.SetActive(ctx, stale.ID, false))

		stale.FirstName = "New"
		require.NoError(t, repo.Update(ctx, stale))

		got, err := repo.GetByID(ctx, stale.ID)
		require.NoError(t, err)
		assert.Equal(t, "New", got.FirstName)
		assert.Equal(t, "new-hash", got.Password, "Update откатил смену пароля")
		assert.Equal(t, user.RoleMember, got.Role, "Update откатил смену роли")
		assert.False(t, got.IsActive, "Update откатил деактивацию")
	})

	t.Run("Update_NotFound", func(t *testing.T) {
		repo := userrepo.NewSQLiteRepository(container.GetTestDatabase(t))
		err := repo.Update(ctx, &user.User{ID: uuid.New(), Email: "ghost@example.com"})
		require.ErrorIs(t, err, user.ErrNotFound)
	})

	// Неактивный пользователь остаётся видимым: активность решает auth, а не репозиторий.
	t.Run("SetActive_Deactivate_StillVisible", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "inactive@example.com",
			Password:  "hashed_password",
			FirstName: "Off",
			LastName:  "Line",
			Role:      user.RoleMember,
		}
		require.NoError(t, repo.Create(ctx, testUser))
		assert.True(t, testUser.IsActive, "Create всегда заводит активного пользователя")

		require.NoError(t, repo.SetActive(ctx, testUser.ID, false))

		byID, err := repo.GetByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.False(t, byID.IsActive)

		byEmail, err := repo.GetByEmail(ctx, testUser.Email)
		require.NoError(t, err)
		assert.False(t, byEmail.IsActive)

		all, err := repo.GetAll(ctx)
		require.NoError(t, err)
		found := false
		for _, u := range all {
			if u.ID == testUser.ID {
				found = true
				assert.False(t, u.IsActive)
			}
		}
		assert.True(t, found, "GetAll скрыл неактивного пользователя")

		require.NoError(t, repo.SetActive(ctx, testUser.ID, true))
		byID, err = repo.GetByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.True(t, byID.IsActive)
	})

	// Проверка «последний активный админ» и запись — одна транзакция: неактивные админы
	// не считаются, повышение и правки не-админов не проверяются.
	t.Run("GuardedWrites_LastAdmin", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)
		newAdmin := func(email string) *user.User {
			u := &user.User{
				ID:        uuid.New(),
				Email:     email,
				Password:  "x",
				FirstName: "A",
				LastName:  "D",
				Role:      user.RoleAdmin,
			}
			require.NoError(t, repo.Create(ctx, u))
			return u
		}
		first := newAdmin("first@example.com")
		second := newAdmin("second@example.com")
		member := &user.User{
			ID:        uuid.New(),
			Email:     "member@example.com",
			Password:  "x",
			FirstName: "M",
			LastName:  "B",
			Role:      user.RoleMember,
		}
		require.NoError(t, repo.Create(ctx, member))

		require.NoError(t, repo.Patch(ctx, member.ID, ptr(user.RoleMember), nil), "правка не-админа не проверяется")
		require.NoError(t, repo.Patch(ctx, first.ID, nil, ptr(false)), "второй админ остаётся")
		require.ErrorIs(t, repo.Patch(ctx, second.ID, nil, ptr(false)), user.ErrLastAdmin,
			"неактивный первый админ не считается")
		require.ErrorIs(t, repo.Patch(ctx, second.ID, ptr(user.RoleMember), nil), user.ErrLastAdmin)
		require.ErrorIs(t, repo.Patch(ctx, second.ID, ptr(user.RoleMember), ptr(false)), user.ErrLastAdmin,
			"оба поля разом проверяются так же")
		require.NoError(t, repo.Patch(ctx, second.ID, ptr(user.RoleAdmin), nil), "повышение/та же роль не проверяется")

		got, err := repo.GetByID(ctx, second.ID)
		require.NoError(t, err)
		assert.Equal(t, user.RoleAdmin, got.Role)
		assert.True(t, got.IsActive, "отклонённая запись не должна была ничего изменить")

		require.NoError(t, repo.Patch(ctx, first.ID, nil, ptr(true)))
		require.NoError(t, repo.Patch(ctx, second.ID, ptr(user.RoleMember), nil), "первый админ снова активен")

		require.ErrorIs(t, repo.Patch(ctx, uuid.New(), ptr(user.RoleMember), nil), user.ErrNotFound)
		require.ErrorIs(t, repo.Patch(ctx, uuid.New(), nil, ptr(false)), user.ErrNotFound)
		require.Error(t, repo.Patch(ctx, second.ID, nil, nil), "патч без полей — ошибка программиста")
	})

	t.Run("GetUsersByRole_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family
		_, err := helper.CreateTestFamily(ctx, "Role Test Family", "USD")
		require.NoError(t, err)

		// Create users with different roles
		adminUser := &user.User{
			ID:        uuid.New(),
			Email:     "admin@role.com",
			Password:  "hashed_password",
			FirstName: "Admin",
			LastName:  "User",
			Role:      user.RoleAdmin,
		}

		memberUser := &user.User{
			ID:        uuid.New(),
			Email:     "member@role.com",
			Password:  "hashed_password",
			FirstName: "Member",
			LastName:  "User",
			Role:      user.RoleMember,
		}

		err = repo.Create(ctx, adminUser)
		require.NoError(t, err)
		err = repo.Create(ctx, memberUser)
		require.NoError(t, err)

		// Get admin users
		adminUsers, err := repo.GetUsersByRole(ctx, user.RoleAdmin)
		require.NoError(t, err)
		assert.Len(t, adminUsers, 1)
		assert.Equal(t, user.RoleAdmin, adminUsers[0].Role)

		// Get member users
		memberUsers, err := repo.GetUsersByRole(ctx, user.RoleMember)
		require.NoError(t, err)
		assert.Len(t, memberUsers, 1)
		assert.Equal(t, user.RoleMember, memberUsers[0].Role)

		// Неизвестная роль (child удалена планом 04) — пустая выборка
		unknownUsers, err := repo.GetUsersByRole(ctx, user.Role("child"))
		require.NoError(t, err)
		assert.Empty(t, unknownUsers)
	})

	t.Run("UpdateLastLogin_Success", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		// Create test family and user
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "lastlogin@example.com",
			Password:  "hashed_password",
			FirstName: "Last",
			LastName:  "Login",
			Role:      user.RoleMember,
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

	t.Run("UpdatePassword", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		testUser := &user.User{
			ID:        uuid.New(),
			Email:     "password@example.com",
			Password:  "old_hash",
			FirstName: "Pass",
			LastName:  "Word",
			Role:      user.RoleMember,
		}
		require.NoError(t, repo.Create(ctx, testUser))

		require.NoError(t, repo.UpdatePassword(ctx, testUser.ID, "new_hash", uuid.Nil))

		retrievedUser, err := repo.GetByID(ctx, testUser.ID)
		require.NoError(t, err)
		assert.Equal(t, "new_hash", retrievedUser.Password)

		err = repo.UpdatePassword(ctx, uuid.New(), "hash", uuid.Nil)
		require.ErrorIs(t, err, user.ErrNotFound)
	})

	t.Run("GetByEmail_NotFound", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)

		_, err := repo.GetByEmail(ctx, "nobody@example.com")
		require.ErrorIs(t, err, user.ErrNotFound)
	})
}

// ptr — указатель на литерал для необязательных полей Patch.
func ptr[T any](v T) *T { return &v }

// createSession кладёт сессию пользователю напрямую: репозиторий сессий здесь не нужен,
// проверяется только то, что удаляет транзакция пользователей.
func createSession(t *testing.T, db *sql.DB, userID uuid.UUID) uuid.UUID {
	t.Helper()
	id := uuid.New()
	now := time.Now().UTC()
	_, err := db.Exec(
		`INSERT INTO sessions (id, user_id, token_hash, device_name, created_at, last_used_at, expires_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		id.String(), userID.String(), id.String(), "test", now, now, now.Add(time.Hour),
	)
	require.NoError(t, err)
	return id
}

func sessionIDs(t *testing.T, db *sql.DB, userID uuid.UUID) []string {
	t.Helper()
	rows, err := db.Query(`SELECT id FROM sessions WHERE user_id = ? ORDER BY id`, userID.String())
	require.NoError(t, err)
	defer func() { _ = rows.Close() }()
	var out []string
	for rows.Next() {
		var id string
		require.NoError(t, rows.Scan(&id))
		out = append(out, id)
	}
	require.NoError(t, rows.Err())
	return out
}

// Смена пароля и деактивация отзывают сессии той же транзакцией, что пишет в users.
func TestUserRepositorySQLite_SessionsRevokedInSameTx(t *testing.T) {
	container := testutils.SetupSQLiteTestDB(t)
	helper := testutils.NewTestDataHelper(container.DB)
	ctx := context.Background()

	newUser := func(t *testing.T, repo *userrepo.SQLiteRepository, email string, role user.Role) *user.User {
		t.Helper()
		u := &user.User{
			ID: uuid.New(), Email: email, Password: "old-hash",
			FirstName: "A", LastName: "B", Role: role,
		}
		require.NoError(t, repo.Create(ctx, u))
		return u
	}

	t.Run("UpdatePassword_KeepsOneSession", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		u := newUser(t, repo, "keep@example.com", user.RoleMember)
		keep := createSession(t, db, u.ID)
		createSession(t, db, u.ID)

		require.NoError(t, repo.UpdatePassword(ctx, u.ID, "new-hash", keep))

		got, err := repo.GetByID(ctx, u.ID)
		require.NoError(t, err)
		assert.Equal(t, "new-hash", got.Password)
		assert.Equal(t, []string{keep.String()}, sessionIDs(t, db, u.ID))

		require.NoError(t, repo.UpdatePassword(ctx, u.ID, "third-hash", uuid.Nil))
		assert.Empty(t, sessionIDs(t, db, u.ID), "uuid.Nil отзывает всё")
	})

	t.Run("Patch_DeactivationRevokesSessions", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		admin := newUser(t, repo, "admin@example.com", user.RoleAdmin)
		member := newUser(t, repo, "member@example.com", user.RoleMember)
		createSession(t, db, member.ID)
		adminSession := createSession(t, db, admin.ID)

		require.NoError(t, repo.Patch(ctx, member.ID, nil, ptr(false)))

		got, err := repo.GetByID(ctx, member.ID)
		require.NoError(t, err)
		assert.False(t, got.IsActive)
		assert.Empty(t, sessionIDs(t, db, member.ID))
		assert.Equal(t, []string{adminSession.String()}, sessionIDs(t, db, admin.ID), "чужие сессии не трогаем")
	})

	t.Run("Patch_RoleOnlyKeepsSessions", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		newUser(t, repo, "admin@example.com", user.RoleAdmin)
		member := newUser(t, repo, "member@example.com", user.RoleMember)
		sess := createSession(t, db, member.ID)

		require.NoError(t, repo.Patch(ctx, member.ID, ptr(user.RoleAdmin), nil))

		got, err := repo.GetByID(ctx, member.ID)
		require.NoError(t, err)
		assert.Equal(t, user.RoleAdmin, got.Role)
		assert.True(t, got.IsActive)
		assert.Equal(t, []string{sess.String()}, sessionIDs(t, db, member.ID))
	})

	t.Run("Patch_BothFieldsOneWrite", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		newUser(t, repo, "admin@example.com", user.RoleAdmin)
		member := newUser(t, repo, "member@example.com", user.RoleMember)
		createSession(t, db, member.ID)

		require.NoError(t, repo.Patch(ctx, member.ID, ptr(user.RoleAdmin), ptr(false)))

		got, err := repo.GetByID(ctx, member.ID)
		require.NoError(t, err)
		assert.Equal(t, user.RoleAdmin, got.Role)
		assert.False(t, got.IsActive)
		assert.Empty(t, sessionIDs(t, db, member.ID))

		require.NoError(t, repo.Patch(ctx, member.ID, ptr(user.RoleAdmin), ptr(true)),
			"повышение неактивного до админа проходит")
	})

	t.Run("Patch_LastAdminKeepsSessions", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		admin := newUser(t, repo, "admin@example.com", user.RoleAdmin)
		sess := createSession(t, db, admin.ID)

		require.ErrorIs(t, repo.Patch(ctx, admin.ID, ptr(user.RoleMember), ptr(false)), user.ErrLastAdmin)

		got, err := repo.GetByID(ctx, admin.ID)
		require.NoError(t, err)
		assert.Equal(t, user.RoleAdmin, got.Role)
		assert.True(t, got.IsActive)
		assert.Equal(t, []string{sess.String()}, sessionIDs(t, db, admin.ID))
	})

	// Сбой DELETE из sessions откатывает и запись в users — иначе пароль сменился бы,
	// а чужие сессии остались живы.
	t.Run("SessionDeleteFailure_RollsBackUserWrite", func(t *testing.T) {
		db := container.GetTestDatabase(t)
		repo := userrepo.NewSQLiteRepository(db)
		_, err := helper.CreateTestFamily(ctx, "Test Family", "USD")
		require.NoError(t, err)

		newUser(t, repo, "admin@example.com", user.RoleAdmin)
		member := newUser(t, repo, "member@example.com", user.RoleMember)
		sess := createSession(t, db, member.ID)

		_, err = db.Exec(`CREATE TRIGGER no_session_delete BEFORE DELETE ON sessions
			BEGIN SELECT RAISE(ABORT, 'no'); END`)
		require.NoError(t, err)
		defer func() { _, _ = db.Exec(`DROP TRIGGER no_session_delete`) }()

		require.Error(t, repo.UpdatePassword(ctx, member.ID, "new-hash", uuid.Nil))
		require.Error(t, repo.Patch(ctx, member.ID, nil, ptr(false)))

		got, err := repo.GetByID(ctx, member.ID)
		require.NoError(t, err)
		assert.Equal(t, "old-hash", got.Password)
		assert.True(t, got.IsActive)
		assert.Equal(t, []string{sess.String()}, sessionIDs(t, db, member.ID))
	})
}
