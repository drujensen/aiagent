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
	storage := flag.String("storage", "file", "Storage type: file or mongo")
	flag.Parse()

	modeStr := "console"
	if len(flag.Args()) > 0 {
		if len(flag.Args()) > 1 {
			fmt.Fprintf(os.Stderr, "Too many arguments\n")
			flag.Usage()
			os.Exit(1)
		}
		arg := flag.Args()[0]
		if arg == "serve" {
			modeStr = "serve"
		} else {
			fmt.Fprintf(os.Stderr, "Invalid command: %s\n", arg)
			flag.Usage()
			os.Exit(1)
		}
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

	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	logger, err := logConfig.Build()
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

	dataDir, err := os.Getwd()
	if err != nil {
		logger.Fatal("Failed to get current directory", zap.Error(err))
	}

	if *storage == "mongo" {
		db, err := database.NewMongoDB(cfg.MongoURI, "aiagent", logger)
		if err != nil {
			logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
		}
		defer db.Disconnect(context.Background())

		agentRepo = repositoriesMongo.NewMongoAgentRepository(db.Collection("agents"))
		chatRepo = repositoriesMongo.NewMongoChatRepository(db.Collection("chats"))
		providerRepo, err = repositoriesJson.NewJSONProviderRepository(dataDir)
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

	if modeStr == "serve" {
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
					{Name: "grok-3-beta", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 131072},
					{Name: "grok-3-mini-beta", InputPricePerMille: 0.30, OutputPricePerMille: 0.50, ContextWindow: 131072},
					{Name: "grok-2", InputPricePerMille: 2.50, OutputPricePerMille: 7.50, ContextWindow: 131072},
				},
			},
			{
				ID:         "D2BB79D4-C11C-407A-AF9D-9713524BB3BF",
				Name:       "OpenAI",
				Type:       "openai",
				BaseURL:    "https://api.openai.com",
				APIKeyName: "OPENAI_API_KEY",
				Models: []entities.ModelPricing{
					{Name: "o1", InputPricePerMille: 15.00, OutputPricePerMille: 60.00, ContextWindow: 200000},
					{Name: "o3-mini", InputPricePerMille: 1.10, OutputPricePerMille: 4.40, ContextWindow: 200000},
					{Name: "gpt-4o", InputPricePerMille: 2.50, OutputPricePerMille: 10.00, ContextWindow: 128000},
					{Name: "gpt-4o-mini", InputPricePerMille: 0.15, OutputPricePerMille: 0.60, ContextWindow: 128000},
				},
			},
			{
				ID:         "28451B8D-1937-422A-BA93-9795204EC5A5",
				Name:       "Anthropic",
				Type:       "anthropic",
				BaseURL:    "https://api.anthropic.com",
				APIKeyName: "ANTHROPIC_API_KEY",
				Models: []entities.ModelPricing{
					{Name: "claude-3-opus-20240229", InputPricePerMille: 15.00, OutputPricePerMille: 75.00, ContextWindow: 200000},
					{Name: "claude-3-7-sonnet-20250219", InputPricePerMille: 3.00, OutputPricePerMille: 15.00, ContextWindow: 200000},
					{Name: "claude-3-haiku-20240307", InputPricePerMille: 0.25, OutputPricePerMille: 1.25, ContextWindow: 200000},
				},
			},
			{
				ID:         "2BD2B8A5-5A2A-439B-8D02-C6BE34705011",
				Name:       "Google",
				Type:       "google",
				BaseURL:    "https://generativelanguage.googleapis.com",
				APIKeyName: "GEMINI_API_KEY",
				Models: []entities.ModelPricing{
					{Name: "gemini-2.5-pro-preview-03-25", InputPricePerMille: 2.50, OutputPricePerMille: 15.00, ContextWindow: 1000000},
					{Name: "gemini-2.0-flash", InputPricePerMille: 0.10, OutputPricePerMille: 0.40, ContextWindow: 1000000},
					{Name: "gemini-2.0-flash-lite", InputPricePerMille: 0.075, OutputPricePerMille: 0.30, ContextWindow: 1000000},
					{Name: "gemma-3-27b-it", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000},
				},
			},
			{
				ID:         "276F9470-664F-4402-98E0-755C342ADFC4",
				Name:       "DeepSeek",
				Type:       "deepseek",
				BaseURL:    "https://api.deepseek.com",
				APIKeyName: "DEEPSEEK_API_KEY",
				Models: []entities.ModelPricing{
					{Name: "deepseek-reasoner", InputPricePerMille: 0.55, OutputPricePerMille: 2.19, ContextWindow: 64000},
					{Name: "deepseek-chat", InputPricePerMille: 0.07, OutputPricePerMille: 1.10, ContextWindow: 64000},
				},
			},
			{
				ID:         "8F2CC161-E463-43B1-9656-8E484A0D7709",
				Name:       "Together",
				Type:       "together",
				BaseURL:    "https://api.together.xyz",
				APIKeyName: "TOGETHER_API_KEY",
				Models: []entities.ModelPricing{
					{Name: "meta-llama/Llama-4-Maverick-17B-128E-Instruct-FP8", InputPricePerMille: 0.27, OutputPricePerMille: 0.85, ContextWindow: 131072},
					{Name: "meta-llama/Llama-4-Scout-17B-16E-Instruct", InputPricePerMille: 0.18, OutputPricePerMille: 0.59, ContextWindow: 131072},
					{Name: "deepseek-ai/DeepSeek-R1-Distill-Llama-70B-free", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 131072},
				},
			},
			{
				ID:         "CFA9E279-2CD3-4929-A92E-EC4584DC5089",
				Name:       "Groq",
				Type:       "groq",
				BaseURL:    "https://api.groq.com",
				APIKeyName: "GROQ_API_KEY",
				Models: []entities.ModelPricing{
					{Name: "llama-3.3-70b-versatile", InputPricePerMille: 0.59, OutputPricePerMille: 0.79, ContextWindow: 128000},
					{Name: "meta-llama/llama-4-maverick-17b-128e-instruct", InputPricePerMille: 0.27, OutputPricePerMille: 0.85, ContextWindow: 131072},
					{Name: "meta-llama/llama-4-scout-17b-16e-instruct", InputPricePerMille: 0.11, OutputPricePerMille: 0.34, ContextWindow: 131072},
					{Name: "deepseek-r1-distill-llama-70b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 128000},
				},
			},
			{
				ID:         "3B369D62-BB4E-4B4F-8C75-219796E9521A",
				Name:       "Ollama",
				Type:       "ollama",
				BaseURL:    "http://localhost:11434",
				APIKeyName: "LOCAL_API_KEY",
				Models: []entities.ModelPricing{
					{Name: "llama3.1:8b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
					{Name: "qwen2.5-coder:14b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
					{Name: "mistral-nemo:12b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
					{Name: "cogito:14b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
					{Name: "gemma:12b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
					{Name: "deepcoder:14b", InputPricePerMille: 0.00, OutputPricePerMille: 0.00, ContextWindow: 8192},
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
		localMaxTokens := 4096
		localContextWindow := 8192
		defaultAgents := []entities.Agent{
			{
				ID:              "1A3F3DCB-255D-46B3-A4F4-E2E118FBA82B",
				Name:            "Grok",
				ProviderID:      "820FE148-851B-4995-81E5-C6DB2E5E5270",
				ProviderType:    "xai",
				Endpoint:        "https://api.x.ai",
				Model:           "grok-3-mini",
				APIKey:          "#{XAI_API_KEY}#",
				Temperature:     &temperature,
				MaxTokens:       &maxTokens,
				ContextWindow:   &contextWindow,
				ReasoningEffort: "high",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "7F1C8EDF-7899-4691-997C-421795719EB3",
				Name:            "GPT",
				ProviderID:      "D2BB79D4-C11C-407A-AF9D-9713524BB3BF",
				ProviderType:    "openai",
				Endpoint:        "https://api.openai.com",
				Model:           "gpt-4o-mini",
				APIKey:          "#{OPENAI_API_KEY}#",
				Temperature:     &temperature,
				MaxTokens:       &maxTokens,
				ContextWindow:   &contextWindow,
				ReasoningEffort: "",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "65DD6A7E-992E-4603-AFB1-F6F9314DFA52",
				Name:            "Claude",
				ProviderID:      "28451B8D-1937-422A-BA93-9795204EC5A5",
				ProviderType:    "anthropic",
				Endpoint:        "https://api.anthropic.com",
				Model:           "claude-3-haiku-20240307",
				APIKey:          "#{ANTHROPIC_API_KEY}#",
				Temperature:     &temperature,
				MaxTokens:       &maxTokens,
				ContextWindow:   &contextWindow,
				ReasoningEffort: "",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "B64CA989-A4E6-4870-9C2D-AF9848C98EF7",
				Name:            "Gemini",
				ProviderID:      "2BD2B8A5-5A2A-439B-8D02-C6BE34705011",
				ProviderType:    "google",
				Endpoint:        "https://generativelanguage.googleapis.com",
				Model:           "gemini-2.0-flash",
				APIKey:          "#{GEMINI_API_KEY}#",
				Temperature:     &temperature,
				MaxTokens:       &maxTokens,
				ContextWindow:   &contextWindow,
				ReasoningEffort: "",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "B9A9C0F4-52F4-4458-9E69-6C7C16F1648B",
				Name:            "Qwen",
				ProviderID:      "3B369D62-BB4E-4B4F-8C75-219796E9521A",
				ProviderType:    "ollama",
				Endpoint:        "http://localhost:11434",
				Model:           "qwen3",
				APIKey:          "n/a",
				Temperature:     &temperature,
				MaxTokens:       &localMaxTokens,
				ContextWindow:   &localContextWindow,
				ReasoningEffort: "",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "B9A9C0F4-52F4-4458-9E69-6C7C16F1648B",
				Name:            "Cogito",
				ProviderID:      "3B369D62-BB4E-4B4F-8C75-219796E9521A",
				ProviderType:    "ollama",
				Endpoint:        "http://localhost:11434",
				Model:           "cogito",
				APIKey:          "n/a",
				Temperature:     &temperature,
				MaxTokens:       &localMaxTokens,
				ContextWindow:   &localContextWindow,
				ReasoningEffort: "",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:              "B9A9C0F4-52F4-4458-9E69-6C7C16F1648B",
				Name:            "Llama",
				ProviderID:      "3B369D62-BB4E-4B4F-8C75-219796E9521A",
				ProviderType:    "ollama",
				Endpoint:        "http://localhost:11434",
				Model:           "llama3.2",
				APIKey:          "n/a",
				Temperature:     &temperature,
				MaxTokens:       &localMaxTokens,
				ContextWindow:   &localContextWindow,
				ReasoningEffort: "",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
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
				Description:   "This tool provides you file system operations including reading, writing, editing, searching, and managing files and directories. The workspace will be prepended to any directories or files specified.",
				Configuration: map[string]string{},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            "501A9EC8-633A-4BD2-91BF-8744B7DC34EC",
				ToolType:      "Search",
				Name:          "Search",
				Description:   "This tool Searches the web using the Tavily API.",
				Configuration: map[string]string{"tavily_api_key": "#{TAVILY_API_KEY}#"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			// Add other tools as specified in your JSON
			{
				ID:            "AE3E4944-253D-4188-BEB0-F370A6F9DC6F",
				ToolType:      "Bash",
				Name:          "Bash",
				Description:   "This tool executes a bash command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory.",
				Configuration: map[string]string{},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            "4EA3F4A2-EFCD-4E9A-A5F8-4DFFAFB018E7",
				ToolType:      "Process",
				Name:          "Git",
				Description:   "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{"command": "git"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            "8C4E1573-59D9-463B-AF5F-1EA7620F469D",
				ToolType:      "Process",
				Name:          "Go",
				Description:   "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{"command": "go"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            "50A77E90-D6D3-410C-A7B4-6A3E5E58253E",
				ToolType:      "Process",
				Name:          "Python",
				Description:   "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{"command": "python"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            "89637725-6050-44BA-B839-F41D1B6067A7",
				ToolType:      "Process",
				Name:          "Grep",
				Description:   "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{"command": "grep"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
			{
				ID:            "F3C1455F-5E89-40F2-8E81-53AFAB096E9E",
				ToolType:      "Process",
				Name:          "Find",
				Description:   "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{"command": "find"},
				CreatedAt:     time.Now(),
				UpdatedAt:     time.Now(),
			},
		}
		data, _ := json.MarshalIndent(defaultTools, "", "  ")
		os.WriteFile(toolsPath, data, 0644)
	}
}
