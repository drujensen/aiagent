package main

import (
	"context"
	"net/http"

	"aiagent/internal/api"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/database"
	"aiagent/internal/infrastructure/repositories"

	"go.uber.org/zap"
)

/**
 * @description
 * This file serves as the entry point for the HTTP server variant of the AI Workflow Automation Platform.
 * It initializes the application’s dependencies (configuration, database, repositories) and starts an HTTP server
 * to handle incoming requests. Currently, it supports a basic /hello endpoint, with plans to expand as services
 * and controllers are implemented.
 *
 * Key features:
 * - Configuration Loading: Uses Viper to load settings from .env.
 * - MongoDB Connection: Establishes a connection to MongoDB for data persistence.
 * - Repository Initialization: Prepares repositories for future use in services and controllers.
 * - HTTP Server: Serves endpoints defined in the api package.
 *
 * @dependencies
 * - aiagent/internal/api: Provides HTTP controllers (e.g., HelloController).
 * - aiagent/internal/infrastructure/config: Loads application configuration.
 * - aiagent/internal/infrastructure/database: Manages MongoDB connection.
 * - aiagent/internal/infrastructure/repositories: Provides MongoDB repository implementations.
 * - go.uber.org/zap: Structured logging for startup and errors.
 *
 * @notes
 * - Repositories are initialized but not yet used; temporary logging prevents compiler errors.
 * - The server listens on port 8080; this can be made configurable in the future.
 * - Graceful shutdown via defer ensures resources are released properly.
 */

func main() {
	// Initialize zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync() // Flushes buffer, if any

	// Load configuration
	cfg, err := config.InitConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	// Initialize MongoDB connection
	db, err := database.NewMongoDB(cfg.MongoURI, "aiagent", logger)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer db.Disconnect(context.Background())

	// Initialize repositories
	agentRepo := repositories.NewMongoAgentRepository(db.Collection("agents"))
	toolRepo := repositories.NewMongoToolRepository(db.Collection("tools"))
	taskRepo := repositories.NewMongoTaskRepository(db.Collection("tasks"))
	conversationRepo := repositories.NewMongoConversationRepository(db.Collection("conversations"))
	auditLogRepo := repositories.NewMongoAuditLogRepository(db.Collection("audit_logs"))

	// Temporary use of repositories to avoid "declared and not used" errors
	// This will be replaced by service/controller integration in later steps
	logger.Info("Repositories initialized",
		zap.Any("agentRepo", agentRepo),
		zap.Any("toolRepo", toolRepo),
		zap.Any("taskRepo", taskRepo),
		zap.Any("conversationRepo", conversationRepo),
		zap.Any("auditLogRepo", auditLogRepo),
	)

	// Start HTTP server
	logger.Info("Starting HTTP server on :8080")
	http.HandleFunc("/hello", api.HelloController)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

// Notes:
// - Edge case: If .env is missing or misconfigured, the program will exit with a fatal log.
// - Assumption: MongoDB is running via Docker Compose as configured in compose.yml.
// - Limitation: Repositories are not yet functional; this is a placeholder until services are implemented.
