package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/google/uuid"

	apicontrollers "aiagent/internal/api/controllers"
	"aiagent/internal/cli"
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/domain/services"
	"aiagent/internal/impl/config"
	"aiagent/internal/impl/database"
	repositoriesJson "aiagent/internal/impl/repositories/json"
	repositoriesMongo "aiagent/internal/impl/repositories/mongo"
	"aiagent/internal/impl/tools"
	uiapicontrollers "aiagent/internal/ui/controllers"

	"github.com/dustin/go-humanize"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	echoSwagger "github.com/swaggo/echo-swagger"
	"github.com/yuin/goldmark"
	gfmext "github.com/yuin/goldmark/extension"
	"go.uber.org/zap"

	_ "aiagent/docs"
)

func main() {
	mode := flag.String("mode", "console", "Application mode: console or server")
	storage := flag.String("storage", "file", "Storage type: file or mongo")
	flag.Parse()
	if *mode == "console" || *mode == "server" {
	} else {
		flag.Usage()
		os.Exit(1)
	}
	if *storage == "file" || *storage == "mongo" {
	} else {
		flag.Usage()
		os.Exit(1)
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aiagent [flags]\n")
		flag.PrintDefaults()
	}

	logger, err := zap.NewDevelopment()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	cfg, err := config.InitConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
	}

	var agentRepo interfaces.AgentRepository
	var chatRepo interfaces.ChatRepository
	var providerRepo interfaces.ProviderRepository
	var toolRepo interfaces.ToolRepository

	if *storage == "mongo" {
		db, err := database.NewMongoDB(cfg.MongoURI, "aiagent", logger)
		if err != nil {
			logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
		}
		defer db.Disconnect(context.Background())

		agentRepo = repositoriesMongo.NewMongoAgentRepository(db.Collection("agents"))
		chatRepo = repositoriesMongo.NewMongoChatRepository(db.Collection("chats"))
		providerRepo, err = repositoriesJson.NewJSONProviderRepository("internal/impl/config")
		if err != nil {
			logger.Fatal("Failed to initialize provider repository", zap.Error(err))
		}
		toolFactory, err := tools.NewToolFactory()
		if err != nil {
			logger.Fatal("Failed to initialize tool factory", zap.Error(err))
		}
		toolRepo, err = repositoriesMongo.NewToolRepository(db.Collection("tools"), toolFactory, logger)
		if err != nil {
			logger.Fatal("Failed to initialize tools", zap.Error(err))
		}
	} else {
		dataDir, err := os.Getwd()
		if err != nil {
			logger.Fatal("Failed to get current directory", zap.Error(err))
		}

		providerRepo, err = repositoriesJson.NewJSONProviderRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize provider repository", zap.Error(err))
		}

		toolFactory, err := tools.NewToolFactory()
		if err != nil {
			logger.Fatal("Failed to initialize tool factory", zap.Error(err))
		}

		toolRepo, err = repositoriesJson.NewJSONToolRepository(dataDir, toolFactory, logger)
		if err != nil {
			logger.Fatal("Failed to initialize tool repository", zap.Error(err))
		}

		agentRepo, err = repositoriesJson.NewJSONAgentRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize agent repository", zap.Error(err))
		}

		chatRepo, err = repositoriesJson.NewJSONChatRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize chat repository", zap.Error(err))
		}
	}

	providerService := services.NewProviderService(providerRepo, logger)
	agentService := services.NewAgentService(agentRepo, logger)
	toolService := services.NewToolService(toolRepo, logger)
	chatService := services.NewChatService(chatRepo, agentRepo, providerRepo, toolRepo, cfg, logger)

	if *mode == "server" {
		funcMap := template.FuncMap{
			"renderMarkdown": renderMarkdown,
			"inArray": func(value string, array []string) bool {
				return slices.Contains(array, value)
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
		}

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
			"internal/ui/templates/chat_cost_partial.html",
			"internal/ui/templates/message_controls.html",
		)
		if err != nil {
			logger.Fatal("Failed to parse templates", zap.Error(err))
		}

		homeController := uiapicontrollers.NewHomeController(logger, tmpl, chatService, agentService, toolService)
		agentController := uiapicontrollers.NewAgentController(logger, tmpl, agentService, toolService, providerService)
		chatController := uiapicontrollers.NewChatController(logger, tmpl, chatService, agentService)
		toolFactory, err := tools.NewToolFactory()
		if err != nil {
			logger.Fatal("Failed to initialize tool factory", zap.Error(err))
		}
		toolController := uiapicontrollers.NewToolController(logger, tmpl, toolService, toolFactory)
		providerController := uiapicontrollers.NewProviderController(logger, tmpl, providerService)

		apiAgentController := apicontrollers.NewAgentController(logger, agentService)
		apiChatController := apicontrollers.NewChatController(logger, chatService)

		e := echo.New()
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

		e.Use(func(next echo.HandlerFunc) echo.HandlerFunc {
			return func(c echo.Context) error {
				c.Response().Header().Set("Content-Language", "en")
				return next(c)
			}
		})

		e.Static("/static", "internal/ui/static")

		homeController.RegisterRoutes(e)
		agentController.RegisterRoutes(e)
		chatController.RegisterRoutes(e)
		toolController.RegisterRoutes(e)
		providerController.RegisterRoutes(e)

		api := e.Group("/api")
		apiAgentController.RegisterRoutes(api)
		apiChatController.RegisterRoutes(api)

		e.GET("/swagger/*", echoSwagger.WrapHandler)

		logger.Info("Starting HTTP server on :8080")
		if err := e.Start(":8080"); err != nil {
			logger.Fatal("Failed to start server", zap.Error(err))
		}
	} else {
		cliApp := cli.NewCLI(chatService, agentService, toolService, logger)
		if err := cliApp.Run(context.Background()); err != nil {
			logger.Fatal("CLI failed", zap.Error(err))
		}
	}
}

