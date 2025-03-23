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
	providerRepo := repositories.NewMongoProviderRepository(db.Collection("providers"))

	configurations := map[string]string{
		"workspace":       cfg.Workspace,
		"tavily_api_key":  cfg.TavilyAPIKey,
		"basic_auth_user": cfg.BasicAuthUser,
		"basic_auth_pass": cfg.BasicAuthPass,
	}

	toolRepo, err := integrations.NewToolRegistry(configurations, logger)
	if err != nil {
		logger.Fatal("Failed to initialize tools", zap.Error(err))
	}

	providerService := services.NewProviderService(providerRepo, logger)
	agentService := services.NewAgentService(agentRepo, logger)
	toolService := services.NewToolService(toolRepo, logger)
	chatService := services.NewChatService(chatRepo, agentRepo, providerRepo, toolRepo, cfg, logger)

	// Initialize default providers if needed
	if err := providerService.InitializeDefaultProviders(context.Background()); err != nil {
		logger.Warn("Failed to initialize default providers", zap.Error(err))
	}

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
		"add": func(a, b int) int {
			return a + b
		},
		"sub": func(a, b int) int {
			return a - b
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
		"internal/ui/templates/message_session_partial.html",
		"internal/ui/templates/provider_form.html",
		"internal/ui/templates/provider_models_partial.html",
		"internal/ui/templates/providers_list_content.html",
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}

	homeController := uicontrollers.NewHomeController(logger, tmpl, chatService, agentService, toolService)
	agentController := uicontrollers.NewAgentController(logger, tmpl, agentService, toolService, providerService)
	chatController := uicontrollers.NewChatController(logger, tmpl, chatService, agentService)
	providerController := uicontrollers.NewProviderController(logger, tmpl, providerService)

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

	// Middleware to set Content-Language header
	e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			c.Response().Header().Set("Content-Language", "en")
			return next(c)
		}
	})

	// Use the BasicAuth middleware
	e.Use(basicAuthMiddleware(configurations["basic_auth_user"], configurations["basic_auth_pass"]))

	// Static file serving
	e.Static("/static", "internal/ui/static")

	// UI Routes
	e.GET("/", homeController.HomeHandler)

	e.GET("/agents/new", agentController.AgentFormHandler)
	e.POST("/agents", agentController.CreateAgentHandler)
	e.GET("/agents/repair-providers", agentController.RepairAgentProvidersHandler) // Admin endpoint to fix provider IDs
	e.GET("/agents/:id/edit", agentController.AgentFormHandler)
	e.PUT("/agents/:id", agentController.UpdateAgentHandler)
	e.DELETE("/agents/:id", agentController.DeleteAgentHandler) // Added DELETE route
	e.GET("/agents/provider-models", agentController.GetProviderModelsHandler)

	e.GET("/chats/new", chatController.ChatFormHandler)
	e.POST("/chats", chatController.CreateChatHandler)
	e.GET("/chats/:id", chatController.ChatHandler)
	e.GET("/chats/:id/edit", chatController.ChatFormHandler)
	e.PUT("/chats/:id", chatController.UpdateChatHandler)
	e.DELETE("/chats/:id", chatController.DeleteChatHandler) // Added DELETE route
	e.POST("/chats/:id/messages", chatController.SendMessageHandler)
	e.POST("/chats/:id/cancel", chatController.CancelMessageHandler) // Added cancel route

	// Provider routes
	e.GET("/providers", providerController.ListProvidersHandler)
	e.GET("/providers/new", providerController.ProviderFormHandler)
	e.POST("/providers", providerController.CreateProviderHandler)
	e.GET("/providers/:id/edit", providerController.ProviderFormHandler)
	e.PUT("/providers/:id", providerController.UpdateProviderHandler)
	e.DELETE("/providers/:id", providerController.DeleteProviderHandler)
	e.GET("/api/debug/providers", providerController.DebugProvidersHandler)
	e.POST("/api/debug/providers/reset", providerController.ResetProvidersHandler)
	e.GET("/api/providers/:id", providerController.GetProviderHandler)

	// Sidebar Partial Routes
	e.GET("/sidebar/chats", homeController.ChatsPartialHandler)
	e.GET("/sidebar/agents", homeController.AgentsPartialHandler)

	// Start server
	logger.Info("Starting HTTP server on :8080")
	if err := e.Start(":8080"); err != nil {
		logger.Fatal("Failed to start server", zap.Error(err))
	}
}

func basicAuthMiddleware(username, password string) echo.MiddlewareFunc {
	return middleware.BasicAuth(func(user, pass string, c echo.Context) (bool, error) {
		if user == username && pass == password {
			return true, nil
		}
		return false, nil
	})
}

func renderMarkdown(markdown string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := goldmark.New(goldmark.WithExtensions(gfmext.GFM)).Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}
