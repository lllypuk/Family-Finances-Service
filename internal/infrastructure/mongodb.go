package infrastructure

import (
	"context"
	"fmt"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

const (
	// MongoConnectTimeout timeout for MongoDB connection establishment
	MongoConnectTimeout = 10 * time.Second
	// MaxConnectionPoolSize maximum number of connections in the pool
	MaxConnectionPoolSize = 100
	// MinConnectionPoolSize minimum number of connections in the pool
	MinConnectionPoolSize = 10
	// MaxConnectionIdleTime maximum time a connection can remain idle
	MaxConnectionIdleTime = 30 * time.Second
)

type MongoDB struct {
	Client   *mongo.Client
	Database *mongo.Database
}

func NewMongoDB(uri, databaseName string) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), MongoConnectTimeout)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)

	// Настройки connection pool
	clientOptions.SetMaxPoolSize(MaxConnectionPoolSize)
	clientOptions.SetMinPoolSize(MinConnectionPoolSize)
	clientOptions.SetMaxConnIdleTime(MaxConnectionIdleTime)

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Проверка подключения
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(databaseName)

	// Ensure unique index on users.email
	usersColl := database.Collection("users")
	indexModel := mongo.IndexModel{
		Keys:    bson.D{{Key: "email", Value: 1}},
		Options: options.Index().SetUnique(true).SetName("uniq_users_email"),
	}
	_, err = usersColl.Indexes().CreateOne(ctx, indexModel)
	if err != nil {
		return nil, fmt.Errorf("failed to create unique index on users.email: %w", err)
	}

	return &MongoDB{
		Client:   client,
		Database: database,
	}, nil
}

func (m *MongoDB) Close(ctx context.Context) error {
	return m.Client.Disconnect(ctx)
}

func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.Database.Collection(name)
}
