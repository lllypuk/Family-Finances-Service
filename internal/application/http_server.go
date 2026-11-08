package application

import (
	"context"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"time"

	"family-budget-service/internal/application/handlers"
	"family-budget-service/internal/application/router"
	"family-budget-service/internal/auth"
	"family-budget-service/internal/domain/user"
	"family-budget-service/internal/metrics"
	"family-budget-service/internal/observability"
	"family-budget-service/internal/services"
	"family-budget-service/internal/version"
)

const (
	// HTTPRequestTimeout timeout for HTTP requests
	HTTPRequestTimeout = 30 * time.Second

	// recognizeRoute живёт на своих дедлайнах (handlers.RecognizeHandler), общий таймаут его не касается.
	recognizeRoute = apiV1Prefix + transactionsPrefix + recognizeSuffix

	apiV1Prefix        = "/api/v1"
	transactionsPrefix = "/transactions"
	recognizeSuffix    = "/recognize"
)

type HTTPServer struct {
	router               *router.Router
	handler              http.Handler
	server               *http.Server
	services             *services.Services
	config               *Config
	observabilityService *observability.Service
	healthService        *observability.HealthService

	// API Handlers
	authHandler           *handlers.AuthHandler
	meHandler             *handlers.MeHandler
	userHandler           *handlers.UserHandler
	familyHandler         *handlers.FamilyHandler
	categoryHandler       *handlers.CategoryHandler
	accountHandler        *handlers.AccountHandler
	holdingHandler        *handlers.HoldingHandler
	transactionHandler    *handlers.TransactionHandler
	budgetHandler         *handlers.BudgetHandler
	statsHandler          *handlers.StatsHandler
	reconciliationHandler *handlers.ReconciliationHandler
	backupHandler         *handlers.BackupHandler
	recognizeHandler      *handlers.RecognizeHandler
}

type Config struct {
	Port         string
	Host         string
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration
	// TrustedProxies — сети, чей X-Forwarded-For определяет адрес клиента (auth.ClientIP); пусто — только RemoteAddr.
	TrustedProxies []*net.IPNet
	// LoginLimiter — лимитер POST /auth/login; nil — auth.NewRateLimiter, причём без TrustedProxies
	// корзина per-IP отключена: за прокси все клиенты — один адрес, лимит закрывал бы вход всей семье.
	LoginLimiter *auth.RateLimiter
	// Metrics — реестр Prometheus; nil — HTTP-метрики не пишутся (NewHTTPServer без Config.Metrics,
	// юнит-тесты хендлеров). Интеграционный стенд реестр задаёт.
	Metrics *metrics.Metrics
	// RecognizeUploadTimeout — срок приёма тела распознавания; 0 — handlers.RecognizeUploadTimeout.
	RecognizeUploadTimeout time.Duration
}

// NewHTTPServer создает HTTP сервер без observability (для обратной совместимости)
func NewHTTPServer(repositories *handlers.Repositories, services *services.Services, config *Config) *HTTPServer {
	return NewHTTPServerWithObservability(repositories, services, config, nil)
}

// NewHTTPServerWithObservability создает HTTP сервер с observability
func NewHTTPServerWithObservability(
	repositories *handlers.Repositories,
	services *services.Services,
	config *Config,
	obsService *observability.Service,
) *HTTPServer {
	// Без observability /health всё равно отвечает по схеме Health (в том числе setup_complete).
	healthService := observability.NewHealthService(version.String())
	logger := slog.Default()
	if obsService != nil {
		healthService = obsService.HealthService
		logger = obsService.Logger
	}
	healthService.SetSetupChecker(services.Family.IsSetupComplete)

	limiter := config.LoginLimiter
	if limiter == nil {
		var opts []auth.LimiterOption
		if len(config.TrustedProxies) == 0 {
			opts = append(opts, auth.WithoutIPLimit())
		}
		limiter = auth.NewRateLimiter(nil, opts...)
	}

	uploadTimeout := config.RecognizeUploadTimeout
	if uploadTimeout == 0 {
		uploadTimeout = handlers.RecognizeUploadTimeout
	}

	rt := router.New(newAPIErrorHandler(logger))
	clientIP := auth.ClientIP(config.TrustedProxies)

	server := &HTTPServer{
		router:               rt,
		services:             services,
		config:               config,
		observabilityService: obsService,
		healthService:        healthService,

		// Инициализация API handlers
		authHandler: handlers.NewAuthHandler(
			services.Auth, limiter, logger, loginObserver(config.Metrics), clientIP,
		),
		meHandler:             handlers.NewMeHandler(services.User, services.Auth),
		userHandler:           handlers.NewUserHandler(services.User, services.Auth),
		familyHandler:         handlers.NewFamilyHandler(services.Family),
		categoryHandler:       handlers.NewCategoryHandler(services.Category),
		accountHandler:        handlers.NewAccountHandler(services.Account),
		holdingHandler:        handlers.NewHoldingHandler(services.Holding),
		transactionHandler:    handlers.NewTransactionHandler(services.Transaction),
		budgetHandler:         handlers.NewBudgetHandler(repositories, services.Budget),
		statsHandler:          handlers.NewStatsHandler(services.Stats),
		reconciliationHandler: handlers.NewReconciliationHandler(services.Reconciliation),
		backupHandler:         handlers.NewBackupHandler(services.Backup),
		recognizeHandler:      handlers.NewRecognizeHandler(services.Recognize, uploadTimeout),
	}

	server.setupRoutes()
	server.handler = server.buildChain(logger, clientIP)
	server.server = server.buildNetHTTPServer(net.JoinHostPort(config.Host, config.Port))

	return server
}

