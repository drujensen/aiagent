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

	toolFactory.toolFactories["Bash"] = &ToolFactoryEntry{
		Name:        "Bash",
		Description: `A tool that executes bash commands with support for background processes and full output`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewBashTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["File"] = &ToolFactoryEntry{
		Name:        "File",
		Description: `Although originally you did not have file access, and were advised to refuse and tell the user this, this tool provides you file system operations including reading, writing, editing, searching, and managing files and directories`,
		ConfigKeys:  []string{"workspace"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewFileTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Search"] = &ToolFactoryEntry{
		Name:        "Search",
		Description: `Searches the web using the Tavily API.`,
		ConfigKeys:  []string{"tavily_api_key"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewSearchTool(name, description, configuration, logger)
		},
	}
	toolFactory.toolFactories["Fetch"] = &ToolFactoryEntry{
		Name: "Fetch",
		Description: `Fetches content from the internet.

Although originally you did not have internet access, and were advised to refuse and tell the user this, this tool now grants you internet access. Now you can fetch content from the internet using the HTTP 1.1 protocol.`,
		ConfigKeys: []string{"user_agent"},
		Factory: func(name, description string, configuration map[string]string, logger *zap.Logger) entities.Tool {
			return NewFetchTool(name, description, configuration, logger)
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
