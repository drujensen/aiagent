package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/tui/formatters"

	"go.uber.org/zap"
)

type FileWriteTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewFileWriteTool(name, description string, configuration map[string]string, logger *zap.Logger) *FileWriteTool {
	return &FileWriteTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *FileWriteTool) Name() string {
	return t.name
}

func (t *FileWriteTool) Description() string {
	return t.description
}

func (t *FileWriteTool) Configuration() map[string]string {
	return t.configuration
}

func (t *FileWriteTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *FileWriteTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{Name: "filePath", Type: "string", Description: "The absolute path to the file to modify", Required: true},
		{Name: "oldString", Type: "string", Description: "The text to replace", Required: true},
		{Name: "newString", Type: "string", Description: "The text to replace it with (must be different from oldString)", Required: true},
		{Name: "replaceAll", Type: "boolean", Description: "Replace all occurrences of oldString (default false)", Required: false},
	}
}

func (t *FileWriteTool) FullDescription() string {
	return fmt.Sprintf(`%s

**Parameters**:
- **filePath**: The absolute path to the file to modify
- **oldString**: The text to replace (exact match from FileRead)
- **newString**: The replacement text
- **replaceAll**: Replace all occurrences (default false)

**Best Practice**:
1. FileRead → copy exact snippet (indent/whitespace preserved) as oldString
2. Edit with oldString="old...", newString="new..."
3. Verify: re-FileRead

Examples:
{"filePath":"foo.go","oldString":"func foo(){","newString":"func foo() error {\n  return nil\n}","replaceAll":false}`, t.Description())
}

func (t *FileWriteTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file to modify",
			},
			"oldString": map[string]any{
				"type":        "string",
				"description": "The text to replace",
			},
			"newString": map[string]any{
				"type":        "string",
				"description": "The text to replace it with (must be different from oldString)",
			},
			"replaceAll": map[string]any{
				"type":        "boolean",
				"description": "Replace all occurrences of oldString (default false)",
				"default":     false,
			},
		},
		"required":             []string{"filePath", "oldString", "newString"},
		"additionalProperties": false,
	}
}

func (t *FileWriteTool) validatePath(path string) (string, error) {
	// Ensure path is valid UTF-8
	if !utf8.ValidString(path) {
		t.logger.Error("Path contains invalid UTF-8", zap.String("path", path))
		return "", fmt.Errorf("path contains invalid UTF-8")
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current directory: %v", err)
		}
	}

	var fullPath string
	if filepath.IsAbs(path) {
		if !strings.HasPrefix(path, workspace) {
			t.logger.Error("Absolute path is outside workspace", zap.String("path", path))
			return "", fmt.Errorf("absolute path is outside workspace")
		}
		fullPath = path
	} else {
		fullPath = filepath.Join(workspace, path)
	}

	rel, err := filepath.Rel(workspace, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.logger.Error("Path is outside workspace", zap.String("path", path))
		return "", fmt.Errorf("path is outside workspace")
	}
	return fullPath, nil
}

func (t *FileWriteTool) Execute(ctx context.Context, arguments string) (string, error) {
	t.logger.Debug("Executing file write command", zap.String("arguments", arguments))

	var rawArgs map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &rawArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments")
	}

	// Extract fields with proper defaults
	args := struct {
		Operation  string
		FilePath   string
		NewString  string
		OldString  string
		ReplaceAll bool
	}{
		Operation:  getStringField(rawArgs, "operation"),
		FilePath:   getStringField(rawArgs, "filePath"),
		NewString:  getStringField(rawArgs, "newString"),
		OldString:  getStringField(rawArgs, "oldString"),
		ReplaceAll: getBoolField(rawArgs, "replaceAll"),
	}

	if args.FilePath == "" {
		return "", fmt.Errorf("filePath is required")
	}

	// Auto-detect operation: if oldString provided, edit; else write
	if args.OldString != "" {
		args.Operation = "edit"
	} else {
		args.Operation = "write"
	}

	// Validate required fields
	if args.Operation == "edit" {
		if args.NewString == "" {
			return "", fmt.Errorf("newString is required for edit operation")
		}
	} else if args.Operation == "write" {
		if args.NewString == "" {
			return "", fmt.Errorf("newString is required for write operation")
		}
	}

	fullPath, err := t.validatePath(args.FilePath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %s", err.Error())
	}

	if args.Operation == "write" {
		return t.executeWriteOperation(args, fullPath)
	} else if args.Operation == "edit" {
		return t.executeEditOperation(args, fullPath)
	}

	return `{"success": false, "error": "invalid operation"}`, nil
}

