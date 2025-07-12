package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

type DirectoryTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

type TreeEntry struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []TreeEntry `json:"children,omitempty"`
}

func NewDirectoryTool(name, description string, configuration map[string]string, logger *zap.Logger) *DirectoryTool {
	return &DirectoryTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *DirectoryTool) Name() string {
	return t.name
}

func (t *DirectoryTool) Description() string {
	return t.description
}

func (t *DirectoryTool) Configuration() map[string]string {
	return t.configuration
}

func (t *DirectoryTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *DirectoryTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("## Usage Instructions\n")
	b.WriteString("This tool supports directory and file management operations:\n")
	b.WriteString("- **create_directory**: Creates a new directory at the specified path.\n")
	b.WriteString("- **list_directory**: Lists files and directories in the specified path.\n")
	b.WriteString("- **directory_tree**: Builds a hierarchical tree of the directory structure.\n")
	b.WriteString("- **move**: Moves a file or directory to a new location.\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *DirectoryTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"create_directory", "list_directory", "directory_tree", "move"},
			Description: "The directory operation to perform",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "The directory or file path",
			Required:    true,
		},
		{
			Name:        "destination",
			Type:        "string",
			Description: "Destination path (for move operation)",
			Required:    false,
		},
	}
}

func (t *DirectoryTool) validatePath(path string) (string, error) {
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

func (t *DirectoryTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing directory command", zap.String("arguments", arguments))
	var args struct {
		Operation   string `json:"operation"`
		Path        string `json:"path"`
		Destination string `json:"destination"`
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
	case "create_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			t.logger.Error("Failed to create directory", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to create directory: %v", err)
		}
		t.logger.Info("Directory created successfully", zap.String("path", fullPath))
		return "Directory created successfully", nil

	case "list_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			t.logger.Error("Failed to list directory", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to list directory: %v", err)
		}
		var formatted []string
		for _, entry := range entries {
			prefix := "[FILE]"
			if entry.IsDir() {
				prefix = "[DIR]"
			}
			formatted = append(formatted, prefix+" "+entry.Name())
		}
		results := strings.Join(formatted, "\n")
		if len(results) > 16384 {
			results = results[:16384] + "...truncated"
		}
		t.logger.Info("Directory listed successfully", zap.String("path", fullPath))
		return results, nil

	case "directory_tree":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		tree, err := t.buildDirectoryTree(fullPath)
		if err != nil {
			t.logger.Error("Failed to build directory tree", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to build directory tree: %v", err)
		}
		jsonTree, _ := json.MarshalIndent(tree, "", "  ")
		results := string(jsonTree)
		if len(results) > 16384 {
			results = results[:16384] + "...truncated"
		}
		t.logger.Info("Directory tree built successfully", zap.String("path", fullPath))
		return results, nil

	case "move":
		if args.Destination == "" {
			t.logger.Error("Destination is required for move operation")
			return "", fmt.Errorf("destination is required")
		}
		srcPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid source path: %v", err)
		}
		dstPath, err := t.validatePath(args.Destination)
		if err != nil {
			return "", fmt.Errorf("invalid destination path: %v", err)
		}
		err = os.Rename(srcPath, dstPath)
		if err != nil {
			t.logger.Error("Failed to move file", zap.String("source", srcPath), zap.String("dest", dstPath), zap.Error(err))
			return "", fmt.Errorf("failed to move file: %v", err)
		}
		t.logger.Info("File moved successfully", zap.String("source", srcPath), zap.String("dest", dstPath))
		return "File moved successfully", nil

	default:
		t.logger.Error("Unknown operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unknown operation: %s", args.Operation)
	}
}

func (t *DirectoryTool) buildDirectoryTree(path string) ([]TreeEntry, error) {
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
				continue
			}
			treeEntry.Children = children
		}
		result = append(result, treeEntry)
	}
	return result, nil
}

var _ entities.Tool = (*DirectoryTool)(nil)
