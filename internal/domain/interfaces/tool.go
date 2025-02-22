package interfaces

/**
 * @description
 * This file defines the Tool interface for the AI Workflow Automation Platform.
 * It specifies the contract that all tool implementations must follow, enabling
 * agents to use tools in a standardized manner without knowing their concrete types.
 *
 * Key features:
 * - Name Method: Provides a unique identifier for the tool.
 * - Execute Method: Executes the tool's functionality with a string input and returns a result.
 *
 * @dependencies
 * - None: Pure interface definition, no external dependencies.
 *
 * @notes
 * - Implementations reside in internal/infrastructure/integrations/tools.
 * - The interface is intentionally minimal to allow flexibility in tool design.
 * - Assumes tool implementations handle their own error logic and input validation.
 */

// Tool defines the interface for tools that can be used by AI agents.
// Each tool must provide a name and an execution method that takes an input string
// and returns a result string or an error.
//
// Methods:
// - Name() string: Returns the tool's unique name.
// - Execute(input string) (string, error): Executes the tool with the given input.
type Tool interface {
	Name() string
	Execute(input string) (string, error)
}
