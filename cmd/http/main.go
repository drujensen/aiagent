/**
 * @description
 * This file serves as the entry point for the HTTP server variant of the AI Workflow Automation Platform.
 * It initializes the application's dependencies (configuration, database, repositories, services, and controllers)
 * and starts an HTTP server to handle incoming requests, supporting both UI and API endpoints.
 *
 * Key features:
 * - UI Routing: Handles home page rendering at "/" via HomeController.
 * - API Routing: Prefixes REST endpoints with "/api/" for agents, tools, tasks, and WebSocket chat.
 * - Static File Serving: Serves assets (e.g., htmx.min.js, styles.css) from the /static/ directory.
 * - Configuration Loading: Uses Viper to load settings from .env via the config package.
 * - MongoDB Connection: Establishes a connection to MongoDB for data persistence.
 * - WebSocket Integration: Initializes ChatHub for real-time chat functionality.
 *
 * @dependencies
 * - aiagent/internal/api/controllers: Provides controllers for API endpoints (agents, tools, tasks).
 * - aiagent/internal/api/websocket: Provides WebSocket handler for chat.
 * - aiagent/internal/ui: Provides controllers for UI endpoints (home page).
 * - aiagent/internal/domain/services: Provides service implementations (e.g., AgentService, ChatService).
 * - aiagent/internal/infrastructure/config: Loads application configuration.
 * - aiagent/internal/infrastructure/database: Manages MongoDB connection.
 * - aiagent/internal/infrastructure/repositories: Provides MongoDB repository implementations.
 * - go.uber.org/zap: Structured logging for startup and errors.
 * - net/http: Standard Go package for running the HTTP server.
 *
 * @notes
 * - Graceful shutdown via defer ensures resources (MongoDB, logger) are released.
 * - Edge case: Missing .env or MongoDB connection failures result in fatal logs and exit.
 * - Assumption: MongoDB runs via Docker Compose as configured in compose.yml.
 * - Static files are served from ./static/; ensure assets exist in this directory.
 */

package main

import (
	"context"
	"net/http"

	"aiagent/internal/api/controllers"
	"aiagent/internal/api/websocket"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/database"
	"aiagent/internal/infrastructure/repositories"
	"aiagent/internal/ui"

	"go.uber.org/zap"
)

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

	// Initialize services
	agentService := services.NewAgentService(agentRepo)
	toolService := services.NewToolService(toolRepo)
	taskService := services.NewTaskService(taskRepo, agentRepo)
	chatService := services.NewChatService(conversationRepo, taskRepo)

	// Initialize ChatHub for WebSocket functionality
	hub := websocket.NewChatHub()
	go hub.Run()

	// Register message listener with ChatService
	chatService.AddMessageListener(hub.MessageListener)

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
	homeController := ui.NewHomeController(logger)

	// Set up HTTP handlers
	// UI routes
	http.HandleFunc("/", homeController.HomeHandler) // Home page rendering
	http.HandleFunc("/agents", homeController.AgentListHandler)
	http.HandleFunc("/agents/new", homeController.AgentFormHandler)
	http.HandleFunc("/agents/edit/", homeController.AgentFormHandler) // With ID parsing

	// API routes (prefixed with /api/)
	http.HandleFunc("/api/agents", agentController.AgentsHandler)
	http.HandleFunc("/api/agents/", agentController.AgentDetailHandler)
	http.HandleFunc("/api/tools", toolController.ListTools)
	http.HandleFunc("/api/workflows", taskController.StartWorkflow)
	http.HandleFunc("/api/tasks/", taskController.TaskDetailHandler)
	http.HandleFunc("/api/ws/chat", websocket.ChatHandler(hub, chatService, cfg))

	// Serve static files (e.g., htmx.min.js, styles.css)
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Start HTTP server
	logger.Info("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
