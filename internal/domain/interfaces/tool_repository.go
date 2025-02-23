package interfaces

import (
	"context"

	"aiagent/internal/domain/entities"
)

type ToolRepository interface {
	CreateTool(ctx context.Context, tool *entities.Tool) error
	UpdateTool(ctx context.Context, tool *entities.Tool) error
	DeleteTool(ctx context.Context, id string) error
	GetTool(ctx context.Context, id string) (*entities.Tool, error)
	ListTools(ctx context.Context) ([]*entities.Tool, error)
}
