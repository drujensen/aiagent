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
	b.WriteString("This tool searches for text in files or directories. Returns a JSON array with line numbers and matching lines. Limited to 1000 lines or 10MB per file.\n")
	b.WriteString("\n## Simplified Usage\n")
	b.WriteString("- **Directory search**: When searching directories (like '.'), `all_files` is automatically enabled\n")
	b.WriteString("- **File search**: For single files, just specify the file path\n")
	b.WriteString("\n## Examples\n")
	b.WriteString("  - Search current directory: `path='.', pattern='function'` (all_files automatically enabled)\n")
	b.WriteString("  - Search single file: `path='file.txt', pattern='text'`\n")
	b.WriteString("  - Search Go files only: `path='.', pattern='func', file_pattern='*.go'`\n")
	b.WriteString("  - Case-sensitive search: `path='file.txt', pattern='Text', case_sensitive=true`\n")
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
			Description: "Search all files in directory recursively (automatically enabled for directories)",
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

	// Check if path is a directory and automatically set all_files=true if not specified
	info, err := os.Stat(fullPath)
	if err != nil {
		return "", fmt.Errorf("path does not exist: %v", err)
	}
	if info.IsDir() && !args.AllFiles {
		// Automatically enable all_files for directories to simplify usage
		args.AllFiles = true
		t.logger.Debug("Automatically enabled all_files for directory search", zap.String("path", fullPath))
	}

	var summary strings.Builder
	var fullResults interface{}
	var totalMatches int

	if args.AllFiles {
		results, err := t.searchMultipleFiles(fullPath, args.Pattern, args.FilePattern, args.CaseSensitive)
		if err != nil {
			t.logger.Error("Failed to search multiple files", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}

		// Count total matches
		for _, fileResults := range results {
			totalMatches += len(fileResults)
		}

		// Create TUI-friendly summary
		summary.WriteString(fmt.Sprintf("🔍 File Search: %s\n", args.Pattern))
		summary.WriteString(fmt.Sprintf("📂 Directory: %s\n", filepath.Base(fullPath)))
		if args.FilePattern != "" {
			summary.WriteString(fmt.Sprintf("📄 File Pattern: %s\n", args.FilePattern))
		}
		summary.WriteString(fmt.Sprintf("📊 Results: %d files with matches, %d total matches\n\n", len(results), totalMatches))

		// Show first 3 files with matches
		fileCount := 0
		for filePath, fileResults := range results {
			if fileCount >= 3 {
				break
			}
			summary.WriteString(fmt.Sprintf("📄 %s (%d matches):\n", filePath, len(fileResults)))

			// Show first 2 matches per file
			matchCount := 0
			for _, result := range fileResults {
				if matchCount >= 2 {
					break
				}
				summary.WriteString(fmt.Sprintf("   %6d: %s\n", result.Line, result.Text))
				matchCount++
			}

			if len(fileResults) > 2 {
				summary.WriteString(fmt.Sprintf("   ... and %d more matches\n", len(fileResults)-2))
			}
			summary.WriteString("\n")
			fileCount++
		}

		if len(results) > 3 {
			summary.WriteString(fmt.Sprintf("... and %d more files with matches\n", len(results)-3))
		}

		fullResults = results
	} else {
		results, err := t.search(fullPath, args.Pattern, args.CaseSensitive)
		if err != nil {
			t.logger.Error("Failed to search file", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}

		totalMatches = len(results)

		// Create TUI-friendly summary
		summary.WriteString(fmt.Sprintf("🔍 File Search: %s\n", args.Pattern))
		summary.WriteString(fmt.Sprintf("📄 File: %s\n", filepath.Base(fullPath)))
		summary.WriteString(fmt.Sprintf("📊 Results: %d matches\n\n", totalMatches))

		// Show first 5 matches
		previewCount := 5
		if len(results) < previewCount {
			previewCount = len(results)
		}

		for i := 0; i < previewCount; i++ {
			result := results[i]
			summary.WriteString(fmt.Sprintf("%6d: %s\n", result.Line, result.Text))
		}

		if len(results) > 5 {
			summary.WriteString(fmt.Sprintf("\n... and %d more matches\n", len(results)-5))
		}

		fullResults = results
	}

	// Create JSON response with summary for TUI and full data for AI
	response := struct {
		Summary       string      `json:"summary"`
		Pattern       string      `json:"pattern"`
		Path          string      `json:"path"`
		AllFiles      bool        `json:"all_files"`
		FilePattern   string      `json:"file_pattern,omitempty"`
		CaseSensitive bool        `json:"case_sensitive"`
		FullResults   interface{} `json:"full_results"`
		TotalMatches  int         `json:"total_matches"`
	}{
		Summary:       summary.String(),
		Pattern:       args.Pattern,
		Path:          fullPath,
		AllFiles:      args.AllFiles,
		FilePattern:   args.FilePattern,
		CaseSensitive: args.CaseSensitive,
		FullResults:   fullResults,
		TotalMatches:  totalMatches,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal file search response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	t.logger.Info("File search completed", zap.String("path", fullPath), zap.Bool("all_files", args.AllFiles), zap.Int("matches", totalMatches))
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
