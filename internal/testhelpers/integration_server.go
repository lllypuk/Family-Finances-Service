package testhelpers

import (
	"context"
	"log/slog"
	"testing"

	"family-budget-service/internal/application"
	"family-budget-service/internal/application/handlers"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	reportrepo "family-budget-service/internal/infrastructure/report"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	userrepo "family-budget-service/internal/infrastructure/user"
	"family-budget-service/internal/services"
)

// TestServer represents a test HTTP server setup
type TestServer struct {
	Repos     *handlers.Repositories
	Services  *services.Services
	Server    *application.HTTPServer
	Container *SQLiteTestDB
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

	// Create BackupService for testing with in-memory database
	backupService := services.NewBackupService(db, ":memory:", slog.Default())

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

	// Create HTTP server configuration for testing
	config := &application.Config{
		Port:          "8080",
		Host:          "localhost",
		SessionSecret: "test-session-secret-for-integration-tests",
		IsProduction:  false,
	}

	// Create HTTP server without observability for testing
	httpServer := application.NewHTTPServer(repos, servicesContainer, config)

	testServer := &TestServer{
		Repos:     repos,
		Services:  servicesContainer,
		Server:    httpServer,
		Container: container,
	}

	// Cleanup handler
	t.Cleanup(func() {
		testServer.Cleanup()
	})

	return testServer
}

// Cleanup cleans up the test server resources
func (ts *TestServer) Cleanup() {
	// Container cleanup is handled by testcontainers automatically
	// No explicit cleanup needed as testcontainers handles it
}

// CheckTableExists checks if a table exists in the database (for debugging)
func (ts *TestServer) CheckTableExists(t *testing.T, tableName string) bool {
	var exists int
	err := ts.Container.DB.QueryRowContext(
		context.Background(),
		"SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name=?",
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Logf("Error checking table existence: %v", err)
		return false
	}
	return exists > 0
}
