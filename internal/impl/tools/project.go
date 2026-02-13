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

type ProjectTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewProjectTool(name, description string, configuration map[string]string, logger *zap.Logger) *ProjectTool {
	return &ProjectTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *ProjectTool) Name() string {
	return t.name
}

func (t *ProjectTool) Description() string {
	return t.description
}

func (t *ProjectTool) Configuration() map[string]string {
	return t.configuration
}

func (t *ProjectTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *ProjectTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- operation: 'read' or 'structure'\n- 'read' reads project description file\n- 'structure' shows directory tree layout", t.Description())
}

func (t *ProjectTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "The operation to perform ('read' for project file, 'structure' for directory layout)",
				"enum":        []string{"read", "structure"},
			},
		},
		"required": []string{"operation"},
	}
}

func (t *ProjectTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing project tool", zap.String("arguments", arguments))

	var args struct {
		Operation string `json:"operation"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	// Get workspace
	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			t.logger.Error("Could not get current directory", zap.Error(err))
			return "", fmt.Errorf("could not get current directory: %v", err)
		}
	}

	switch args.Operation {
	case "read":
		return t.executeRead(workspace)
	case "structure":
		return t.executeStructure(workspace)
	default:
		t.logger.Error("Invalid operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("invalid operation: %s", args.Operation)
	}
}

func (t *ProjectTool) executeRead(workspace string) (string, error) {
	// Get the project file path from configuration
	projectFile, ok := t.configuration["project_file"]
	if !ok || projectFile == "" {
		t.logger.Error("Project file not configured")
		return "", fmt.Errorf("project_file configuration is required")
	}

	fullPath := filepath.Join(workspace, projectFile)
	rel, err := filepath.Rel(workspace, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.logger.Error("Project file path is outside workspace", zap.String("path", projectFile))
		return "", fmt.Errorf("project file path is outside workspace")
	}

	// Check if the file exists, create it with default content if it doesn't
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.logger.Info("Project file does not exist, creating with default content", zap.String("path", fullPath))
		defaultContent := `# Project Details

This is a default project description file created automatically by the ProjectTool.
Please update this file with relevant project information.

## Overview
- Project Name: [Your Project Name]
- Description: [Brief description of the project]
- Repository: [URL to the project repository, if applicable]

## Instructions
- [Add specific instructions for AI agents like GitHub Copilot or Claude]

## Additional Notes
- [Add any additional context or notes]
`
		if err := os.WriteFile(fullPath, []byte(defaultContent), 0644); err != nil {
			t.logger.Error("Failed to create project file", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to create project file: %v", err)
		}
		t.logger.Info("Project file created successfully", zap.String("path", fullPath))
	} else if err != nil {
		t.logger.Error("Failed to check project file", zap.String("path", fullPath), zap.Error(err))
		return "", fmt.Errorf("failed to check project file: %v", err)
	}

	// Read the project file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.logger.Error("Failed to read project file", zap.String("path", fullPath), zap.Error(err))
		return "", fmt.Errorf("failed to read project file: %v", err)
	}

	contentStr := string(content)

	// Create TUI-friendly summary (first 5 lines only)
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📄 Project File: %s\n\n", filepath.Base(fullPath)))

	lines := strings.Split(contentStr, "\n")
	previewLines := 5
	if len(lines) < previewLines {
		previewLines = len(lines)
	}

	for i := 0; i < previewLines; i++ {
		if lines[i] != "" {
			summary.WriteString(fmt.Sprintf("  %s\n", lines[i]))
		}
	}

	if len(lines) > 5 {
		summary.WriteString(fmt.Sprintf("  ... and %d more lines\n", len(lines)-5))
	}

	// Create JSON response with summary for TUI and full content for AI
	response := struct {
		Summary string `json:"summary"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}{
		Summary: summary.String(),
		Path:    fullPath,
		Content: contentStr,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal project read response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	t.logger.Info("Project file read successfully", zap.String("path", fullPath))
	return string(jsonResult), nil
}

func (t *ProjectTool) executeStructure(workspace string) (string, error) {
	// Build directory tree (ignores .git and common non-source directories)
	tree, err := buildDirectoryTree(workspace)
	if err != nil {
		t.logger.Error("Failed to build directory tree", zap.Error(err))
		return "", fmt.Errorf("failed to build directory tree: %v", err)
	}

	// Create TUI-friendly summary
	var summary strings.Builder
	summary.WriteString("📁 Project Structure:\n\n")
	summary.WriteString(tree)

	// Create JSON response with summary for TUI and tree for AI
	response := struct {
		Summary string `json:"summary"`
		Tree    string `json:"tree"`
		Path    string `json:"path"`
	}{
		Summary: summary.String(),
		Tree:    tree,
		Path:    workspace,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal structure response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	t.logger.Info("Directory structure retrieved successfully", zap.String("workspace", workspace))
	return string(jsonResult), nil
}

// buildDirectoryTree generates a tree representation of the directory (ignores .git and common build directories)
func buildDirectoryTree(root string) (string, error) {
	var buf strings.Builder
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			buf.WriteString(".\n")
			return nil
		}

		// Skip .git, .aiagent, and common non-source directories
		if info.IsDir() && (info.Name() == ".git" || info.Name() == ".aiagent" ||
			info.Name() == "node_modules" || info.Name() == "bin" ||
			info.Name() == "dist" || info.Name() == "build" ||
			info.Name() == "target" || info.Name() == ".next" ||
			info.Name() == ".nuxt" || info.Name() == ".vuepress" ||
			info.Name() == "__pycache__" || info.Name() == ".pytest_cache") {
			return filepath.SkipDir
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		prefix := strings.Repeat("  ", depth)
		if info.IsDir() {
			buf.WriteString(fmt.Sprintf("%s📁 %s/\n", prefix, info.Name()))
		} else {
			// Show file extension for quick identification
			ext := filepath.Ext(info.Name())
			if ext != "" {
				buf.WriteString(fmt.Sprintf("%s📄 %s\n", prefix, info.Name()))
			} else {
				buf.WriteString(fmt.Sprintf("%s📄 %s\n", prefix, info.Name()))
			}
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

var _ entities.Tool = (*ProjectTool)(nil)
