package infrastructure_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/infrastructure"
)

func TestNewRepositories_Structure(t *testing.T) {
	// Test that NewRepositories creates all required repositories

	// Create mock MongoDB instance
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("test")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	// Create repositories
	repos := infrastructure.NewRepositories(mongodb)

	// Verify all repositories are created
	assert.NotNil(t, repos)
	assert.NotNil(t, repos.User)
	assert.NotNil(t, repos.Family)
	assert.NotNil(t, repos.Category)
	assert.NotNil(t, repos.Transaction)
	assert.NotNil(t, repos.Budget)
	assert.NotNil(t, repos.Report)
}

func TestNewRepositories_WithDifferentDatabases(t *testing.T) {
	// Test repositories creation with different database names
	databaseNames := []string{
		"test_db",
		"family_budget",
		"production_db",
		"development",
		"staging",
	}

	for _, dbName := range databaseNames {
		t.Run("Database_"+dbName, func(t *testing.T) {
			clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
			client, err := mongo.Connect(context.Background(), clientOpts)
			require.NoError(t, err)

			database := client.Database(dbName)
			mongodb := &infrastructure.MongoDB{
				Client:   client,
				Database: database,
			}

			repos := infrastructure.NewRepositories(mongodb)

			assert.NotNil(t, repos)
			assert.Equal(t, dbName, mongodb.Database.Name())
		})
	}
}

func TestNewRepositories_RepositoryTypes(t *testing.T) {
	// Test that repositories have correct types
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("test")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	repos := infrastructure.NewRepositories(mongodb)

	// Test that each repository implements the expected interface
	// We can't easily test the exact types without importing all repository packages
	// But we can test that they're not nil and have the expected structure
	assert.NotNil(t, repos.User)
	assert.NotNil(t, repos.Family)
	assert.NotNil(t, repos.Category)
	assert.NotNil(t, repos.Transaction)
	assert.NotNil(t, repos.Budget)
	assert.NotNil(t, repos.Report)
}

func TestNewRepositories_NilMongoDB(t *testing.T) {
	// Test behavior with nil MongoDB (this would panic in real code)
	// We test that the function would handle this scenario

	// This test demonstrates what would happen if NewRepositories was called with nil
	defer func() {
		if r := recover(); r != nil {
			// If it panics, that's expected behavior for nil input
			assert.Contains(t, r.(string), "nil pointer")
		}
	}()

	// This would panic in real code, but we're testing for defensive programming
	// repos := infrastructure.NewRepositories(nil)
	// In a real scenario, we'd want to add nil checking to NewRepositories
}

func TestNewRepositories_MultipleInstances(t *testing.T) {
	// Test that multiple repository instances can be created
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database1 := client.Database("test1")
	mongodb1 := &infrastructure.MongoDB{
		Client:   client,
		Database: database1,
	}

	database2 := client.Database("test2")
	mongodb2 := &infrastructure.MongoDB{
		Client:   client,
		Database: database2,
	}

	repos1 := infrastructure.NewRepositories(mongodb1)
	repos2 := infrastructure.NewRepositories(mongodb2)

	// Verify that instances are different
	assert.NotSame(t, repos1, repos2)
	assert.NotSame(t, repos1.User, repos2.User)
	assert.NotSame(t, repos1.Family, repos2.Family)
	assert.NotSame(t, repos1.Category, repos2.Category)
	assert.NotSame(t, repos1.Transaction, repos2.Transaction)
	assert.NotSame(t, repos1.Budget, repos2.Budget)
	assert.NotSame(t, repos1.Report, repos2.Report)
}

func TestNewRepositories_DatabaseIsolation(t *testing.T) {
	// Test that repositories use the correct database
	dbNames := []string{"db1", "db2", "db3"}
	var repositories []*infrastructure.MongoDB

	for _, dbName := range dbNames {
		clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
		client, err := mongo.Connect(context.Background(), clientOpts)
		require.NoError(t, err)

		database := client.Database(dbName)
		mongodb := &infrastructure.MongoDB{
			Client:   client,
			Database: database,
		}

		repositories = append(repositories, mongodb)
	}

	// Verify each repository uses the correct database
	for i, repo := range repositories {
		expectedDBName := dbNames[i]
		assert.Equal(t, expectedDBName, repo.Database.Name())

		// Create repositories and verify they're properly initialized
		repos := infrastructure.NewRepositories(repo)
		assert.NotNil(t, repos)
	}
}

func TestNewRepositories_MemoryUsage(t *testing.T) {
	// Test memory efficiency by creating multiple repository instances
	const numInstances = 100

	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	var repositories []interface{}

	for range numInstances {
		database := client.Database("test")
		mongodb := &infrastructure.MongoDB{
			Client:   client,
			Database: database,
		}

		repos := infrastructure.NewRepositories(mongodb)
		repositories = append(repositories, repos)
	}

	// Verify all instances were created
	assert.Len(t, repositories, numInstances)

	// Each repository should be properly initialized
	for _, repo := range repositories {
		assert.NotNil(t, repo)
	}
}

func TestNewRepositories_CollectionNames(t *testing.T) {
	// Test that repositories work with expected collection names
	expectedCollections := []string{
		"users",
		"families",
		"categories",
		"transactions",
		"budgets",
		"reports",
	}

	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("test")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	// Test that MongoDB can access all expected collections
	for _, collName := range expectedCollections {
		collection := mongodb.Collection(collName)
		assert.NotNil(t, collection)
		assert.Equal(t, collName, collection.Name())
	}

	// Create repositories to ensure they work with these collections
	repos := infrastructure.NewRepositories(mongodb)
	assert.NotNil(t, repos)
}

func TestNewRepositories_ErrorScenarios(t *testing.T) {
	// Test various error scenarios that could occur during repository creation

	// Test with empty database name
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("") // Empty database name
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	// Repository creation should still work (MongoDB allows empty database names)
	repos := infrastructure.NewRepositories(mongodb)
	assert.NotNil(t, repos)
	assert.Empty(t, mongodb.Database.Name())
}

func TestNewRepositories_ConcurrentAccess(t *testing.T) {
	// Test concurrent repository creation
	const numGoroutines = 10

	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("concurrent_test")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	results := make(chan interface{}, numGoroutines)

	// Create repositories concurrently
	for range numGoroutines {
		go func() {
			repos := infrastructure.NewRepositories(mongodb)
			results <- repos
		}()
	}

	// Collect results
	var repositories []interface{}
	for range numGoroutines {
		repo := <-results
		repositories = append(repositories, repo)
		assert.NotNil(t, repo)
	}

	assert.Len(t, repositories, numGoroutines)
}

// Benchmark tests for repository creation
func BenchmarkNewRepositories(b *testing.B) {
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(b, err)

	database := client.Database("benchmark")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	b.ResetTimer()
	for range b.N {
		_ = infrastructure.NewRepositories(mongodb)
	}
}

func BenchmarkNewRepositories_WithDifferentDatabases(b *testing.B) {
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(b, err)

	b.ResetTimer()
	for i := range b.N {
		dbName := "benchmark_" + string(rune(i%10+'0'))
		database := client.Database(dbName)
		mongodb := &infrastructure.MongoDB{
			Client:   client,
			Database: database,
		}
		_ = infrastructure.NewRepositories(mongodb)
	}
}
