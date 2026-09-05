package auth_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"

	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
)

const (
	testPassword = "correct-horse-battery"
	testDevice   = "Pixel 8"
)

// fakeSessions — in-memory SessionRepository, считает обращения, которые пишут в БД.
// fail[method] — ошибка, которую вернёт метод вместо работы (имитация сбоя хранилища).
type fakeSessions struct {
	byHash  map[string]*auth.Session
	owners  map[uuid.UUID]*user.User
	touches int
	deleted []uuid.UUID
	revoked []revokeCall
	expired []time.Time
	fail    map[string]error
}

type revokeCall struct {
	userID, except uuid.UUID
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{
		byHash: map[string]*auth.Session{},
		owners: map[uuid.UUID]*user.User{},
		fail:   map[string]error{},
	}
}

func (f *fakeSessions) Create(_ context.Context, s *auth.Session) error {
	if err := f.fail["Create"]; err != nil {
		return err
	}
	f.byHash[s.TokenHash] = s
	return nil
}

func (f *fakeSessions) FindByTokenHash(_ context.Context, hash string) (*auth.Session, *user.User, error) {
	if err := f.fail["FindByTokenHash"]; err != nil {
		return nil, nil, err
	}
	s, ok := f.byHash[hash]
	if !ok {
		return nil, nil, auth.ErrSessionNotFound
	}
	return s, f.owners[s.UserID], nil
}

func (f *fakeSessions) Touch(_ context.Context, id uuid.UUID, lastUsedAt, expiresAt time.Time) error {
	if err := f.fail["Touch"]; err != nil {
		return err
	}
	f.touches++
	for _, s := range f.byHash {
		if s.ID == id {
			s.LastUsedAt = lastUsedAt
			s.ExpiresAt = expiresAt
			return nil
		}
	}
	return auth.ErrSessionNotFound
}

func (f *fakeSessions) Delete(_ context.Context, id uuid.UUID) error {
	if err := f.fail["Delete"]; err != nil {
		return err
	}
	f.deleted = append(f.deleted, id)
	for hash, s := range f.byHash {
		if s.ID == id {
			delete(f.byHash, hash)
			return nil
		}
	}
	return auth.ErrSessionNotFound
}

func (f *fakeSessions) DeleteOwned(_ context.Context, userID, id uuid.UUID) error {
	if err := f.fail["DeleteOwned"]; err != nil {
		return err
	}
	for hash, s := range f.byHash {
		if s.ID == id && s.UserID == userID {
			delete(f.byHash, hash)
			return nil
		}
	}
	return auth.ErrSessionNotFound
}

func (f *fakeSessions) DeleteByUser(_ context.Context, userID, exceptID uuid.UUID) error {
	if err := f.fail["DeleteByUser"]; err != nil {
		return err
	}
	f.revoked = append(f.revoked, revokeCall{userID: userID, except: exceptID})
	for hash, s := range f.byHash {
		if s.UserID == userID && s.ID != exceptID {
			delete(f.byHash, hash)
		}
	}
	return nil
}

