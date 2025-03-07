package integrations

import (
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/infrastructure/integrations/tools"
	"fmt"
)

var predefinedTools = []struct {
	name    string
	factory func(configuration map[string]string) interfaces.ToolIntegration
}{
	{
		name: "Search",
		factory: func(configuration map[string]string) interfaces.ToolIntegration {
			return tools.NewSearchTool(configuration)
		},
	},
	{
		name: "Bash",
		factory: func(configuration map[string]string) interfaces.ToolIntegration {
			return tools.NewBashTool(configuration)
		},
	},
	{
		name: "File",
		factory: func(configuration map[string]string) interfaces.ToolIntegration {
			return tools.NewFileTool(configuration)
		},
	},
}

type ToolRegistry struct {
	toolInstancesByName map[string]*interfaces.ToolIntegration
}

func NewToolRegistry(configuration map[string]string) (*ToolRegistry, error) {
	toolRegistry := &ToolRegistry{}
	toolRegistry.toolInstancesByName = make(map[string]*interfaces.ToolIntegration)

	for _, pt := range predefinedTools {
		toolInstance := pt.factory(configuration)
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
		return nil, fmt.Errorf("tool with name %s not found", name)
	}
	return tool, nil
}