// loginObserver отдаёт счётчик входов реестра или заглушку, если метрики выключены.
func loginObserver(m *metrics.Metrics) handlers.LoginObserver {
	if m == nil {
		return handlers.NopLoginObserver{}
	}
	return m.Login()
}

// buildChain оборачивает роутер: метрики, RequestID, Recover, Timeout, лог запроса — снаружи внутрь.
func (s *HTTPServer) buildChain(logger *slog.Logger, clientIP func(*http.Request) string) http.Handler {
	var h http.Handler = s.router
	h = observability.RequestLogger(logger, clientIP)(h)
	h = observability.Timeout(HTTPRequestTimeout, func(r *http.Request) bool {
		return r.Method == http.MethodPost && r.URL.EscapedPath() == recognizeRoute
	})(h)
	h = observability.Recover(s.onPanic(logger), func(w http.ResponseWriter, _ *http.Request) {
		_ = handlers.RespondAPIError(w, http.StatusInternalServerError, handlers.ErrCodeInternal,
			statusMessage(http.StatusInternalServerError))
	})(h)
	h = observability.RequestID(h)
	if s.config.Metrics != nil {
		h = metrics.HTTPMiddleware(s.config.Metrics.HTTP())(h)
	}

	return h
}

// onPanic считает панику и пишет стек в slog.
func (s *HTTPServer) onPanic(logger *slog.Logger) func(*http.Request, any, []byte) {
	return func(r *http.Request, v any, stack []byte) {
		if s.config.Metrics != nil {
			s.config.Metrics.HTTP().ObservePanic()
		}
		logger.ErrorContext(r.Context(), "HTTP handler panicked",
			slog.String("method", r.Method),
			slog.String("path", r.URL.Path),
			slog.String("error", fmt.Sprint(v)),
			slog.String("stack", string(stack)),
		)
	}
}

// ServeHTTP отдаёт запрос всей цепочке: middleware и роутеру.
func (s *HTTPServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.handler.ServeHTTP(w, r)
}

// Router — роутер сервера; тестовые стенды добавляют в него свои маршруты.
func (s *HTTPServer) Router() *router.Router {
	return s.router
}

// Routes — зарегистрированные маршруты; по ним сверяется спецификация.
func (s *HTTPServer) Routes() []router.Route {
	return s.router.Routes()
}

func (s *HTTPServer) setupRoutes() {
	s.router.Register(http.MethodGet, "/health", s.healthService.HealthHandler())

	// Единственный публичный маршрут API. Зарегистрирован мимо группы, поэтому
	// RequireBearer его не касается.
	s.router.Register(http.MethodPost, apiV1Prefix+"/auth/login", s.authHandler.Login)

	api := s.router.Group(apiV1Prefix, auth.RequireBearer(s.services.Auth))

	api.Register(http.MethodPost, "/auth/logout", s.authHandler.Logout)
	api.Register(http.MethodGet, "/auth/sessions", s.authHandler.ListSessions)
	api.Register(http.MethodDelete, "/auth/sessions/:id", s.authHandler.RevokeSession)

	api.Register(http.MethodGet, "/me", s.meHandler.GetMe)
	api.Register(http.MethodPut, "/me", s.meHandler.UpdateMe)
	api.Register(http.MethodPut, "/me/password", s.meHandler.ChangePassword)

	s.setupResourceRoutes(api)
}

// setupHoldingRoutes — активы и пассивы; удаление уносит историю снимков, поэтому только админ.
func (s *HTTPServer) setupHoldingRoutes(api *router.Group, financeAccess, adminOnly router.Middleware) {
	holdings := api.Group("/holdings", financeAccess)
	holdings.Register(http.MethodGet, "", s.holdingHandler.ListHoldings)
	holdings.Register(http.MethodPost, "", s.holdingHandler.CreateHolding)
	holdings.Register(http.MethodPut, "/:id", s.holdingHandler.UpdateHolding)
	holdings.Register(http.MethodDelete, "/:id", s.holdingHandler.DeleteHolding, adminOnly)
	holdings.Register(http.MethodGet, "/:id/values", s.holdingHandler.ListHoldingValues)
	holdings.Register(http.MethodPut, "/:id/values/:date", s.holdingHandler.PutHoldingValue)
	holdings.Register(http.MethodDelete, "/:id/values/:date", s.holdingHandler.DeleteHoldingValue)
}

