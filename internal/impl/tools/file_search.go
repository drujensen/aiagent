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

type FileSearchTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewFileSearchTool(name, description string, configuration map[string]string, logger *zap.Logger) *FileSearchTool {
	return &FileSearchTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *FileSearchTool) Name() string {
	return t.name
}

func (t *FileSearchTool) Description() string {
	return t.description
}

func (t *FileSearchTool) Configuration() map[string]string {
	return t.configuration
}

func (t *FileSearchTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *FileSearchTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- path: file or directory path\n- pattern: regex pattern to search for\n- file_pattern: file pattern filter (optional)\n- case_sensitive: boolean (optional)\n- all_files: search all files in directory (optional)", t.Description())
}

func (t *FileSearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"pattern": map[string]any{
				"type":        "string",
				"description": "Search regex or string",
			},
			"directory": map[string]any{
				"type":        "string",
				"description": "Root dir to search",
			},
			"file_ext": map[string]any{
				"type":        "string",
				"description": "File extension filter (e.g., '.go')",
			},
		},
		"required": []string{"pattern", "directory"},
	}
}

func (t *FileSearchTool) validatePath(path string) (string, error) {
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

func (t *FileSearchTool) checkFileSize(path string) (bool, error) {
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

func (t *FileSearchTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing code search", zap.String("arguments", arguments))
	var args struct {
		Pattern   string `json:"pattern"`
		Directory string `json:"directory"`
		FileExt   string `json:"file_ext,omitempty"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		return `{"results": [], "error": "failed to parse arguments"}`, nil
	}

	if args.Pattern == "" {
		return `{"results": [], "error": "pattern is required"}`, nil
	}
	if args.Directory == "" {
		return `{"results": [], "error": "directory is required"}`, nil
	}

	fullPath, err := t.validatePath(args.Directory)
	if err != nil {
		return fmt.Sprintf(`{"results": [], "error": "invalid directory: %s"}`, err.Error()), nil
	}

	filePattern := "*"
	if args.FileExt != "" {
		filePattern = "*" + args.FileExt
	}

	results, err := t.searchMultipleFiles(fullPath, args.Pattern, filePattern, false)
	if err != nil {
		return fmt.Sprintf(`{"results": [], "error": "search failed: %s"}`, err.Error()), nil
	}

	var grokResults []map[string]any
	for filePath, fileResults := range results {
		for _, result := range fileResults {
			grokResults = append(grokResults, map[string]any{
				"file":    filePath,
				"line":    result.Line,
				"snippet": result.Text,
			})
		}
	}

	jsonResult, err := json.Marshal(map[string]any{
		"results": grokResults,
		"error":   "",
	})
	if err != nil {
		return `{"results": [], "error": "failed to marshal response"}`, nil
	}

	return string(jsonResult), nil
}

func (t *FileSearchTool) search(filePath, pattern string, caseSensitive bool) ([]LineResult, error) {
	if ok, err := t.checkFileSize(filePath); !ok {
		return nil, err
	}
	file, err := os.Open(filePath)
	if err != nil {
		t.logger.Error("Failed to open file for search", zap.String("path", filePath), zap.Error(err))
		return nil, fmt.Errorf("failed to open file: %v", err)
	}
	defer file.Close()

	var results []LineResult
	scanner := bufio.NewScanner(file)
	lineNum := 0
	const maxLines = 1000

	for scanner.Scan() {
		lineNum++
		if lineNum > maxLines {
			t.logger.Warn("Line count exceeds limit", zap.String("path", filePath), zap.Int("limit", maxLines))
			return nil, fmt.Errorf("file exceeds line limit of %d lines", maxLines)
		}
		line := scanner.Text()
		if caseSensitive {
			if strings.Contains(line, pattern) {
				results = append(results, LineResult{
					Line: lineNum,
					Text: line,
				})
			}
		} else {
			if strings.Contains(strings.ToLower(line), strings.ToLower(pattern)) {
				results = append(results, LineResult{
					Line: lineNum,
					Text: line,
				})
			}
		}
	}
	if err := scanner.Err(); err != nil {
		t.logger.Error("Error reading file during search", zap.String("path", filePath), zap.Error(err))
		return nil, fmt.Errorf("error reading file: %v", err)
	}

	if len(results) == 0 {
		t.logger.Info("No matches found during file search", zap.String("pattern", pattern), zap.String("path", filePath))
		return []LineResult{}, nil
	}
	t.logger.Info("File searched successfully", zap.String("path", filePath), zap.Int("matches", len(results)))
	return results, nil
}

func (t *FileSearchTool) searchMultipleFiles(dirPath, pattern, filePattern string, caseSensitive bool) (map[string][]LineResult, error) {
	results := make(map[string][]LineResult)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.logger.Warn("Error accessing path", zap.String("path", path), zap.Error(err))
			return nil // Continue walking despite errors
		}
		if info.IsDir() {
			// Skip .git and .aiagent directories
			if info.Name() == ".git" || info.Name() == ".aiagent" {
				return filepath.SkipDir
			}
			return nil
		}
		if filePattern != "" {
			matched, err := filepath.Match(filePattern, info.Name())
			if err != nil || !matched {
				return nil
			}
		}
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			t.logger.Warn("Failed to get relative path", zap.String("path", path), zap.Error(err))
			return nil
		}
		fileResults, err := t.search(path, pattern, caseSensitive)
		if err == nil && len(fileResults) > 0 {
			results[relPath] = fileResults
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("error walking directory: %v", err)
	}
	if len(results) == 0 {
		t.logger.Info("No matches found in directory", zap.String("pattern", pattern), zap.String("path", dirPath))
		return make(map[string][]LineResult), nil
	}
	t.logger.Info("Multiple files searched successfully", zap.String("path", dirPath), zap.Int("files_with_matches", len(results)))
	return results, nil
}

var _ entities.Tool = (*FileSearchTool)(nil)
