package repositories

import (
	"aiagent/internal/domain/errors"
	"aiagent/internal/domain/interfaces"
)

type ToolRepository struct {
	toolInstances map[string]*interfaces.ToolIntegration
}

func NewToolRepository() (*ToolRepository, error) {
	toolRepository := &ToolRepository{}
	toolRepository.toolInstances = make(map[string]*interfaces.ToolIntegration)

	return toolRepository, nil
}

func (t *ToolRepository) ListTools() ([]*interfaces.ToolIntegration, error) {
	var tools []*interfaces.ToolIntegration
	for _, tool := range t.toolInstances {
		tools = append(tools, tool)
	}
	return tools, nil
}

func (t *ToolRepository) GetToolByName(name string) (*interfaces.ToolIntegration, error) {
	tool, exists := t.toolInstances[name]
	if !exists {
		return nil, nil
	}
	return tool, nil
}

func (t *ToolRepository) RegisterTool(name string, tool *interfaces.ToolIntegration) error {
	if _, exists := t.toolInstances[name]; exists {
		return errors.DuplicateErrorf("tool with the same name already exists")
	}
	t.toolInstances[name] = tool
	return nil
}

var _ interfaces.ToolRepository = (*ToolRepository)(nil)
