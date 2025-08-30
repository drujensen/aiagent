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
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("## Usage Instructions\n")
	b.WriteString("This tool searches for text in files or directories. Returns a JSON array with line numbers and matching lines. Use `all_files=true` to search all files in a directory recursively. Limited to 1000 lines or 10MB per file.\n")
	b.WriteString("  - Example: Search single file: `path='file.txt', pattern='text'`\n")
	b.WriteString("  - Example: Search all files: `path='.', pattern='text', all_files=true`\n")
	b.WriteString("  - Example: Search Go files: `path='.', pattern='func', all_files=true, file_pattern='*.go'`\n")
	b.WriteString("  - Example: Case-sensitive: `path='file.txt', pattern='Text', case_sensitive=true`\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *FileSearchTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "path",
			Type:        "string",
			Description: "The file or directory path",
			Required:    true,
		},
		{
			Name:        "pattern",
			Type:        "string",
			Description: "Search pattern",
			Required:    true, // Making pattern required for simplicity, as it's essential for search
		},
		{
			Name:        "all_files",
			Type:        "boolean",
			Description: "Search all files in directory recursively",
			Required:    false,
		},
		{
			Name:        "file_pattern",
			Type:        "string",
			Description: "Glob pattern to filter files (e.g., '*.go') when all_files=true",
			Required:    false,
		},
		{
			Name:        "case_sensitive",
			Type:        "boolean",
			Description: "Perform case-sensitive search (default: false)",
			Required:    false,
		},
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
	t.logger.Debug("Executing file search command", zap.String("arguments", arguments))
	var args struct {
		Path          string `json:"path"`
		Pattern       string `json:"pattern"`
		AllFiles      bool   `json:"all_files"`
		FilePattern   string `json:"file_pattern"`
		CaseSensitive bool   `json:"case_sensitive"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Path == "" {
		t.logger.Error("Path is required")
		return "", fmt.Errorf("path is required")
	}
	if args.Pattern == "" {
		t.logger.Error("Pattern is required")
		return "", fmt.Errorf("pattern is required")
	}

	fullPath, err := t.validatePath(args.Path)
	if err != nil {
		return "", err
	}
	var jsonResponse []byte
	if args.AllFiles {
		results, err := t.searchMultipleFiles(fullPath, args.Pattern, args.FilePattern, args.CaseSensitive)
		if err != nil {
			t.logger.Error("Failed to search multiple files", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		response := struct {
			FileResponse struct {
				Results map[string][]LineResult `json:"results"`
			} `json:"File_response"`
		}{}
		response.FileResponse.Results = results
		jsonResponse, err = json.Marshal(response)
		if err != nil {
			t.logger.Error("Failed to marshal multi-file search results", zap.Error(err))
			return "", fmt.Errorf("failed to marshal multi-file search results: %v", err)
		}
	} else {
		results, err := t.search(fullPath, args.Pattern, args.CaseSensitive)
		if err != nil {
			t.logger.Error("Failed to search file", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		response := struct {
			FileResponse struct {
				Results []LineResult `json:"results"`
			} `json:"File_response"`
		}{}
		response.FileResponse.Results = results
		jsonResponse, err = json.Marshal(response)
		if err != nil {
			t.logger.Error("Failed to marshal search results", zap.Error(err))
			return "", fmt.Errorf("failed to marshal search results: %v", err)
		}
	}
	resultsStr := string(jsonResponse)
	if len(resultsStr) > 16384 {
		resultsStr = resultsStr[:16384] + "...truncated"
	}
	t.logger.Info("File search completed", zap.String("path", fullPath), zap.Bool("all_files", args.AllFiles))
	return resultsStr, nil
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
		t.logger.Warn("No matches found during file search", zap.String("pattern", pattern), zap.String("path", filePath))
		return nil, fmt.Errorf("no matches found for pattern '%s' in file '%s'", pattern, filePath)
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
		t.logger.Warn("No matches found in directory", zap.String("pattern", pattern), zap.String("path", dirPath))
		return nil, fmt.Errorf("no matches found for pattern '%s' in directory '%s'", pattern, dirPath)
	}
	t.logger.Info("Multiple files searched successfully", zap.String("path", dirPath), zap.Int("files_with_matches", len(results)))
	return results, nil
}

var _ entities.Tool = (*FileSearchTool)(nil)
