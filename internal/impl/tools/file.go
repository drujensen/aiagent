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
	b.WriteString("This tool supports various file operations. Below are the key operations and their parameters:\n\n")
	b.WriteString("- **read**: Reads content from a file. Use `start_line` and `end_line` to specify a range of lines to read.\n")
	b.WriteString("- **write**: Writes or overwrites content to a file. Provide `content` to specify the new file content.\n")
	b.WriteString("- **edit**: Modifies specific lines in a file. Use `start_line`, `end_line`, and `content` to replace the specified lines. The `search` operation can help identify line numbers.\n")
	b.WriteString("  - Example: To edit lines 5 to 7, set `start_line=5`, `end_line=7`, and provide the new `content` for those lines.\n")
	b.WriteString("  - Use `dry_run=true` to preview changes without applying them.\n")
	b.WriteString("- **search**: Searches files for a pattern. Returns file paths and line numbers, useful for identifying edit locations.\n")
	b.WriteString("- **create_directory**, **list_directory**, **directory_tree**, **move**, **get_info**: Perform directory and file management tasks.\n\n")

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
			Enum:        []string{"read", "write", "edit", "create_directory", "list_directory", "directory_tree", "move", "search", "get_info"},
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
			Name:        "content",
			Type:        "string",
			Description: "Content to write or edit (for write or edit operations)",
			Required:    false,
		},
		{
			Name:        "dry_run",
			Type:        "boolean",
			Description: "Preview changes without applying them (for edit operation)",
			Required:    false,
		},
		{
			Name:        "destination",
			Type:        "string",
			Description: "Destination path (for move operation)",
			Required:    false,
		},
		{
			Name:        "start_line",
			Type:        "integer",
			Description: "The start line for reading or editing",
			Required:    false,
		},
		{
			Name:        "end_line",
			Type:        "integer",
			Description: "The end line for reading or editing",
			Required:    false,
		},
		{
			Name:        "pattern",
			Type:        "string",
			Description: "Search pattern (for search operation)",
			Required:    false,
		},
		{
			Name:        "exclude_patterns",
			Type:        "array",
			Items:       []entities.Item{{Type: "string"}},
			Description: "Patterns to exclude (for search operation)",
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
		return "", nil
	}
	return fullPath, nil
}

