package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"aiagent/internal/cli"
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/services"
	"aiagent/internal/impl/config"
	repositories "aiagent/internal/impl/repositories/json"
	"aiagent/internal/impl/tools"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

func main() {
	log := zap.NewProductionConfig()
	log.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
	logger, err := log.Build()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize logger: %v\n", err)
		os.Exit(1)
	}
	defer logger.Sync()

	dataDir, err := os.Getwd()
	if err != nil {
		logger.Fatal("failed to get current directory", zap.Error(err))
		os.Exit(1)
	}

	cfg, err := config.InitConfig()
	if err != nil {
		logger.Fatal("Failed to load config", zap.Error(err))
		os.Exit(1)
	}

	// Initialize JSON repositories
	providerRepo, err := repositories.NewJSONProviderRepository(dataDir)
	if err != nil {
		logger.Fatal("failed to initialize provider repository", zap.Error(err))
		os.Exit(1)
	}

	toolFactory, err := tools.NewToolFactory()
	if err != nil {
		logger.Fatal("failed to initialize tool factory", zap.Error(err))
		os.Exit(1)
	}

	toolRepo, err := repositories.NewJSONToolRepository(dataDir, toolFactory, logger)
	if err != nil {
		logger.Fatal("failed to initialize tool repository", zap.Error(err))
		os.Exit(1)
	}

	agentRepo, err := repositories.NewJSONAgentRepository(dataDir)
	if err != nil {
		logger.Fatal("failed to initialize agent repository", zap.Error(err))
		os.Exit(1)
	}

	chatRepo, err := repositories.NewJSONChatRepository(dataDir)
	if err != nil {
		logger.Fatal("failed to initialize chat repository", zap.Error(err))
		os.Exit(1)
	}

	chatService := services.NewChatService(chatRepo, agentRepo, providerRepo, toolRepo, cfg, logger)
	agentService := services.NewAgentService(agentRepo, logger)
	toolService := services.NewToolService(toolRepo, logger)

	// Initialize and run the CLI
	cli := cli.NewCLI(chatService, agentService, toolService, logger)
	if err := cli.Run(context.Background()); err != nil {
		logger.Fatal("CLI failed", zap.Error(err))
	}
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

	// Initialize agents.json
	temperature := 1.0
	maxTokens := 8192
	contextWindow := 131072
	agentsPath := filepath.Join(aiagentDir, "agents.json")
	if _, err := os.Stat(agentsPath); os.IsNotExist(err) {
		defaultAgents := []*entities.Agent{
			{
				ID:           "1A3F3DCB-255D-46B3-A4F4-E2E118FBA82B",
				Name:         "Grok",
				ProviderID:   "820FE148-851B-4995-81E5-C6DB2E5E5270",
				ProviderType: "xai",
				Endpoint:     "https://api.x.ai",
				Model:        "grok-3-mini-beta",
				APIKey:       "#{XAI_API_KEY}#",
				SystemPrompt: `### Introduction and Role

You are an interactive CLI tool that helps users with software engineering tasks. Use the instructions below and the tools available to you to assist the user.

### Memory

If the current working directory contains a AIAGENT.md file, it is added to context for:

1. Storing bash commands (e.g., build, test).
2. Recording code style preferences.
3. Maintaining codebase information.

Proactively ask users to add commands or preferences to AIAGENT.md for future reference.

### Tone and Style

- Be concise, direct, and to the point.
- Use GitHub-flavored Markdown for formatting.
- Output text for user communication; use tools only for tasks.
- If unable to help, offer alternatives in 1-2 sentences without explanations.
- Minimize tokens: Respond in 1-3 sentences or a short paragraph, fewer than 4 lines unless detailed.
- Avoid unnecessary preamble or postamble (e.g., no "The answer is..." unless asked).
- Examples of concise responses:
    - User: "2 + 2" → Assistant: "4"
    - User: "Is 11 a prime number?" → Assistant: "true"

### Proactiveness

Be proactive only when directly asked. Balance actions with user confirmation. Do not explain code changes unless requested.

### Synthetic Messages

Ignore system-added messages like "[Request interrupted by user]"; do not generate them.

### Following Conventions

- Mimic existing code styles, libraries, and patterns.
- Verify library availability before use.
- Follow security best practices (e.g., never commit secrets).
- Do not add comments to code unless requested.

### Doing Tasks

For software engineering tasks (e.g., bugs, features):

1. Use search tools to understand the codebase.
2. Implement using available tools.
3. Verify with tests; check for testing commands.
4. Run lint and typecheck commands if available; suggest adding to AIAGENT.md.

- Never commit changes unless explicitly asked.

### Tool Usage Policy

- Prefer Agent for open-ended searches.
- Make independent tool calls in the same block.
- Be concise in responses.`,
				Temperature:     &temperature,
				MaxTokens:       &maxTokens,
				ContextWindow:   &contextWindow,
				ReasoningEffort: "medium",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
			},
			{
				ID:           "7F1C8EDF-7899-4691-997C-421795719EB3",
				Name:         "GPT",
				ProviderID:   "D2BB79D4-C11C-407A-AF9D-9713524BB3BF",
				ProviderType: "openai",
				Endpoint:     "https://api.openai.com",
				Model:        "gpt-4o-mini",
				APIKey:       "#{OPENAI_API_KEY}#",
				SystemPrompt: `### Introduction and Role

You are an interactive CLI tool that helps users with software engineering tasks. Use the instructions below and the tools available to you to assist the user.

### Memory

If the current working directory contains a AIAGENT.md file, it is added to context for:

1. Storing bash commands (e.g., build, test).
2. Recording code style preferences.
3. Maintaining codebase information.

Proactively ask users to add commands or preferences to AIAGENT.md for future reference.

### Tone and Style

- Be concise, direct, and to the point.
- Use GitHub-flavored Markdown for formatting.
- Output text for user communication; use tools only for tasks.
- If unable to help, offer alternatives in 1-2 sentences without explanations.
- Minimize tokens: Respond in 1-3 sentences or a short paragraph, fewer than 4 lines unless detailed.
- Avoid unnecessary preamble or postamble (e.g., no "The answer is..." unless asked).
- Examples of concise responses:
    - User: "2 + 2" → Assistant: "4"
    - User: "Is 11 a prime number?" → Assistant: "true"

### Proactiveness

Be proactive only when directly asked. Balance actions with user confirmation. Do not explain code changes unless requested.

### Synthetic Messages

Ignore system-added messages like "[Request interrupted by user]"; do not generate them.

### Following Conventions

- Mimic existing code styles, libraries, and patterns.
- Verify library availability before use.
- Follow security best practices (e.g., never commit secrets).
- Do not add comments to code unless requested.

### Doing Tasks

For software engineering tasks (e.g., bugs, features):

1. Use search tools to understand the codebase.
2. Implement using available tools.
3. Verify with tests; check for testing commands.
4. Run lint and typecheck commands if available; suggest adding to AIAGENT.md.

- Never commit changes unless explicitly asked.

### Tool Usage Policy

- Prefer Agent for open-ended searches.
- Make independent tool calls in the same block.
- Be concise in responses.`,
				Temperature:     &temperature,
				MaxTokens:       &maxTokens,
				ContextWindow:   &contextWindow,
				ReasoningEffort: "",
				Tools:           []string{"File", "Search", "Bash", "Git", "Go", "Python", "Grep", "Find"},
				CreatedAt:       time.Now(),
				UpdatedAt:       time.Now(),
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
				ID:          "4EA3F4A2-EFCD-4E9A-A5F8-4DFFAFB018E7",
				ToolType:    "Process",
				Name:        "Git",
				Description: "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{
					"command": "git",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "8C4E1573-59D9-463B-AF5F-1EA7620F469D",
				ToolType:    "Process",
				Name:        "Go",
				Description: "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{
					"command": "go",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "50A77E90-D6D3-410C-A7B4-6A3E5E58253E",
				ToolType:    "Process",
				Name:        "Python",
				Description: "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{
					"command": "python",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "89637725-6050-44BA-B839-F41D1B6067A7",
				ToolType:    "Process",
				Name:        "Grep",
				Description: "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{
					"command": "grep",
				},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "F3C1455F-5E89-40F2-8E81-53AFAB096E9E",
				ToolType:    "Process",
				Name:        "Find",
				Description: "This tool executes a configured CLI command with support for background processes, timeouts, and full output.\n\nThe command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.",
				Configuration: map[string]string{
					"command": "find",
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
