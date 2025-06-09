package tools

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"aiagent/internal/domain/entities"

	"github.com/dustin/go-humanize"
	"go.uber.org/zap"
)

type FileReadTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

type LineResult struct {
	Line int    `json:"line"`
	Text string `json:"text"`
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
	b.WriteString("This tool supports reading and searching text files. **Critical**: Use these operations to inspect file content and metadata.\n")
	b.WriteString("- **search**: Searches for text within a file or directory using a pattern. Returns a JSON array with line numbers and matching lines. Use `all_files=true` to search all files in a directory recursively. Limited to 1000 lines or 10MB per file.\n")
	b.WriteString("  - Example: Search single file: `operation='search', path='file.txt', pattern='text'`\n")
	b.WriteString("  - Example: Search all files: `operation='search', path='.', pattern='text', all_files=true`\n")
	b.WriteString("- **read**: Reads content from a file. Returns a JSON array with line numbers and text. Use `start_line` and `end_line` to specify a range of lines to read. Limited to 1000 lines or 10MB per file.\n")
	b.WriteString("- **get_info**: Retrieves file information such as size, creation date, modification date, and permissions.\n")
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
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"read", "search", "get_info"},
			Description: "The file read operation to perform",
			Required:    true,
		},
		{
			Name:        "path",
			Type:        "string",
			Description: "The file or directory path",
			Required:    true,
		},
		{
			Name:        "pattern",
			Type:        "string",
			Description: "Search pattern (for search operation)",
			Required:    false,
		},
		{
			Name:        "start_line",
			Type:        "integer",
			Description: "The start line for reading",
			Required:    false,
		},
		{
			Name:        "end_line",
			Type:        "integer",
			Description: "The end line for reading",
			Required:    false,
		},
		{
			Name:        "all_files",
			Type:        "boolean",
			Description: "Search all files in directory recursively (for search operation)",
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
		return false, fmt.Errorf("file size %s exceeds limit of %s", humanize.Bytes(uint64(info.Size())), humanize.Bytes(maxFileSize))
	}
	return true, nil
}

func (t *FileReadTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing file read command", zap.String("arguments", arguments))
	fmt.Println("\rExecuting file read command", arguments)
	var args struct {
		Operation string `json:"operation"`
		Path      string `json:"path"`
		Pattern   string `json:"pattern"`
		StartLine int    `json:"start_line"`
		EndLine   int    `json:"end_line"`
		AllFiles  bool   `json:"all_files"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	if args.Operation == "" {
		t.logger.Error("Operation is required")
		return "", fmt.Errorf("operation is required")
	}
	if args.Path == "" {
		t.logger.Error("Path is required")
		return "", fmt.Errorf("path is required")
	}

	switch args.Operation {
	case "search":
		if args.Pattern == "" {
			t.logger.Error("Pattern is required for search operation")
			return "", fmt.Errorf("pattern is required")
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		var jsonResponse []byte
		if args.AllFiles {
			results, err := t.searchMultipleFiles(fullPath, args.Pattern)
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
			results, err := t.search(fullPath, args.Pattern)
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

	case "read":
		fullPath, err := t.validatePath(args.Path)
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
		startLine := args.StartLine
		endLine := args.EndLine
		const maxLines = 1000

		var lines []LineResult
		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			if lineNum > maxLines {
				t.logger.Warn("Line count exceeds limit", zap.String("path", fullPath), zap.Int("limit", maxLines))
				return "", fmt.Errorf("file exceeds line limit of %d lines", maxLines)
			}
			if startLine > 0 && lineNum < startLine {
				continue
			}
			if endLine > 0 && lineNum > endLine {
				break
			}
			lines = append(lines, LineResult{
				Line: lineNum,
				Text: scanner.Text(),
			})
		}

		if err := scanner.Err(); err != nil {
			t.logger.Error("Error reading file", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("error reading file: %v", err)
		}

		if len(lines) == 0 {
			return "No lines found in file", fmt.Errorf("no lines found in file")
		}

		jsonResponse, err := json.Marshal(lines)
		if err != nil {
			t.logger.Error("Failed to marshal read results", zap.Error(err))
			return "", fmt.Errorf("failed to marshal read results: %v", err)
		}

		results := string(jsonResponse)
		if len(results) > 16384 {
			results = results[:16384] + "...truncated"
		}

		t.logger.Info("File read successfully", zap.String("path", fullPath), zap.Int("lines", len(lines)))
		return results, nil

	case "get_info":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			t.logger.Error("Failed to get file info", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to get file info: %v", err)
		}
		result := []string{
			"size: " + formatSize(info.Size()),
			"created: " + info.ModTime().Format(time.RFC3339),
			"modified: " + info.ModTime().Format(time.RFC3339),
			"accessed: " + info.ModTime().Format(time.RFC3339),
			"isDirectory: " + boolToString(info.IsDir()),
			"isFile: " + boolToString(!info.IsDir()),
			"permissions: " + info.Mode().String(),
		}
		t.logger.Info("File info retrieved successfully", zap.String("path", fullPath))
		return strings.Join(result, "\n"), nil

	default:
		t.logger.Error("Unknown operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("unknown operation: %s", args.Operation)
	}
}

func (t *FileReadTool) search(filePath, pattern string) ([]LineResult, error) {
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
	lowerPattern := strings.ToLower(pattern)
	const maxLines = 1000

	for scanner.Scan() {
		lineNum++
		if lineNum > maxLines {
			t.logger.Warn("Line count exceeds limit", zap.String("path", filePath), zap.Int("limit", maxLines))
			return nil, fmt.Errorf("file exceeds line limit of %d lines", maxLines)
		}
		line := scanner.Text()
		if strings.Contains(strings.ToLower(line), lowerPattern) {
			results = append(results, LineResult{
				Line: lineNum,
				Text: line,
			})
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

func (t *FileReadTool) searchMultipleFiles(dirPath, pattern string) (map[string][]LineResult, error) {
	results := make(map[string][]LineResult)
	err := filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			t.logger.Warn("Error accessing path", zap.String("path", path), zap.Error(err))
			return nil // Continue walking despite errors
		}
		if info.IsDir() {
			return nil
		}
		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			t.logger.Warn("Failed to get relative path", zap.String("path", path), zap.Error(err))
			return nil
		}
		fileResults, err := t.search(path, pattern)
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

var _ entities.Tool = (*FileReadTool)(nil)
