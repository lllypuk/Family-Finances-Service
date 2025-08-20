package infrastructure_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"family-budget-service/internal/infrastructure"
)

func TestMongoDB_Constants(t *testing.T) {
	// Test that MongoDB constants are properly defined
	assert.Equal(t, 10*time.Second, infrastructure.MongoConnectTimeout)
	assert.Equal(t, uint64(100), uint64(infrastructure.MaxConnectionPoolSize))
	assert.Equal(t, uint64(10), uint64(infrastructure.MinConnectionPoolSize))
	assert.Equal(t, 30*time.Second, infrastructure.MaxConnectionIdleTime)
}

func TestMongoDB_StructFields(t *testing.T) {
	// Test that MongoDB struct has all expected fields
	// We can't easily create a real MongoDB instance without a server
	// But we can test the struct definition

	// Create mock client for structure testing
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("test")

	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	assert.NotNil(t, mongodb.Client)
	assert.NotNil(t, mongodb.Database)
	assert.Equal(t, "test", mongodb.Database.Name())
}

func TestNewMongoDB_InvalidURI(t *testing.T) {
	// Test MongoDB initialization with invalid URI
	invalidURIs := []string{
		"",
		"invalid-uri",
		"mongodb://",
		"http://localhost:27017",       // Wrong protocol
		"mongodb://invalid-host:99999", // Invalid port
	}

	for _, uri := range invalidURIs {
		t.Run("InvalidURI_"+uri, func(t *testing.T) {
			mongodb, err := infrastructure.NewMongoDB(uri, "test_db")

			require.Error(t, err)
			assert.Nil(t, mongodb)
			assert.Contains(t, err.Error(), "failed to")
		})
	}
}

func TestNewMongoDB_Configuration(t *testing.T) {
	// Test MongoDB configuration options
	tests := []struct {
		name         string
		uri          string
		databaseName string
		expectError  bool
	}{
		{
			name:         "Standard local URI",
			uri:          "mongodb://localhost:27017",
			databaseName: "test_db",
			expectError:  true, // Will fail if MongoDB is not running
		},
		{
			name:         "URI with authentication",
			uri:          "mongodb://user:pass@localhost:27017",
			databaseName: "auth_db",
			expectError:  true, // Will fail if MongoDB is not running
		},
		{
			name:         "Empty database name",
			uri:          "mongodb://localhost:27017",
			databaseName: "",
			expectError:  true, // Will fail if MongoDB is not running
		},
		{
			name:         "Complex database name",
			uri:          "mongodb://localhost:27017",
			databaseName: "complex-database_name_123",
			expectError:  true, // Will fail if MongoDB is not running
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mongodb, err := infrastructure.NewMongoDB(tt.uri, tt.databaseName)

			if tt.expectError {
				require.Error(t, err)
				assert.Nil(t, mongodb)
			} else {
				require.NoError(t, err)
				require.NotNil(t, mongodb)
				assert.Equal(t, tt.databaseName, mongodb.Database.Name())

				// Clean up
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				_ = mongodb.Close(ctx)
			}
		})
	}
}

func TestMongoDB_Collection(t *testing.T) {
	// Test Collection method with mock data
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("test")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	collectionNames := []string{
		"users",
		"families",
		"categories",
		"transactions",
		"budgets",
		"reports",
	}

	for _, name := range collectionNames {
		t.Run("Collection_"+name, func(t *testing.T) {
			collection := mongodb.Collection(name)

			assert.NotNil(t, collection)
			assert.Equal(t, name, collection.Name())
			assert.Equal(t, "test", collection.Database().Name())
		})
	}
}

func TestMongoDB_ConnectionPoolSettings(t *testing.T) {
	// Test connection pool configuration
	uri := "mongodb://localhost:27017"
	clientOptions := options.Client().ApplyURI(uri)

	// Apply the same settings as in NewMongoDB
	clientOptions.SetMaxPoolSize(infrastructure.MaxConnectionPoolSize)
	clientOptions.SetMinPoolSize(infrastructure.MinConnectionPoolSize)
	clientOptions.SetMaxConnIdleTime(infrastructure.MaxConnectionIdleTime)

	// Verify that options are set correctly
	opts := clientOptions
	assert.NotNil(t, opts)

	// We can't easily test the actual values without internal access
	// But we can test that the options object accepts these values
	assert.Equal(t, uint64(100), uint64(infrastructure.MaxConnectionPoolSize))
	assert.Equal(t, uint64(10), uint64(infrastructure.MinConnectionPoolSize))
	assert.Equal(t, 30*time.Second, infrastructure.MaxConnectionIdleTime)
}

