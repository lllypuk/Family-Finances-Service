package testhelpers

import (
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/require"

	"family-budget-service/internal/application"
	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	accountrepo "family-budget-service/internal/infrastructure/account"
	authrepo "family-budget-service/internal/infrastructure/auth"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	reconciliationrepo "family-budget-service/internal/infrastructure/reconciliation"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	userrepo "family-budget-service/internal/infrastructure/user"
	"family-budget-service/internal/metrics"
	"family-budget-service/internal/services"
	"family-budget-service/internal/version"
)

// testDeviceName — device_name сессий, выданных LoginAs.
const testDeviceName = "integration-test"

// TestServer represents a test HTTP server setup
type TestServer struct {
	Repos     *handlers.Repositories
	Services  *services.Services
	Server    *application.HTTPServer
	Container *SQLiteTestDB
	// Metrics — тот же реестр, что считает запросы сервера.
	Metrics *metrics.Metrics

	// AuthFamily/AuthUser заполняются при первом вызове Auth
	AuthFamily *user.Family
	AuthUser   *user.User

	authSession *AuthSession
}

// serverParams — то, что опции меняют до сборки сервисов и сервера.
type serverParams struct {
	config     *application.Config
	recognizer services.Recognizer
}

// ServerOption настраивает параметры тестового сервера; применяется до сборки сервисов.
type ServerOption func(*serverParams)

// WithTrustedProxies — CIDR, чьи X-Forwarded-For сервер принимает за адрес клиента.
func WithTrustedProxies(t *testing.T, cidrs string) ServerOption {
	t.Helper()
	ranges, err := auth.ParseTrustedProxies(cidrs)
	require.NoError(t, err)
	return func(p *serverParams) { p.config.TrustedProxies = ranges }
}

// WithRecognizer подставляет движок распознавания; без опции плечо выключено.
func WithRecognizer(r services.Recognizer) ServerOption {
	return func(p *serverParams) { p.recognizer = r }
}

// WithRecognizeUploadTimeout сокращает срок приёма тела распознавания — для тестов сетевых дедлайнов.
func WithRecognizeUploadTimeout(d time.Duration) ServerOption {
	return func(p *serverParams) { p.config.RecognizeUploadTimeout = d }
}

// SetupHTTPServer creates a test HTTP server with real database connections
func SetupHTTPServer(t *testing.T, opts ...ServerOption) *TestServer {
	// Setup SQLite in-memory database
	container := SetupSQLiteTestDB(t)

	// Get test database
	db := container.DB

	// Create repositories
	userRepo := userrepo.NewSQLiteRepository(db)
	categoryRepo := categoryrepo.NewSQLiteRepository(db)
	repos := &handlers.Repositories{
		User:           userRepo,
		Family:         userrepo.NewSQLiteFamilyRepository(db, categoryRepo, userRepo),
		Budget:         budgetrepo.NewSQLiteRepository(db),
		Category:       categoryRepo,
		Account:        accountrepo.NewSQLiteRepository(db),
		Reconciliation: reconciliationrepo.NewSQLiteRepository(db),
		Transaction:    transactionrepo.NewSQLiteRepository(db),
		Session:        authrepo.NewSessionSQLiteRepository(db),
	}

	authService := auth.NewService(repos.Session, repos.User, repos.Family)

	registry := metrics.New(version.String(), slog.Default())

	// Create BackupService for testing with in-memory database.
	// Каталог бэкапов — временный: иначе сервис пишет ./backups в каталог пакета.
	backupService := services.NewBackupService(
		db,
		":memory:",
		t.TempDir(),
		services.DefaultBackupKeep,
		slog.Default(),
		registry.Backup(),
	)

	params := &serverParams{config: &application.Config{
		Port:    "8080",
		Host:    "localhost",
		Metrics: registry,
	}}
	for _, opt := range opts {
		opt(params)
	}

	// Create services for testing - use simplified version to avoid circular dependencies
	servicesContainer := services.NewServices(
		repos.User,           // userRepo
		repos.Family,         // familyRepo
		repos.Category,       // categoryRepo
		repos.Account,        // accountRepo
		repos.Reconciliation, // reconciliationRepo
		repos.Transaction,    // transactionRepo
		repos.Budget,         // budgetRepo for transactions
		repos.Budget,         // fullBudgetRepo
		backupService,        // backupService
		authService,          // authService
		params.recognizer,
		registry.Recognize(),
		slog.Default(), // logger
	)

	// Create HTTP server without observability for testing
	httpServer := application.NewHTTPServer(repos, servicesContainer, params.config)

	testServer := &TestServer{
		Repos:     repos,
		Services:  servicesContainer,
		Server:    httpServer,
		Container: container,
		Metrics:   registry,
	}

	// Явного освобождения ресурсов не требуется: БД in-memory закрывается своим
	// t.Cleanup из SetupSQLiteTestDB.
	return testServer
}

// RepoRoot возвращает абсолютный путь к корню репозитория.
// Нужен потому, что миграции резолвятся относительно cwd,
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

// AuthSession — bearer-токен для запросов интеграционных тестов.
type AuthSession struct {
	Token string
}

// Apply ставит заголовок Authorization: Bearer.
func (s *AuthSession) Apply(req *http.Request) {
	req.Header.Set(echo.HeaderAuthorization, "Bearer "+s.Token)
}

// LoginAs выдаёт токен указанному пользователю, минуя проверку пароля:
// у пользователей из фабрик вместо хеша заглушка, через Service.Login не пройти.
// Сессия пишется в БД тестового сервера, поэтому RequireBearer принимает её как настоящую.
func LoginAs(t *testing.T, ts *TestServer, u *user.User) *AuthSession {
	t.Helper()

	plain, hash := auth.GenerateToken()
	require.NoError(t, ts.Repos.Session.Create(t.Context(), auth.NewSession(u.ID, hash, testDeviceName, time.Now())))

	return &AuthSession{Token: plain}
}

// Auth возвращает сессию администратора тестовой семьи, создавая семью и
// пользователя в БД при первом обращении. Семья нужна и без запросов к API:
// без неё логин отвечает SETUP_REQUIRED.
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
	ts.authSession = LoginAs(t, ts, admin)

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

	return member, LoginAs(t, ts, member)
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
