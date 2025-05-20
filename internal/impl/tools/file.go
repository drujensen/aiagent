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
	"github.com/pmezard/go-difflib/difflib"
	"go.uber.org/zap"
)

type FileTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

type LineResult struct {
	Line int    `json:"line"`
	Text string `json:"text"`
}

type RefreshResult struct {
	LineCount int          `json:"line_count"`
	Preview   []LineResult `json:"preview,omitempty"`
}

func NewFileTool(name, description string, configuration map[string]string, logger *zap.Logger) *FileTool {
	return &FileTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *FileTool) Name() string {
	return t.name
}

func (t *FileTool) Description() string {
	return t.description
}

func (t *FileTool) Configuration() map[string]string {
	return t.configuration
}

func (t *FileTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *FileTool) FullDescription() string {
	var b strings.Builder

	// Add description
	b.WriteString(t.Description())
	b.WriteString("\n\n")

	// Add detailed usage instructions
	b.WriteString("## Usage Instructions\n")
	b.WriteString("This tool supports file operations for any text file. **Critical**: Follow these steps to avoid errors when inserting or editing:\n")
	b.WriteString("1. Use `search` or `read` to confirm the exact line number and surrounding content before making changes.\n")
	b.WriteString("2. Use `dry_run=true` with `edit`, `insert`, or `delete` to preview changes and verify the line is correct.\n")
	b.WriteString("3. After any change, use `read` to check the updated file and get new line numbers, as insertions or deletions shift lines.\n\n")
	b.WriteString("- **search**: Searches for text within a file or directory using a pattern. Returns a JSON array with line numbers and matching lines. Use `all_files=true` to search all files in a directory recursively. Limited to 1000 lines or 10MB per file.\n")
	b.WriteString("  - Example: Search single file: `operation='search', path='file.txt', pattern='text'`\n")
	b.WriteString("  - Example: Search all files: `operation='search', path='.', pattern='text', all_files=true`\n")
	b.WriteString("- **read**: Reads content from a file. Returns a JSON array with line numbers and text. Use `start_line` and `end_line` to specify a range of lines to read. Limited to 1000 lines or 10MB per file.\n")
	b.WriteString("- **write**: Overwrites or creates a file with new content. Provide `content` to specify the full file content. For line-specific changes, use `edit`, `insert`, or `delete`.\n")
	b.WriteString("- **edit**: Replaces specific lines in a file with new content. Use `start_line`, `end_line`, and `content` to replace the specified lines. For full file replacement, use `write` instead.\n")
	b.WriteString("  - Example: To replace lines 5 to 7, set `operation='edit', start_line=5, end_line=7`, and provide the new `content` for those lines.\n")
	b.WriteString("- **insert**: Inserts new content at a specific line. Use `start_line` and `content` to insert the content before the specified line.\n")
	b.WriteString("  - Example: To insert content at line 5, set `operation='insert', start_line=5`, and provide the `content` to insert. Then use `read` to get updated line numbers.\n")
	b.WriteString("- **delete**: Deletes specific lines in a file. Use `start_line` and `end_line` to specify the lines to delete.\n")
	b.WriteString("  - Example: To delete lines 5 to 7, set `operation='delete', start_line=5, end_line=7`. Then use `read` to get updated line numbers.\n")
	b.WriteString("  - Use `dry_run=true` to preview changes without applying them for `edit`, `insert`, or `delete` operations.\n")
	b.WriteString("This tool also supports various directory operations:\n\n")
	b.WriteString("- **directory_tree**, **create_directory**, **list_directory**, **move**: Perform directory and file management tasks.\n\n")
	b.WriteString("- **get_info**: Retrieves file information such as size, creation date, modification date, and permissions.\n")

	// Add configuration header
	b.WriteString("## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")

	// Loop through configuration and add key-value pairs to the table
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}

	return b.String()
}

