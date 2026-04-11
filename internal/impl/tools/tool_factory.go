package tools

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"
	"github.com/drujensen/aiagent/internal/domain/services"

	"go.uber.org/zap"
)

type ToolFactoryEntry struct {
	Name        string
	Description string
	ConfigKeys  []string
	Factory     func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool
}

type ToolFactory struct {
	toolFactories map[string]*ToolFactoryEntry
	chatService   services.ChatService
	agentService  services.AgentService
	modelService  services.ModelService
}

// SetServices wires the application services into the factory so that
// service-dependent tools (e.g. AgentTool) can access them at execute time.
// Call this in main after all services have been constructed.
func (t *ToolFactory) SetServices(chatService services.ChatService, agentService services.AgentService, modelService services.ModelService) {
	t.chatService = chatService
	t.agentService = agentService
	t.modelService = modelService
}

func (t *ToolFactory) GetChatService() services.ChatService   { return t.chatService }
func (t *ToolFactory) GetAgentService() services.AgentService { return t.agentService }
func (t *ToolFactory) GetModelService() services.ModelService { return t.modelService }

func NewToolFactory() (*ToolFactory, error) {
	toolFactory := &ToolFactory{}
	toolFactory.toolFactories = make(map[string]*ToolFactoryEntry)

	toolFactory.toolFactories["Process"] = &ToolFactoryEntry{
		Name:        "Process",
		Description: `This tool executes a configured CLI command with support for background processes, timeouts, and full output. The command is executed in the workspace directory. The extraArgs are prepended with the arguments passed to the tool.`,
		ConfigKeys:  []string{"workspace", "command", "extraArgs"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewProcessTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["FileSearch"] = &ToolFactoryEntry{
		Name:        "FileSearch",
		Description: `This tool provides the ability to search for text in files. The workspace directory is prepended to any file paths specified.`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewFileSearchTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["FileRead"] = &ToolFactoryEntry{
		Name:        "FileRead",
		Description: `This tool provides the ability to read files. The workspace directory is prepended to any file paths specified.`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewFileReadTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["FileWrite"] = &ToolFactoryEntry{
		Name:        "FileWrite",
		Description: `This tool provides file writing and modification operations, including overwriting, editing, inserting, and deleting content in files. The workspace directory is prepended to relative file paths specified. Absolute paths are used as-is.`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewFileWriteTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Directory"] = &ToolFactoryEntry{
		Name:        "Directory",
		Description: `This tool provides directory and file management operations, including creating directories, listing directory contents, building directory trees, and moving files or directories. The workspace directory is prepended to any paths specified.`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewDirectoryTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["WebSearch"] = &ToolFactoryEntry{
		Name:        "WebSearch",
		Description: `This tool searches the web using the Tavily API.`,
		ConfigKeys:  []string{"tavily_api_key"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewWebSearchTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Fetch"] = &ToolFactoryEntry{
		Name:        "Fetch",
		Description: `This tool provides the ability to fetch content from the internet using the HTTP 1.1 protocol. This is useful when paired with the Swagger tool.`,
		ConfigKeys:  []string{"user_agent"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewFetchTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Swagger"] = &ToolFactoryEntry{
		Name:        "Swagger",
		Description: `This tool provides a Swagger/OpenAPI specification for a configured URL, providing available endpoints for REST API interactions. Use this in conjunction with the Fetch tool to perform the actions in the specification.`,
		ConfigKeys:  []string{"swagger_url"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewSwaggerTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Memory"] = &ToolFactoryEntry{
		Name:        "Memory",
		Description: `This tool manages a knowledge graph with entities, relations, and observations, allowing creation, modification, deletion, and querying of structured data.`,
		ConfigKeys:  []string{"mongo_uri", "mongo_collection"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewMemoryTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Browser"] = &ToolFactoryEntry{
		Name:        "Browser",
		Description: `This tool provides headless browser control using the Rod library for navigation and interaction.`,
		ConfigKeys:  []string{"headless", "workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewBrowserTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Image"] = &ToolFactoryEntry{
		Name:        "Image",
		Description: `This tool generates images using AI providers like XAI or OpenAI.`,
		ConfigKeys:  []string{"provider", "api_key", "base_url", "model"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewImageTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Vision"] = &ToolFactoryEntry{
		Name:        "Vision",
		Description: "This tool provides image understanding capabilities using providers like XAI or OpenAI, allowing processing of images via base64 or URLs combined with text prompts.",
		ConfigKeys:  []string{"provider", "api_key", "base_url", "model"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return &VisionTool{
				NameField:            name,
				DescriptionField:     description,
				FullDescriptionField: description,
				ConfigurationField:   configuration,
			}
		},
	}
	toolFactory.toolFactories["Todo"] = &ToolFactoryEntry{
		Name:        "Todo",
		Description: "This tool manages a structured task list for complex tasks, allowing creation, reading, and status updates of todos with workflow grouping support.",
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewTodoTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Compression"] = &ToolFactoryEntry{
		Name:        "Compression",
		Description: "This tool provides intelligent context compression for managing conversation history, allowing selective summarization of message ranges based on different compression strategies.",
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewCompressionTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Agent"] = &ToolFactoryEntry{
		Name:        "Agent",
		Description: "Launches a sub-agent by name to complete a specific task and returns its response. Use this to delegate work to specialised agents such as Architect, Coder, QA, or DevOps.",
		ConfigKeys:  []string{},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			// Services are injected lazily via SetServices(); AgentTool reads
			// them from the factory at Execute time, not at construction time.
			return NewAgentTool(name, description, configuration, toolFactory, logger)
		},
	}
	return toolFactory, nil
}

func (t *ToolFactory) ListFactories() ([]*ToolFactoryEntry, error) {
	var factories []*ToolFactoryEntry
	for _, factory := range t.toolFactories {
		factories = append(factories, factory)
	}
	return factories, nil
}

func (t *ToolFactory) GetFactoryByName(name string) (*ToolFactoryEntry, error) {
	factory, exists := t.toolFactories[name]
	if !exists {
		return nil, errors.NotFoundErrorf("Tool factory with name '%s' not found", name)
	}
	return factory, nil
}
