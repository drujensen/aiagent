package tools

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	errors "github.com/drujensen/aiagent/internal/domain/errs"

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
}

func NewToolFactory() (*ToolFactory, error) {
	toolFactory := &ToolFactory{}
	toolFactory.toolFactories = make(map[string]*ToolFactoryEntry)

	toolFactory.toolFactories["Project"] = &ToolFactoryEntry{
		Name:        "Project",
		Description: `This tool reads project details from a configurable markdown file to provide context for AI agents.`,
		ConfigKeys:  []string{"workspace", "project_file"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewProjectTool(name, description, configuration, logger)
		},
	}

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
		Description: `This tool provides file writing and modification operations, including overwriting, editing, inserting, and deleting content in files. The workspace directory is prepended to any file paths specified.`,
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
	toolFactory.toolFactories["MCP"] = &ToolFactoryEntry{
		Name:        "MCP",
		Description: `This tool provides a command line interface for the MCP (Multi-Cloud Provider) API, allowing users to interact with various cloud services and perform operations such as creating, updating, and deleting resources across multiple cloud providers.`,
		ConfigKeys:  []string{"workspace", "command", "args"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewMCPTool(name, description, configuration, logger)
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
