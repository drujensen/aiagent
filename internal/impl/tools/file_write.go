package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"github.com/pmezard/go-difflib/difflib"
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

func (t *FileWriteTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- operation: 'write' or 'edit'\n- path: file path (absolute or relative to workspace)\n- content: file content (for write)\n- old_string, new_string: exact strings to replace (for edit)\n- replace_all: boolean (optional)", t.Description())
}

func (t *FileWriteTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"write", "edit"},
			Description: "The file operation to perform (write or edit)",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "The file path to edit",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "Content to write (for write operation) or new_string (for edit operation)",
			Required:    true,
		},
		{
			Name:        "old_string",
			Type:        "string",
			Description: "The exact string to replace (for edit operation only)",
			Required:    false,
		},
		{
			Name:        "replace_all",
			Type:        "boolean",
			Description: "Replace all occurrences (for edit operation, defaults to false)",
			Required:    false,
		},
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
	t.logger.Debug("Executing precise file write command", zap.String("arguments", arguments))

	var args struct {
		Operation  string `json:"operation"`
		Path       string `json:"path"`
		Content    string `json:"content"`
		OldString  string `json:"old_string"`
		ReplaceAll bool   `json:"replace_all"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Operation == "" {
		return "", fmt.Errorf("operation is required")
	}
	if args.Path == "" {
		return "", fmt.Errorf("path is required")
	}

	fullPath, err := t.validatePath(args.Path)
	if err != nil {
		return "", fmt.Errorf("invalid path: %v", err)
	}

	switch args.Operation {
	case "write":
		if args.Content == "" {
			return "", fmt.Errorf("content is required for write operation")
		}

		// Read existing content for diff generation
		var oldContent string
		if existingContent, err := os.ReadFile(fullPath); err == nil {
			if utf8.Valid(existingContent) {
				oldContent = string(existingContent)
			} else {
				oldContent = fmt.Sprintf("Binary file: %s", filepath.Base(fullPath))
			}
		} else {
			// File doesn't exist, old content is empty
			oldContent = ""
		}

		newContent := args.Content

		// Generate diff
		sanitizedPath := filepath.Base(fullPath)
		if !utf8.ValidString(sanitizedPath) {
			sanitizedPath = "file"
		}
		diff := difflib.UnifiedDiff{
			A:        difflib.SplitLines(oldContent),
			B:        difflib.SplitLines(newContent),
			FromFile: sanitizedPath,
			ToFile:   sanitizedPath,
			Context:  3,
		}
		diffStr, err := difflib.GetUnifiedDiffString(diff)
		if err != nil {
			t.logger.Error("Failed to generate diff", zap.Error(err))
			diffStr = fmt.Sprintf("File written: %s", filepath.Base(fullPath))
		}

		// Ensure diffStr is valid UTF-8
		if !utf8.ValidString(diffStr) {
			t.logger.Warn("Generated diff contains invalid UTF-8, using placeholder", zap.String("path", fullPath))
			diffStr = fmt.Sprintf("File written: %s", filepath.Base(fullPath))
		}

		err = os.WriteFile(fullPath, []byte(args.Content), 0644)
		if err != nil {
			t.logger.Error("Failed to write file", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to write file: %v", err)
		}

		t.logger.Info("File written successfully", zap.String("path", fullPath))

		// Create TUI-friendly summary with diff preview
		var summary strings.Builder
		if oldContent == "" {
			summary.WriteString(fmt.Sprintf("✏️  File Created: %s\n", filepath.Base(fullPath)))
		} else {
			summary.WriteString(fmt.Sprintf("✏️  File Overwritten: %s\n", filepath.Base(fullPath)))
		}

		// Add a preview of the diff for TUI
		lines := strings.Split(diffStr, "\n")
		previewLines := 0
		maxPreviewLines := 20

		for _, line := range lines {
			if line != "" {
				if previewLines >= maxPreviewLines {
					summary.WriteString("... (diff truncated)\n")
					break
				}
				// Add color indicators for TUI
				if strings.HasPrefix(line, "+") {
					summary.WriteString("🟢 " + line + "\n")
				} else if strings.HasPrefix(line, "-") {
					summary.WriteString("🔴 " + line + "\n")
				} else if strings.HasPrefix(line, "@@") {
					summary.WriteString("🔵 " + line + "\n")
				} else if strings.HasPrefix(line, " ") {
					summary.WriteString("   " + line + "\n")
				} else {
					summary.WriteString(line + "\n")
				}
				previewLines++
			}
		}

		// Truncate diff if too long to prevent token bloat
		const maxDiffLength = 10000 // 10KB limit for diff
		if len(diffStr) > maxDiffLength {
			diffStr = diffStr[:maxDiffLength] + "\n\n[Diff truncated due to length]"
		}

		// Create JSON response with summary for TUI and full data for AI
		response := map[string]interface{}{
			"summary":   summary.String(),
			"success":   true,
			"path":      fullPath,
			"operation": "write",
			"diff":      diffStr,
		}

		jsonResult, err := json.Marshal(response)
		if err != nil {
			t.logger.Error("Failed to marshal file write response", zap.Error(err))
			return summary.String(), nil
		}

		return string(jsonResult), nil

	case "edit":
		if args.OldString == "" {
			return "", fmt.Errorf("old_string is required for edit operation")
		}
		if args.Content == "" {
			return "", fmt.Errorf("content (new_string) is required for edit operation")
		}
		return t.applyPreciseEdit(fullPath, args.OldString, args.Content, args.ReplaceAll)

	default:
		t.logger.Error("Unknown operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unknown operation: %s (supported: write, edit)", args.Operation)
	}
}

func (t *FileWriteTool) applyPreciseEdit(filePath, oldString, newString string, replaceAll bool) (string, error) {
	// Read the file
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.logger.Error("Failed to read file", zap.String("path", filePath), zap.Error(err))
		// Provide helpful guidance for file not found errors
		if os.IsNotExist(err) {
			return "", fmt.Errorf("file does not exist: %s", filePath)
		}
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	// Declare variables at function scope
	var fileContent string
	var newContent string
	var occurrences int
	var diffStr string

	// Ensure content is valid UTF-8
	if !utf8.Valid(content) {
		t.logger.Warn("File contains invalid UTF-8, treating as binary", zap.String("path", filePath))
		// For binary files, don't generate diff
		fileContent = fmt.Sprintf("Binary file: %s", filepath.Base(filePath))
		newContent = fileContent
		occurrences = 1 // Pretend we made a change
		diffStr = fmt.Sprintf("Binary file modified: %s", filepath.Base(filePath))
	} else {
		fileContent = string(content)
	}

	// Check if old_string exists
	if !strings.Contains(fileContent, oldString) {
		t.logger.Error("old_string not found in file", zap.String("path", filePath))
		return "", fmt.Errorf("old_string not found in file")
	}

	// Count occurrences
	occurrences = strings.Count(fileContent, oldString)

	// Perform replacement
	if replaceAll {
		newContent = strings.ReplaceAll(fileContent, oldString, newString)
	} else {
		newContent = strings.Replace(fileContent, oldString, newString, 1)
	}

	// Generate diff
	// Sanitize file path for diff header
	sanitizedPath := filepath.Base(filePath)
	if !utf8.ValidString(sanitizedPath) {
		sanitizedPath = "file"
	}
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(fileContent),
		B:        difflib.SplitLines(newContent),
		FromFile: sanitizedPath,
		ToFile:   sanitizedPath,
		Context:  3,
	}
	diffStr, err = difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.logger.Error("Failed to generate diff", zap.Error(err))
		return "", fmt.Errorf("failed to generate diff: %v", err)
	}

	// Ensure diffStr is valid UTF-8
	if !utf8.ValidString(diffStr) {
		t.logger.Warn("Generated diff contains invalid UTF-8, using placeholder", zap.String("path", filePath))
		diffStr = fmt.Sprintf("File modified: %s", filepath.Base(filePath))
	}

	// Truncate diffStr if extremely large (to prevent memory issues)
	if len(diffStr) > 200000 {
		t.logger.Warn("Generated diff is extremely large, truncating", zap.String("path", filePath), zap.Int("size", len(diffStr)))
		lines := strings.Split(diffStr, "\n")
		truncatedLines := lines
		if len(lines) > 500 {
			truncatedLines = lines[:500]
		}
		diffStr = strings.Join(truncatedLines, "\n") + "\n... (diff truncated)"
	}

	// Write the file
	err = os.WriteFile(filePath, []byte(newContent), 0644)
	if err != nil {
		t.logger.Error("Failed to write file", zap.String("path", filePath), zap.Error(err))
		return "", fmt.Errorf("failed to write file: %v", err)
	}

	// Auto-format Go files
	if strings.HasSuffix(filePath, ".go") {
		if err := exec.Command("go", "fmt", filePath).Run(); err != nil {
			t.logger.Warn("Failed to run go fmt", zap.String("path", filePath), zap.Error(err))
			// Don't fail the edit, just warn
		}
	}

	t.logger.Info("File edited successfully",
		zap.String("path", filePath),
		zap.Int("occurrences", occurrences),
		zap.Bool("replace_all", replaceAll))

	// Create TUI-friendly summary with diff preview
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("✏️  File Edit: %s\n", filepath.Base(filePath)))

	if occurrences == 0 {
		summary.WriteString("No changes made - pattern not found")
	} else {
		action := "replaced"
		if replaceAll {
			action = "replaced all"
		}
		summary.WriteString(fmt.Sprintf("✅ %s %d occurrence(s)\n\n", action, occurrences))

		// Add a preview of the diff for TUI
		lines := strings.Split(diffStr, "\n")
		previewLines := 0

		for _, line := range lines {
			if line != "" {
				// Add color indicators for TUI
				if strings.HasPrefix(line, "+") {
					summary.WriteString("🟢 " + line + "\n")
				} else if strings.HasPrefix(line, "-") {
					summary.WriteString("🔴 " + line + "\n")
				} else if strings.HasPrefix(line, "@@") {
					summary.WriteString("🔵 " + line + "\n")
				} else if strings.HasPrefix(line, " ") {
					summary.WriteString("   " + line + "\n")
				} else {
					summary.WriteString(line + "\n")
				}
				previewLines++
			}
		}
	}

	// Truncate diff if too long to prevent token bloat
	const maxDiffLength = 10000 // 10KB limit for diff
	if len(diffStr) > maxDiffLength {
		diffStr = diffStr[:maxDiffLength] + "\n\n[Diff truncated due to length]"
	}

	// Create JSON response with summary for TUI and full data for AI
	response := map[string]interface{}{
		"summary":      summary.String(),
		"success":      true,
		"path":         filePath,
		"occurrences":  occurrences,
		"replaced_all": replaceAll,
		"diff":         diffStr,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal file write response", zap.Error(err))
		// Try to marshal without the diff
		fallbackResponse := map[string]interface{}{
			"summary":      summary.String(),
			"success":      true,
			"path":         filePath,
			"occurrences":  occurrences,
			"replaced_all": replaceAll,
			"diff":         "Diff too large or invalid",
		}
		fallbackJson, err2 := json.Marshal(fallbackResponse)
		if err2 != nil {
			t.logger.Error("Failed to marshal fallback response", zap.Error(err2))
			return summary.String(), nil // Fallback to summary only
		}
		return string(fallbackJson), nil
	}

	return string(jsonResult), nil
}

var _ entities.Tool = (*FileWriteTool)(nil)