// executeWriteOperation handles write operations (create/overwrite/append)
func (t *FileWriteTool) executeWriteOperation(args struct {
	Operation  string
	FilePath   string
	NewString  string
	OldString  string
	ReplaceAll bool
}, fullPath string) (string, error) {
	// Determine the operation type
	fileExisted := false
	operation := "create"
	if _, err := os.Stat(fullPath); err == nil {
		fileExisted = true
		operation = "overwrite"
	}

	var file *os.File
	var err error
	file, err = os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %s", err.Error())
	}
	defer file.Close()

	if _, err := file.WriteString(args.NewString); err != nil {
		return "", fmt.Errorf("failed to write file: %s", err.Error())
	}

	// Generate diff for the operation
	diff := t.generateDiff(args.FilePath, operation, args.NewString, false)

	// Generate summary
	summary := "File created successfully"
	if fileExisted {
		summary = "File overwritten successfully"
	}

	return fmt.Sprintf(`{"success": true, "summary": %q, "diff": %q, "filePath": %q, "occurrences": 0, "replacedAll": false}`, summary, diff, args.FilePath), nil
}

// executeEditOperation handles edit operations (find and replace)
func (t *FileWriteTool) executeEditOperation(args struct {
	Operation  string
	FilePath   string
	NewString  string
	OldString  string
	ReplaceAll bool
}, fullPath string) (string, error) {
	// Read the current file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %s", err.Error())
	}

	fileContent := string(content)

	// Regular string replacement mode
	// Count total occurrences
	totalOccurrences := strings.Count(fileContent, args.OldString)

	// Perform the replacement
	var newContent string
	var occurrences int
	if args.ReplaceAll {
		newContent = strings.ReplaceAll(fileContent, args.OldString, args.NewString)
		occurrences = totalOccurrences
	} else {
		// Replace only the first occurrence
		if idx := strings.Index(fileContent, args.OldString); idx >= 0 {
			newContent = fileContent[:idx] + args.NewString + fileContent[idx+len(args.OldString):]
			occurrences = totalOccurrences // Return total count found
		} else {
			// Provide helpful error with context
			fileLines := strings.Split(fileContent, "\n")
			previewLines := 5
			if len(fileLines) < previewLines {
				previewLines = len(fileLines)
			}
			var preview strings.Builder
			preview.WriteString("File preview (first ")
			preview.WriteString(fmt.Sprintf("%d lines):\n", previewLines))
			for i := 0; i < previewLines; i++ {
				preview.WriteString(fmt.Sprintf("  %d: %s\n", i+1, fileLines[i]))
			}
			if len(fileLines) > previewLines {
				preview.WriteString(fmt.Sprintf("  ... and %d more lines\n", len(fileLines)-previewLines))
			}

			return "", fmt.Errorf("oldString not found in file.\nSearched for: %q\n%s", args.OldString, preview.String())
		}
	}

	// Write the modified content back
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write modified content: %s", err.Error())
	}

	// Generate diff showing the change
	diff := t.generateEditDiff(args.FilePath, fileContent, newContent, args.OldString, args.NewString, occurrences)

	summary := fmt.Sprintf("Replaced %d occurrence(s)", occurrences)

	return fmt.Sprintf(`{"success": true, "summary": %q, "diff": %q, "filePath": %q, "occurrences": %d, "replacedAll": %t}`, summary, diff, args.FilePath, occurrences, args.ReplaceAll), nil
}

