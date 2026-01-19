package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"slices"

	"github.com/drujensen/aiagent/internal/domain/interfaces"
	"github.com/drujensen/aiagent/internal/domain/services"
	"github.com/drujensen/aiagent/internal/impl/config"
	"github.com/drujensen/aiagent/internal/impl/database"
	"github.com/drujensen/aiagent/internal/impl/defaults"
	"github.com/drujensen/aiagent/internal/impl/modelsdev"
	repositoriesJson "github.com/drujensen/aiagent/internal/impl/repositories/json"
	repositoriesMongo "github.com/drujensen/aiagent/internal/impl/repositories/mongo"
	"github.com/drujensen/aiagent/internal/impl/tools"
	"github.com/drujensen/aiagent/internal/tui"
	"github.com/drujensen/aiagent/internal/ui"

	tea "github.com/charmbracelet/bubbletea"
	"go.uber.org/zap"
)

var (
	version = "unknown" // This should be set during build with -ldflags="-X main.version=1.0.0"
)

func main() {
	// Check version flag first
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		os.Exit(0)
	}

	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: aiagent [serve|tui|refresh] [--storage=type]\n")
		flag.PrintDefaults()
	}

	storage := flag.String("storage", "file", "Storage type: file or mongo")

	// Preserve the flags by not calling flag.Parse() yet
	flag.CommandLine.Parse([]string{})

	// Default mode is "tui"
	modeStr := "tui"

	// Check the first non-flag argument for the mode
	if len(os.Args) > 1 && os.Args[1] == "serve" {
		modeStr = "serve"
		os.Args = slices.Delete(os.Args, 0, 1)
	}

	if len(os.Args) > 1 && os.Args[1] == "tui" {
		modeStr = "tui"
		os.Args = slices.Delete(os.Args, 0, 1)
	}

	if len(os.Args) > 1 && os.Args[1] == "refresh" {
		modeStr = "refresh"
		os.Args = slices.Delete(os.Args, 0, 1)
	}

	// Parse the remaining arguments which are flags
	flag.Parse()

	if *storage != "file" && *storage != "mongo" {
		fmt.Fprintf(os.Stderr, "Invalid storage type: %s\n", *storage)
		flag.Usage()
		os.Exit(1)
	}

	logConfig := zap.NewDevelopmentConfig()
	logConfig.Level = zap.NewAtomicLevelAt(zap.DebugLevel)
	if modeStr == "tui" {
		mkdirErr := os.MkdirAll(".aiagent", 0755)
		if mkdirErr != nil {
			fmt.Fprintf(os.Stderr, "Failed to create .aiagent directory: %v\n", mkdirErr)
			os.Exit(1)
		}
		logConfig.OutputPaths = []string{".aiagent/aiagent.log"}
		logConfig.ErrorOutputPaths = []string{".aiagent/aiagent.log"}
	} else {
		logConfig.OutputPaths = []string{"stdout"}
		logConfig.ErrorOutputPaths = []string{"stderr"}
	}

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
	var modelRepo interfaces.ModelRepository
	var chatRepo interfaces.ChatRepository
	var providerRepo interfaces.ProviderRepository
	var toolRepo interfaces.ToolRepository

	dataDir, err := os.Getwd()
	if err != nil {
		logger.Fatal("Failed to get current directory", zap.Error(err))
	}

	// Initialize tool factory
	toolFactory, err := tools.NewToolFactory()
	if err != nil {
		logger.Fatal("Failed to initialize tool factory", zap.Error(err))
	}

	if *storage == "mongo" {
		db, err := database.NewMongoDB(cfg.MongoURI, "aiagent", logger)
		if err != nil {
			logger.Fatal("Failed to connect to MongoDB", zap.Error(err))
		}
		defer db.Disconnect(context.Background())

		// Initialize repositories
		agentRepo = repositoriesMongo.NewMongoAgentRepository(db.Collection("agents"))
		chatRepo = repositoriesMongo.NewMongoChatRepository(db.Collection("chats"))
		providerRepo = repositoriesMongo.NewMongoProviderRepository(db.Collection("providers"))
		// TODO: Implement MongoDB model repository
		modelRepo, err = repositoriesJson.NewJSONModelRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize model repository", zap.Error(err))
		}
		toolRepo, err = repositoriesMongo.NewToolRepository(db.Collection("tools"), toolFactory, logger)
		if err != nil {
			logger.Fatal("Failed to initialize tool repository", zap.Error(err))
		}
	} else {
		// Initialize JSON repositories
		providerRepo, err = repositoriesJson.NewJSONProviderRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize provider repository", zap.Error(err))
		}
		toolRepo, err = repositoriesJson.NewJSONToolRepository(dataDir, toolFactory, logger)
		if err != nil {
			logger.Fatal("Failed to initialize tool repository", zap.Error(err))
		}
		agentRepo, err = repositoriesJson.NewJSONAgentRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize agent repository", zap.Error(err))
		}
		modelRepo, err = repositoriesJson.NewJSONModelRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize model repository", zap.Error(err))
		}
		chatRepo, err = repositoriesJson.NewJSONChatRepository(dataDir)
		if err != nil {
			logger.Fatal("Failed to initialize chat repository", zap.Error(err))
		}
	}

	// Initialize default data
	if err := initializeDefaults(context.Background(), providerRepo, agentRepo, modelRepo, toolRepo, logger); err != nil {
		logger.Fatal("Failed to initialize defaults", zap.Error(err))
	}

	providerService := services.NewProviderService(providerRepo, logger)
	agentService := services.NewAgentService(agentRepo, logger)
	modelService := services.NewModelService(modelRepo, logger)
	toolService := services.NewToolService(toolRepo, logger)
	chatService := services.NewChatService(chatRepo, agentRepo, modelRepo, providerRepo, toolRepo, cfg, logger)

	// Create ModelRefreshService for refresh functionality
	modelsDevClient := modelsdev.NewModelsDevClient(logger)
	modelRefreshService := services.NewModelRefreshService(providerRepo, modelsDevClient, logger)

	if modeStr == "refresh" {
		// Create the ModelRefreshService
		modelsDevClient := modelsdev.NewModelsDevClient(logger)
		refreshService := services.NewModelRefreshService(providerRepo, modelsDevClient, logger)

		fmt.Println("Refreshing providers from models.dev...")
		if err := refreshService.RefreshAllProviders(context.Background()); err != nil {
			logger.Fatal("Failed to refresh providers", zap.Error(err))
		}
		fmt.Println("Provider refresh completed successfully!")
		return
	}

	if modeStr == "serve" {
		uiApp := ui.NewUI(chatService, agentService, modelService, toolService, providerService, modelRefreshService, logger)
		if err := uiApp.Run(); err != nil {
			logger.Fatal("UI failed", zap.Error(err))
		}
	} else {
		p := tea.NewProgram(tui.NewTUI(chatService, agentService, modelService, providerService, toolService), tea.WithAltScreen(), tea.WithMouseAllMotion())

		if _, err := p.Run(); err != nil {
			log.Fatal(err)
		}
	}
}

