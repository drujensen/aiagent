package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/drujensen/aiagent/internal/domain/entities"

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
		{Name: "operation", Type: "string", Enum: []string{"write", "edit"}, Description: "Operation to perform (auto-detected if not specified)", Required: false},
		{Name: "path", Type: "string", Description: "File path", Required: true},
		{Name: "content", Type: "string", Description: "File content for write operation", Required: true},
		{Name: "old_string", Type: "string", Description: "String to replace for edit operation", Required: false},
		{Name: "replace_all", Type: "boolean", Description: "Replace all occurrences", Required: false},
	}
}

func (t *FileWriteTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- operation: 'write' or 'edit'\n- path: file path (absolute or relative to workspace)\n- content: file content (for write)\n- old_string, new_string: exact strings to replace (for edit)\n- replace_all: boolean (optional)", t.Description())
}

func (t *FileWriteTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to write to",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Text to write",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "\"overwrite\" or \"append\"",
				"enum":        []string{"overwrite", "append"},
				"default":     "overwrite",
			},
		},
		"required": []string{"path", "content"},
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

func (t *FileWriteTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing file write command", zap.String("arguments", arguments))

	var rawArgs map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &rawArgs); err != nil {
		return "", fmt.Errorf("failed to parse arguments")
	}

	// Extract fields with proper defaults
	args := struct {
		Operation  string
		Path       string
		Content    string
		Mode       string
		OldString  string
		ReplaceAll bool
	}{
		Operation:  getStringField(rawArgs, "operation"),
		Path:       getStringField(rawArgs, "path"),
		Content:    getStringField(rawArgs, "content"),
		Mode:       getStringField(rawArgs, "mode"),
		OldString:  getStringField(rawArgs, "old_string"),
		ReplaceAll: getBoolField(rawArgs, "replace_all"),
	}

	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	// Auto-detect operation based on parameters
	if args.Operation == "" {
		if args.OldString != "" {
			args.Operation = "edit"
		} else {
			args.Operation = "write"
		}
	}

	// Validate operation
	if args.Operation != "write" && args.Operation != "edit" {
		return "", fmt.Errorf("operation must be 'write' or 'edit'")
	}

	// Validate required fields based on operation
	if args.Operation == "write" {
		if args.Content == "" {
			return "", fmt.Errorf("content is required")
		}
		if args.Mode == "" {
			args.Mode = "overwrite"
		}
		if args.Mode != "overwrite" && args.Mode != "append" {
			return "", fmt.Errorf("mode must be 'overwrite' or 'append'")
		}
	} else if args.Operation == "edit" {
		if args.OldString == "" {
			return "", fmt.Errorf("old_string is required for edit operation")
		}
		if args.Content == "" {
			return "", fmt.Errorf("content (new_string) is required for edit operation")
		}
	}

	// Validate operation
	if args.Operation != "write" && args.Operation != "edit" {
		return "", fmt.Errorf("operation must be 'write' or 'edit'")
	}

	// Validate required fields based on operation
	if args.Operation == "write" {
		if args.Content == "" {
			return "", fmt.Errorf("content is required for write operation")
		}
		if args.Mode == "" {
			args.Mode = "overwrite"
		}
		if args.Mode != "overwrite" && args.Mode != "append" {
			return "", fmt.Errorf("mode must be 'overwrite' or 'append'")
		}
	} else if args.Operation == "edit" {
		if args.OldString == "" {
			return "", fmt.Errorf("old_string is required for edit operation")
		}
		if args.Content == "" {
			return "", fmt.Errorf("content (new_string) is required for edit operation")
		}
	}

	fullPath, err := t.validatePath(args.Path)
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
	Path       string
	Content    string
	Mode       string
	OldString  string
	ReplaceAll bool
}, fullPath string) (string, error) {
	// Determine the operation type and create backup if needed
	fileExisted := false
	operation := "create"
	if _, err := os.Stat(fullPath); err == nil {
		fileExisted = true
		if args.Mode == "overwrite" {
			operation = "overwrite"
			backupPath := fullPath + ".backup"
			if err := t.createBackup(fullPath, backupPath); err != nil {
				t.logger.Warn("Failed to create backup", zap.Error(err))
			}
		} else if args.Mode == "append" {
			operation = "append"
		}
	} else if args.Mode == "append" {
		operation = "append"
	}

	var file *os.File
	var err error
	if args.Mode == "append" {
		file, err = os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		file, err = os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	}
	if err != nil {
		return "", fmt.Errorf("failed to open file: %s", err.Error())
	}
	defer file.Close()

	if _, err := file.WriteString(args.Content); err != nil {
		return "", fmt.Errorf("failed to write file: %s", err.Error())
	}

	// Generate diff for the operation
	diff := t.generateDiff(args.Path, operation, args.Content, args.Mode == "append")

	// Generate summary
	summary := "File created successfully"
	if fileExisted && args.Mode == "overwrite" {
		summary = "File overwritten successfully"
	} else if args.Mode == "append" {
		summary = "Content appended successfully"
	}

	return fmt.Sprintf(`{"success": true, "summary": %q, "diff": %q, "path": %q, "occurrences": 0, "replaced_all": false}`, summary, diff, args.Path), nil
}