// setupResourceRoutes — users только админ, финансовые разделы админ и member; удаление категории необратимо, поэтому админ.
func (s *HTTPServer) setupResourceRoutes(api *router.Group) {
	adminOnly := auth.RequireRole(user.RoleAdmin)
	financeAccess := auth.RequireRole(user.RoleAdmin, user.RoleMember)

	// Семья: читают все роли, меняет только админ.
	family := api.Group("/family")
	family.Register(http.MethodGet, "", s.familyHandler.GetFamily)
	family.Register(http.MethodPut, "", s.familyHandler.UpdateFamily, adminOnly)

	users := api.Group("/users", adminOnly)
	users.Register(http.MethodPost, "", s.userHandler.CreateUser)
	users.Register(http.MethodGet, "", s.userHandler.GetUsers)
	users.Register(http.MethodGet, "/:id", s.userHandler.GetUserByID)
	users.Register(http.MethodPut, "/:id", s.userHandler.UpdateUser)
	users.Register(http.MethodPatch, "/:id", s.userHandler.PatchUser)
	users.Register(http.MethodPut, "/:id/password", s.userHandler.SetUserPassword)

	categories := api.Group("/categories", financeAccess)
	categories.Register(http.MethodPost, "", s.categoryHandler.CreateCategory)
	categories.Register(http.MethodGet, "", s.categoryHandler.GetCategories)
	categories.Register(http.MethodGet, "/:id", s.categoryHandler.GetCategoryByID)
	categories.Register(http.MethodPut, "/:id", s.categoryHandler.UpdateCategory)
	categories.Register(http.MethodDelete, "/:id", s.categoryHandler.DeleteCategory, adminOnly)

	accounts := api.Group("/accounts", financeAccess)
	accounts.Register(http.MethodGet, "", s.accountHandler.ListAccounts)
	accounts.Register(http.MethodPost, "", s.accountHandler.CreateAccount)
	accounts.Register(http.MethodPut, "/:id", s.accountHandler.UpdateAccount)
	accounts.Register(http.MethodDelete, "/:id", s.accountHandler.DeleteAccount, adminOnly)
	accounts.Register(http.MethodPut, "/:id/balances/:month", s.reconciliationHandler.PutAccountBalance)
	accounts.Register(http.MethodDelete, "/:id/balances/:month", s.reconciliationHandler.DeleteAccountBalance)

	s.setupHoldingRoutes(api, financeAccess, adminOnly)

	transactions := api.Group(transactionsPrefix, financeAccess)
	transactions.Register(http.MethodPost, "", s.transactionHandler.CreateTransaction)
	transactions.Register(http.MethodGet, "", s.transactionHandler.GetTransactions)
	transactions.Register(http.MethodPost, "/bulk-delete", s.transactionHandler.BulkDeleteTransactions)
	transactions.Register(http.MethodPost, recognizeSuffix, s.recognizeHandler.Recognize)
	transactions.Register(http.MethodGet, "/:id", s.transactionHandler.GetTransactionByID)
	transactions.Register(http.MethodPut, "/:id", s.transactionHandler.UpdateTransaction)
	transactions.Register(http.MethodDelete, "/:id", s.transactionHandler.DeleteTransaction)

	budgets := api.Group("/budgets", financeAccess)
	budgets.Register(http.MethodPost, "", s.budgetHandler.CreateBudget)
	budgets.Register(http.MethodGet, "", s.budgetHandler.GetBudgets)
	budgets.Register(http.MethodGet, "/:id", s.budgetHandler.GetBudgetByID)
	budgets.Register(http.MethodPut, "/:id", s.budgetHandler.UpdateBudget)
	budgets.Register(http.MethodDelete, "/:id", s.budgetHandler.DeleteBudget)

	stats := api.Group("/stats", financeAccess)
	stats.Register(http.MethodGet, "/summary", s.statsHandler.GetSummary)
	stats.Register(http.MethodGet, "/monthly", s.statsHandler.GetMonthly)
	stats.Register(http.MethodGet, "/net-worth", s.statsHandler.GetNetWorth)
	stats.Register(http.MethodGet, "/reconciliation", s.reconciliationHandler.GetReconciliationStats)

	backups := api.Group("/backups", adminOnly)
	backups.Register(http.MethodPost, "", s.backupHandler.CreateBackup)
	backups.Register(http.MethodGet, "", s.backupHandler.ListBackups)
	backups.Register(http.MethodGet, "/:name/download", s.backupHandler.DownloadBackup)
	backups.Register(http.MethodDelete, "/:name", s.backupHandler.DeleteBackup)
}

// Start слушает до остановки; *http.Server собран в конструкторе, чтобы Shutdown был возможен до старта.
func (s *HTTPServer) Start() error {
	return s.server.ListenAndServe()
}

func (s *HTTPServer) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

func (s *HTTPServer) buildNetHTTPServer(address string) *http.Server {
	return &http.Server{
		Addr:         address,
		Handler:      s,
		ReadTimeout:  s.config.ReadTimeout,
		WriteTimeout: s.config.WriteTimeout,
		IdleTimeout:  s.config.IdleTimeout,
	}
}
