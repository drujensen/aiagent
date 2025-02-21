/**
 * @description
 * This file serves as the entry point for the HTTP server variant of the AI Workflow Automation Platform.
 * It initializes dependencies (configuration, database, repositories, services, and controllers)
 * and starts an HTTP server to handle UI and API requests.
 *
 * Key features:
 * - UI Routing: Handles home page, agent-related, workflow-related, and chat routes via HomeController.
 * - API Routing: Manages REST and WebSocket endpoints under /api/.
 * - Static File Serving: Serves assets from /static/ (e.g., htmx.min.js, styles.css).
 *
 * @dependencies
 * - aiagent/internal/api/controllers: API endpoint controllers.
 * - aiagent/internal/api/websocket: WebSocket chat handler.
 * - aiagent/internal/ui: UI controllers (HomeController).
 * - aiagent/internal/domain/services: Service implementations.
 * - aiagent/internal/infrastructure/*: Config, database, and repository implementations.
 * - go.uber.org/zap: Structured logging.
 * - net/http: HTTP server functionality.
 *
 * @notes
 * - Graceful shutdown ensures resource cleanup (MongoDB, logger).
 * - Edge case: Missing .env or MongoDB connection results in fatal logs and exit.
 * - Assumption: MongoDB runs via Docker Compose as per compose.yml.
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

// WorkflowHandlerFunc creates a handler function for /workflows that routes based on HTTP method.
func WorkflowHandlerFunc(homeController *ui.HomeController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			homeController.WorkflowHandler(w, r)
		case http.MethodPost:
			homeController.WorkflowSubmitHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

func main() {
	// Initialize zap logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

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
		TaskService:  taskService,
		AgentService: agentService, // Added for task list enrichment
		Config:       cfg,
	}
	homeController := ui.NewHomeController(logger, agentService, taskService, chatService)

	// Set up HTTP handlers
	// UI routes
	http.HandleFunc("/", homeController.HomeHandler)
	http.HandleFunc("/agents", homeController.AgentListHandler)
	http.HandleFunc("/agents/new", homeController.AgentFormHandler)
	http.HandleFunc("/agents/edit/", homeController.AgentFormHandler)
	http.HandleFunc("/workflows", WorkflowHandlerFunc(homeController)) // Use custom handler for method routing
	http.HandleFunc("/tasks", homeController.TaskListHandler)          // GET for task list partial
	http.HandleFunc("/chat", homeController.ChatHandler)               // GET for chat page
	http.HandleFunc("/chat/", homeController.ChatConversationHandler)  // GET for message history partial

	// API routes (prefixed with /api/)
	http.HandleFunc("/api/agents", agentController.AgentsHandler)
	http.HandleFunc("/api/agents/", agentController.AgentDetailHandler)
	http.HandleFunc("/api/tools", toolController.ListTools)
	http.HandleFunc("/api/workflows", taskController.StartWorkflow)  // JSON API endpoint
	http.HandleFunc("/api/tasks", taskController.ListTasks)          // JSON API endpoint
	http.HandleFunc("/api/tasks/", taskController.TaskDetailHandler) // JSON API endpoint
	http.HandleFunc("/api/ws/chat", websocket.ChatHandler(hub, chatService, cfg))

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Start HTTP server
	logger.Info("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