// executeEditOperation handles edit operations (find and replace)
func (t *FileWriteTool) executeEditOperation(args struct {
	Operation  string
	Path       string
	Content    string
	Mode       string
	OldString  string
	ReplaceAll bool
}, fullPath string) (string, error) {
	// Read the current file content
	content, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file: %s", err.Error())
	}

	fileContent := string(content)

	// Check if we're working with hashline format content
	isHashline := strings.Contains(fileContent, ":") && strings.Contains(fileContent, "|") && strings.Contains(fileContent, "\n")

	if isHashline {
		// Hashline mode - parse the content to find line matches
		// Parse hashline content to understand structure
		lines := strings.Split(strings.TrimSuffix(fileContent, "\n"), "\n")

		// Extract line information from hashline format
		lineMap := make(map[string]string)    // hash -> line content
		lineNumberMap := make(map[string]int) // hash -> line number (1-based)

		for _, line := range lines {
			if strings.Contains(line, ":") {
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					hash := parts[0]
					content := strings.SplitN(parts[1], "|", 2)
					if len(content) == 2 {
						lineMap[hash] = content[1]
						// Extract line number from the beginning of the line (first part)
						lineNumStr := strings.Split(line, ":")[0]
						if lineNum, err := strconv.Atoi(lineNumStr); err == nil {
							lineNumberMap[hash] = lineNum
						}
					}
				}
			} else {
				// Simple line without hash, add to map for backward compatibility
				// We're not using index here, so we don't generate hash based on line number
				lineMap[line] = line
				lineNumberMap[line] = 0 // Placeholder for line number
			}
		}

		// Find exact hash matches for the content to replace
		occurrences := 0
		foundLineHashes := []string{}

		for hash, content := range lineMap {
			if content == args.OldString {
				occurrences++
				foundLineHashes = append(foundLineHashes, hash)
			}
		}

		// If no exact match, try partial match (hashline approach - for LLMs that may produce slight variations)
		if occurrences == 0 {
			// For LLM-generated content, try to find a matching line by content
			// This is a more lenient approach that could be used in LLM context
			for hash, content := range lineMap {
				// Try to match the content - this allows for more robust LLM-based editing
				// This is a more lenient approach that could be used in LLM context
				if strings.Contains(content, args.OldString) ||
					strings.Contains(args.OldString, content) {
					occurrences++
					foundLineHashes = append(foundLineHashes, hash)
				}
			}
		}

		// If still no matches, this means the content was modified since the file was read
		// and the hash doesn't match anymore, which should fail the operation as expected
		if occurrences == 0 {
			return "", fmt.Errorf("old_string not found in file (content has changed since file was read - hash mismatch)")
		}

		// Build new file content with replacements
		var newContent strings.Builder
		lineIndex := 0

		for _, line := range lines {
			if strings.Contains(line, ":") {
				// Hashline format
				parts := strings.SplitN(line, ":", 2)
				if len(parts) == 2 {
					hash := parts[0]
					content := strings.SplitN(parts[1], "|", 2)
					if len(content) == 2 {
						if lineIndex < len(foundLineHashes) && foundLineHashes[lineIndex] == hash {
							// Replace this line with new content that preserves the same hash
							// This ensures the line can still be identified if modified in the future
							newHash := generateSimpleHash(args.Content)
							newContent.WriteString(fmt.Sprintf("%s:%s|%s\n", hash, newHash, args.Content))
							lineIndex++
						} else {
							newContent.WriteString(line + "\n")
						}
					} else {
						newContent.WriteString(line + "\n")
					}
				} else {
					newContent.WriteString(line + "\n")
				}
			} else {
				// Regular line without hash
				if lineIndex < len(foundLineHashes) && strings.Contains(line, args.OldString) {
					newContent.WriteString(fmt.Sprintf("%s\n", args.Content))
					lineIndex++
				} else {
					newContent.WriteString(line + "\n")
				}
			}
		}

		// Write the modified content back
		if err := os.WriteFile(fullPath, []byte(newContent.String()), 0644); err != nil {
			return "", fmt.Errorf("failed to write modified content: %s", err.Error())
		}

		summary := fmt.Sprintf("Replaced %d occurrence(s) in hashline file", occurrences)

		return fmt.Sprintf(`{"success": true, "summary": %q, "path": %q, "occurrences": %d, "replaced_all": %t}`, summary, args.Path, occurrences, args.ReplaceAll), nil
	}

	// Regular string replacement mode
	// Count total occurrences
	totalOccurrences := strings.Count(fileContent, args.OldString)

	// Perform the replacement
	var newContent string
	var occurrences int
	if args.ReplaceAll {
		newContent = strings.ReplaceAll(fileContent, args.OldString, args.Content)
		occurrences = totalOccurrences
	} else {
		// Replace only the first occurrence
		if idx := strings.Index(fileContent, args.OldString); idx >= 0 {
			newContent = fileContent[:idx] + args.Content + fileContent[idx+len(args.OldString):]
			occurrences = totalOccurrences // Return total count found
		} else {
			return "", fmt.Errorf("old_string not found in file")
		}
	}

	// Write the modified content back
	if err := os.WriteFile(fullPath, []byte(newContent), 0644); err != nil {
		return "", fmt.Errorf("failed to write modified content: %s", err.Error())
	}

	// Generate diff showing the change
	diff := t.generateEditDiff(args.Path, fileContent, newContent, args.OldString, args.Content, occurrences)

	summary := fmt.Sprintf("Replaced %d occurrence(s) of '%s' with '%s'", occurrences, args.OldString, args.Content)

	return fmt.Sprintf(`{"success": true, "summary": %q, "diff": %q, "path": %q, "occurrences": %d, "replaced_all": %t}`, summary, diff, args.Path, occurrences, args.ReplaceAll), nil
}

func (t *FileWriteTool) createBackup(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
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
func (t *FileWriteTool) generateEditDiff(path, oldContent, newContent, oldString, newString string, occurrences int) string {
	var diff strings.Builder

	// Create a unified diff header
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	diff.WriteString(fmt.Sprintf("--- %s\t%s\n", path, timestamp))
	diff.WriteString(fmt.Sprintf("+++ %s\t%s\n", path, timestamp))

	// Simple diff showing the change
	diff.WriteString("@@ -1 +1 @@\n")
	if occurrences > 0 {
		diff.WriteString("-")
		diff.WriteString(oldString)
		diff.WriteString("\n+")
		diff.WriteString(newString)
		diff.WriteString("\n")
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

var _ entities.Tool = (*FileWriteTool)(nil)
