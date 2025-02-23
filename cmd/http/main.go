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
	uicontrollers "aiagent/internal/ui/controllers"

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

	homeController := uicontrollers.NewHomeController(logger, tmpl)
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

	http.HandleFunc("/", homeController.HomeHandler)
	http.HandleFunc("/agents", agentController.AgentListHandler)
	http.HandleFunc("/agents/new", agentController.AgentFormHandler)
	http.HandleFunc("/agents/edit/", agentController.AgentFormHandler)
	http.HandleFunc("/chat", chatController.ChatHandler)
	http.HandleFunc("/chat/", chatController.ChatConversationHandler)

	http.HandleFunc("/api/agents", apiAgentController.AgentsHandler)
	http.HandleFunc("/api/agents/", apiAgentController.AgentDetailHandler)
	http.HandleFunc("/api/tools", apiToolController.ListTools)
	http.HandleFunc("/api/ws/chat", websocket.ChatHandler(hub, chatService, cfg))

	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("internal/ui/static"))))

	logger.Info("Starting HTTP server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}