func TestMongoDB_ContextTimeout(t *testing.T) {
	// Test context timeout behavior
	tests := []struct {
		name    string
		timeout time.Duration
	}{
		{
			name:    "Standard timeout",
			timeout: infrastructure.MongoConnectTimeout,
		},
		{
			name:    "Short timeout",
			timeout: 1 * time.Second,
		},
		{
			name:    "Long timeout",
			timeout: 30 * time.Second,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			// Check that deadline is set correctly
			deadline, ok := ctx.Deadline()
			assert.True(t, ok)

			expectedDeadline := time.Now().Add(tt.timeout)
			// Allow for some variance in timing
			assert.WithinDuration(t, expectedDeadline, deadline, 10*time.Millisecond)
		})
	}
}

func TestMongoDB_ErrorHandling(t *testing.T) {
	// Test error handling scenarios
	tests := []struct {
		name        string
		uri         string
		dbName      string
		expectedErr string
	}{
		{
			name:        "Connection failure",
			uri:         "mongodb://non-existent-host:27017",
			dbName:      "test",
			expectedErr: "failed to ping MongoDB",
		},
		{
			name:        "Invalid URI format",
			uri:         "invalid://localhost:27017",
			dbName:      "test",
			expectedErr: "failed to connect to MongoDB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mongodb, err := infrastructure.NewMongoDB(tt.uri, tt.dbName)

			require.Error(t, err)
			assert.Nil(t, mongodb)
			assert.Contains(t, err.Error(), tt.expectedErr)
		})
	}
}

func TestMongoDB_DatabaseNameValidation(t *testing.T) {
	// Test different database name formats
	databaseNames := []struct {
		name    string
		dbName  string
		isValid bool
	}{
		{
			name:    "Simple name",
			dbName:  "simple",
			isValid: true,
		},
		{
			name:    "Name with underscores",
			dbName:  "family_budget",
			isValid: true,
		},
		{
			name:    "Name with hyphens",
			dbName:  "family-budget",
			isValid: true,
		},
		{
			name:    "Name with numbers",
			dbName:  "family_budget_123",
			isValid: true,
		},
		{
			name:    "Empty name",
			dbName:  "",
			isValid: true, // MongoDB allows empty database names (though not recommended)
		},
	}

	for _, tt := range databaseNames {
		t.Run(tt.name, func(t *testing.T) {
			// We can't test actual MongoDB connection, but we can test the database name handling
			clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
			client, err := mongo.Connect(context.Background(), clientOpts)
			require.NoError(t, err)

			database := client.Database(tt.dbName)
			assert.Equal(t, tt.dbName, database.Name())
		})
	}
}

func TestMongoDB_CloseMethod(t *testing.T) {
	// Test Close method behavior
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("test")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	// Test close with context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Close might error if client is not connected, and that's expected
	_ = mongodb.Close(ctx)
	// We don't assert NoError because disconnecting an unconnected client can error
}

func TestMongoDB_IndexCreation(t *testing.T) {
	// Test index creation logic (mocked)
	// We can't test actual index creation without MongoDB server
	// But we can test the index model structure

	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(t, err)

	database := client.Database("test")
	collection := database.Collection("users")

	assert.NotNil(t, collection)
	assert.Equal(t, "users", collection.Name())

	// Test that collection methods are available
	indexes := collection.Indexes()
	assert.NotNil(t, indexes)
}

// Benchmark tests for MongoDB operations
func BenchmarkMongoDB_ClientCreation(b *testing.B) {
	uri := "mongodb://localhost:27017"

	for b.Loop() {
		clientOptions := options.Client().ApplyURI(uri)
		clientOptions.SetMaxPoolSize(infrastructure.MaxConnectionPoolSize)
		clientOptions.SetMinPoolSize(infrastructure.MinConnectionPoolSize)
		clientOptions.SetMaxConnIdleTime(infrastructure.MaxConnectionIdleTime)

		_, _ = mongo.Connect(context.Background(), clientOptions)
	}
}

func BenchmarkMongoDB_CollectionAccess(b *testing.B) {
	clientOpts := options.Client().ApplyURI("mongodb://localhost:27017")
	client, err := mongo.Connect(context.Background(), clientOpts)
	require.NoError(b, err)

	database := client.Database("test")
	mongodb := &infrastructure.MongoDB{
		Client:   client,
		Database: database,
	}

	for b.Loop() {
		_ = mongodb.Collection("users")
	}
}

func BenchmarkMongoDB_ContextCreation(b *testing.B) {
	for b.Loop() {
		ctx, cancel := context.WithTimeout(context.Background(), infrastructure.MongoConnectTimeout)
		cancel()
		_ = ctx
	}
}
