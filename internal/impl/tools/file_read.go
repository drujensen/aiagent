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
	return fmt.Sprintf("%s\n\nParameters:\n- filePath: relative path to file from workspace root\n- offset: starting line number (0-based, optional)\n- limit: max lines to read (optional, default 2000)", t.Description())
}

func (t *FileReadTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "filePath",
			Type:        "string",
			Description: "The relative path to the file to read from workspace root",
			Required:    true,
		},
		{
			Name:        "offset",
			Type:        "integer",
			Description: "The line number to start reading from (0-based)",
			Required:    false,
		},
		{
			Name:        "limit",
			Type:        "integer",
			Description: "The number of lines to read (defaults to 2000)",
			Required:    false,
		},
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
	var args struct {
		FilePath string `json:"filePath"`
		Offset   int    `json:"offset"`
		Limit    int    `json:"limit"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.FilePath == "" {
		t.logger.Error("FilePath is required")
		return "", fmt.Errorf("filePath is required")
	}

	if args.Limit == 0 {
		args.Limit = 2000
	}

	fullPath, err := t.validatePath(args.FilePath)
	if err != nil {
		return "", err
	}
	if ok, err := t.checkFileSize(fullPath); !ok {
		return "", err
	}
	file, err := os.Open(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	const maxLines = 2000
	if args.Limit > maxLines {
		args.Limit = maxLines
	}

	var lines []LineResult
	scanner := bufio.NewScanner(file)
	lineNum := 0
	readCount := 0
	hasMore := false

	for scanner.Scan() {
		lineNum++
		if lineNum-1 < args.Offset {
			continue
		}
		if readCount >= args.Limit {
			hasMore = true
			break
		}
		lines = append(lines, LineResult{
			Line: lineNum,
			Text: scanner.Text(),
		})
		readCount++
	}

	if err := scanner.Err(); err != nil {
		t.logger.Error("Error reading file", zap.String("path", fullPath), zap.Error(err))
		return "", fmt.Errorf("error reading file: %v", err)
	}

	if len(lines) == 0 {
		return "No lines found in file", fmt.Errorf("no lines found in file")
	}

	// Create TUI-friendly summary (first 5 lines only)
	var result strings.Builder
	result.WriteString(fmt.Sprintf("📄 %s (%d lines total)\n\n", filepath.Base(fullPath), lineNum))

	previewLines := 5
	if len(lines) < previewLines {
		previewLines = len(lines)
	}

	for i := 0; i < previewLines; i++ {
		result.WriteString(fmt.Sprintf("%6d: %s\n", lines[i].Line, lines[i].Text))
	}

	if len(lines) > 5 {
		result.WriteString(fmt.Sprintf("... and %d more lines\n", len(lines)-5))
	}

	if hasMore {
		result.WriteString("\n⚠️  File has more content beyond the limit. Use offset and limit to read additional sections.")
	}

	// For AI processing, include full content as structured data
	// The TUI will only show the summary above
	fullContent := make([]string, len(lines))
	for i, line := range lines {
		fullContent[i] = fmt.Sprintf("%6d: %s", line.Line, line.Text)
	}

	// Create JSON response with summary for TUI and full data for AI
	response := struct {
		Summary     string   `json:"summary"`
		FullContent []string `json:"full_content"`
		FilePath    string   `json:"file_path"`
		TotalLines  int      `json:"total_lines"`
		HasMore     bool     `json:"has_more"`
	}{
		Summary:     result.String(),
		FullContent: fullContent,
		FilePath:    fullPath,
		TotalLines:  lineNum,
		HasMore:     hasMore,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal file read response", zap.Error(err))
		return result.String(), nil // Fallback to summary only
	}

	t.logger.Info("File read successfully", zap.String("path", fullPath), zap.Int("lines", len(lines)), zap.Bool("hasMore", hasMore))
	return string(jsonResult), nil
}

var _ entities.Tool = (*FileReadTool)(nil)
