package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"html"
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

	var fullPath string
	if filepath.IsAbs(path) {
		if !strings.HasPrefix(path, workspace) {
			t.logger.Error("Absolute path is outside workspace", zap.String("path", path))
			return "", fmt.Errorf("absolute path is outside workspace")
		}
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
		limit = 1000
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

func (t *FileReadTool) DisplayName(ui string, arguments string) (string, string) {
	// For FileRead, filename is shown in result, so return empty suffix
	return t.Name(), ""
}

func (t *FileReadTool) FormatResult(ui string, result string, diff string, arguments string) string {
	if ui == "tui" {
		return t.formatResultTUI(result, arguments)
	} else if ui == "webui" {
		return t.formatResultWebUI(result)
	}
	return result // Fallback
}

func (t *FileReadTool) formatResultTUI(result string, arguments string) string {
	var response struct {
		Content string `json:"content"`
		Error   string `json:"error"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		return result
	}

	if response.Error != "" {
		return fmt.Sprintf("Error reading file: %s", response.Error)
	}

	// Extract filename from arguments
	var args struct {
		FilePath string `json:"filePath"`
	}
	var filename string
	if err := json.Unmarshal([]byte(arguments), &args); err == nil && args.FilePath != "" {
		filename = args.FilePath
	}

	// Generate TUI-friendly summary with filename and line numbers
	lines := strings.Split(response.Content, "\n")
	var summary strings.Builder

	if filename != "" {
		summary.WriteString(fmt.Sprintf("FileName: %s\n", filename))
	}
	summary.WriteString(fmt.Sprintf("📄 File content (%d lines)\n\n", len(lines)))

	previewCount := 20
	if len(lines) < previewCount {
		previewCount = len(lines)
	}

	for i := 0; i < previewCount; i++ {
		summary.WriteString(fmt.Sprintf("%4d: %s\n", i+1, lines[i]))
	}

	if len(lines) > 20 {
		summary.WriteString(fmt.Sprintf("\n... and %d more lines\n", len(lines)-20))
	}

	return summary.String()
}

func (t *FileReadTool) formatResultWebUI(result string) string {
	var response struct {
		Path    string `json:"path"`
		Content string `json:"content"`
		Lines   int    `json:"lines"`
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		// Fallback to generic
		return t.formatGenericWebUI(result)
	}

	// For Web UI, show simple summary without content preview
	if response.Lines > 0 {
		fileName := "file"
		if response.Path != "" {
			fileName = filepath.Base(response.Path)
		}
		return fmt.Sprintf("<div class=\"tool-summary\">📄 %s (%d lines read)</div>", html.EscapeString(fileName), response.Lines)
	}

	if response.Content != "" {
		lines := strings.Split(response.Content, "\n")
		fileName := "file"
		if response.Path != "" {
			fileName = filepath.Base(response.Path)
		}
		return fmt.Sprintf("<div class=\"tool-summary\">📄 %s (%d lines read)</div>", html.EscapeString(fileName), len(lines))
	}

	return "<div class=\"tool-summary\">File read successfully</div>"
}

func (t *FileReadTool) formatGenericWebUI(result string) string {
	var jsonData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &jsonData); err != nil {
		// If not JSON, escape and return
		return fmt.Sprintf("<div class=\"tool-result\"><pre>%s</pre></div>", html.EscapeString(result))
	}

	var output strings.Builder
	output.WriteString("<div class=\"tool-result-json\">")
	for key, value := range jsonData {
		output.WriteString(fmt.Sprintf("<div><strong>%s:</strong> %v</div>", html.EscapeString(key), value))
	}
	output.WriteString("</div>")

	return output.String()
}

var _ entities.Tool = (*FileReadTool)(nil) // Confirms interface implementation
