package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiagent/internal/domain/entities"

	"github.com/dustin/go-humanize"
	"github.com/pmezard/go-difflib/difflib"
	"go.uber.org/zap"
)

type FileTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewFileTool(name, description string, configuration map[string]string, logger *zap.Logger) *FileTool {
	return &FileTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *FileTool) Name() string {
	return t.name
}

func (t *FileTool) Description() string {
	return t.description
}

func (t *FileTool) Configuration() map[string]string {
	return t.configuration
}

func (t *FileTool) FullDescription() string {
	var b strings.Builder

	// Add description
	b.WriteString(t.Description())
	b.WriteString("\n\n")

	// Add configuration header
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")

	// Loop through configuration and add key-value pairs to the table
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}

	return b.String()
}

func (t *FileTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"read", "write", "edit", "create_directory", "list_directory", "directory_tree", "move", "search", "get_info"},
			Description: "The file operation to perform",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "The file or directory path",
			Required:    true,
		},
		{
			Name:        "content",
			Type:        "string",
			Description: "Content to write (for write operation)",
			Required:    false,
		},
		{
			Name:        "edits",
			Type:        "array",
			Items:       []entities.Item{{Type: "object"}},
			Description: "Array of edit operations with oldText and newText (for edit operation)",
			Required:    false,
		},
		{
			Name:        "dry_run",
			Type:        "boolean",
			Description: "Preview changes without applying them (for edit operation)",
			Required:    false,
		},
		{
			Name:        "destination",
			Type:        "string",
			Description: "Destination path (for move operation)",
			Required:    false,
		},
		{
			Name:        "pattern",
			Type:        "string",
			Description: "Search pattern (for search operation)",
			Required:    false,
		},
		{
			Name:        "exclude_patterns",
			Type:        "array",
			Items:       []entities.Item{{Type: "string"}},
			Description: "Patterns to exclude (for search operation)",
			Required:    false,
		},
	}
}

type EditOperation struct {
	OldText string `json:"oldText"`
	NewText string `json:"newText"`
}

func (t *FileTool) validatePath(path string) (string, error) {
	workspace := t.configuration["workspace"]
	if workspace == "" {
		t.logger.Error("Workspace configuration is missing")
		return "", nil
	}

	fullPath := filepath.Join(workspace, path)
	rel, err := filepath.Rel(workspace, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.logger.Error("Path is outside workspace", zap.String("path", path))
		return "", nil
	}
	return fullPath, nil
}