func renderMarkdown(markdown string) (template.HTML, error) {
	var buf bytes.Buffer
	if err := goldmark.New(goldmark.WithExtensions(gfmext.GFM)).Convert([]byte(markdown), &buf); err != nil {
		return "", err
	}
	return template.HTML(buf.String()), nil
}

func init() {
	dataDir, err := os.Getwd()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to get current directory: %v\n", err)
		os.Exit(1)
	}

	aiagentDir := filepath.Join(dataDir, ".aiagent")
	if err := os.MkdirAll(aiagentDir, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to create .aiagent directory: %v\n", err)
		os.Exit(1)
	}

	providersPath := filepath.Join(aiagentDir, "providers.json")
	if _, err := os.Stat(providersPath); os.IsNotExist(err) {
		defaultProviders := []*entities.Provider{
			{
				ID:         "820FE148-851B-4995-81E5-C6DB2E5E5270",
				Name:       "X.AI",
				Type:       "xai",
				BaseURL:    "https://api.x.ai",
				APIKeyName: "XAI_API_KEY",
				Models: []entities.ModelPricing{
					{
						Name:                "grok-3-mini-beta",
						InputPricePerMille:  0.30,
						OutputPricePerMille: 0.50,
						ContextWindow:       131072,
					},
				},
			},
			{
				ID:         "D2BB79D4-C11C-407A-AF9D-9713524BB3BF",
				Name:       "OpenAI",
				Type:       "openai",
				BaseURL:    "https://api.openai.com",
				APIKeyName: "OPENAI_API_KEY",
				Models: []entities.ModelPricing{
					{
						Name:                "gpt-4o-mini",
						InputPricePerMille:  0.15,
						OutputPricePerMille: 0.60,
						ContextWindow:       128000,
					},
				},
			},
		}
		data, _ := json.MarshalIndent(defaultProviders, "", "  ")
		os.WriteFile(providersPath, data, 0644)
	}

	agentsPath := filepath.Join(aiagentDir, "agents.json")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		temperature := 1.0
		maxTokens := 8192
		contextWindow := 131072
		defaultAgents := []*entities.Agent{
			{
				ID:              "1A3F3DCB-255D-46B3-A4F4-E2E118FBA82B",
				Name:            "Grok",
				ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
				ProviderType:    "xai",
				Endpoint:        "https://api.x.ai",
				Model:           "grok-3-mini-beta",
				APIKey:          "#{XAI_API_KEY}#",
				SystemPrompt:    `...`, // Replace with the actual system prompt content
				Temperature:     &temperature,
				MaxTokens:       &maxTokens,
				ContextWindow:   &contextWindow,
				ReasoningEffort: "medium",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			// Add more agents as needed
		}
		data, _ := json.MarshalIndent(defaultAgents, "", "  ")
		os.WriteFile(agentsPath, data, 0644)
	}

	toolsPath := filepath.Join(aiagentDir, "tools.json")
	if _, err := os.Stat(toolsPath); os.IsNotExist(err) {
		defaultTools := []*entities.ToolData{
			{
				ID:            "436F6B15-D874-4498-A243-A4711D09FB66",
				ToolType:      "File",
				Name:          "File",
				Description:   "This tool provides file system operations.",
				Configuration: map[string]string{},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			// Add more tools as needed
		}
		data, _ := json.MarshalIndent(defaultTools, "", "  ")
		os.WriteFile(toolsPath, data, 0644)
	}

	chatsPath := filepath.Join(aiagentDir, "chats.json")
	if _, err := os.Stat(chatsPath); os.IsNotExist(err) {
		defaultChats := []*entities.Chat{
			{
				ID:        uuid.New().String(),
				AgentID:   "1A3F3DCB-255D-46B3-A4F4-E2E118FBA82B",
				Messages:  []entities.Message{},
				Usage:     &entities.ChatUsage{},
				Active:    true,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		data, _ := json.MarshalIndent(defaultChats, "", "  ")
		os.WriteFile(chatsPath, data, 0644)
	}
}
