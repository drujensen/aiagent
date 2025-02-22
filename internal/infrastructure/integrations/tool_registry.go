package integrations

import (
	"context"
	"time"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"
	"aiagent/internal/domain/repositories"
	"aiagent/internal/infrastructure/integrations/tools"
)

/**
 * @description
 * This file implements the tool registry for the AI Workflow Automation Platform.
 * It manages predefined tools (Search, Bash, File), ensuring they are registered
 * in the database and providing access to their instances via tool IDs. The registry
 * is initialized at application startup with a workspace path for tools requiring it.
 *
 * Key features:
 * - Tool Initialization: Registers predefined tools in MongoDB if not present.
 * - Instance Management: Maps tool IDs to their concrete implementations.
 * - Workspace Support: Configures Bash and File tools with the workspace directory.
 *
 * @dependencies
 * - aiagent/internal/domain/entities: For Tool entity definition.
 * - aiagent/internal/domain/interfaces: For Tool interface definition.
 * - aiagent/internal/domain/repositories: For ToolRepository interface.
 * - aiagent/internal/infrastructure/integrations/tools: For tool implementations (SearchTool, BashTool, FileTool).
 * - context: For database operation timeouts.
 * - time: For setting creation/update timestamps.
 *
 * @notes
 * - Predefined tools are hardcoded as per the spec (Search, Bash, File).
 * - Workspace is set to "/workspace" per Docker compose.yml; could be made configurable.
 * - Tool instances are stateless or safely reusable; BashTool and FileTool use workspace confinement.
 * - Error handling ensures database failures are propagated to the caller.
 * - Assumes MongoDB generates unique IDs for tools; no duplicate name checks beyond initial setup.
 * - Limitation: No dynamic tool registration; only predefined tools are supported currently.
 */

// predefinedTools defines the list of tools available by default in the system.
// Each entry includes the tool’s name, category, and a factory function to create the tool instance.
//
// Structure:
// - name: Unique identifier for the tool (stored in database).
// - category: Groups tools by type (e.g., "search", "bash").
// - factory: Function accepting workspace path, returning a Tool implementation.
var predefinedTools = []struct {
	name     string
	category string
	factory  func(workspace string) interfaces.Tool
}{
	{
		name:     "Search",
		category: "search",
		factory:  func(_ string) interfaces.Tool { return tools.NewSearchTool() },
	},
	{
		name:     "Bash",
		category: "bash",
		factory:  func(workspace string) interfaces.Tool { return tools.NewBashTool(workspace) },
	},
	{
		name:     "File",
		category: "file",
		factory:  func(workspace string) interfaces.Tool { return tools.NewFileTool(workspace) },
	},
}

// toolInstancesByID maps tool IDs to their corresponding tool instances.
// This allows quick retrieval of tool instances by their database IDs.
//
// Type: map[string]interfaces.Tool
// - Key: Tool ID from MongoDB (string representation of ObjectID).
// - Value: Concrete tool implementation satisfying the Tool interface.
var toolInstancesByID map[string]interfaces.Tool

// InitializeTools ensures that predefined tools are present in the database and initializes tool instances.
// It checks for each predefined tool in the database and creates it if it doesn’t exist.
// Then, it creates tool instances using the provided workspace path and stores them by their IDs.
//
// Parameters:
// - ctx: Context for database operations, handling timeouts and cancellations.
// - repo: ToolRepository to interact with the MongoDB tools collection.
// - workspace: Path to the workspace directory (e.g., "/workspace") for tools that require it.
//
// Returns:
// - error: Nil on success, or an error if database operations fail (e.g., connection issues).
func InitializeTools(ctx context.Context, repo repositories.ToolRepository, workspace string) error {
	toolInstancesByID = make(map[string]interfaces.Tool)

	// Iterate over predefined tools to ensure registration and instance creation
	for _, pt := range predefinedTools {
		// Retrieve all existing tools from the database
		existingTools, err := repo.ListTools(ctx)
		if err != nil {
			return err // Propagate database errors (e.g., connection failure)
		}

		// Check if the tool already exists by name
		var toolEntity *entities.Tool
		for _, t := range existingTools {
			if t.Name == pt.name {
				toolEntity = t
				break
			}
		}

		// If tool doesn’t exist, create it
		if toolEntity == nil {
			toolEntity = &entities.Tool{
				Name:      pt.name,
				Category:  pt.category,
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			}
			if err := repo.CreateTool(ctx, toolEntity); err != nil {
				return err // Propagate creation errors (e.g., MongoDB write failure)
			}
		}

		// Create tool instance using the factory function with workspace
		toolInstance := pt.factory(workspace)
		// Map the instance to its database ID
		toolInstancesByID[toolEntity.ID] = toolInstance
	}

	return nil
}

// GetToolByID retrieves a tool instance by its ID.
// It looks up the tool instance in the map using the provided ID.
//
// Parameters:
// - id: The ID of the tool to retrieve (MongoDB ObjectID as string).
//
// Returns:
// - interfaces.Tool: The tool instance, or nil if not found (e.g., ID not in map).
func GetToolByID(id string) interfaces.Tool {
	return toolInstancesByID[id]
}