func (t *FileTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"read", "write", "edit", "insert", "delete", "create_directory", "list_directory", "directory_tree", "move", "search", "get_info"},
			Description: "The file operation to perform",
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
			Name:        "content",
			Type:        "string",
			Description: "Content to write, edit, or insert",
			Required:    false,
		},
		{
			Name:        "start_line",
			Type:        "integer",
			Description: "The start line for reading, editing, inserting, or deleting",
			Required:    false,
		},
		{
			Name:        "end_line",
			Type:        "integer",
			Description: "The end line for reading, editing, or deleting",
			Required:    false,
		},
		{
			Name:        "dry_run",
			Type:        "boolean",
			Description: "Preview changes without applying them (for edit, insert, or delete operations)",
			Required:    false,
		},
		{
			Name:        "destination",
			Type:        "string",
			Description: "Destination path (for move operation)",
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

func (t *FileTool) validatePath(path string) (string, error) {
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

func (t *FileTool) checkFileSize(path string) (bool, error) {
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

func (t *FileTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing file command", zap.String("arguments", arguments))
	fmt.Println("\rExecuting file command", arguments)
	var args struct {
		Operation   string `json:"operation"`
		Path        string `json:"path"`
		Pattern     string `json:"pattern"`
		Content     string `json:"content"`
		DryRun      bool   `json:"dry_run"`
		StartLine   int    `json:"start_line"`
		EndLine     int    `json:"end_line"`
		Destination string `json:"destination"`
		AllFiles    bool   `json:"all_files"`
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

	case "write":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		err = os.WriteFile(fullPath, []byte(args.Content), 0644)
		if err != nil {
			t.logger.Error("Failed to write file", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to write file: %v", err)
		}
		t.logger.Info("File written successfully", zap.String("path", fullPath))
		return "File written successfully", nil

	case "edit", "insert", "delete":
		if args.StartLine == 0 {
			args.StartLine = 1
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		if args.Operation == "edit" && args.EndLine == 0 {
			t.logger.Error("End line is required for edit operation")
			return "", fmt.Errorf("end_line is required for edit operation")
		}
		if args.Operation == "edit" && args.Content == "" {
			t.logger.Error("Content is required for edit operation")
			return "", fmt.Errorf("content is required for edit operation")
		}
		if args.Operation == "insert" && args.Content == "" {
			t.logger.Error("Content is required for insert operation")
			return "", fmt.Errorf("content is required for insert operation")
		}
		if args.Operation == "delete" && args.EndLine == 0 {
			args.EndLine = args.StartLine
		}
		results, err := t.applyLineEdit(fullPath, args.Operation, args.StartLine, args.EndLine, args.Content, args.DryRun)
		return results, err

	case "create_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			t.logger.Error("Failed to create directory", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to create directory: %v", err)
		}
		t.logger.Info("Directory created successfully", zap.String("path", fullPath))
		return "Directory created successfully", nil

	case "list_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			t.logger.Error("Failed to list directory", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to list directory: %v", err)
		}
		var formatted []string
		for _, entry := range entries {
			prefix := "[FILE]"
			if entry.IsDir() {
				prefix = "[DIR]"
			}
			formatted = append(formatted, prefix+" "+entry.Name())
		}
		results := strings.Join(formatted, "\n")
		if len(results) > 16384 {
			results = results[:16384] + "...truncated"
		}
		t.logger.Info("Directory listed successfully", zap.String("path", fullPath))
		return results, nil

	case "directory_tree":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid path: %v", err)
		}
		tree, err := t.buildDirectoryTree(fullPath)
		if err != nil {
			t.logger.Error("Failed to build directory tree", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to build directory tree: %v", err)
		}
		jsonTree, _ := json.MarshalIndent(tree, "", "  ")
		results := string(jsonTree)
		if len(results) > 16384 {
			results = results[:16384] + "...truncated"
		}
		t.logger.Info("Directory tree built successfully", zap.String("path", fullPath))
		return results, nil

	case "move":
		if args.Destination == "" {
			t.logger.Error("Destination is required for move operation")
			return "", fmt.Errorf("destination is required")
		}
		srcPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", fmt.Errorf("invalid source path: %v", err)
		}
		dstPath, err := t.validatePath(args.Destination)
		if err != nil {
			return "", fmt.Errorf("invalid destination path: %v", err)
		}
		err = os.Rename(srcPath, dstPath)
		if err != nil {
			t.logger.Error("Failed to move file", zap.String("source", srcPath), zap.String("dest", dstPath), zap.Error(err))
			return "", fmt.Errorf("failed to move file: %v", err)
		}
		t.logger.Info("File moved successfully", zap.String("source", srcPath), zap.String("dest", dstPath))
		return "File moved successfully", nil

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

func (t *FileTool) applyLineEdit(filePath, operation string, startLine, endLine int, content string, dryRun bool) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		t.logger.Error("Failed to read file for edit", zap.String("path", filePath), zap.Error(err))
		return "", fmt.Errorf("failed to read file: %v", err)
	}
	defer file.Close()

	var originalLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		originalLines = append(originalLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.logger.Error("Error reading file lines", zap.Error(err))
		return "", fmt.Errorf("error reading file lines: %v", err)
	}

	if startLine > len(originalLines)+1 || startLine < 1 {
		t.logger.Error("Invalid start line", zap.Int("start_line", startLine), zap.Int("file_lines", len(originalLines)))
		return "", fmt.Errorf("start_line %d is invalid, must be between 1 and %d. Use 'read' to get current line numbers", startLine, len(originalLines)+1)
	}
	if (operation == "edit" || operation == "delete") && endLine > len(originalLines) {
		t.logger.Error("End line exceeds file length", zap.Int("end_line", endLine), zap.Int("file_lines", len(originalLines)))
		return "", fmt.Errorf("end_line %d exceeds file length %d. Use 'read' to get current line numbers", endLine, len(originalLines))
	}
	if (operation == "edit" || operation == "delete") && endLine < startLine {
		t.logger.Error("End line is less than start line", zap.Int("start_line", startLine), zap.Int("end_line", endLine))
		return "", fmt.Errorf("end_line %d is less than start_line %d", endLine, startLine)
	}

	var modifiedLines []string
	switch operation {
	case "edit":
		newContentLines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		modifiedLines = append(modifiedLines, originalLines[:startLine-1]...)
		modifiedLines = append(modifiedLines, newContentLines...)
		if endLine < len(originalLines) {
			modifiedLines = append(modifiedLines, originalLines[endLine:]...)
		}
	case "insert":
		newContentLines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
		modifiedLines = append(modifiedLines, originalLines[:startLine-1]...)
		modifiedLines = append(modifiedLines, newContentLines...)
		modifiedLines = append(modifiedLines, originalLines[startLine-1:]...)
	case "delete":
		modifiedLines = append(modifiedLines, originalLines[:startLine-1]...)
		if endLine < len(originalLines) {
			modifiedLines = append(modifiedLines, originalLines[endLine:]...)
		}
	}

	original := strings.Join(originalLines, "\n")
	modified := strings.Join(modifiedLines, "\n")

	if modified == original {
		t.logger.Warn("No changes made to the file", zap.String("path", filePath))
		return fmt.Sprintf("No changes made to the file\nLine count: %d", len(originalLines)), nil
	}

	diff := difflib.UnifiedDiff{
		A:        difflib.SplitLines(original),
		B:        difflib.SplitLines(modified),
		FromFile: filePath,
		ToFile:   filePath,
		Context:  3,
	}
	diffStr, err := difflib.GetUnifiedDiffString(diff)
	if err != nil {
		t.logger.Error("Failed to generate diff", zap.Error(err))
		return "", fmt.Errorf("failed to generate diff: %v", err)
	}

	if !dryRun {
		err = os.WriteFile(filePath, []byte(modified+"\n"), 0644)
		if err != nil {
			t.logger.Error("Failed to write edited file", zap.String("path", filePath), zap.Error(err))
			return "", fmt.Errorf("failed to write to file: %v", err)
		}
		t.logger.Info("File edited successfully", zap.String("path", filePath), zap.String("operation", operation))
	}

	result := struct {
		Diff      string `json:"diff"`
		LineCount int    `json:"line_count"`
	}{
		Diff:      "```diff\n" + diffStr + "\n```",
		LineCount: len(modifiedLines),
	}
	jsonResponse, err := json.Marshal(result)
	if err != nil {
		t.logger.Error("Failed to marshal edit results", zap.Error(err))
		return "", fmt.Errorf("failed to marshal edit results: %v", err)
	}

	return string(jsonResponse), nil
}

type TreeEntry struct {
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	Children []TreeEntry `json:"children,omitempty"`
}

func (t *FileTool) buildDirectoryTree(path string) ([]TreeEntry, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	var result []TreeEntry
	for _, entry := range entries {
		treeEntry := TreeEntry{
			Name: entry.Name(),
			Type: "file",
		}
		if entry.IsDir() {
			treeEntry.Type = "directory"
			children, err := t.buildDirectoryTree(filepath.Join(path, entry.Name()))
			if err != nil {
				continue
			}
			treeEntry.Children = children
		}
		result = append(result, treeEntry)
	}
	return result, nil
}

func (t *FileTool) search(filePath, pattern string) ([]LineResult, error) {
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

func (t *FileTool) searchMultipleFiles(dirPath, pattern string) (map[string][]LineResult, error) {
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

func formatSize(size int64) string {
	return humanize.Bytes(uint64(size))
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}

var _ entities.Tool = (*FileTool)(nil)