// initializeDefaults populates repositories with default data if they are empty.
func initializeDefaults(ctx context.Context, providerRepo interfaces.ProviderRepository, agentRepo interfaces.AgentRepository, modelRepo interfaces.ModelRepository, toolRepo interfaces.ToolRepository, logger *zap.Logger) error {
	// Check and populate providers
	providers, err := providerRepo.ListProviders(ctx)
	if err != nil {
		logger.Error("Failed to list providers", zap.Error(err))
		return err
	}
	if len(providers) == 0 {
		for _, provider := range defaults.DefaultProviders() {
			if err := providerRepo.CreateProvider(ctx, provider); err != nil {
				logger.Error("Failed to create default provider", zap.String("provider", provider.Name), zap.Error(err))
				return err
			}
		}
		logger.Info("Initialized providers with default data")
	}

	// Check and populate agents
	agents, err := agentRepo.ListAgents(ctx)
	if err != nil {
		logger.Error("Failed to list agents", zap.Error(err))
		return err
	}
	if len(agents) == 0 {
		for _, agent := range defaults.DefaultAgents() {
			if err := agentRepo.CreateAgent(ctx, &agent); err != nil {
				logger.Error("Failed to create default agent", zap.String("agent", agent.Name), zap.Error(err))
				return err
			}
		}
		logger.Info("Initialized agents with default data")
	}

	// Check and populate tools
	tools, err := toolRepo.ListToolData(ctx)
	if err != nil {
		logger.Error("Failed to list tools", zap.Error(err))
		return err
	}
	if len(tools) == 0 {
		for _, tool := range defaults.DefaultTools() {
			if err := toolRepo.CreateToolData(ctx, tool); err != nil {
				logger.Error("Failed to create default tool", zap.String("tool", tool.Name), zap.Error(err))
				return err
			}
		}
		logger.Info("Initialized tools with default data")
	}

	// Check and populate models
	models, err := modelRepo.ListModels(ctx)
	if err != nil {
		logger.Error("Failed to list models", zap.Error(err))
		return err
	}
	if len(models) == 0 {
		for _, model := range defaults.DefaultModels() {
			if err := modelRepo.CreateModel(ctx, model); err != nil {
				logger.Error("Failed to create default model", zap.String("model", model.Name), zap.Error(err))
				return err
			}
		}
		logger.Info("Initialized models with default data")
	}

	return nil
}