func (f *fakeSessions) ListByUser(_ context.Context, userID uuid.UUID) ([]*auth.Session, error) {
	if err := f.fail["ListByUser"]; err != nil {
		return nil, err
	}
	var out []*auth.Session
	for _, s := range f.byHash {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSessions) DeleteExpired(_ context.Context, now time.Time) error {
	if err := f.fail["DeleteExpired"]; err != nil {
		return err
	}
	f.expired = append(f.expired, now)
	for hash, s := range f.byHash {
		if s.ExpiresAt.Before(now) {
			delete(f.byHash, hash)
		}
	}
	return nil
}

type fakeUsers struct {
	users   map[uuid.UUID]*user.User
	updates map[uuid.UUID]string
	failErr error
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (*user.User, error) {
	if f.failErr != nil {
		return nil, f.failErr
	}
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func (f *fakeUsers) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	if f.failErr != nil {
		return nil, f.failErr
	}
	u, ok := f.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if f.failErr != nil {
		return f.failErr
	}
	if _, ok := f.users[id]; !ok {
		return user.ErrNotFound
	}
	f.updates[id] = hash
	f.users[id].Password = hash
	return nil
}

type fakeSetup struct {
	exists  bool
	failErr error
}

func (f *fakeSetup) Exists(context.Context) (bool, error) { return f.exists, f.failErr }

type fixture struct {
	svc      *auth.Service
	sessions *fakeSessions
	users    *fakeUsers
	setup    *fakeSetup
	user     *user.User
	now      time.Time
}

func newFixture(t *testing.T, setupDone bool) *fixture {
	t.Helper()

	// MinCost: в тестах важна проверка, а не стоимость хеша.
	hash, err := bcrypt.GenerateFromPassword([]byte(testPassword), bcrypt.MinCost)
	require.NoError(t, err)

	u := user.NewUser("anna@example.com", "Anna", "Ivanova", user.RoleMember)
	u.Password = string(hash)

	sessions := newFakeSessions()
	sessions.owners[u.ID] = u
	users := &fakeUsers{users: map[uuid.UUID]*user.User{u.ID: u}, updates: map[uuid.UUID]string{}}

	now := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	setup := &fakeSetup{exists: setupDone}
	f := &fixture{sessions: sessions, users: users, setup: setup, user: u, now: now}
	f.svc = auth.NewService(sessions, users, setup)
	f.svc.SetClock(func() time.Time { return f.now })
	return f
}

func (f *fixture) login(t *testing.T) string {
	t.Helper()
	res, err := f.svc.Login(context.Background(), f.user.Email, testPassword, testDevice)
	require.NoError(t, err)
	require.Equal(t, f.user.ID, res.User.ID)
	require.Equal(t, auth.HashToken(res.Token), res.Session.TokenHash)
	return res.Token
}

func TestService_Login_Success(t *testing.T) {
	f := newFixture(t, true)

	token := f.login(t)

	require.NotEmpty(t, token)
	s, ok := f.sessions.byHash[auth.HashToken(token)]
	require.True(t, ok, "session stored under hash of the token")
	assert.Equal(t, f.user.ID, s.UserID)
	assert.Equal(t, testDevice, s.DeviceName)
	assert.True(t, s.CreatedAt.Equal(f.now))
	assert.True(t, s.ExpiresAt.Equal(f.now.Add(auth.IdleTTL)))
	assert.Equal(t, []time.Time{f.now}, f.sessions.expired, "логин чистит истёкшие сессии")
}

// Истёкшая сессия, которой не пользовались, исчезает при следующем логине любого пользователя.
func TestService_Login_DeletesExpiredSessions(t *testing.T) {
	f := newFixture(t, true)
	stale := f.login(t)
	f.now = f.now.Add(auth.IdleTTL + time.Minute)

	f.login(t)

	_, ok := f.sessions.byHash[auth.HashToken(stale)]
	assert.False(t, ok)
	assert.Len(t, f.sessions.byHash, 1)
}

func TestService_Login_WrongPassword(t *testing.T) {
	f := newFixture(t, true)

	res, err := f.svc.Login(context.Background(), f.user.Email, "wrong-password", testDevice)

	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	assert.Nil(t, res)
	assert.Empty(t, f.sessions.byHash)
}

func TestService_Login_UnknownEmail_SameError(t *testing.T) {
	f := newFixture(t, true)

	_, errUnknown := f.svc.Login(context.Background(), "nobody@example.com", testPassword, testDevice)
	_, errWrong := f.svc.Login(context.Background(), f.user.Email, "wrong-password", testDevice)

	require.ErrorIs(t, errUnknown, auth.ErrInvalidCredentials)
	assert.Equal(t, errWrong, errUnknown)
	assert.Empty(t, f.sessions.byHash)
}

// Деактивированный пользователь получает тот же ответ, что и при неверном пароле.
func TestService_Login_InactiveUser(t *testing.T) {
	f := newFixture(t, true)
	f.user.IsActive = false

	res, err := f.svc.Login(context.Background(), f.user.Email, testPassword, testDevice)

	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	assert.Nil(t, res)
	assert.Empty(t, f.sessions.byHash)
}

// Деактивация после логина: сессия ещё лежит в БД, но токен уже не принимается.
func TestService_Authenticate_InactiveUser(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	f.user.IsActive = false

	p, err := f.svc.Authenticate(context.Background(), token)

	require.ErrorIs(t, err, auth.ErrUnauthorized)
	assert.Nil(t, p)
}

func TestService_RevokeAllSessions(t *testing.T) {
	f := newFixture(t, true)
	first := f.login(t)
	second := f.login(t)

	require.NoError(t, f.svc.RevokeAllSessions(context.Background(), f.user.ID))

	_, err := f.svc.Authenticate(context.Background(), first)
	require.ErrorIs(t, err, auth.ErrUnauthorized)
	_, err = f.svc.Authenticate(context.Background(), second)
	require.ErrorIs(t, err, auth.ErrUnauthorized)
}

func TestService_Login_SetupRequired(t *testing.T) {
	f := newFixture(t, false)

	_, err := f.svc.Login(context.Background(), f.user.Email, testPassword, testDevice)

	require.ErrorIs(t, err, auth.ErrSetupRequired)
}

func TestService_Authenticate_UnknownToken(t *testing.T) {
	f := newFixture(t, true)

	p, err := f.svc.Authenticate(context.Background(), "garbage")

	require.ErrorIs(t, err, auth.ErrUnauthorized)
	assert.Nil(t, p)
}

func TestService_Authenticate_Success_NoTouchWithinInterval(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	f.now = f.now.Add(auth.TouchInterval)

	p, err := f.svc.Authenticate(context.Background(), token)

	require.NoError(t, err)
	assert.Equal(t, f.user.ID, p.UserID)
	assert.Equal(t, f.user.Email, p.Email)
	assert.Equal(t, user.RoleMember, p.Role)
	assert.Equal(t, f.sessions.byHash[auth.HashToken(token)].ID, p.SessionID)
	assert.Equal(t, 0, f.sessions.touches, "last_used_at is not written before TouchInterval passes")
}

func TestService_Authenticate_TouchAfterInterval(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	f.now = f.now.Add(auth.TouchInterval + time.Second)

	_, err := f.svc.Authenticate(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, 1, f.sessions.touches)

	s := f.sessions.byHash[auth.HashToken(token)]
	assert.True(t, s.LastUsedAt.Equal(f.now))
	assert.True(t, s.ExpiresAt.Equal(f.now.Add(auth.IdleTTL)))

	_, err = f.svc.Authenticate(context.Background(), token)
	require.NoError(t, err)
	assert.Equal(t, 1, f.sessions.touches, "second request right after touch does not write again")
}

// Отзыв между FindByTokenHash и Touch — 401, а не 500: сессии уже нет.
func TestService_Authenticate_RevokedBeforeTouch(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	f.now = f.now.Add(auth.TouchInterval + time.Second)
	f.sessions.fail["Touch"] = auth.ErrSessionNotFound

	_, err := f.svc.Authenticate(context.Background(), token)

	require.ErrorIs(t, err, auth.ErrUnauthorized)
}

func TestService_Authenticate_Expired(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	sessionID := f.sessions.byHash[auth.HashToken(token)].ID
	f.now = f.now.Add(auth.IdleTTL)

	p, err := f.svc.Authenticate(context.Background(), token)

	require.ErrorIs(t, err, auth.ErrUnauthorized)
	assert.Nil(t, p)
	assert.Equal(t, []uuid.UUID{sessionID}, f.sessions.deleted)
	assert.Empty(t, f.sessions.byHash)
}

func TestService_Logout(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	p, err := f.svc.Authenticate(context.Background(), token)
	require.NoError(t, err)

	require.NoError(t, f.svc.Logout(context.Background(), p.SessionID))

	_, err = f.svc.Authenticate(context.Background(), token)
	require.ErrorIs(t, err, auth.ErrUnauthorized)
	require.ErrorIs(t, f.svc.Logout(context.Background(), p.SessionID), auth.ErrSessionNotFound)
}

func TestService_RevokeSession(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	sessionID := f.sessions.byHash[auth.HashToken(token)].ID

	err := f.svc.RevokeSession(context.Background(), uuid.New(), sessionID)
	require.ErrorIs(t, err, auth.ErrSessionNotFound, "someone else's session is invisible")
	assert.Len(t, f.sessions.byHash, 1)

	require.NoError(t, f.svc.RevokeSession(context.Background(), f.user.ID, sessionID))
	assert.Empty(t, f.sessions.byHash)

	list, err := f.svc.ListSessions(context.Background(), f.user.ID)
	require.NoError(t, err)
	assert.Empty(t, list)
}

// GET /auth/sessions не показывает истёкшие сессии, даже если строки ещё лежат в БД.
func TestService_ListSessions_HidesExpired(t *testing.T) {
	f := newFixture(t, true)
	f.login(t)
	f.now = f.now.Add(auth.IdleTTL - time.Hour)
	fresh := f.login(t)
	f.now = f.now.Add(2 * time.Hour)

	list, err := f.svc.ListSessions(context.Background(), f.user.ID)

	require.NoError(t, err)
	require.Len(t, list, 1)
	assert.Equal(t, auth.HashToken(fresh), list[0].TokenHash)
	assert.Len(t, f.sessions.byHash, 2, "строка истёкшей сессии лежит до следующего логина")
}

// Сбой хранилища не схлопывается в ErrUnauthorized/ErrInvalidCredentials: клиент не должен
// выбрасывать рабочий токен, а лимитер — считать сбой неверным паролем.
func TestService_StorageFailure_IsNotAuthError(t *testing.T) {
	errDB := errors.New("database is locked")
	ctx := context.Background()

	cases := map[string]func(t *testing.T, f *fixture) error{
		"Login: setup check": func(_ *testing.T, f *fixture) error {
			f.setup.failErr = errDB
			_, err := f.svc.Login(ctx, f.user.Email, testPassword, testDevice)
			return err
		},
		"Login: GetByEmail": func(_ *testing.T, f *fixture) error {
			f.users.failErr = errDB
			_, err := f.svc.Login(ctx, f.user.Email, testPassword, testDevice)
			return err
		},
		"Login: DeleteExpired": func(_ *testing.T, f *fixture) error {
			f.sessions.fail["DeleteExpired"] = errDB
			_, err := f.svc.Login(ctx, f.user.Email, testPassword, testDevice)
			return err
		},
		"Login: Create": func(_ *testing.T, f *fixture) error {
			f.sessions.fail["Create"] = errDB
			_, err := f.svc.Login(ctx, f.user.Email, testPassword, testDevice)
			return err
		},
		"Authenticate: FindByTokenHash": func(t *testing.T, f *fixture) error {
			token := f.login(t)
			f.sessions.fail["FindByTokenHash"] = errDB
			_, err := f.svc.Authenticate(ctx, token)
			return err
		},
		"Authenticate: Delete of expired": func(t *testing.T, f *fixture) error {
			token := f.login(t)
			f.now = f.now.Add(auth.IdleTTL)
			f.sessions.fail["Delete"] = errDB
			_, err := f.svc.Authenticate(ctx, token)
			return err
		},
		"Authenticate: Touch": func(t *testing.T, f *fixture) error {
			token := f.login(t)
			f.now = f.now.Add(auth.TouchInterval + time.Second)
			f.sessions.fail["Touch"] = errDB
			_, err := f.svc.Authenticate(ctx, token)
			return err
		},
		"ListSessions": func(_ *testing.T, f *fixture) error {
			f.sessions.fail["ListByUser"] = errDB
			_, err := f.svc.ListSessions(ctx, f.user.ID)
			return err
		},
		"RevokeSession": func(_ *testing.T, f *fixture) error {
			f.sessions.fail["DeleteOwned"] = errDB
			return f.svc.RevokeSession(ctx, f.user.ID, uuid.New())
		},
		"ChangePassword: GetByID": func(_ *testing.T, f *fixture) error {
			f.users.failErr = errDB
			return f.svc.ChangePassword(ctx, f.user.ID, testPassword, "new-password-1", uuid.Nil)
		},
		"AdminSetPassword: UpdatePassword": func(_ *testing.T, f *fixture) error {
			f.users.failErr = errDB
			return f.svc.AdminSetPassword(ctx, f.user.ID, "new-password-1")
		},
		"AdminSetPassword: DeleteByUser": func(_ *testing.T, f *fixture) error {
			f.sessions.fail["DeleteByUser"] = errDB
			return f.svc.AdminSetPassword(ctx, f.user.ID, "new-password-1")
		},
	}
	for name, run := range cases {
		t.Run(name, func(t *testing.T) {
			f := newFixture(t, true)

			err := run(t, f)

			require.ErrorIs(t, err, errDB)
			require.NotErrorIs(t, err, auth.ErrUnauthorized)
			require.NotErrorIs(t, err, auth.ErrInvalidCredentials)
			assert.NotErrorIs(t, err, auth.ErrSessionNotFound)
		})
	}
}

// Истёкшая сессия, уже удалённая кем-то другим, — всё равно 401, а не 500.
func TestService_Authenticate_ExpiredAlreadyDeleted(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	f.now = f.now.Add(auth.IdleTTL)
	f.sessions.fail["Delete"] = auth.ErrSessionNotFound

	_, err := f.svc.Authenticate(context.Background(), token)

	require.ErrorIs(t, err, auth.ErrUnauthorized)
}

func TestService_ChangePassword_WrongCurrent(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)
	f.login(t)

	err := f.svc.ChangePassword(context.Background(), f.user.ID, "wrong-password", "new-password-1", uuid.Nil)

	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	assert.Empty(t, f.users.updates)
	assert.Empty(t, f.sessions.revoked)
	assert.Len(t, f.sessions.byHash, 2, "sessions survive a failed change")
	_, err = f.svc.Authenticate(context.Background(), token)
	require.NoError(t, err)
}

func TestService_ChangePassword_WeakNew(t *testing.T) {
	f := newFixture(t, true)

	err := f.svc.ChangePassword(context.Background(), f.user.ID, testPassword, "short", uuid.Nil)

	require.ErrorIs(t, err, auth.ErrInvalidPassword)
	assert.Empty(t, f.users.updates)
}

func TestService_ChangePassword_Success_KeepsCurrentSession(t *testing.T) {
	f := newFixture(t, true)
	keepToken := f.login(t)
	otherToken := f.login(t)
	keepID := f.sessions.byHash[auth.HashToken(keepToken)].ID

	err := f.svc.ChangePassword(context.Background(), f.user.ID, testPassword, "new-password-1", keepID)
	require.NoError(t, err)

	assert.True(t, auth.ComparePassword(f.users.updates[f.user.ID], "new-password-1"))
	assert.Equal(t, []revokeCall{{userID: f.user.ID, except: keepID}}, f.sessions.revoked)

	_, err = f.svc.Authenticate(context.Background(), keepToken)
	require.NoError(t, err)
	_, err = f.svc.Authenticate(context.Background(), otherToken)
	require.ErrorIs(t, err, auth.ErrUnauthorized)

	_, err = f.svc.Login(context.Background(), f.user.Email, testPassword, testDevice)
	require.ErrorIs(t, err, auth.ErrInvalidCredentials, "old password no longer works")
	_, err = f.svc.Login(context.Background(), f.user.Email, "new-password-1", testDevice)
	require.NoError(t, err)
}

func TestService_AdminSetPassword_RevokesAll(t *testing.T) {
	f := newFixture(t, true)
	token := f.login(t)

	require.NoError(t, f.svc.AdminSetPassword(context.Background(), f.user.ID, "reset-password-1"))

	assert.Equal(t, []revokeCall{{userID: f.user.ID, except: uuid.Nil}}, f.sessions.revoked)
	_, err := f.svc.Authenticate(context.Background(), token)
	require.ErrorIs(t, err, auth.ErrUnauthorized)

	err = f.svc.AdminSetPassword(context.Background(), uuid.New(), "reset-password-1")
	require.Error(t, err)
	assert.ErrorIs(t, err, user.ErrNotFound)
}
