package services

import (
	"context"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/repositories"
)

/**
 * @description
 * ToolService provides business logic for managing tools in the AI Workflow Automation Platform.
 * It acts as an intermediary between the API controllers and the data repository, encapsulating
 * any necessary logic or validation for tool-related operations.
 *
 * Key features:
 * - Listing Tools: Retrieves the list of available tools from the repository.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: For the Tool entity definition.
 * - aiagent/internal/domain/repositories: For the ToolRepository interface.
 *
 * @notes
 * - Currently, the service only provides a ListTools method, which directly calls the repository.
 * - Future enhancements could include tool validation, categorization, or custom tool management.
 * - Error handling is minimal, relying on the repository's error responses.
 */

type ToolService interface {
	// ListTools retrieves all available tools from the repository.
	// Returns a slice of Tool pointers and an error if the operation fails.
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
