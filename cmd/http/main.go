package main

import (
	"context"
	"net/http"

	"aiagent/internal/api/controllers"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/database"
	"aiagent/internal/infrastructure/repositories"

	"go.uber.org/zap"
)

// This file serves as the entry point for the HTTP server variant of the AI Workflow Automation Platform.
// It initializes the application's dependencies (configuration, database, repositories, services, and controllers)
// and starts an HTTP server to handle incoming requests, supporting agent, tool, and task management endpoints,
// as well as a basic /hello endpoint.
//
// Key features:
// - Configuration Loading: Uses Viper to load settings from .env via the config package.
// - MongoDB Connection: Establishes a connection to MongoDB for data persistence.
// - Repository Initialization: Sets up repositories for agents, tools, tasks, conversations, and audit logs.
// - Service Initialization: Initializes AgentService, ToolService, TaskService, and ChatService for business logic.
// - HTTP Server: Serves endpoints for agent, tool, and task management, with plans for further expansion.
//
// Dependencies:
// - aiagent/internal/api/controllers: Provides controllers for agents, tools, and tasks.
// - aiagent/internal/domain/services: Provides service implementations (e.g., AgentService, ToolService).
// - aiagent/internal/infrastructure/config: Loads application configuration.
// - aiagent/internal/infrastructure/database: Manages MongoDB connection.
// - aiagent/internal/infrastructure/repositories: Provides MongoDB repository implementations.
// - go.uber.org/zap: Structured logging for startup and errors.
// - net/http: Standard Go package for running the HTTP server.
// - context: For managing timeouts and cancellations during shutdown.
//
// Notes:
// - Services like ChatService are initialized but not yet fully integrated; expanded in future steps.
// - Server listens on port 8080, hardcoded for simplicity; configurable in future updates.
// - Graceful shutdown via defer ensures resources like MongoDB connections are released.
// - Edge case: Missing .env or MongoDB connection failures result in fatal logs and program exit.
// - Assumption: MongoDB runs via Docker Compose as configured in compose.yml.
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
	//conversationRepo := repositories.NewMongoConversationRepository(db.Collection("conversations"))
	//auditLogRepo := repositories.NewMongoAuditLogRepository(db.Collection("audit_logs"))

	// Initialize services
	agentService := services.NewAgentService(agentRepo)
	toolService := services.NewToolService(toolRepo)
	taskService := services.NewTaskService(taskRepo, agentRepo)
	//chatService := services.NewChatService(conversationRepo, taskRepo)

	// Initialize controllers
	agentController := &controllers.AgentController{
		Service: agentService,
		Config:  cfg,
	}
	toolController := &controllers.ToolController{
		ToolService: toolService,
		Config:      cfg,
	}
	taskController := &controllers.TaskController{
		TaskService: taskService,
		Config:      cfg,
	}

	// Set up HTTP handlers
	http.HandleFunc("/agents", agentController.AgentsHandler)
	http.HandleFunc("/agents/", agentController.AgentDetailHandler)
	http.HandleFunc("/tools", toolController.ListTools)
	http.HandleFunc("/workflows", taskController.StartWorkflow)
	http.HandleFunc("/tasks/", taskController.TaskDetailHandler)

	// Start HTTP server
	logger.Info("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
