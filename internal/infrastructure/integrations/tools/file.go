package tools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// FileTool implements the Tool interface for file operations within a specified workspace directory.
// It supports reading, writing, and patching files, ensuring all actions are confined to the workspace.
//
// Key features:
// - Operation Parsing: Interprets input as "operation:path[:content]" (e.g., "read:/file.txt").
// - Path Security: Validates file paths to prevent directory traversal outside the workspace.
// - File Operations: Handles read (os.ReadFile), write (os.WriteFile), and patch (executes `patch` command).
//
// Dependencies:
// - os: For file read/write operations and temporary file creation.
// - path/filepath: For constructing and validating file paths.
// - strings: For parsing input strings.
// - io: For writing to temporary files.
// - bytes: For buffering patch command output.
//
// Notes:
// - Input format: "read:path", "write:path:content", "patch:path:diff".
// - Edge case: Invalid paths or operations return descriptive errors.
// - Assumption: The `patch` command is available in the environment (e.g., Docker container).
// - Limitation: Patch operation relies on `patch` command; no native diff parsing.
type FileTool struct {
	workspace string // Directory path where file operations are allowed
}

// NewFileTool creates a new FileTool instance with the specified workspace directory.
// It ensures the tool is initialized with a valid operation context.
//
// Parameters:
// - workspace: The directory path to restrict file operations to.
//
// Returns:
// - *FileTool: A new instance of FileTool.
func NewFileTool(workspace string) *FileTool {
	return &FileTool{workspace: workspace}
}

// Name returns the identifier for this tool, used by agents to invoke it.
// It adheres to the Tool interface requirement.
//
// Returns:
// - string: The tool's name, "File".
func (t *FileTool) Name() string {
	return "File"
}

// Execute performs the specified file operation based on the input string.
// It parses the operation and path, validates the path, and executes the requested action.
//
// Parameters:
// - input: The operation string in format "operation:path[:content]".
//
// Returns:
// - string: Result of the operation (e.g., file contents, success message), or empty on error.
// - error: Nil on success, or an error if parsing, validation, or execution fails.
//
// Behavior:
// - Validates paths using filepath.Rel to ensure confinement within workspace.
// - Supports read, write, and patch operations with appropriate error handling.
// - Cleans up temporary files in patch operations using defer.
func (t *FileTool) Execute(input string) (string, error) {
	// Parse input into operation, path, and optional content
	parts := strings.SplitN(input, ":", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid input format: expected at least 'operation:path'")
	}

	operation := parts[0]
	path := parts[1]
	var content string
	if len(parts) == 3 {
		content = parts[2]
	}

	// Construct and validate full path
	fullPath := filepath.Join(t.workspace, path)
	rel, err := filepath.Rel(t.workspace, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	switch operation {
	case "read":
		// Read file contents
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
		}
		return string(data), nil

	case "write":
		// Validate content presence for write
		if len(parts) != 3 {
			return "", fmt.Errorf("write operation requires content")
		}
		// Write content to file with 0644 permissions
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write to file %s: %w", fullPath, err)
		}
		return "File written successfully", nil

	case "patch":
		// Validate diff presence for patch
		if len(parts) != 3 {
			return "", fmt.Errorf("patch operation requires diff content")
		}
		// Create temporary file for diff
		tempFile, err := os.CreateTemp("", "diff-*.txt")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary file for diff: %w", err)
		}
		defer os.Remove(tempFile.Name()) // Clean up after execution

		// Write diff content to temporary file
		_, err = tempFile.WriteString(content)
		if err != nil {
			tempFile.Close()
			return "", fmt.Errorf("failed to write diff to temporary file: %w", err)
		}
		tempFile.Close()

		// Execute patch command
		cmd := exec.Command("patch", fullPath, tempFile.Name())
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("patch failed for %s: %s\n%s", fullPath, err, stderr.String())
		}
		return "File patched successfully", nil

	default:
		return "", fmt.Errorf("unknown operation: %s", operation)
	}
}
