package main

// This is an example of how to use TestMain in your test packages
// Copy this pattern to packages that use MongoDB containers

// Example for internal/infrastructure/user/user_repository_test.go:
/*
package user

import (
	"os"
	"testing"
	"family-budget-service/internal/testhelpers"
)

func TestMain(m *testing.M) {
	// Enable fast testing with shared container
	testhelpers.TestMainWithSharedMongo(m)
}

// Your tests continue as normal...
func TestUserRepository(t *testing.T) {
	mongoContainer := testhelpers.SetupMongoDB(t)
	// Container will be reused across tests in this package
	// Each test gets its own database: testdb_<timestamp>
}
*/

// Example for tests/integration/users_test.go:
/*
package integration

import (
	"os"
	"testing"
	"family-budget-service/internal/testhelpers"
)

func TestMain(m *testing.M) {
	// Enable fast testing with shared container
	testhelpers.TestMainWithSharedMongo(m)
}
*/

// If you want to keep the original behavior (new container per test):
/*
func TestMain(m *testing.M) {
	// Use separate containers (slower but safer)
	testhelpers.TestMainWithNewMongo(m)
}
*/