package testhelpers

import (
	"log/slog"
	"os"
	"testing"
)

// TestMainWithSharedMongo sets up shared MongoDB container for fast testing
// Use this in TestMain functions of test packages that need MongoDB
//
// Example usage:
//
//	func TestMain(m *testing.M) {
//	    testhelpers.TestMainWithSharedMongo(m)
//	}
func TestMainWithSharedMongo(m *testing.M) {
	// Set environment variable to enable container reuse
	if err := os.Setenv("REUSE_MONGO_CONTAINER", "true"); err != nil {
		slog.String("testhelpers: failed to set REUSE_MONGO_CONTAINER: %v", err.Error())
	}

	// Run tests
	code := m.Run()

	// Cleanup shared container
	if err := CleanupSharedContainer(); err != nil {
		// Don't fail if cleanup fails, just log it
		// (this might happen in parallel test execution)
		_ = err
	}

	os.Exit(code)
}

// TestMainWithNewMongo uses separate containers for each test (safer but slower)
// This is the default behavior when TestMainWithSharedMongo is not used
func TestMainWithNewMongo(m *testing.M) {
	// Ensure we don't reuse containers
	os.Unsetenv("REUSE_MONGO_CONTAINER")

	// Run tests
	code := m.Run()

	os.Exit(code)
}
