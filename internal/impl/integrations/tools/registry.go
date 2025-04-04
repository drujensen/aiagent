package tools

import (
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type ToolRegistry struct {
	toolInstancesByName map[string]*interfaces.ToolIntegration
}

func NewToolRegistry(configuration map[string]string, logger *zap.Logger) (*ToolRegistry, error) {
	toolRegistry := &ToolRegistry{}
	toolRegistry.toolInstancesByName = make(map[string]*interfaces.ToolIntegration)

	predefinedTools := []struct {
		name    string
		factory func(configuration map[string]string, logger *zap.Logger) interfaces.ToolIntegration
	}{
		{
			name: "Search",
			factory: func(configuration map[string]string, logger *zap.Logger) interfaces.ToolIntegration {
				return NewSearchTool(configuration, logger)
			},
		},
		{
			name: "Bash",
			factory: func(configuration map[string]string, logger *zap.Logger) interfaces.ToolIntegration {
				return NewBashTool(configuration, logger)
			},
		},
		{
			name: "File",
			factory: func(configuration map[string]string, logger *zap.Logger) interfaces.ToolIntegration {
				return NewFileTool(configuration, logger)
			},
		},
		{
			name: "Fetch",
			factory: func(configuration map[string]string, logger *zap.Logger) interfaces.ToolIntegration {
				return NewFetchTool(configuration, logger)
			},
		},
	}

	for _, pt := range predefinedTools {
		toolInstance := pt.factory(configuration, logger)
		toolRegistry.toolInstancesByName[toolInstance.Name()] = &toolInstance
	}

	return toolRegistry, nil
}

func (t *ToolRegistry) ListTools() ([]*interfaces.ToolIntegration, error) {
	var tools []*interfaces.ToolIntegration
	for _, tool := range t.toolInstancesByName {
		tools = append(tools, tool)
	}
	return tools, nil
}

func (t *ToolRegistry) GetToolByName(name string) (*interfaces.ToolIntegration, error) {
	tool, exists := t.toolInstancesByName[name]
	if !exists {
		return nil, nil
	}
	return tool, nil
}
