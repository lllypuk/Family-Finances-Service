package testhelpers

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// TestMongoTimeout timeout for test MongoDB operations
	TestMongoTimeout = 10 * time.Second
)

var (
	// Global MongoDB container for reuse across tests
	globalMongoContainer *MongoDBContainer
	containerMutex       sync.RWMutex
	initOnce             sync.Once
)

// MongoDBContainer wraps the testcontainers MongoDB instance
type MongoDBContainer struct {
	Container *mongodb.MongoDBContainer
	URI       string
	Client    *mongo.Client
	Database  *mongo.Database
}

// SetupMongoDB creates a new MongoDB testcontainer and returns a configured client
// For faster tests, it can reuse an existing container if REUSE_MONGO_CONTAINER=true
func SetupMongoDB(t *testing.T) *MongoDBContainer {
	t.Helper()

	// Check if we should reuse container
	if os.Getenv("REUSE_MONGO_CONTAINER") == "true" {
		return GetOrCreateSharedMongoDB(t)
	}

	// Original behavior - create new container per test
	return createNewMongoContainer(t)
}

// GetOrCreateSharedMongoDB returns a shared MongoDB container, creating it if necessary
func GetOrCreateSharedMongoDB(t *testing.T) *MongoDBContainer {
	t.Helper()

	containerMutex.RLock()
	if globalMongoContainer != nil {
		// Test connection to ensure container is still alive
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		
		if err := globalMongoContainer.Client.Ping(ctx, nil); err == nil {
			// Container is alive, create a new database for this test
			testDB := globalMongoContainer.Client.Database(fmt.Sprintf("testdb_%d", time.Now().UnixNano()))
			containerMutex.RUnlock()
			
			// Cleanup only the test database, not the container
			t.Cleanup(func() {
				ctx := context.Background()
				if dropErr := testDB.Drop(ctx); dropErr != nil {
					t.Logf("Failed to drop test database: %v", dropErr)
				}
			})

			return &MongoDBContainer{
				Container: globalMongoContainer.Container,
				URI:       globalMongoContainer.URI,
				Client:    globalMongoContainer.Client,
				Database:  testDB,
			}
		}
		
		// Container is dead, need to recreate
		globalMongoContainer = nil
	}
	containerMutex.RUnlock()

	containerMutex.Lock()
	defer containerMutex.Unlock()

	// Double-check pattern
	if globalMongoContainer != nil {
		testDB := globalMongoContainer.Client.Database(fmt.Sprintf("testdb_%d", time.Now().UnixNano()))
		t.Cleanup(func() {
			ctx := context.Background()
			if dropErr := testDB.Drop(ctx); dropErr != nil {
				t.Logf("Failed to drop test database: %v", dropErr)
			}
		})

		return &MongoDBContainer{
			Container: globalMongoContainer.Container,
			URI:       globalMongoContainer.URI,
			Client:    globalMongoContainer.Client,
			Database:  testDB,
		}
	}

	// Create new shared container
	initOnce.Do(func() {
		ctx := context.Background()

		mongoContainer, err := mongodb.Run(ctx,
			"mongodb/mongodb-community-server:8.0-ubi8",
			mongodb.WithUsername("testuser"),
			mongodb.WithPassword("testpass"),
		)
		require.NoError(t, err)

		uri, err := mongoContainer.ConnectionString(ctx)
		require.NoError(t, err)

		client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
		require.NoError(t, err)

		// Test the connection
		ctxTimeout, cancel := context.WithTimeout(ctx, TestMongoTimeout)
		defer cancel()

		err = client.Ping(ctxTimeout, nil)
		require.NoError(t, err)

		globalMongoContainer = &MongoDBContainer{
			Container: mongoContainer,
			URI:       uri,
			Client:    client,
			Database:  nil, // Will be set per test
		}
	})

	testDB := globalMongoContainer.Client.Database(fmt.Sprintf("testdb_%d", time.Now().UnixNano()))
	
	t.Cleanup(func() {
		ctx := context.Background()
		if dropErr := testDB.Drop(ctx); dropErr != nil {
			t.Logf("Failed to drop test database: %v", dropErr)
		}
	})

	return &MongoDBContainer{
		Container: globalMongoContainer.Container,
		URI:       globalMongoContainer.URI,
		Client:    globalMongoContainer.Client,
		Database:  testDB,
	}
}

// createNewMongoContainer creates a new container for each test (original behavior)
func createNewMongoContainer(t *testing.T) *MongoDBContainer {
	t.Helper()

	ctx := context.Background()

	mongoContainer, err := mongodb.Run(ctx,
		"mongodb/mongodb-community-server:8.0-ubi8",
		mongodb.WithUsername("testuser"),
		mongodb.WithPassword("testpass"),
	)
	require.NoError(t, err)

	uri, err := mongoContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.NoError(t, err)

	// Test the connection
	ctxTimeout, cancel := context.WithTimeout(ctx, TestMongoTimeout)
	defer cancel()

	err = client.Ping(ctxTimeout, nil)
	require.NoError(t, err)

	database := client.Database("testdb")

	// Cleanup function to be called in test teardown
	t.Cleanup(func() {
		if client != nil {
			if disconnectErr := client.Disconnect(context.Background()); disconnectErr != nil {
				t.Logf("Failed to disconnect MongoDB client: %v", disconnectErr)
			}
		}
		if mongoContainer != nil {
			if terminateErr := mongoContainer.Terminate(context.Background()); terminateErr != nil {
				t.Logf("Failed to terminate MongoDB container: %v", terminateErr)
			}
		}
	})

	return &MongoDBContainer{
		Container: mongoContainer,
		URI:       uri,
		Client:    client,
		Database:  database,
	}
}

// CleanupSharedContainer terminates the shared container (call this from TestMain)
func CleanupSharedContainer() error {
	containerMutex.Lock()
	defer containerMutex.Unlock()

	if globalMongoContainer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if globalMongoContainer.Client != nil {
			_ = globalMongoContainer.Client.Disconnect(ctx)
		}

		if globalMongoContainer.Container != nil {
			return globalMongoContainer.Container.Terminate(ctx)
		}
	}

	return nil
}

// CleanupCollections drops all collections in the test database
func (m *MongoDBContainer) CleanupCollections(t *testing.T) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), TestMongoTimeout)
	defer cancel()

	// Check if database and client are still valid
	if m.Database == nil || m.Client == nil {
		t.Log("MongoDB database or client is nil, skipping cleanup")
		return
	}

	// Ping to check connection is still alive
	if err := m.Client.Ping(ctx, nil); err != nil {
		t.Logf("MongoDB connection lost, skipping cleanup: %v", err)
		return
	}

	collections, err := m.Database.ListCollectionNames(ctx, nil)
	if err != nil {
		// Just log the error, don't fail the test
		t.Logf("Failed to list collections during cleanup: %v", err)
		return
	}

	for _, collectionName := range collections {
		err = m.Database.Collection(collectionName).Drop(ctx)
		if err != nil {
			// Just log the error, don't fail the test
			t.Logf("Failed to drop collection %s: %v", collectionName, err)
		}
	}
}
