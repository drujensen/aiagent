package tools

import (
	"context"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

// ToolSource is a source of dynamically-discovered tools, as opposed to the
// compiled-in ToolFactory registry. It exists so that a tool repository can
// merge tools from an external protocol (MCP) with its native, factory-built
// tools without the domain layer ever seeing MCP-specific types - only this
// impl-level interface and entities.Tool cross that boundary.
type ToolSource interface {
	// Name identifies this source (e.g. a configured MCP server's name),
	// used for logging/diagnostics and to disambiguate tools by origin.
	Name() string
	// ListTools returns the tools currently available from this source.
	ListTools(ctx context.Context) ([]entities.Tool, error)
}