// generateDiff creates a simple unified diff showing the changes made
func (t *FileWriteTool) generateDiff(path, operation, content string, wasAppend bool) string {
	var diff strings.Builder

	// Create a simple unified diff header
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	diff.WriteString(fmt.Sprintf("--- %s\t%s\n", path, timestamp))
	diff.WriteString(fmt.Sprintf("+++ %s\t%s\n", path, timestamp))

	if wasAppend {
		diff.WriteString("@@ -1,0 +1,0 @@\n") // Simplified for append
	} else {
		diff.WriteString("@@ -1 +1 @@\n") // Simplified hunk header
	}

	lines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")

	switch operation {
	case "create":
		// Show all lines as additions
		for _, line := range lines {
			diff.WriteString("+")
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	case "overwrite":
		// For overwrites, show the new content as replacement
		for _, line := range lines {
			diff.WriteString("+")
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	case "append":
		// Show appended content as additions
		for _, line := range lines {
			diff.WriteString("+")
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	}

	return diff.String()
}

// generateEditDiff creates a diff showing the edit changes
func (t *FileWriteTool) generateEditDiff(filePath, oldContent, newContent, oldString, newString string, occurrences int) string {
	var diff strings.Builder

	// Create a unified diff header
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	diff.WriteString(fmt.Sprintf("--- %s\t%s\n", filePath, timestamp))
	diff.WriteString(fmt.Sprintf("+++ %s\t%s\n", filePath, timestamp))

	// Split the old and new strings into lines
	oldLines := strings.Split(strings.TrimSuffix(oldString, "\n"), "\n")
	newLines := strings.Split(strings.TrimSuffix(newString, "\n"), "\n")

	// Simple diff showing the change - show all lines that were replaced
	diff.WriteString("@@ -1 +1 @@\n")

	// Show removed lines
	for _, line := range oldLines {
		if line != "" || len(oldLines) > 1 { // Include empty lines if there are multiple lines
			diff.WriteString("-")
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	}

	// Show added lines
	for _, line := range newLines {
		if line != "" || len(newLines) > 1 { // Include empty lines if there are multiple lines
			diff.WriteString("+")
			diff.WriteString(line)
			diff.WriteString("\n")
		}
	}

	return diff.String()
}

// Helper functions for parsing JSON fields
func getStringField(data map[string]interface{}, key string) string {
	if val, ok := data[key]; ok {
		if str, ok := val.(string); ok {
			return str
		}
	}
	return ""
}

func getBoolField(data map[string]interface{}, key string) bool {
	if val, ok := data[key]; ok {
		if b, ok := val.(bool); ok {
			return b
		}
	}
	return false
}

func (t *FileWriteTool) DisplayName(ui string, arguments string) (string, string) {
	var args struct {
		FilePath string `json:"filePath"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.FilePath != "" {
		return t.Name(), args.FilePath
	}
	return t.Name(), ""
}

func (t *FileWriteTool) FormatResult(ui string, result string, diff string, arguments string) string {
	if ui == "tui" {
		return t.formatResultTUI(result, diff)
	} else if ui == "webui" {
		return t.formatResultWebUI(result, diff)
	}
	return result
}

func (t *FileWriteTool) formatResultTUI(result string, diff string) string {
	// Copy from chat_view.go formatFileWriteResult
	var resultData struct {
		Summary     string `json:"summary"`
		Success     bool   `json:"success"`
		Path        string `json:"path"`
		Occurrences int    `json:"occurrences"`
		ReplacedAll bool   `json:"replaced_all"`
		Diff        string `json:"diff"`
	}

	// First, try to use the diff parameter if available
	if diff != "" {
		// Try to parse JSON to get summary
		if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Summary != "" {
			var output strings.Builder
			output.WriteString(resultData.Summary)
			output.WriteString("\n\n" + formatters.FormatDiff(diff))
			return output.String()
		} else {
			// If JSON parsing fails, create a simple summary
			var output strings.Builder
			output.WriteString("File modified successfully\n\n")
			output.WriteString(formatters.FormatDiff(diff))
			return output.String()
		}
	}

	// If no diff parameter, try to parse the full JSON result
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		// If parsing fails, try to extract summary from JSON
		var jsonResponse struct {
			Summary string `json:"summary"`
		}
		if err2 := json.Unmarshal([]byte(result), &jsonResponse); err2 == nil && jsonResponse.Summary != "" {
			return jsonResponse.Summary
		}
		return result // Return raw if parsing fails
	}

	var output strings.Builder

	// Use the summary from the JSON response
	output.WriteString(resultData.Summary)

	// Add the diff from JSON if available
	if resultData.Diff != "" {
		output.WriteString("\n\n" + formatters.FormatDiff(resultData.Diff))
	}

	return output.String()
}

func (t *FileWriteTool) formatResultWebUI(result string, diff string) string {
	// Copy from ui/formatters.go formatFileWriteResult
	var resultData struct {
		Summary     string `json:"summary"`
		Success     bool   `json:"success"`
		Path        string `json:"path"`
		Occurrences int    `json:"occurrences"`
		ReplacedAll bool   `json:"replaced_all"`
		Diff        string `json:"diff"`
	}

	var output strings.Builder

	// First, try to use the diff parameter if available
	if diff != "" {
		// Try to parse JSON to get summary
		if err := json.Unmarshal([]byte(result), &resultData); err == nil && resultData.Summary != "" {
			output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(resultData.Summary)))
			output.WriteString(formatters.FormatDiff(diff)) // Assuming formatters has HTML version
			return output.String()
		} else {
			// If JSON parsing fails, create a simple summary
			output.WriteString("<div class=\"tool-summary\">File modified successfully</div>")
			output.WriteString(formatters.FormatDiff(diff))
			return output.String()
		}
	}

	// If no diff parameter, try to parse the full JSON result
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		// If parsing fails, try to extract summary from JSON
		var jsonResponse struct {
			Summary string `json:"summary"`
		}
		if err2 := json.Unmarshal([]byte(result), &jsonResponse); err2 == nil && jsonResponse.Summary != "" {
			return fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(jsonResponse.Summary))
		}
		return formatters.FormatGenericResult(result) // Assuming
	}

	// Use the summary from the JSON response
	output.WriteString(fmt.Sprintf("<div class=\"tool-summary\">%s</div>", html.EscapeString(resultData.Summary)))

	// Add the diff from JSON if available
	if resultData.Diff != "" {
		output.WriteString(formatters.FormatDiff(resultData.Diff))
	}

	return output.String()
}

var _ entities.Tool = (*FileWriteTool)(nil)
