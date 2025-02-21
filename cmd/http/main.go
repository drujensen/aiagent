package main

import (
	"context"
	"net/http"

	"aiagent/internal/api"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/database"
	"aiagent/internal/infrastructure/repositories"

	"go.uber.org/zap"
)

/**
 * @description
 * This file serves as the entry point for the HTTP server variant of the AI Workflow Automation Platform.
 * It initializes the application’s dependencies (configuration, database, repositories, and services) and starts
 * an HTTP server to handle incoming requests. It supports a basic /hello endpoint and prepares services like
 * ChatService for future integration with API controllers and WebSocket handlers.
 *
 * Key features:
 * - Configuration Loading: Uses Viper to load settings from .env via the config package.
 * - MongoDB Connection: Establishes a connection to MongoDB for data persistence.
 * - Repository Initialization: Sets up repositories for agents, tools, tasks, conversations, and audit logs.
 * - Service Initialization: Initializes ChatService for human interaction management.
 * - HTTP Server: Serves endpoints defined in the api package, currently only /hello.
 *
 * @dependencies
 * - aiagent/internal/api: Provides HTTP controllers (e.g., HelloController).
 * - aiagent/internal/domain/services: Provides service implementations (e.g., ChatService).
 * - aiagent/internal/infrastructure/config: Loads application configuration.
 * - aiagent/internal/infrastructure/database: Manages MongoDB connection.
 * - aiagent/internal/infrastructure/repositories: Provides MongoDB repository implementations.
 * - go.uber.org/zap: Structured logging for startup and errors.
 * - net/http: Standard Go package for running the HTTP server.
 * - context: For managing timeouts and cancellations during shutdown.
 *
 * @notes
 * - Services like ChatService are initialized but not yet used; they’re logged temporarily to avoid compiler errors.
 * - The server listens on port 8080, hardcoded for simplicity; this could be made configurable later.
 * - Graceful shutdown via defer ensures resources like MongoDB connections are released properly.
 * - Edge case: Missing .env or MongoDB connection failures result in fatal logs and program exit.
 * - Assumption: MongoDB is running via Docker Compose as configured in compose.yml.
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

	// Initialize services
	chatService := services.NewChatService(conversationRepo, taskRepo)

	// Temporary logging to avoid "declared and not used" errors
	// These will be integrated into controllers or WebSocket handlers in later steps (e.g., Step 9, Step 11)
	logger.Info("Dependencies initialized",
		zap.Any("agentRepo", agentRepo),
		zap.Any("toolRepo", toolRepo),
		zap.Any("taskRepo", taskRepo),
		zap.Any("conversationRepo", conversationRepo),
		zap.Any("auditLogRepo", auditLogRepo),
		zap.Any("chatService", chatService),
	)

	// Start HTTP server
	logger.Info("Starting HTTP server on :8080")
	http.HandleFunc("/hello", api.HelloController)
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
