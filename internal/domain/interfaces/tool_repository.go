package interfaces

import (
	"context"
	"github.com/drujensen/aiagent/internal/domain/entities"
)

type ToolRepository interface {
	RegisterTool(name string, tool entities.Tool) error
	GetToolByName(name string) (entities.Tool, error)
	ListTools() ([]entities.Tool, error)

	// GetToolForChat returns a tool instance ready to execute within one
	// chat turn, with its configuration already resolved (no #{VAR}#
	// placeholders). For most tool types this mints a fresh instance
	// scoped to the call, so concurrent chat turns using the same tool
	// name never share (and never race on) an instance's configuration.
	// For tools whose factory entry is marked Stateful, this instead
	// returns the existing shared singleton - which already has resolved
	// configuration applied once, at construction time - since minting a
	// fresh instance would break state that must persist across turns
	// (an open browser page, tracked background processes, a DB handle).
	// config is merged over the tool's stored configuration before
	// resolution; pass nil if there is no per-call override.
	GetToolForChat(ctx context.Context, name string, config map[string]string) (entities.Tool, error)

	CreateToolData(ctx context.Context, toolData *entities.ToolData) error
	UpdateToolData(ctx context.Context, toolData *entities.ToolData) error
	DeleteToolData(ctx context.Context, id string) error
	GetToolData(ctx context.Context, id string) (*entities.ToolData, error)
	ListToolData(ctx context.Context) ([]*entities.ToolData, error)
}
