package tools

import (
	"bytes"
	"fmt"
	"os/exec"
)

// BashTool implements the Tool interface for executing bash commands within a specified workspace directory.
// It allows AI agents to run shell commands securely, confined to a designated directory.
//
// Key features:
// - Workspace Restriction: Executes commands only within the specified workspace directory.
// - Output Capture: Returns both stdout and stderr for full command feedback.
// - Error Handling: Reports command failures with error details.
//
// Dependencies:
// - os/exec: For executing bash commands and capturing output.
// - bytes: For buffering command output and errors.
//
// Notes:
// - Commands are executed via `bash -c` to support complex shell syntax.
// - Edge case: Empty input returns an error to prompt agent correction.
// - Assumption: Agent-generated commands are safe; minimal validation applied.
// - Limitation: No advanced command sanitization; relies on workspace confinement for security.
type BashTool struct {
	workspace string // Directory path where commands are executed
}

// NewBashTool creates a new BashTool instance with the specified workspace directory.
// It ensures the tool is initialized with a valid execution context.
//
// Parameters:
// - workspace: The directory path to restrict command execution to.
//
// Returns:
// - *BashTool: A new instance of BashTool.
func NewBashTool(workspace string) *BashTool {
	return &BashTool{workspace: workspace}
}

// Name returns the identifier for this tool, used by agents to invoke it.
// It adheres to the Tool interface requirement.
//
// Returns:
// - string: The tool's name, "Bash".
func (t *BashTool) Name() string {
	return "Bash"
}

// Execute runs the provided bash command within the workspace directory.
// It captures output and errors, returning them as a string or an error if the command fails.
//
// Parameters:
// - command: The bash command string to execute.
//
// Returns:
// - string: The command's stdout output, or empty if failed.
// - error: Nil on success, or an error with stderr details if the command fails.
//
// Behavior:
// - Sets the working directory to workspace to confine execution.
// - Uses buffers to capture stdout and stderr separately.
// - Returns detailed error messages for command failures.
func (t *BashTool) Execute(command string) (string, error) {
	if command == "" {
		return "", fmt.Errorf("bash command cannot be empty")
	}

	// Create command with workspace directory set
	cmd := exec.Command("bash", "-c", command)
	cmd.Dir = t.workspace

	// Buffers for capturing output and errors
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	// Execute the command
	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("command execution failed: %s\n%s", err, stderr.String())
	}

	return out.String(), nil
}
