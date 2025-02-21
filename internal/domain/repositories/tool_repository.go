package repositories

import (
	"context"

	"aiagent/internal/domain/entities"
)

// ToolRepository defines the interface for managing Tool entities in the domain layer.
// It provides methods to create, update, delete, and retrieve tools, abstracting the data
// storage implementation (e.g., MongoDB). This interface ensures the domain logic can interact
// with tools without depending on specific infrastructure details.
//
// Key features:
// - CRUD Operations: Supports full lifecycle management of tools.
// - Context Usage: Includes context.Context for timeout and cancellation support.
// - Error Handling: Uses ErrNotFound (from package scope) for non-existent tools in GetTool and DeleteTool.
//
// Dependencies:
// - context: For handling request timeouts and cancellations.
// - aiagent/internal/domain/entities: Provides the Tool entity definition.
//
// Notes:
// - CreateTool sets the tool's ID in place, assuming the storage layer generates it.
// - UpdateTool requires a valid ID in the Tool struct.
// - ListTools returns pointers to avoid unnecessary copying of Tool structs.
type ToolRepository interface {
	// CreateTool inserts a new tool into the repository and sets the tool's ID.
	// The tool's ID field is updated with the generated ID upon successful insertion.
	// Returns an error if the insertion fails (e.g., duplicate name, database error).
	CreateTool(ctx context.Context, tool *entities.Tool) error

	// UpdateTool updates an existing tool in the repository.
	// The tool must have a valid ID; returns an error if the tool does not exist
	// or if the update fails (e.g., validation error, database issue).
	UpdateTool(ctx context.Context, tool *entities.Tool) error

	// DeleteTool deletes a tool by its ID.
	// Returns ErrNotFound if no tool with the given ID exists; otherwise, returns
	// an error if the deletion fails (e.g., database connectivity issue).
	DeleteTool(ctx context.Context, id string) error

	// GetTool retrieves a tool by its ID.
	// Returns a pointer to the Tool and nil error on success, or nil and ErrNotFound
	// if the tool does not exist, or nil and another error for other failures (e.g., database error).
	GetTool(ctx context.Context, id string) (*entities.Tool, error)

	// ListTools retrieves all tools in the repository.
	// Returns a slice of pointers to Tool entities; returns an empty slice if no tools exist,
	// or an error if the retrieval fails (e.g., database connection lost).
	ListTools(ctx context.Context) ([]*entities.Tool, error)
}

// Notes:
// - Error handling expects implementations to provide detailed errors for debugging (e.g., "tool not found: id=xyz").
// - Edge case: Duplicate tool names are allowed but should be unique within categories, enforced elsewhere.
// - Assumption: Tool IDs are strings, consistent with MongoDB ObjectID usage in entities.
// - Limitation: No filtering in ListTools; could be extended for category-based queries if needed.
