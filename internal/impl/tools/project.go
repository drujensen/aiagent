package tools

import (
	"aiagent/internal/domain/entities"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

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
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *ProjectTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"read"},
			Description: "The operation to perform (currently only 'read' is supported)",
			Required:    true,
		},
	}
}

func (t *ProjectTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing project tool", zap.String("arguments", arguments))
	fmt.Println("\rExecuting project tool", arguments)

	var args struct {
		Operation string `json:"operation"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Operation != "read" {
		t.logger.Error("Invalid operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("invalid operation: %s, only 'read' is supported", args.Operation)
	}

	// Get the project file path from configuration
	projectFile, ok := t.configuration["project_file"]
	if !ok || projectFile == "" {
		t.logger.Error("Project file not configured")
		return "", fmt.Errorf("project_file configuration is required")
	}

	// Resolve the full path relative to workspace
	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			t.logger.Error("Could not get current directory", zap.Error(err))
			return "", fmt.Errorf("could not get current directory: %v", err)
		}
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

	// Return the file contents as a JSON response
	response := struct {
		Content string `json:"content"`
	}{
		Content: string(content),
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	t.logger.Info("Project file read successfully", zap.String("path", fullPath))
	return string(jsonResponse), nil
}

var _ entities.Tool = (*ProjectTool)(nil)
