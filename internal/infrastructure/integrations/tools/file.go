package tools

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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
	return "A tool to read, write or patch files"
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
			Enum:        []string{"read", "write", "patch"},
			Description: "read, write or patch operation",
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
			Description: "The content to write or the diff to patch",
			Required:    false,
		},
	}
}

func (t *FileTool) Execute(arguments string) (string, error) {
	// Log the file operation arguments
	t.logger.Debug("Executing file operation", zap.String("arguments", arguments))

	var args struct {
		Operation string `json:"operation"`
		Path      string `json:"path"`
		Content   string `json:"content"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}

	operation := args.Operation
	path := args.Path
	content := args.Content

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
		if content == "" {
			t.logger.Error("Content is required for write operation")
			return "", nil
		}
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			t.logger.Error("Failed to write file",
				zap.String("path", fullPath),
				zap.Error(err))
			return "", err
		}
		t.logger.Info("File written successfully", zap.String("path", fullPath))
		return "File written successfully", nil

	case "patch":
		if content == "" {
			t.logger.Error("Content is required for patch operation")
			return "", nil
		}
		tempFile, err := os.CreateTemp("", "diff-*.txt")
		if err != nil {
			t.logger.Error("Failed to create temp file for patch", zap.Error(err))
			return "", err
		}
		//defer os.Remove(tempFile.Name())
		t.logger.Debug("TEMP: " + tempFile.Name())

		_, err = tempFile.WriteString(content)
		if err != nil {
			tempFile.Close()
			t.logger.Error("Failed to write diff to temp file", zap.Error(err))
			return "", err
		}
		tempFile.Close()

		cmd := exec.Command("patch", fullPath, tempFile.Name())
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			t.logger.Error("Patch operation failed",
				zap.String("path", fullPath),
				zap.Error(err),
				zap.String("stderr", stderr.String()))
			return "", err
		}
		t.logger.Info("File patched successfully", zap.String("path", fullPath))
		return "File patched successfully", nil

	default:
		t.logger.Error("Unknown operation", zap.String("operation", operation))
		return "", nil
	}
}
