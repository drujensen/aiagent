package main

import (
	"bytes"
	"context"
	"html/template"

	"aiagent/internal/domain/services"
	"aiagent/internal/infrastructure/config"
	"aiagent/internal/infrastructure/database"
	"aiagent/internal/infrastructure/integrations"
	"aiagent/internal/infrastructure/repositories"
	uicontrollers "aiagent/internal/ui/controllers"

	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"github.com/yuin/goldmark"
	gfmext "github.com/yuin/goldmark/extension"
	"go.uber.org/zap"
)

func main() {
	logger, _ := zap.NewDevelopment()
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
	chatRepo := repositories.NewMongoChatRepository(db.Collection("chats"))

	configurations := map[string]string{
		"workspace":      cfg.Workspace,
		"tavily_api_key": cfg.TavilyAPIKey,
	}

	toolRepo, err := integrations.NewToolRegistry(configurations, logger)
	if err != nil {
		logger.Fatal("Failed to initialize tools", zap.Error(err))
	}

	agentService := services.NewAgentService(agentRepo, logger)
	toolService := services.NewToolService(toolRepo, logger)
	chatService := services.NewChatService(chatRepo, agentRepo, toolRepo, cfg, logger)

	// Define custom template functions
	funcMap := template.FuncMap{
		"renderMarkdown": renderMarkdown,
		"inArray": func(value string, array []string) bool {
			for _, item := range array {
				if item == value {
					return true
				}
			}
			return false
		},
	}

	// Parse templates with custom function map
	tmpl, err := template.New("").Funcs(funcMap).ParseFiles(
		"internal/ui/templates/layout.html",
		"internal/ui/templates/header.html",
		"internal/ui/templates/sidebar.html",
		"internal/ui/templates/sidebar_chats.html",
		"internal/ui/templates/sidebar_agents.html",
		"internal/ui/templates/home.html",
		"internal/ui/templates/agent_form.html",
		"internal/ui/templates/chat.html",
		"internal/ui/templates/chat_form.html",
		"internal/ui/templates/messages_partial.html",
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}

	homeController := uicontrollers.NewHomeController(logger, tmpl, chatService, agentService, toolService)
	agentController := uicontrollers.NewAgentController(logger, tmpl, agentService, toolService)
	chatController := uicontrollers.NewChatController(logger, tmpl, chatService, agentService)

	// Initialize Echo
	e := echo.New()

	// Middleware
	e.Use(middleware.Logger())
	e.Use(middleware.Recover())
	e.Use(middleware.RequestID())
	e.Use(middleware.CORS())
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Set("logger", logger)
			return next(c)
		}
	})

	// Static file serving
	e.Static("/static", "internal/ui/static")

	// UI Routes
	e.GET("/", homeController.HomeHandler)

	e.GET("/agents/new", agentController.AgentFormHandler)
	e.POST("/agents", agentController.CreateAgentHandler)
	e.GET("/agents/:id/edit", agentController.AgentFormHandler)
	e.PUT("/agents/:id", agentController.UpdateAgentHandler)
	e.DELETE("/agents/:id", agentController.DeleteAgentHandler) // Added DELETE route

	e.GET("/chats/new", chatController.ChatFormHandler)
	e.POST("/chats", chatController.CreateChatHandler)
	e.GET("/chats/:id", chatController.ChatHandler)
	e.GET("/chats/:id/edit", chatController.ChatFormHandler)
	e.PUT("/chats/:id", chatController.UpdateChatHandler)
	e.DELETE("/chats/:id", chatController.DeleteChatHandler) // Added DELETE route
	e.POST("/chats/:id/messages", chatController.SendMessageHandler)

	// Sidebar Partial Routes
	e.GET("/sidebar/chats", homeController.ChatsPartialHandler)
	e.GET("/sidebar/agents", homeController.AgentsPartialHandler)

	// Start server
	logger.Info("Starting HTTP server on :8080")
	if err := e.Start(":8080"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func renderMarkdown(markdown string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := goldmark.New(goldmark.WithExtensions(gfmext.GFM)).Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
