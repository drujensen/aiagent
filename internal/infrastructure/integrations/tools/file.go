package tools

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type FileTool struct {
	workspace string
}

func NewFileTool(workspace string) *FileTool {
	return &FileTool{workspace: workspace}
}

func (t *FileTool) Name() string {
	return "File"
}

func (t *FileTool) Execute(input string) (string, error) {
	parts := strings.SplitN(input, ":", 3)
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid input format: expected at least 'operation:path'")
	}

	operation := parts[0]
	path := parts[1]
	var content string
	if len(parts) == 3 {
		content = parts[2]
	}

	fullPath := filepath.Join(t.workspace, path)
	rel, err := filepath.Rel(t.workspace, fullPath)
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
		if len(parts) != 3 {
			return "", fmt.Errorf("write operation requires content")
		}
		err := os.WriteFile(fullPath, []byte(content), 0644)
		if err != nil {
			return "", fmt.Errorf("failed to write to file %s: %w", fullPath, err)
		}
		return "File written successfully", nil

	case "patch":
		if len(parts) != 3 {
			return "", fmt.Errorf("patch operation requires diff content")
		}
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
