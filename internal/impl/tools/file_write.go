package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
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
	b.WriteString("This tool supports writing and modifying text files. **Critical**: Follow these steps to avoid errors:\n")
	b.WriteString("1. Use the FileReadTool to confirm the exact line number and surrounding content before making changes.\n")
	b.WriteString("2. Use `dry_run=true` with `edit`, `insert`, or `delete` to preview changes and verify the line is correct.\n")
	b.WriteString("3. After any change, use FileReadTool to check the updated file and get new line numbers, as insertions or deletions shift lines.\n\n")
	b.WriteString("- **write**: Overwrites or creates a file with new content. Provide `content` to specify the full file content.\n")
	b.WriteString("- **edit**: Replaces specific lines in a file with new content. Use `start_line`, `end_line` (optional, defaults to start_line), and `content` to replace the specified lines.\n")
	b.WriteString("  - Example: To replace lines 5 to 7, set `operation='edit', start_line=5, end_line=7`, and provide the new `content` for those lines. If end_line is omitted, only line 5 is replaced.\n")
	b.WriteString("- **insert**: Inserts new content at a specific line. Use `start_line` and `content` to insert the content before the specified line.\n")
	b.WriteString("  - Example: To insert content at line 5, set `operation='insert', start_line=5`, and provide the `content` to insert.\n")
	b.WriteString("- **delete**: Deletes specific lines in a file. Use `start_line` and `end_line` to specify the lines to delete.\n")
	b.WriteString("  - Example: To delete lines 5 to 7, set `operation='delete', start_line=5, end_line=7`.\n")
	b.WriteString("  - Use `dry_run=true` to preview changes without applying them for `edit`, `insert`, or `delete` operations.\n")
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
			Enum:        []string{"write", "edit", "insert", "delete"},
			Description: "The file write operation to perform",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "The file path",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "Content to write, edit, or insert",
			Required:    true,
		},
		{
			Name:        "start_line",
			Type:        "integer",
			Description: "The start line for editing, inserting, or deleting",
			Required:    false,
		},
		{
			Name:        "end_line",
			Type:        "integer",
			Description: "The end line for editing or deleting (optional for edit, defaults to start_line)",
			Required:    false,
		},
		{
			Name:        "dry_run",
			Type:        "boolean",
			Description: "Preview changes without applying them (for edit, insert, or delete operations)",
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
	t.logger.Debug("Executing file write command", zap.String("arguments", arguments))
	var args struct {
		Operation string `json:"operation"`
		Path      string `json:"path"`
		Content   string `json:"content"`
		DryRun    bool   `json:"dry_run"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Operation == "" {
		t.logger.Error("Operation is required")
		return "", fmt.Errorf("operation is required")
	}
	if args.Path == "" {
		t.logger.Error("Path is required")
		return "", fmt.Errorf("path is required")
	}

	switch args.Operation {
	case "write":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(args.Content), 0644)
		if err != nil {
			t.logger.Error("Failed to write file", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to write file: %v", err)
		}
		t.logger.Info("File written successfully", zap.String("path", fullPath))
		return "File written successfully", nil

	case "edit", "insert", "delete":
		if args.StartLine == 0 {
			args.StartLine = 1
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		if args.Operation == "edit" && args.EndLine == 0 {
			args.EndLine = args.StartLine
		}
		if args.Operation == "edit" && args.Content == "" {
			t.logger.Error("Content is required for edit operation")
			return "", fmt.Errorf("content is required for edit operation")
		}
		if args.Operation == "insert" && args.Content == "" {
			t.logger.Error("Content is required for insert operation")
			return "", fmt.Errorf("content is required for insert operation")
		}
		if args.Operation == "delete" && args.EndLine == 0 {
			args.EndLine = args.StartLine
		}
		results, err := t.applyLineEdit(fullPath, args.Operation, args.StartLine, args.EndLine, args.Content, args.DryRun)
		return results, err

	default:
		t.logger.Error("Unknown operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unknown operation: %s", args.Operation)
	}
}

func (t *FileWriteTool) applyLineEdit(filePath, operation string, startLine, endLine int, content string, dryRun bool) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		t.logger.Error("Failed to read file for edit", zap.String("path", filePath), zap.Error(err))
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	defer file.Close()

	var originalLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		originalLines = append(originalLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.logger.Error("Error reading file lines", zap.Error(err))
		return "", fmt.Errorf("error reading file lines: %v", err)
	}

	if startLine > len(originalLines)+1 || startLine < 1 {
		t.logger.Error("Invalid start line", zap.Int("start_line", startLine), zap.Int("file_lines", len(originalLines)))
		return "", fmt.Errorf("start_line %d is invalid, must be between 1 and %d. Use 'read' to get current line numbers", startLine, len(originalLines)+1)
	}
	if (operation == "edit" || operation == "delete") && endLine > len(originalLines) {
		t.logger.Error("End line exceeds file length", zap.Int("end_line", endLine), zap.Int("file_lines", len(originalLines)))
		return "", fmt.Errorf("end_line %d exceeds file length %d. Use 'read' to get current line numbers", endLine, len(originalLines))
	}
	if (operation == "edit" || operation == "delete") && endLine < startLine {
		t.logger.Error("End line is less than start line", zap.Int("start_line", startLine), zap.Int("end_line", endLine))
		return "", fmt.Errorf("end_line %d is less than start_line %d", endLine, startLine)
	}

	var modifiedLines []string
	switch operation {
	case "edit":
		newContentLines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		modifiedLines = append(modifiedLines, originalLines[:startLine-1]...)
		modifiedLines = append(modifiedLines, newContentLines...)
		if endLine < len(originalLines) {
			modifiedLines = append(modifiedLines, originalLines[endLine:]...)
		}
	case "insert":
		newContentLines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		modifiedLines = append(modifiedLines, originalLines[:startLine-1]...)
		modifiedLines = append(modifiedLines, newContentLines...)
		modifiedLines = append(modifiedLines, originalLines[startLine-1:]...)
	case "delete":
		modifiedLines = append(modifiedLines, originalLines[:startLine-1]...)
		if endLine < len(originalLines) {
			modifiedLines = append(modifiedLines, originalLines[endLine:]...)
		}
	}

	original := strings.Join(originalLines, "\n")
	modified := strings.Join(modifiedLines, "\n")

	if modified == original {
		t.logger.Warn("No changes made to the file", zap.String("path", filePath))
		return fmt.Sprintf("No changes made to the file\nLine count: %d", len(originalLines)), nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(original),
		B:        difflib.SplitLines(modified),
		FromFile: filePath,
		ToFile:   filePath,
		Context:  3,
	}
	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.logger.Error("Failed to generate diff", zap.Error(err))
		return "", fmt.Errorf("failed to generate diff: %v", err)
	}

	if !dryRun {
		err = os.WriteFile(filePath, []byte(modified+"\n"), 0644)
		if err != nil {
			t.logger.Error("Failed to write edited file", zap.String("path", filePath), zap.Error(err))
			return "", fmt.Errorf("failed to write to file: %v", err)
		}
		t.logger.Info("File edited successfully", zap.String("path", filePath), zap.String("operation", operation))
	}

	result := struct {
		Diff      string `json:"diff"`
		LineCount int    `json:"line_count"`
	}{
		Diff:      "```diff\n" + diffStr + "\n```",
		LineCount: len(modifiedLines),
	}
	jsonResponse, err := json.Marshal(result)
	if err != nil {
		t.logger.Error("Failed to marshal edit results", zap.Error(err))
		return "", fmt.Errorf("failed to marshal edit results: %v", err)
	}

	return string(jsonResponse), nil
}

var _ entities.Tool = (*FileWriteTool)(nil)
