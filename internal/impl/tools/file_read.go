package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

type FileReadTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewFileReadTool(name, description string, configuration map[string]string, logger *zap.Logger) *FileReadTool {
	return &FileReadTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *FileReadTool) Name() string {
	return t.name
}

func (t *FileReadTool) Description() string {
	return t.description
}

func (t *FileReadTool) Configuration() map[string]string {
	return t.configuration
}

func (t *FileReadTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *FileReadTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- filePath: The absolute path to the file or directory to read\n- offset: The line number to start reading from (1-indexed)\n- limit: The maximum number of lines to read (defaults to 2000)", t.Description())
}

func (t *FileReadTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"filePath": map[string]any{
				"type":        "string",
				"description": "The absolute path to the file or directory to read",
			},
			"offset": map[string]any{
				"type":        "number",
				"description": "The line number to start reading from (1-indexed)",
			},
			"limit": map[string]any{
				"type":        "number",
				"description": "The maximum number of lines to read (defaults to 2000)",
			},
		},
		"required":             []string{"filePath"},
		"additionalProperties": false,
	}
}

func (t *FileReadTool) validatePath(path string) (string, error) {
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

func (t *FileReadTool) checkFileSize(path string) (bool, error) {
	const maxFileSize = 10 * 1024 * 1024 // 10MB
	info, err := os.Stat(path)
	if err != nil {
		return false, err
	}
	if info.Size() > maxFileSize {
		t.logger.Error("File size exceeds limit", zap.String("path", path), zap.Int64("size", info.Size()), zap.Int64("limit", maxFileSize))
		return false, fmt.Errorf("file size exceeds limit")
	}
	return true, nil
}

func (t *FileReadTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing file read command", zap.String("arguments", arguments))
	var rawArgs map[string]interface{}
	if err := json.Unmarshal([]byte(arguments), &rawArgs); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return `{"content": "", "error": "failed to parse arguments"}`, nil
	}

	filePath, _ := rawArgs["filePath"].(string)
	offsetVal, _ := rawArgs["offset"].(float64)
	limitVal, _ := rawArgs["limit"].(float64)
	offset := int(offsetVal)
	limit := int(limitVal)

	if filePath == "" {
		t.logger.Error("filePath is required")
		return `{"content": "", "error": "filePath is required"}`, nil
	}

	// Default values
	if offset <= 0 {
		offset = 1 // 1-indexed
	}
	if limit <= 0 {
		limit = 2000
	}

	fullPath, err := t.validatePath(filePath)
	if err != nil {
		return fmt.Sprintf(`{"content": "", "error": "%s"}`, err.Error()), nil
	}
	if ok, err := t.checkFileSize(fullPath); !ok {
		return fmt.Sprintf(`{"content": "", "error": "%s"}`, err.Error()), nil
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return fmt.Sprintf(`{"content": "", "error": "failed to open file: %s"}`, err.Error()), nil
	}
	defer file.Close()

	const maxLines = 2000
	if limit > maxLines {
		limit = maxLines
	}

	var lines []string
	scanner := bufio.NewScanner(file)
	lineNum := 0
	readCount := 0

	for scanner.Scan() {
		lineNum++
		if lineNum < offset {
			continue
		}
		if readCount >= limit {
			break
		}
		lines = append(lines, scanner.Text())
		readCount++
	}

	if err := scanner.Err(); err != nil {
		return fmt.Sprintf(`{"content": "", "error": "error reading file: %s"}`, err.Error()), nil
	}

	content := strings.Join(lines, "\n")

	return fmt.Sprintf(`{"content": %q, "lines": %d, "error": ""}`, content, len(lines)), nil
}

var _ entities.Tool = (*FileReadTool)(nil)
