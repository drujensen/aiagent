/**
 * @description
 * This file serves as the entry point for the HTTP server variant of the AI Workflow Automation Platform.
 * It initializes dependencies (configuration, database, repositories, services, and controllers)
 * and starts an HTTP server to handle UI and API requests.
 *
 * Key features:
 * - UI Routing: Handles home page, agent-related, workflow-related, task, and chat routes via respective controllers.
 * - API Routing: Manages REST and WebSocket endpoints under /api/.
 * - Static File Serving: Serves assets from /static/ (e.g., htmx.min.js, styles.css).
 * - Tool Registry Initialization: Sets up predefined tools at startup.
 *
 * @dependencies
 * - aiagent/internal/api/controllers: API endpoint controllers.
 * - aiagent/internal/api/websocket: WebSocket chat handler.
 * - aiagent/internal/ui: UI controllers (HomeController, AgentController, etc.).
 * - aiagent/internal/domain/services: Service implementations.
 * - aiagent/internal/infrastructure/*: Config, database, repository, and integration implementations.
 * - go.uber.org/zap: Structured logging.
 * - net/http: HTTP server functionality.
 * - context: For managing timeouts and cancellations.
 *
 * @notes
 * - Graceful shutdown ensures resource cleanup (MongoDB, logger).
 * - Edge case: Missing .env or MongoDB connection results in fatal logs and exit.
 * - Assumption: MongoDB runs via Docker Compose as per compose.yml.
 * - Tool registry uses /workspace as per Docker volume mount; could be configurable in future.
 */
package main

import (
	"context"
	"html/template"
	"net/http"

	"aiagent/internal/api/controllers"
	"aiagent/internal/api/websocket"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/database"
	"aiagent/internal/infrastructure/integrations"
	"aiagent/internal/infrastructure/repositories"
	"aiagent/internal/ui"

	"go.uber.org/zap"
)

// AgentHandlerFunc creates a handler function for /agents that routes based on HTTP method.
func AgentHandlerFunc(agentController *ui.AgentController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			agentController.AgentListHandler(w, r)
		case http.MethodPost:
			agentController.AgentSubmitHandler(w, r)
		default:
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		}
	}
}

// WorkflowHandlerFunc creates a handler function for /workflows that routes based on HTTP method.
func WorkflowHandlerFunc(workflowController *ui.WorkflowController) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			workflowController.WorkflowHandler(w, r)
		case http.MethodPost:
			workflowController.WorkflowSubmitHandler(w, r)
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

	// Initialize predefined tools with workspace path from Docker setup
	workspace := "/workspace"
	if err := integrations.InitializeTools(context.Background(), toolRepo, workspace); err != nil {
		logger.Fatal("Failed to initialize tools", zap.Error(err))
	}

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

	// Parse templates once for all controllers
	tmpl, err := template.ParseFiles(
		"./internal/ui/templates/layout.html",
		"./internal/ui/templates/header.html",
		"./internal/ui/templates/sidebar.html",
		"./internal/ui/templates/home.html",
		"./internal/ui/templates/agent_list.html",
		"./internal/ui/templates/agent_form.html",
		"./internal/ui/templates/workflow.html",
		"./internal/ui/templates/chat.html",
		"./internal/ui/templates/message_history.html",
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}

	// Initialize UI controllers with shared template
	homeController := ui.NewHomeController(logger, tmpl, agentService)
	agentController := ui.NewAgentController(logger, tmpl, agentService, toolService)
	workflowController := ui.NewWorkflowController(logger, tmpl, agentService, taskService)
	taskController := ui.NewTaskController(logger, tmpl, taskService, agentService)
	chatController := ui.NewChatController(logger, tmpl, chatService, agentService)

	// Initialize API controllers
	apiAgentController := &controllers.AgentController{
		Service: agentService,
		Config:  cfg,
	}
	apiToolController := &controllers.ToolController{
		ToolService: toolService,
		Config:      cfg,
	}
	apiTaskController := &controllers.TaskController{
		TaskService:  taskService,
		AgentService: agentService,
		Config:       cfg,
	}

	// Set up HTTP handlers
	// UI routes
	http.HandleFunc("/", homeController.HomeHandler)
	http.HandleFunc("/agents", AgentHandlerFunc(agentController)) // GET and POST
	http.HandleFunc("/agents/new", agentController.AgentFormHandler)
	http.HandleFunc("/agents/edit/", agentController.AgentFormHandler)
	http.HandleFunc("/workflows", WorkflowHandlerFunc(workflowController)) // GET and POST
	http.HandleFunc("/tasks", taskController.TaskListHandler)              // GET for task list partial
	http.HandleFunc("/chat", chatController.ChatHandler)                   // GET for chat page
	http.HandleFunc("/chat/", chatController.ChatConversationHandler)      // GET for message history partial

	// API routes (prefixed with /api/)
	http.HandleFunc("/api/agents", apiAgentController.AgentsHandler)
	http.HandleFunc("/api/agents/", apiAgentController.AgentDetailHandler)
	http.HandleFunc("/api/tools", apiToolController.ListTools)
	http.HandleFunc("/api/workflows", apiTaskController.StartWorkflow)  // JSON API endpoint
	http.HandleFunc("/api/tasks", apiTaskController.ListTasks)          // JSON API endpoint
	http.HandleFunc("/api/tasks/", apiTaskController.TaskDetailHandler) // JSON API endpoint
	http.HandleFunc("/api/ws/chat", websocket.ChatHandler(hub, chatService, cfg))

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Start HTTP server
	logger.Info("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
