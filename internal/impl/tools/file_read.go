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
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("## Usage Instructions\n")
	b.WriteString("This tool reads content from a text file. Use `offset` (0-based) and `limit` to specify a range of lines to read. Limited to 2000 lines or 10MB per file.\n")
	b.WriteString("The tool returns a JSON array with line numbers and text.\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *FileReadTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "filePath",
			Type:        "string",
			Description: "The absolute path to the file to read",
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

	var b strings.Builder
	for _, line := range lines {
		b.WriteString(fmt.Sprintf("%6d: %s\n", line.Line, line.Text))
	}

	results := b.String()
	if hasMore {
		results += "\nWarning: File has more lines beyond the limit. Use offset and limit to read additional sections."
	}
	if len(results) > 16384 {
		results = results[:16384] + "...truncated"
	}

	t.logger.Info("File read successfully", zap.String("path", fullPath), zap.Int("lines", len(lines)), zap.Bool("hasMore", hasMore))
	return results, nil
}

var _ entities.Tool = (*FileReadTool)(nil)
