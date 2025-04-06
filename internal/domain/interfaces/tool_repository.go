package interfaces

import "aiagent/internal/domain/entities"

type ToolRepository interface {
	RegisterTool(name string, tool *entities.Tool) error
	GetToolByName(name string) (*entities.Tool, error)
	ListTools() ([]*entities.Tool, error)
}
