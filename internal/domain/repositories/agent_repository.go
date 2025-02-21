package repositories

import (
	"context"

	"aiagent/internal/domain/entities"
)

// AgentRepository defines the interface for managing AIAgent entities in the domain layer.
// It provides methods to create, update, delete, and retrieve agents, abstracting the underlying
// data storage mechanism (e.g., MongoDB). This interface follows the Dependency Inversion Principle,
// ensuring the domain logic remains independent of infrastructure details.
//
// Key features:
// - CRUD Operations: Full support for creating, updating, deleting, and retrieving agents.
// - Context Usage: All methods accept a context.Context for cancellation and timeout handling.
// - Error Handling: Returns ErrNotFound (from package scope) for non-existent agents in GetAgent and DeleteAgent.
//
// Dependencies:
// - context: For managing request timeouts and cancellations.
// - aiagent/internal/domain/entities: Provides the AIAgent entity definition.
//
// Notes:
// - CreateAgent modifies the agent's ID in place, assuming the underlying storage generates it.
// - UpdateAgent expects a fully populated AIAgent with a valid ID.
// - ListAgents returns a slice of pointers to avoid copying large structs unnecessarily.
type AgentRepository interface {
	// CreateAgent inserts a new agent into the repository and sets the agent's ID.
	// The agent's ID field is updated with the generated ID upon successful insertion.
	// Returns an error if the insertion fails (e.g., duplicate name, database error).
	CreateAgent(ctx context.Context, agent *entities.AIAgent) error

	// UpdateAgent updates an existing agent in the repository.
	// The agent must have a valid ID set; returns an error if the agent does not exist
	// or if the update fails due to validation or database issues.
	UpdateAgent(ctx context.Context, agent *entities.AIAgent) error

	// DeleteAgent deletes an agent by its ID.
	// Returns ErrNotFound if no agent with the given ID exists; otherwise, returns
	// an error if the deletion fails (e.g., database connectivity issue).
	DeleteAgent(ctx context.Context, id string) error

	// GetAgent retrieves an agent by its ID.
	// Returns a pointer to the AIAgent and nil error on success, or nil and ErrNotFound
	// if the agent does not exist, or nil and another error for other failures (e.g., database error).
	GetAgent(ctx context.Context, id string) (*entities.AIAgent, error)

	// ListAgents retrieves all agents in the repository.
	// Returns a slice of pointers to AIAgent entities; returns an empty slice if no agents exist,
	// or an error if the retrieval fails (e.g., database connection lost).
	ListAgents(ctx context.Context) ([]*entities.AIAgent, error)
}

// Notes:
// - Error handling assumes implementations will wrap underlying errors with context (e.g., "database error: timeout").
// - Edge case: Empty ID in UpdateAgent or DeleteAgent should return an error, handled by the implementation.
// - Assumption: IDs are strings matching MongoDB's ObjectID hex format, consistent with entity definitions.
// - Limitation: No pagination in ListAgents; could be added later if large datasets are expected.
