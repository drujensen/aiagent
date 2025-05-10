package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
	repositories "aiagent/internal/impl/repositories/json"
	"aiagent/internal/impl/tools"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

type App struct {
	ProviderRepo interfaces.ProviderRepository
	AgentRepo    interfaces.AgentRepository
	ChatRepo     interfaces.ChatRepository
	ToolRepo     interfaces.ToolRepository
	logger       *zap.Logger
}

func main() {
	logger, err := zap.NewProduction()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	// Initialize application with JSON repositories
	app, err := initializeApp(logger)
	if err != nil {
		logger.Fatal("Failed to initialize application", zap.Error(err))
	}

	// TODO: Implement CLI commands (e.g., using cobra or flag)
	fmt.Println("AIAgent CLI started. Use commands to interact with chat sessions.", app)
}

func initializeApp(logger *zap.Logger) (*App, error) {
	dataDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("failed to get current directory: %v", err)
	}

	// Initialize JSON repositories
	providerRepo, err := repositories.NewJSONProviderRepository(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize provider repository: %v", err)
	}

	toolFactory, err := tools.NewToolFactory()
	toolRepo, err := repositories.NewJSONToolRepository(dataDir, toolFactory, logger)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tool repository: %v", err)
	}

	agentRepo, err := repositories.NewJSONAgentRepository(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize agent repository: %v", err)
	}

	chatRepo, err := repositories.NewJSONChatRepository(dataDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize chat repository: %v", err)
	}

	return &App{
		ProviderRepo: providerRepo,
		AgentRepo:    agentRepo,
		ChatRepo:     chatRepo,
		ToolRepo:     toolRepo,
		logger:       logger,
	}, nil
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

	// Initialize providers.json
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
		}
		data, _ := json.MarshalIndent(defaultProviders, "", "  ")
		os.WriteFile(providersPath, data, 0644)
	}

	// Initialize agents.json
	agentsPath := filepath.Join(aiagentDir, "agents.json")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		defaultAgents := []*entities.Agent{
			{
				ID:         "1A3F3DCB-255D-46B3-A4F4-E2E118FBA82B",
				Name:       "Grok",
				ProviderID: "820FE148-851B-4995-81E5-C6DB2E5E5270",
				Model:      "grok-3-mini-beta",
				Tools:      []string{"File", "Search"},
				CreatedAt:  time.Now(),
				UpdatedAt:  time.Now(),
			},
		}
		data, _ := json.MarshalIndent(defaultAgents, "", "  ")
		os.WriteFile(agentsPath, data, 0644)
	}

	// Initialize tools.json
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
				ID:          "501A9EC8-633A-4BD2-91BF-8744B7DC34EC",
				ToolType:    "Search",
				Name:        "Search",
				Description: "This tool Searches the web using the Tavily API.",
				Configuration: map[string]string{
					"tavily_api_key": "#{TAVILY_API_KEY}#",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		data, _ := json.MarshalIndent(defaultTools, "", "  ")
		os.WriteFile(toolsPath, data, 0644)
	}

	// Initialize chats.json
	chatsPath := filepath.Join(aiagentDir, "chats.json")
	if _, err := os.Stat(chatsPath); os.IsNotExist(err) {
		defaultChats := []*entities.Chat{
			{
				ID:        uuid.New().String(),
				AgentID:   "1A3F3DCB-255D-46B3-A4F4-E2E118FBA82B",
				Messages:  []entities.Message{},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		}
		data, _ := json.MarshalIndent(defaultChats, "", "  ")
		os.WriteFile(chatsPath, data, 0644)
	}
}
