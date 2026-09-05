package testhelpers

import (
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/gorilla/sessions"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application"
	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/domain/user"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	reportrepo "family-budget-service/internal/infrastructure/report"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	userrepo "family-budget-service/internal/infrastructure/user"
	"family-budget-service/internal/services"
	"family-budget-service/internal/web/middleware"
)

const (
	// TestSessionSecret — секрет подписи сессий тестового сервера.
	// LoginAs использует его же, чтобы собрать валидную cookie.
	TestSessionSecret = "test-session-secret-for-integration-tests"

	// testCSRFTokenLength — длина случайного CSRF-токена в байтах.
	testCSRFTokenLength = 32
)

// TestServer represents a test HTTP server setup
type TestServer struct {
	Repos     *handlers.Repositories
	Services  *services.Services
	Server    *application.HTTPServer
	Container *SQLiteTestDB

	// AuthFamily/AuthUser заполняются при первом вызове Auth
	AuthFamily *user.Family
	AuthUser   *user.User

	authSession *AuthSession
}

// SetupHTTPServer creates a test HTTP server with real database connections
func SetupHTTPServer(t *testing.T) *TestServer {
	// Setup SQLite in-memory database
	container := SetupSQLiteTestDB(t)

	// Get test database
	db := container.DB

	// Create repositories
	repos := &handlers.Repositories{
		User:        userrepo.NewSQLiteRepository(db),
		Family:      userrepo.NewSQLiteFamilyRepository(db),
		Budget:      budgetrepo.NewSQLiteRepository(db),
		Category:    categoryrepo.NewSQLiteRepository(db),
		Transaction: transactionrepo.NewSQLiteRepository(db),
		Report:      reportrepo.NewSQLiteRepository(db),
		Invite:      userrepo.NewInviteSQLiteRepository(db),
	}

	// Create BackupService for testing with in-memory database.
	// Каталог бэкапов — временный: иначе сервис пишет ./backups в каталог пакета.
	backupService := services.NewBackupService(db, ":memory:", t.TempDir(), slog.Default())

	// Create services for testing - use simplified version to avoid circular dependencies
	servicesContainer := services.NewServices(
		repos.User,        // userRepo
		repos.Family,      // familyRepo
		repos.Category,    // categoryRepo
		repos.Transaction, // transactionRepo
		repos.Budget,      // budgetRepo for transactions
		repos.Budget,      // fullBudgetRepo
		repos.Report,      // reportRepo
		repos.Invite,      // inviteRepo
		backupService,     // backupService
		slog.Default(),    // logger
	)

	// Create HTTP server configuration for testing.
	// Путь к шаблонам резолвится от корня репозитория, а не от cwd теста, иначе
	// веб-слой (сессии, CSRF, HTML-маршруты) не поднимется.
	config := &application.Config{
		Port:          "8080",
		Host:          "localhost",
		SessionSecret: TestSessionSecret,
		CookieSecure:  false,
		TemplatesDir:  filepath.Join(RepoRoot(t), "internal", "web", "templates"),
		StaticDir:     filepath.Join(RepoRoot(t), "internal", "web", "static"),
	}

	// Create HTTP server without observability for testing
	httpServer := application.NewHTTPServer(repos, servicesContainer, config)
	if err := httpServer.WebServerInitError(); err != nil {
		t.Fatalf("web server initialization failed, test server has no session/CSRF/web routes: %v", err)
	}

	testServer := &TestServer{
		Repos:     repos,
		Services:  servicesContainer,
		Server:    httpServer,
		Container: container,
	}

	// Явного освобождения ресурсов не требуется: БД in-memory закрывается своим
	// t.Cleanup из SetupSQLiteTestDB.
	return testServer
}

// RepoRoot возвращает абсолютный путь к корню репозитория.
// Нужен потому, что шаблоны, статика и миграции резолвятся относительно cwd,
// а cwd у `go test` — каталог тестируемого пакета.
func RepoRoot(t *testing.T) string {
	t.Helper()

	_, thisFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("runtime.Caller failed, cannot resolve repository root")
	}

	dir := filepath.Dir(thisFile)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("go.mod not found above %s", filepath.Dir(thisFile))
			return ""
		}
		dir = parent
	}
}

// AuthSession — аутентифицированная сессия для интеграционных тестов:
// подписанная cookie сессии плюс CSRF-токен, лежащий в той же сессии.
type AuthSession struct {
	Cookie    *http.Cookie
	CSRFToken string
}

