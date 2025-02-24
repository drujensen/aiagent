package main

import (
	"context"
	"html/template"

	"aiagent/internal/api/controllers"
	"aiagent/internal/api/websocket"
	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/database"
	"aiagent/internal/infrastructure/integrations"
	"aiagent/internal/infrastructure/repositories"
	uicontrollers "aiagent/internal/ui/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	cfg, err := config.InitConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	db, err := database.NewMongoDB(cfg.MongoURI, "aiagent", logger)
	if err != nil {
		logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
	}
	defer db.Disconnect(context.Background())

	agentRepo := repositories.NewMongoAgentRepository(db.Collection("agents"))
	toolRepo := repositories.NewMongoToolRepository(db.Collection("tools"))
	conversationRepo := repositories.NewMongoConversationRepository(db.Collection("conversations"))

	workspace := "/workspace"
	if err := integrations.InitializeTools(context.Background(), toolRepo, workspace); err != nil {
		logger.Fatal("Failed to initialize tools", zap.Error(err))
	}

	agentService := services.NewAgentService(agentRepo)
	toolService := services.NewToolService(toolRepo)
	chatService := services.NewChatService(conversationRepo, agentRepo)

	hub := websocket.NewChatHub()
	go hub.Run()

	chatService.AddMessageListener(hub.MessageListener)

	tmpl, err := template.ParseFiles(
		"internal/ui/templates/layout.html",
		"internal/ui/templates/header.html",
		"internal/ui/templates/sidebar.html",
		"internal/ui/templates/home.html",
		"internal/ui/templates/agent_list.html",
		"internal/ui/templates/agent_form.html",
		"internal/ui/templates/chat.html",
		"internal/ui/templates/messages.html",
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}

	homeController := uicontrollers.NewHomeController(logger, tmpl, chatService, agentService, toolService)
	agentController := uicontrollers.NewAgentController(logger, tmpl, agentService, toolService)
	chatController := uicontrollers.NewChatController(logger, tmpl, chatService, agentService)

	apiAgentController := &controllers.AgentController{
		Service: agentService,
		Config:  cfg,
	}
	apiToolController := &controllers.ToolController{
		ToolService: toolService,
		Config:      cfg,
	}

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())    // Logs requests
	e.Use(middleware.Recover())   // Recovers from panics
	e.Use(middleware.RequestID()) // Adds a unique request ID
	e.Use(middleware.CORS())      // Enables CORS for UI
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("logger", logger) // Pass logger to context
			return next(c)
		}
	})

	// Static file serving
	e.Static("/static", "internal/ui/static")

	// UI Routes
	e.GET("/", homeController.HomeHandler)
	e.GET("/agents", agentController.AgentListHandler)
	e.GET("/agents/new", agentController.AgentFormHandler)
	e.GET("/agents/edit/:id", agentController.AgentFormHandler)
	e.GET("/chat", chatController.ChatHandler)
	e.GET("/chat/:id", chatController.ChatConversationHandler)

	// API Routes
	e.GET("/api/agents", apiAgentController.AgentsHandler)
	e.POST("/api/agents", apiAgentController.AgentsHandler)
	e.GET("/api/agents/:id", apiAgentController.AgentDetailHandler)
	e.PUT("/api/agents/:id", apiAgentController.AgentDetailHandler)
	e.DELETE("/api/agents/:id", apiAgentController.AgentDetailHandler)
	e.GET("/api/tools", apiToolController.ListTools)

	// WebSocket Route
	e.GET("/api/ws/chat", func(c echo.Context) error {
		websocket.ChatHandler(hub, chatService, cfg)(c.Response().Writer, c.Request())
		return nil
	})

	// Start server
	logger.Info("Starting HTTP server on :8080")
	if err := e.Start(":8080"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
