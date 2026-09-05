package auth_test

import (
	"context"
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
type fakeSessions struct {
	byHash  map[string]*auth.Session
	owners  map[uuid.UUID]*user.User
	touches int
	deleted []uuid.UUID
	revoked []revokeCall
}

type revokeCall struct {
	userID, except uuid.UUID
}

func newFakeSessions() *fakeSessions {
	return &fakeSessions{byHash: map[string]*auth.Session{}, owners: map[uuid.UUID]*user.User{}}
}

func (f *fakeSessions) Create(_ context.Context, s *auth.Session) error {
	f.byHash[s.TokenHash] = s
	return nil
}

func (f *fakeSessions) FindByTokenHash(_ context.Context, hash string) (*auth.Session, *user.User, error) {
	s, ok := f.byHash[hash]
	if !ok {
		return nil, nil, auth.ErrSessionNotFound
	}
	return s, f.owners[s.UserID], nil
}

func (f *fakeSessions) Touch(_ context.Context, id uuid.UUID, at time.Time) error {
	f.touches++
	for _, s := range f.byHash {
		if s.ID == id {
			s.LastUsedAt = at
			s.ExpiresAt = s.ExpiryAfter(at)
		}
	}
	return nil
}

func (f *fakeSessions) Delete(_ context.Context, id uuid.UUID) error {
	f.deleted = append(f.deleted, id)
	for hash, s := range f.byHash {
		if s.ID == id {
			delete(f.byHash, hash)
			return nil
		}
	}
	return auth.ErrSessionNotFound
}

func (f *fakeSessions) DeleteByUser(_ context.Context, userID, exceptID uuid.UUID) error {
	f.revoked = append(f.revoked, revokeCall{userID: userID, except: exceptID})
	for hash, s := range f.byHash {
		if s.UserID == userID && s.ID != exceptID {
			delete(f.byHash, hash)
		}
	}
	return nil
}

func (f *fakeSessions) ListByUser(_ context.Context, userID uuid.UUID) ([]*auth.Session, error) {
	var out []*auth.Session
	for _, s := range f.byHash {
		if s.UserID == userID {
			out = append(out, s)
		}
	}
	return out, nil
}

func (f *fakeSessions) DeleteExpired(context.Context, time.Time) error { return nil }

type fakeUsers struct {
	users   map[uuid.UUID]*user.User
	updates map[uuid.UUID]string
}

func (f *fakeUsers) GetByEmail(_ context.Context, email string) (*user.User, error) {
	for _, u := range f.users {
		if u.Email == email {
			return u, nil
		}
	}
	return nil, user.ErrNotFound
}

func (f *fakeUsers) GetByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	u, ok := f.users[id]
	if !ok {
		return nil, user.ErrNotFound
	}
	return u, nil
}

func (f *fakeUsers) UpdatePassword(_ context.Context, id uuid.UUID, hash string) error {
	if _, ok := f.users[id]; !ok {
		return user.ErrNotFound
	}
	f.updates[id] = hash
	f.users[id].Password = hash
	return nil
}

type fakeSetup struct{ exists bool }

func (f fakeSetup) Exists(context.Context) (bool, error) { return f.exists, nil }

type fixture struct {
	svc      *auth.Service
	sessions *fakeSessions
	users    *fakeUsers
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
	f := &fixture{sessions: sessions, users: users, user: u, now: now}
	f.svc, err = auth.NewService(sessions, users, fakeSetup{exists: setupDone}, auth.WithClock(func() time.Time {
		return f.now
	}))
	require.NoError(t, err)
	return f
}

func (f *fixture) login(t *testing.T) string {
	t.Helper()
	token, u, err := f.svc.Login(context.Background(), f.user.Email, testPassword, testDevice)
	require.NoError(t, err)
	require.Equal(t, f.user.ID, u.ID)
	return token
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
}

func TestService_Login_WrongPassword(t *testing.T) {
	f := newFixture(t, true)

	token, u, err := f.svc.Login(context.Background(), f.user.Email, "wrong-password", testDevice)

	require.ErrorIs(t, err, auth.ErrInvalidCredentials)
	assert.Empty(t, token)
	assert.Nil(t, u)
	assert.Empty(t, f.sessions.byHash)
}

func TestService_Login_UnknownEmail_SameError(t *testing.T) {
	f := newFixture(t, true)

	_, _, errUnknown := f.svc.Login(context.Background(), "nobody@example.com", testPassword, testDevice)
	_, _, errWrong := f.svc.Login(context.Background(), f.user.Email, "wrong-password", testDevice)

	require.ErrorIs(t, errUnknown, auth.ErrInvalidCredentials)
	assert.Equal(t, errWrong, errUnknown)
	assert.Empty(t, f.sessions.byHash)
}

func TestService_Login_SetupRequired(t *testing.T) {
	f := newFixture(t, false)

	_, _, err := f.svc.Login(context.Background(), f.user.Email, testPassword, testDevice)

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

	_, _, err = f.svc.Login(context.Background(), f.user.Email, testPassword, testDevice)
	require.ErrorIs(t, err, auth.ErrInvalidCredentials, "old password no longer works")
	_, _, err = f.svc.Login(context.Background(), f.user.Email, "new-password-1", testDevice)
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
