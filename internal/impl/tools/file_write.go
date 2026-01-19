package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"

	"github.com/drujensen/aiagent/internal/domain/entities"

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

func (t *FileWriteTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{Name: "operation", Type: "string", Enum: []string{"write", "edit"}, Description: "Operation to perform", Required: true},
		{Name: "path", Type: "string", Description: "File path", Required: true},
		{Name: "content", Type: "string", Description: "File content for write operation", Required: false},
		{Name: "old_string", Type: "string", Description: "String to replace for edit operation", Required: false},
		{Name: "replace_all", Type: "boolean", Description: "Replace all occurrences", Required: false},
	}
}

func (t *FileWriteTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- operation: 'write' or 'edit'\n- path: file path (absolute or relative to workspace)\n- content: file content (for write)\n- old_string, new_string: exact strings to replace (for edit)\n- replace_all: boolean (optional)", t.Description())
}

func (t *FileWriteTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Path to write to",
			},
			"content": map[string]any{
				"type":        "string",
				"description": "Text to write",
			},
			"mode": map[string]any{
				"type":        "string",
				"description": "\"overwrite\" or \"append\"",
				"enum":        []string{"overwrite", "append"},
				"default":     "overwrite",
			},
		},
		"required": []string{"path", "content"},
	}
}

func (t *FileWriteTool) validatePath(path string) (string, error) {
	// Ensure path is valid UTF-8
	if !utf8.ValidString(path) {
		t.logger.Error("Path contains invalid UTF-8", zap.String("path", path))
		return "", fmt.Errorf("path contains invalid UTF-8")
	}

	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("could not get current directory: %v", err)
		}
	}

	var fullPath string
	if filepath.IsAbs(path) {
		fullPath = path
	} else {
		fullPath = filepath.Join(workspace, path)
	}

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
		Path    string `json:"path"`
		Content string `json:"content"`
		Mode    string `json:"mode,omitempty"`
	}

	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"success": false, "error": "failed to parse arguments"}`, nil
	}

	if args.Path == "" {
		return `{"success": false, "error": "path is required"}`, nil
	}
	if args.Content == "" {
		return `{"success": false, "error": "content is required"}`, nil
	}

	if args.Mode == "" {
		args.Mode = "overwrite"
	}
	if args.Mode != "overwrite" && args.Mode != "append" {
		return `{"success": false, "error": "mode must be 'overwrite' or 'append'"}`, nil
	}

	fullPath, err := t.validatePath(args.Path)
	if err != nil {
		return fmt.Sprintf(`{"success": false, "error": "invalid path: %s"}`, err.Error()), nil
	}

	// Create backup if file exists and we're overwriting
	if args.Mode == "overwrite" {
		if _, err := os.Stat(fullPath); err == nil {
			backupPath := fullPath + ".backup"
			if err := t.createBackup(fullPath, backupPath); err != nil {
				t.logger.Warn("Failed to create backup", zap.Error(err))
			}
		}
	}

	var file *os.File
	if args.Mode == "append" {
		file, err = os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
	} else {
		file, err = os.OpenFile(fullPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	}
	if err != nil {
		return fmt.Sprintf(`{"success": false, "error": "failed to open file: %s"}`, err.Error()), nil
	}
	defer file.Close()

	if _, err := file.WriteString(args.Content); err != nil {
		return fmt.Sprintf(`{"success": false, "error": "failed to write file: %s"}`, err.Error()), nil
	}

	return `{"success": true, "error": ""}`, nil
}

func (t *FileWriteTool) createBackup(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

var _ entities.Tool = (*FileWriteTool)(nil)