func (t *FileTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing file operation", zap.String("arguments", arguments))

	var args struct {
		Operation       string          `json:"operation"`
		Path            string          `json:"path"`
		Content         string          `json:"content"`
		Edits           []EditOperation `json:"edits"`
		DryRun          bool            `json:"dry_run"`
		Destination     string          `json:"destination"`
		Pattern         string          `json:"pattern"`
		ExcludePatterns []string        `json:"exclude_patterns"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	if args.Operation == "" || args.Path == "" {
		t.logger.Error("Operation and path are required")
		return "", fmt.Errorf("operation and path are required")
	}

	switch args.Operation {
	case "read":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.logger.Error("Failed to read file", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		t.logger.Info("File read successfully", zap.String("path", fullPath))
		return string(data), nil

	case "write":
		if args.Content == "" {
			t.logger.Error("Content is required for write operation")
			return "", fmt.Errorf("content is required")
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		err = os.WriteFile(fullPath, []byte(args.Content), 0644)
		if err != nil {
			t.logger.Error("Failed to write file", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		t.logger.Info("File written successfully", zap.String("path", fullPath))
		return "File written successfully", nil

	case "edit":
		if len(args.Edits) == 0 {
			t.logger.Error("Edits array is required for edit operation")
			return "", fmt.Errorf("edits array is required")
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		return t.applyEdits(fullPath, args.Edits, args.DryRun)

	case "create_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			t.logger.Error("Failed to create directory", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		t.logger.Info("Directory created successfully", zap.String("path", fullPath))
		return "Directory created successfully", nil

	case "list_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			t.logger.Error("Failed to list directory", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		var formatted []string
		for _, entry := range entries {
			prefix := "[FILE]"
			if entry.IsDir() {
				prefix = "[DIR]"
			}
			formatted = append(formatted, prefix+" "+entry.Name())
		}
		t.logger.Info("Directory listed successfully", zap.String("path", fullPath))
		return strings.Join(formatted, "\n"), nil

	case "directory_tree":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		tree, err := t.buildDirectoryTree(fullPath)
		if err != nil {
			t.logger.Error("Failed to build directory tree", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		jsonTree, _ := json.MarshalIndent(tree, "", "  ")
		t.logger.Info("Directory tree built successfully", zap.String("path", fullPath))
		return string(jsonTree), nil

	case "move":
		if args.Destination == "" {
			t.logger.Error("Destination is required for move operation")
			return "", fmt.Errorf("destination is required")
		}
		srcPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		dstPath, err := t.validatePath(args.Destination)
		if err != nil {
			return "", err
		}
		err = os.Rename(srcPath, dstPath)
		if err != nil {
			t.logger.Error("Failed to move file", zap.String("source", srcPath), zap.String("dest", dstPath), zap.Error(err))
			return "", err
		}
		t.logger.Info("File moved successfully", zap.String("source", srcPath), zap.String("dest", dstPath))
		return "File moved successfully", nil

	case "search":
		if args.Pattern == "" {
			t.logger.Error("Pattern is required for search operation")
			return "", fmt.Errorf("pattern is required")
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		results, err := t.searchFiles(fullPath, args.Pattern, args.ExcludePatterns)
		if err != nil {
			t.logger.Error("Failed to search files", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		if len(results) == 0 {
			return "No matches found", nil
		}
		t.logger.Info("Files searched successfully", zap.String("path", fullPath), zap.Int("matches", len(results)))
		return strings.Join(results, "\n"), nil

	case "get_info":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			t.logger.Error("Failed to get file info", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		result := []string{
			"size: " + formatSize(info.Size()),
			"created: " + info.ModTime().Format(time.RFC3339), // Note: Go doesn't provide birth time
			"modified: " + info.ModTime().Format(time.RFC3339),
			"accessed: " + info.ModTime().Format(time.RFC3339), // Note: Go doesn't provide last access time
			"isDirectory: " + boolToString(info.IsDir()),
			"isFile: " + boolToString(!info.IsDir()),
			"permissions: " + info.Mode().String(),
		}
		t.logger.Info("File info retrieved successfully", zap.String("path", fullPath))
		return strings.Join(result, "\n"), nil

	default:
		t.logger.Error("Unknown operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unknown operation: %s", args.Operation)
	}
}

func (t *FileTool) applyEdits(filePath string, edits []EditOperation, dryRun bool) (string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.logger.Error("Failed to read file for edit", zap.String("path", filePath), zap.Error(err))
		return "", err
	}
	original := string(content)
	modified := original

	for _, edit := range edits {
		if !strings.Contains(modified, edit.OldText) {
			t.logger.Error("Edit text not found", zap.String("oldText", edit.OldText))
			return "", fmt.Errorf("edit failed: text '%s' not found in file", edit.OldText)
		}
		modified = strings.ReplaceAll(modified, edit.OldText, edit.NewText)
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
		return "", err
	}

	if !dryRun && modified != original {
		err = os.WriteFile(filePath, []byte(modified), 0644)
		if err != nil {
			t.logger.Error("Failed to write edited file", zap.String("path", filePath), zap.Error(err))
			return "", err
		}
		t.logger.Info("File edited successfully", zap.String("path", filePath))
	}

	return "```diff\n" + diffStr + "\n```", nil
}

type TreeEntry struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []TreeEntry `json:"children,omitempty"`
}

func (t *FileTool) buildDirectoryTree(path string) ([]TreeEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var result []TreeEntry
	for _, entry := range entries {
		treeEntry := TreeEntry{
			Name: entry.Name(),
			Type: "file",
		}
		if entry.IsDir() {
			treeEntry.Type = "directory"
			children, err := t.buildDirectoryTree(filepath.Join(path, entry.Name()))
			if err != nil {
				continue // Skip invalid subdirectories
			}
			treeEntry.Children = children
		}
		result = append(result, treeEntry)
	}
	return result, nil
}

func (t *FileTool) searchFiles(rootPath, pattern string, excludePatterns []string) ([]string, error) {
	var results []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(rootPath, path)
		for _, exclude := range excludePatterns {
			if matched, _ := filepath.Match(exclude, relPath); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(pattern)) {
			results = append(results, path)
		}
		return nil
	})
	if err != nil {
		t.logger.Error("Failed to search files", zap.String("path", rootPath), zap.Error(err))
		return nil, err
	}
	if len(results) == 0 {
		t.logger.Warn("No matches found during file search", zap.String("pattern", pattern), zap.String("path", rootPath))
		return nil, fmt.Errorf("no matches found for pattern '%s' in path '%s'", pattern, rootPath)
	}
	t.logger.Info("Files searched successfully", zap.String("path", rootPath), zap.Int("matches", len(results)))
	return results, nil
}

func formatSize(size int64) string {
	return humanize.Bytes(uint64(size))
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

var _ entities.Tool = (*FileTool)(nil)
