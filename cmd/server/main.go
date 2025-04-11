package main

import (
	"bytes"
	"context"
	"html/template"
	"strings"

	apicontrollers "aiagent/internal/api/controllers"
	"aiagent/internal/domain/services"
	"aiagent/internal/impl/config"
	"aiagent/internal/impl/database"
	"aiagent/internal/impl/repositories"
	"aiagent/internal/impl/tools"
	uicontrollers "aiagent/internal/ui/controllers"

	"github.com/dustin/go-humanize"
	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/yuin/goldmark"
	gfmext "github.com/yuin/goldmark/extension"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.uber.org/zap"

	_ "aiagent/docs" // Import the generated docs package
)

//	@title			AI Agent API
//	@version		1.0
//	@description	This is the API for the AI Agent application.
//	@termsOfService	http://swagger.io/terms/

//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io

//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html

// @host	localhost:8080
// @BasePath	/
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
	providerRepo, err := repositories.NewJSONProviderRepository("internal/impl/config")
	if err != nil {
		logger.Fatal("Failed to initialize provider repository", zap.Error(err))
	}

	toolFactory, err := tools.NewToolFactory()
	if err != nil {
		logger.Fatal("Failed to initialize tools", zap.Error(err))
	}

	toolRepo, err := repositories.NewToolRepository(db.Collection("tools"), toolFactory, logger)
	if err != nil {
		logger.Fatal("Failed to initialize tools", zap.Error(err))
	}

	providerService := services.NewProviderService(providerRepo, logger)
	agentService := services.NewAgentService(agentRepo, logger)
	toolService := services.NewToolService(toolRepo, logger)
	chatService := services.NewChatService(chatRepo, agentRepo, providerRepo, toolRepo, cfg, logger)

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
		"formatNumber": func(num int) string {
			return humanize.Comma(int64(num))
		},
		"compareUUIDtoObjectID": func(UUID string, objectID primitive.ObjectID) (bool, error) {
			// Convert the ObjectID to a byte slice
			bytes := objectID[:]

			// Pad the byte slice with 4 additional bytes to make it 16 bytes long
			paddedBytes := append(bytes, []byte{0, 0, 0, 0}...)

			// Create a UUID from the padded bytes
			u, err := uuid.FromBytes(paddedBytes)
			if err != nil {
				return false, err
			}

			// Convert the UUID to lowercase
			lowerUUID := strings.ToLower(UUID)

			logger.Debug("UUID", zap.String("UUID", lowerUUID), zap.String("ObjectID", u.String()))
			// Compare the UUID with the string representation of the ObjectID
			if lowerUUID[:12] == u.String()[:12] {
				return true, nil
			}
			return false, nil
		},
	}

	// Parse templates with custom function map
	tmpl, err := template.New("").Funcs(funcMap).ParseFiles(
		"internal/ui/templates/layout.html",
		"internal/ui/templates/header.html",
		"internal/ui/templates/sidebar.html",
		"internal/ui/templates/home.html",
		"internal/ui/templates/sidebar_chats.html",
		"internal/ui/templates/sidebar_agents.html",
		"internal/ui/templates/sidebar_tools.html",
		"internal/ui/templates/chat_form.html",
		"internal/ui/templates/agent_form.html",
		"internal/ui/templates/tool_form.html",
		"internal/ui/templates/chat.html",
		"internal/ui/templates/messages_partial.html",
		"internal/ui/templates/message_session_partial.html",
		"internal/ui/templates/provider_models_partial.html",
		"internal/ui/templates/providers_list_content.html",
	)
	if err != nil {
		logger.Fatal("Failed to parse templates", zap.Error(err))
	}

	// UI Controllers
	homeController := uicontrollers.NewHomeController(logger, tmpl, chatService, agentService, toolService)
	agentController := uicontrollers.NewAgentController(logger, tmpl, agentService, toolService, providerService)
	chatController := uicontrollers.NewChatController(logger, tmpl, chatService, agentService)
	toolController := uicontrollers.NewToolController(logger, tmpl, toolService, toolFactory)
	providerController := uicontrollers.NewProviderController(logger, tmpl, providerService)

	// API Controllers
	apiAgentController := apicontrollers.NewAgentController(logger, agentService)
	apiChatController := apicontrollers.NewChatController(logger, chatService)

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

	// Static file serving
	e.Static("/static", "internal/ui/static")

	// UI Routes
	homeController.RegisterRoutes(e)
	agentController.RegisterRoutes(e)
	chatController.RegisterRoutes(e)
	toolController.RegisterRoutes(e)
	providerController.RegisterRoutes(e)

	// API Routes
	api := e.Group("/api")
	apiAgentController.RegisterRoutes(api)
	apiChatController.RegisterRoutes(api)

	// Swagger route
	e.GET("/swagger/*", echoSwagger.WrapHandler)

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
