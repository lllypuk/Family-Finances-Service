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
		"mongo:6",
		mongodb.WithUsername("testuser"),
		mongodb.WithPassword("testpass"),
	)
	require.NoError(t, err)

	uri, err := mongoContainer.ConnectionString(ctx)
	require.NoError(t, err)

	client, err := mongo.Connect(ctx, options.Client().ApplyURI(uri))
	require.NoError(t, err)

	// Test the connection
	ctxTimeout, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	
	err = client.Ping(ctxTimeout, nil)
	require.NoError(t, err)

	database := client.Database("testdb")

	// Cleanup function to be called in test teardown
	t.Cleanup(func() {
		if client != nil {
			client.Disconnect(context.Background())
		}
		if mongoContainer != nil {
			mongoContainer.Terminate(context.Background())
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

	ctx := context.Background()
	collections, err := m.Database.ListCollectionNames(ctx, nil)
	require.NoError(t, err)

	for _, collectionName := range collections {
		err := m.Database.Collection(collectionName).Drop(ctx)
		if err != nil {
			// Just log the error, don't fail the test
			t.Logf("Failed to drop collection %s: %v", collectionName, err)
		}
	}
}