func (t *FileTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing file operation", zap.String("arguments", arguments))
	fmt.Println("\rExecuting file operation", arguments)

	var args struct {
		Operation       string   `json:"operation"`
		Path            string   `json:"path"`
		Content         string   `json:"content"`
		DryRun          bool     `json:"dry_run"`
		Destination     string   `json:"destination"`
		StartLine       int      `json:"start_line"`
		EndLine         int      `json:"end_line"`
		Pattern         string   `json:"pattern"`
		ExcludePatterns []string `json:"exclude_patterns"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
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
	case "read":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		file, err := os.Open(fullPath)
		if err != nil {
			return "", err
		}
		defer file.Close()
		startLine := args.StartLine
		if startLine == 0 {
			startLine = 1
		}
		endLine := args.EndLine
		if endLine == 0 {
			endLine = startLine + 1000
		}

		var lines []string
		scanner := bufio.NewScanner(file)
		lineNum := 0

		for scanner.Scan() {
			lineNum++
			if startLine > 0 && lineNum <= startLine-1 {
				continue
			}
			if endLine > 0 && lineNum >= endLine {
				break
			}
			lines = append(lines, scanner.Text())
		}

		if len(lines) == 0 || (startLine > 0 && len(lines) < startLine) {
			return "", fmt.Errorf("no lines found in file")
		}
		results := strings.Join(lines, "\n")
		if len(results) > 16384 {
			results = results[:16384] + "...truncated"
		}

		return results, nil

	case "write":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		err = os.WriteFile(fullPath, []byte(args.Content), 0644)
		if err != nil {
			t.logger.Error("Failed to write file", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		t.logger.Info("File written successfully", zap.String("path", fullPath))
		return "File written successfully", nil

	case "edit":
		if args.StartLine <= 0 || args.EndLine < args.StartLine {
			t.logger.Error("Invalid line range for edit", zap.Int("start_line", args.StartLine), zap.Int("end_line", args.EndLine))
			return "", fmt.Errorf("start_line and end_line must be positive and end_line >= start_line")
		}
		if args.Content == "" {
			t.logger.Error("Content is required for edit operation")
			return "", fmt.Errorf("content is required for edit operation")
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		results, err := t.applyLineEdit(fullPath, args.StartLine, args.EndLine, args.Content, args.DryRun)
		return results, err

	case "create_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		err = os.MkdirAll(fullPath, 0755)
		if err != nil {
			t.logger.Error("Failed to create directory", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		t.logger.Info("Directory created successfully", zap.String("path", fullPath))
		return "Directory created successfully", nil

	case "list_directory":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			t.logger.Error("Failed to list directory", zap.String("path", fullPath), zap.Error(err))
			return "", err
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
			return "", err
		}
		tree, err := t.buildDirectoryTree(fullPath)
		if err != nil {
			t.logger.Error("Failed to build directory tree", zap.String("path", fullPath), zap.Error(err))
			return "", err
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
			return "", err
		}
		dstPath, err := t.validatePath(args.Destination)
		if err != nil {
			return "", err
		}
		err = os.Rename(srcPath, dstPath)
		if err != nil {
			t.logger.Error("Failed to move file", zap.String("source", srcPath), zap.String("dest", dstPath), zap.Error(err))
			return "", err
		}
		t.logger.Info("File moved successfully", zap.String("source", srcPath), zap.String("dest", dstPath))
		return "File moved successfully", nil

	case "search":
		if args.Pattern == "" {
			t.logger.Error("Pattern is required for search operation")
			return "", fmt.Errorf("pattern is required")
		}
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		results, err := t.searchFiles(fullPath, args.Pattern, args.ExcludePatterns)
		if err != nil {
			t.logger.Error("Failed to search files", zap.String("path", fullPath), zap.Error(err))
			return "", err
		}
		if len(results) == 0 {
			return "No matches found", nil
		}
		resultsStr := strings.Join(results, "\n")
		if len(resultsStr) > 16384 {
			resultsStr = resultsStr[:16384] + "...truncated"
		}
		t.logger.Info("Files searched successfully", zap.String("path", fullPath), zap.Int("matches", len(results)))
		return resultsStr, nil

	case "get_info":
		fullPath, err := t.validatePath(args.Path)
		if err != nil {
			return "", err
		}
		info, err := os.Stat(fullPath)
		if err != nil {
			t.logger.Error("Failed to get file info", zap.String("path", fullPath), zap.Error(err))
			return "", err
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

func (t *FileTool) applyLineEdit(filePath string, startLine, endLine int, content string, dryRun bool) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		t.logger.Error("Failed to read file for edit", zap.String("path", filePath), zap.Error(err))
		return "", err
	}
	defer file.Close()

	var originalLines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		originalLines = append(originalLines, scanner.Text())
	}
	if err := scanner.Err(); err != nil {
		t.logger.Error("Error reading file lines", zap.Error(err))
		return "", err
	}

	if startLine > len(originalLines) {
		t.logger.Error("Start line exceeds file length", zap.Int("start_line", startLine), zap.Int("file_lines", len(originalLines)))
		return "", fmt.Errorf("start_line %d exceeds file length %d", startLine, len(originalLines))
	}

	newContentLines := strings.Split(strings.TrimSuffix(content, "\n"), "\n")
	var modifiedLines []string
	modifiedLines = append(modifiedLines, originalLines[:startLine-1]...)
	modifiedLines = append(modifiedLines, newContentLines...)
	if endLine < len(originalLines) {
		modifiedLines = append(modifiedLines, originalLines[endLine:]...)
	}

	original := strings.Join(originalLines, "\n")
	modified := strings.Join(modifiedLines, "\n")

	if modified == original {
		t.logger.Warn("No changes made to the file", zap.String("path", filePath))
		return "No changes made to the file", nil
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
		return "", err
	}

	if !dryRun {
		err = os.WriteFile(filePath, []byte(modified+"\n"), 0644)
		if err != nil {
			t.logger.Error("Failed to write edited file", zap.String("path", filePath), zap.Error(err))
			return "", err
		}
		t.logger.Info("File edited successfully", zap.String("path", filePath))
	}

	return "```diff\n" + diffStr + "\n```", nil
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

func (t *FileTool) searchFiles(rootPath, pattern string, excludePatterns []string) ([]string, error) {
	var results []string
	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, _ := filepath.Rel(rootPath, path)
		for _, exclude := range excludePatterns {
			if matched, _ := filepath.Match(exclude, relPath); matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}
		if strings.Contains(strings.ToLower(info.Name()), strings.ToLower(pattern)) {
			results = append(results, path)
		}
		return nil
	})
	if err != nil {
		t.logger.Error("Failed to search files", zap.String("path", rootPath), zap.Error(err))
		return nil, err
	}
	if len(results) == 0 {
		t.logger.Warn("No matches found during file search", zap.String("pattern", pattern), zap.String("path", rootPath))
		return nil, fmt.Errorf("no matches found for pattern '%s' in path '%s'", pattern, rootPath)
	}
	t.logger.Info("Files searched successfully", zap.String("path", rootPath), zap.Int("matches", len(results)))
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
