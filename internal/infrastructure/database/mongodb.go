package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

/**
 * @description
 * This package provides utilities for managing the MongoDB connection in the AI Workflow Automation Platform.
 * It initializes a single MongoDB client instance and provides access to database collections,
 * ensuring efficient and consistent database interactions across repositories.
 *
 * Key features:
 * - Singleton Connection: Initializes the MongoDB client once and reuses it.
 * - Collection Access: Provides a method to retrieve collection handles for specific entities.
 *
 * @dependencies
 * - go.mongodb.org/mongo-driver/mongo: Official MongoDB driver for Go.
 * - go.uber.org/zap: Structured logging for connection events and errors.
 *
 * @notes
 * - The MongoDB client is initialized with a 10-second timeout for connection and ping operations.
 * - Errors during connection or ping are logged and returned for handling by the caller.
 * - The Disconnect method should be called during application shutdown to release resources.
 */

// MongoDB holds the MongoDB client and database handle.
// It provides a centralized way to access MongoDB collections across the application.
type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

// NewMongoDB creates a new MongoDB instance by connecting to the specified URI and database name.
// It pings the server to ensure the connection is successful.
//
// Parameters:
// - uri: The MongoDB connection string (e.g., "mongodb://localhost:27017").
// - dbName: The name of the database to use (e.g., "aiagent").
// - logger: A Zap logger for logging connection events and errors.
//
// Returns:
// - *MongoDB: The initialized MongoDB instance.
// - error: Any error encountered during connection or ping (e.g., network failure, invalid URI).
func NewMongoDB(uri string, dbName string, logger *zap.Logger) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Configure client options with the provided URI
	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", zap.Error(err), zap.String("uri", uri))
		return nil, err
	}

	// Verify the connection with a ping
	err = client.Ping(ctx, nil)
	if err != nil {
		logger.Error("Failed to ping MongoDB", zap.Error(err), zap.String("uri", uri))
		return nil, err
	}

	logger.Info("Successfully connected to MongoDB", zap.String("database", dbName))

	// Access the specified database
	database := client.Database(dbName)
	return &MongoDB{
		client:   client,
		database: database,
	}, nil
}

// Collection returns a handle to the specified collection in the database.
//
// Parameters:
// - name: The name of the collection (e.g., "agents", "tools").
//
// Returns:
// - *mongo.Collection: The collection handle for performing CRUD operations.
func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

// Disconnect closes the MongoDB client connection.
// It should be called when the application is shutting down to release resources.
//
// Parameters:
// - ctx: Context for controlling the disconnection timeout.
//
// Returns:
// - error: Any error encountered during disconnection (e.g., context deadline exceeded).
func (m *MongoDB) Disconnect(ctx context.Context) error {
	err := m.client.Disconnect(ctx)
	if err != nil {
		return err
	}
	return nil
}

// Notes:
// - Edge case: If the URI is invalid or the server is unreachable, an error is returned and logged.
// - Assumption: The database name "aiagent" matches the configuration in compose.yml and .env.
// - Limitation: No connection pooling configuration beyond the driver's defaults; can be tuned if needed.
