package database

import (
	"context"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"
)

type MongoDB struct {
	client   *mongo.Client
	database *mongo.Database
}

func NewMongoDB(uri string, dbName string, logger *zap.Logger) (*MongoDB, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	clientOptions := options.Client().ApplyURI(uri)
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		logger.Error("Failed to connect to MongoDB", zap.Error(err), zap.String("uri", uri))
		return nil, err
	}

	err = client.Ping(ctx, nil)
	if err != nil {
		logger.Error("Failed to ping MongoDB", zap.Error(err), zap.String("uri", uri))
		return nil, err
	}

	logger.Info("Successfully connected to MongoDB", zap.String("database", dbName))

	database := client.Database(dbName)
	return &MongoDB{
		client:   client,
		database: database,
	}, nil
}

func (m *MongoDB) Collection(name string) *mongo.Collection {
	return m.database.Collection(name)
}

func (m *MongoDB) Disconnect(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}
