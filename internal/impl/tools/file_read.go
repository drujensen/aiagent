package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
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

func (t *FileReadTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path": map[string]any{
				"type":        "string",
				"description": "Full or relative path to the file",
			},
			"lines": map[string]any{
				"type":        "string",
				"description": "Optional range of lines to read (e.g., \"1-10\" for lines 1 through 10)",
			},
		},
		"required": []string{"path"},
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
		Path  string `json:"path"`
		Lines string `json:"lines,omitempty"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return `{"content": "", "error": "failed to parse arguments"}`, nil
	}

	if args.Path == "" {
		t.logger.Error("Path is required")
		return `{"content": "", "error": "path is required"}`, nil
	}

	// Parse lines range
	offset := 0
	limit := 2000
	if args.Lines != "" {
		if strings.Contains(args.Lines, "-") {
			parts := strings.Split(args.Lines, "-")
			if len(parts) == 2 {
				start, err1 := strconv.Atoi(strings.TrimSpace(parts[0]))
				end, err2 := strconv.Atoi(strings.TrimSpace(parts[1]))
				if err1 == nil && err2 == nil && start > 0 && end >= start {
					offset = start - 1 // 0-based
					limit = end - start + 1
				}
			}
		} else {
			// Single line number
			if line, err := strconv.Atoi(args.Lines); err == nil && line > 0 {
				offset = line - 1
				limit = 1
			}
		}
	}

	fullPath, err := t.validatePath(args.Path)
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
		if lineNum-1 < offset {
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

	// Generate hashline format for all lines
	var hashlineContent strings.Builder
	for i, line := range lines {
		// Create a simple hash based on line content (this is the hashline format)
		hash := t.generateSimpleHash(line)
		hashlineContent.WriteString(fmt.Sprintf("%d:%s|%s\n", i+1, hash, line))
	}

	// Create TUI-friendly summary
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📄 File content (%d lines)\n\n", len(lines)))

	// Show first 20 lines with hashline format
	previewCount := 20
	if len(lines) < previewCount {
		previewCount = len(lines)
	}

	for i := 0; i < previewCount; i++ {
		hash := t.generateSimpleHash(lines[i])
		summary.WriteString(fmt.Sprintf("%4d:%s| %s\n", i+1, hash, lines[i]))
	}

	if len(lines) > 20 {
		summary.WriteString(fmt.Sprintf("\n... and %d more lines\n", len(lines)-20))
	}

	return fmt.Sprintf(`{"content": %q, "summary": %q, "lines": %d, "error": ""}`, hashlineContent.String(), summary.String(), len(lines)), nil
}

// generateSimpleHash creates a simple hash based on line content for hashline functionality
func generateSimpleHash(line string) string {
	// Use a proper hash of the full line content for better uniqueness
	if len(line) == 0 {
		return "000"
	}

	// Calculate a hash based on the full line content using a simple but effective algorithm
	// This approach creates different hashes for different content, which is the key requirement
	var hash uint32 = 5381
	for _, char := range line {
		hash = ((hash << 5) + hash) + uint32(char)
	}

	// Convert to 3-character string representation
	// Using modulo to get a 3-digit hash that's more unique than just first few chars
	hashValue := hash % 1000
	result := fmt.Sprintf("%03d", hashValue)

	return result
}

// generateSimpleHash creates a simple hash based on line content for hashline functionality
func (t *FileReadTool) generateSimpleHash(line string) string {
	return generateSimpleHash(line)
}

var _ entities.Tool = (*FileReadTool)(nil)
