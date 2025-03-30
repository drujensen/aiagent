package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"aiagent/internal/domain/interfaces"
	"aiagent/internal/domain/services"
	"aiagent/internal/impl/config"
	"aiagent/internal/impl/database"
	"aiagent/internal/impl/repositories"

	"go.uber.org/zap"
)

func main() {
	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync() // flushes buffer, if any

	// Initialize config
	cfg, err := config.InitConfig()
	if err != nil {
		logger.Fatal("Failed to initialize config", zap.Error(err))
		os.Exit(1)
	}

	// Initialize MongoDB connection
	mongodb, err := database.NewMongoDB(cfg.MongoURI, "aiagent", logger)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
		os.Exit(1)
	}

	// Get collections
	providersCollection := mongodb.Collection("providers")

	// Initialize repositories
	var providerRepo interfaces.ProviderRepository
	providerRepo = repositories.NewMongoProviderRepository(providersCollection)

	// Initialize services
	providerService := services.NewProviderService(providerRepo, logger)

	// Create a timeout context
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Reset providers
	fmt.Println("Resetting all providers to defaults...")
	if err := providerService.ResetDefaultProviders(ctx); err != nil {
		logger.Fatal("Failed to reset providers", zap.Error(err))
		os.Exit(1)
	}

	fmt.Println("Successfully reset all providers to defaults!")

	// Disconnect from MongoDB
	if err := mongodb.Disconnect(context.Background()); err != nil {
		logger.Error("Failed to disconnect from MongoDB", zap.Error(err))
	}
}
