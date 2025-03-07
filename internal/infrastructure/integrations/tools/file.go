package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"aiagent/internal/domain/interfaces"
)

type FileTool struct {
	configuration map[string]string
}

func NewFileTool(configuration map[string]string) *FileTool {
	return &FileTool{configuration: configuration}
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
	var args struct {
		operation string `json:"operation"`
		path      string `json:"path"`
		content   string `json:"content"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return "", fmt.Errorf("error parsing tool arguments: %v", err)
	}

	operation := args.operation
	if operation == "" {
		return "", fmt.Errorf("operation cannot be empty")
	}
	path := args.path
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}
	content := args.content
	if operation != "read" && content == "" {
		return "", fmt.Errorf("content cannot be empty for %s operation", operation)
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		return "", fmt.Errorf("workspace configuration is required")
	}

	fullPath := filepath.Join(workspace, path)
	rel, err := filepath.Rel(workspace, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("path is outside workspace: %s", path)
	}

	switch operation {
	case "read":
		data, err := os.ReadFile(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to read file %s: %w", fullPath, err)
		}
		return string(data), nil

	case "write":
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write to file %s: %w", fullPath, err)
		}
		return "File written successfully", nil

	case "patch":
		tempFile, err := os.CreateTemp("", "diff-*.txt")
		if err != nil {
			return "", fmt.Errorf("failed to create temporary file for diff: %w", err)
		}
		defer os.Remove(tempFile.Name())

		_, err = tempFile.WriteString(content)
		if err != nil {
			tempFile.Close()
			return "", fmt.Errorf("failed to write diff to temporary file: %w", err)
		}
		tempFile.Close()

		cmd := exec.Command("patch", fullPath, tempFile.Name())
		var out bytes.Buffer
		var stderr bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &stderr
		err = cmd.Run()
		if err != nil {
			return "", fmt.Errorf("patch failed for %s: %s\n%s", fullPath, err, stderr.String())
		}
		return "File patched successfully", nil

	default:
		return "", fmt.Errorf("unknown operation: %s", operation)
	}
}
