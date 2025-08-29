package testhelpers

import (
	"context"
	"crypto/rand"
	"math/big"
	"strconv"
	"testing"
	"time"

	"family-budget-service/internal/application"
	"family-budget-service/internal/application/handlers"
	budgetRepo "family-budget-service/internal/infrastructure/budget"
	categoryRepo "family-budget-service/internal/infrastructure/category"
	reportRepo "family-budget-service/internal/infrastructure/report"
	transactionRepo "family-budget-service/internal/infrastructure/transaction"
	userRepo "family-budget-service/internal/infrastructure/user"
	"family-budget-service/internal/services"
)

const (
	StartupDelay = 100 * time.Millisecond
)

// TestHTTPServer wraps HTTP server setup for testing
type TestHTTPServer struct {
	Server  *application.HTTPServer
	MongoDB *MongoDBContainer
	Repos   *handlers.Repositories
	Port    string
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

	// Create services
	serviceContainer := services.NewServices(
		userRepository,
		familyRepository,
		categoryRepository,
		transactionRepository,
		budgetRepository,
		budgetRepository,
	)

	rndPort := getRandomPort()

	// Create HTTP server config
	config := &application.Config{
		Port: strconv.Itoa(rndPort),
		Host: "localhost",
	}

	// Create HTTP server
	server := application.NewHTTPServer(repositories, serviceContainer, config)

	// Start server in background
	go func() {
		if err := server.Start(context.Background()); err != nil {
			t.Logf("Server failed to start: %v", err)
		}
	}()

	// Give the server a moment to start
	time.Sleep(StartupDelay)

	// Cleanup function to stop the server
	t.Cleanup(func() {
		if shutdownErr := server.Shutdown(context.Background()); shutdownErr != nil {
			t.Logf("Failed to shutdown HTTP server: %v", shutdownErr)
		}
	})

	return &TestHTTPServer{
		Server:  server,
		MongoDB: mongoContainer,
		Repos:   repositories,
		Port:    config.Port,
	}
}

func getRandomPort() int {
	const (
		portRangeSize = 10000 // Port range size for random selection
		basePort      = 30000 // Starting port for random range
	)

	// Generate cryptographically secure random number
	maxBig := big.NewInt(portRangeSize)
	n, err := rand.Int(rand.Reader, maxBig)
	if err != nil {
		// Fallback to a deterministic port offset if random generation fails
		const fallbackOffset = 8080
		return basePort + fallbackOffset
	}
	return int(n.Int64()) + basePort // Random port between 30000 and 39999
}
