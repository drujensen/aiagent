package services

import (
	"context"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"
)

/**
 * @description
 * ToolService provides business logic for managing tools in the AI Workflow Automation Platform.
 * It acts as an intermediary between API controllers, UI controllers, and the data repository,
 * encapsulating tool-related operations like listing available tools.
 *
 * Key features:
 * - Listing Tools: Retrieves all tools from the repository for use in API responses and UI displays.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: For the Tool entity definition.
 * - aiagent/internal/domain/repositories: For the ToolRepository interface.
 * - context: For managing request timeouts and cancellations.
 *
 * @notes
 * - Currently provides only ListTools; future enhancements could include tool validation or categorization.
 * - Errors from the repository (e.g., database failures) are propagated to the caller for handling.
 * - Used by AgentFormHandler in the UI to populate the tools dropdown dynamically.
 * - Assumes tools are pre-registered via the tool registry (Step 18).
 */

type ToolService interface {
	// ListTools retrieves all available tools from the repository.
	// Returns a slice of Tool pointers and an error if the operation fails.
	//
	// Returns:
	// - []*entities.Tool: List of tools, empty if none exist.
	// - error: Nil on success, or repository-specific error (e.g., database connection lost).
	ListTools(ctx context.Context) ([]*entities.Tool, error)
}

type toolService struct {
	toolRepo repositories.ToolRepository
}

// NewToolService creates a new instance of toolService with the provided ToolRepository.
// This constructor enforces dependency injection for easier testing and modularity.
//
// Parameters:
// - toolRepo: The repository instance for accessing tool data.
//
// Returns:
// - *toolService: A new instance implementing the ToolService interface.
func NewToolService(toolRepo repositories.ToolRepository) *toolService {
	return &toolService{
		toolRepo: toolRepo,
	}
}

// ListTools implements the ToolService interface by calling the repository's ListTools method.
// It passes through the context for cancellation and timeout management.
//
// Parameters:
// - ctx: Context for managing request lifecycle.
//
// Returns:
// - []*entities.Tool: Slice of tools, empty if none exist.
// - error: Any error returned by the repository, such as database errors.
func (s *toolService) ListTools(ctx context.Context) ([]*entities.Tool, error) {
	return s.toolRepo.ListTools(ctx)
}
