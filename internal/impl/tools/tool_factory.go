package tools

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/errors"

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

	toolFactory.toolFactories["Process"] = &ToolFactoryEntry{
		Name:        "Process",
		Description: `This tool executes a configured CLI command (e.g., bash, git, gcc, go, rustc, java, dotnet, python, ruby, node, mysql, psql, mongo, redis-cli, aws, az, docker, kubectl) with support for background processes, timeouts, and full output`,
		ConfigKeys:  []string{"command", "workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewProcessTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Bash"] = &ToolFactoryEntry{
		Name:        "Bash",
		Description: `This tool executes a Bash shell with support for background processes, timeouts, and full output`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewBashTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["File"] = &ToolFactoryEntry{
		Name:        "File",
		Description: `This tool provides you file system operations including reading, writing, editing, searching, and managing files and directories`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewFileTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Search"] = &ToolFactoryEntry{
		Name:        "Search",
		Description: `This tool Searches the web using the Tavily API.`,
		ConfigKeys:  []string{"tavily_api_key"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewSearchTool(name, description, configuration, logger)
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
