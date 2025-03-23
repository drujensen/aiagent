package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type FileTool struct {
	configuration map[string]string
	logger        *zap.Logger
}

func NewFileTool(configuration map[string]string, logger *zap.Logger) *FileTool {
	return &FileTool{
		configuration: configuration,
		logger:        logger,
	}
}

func (t *FileTool) Name() string {
	return "File"
}

func (t *FileTool) Description() string {
	return "A tool to read, write, find, or replace content in files"
}

func (t *FileTool) Configuration() []string {
	return []string{
		"workspace",
	}
}

func (t *FileTool) Parameters() []interfaces.Parameter {
	return []interfaces.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"read", "write", "find", "replace"},
			Description: "read, write, find, or replace operation",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "The file path",
			Required:    true,
		},
		{
			Name:        "pattern",
			Type:        "string",
			Description: "The regex pattern to find or replace (for find/replace operations)",
			Required:    false,
		},
		{
			Name:        "replacement",
			Type:        "string",
			Description: "The content to replace the matched pattern (for replace operation)",
			Required:    false,
		},
	}
}

func (t *FileTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing file operation", zap.String("arguments", arguments))

	var args struct {
		Operation   string `json:"operation"`
		Path        string `json:"path"`
		Pattern     string `json:"pattern"`
		Replacement string `json:"replacement"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	operation := args.Operation
	path := args.Path
	pattern := args.Pattern
	replacement := args.Replacement

	if operation == "" || path == "" {
		t.logger.Error("Operation and path are required")
		return "", nil
	}

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

	switch operation {
	case "read":
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.logger.Error("Failed to read file",
				zap.String("path", fullPath),
				zap.Error(err))
			return "", err
		}
		t.logger.Info("File read successfully", zap.String("path", fullPath))
		return string(data), nil

	case "write":
		if replacement == "" {
			t.logger.Error("Replacement content is required for write operation")
			return "", nil
		}
		err := os.WriteFile(fullPath, []byte(replacement), 0644)
		if err != nil {
			t.logger.Error("Failed to write file",
				zap.String("path", fullPath),
				zap.Error(err))
			return "", err
		}
		t.logger.Info("File written successfully", zap.String("path", fullPath))
		return "File written successfully", nil

	case "find":
		if pattern == "" {
			t.logger.Error("Pattern is required for find operation")
			return "", nil
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.logger.Error("Failed to read file for find",
				zap.String("path", fullPath),
				zap.Error(err))
			return "", err
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.logger.Error("Invalid regex pattern",
				zap.String("pattern", pattern),
				zap.Error(err))
			return "", err
		}
		matches := re.FindAllString(string(data), -1)
		if len(matches) == 0 {
			t.logger.Info("No matches found",
				zap.String("path", fullPath),
				zap.String("pattern", pattern))
			return "No matches found", nil
		}
		result := strings.Join(matches, "\n")
		t.logger.Info("Matches found",
			zap.String("path", fullPath),
			zap.String("pattern", pattern),
			zap.Int("count", len(matches)))
		return result, nil

	case "replace":
		if pattern == "" || replacement == "" {
			t.logger.Error("Both pattern and replacement are required for replace operation")
			return "", nil
		}
		data, err := os.ReadFile(fullPath)
		if err != nil {
			t.logger.Error("Failed to read file for replace",
				zap.String("path", fullPath),
				zap.Error(err))
			return "", err
		}
		re, err := regexp.Compile(pattern)
		if err != nil {
			t.logger.Error("Invalid regex pattern",
				zap.String("pattern", pattern),
				zap.Error(err))
			return "", err
		}
		original := string(data)
		newContent := re.ReplaceAllString(original, replacement)
		if newContent == original {
			t.logger.Info("No replacements made",
				zap.String("path", fullPath),
				zap.String("pattern", pattern))
			return "No replacements made", nil
		}
		err = os.WriteFile(fullPath, []byte(newContent), 0644)
		if err != nil {
			t.logger.Error("Failed to write updated file",
				zap.String("path", fullPath),
				zap.Error(err))
			return "", err
		}
		t.logger.Info("File replaced successfully",
			zap.String("path", fullPath),
			zap.String("pattern", pattern))
		return "File replaced successfully", nil

	default:
		t.logger.Error("Unknown operation", zap.String("operation", operation))
		return "", nil
	}
}
