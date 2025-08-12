package testhelpers

import (
	"testing"

	"family-budget-service/internal/application"
	"family-budget-service/internal/handlers"
	budgetRepo "family-budget-service/internal/infrastructure/budget"
	categoryRepo "family-budget-service/internal/infrastructure/category"
	reportRepo "family-budget-service/internal/infrastructure/report"
	transactionRepo "family-budget-service/internal/infrastructure/transaction"
	userRepo "family-budget-service/internal/infrastructure/user"
)

// TestHTTPServer wraps HTTP server setup for testing
type TestHTTPServer struct {
	Server    *application.HTTPServer
	MongoDB   *MongoDBContainer
	Repos     *handlers.Repositories
}

// SetupHTTPServer creates a test HTTP server with MongoDB testcontainers
func SetupHTTPServer(t *testing.T) *TestHTTPServer {
	t.Helper()

	// Set up MongoDB container
	mongoContainer := SetupMongoDB(t)

	// Create repository instances
	userRepository := userRepo.NewRepository(mongoContainer.Database)
	familyRepository := userRepo.NewFamilyRepository(mongoContainer.Database)
	categoryRepository := categoryRepo.NewRepository(mongoContainer.Database)
	transactionRepository := transactionRepo.NewRepository(mongoContainer.Database)
	budgetRepository := budgetRepo.NewRepository(mongoContainer.Database)
	reportRepository := reportRepo.NewRepository(mongoContainer.Database)

	// Create repositories struct
	repositories := &handlers.Repositories{
		User:        userRepository,
		Family:      familyRepository,
		Category:    categoryRepository,
		Transaction: transactionRepository,
		Budget:      budgetRepository,
		Report:      reportRepository,
	}

	// Create HTTP server config
	config := &application.Config{
		Port: "8080",
		Host: "localhost",
	}

	// Create HTTP server
	server := application.NewHTTPServer(repositories, config)

	return &TestHTTPServer{
		Server:  server,
		MongoDB: mongoContainer,
		Repos:   repositories,
	}
}