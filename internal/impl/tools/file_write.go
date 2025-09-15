package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("## Usage Instructions\n")
	b.WriteString("This tool provides PRECISE file editing using exact string matching. **Critical**: Always read the file first to get the exact content.\n")
	b.WriteString("1. Use FileReadTool to get the current file content and exact strings to replace\n")
	b.WriteString("2. Provide the EXACT old_string including whitespace, indentation, and line breaks\n")
	b.WriteString("3. The replacement will ONLY occur if old_string matches exactly\n")
	b.WriteString("4. Use replace_all=true to replace all occurrences, false for first occurrence only\n")
	b.WriteString("5. If you get 'old_string not found', re-read the file to get the exact content\n\n")
	b.WriteString("- **write**: Overwrites or creates a file with new content\n")
	b.WriteString("- **edit**: Replace exact string matches in a file (RECOMMENDED)\n")
	b.WriteString("  - Requires: path, old_string, new_string\n")
	b.WriteString("  - Optional: replace_all (defaults to false)\n")
	b.WriteString("- **Safety**: Only replaces when strings match exactly\n")
	b.WriteString("- **Precision**: No line number drift or positioning errors\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
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
	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current directory: %v", err)
		}
	}
	fullPath := filepath.Join(workspace, path)
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
		err = os.WriteFile(fullPath, []byte(args.Content), 0644)
		if err != nil {
			t.logger.Error("Failed to write file", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to write file: %v", err)
		}
		t.logger.Info("File written successfully", zap.String("path", fullPath))
		return "File written successfully", nil

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
		return "", fmt.Errorf("failed to read file: %v", err)
	}

	fileContent := string(content)

	// Check if old_string exists
	if !strings.Contains(fileContent, oldString) {
		t.logger.Error("old_string not found in file", zap.String("path", filePath))
		return "", fmt.Errorf("old_string not found in file - ensure exact match including whitespace and indentation. Use FileReadTool to get the exact content")
	}

	// Count occurrences
	occurrences := strings.Count(fileContent, oldString)

	// Perform replacement
	var newContent string
	if replaceAll {
		newContent = strings.ReplaceAll(fileContent, oldString, newString)
	} else {
		newContent = strings.Replace(fileContent, oldString, newString, 1)
	}

	// Generate diff
	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(fileContent),
		B:        difflib.SplitLines(newContent),
		FromFile: filePath,
		ToFile:   filePath,
		Context:  3,
	}
	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.logger.Error("Failed to generate diff", zap.Error(err))
		return "", fmt.Errorf("failed to generate diff: %v", err)
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

	result := struct {
		Success     bool   `json:"success"`
		Path        string `json:"path"`
		Occurrences int    `json:"occurrences"`
		ReplacedAll bool   `json:"replaced_all"`
		Diff        string `json:"diff"`
	}{
		Success:     true,
		Path:        filePath,
		Occurrences: occurrences,
		ReplacedAll: replaceAll,
		Diff:        "```diff\n" + diffStr + "\n```",
	}

	jsonResponse, err := json.Marshal(result)
	if err != nil {
		t.logger.Error("Failed to marshal edit results", zap.Error(err))
		return "", fmt.Errorf("failed to marshal edit results: %v", err)
	}

	return string(jsonResponse), nil
}

var _ entities.Tool = (*FileWriteTool)(nil)
