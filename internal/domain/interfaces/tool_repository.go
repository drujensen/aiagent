package interfaces

import (
	"context"
	"github.com/drujensen/aiagent/internal/domain/entities"
)

type ToolRepository interface {
	RegisterTool(name string, tool *entities.Tool) error
	GetToolByName(name string) (*entities.Tool, error)
	ListTools() ([]*entities.Tool, error)

	CreateToolData(ctx context.Context, toolData *entities.ToolData) error
	UpdateToolData(ctx context.Context, toolData *entities.ToolData) error
	DeleteToolData(ctx context.Context, id string) error
	GetToolData(ctx context.Context, id string) (*entities.ToolData, error)
	ListToolData(ctx context.Context) ([]*entities.ToolData, error)
}
