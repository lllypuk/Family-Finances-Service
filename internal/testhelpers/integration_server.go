package testhelpers

import (
	"context"
	"testing"

	"family-budget-service/internal/application"
	"family-budget-service/internal/application/handlers"
	budgetrepo "family-budget-service/internal/infrastructure/budget"
	categoryrepo "family-budget-service/internal/infrastructure/category"
	reportrepo "family-budget-service/internal/infrastructure/report"
	transactionrepo "family-budget-service/internal/infrastructure/transaction"
	userrepo "family-budget-service/internal/infrastructure/user"
	"family-budget-service/internal/services"
	dbhelper "family-budget-service/internal/testing"
)

// TestServer represents a test HTTP server setup
type TestServer struct {
	Repos     *handlers.Repositories
	Services  *services.Services
	Server    *application.HTTPServer
	container *dbhelper.PostgreSQLTestContainer
}

// SetupHTTPServer creates a test HTTP server with real database connections
func SetupHTTPServer(t *testing.T) *TestServer {
	// Setup PostgreSQL testcontainer
	container := dbhelper.SetupPostgreSQLContainer(t)

	// Get test database
	db := container.DB

	// Create repositories
	repos := &handlers.Repositories{
		User:        userrepo.NewPostgreSQLRepository(db),
		Family:      userrepo.NewPostgreSQLFamilyRepository(db),
		Budget:      budgetrepo.NewPostgreSQLRepository(db),
		Category:    categoryrepo.NewPostgreSQLRepository(db),
		Transaction: transactionrepo.NewPostgreSQLRepository(db),
		Report:      reportrepo.NewPostgreSQLRepository(db),
	}

	// Create services for testing - use simplified version to avoid circular dependencies
	servicesContainer := services.NewServices(
		repos.User,        // userRepo
		repos.Family,      // familyRepo
		repos.Category,    // categoryRepo
		repos.Transaction, // transactionRepo
		repos.Budget,      // budgetRepo for transactions
		repos.Budget,      // fullBudgetRepo
		repos.Report,      // reportRepo
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
		container: container,
	}

	// Cleanup handler
	t.Cleanup(func() {
		testServer.Cleanup()
	})

	return testServer
}

// Cleanup cleans up the test server resources
func (ts *TestServer) Cleanup() {
	if ts.container != nil {
		// Container cleanup is handled by testcontainers automatically
		// but we can add explicit cleanup if needed
	}
}

// CheckTableExists checks if a table exists in the database (for debugging)
func (ts *TestServer) CheckTableExists(t *testing.T, tableName string) bool {
	var exists bool
	err := ts.container.DB.QueryRow(
		context.Background(),
		"SELECT EXISTS (SELECT FROM information_schema.tables WHERE table_schema = 'family_budget' AND table_name = $1)",
		tableName,
	).Scan(&exists)
	if err != nil {
		t.Logf("Error checking table existence: %v", err)
		return false
	}
	return exists
}