// Apply добавляет к запросу cookie сессии и CSRF-токен в заголовке.
func (s *AuthSession) Apply(req *http.Request) {
	req.AddCookie(s.Cookie)
	req.Header.Set(middleware.CSRFHeaderKey, s.CSRFToken)
}

// LoginAs собирает валидную сессию для указанного пользователя без прохода
// через форму входа: cookie подписывается тем же секретом, что и у тестового
// сервера, поэтому SessionStore её принимает.
func LoginAs(t *testing.T, u *user.User) *AuthSession {
	t.Helper()

	store := sessions.NewCookieStore([]byte(TestSessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   int(middleware.SessionTimeout.Seconds()),
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
	}

	req := httptest.NewRequestWithContext(t.Context(), http.MethodGet, "/", nil)
	sess, err := store.New(req, middleware.SessionName)
	require.NoError(t, err)

	csrfToken := randomCSRFToken(t)
	sess.Values[middleware.SessionUserKey] = u.ID
	sess.Values[middleware.SessionRoleKey] = u.Role
	sess.Values[middleware.SessionEmailKey] = u.Email
	sess.Values[middleware.CSRFTokenKey] = csrfToken

	rec := httptest.NewRecorder()
	require.NoError(t, sess.Save(req, rec))

	cookies := (&http.Response{Header: rec.Header()}).Cookies()
	require.NotEmpty(t, cookies, "session cookie was not written")

	return &AuthSession{Cookie: cookies[0], CSRFToken: csrfToken}
}

// randomCSRFToken генерирует токен в том же формате, что и middleware.
func randomCSRFToken(t *testing.T) string {
	t.Helper()

	buf := make([]byte, testCSRFTokenLength)
	_, err := rand.Read(buf)
	require.NoError(t, err)

	return base64.URLEncoding.EncodeToString(buf)
}

// Auth возвращает сессию администратора тестовой семьи, создавая семью и
// пользователя в БД при первом обращении. Нужна интеграционным тестам, потому
// что тестовый сервер несёт полный веб-слой: RequireSetup требует существующей
// семьи, а CSRFProtection — токена на каждый запрос записи.
func (ts *TestServer) Auth(t *testing.T) *AuthSession {
	t.Helper()

	if ts.authSession != nil {
		return ts.authSession
	}

	family := ts.ensureFamily(t)

	admin := CreateTestUser(family.ID)
	admin.Role = user.RoleAdmin
	require.NoError(t, ts.Repos.User.Create(t.Context(), admin))

	ts.AuthFamily = family
	ts.AuthUser = admin
	ts.authSession = LoginAs(t, admin)

	return ts.authSession
}

// AuthAs создаёт нового пользователя с указанной ролью в той же тестовой семье
// и возвращает его вместе с сессией. Вторую семью заводить нельзя (см.
// ensureFamily), поэтому ролевые тесты добавляют пользователей в существующую.
func (ts *TestServer) AuthAs(t *testing.T, role user.Role) (*user.User, *AuthSession) {
	t.Helper()

	family := ts.ensureFamily(t)
	if ts.AuthFamily == nil {
		ts.AuthFamily = family
	}

	member := CreateTestUser(family.ID)
	member.Role = role
	require.NoError(t, ts.Repos.User.Create(t.Context(), member))

	return member, LoginAs(t, member)
}

// ensureFamily возвращает уже существующую семью или создаёт новую.
// Вторую семью заводить нельзя: SQLite-репозитории следуют однофамильной модели
// и берут family_id как `SELECT id FROM families LIMIT 1`.
func (ts *TestServer) ensureFamily(t *testing.T) *user.Family {
	t.Helper()

	var idStr string
	err := ts.Container.DB.QueryRowContext(t.Context(), "SELECT id FROM families LIMIT 1").Scan(&idStr)
	if errors.Is(err, sql.ErrNoRows) {
		family := CreateTestFamily()
		require.NoError(t, ts.Repos.Family.Create(t.Context(), family))
		return family
	}
	require.NoError(t, err)

	// Возвращаем запись целиком: тестам нужны Name/Currency, а не только ID.
	family, getErr := ts.Repos.Family.Get(t.Context())
	require.NoError(t, getErr)
	require.NotNil(t, family)

	return family
}

// CheckTableExists checks if a table exists in the database (for debugging)
func (ts *TestServer) CheckTableExists(t *testing.T, tableName string) bool {
	var exists int
	err := ts.Container.DB.QueryRowContext(
		t.Context(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Logf("Error checking table existence: %v", err)
		return false
	}
	return exists > 0
}
