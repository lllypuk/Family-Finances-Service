package testhelpers

import (
	"context"
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

// MongoDBContainer wraps the testcontainers MongoDB instance
type MongoDBContainer struct {
	Container *mongodb.MongoDBContainer
	URI       string
	Client    *mongo.Client
	Database  *mongo.Database
}

// SetupMongoDB creates a new MongoDB testcontainer and returns a configured client
func SetupMongoDB(t *testing.T) *MongoDBContainer {
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
