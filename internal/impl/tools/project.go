package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

type ProjectTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

func NewProjectTool(name, description string, configuration map[string]string, logger *zap.Logger) *ProjectTool {
	return &ProjectTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *ProjectTool) Name() string {
	return t.name
}

func (t *ProjectTool) Description() string {
	return t.description
}

func (t *ProjectTool) Configuration() map[string]string {
	return t.configuration
}

func (t *ProjectTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *ProjectTool) FullDescription() string {
	return fmt.Sprintf("%s\n\nParameters:\n- operation: 'read' or 'get_source'\n- language: programming language (optional, auto-detected)\n- filters: custom file patterns (optional)\n- max_file_size: file size limit in bytes (optional)", t.Description())
}

func (t *ProjectTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"operation": map[string]any{
				"type":        "string",
				"description": "The operation to perform ('read' for project file, 'get_source' for file map and source code)",
				"enum":        []string{"read", "get_source"},
			},
			"language": map[string]any{
				"type":        "string",
				"description": "Programming language to determine default file patterns and parsing (e.g., 'go', 'csharp')",
				"enum":        []string{"all", "shell", "assembly", "c", "cpp", "rust", "go", "csharp", "objective-c", "swift", "java", "kotlin", "clojure", "groovy", "lua", "elixir", "scala", "dart", "haskell", "javascript", "typescript", "python", "ruby", "php", "perl", "r", "html", "stylesheet"},
			},
			"filters": map[string]any{
				"type":        "array",
				"description": "List of file patterns to include (e.g., '**/*.go', 'go.mod'). Overrides language defaults.",
				"items": map[string]any{
					"type": "string",
				},
			},
			"max_file_size": map[string]any{
				"type":        "integer",
				"description": "Maximum file size in bytes; if exceeded, content/structure is replaced with a message (default: 0, no limit)",
			},
			"max_total_size": map[string]any{
				"type":        "integer",
				"description": "Maximum total output size in characters; if exceeded, remaining files are skipped (default: 200000)",
			},
		},
		"required": []string{"operation"},
	}
}

func (t *ProjectTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing project tool", zap.String("arguments", arguments))

	var args struct {
		Operation    string   `json:"operation"`
		Language     string   `json:"language,omitempty"`
		Filters      []string `json:"filters,omitempty"`
		MaxFileSize  int      `json:"max_file_size,omitempty"`
		MaxTotalSize int      `json:"max_total_size,omitempty"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
	}

	// Set default max_total_size if not provided
	if args.MaxTotalSize == 0 {
		args.MaxTotalSize = 200000
	}

	// Get workspace
	workspace := t.configuration["workspace"]
	if workspace == "" {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			t.logger.Error("Could not get current directory", zap.Error(err))
			return "", fmt.Errorf("could not get current directory: %v", err)
		}
	}

	switch args.Operation {
	case "read":
		return t.executeRead(workspace)
	case "get_source":
		return t.executeGetSource(workspace, args.Language, args.Filters, args.MaxFileSize, args.MaxTotalSize)
	default:
		t.logger.Error("Invalid operation", zap.String("operation", args.Operation))
		return "", fmt.Errorf("invalid operation: %s", args.Operation)
	}
}

func (t *ProjectTool) executeRead(workspace string) (string, error) {
	// Get the project file path from configuration
	projectFile, ok := t.configuration["project_file"]
	if !ok || projectFile == "" {
		t.logger.Error("Project file not configured")
		return "", fmt.Errorf("project_file configuration is required")
	}

	fullPath := filepath.Join(workspace, projectFile)
	rel, err := filepath.Rel(workspace, fullPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.logger.Error("Project file path is outside workspace", zap.String("path", projectFile))
		return "", fmt.Errorf("project file path is outside workspace")
	}

	// Check if the file exists, create it with default content if it doesn't
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.logger.Info("Project file does not exist, creating with default content", zap.String("path", fullPath))
		defaultContent := `# Project Details

This is a default project description file created automatically by the ProjectTool.
Please update this file with relevant project information.

## Overview
- Project Name: [Your Project Name]
- Description: [Brief description of the project]
- Repository: [URL to the project repository, if applicable]

## Instructions
- [Add specific instructions for AI agents like GitHub Copilot or Claude]

## Additional Notes
- [Add any additional context or notes]
`
		if err := os.WriteFile(fullPath, []byte(defaultContent), 0644); err != nil {
			t.logger.Error("Failed to create project file", zap.String("path", fullPath), zap.Error(err))
			return "", fmt.Errorf("failed to create project file: %v", err)
		}
		t.logger.Info("Project file created successfully", zap.String("path", fullPath))
	} else if err != nil {
		t.logger.Error("Failed to check project file", zap.String("path", fullPath), zap.Error(err))
		return "", fmt.Errorf("failed to check project file: %v", err)
	}

	// Read the project file
	content, err := os.ReadFile(fullPath)
	if err != nil {
		t.logger.Error("Failed to read project file", zap.String("path", fullPath), zap.Error(err))
		return "", fmt.Errorf("failed to read project file: %v", err)
	}

	contentStr := string(content)

	// Create TUI-friendly summary (first 5 lines only)
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📄 Project File: %s\n\n", filepath.Base(fullPath)))

	lines := strings.Split(contentStr, "\n")
	previewLines := 5
	if len(lines) < previewLines {
		previewLines = len(lines)
	}

	for i := 0; i < previewLines; i++ {
		if lines[i] != "" {
			summary.WriteString(fmt.Sprintf("  %s\n", lines[i]))
		}
	}

	if len(lines) > 5 {
		summary.WriteString(fmt.Sprintf("  ... and %d more lines\n", len(lines)-5))
	}

	// Create JSON response with summary for TUI and full content for AI
	response := struct {
		Summary string `json:"summary"`
		Path    string `json:"path"`
		Content string `json:"content"`
	}{
		Summary: summary.String(),
		Path:    fullPath,
		Content: contentStr,
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal project read response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	t.logger.Info("Project file read successfully", zap.String("path", fullPath))
	return string(jsonResult), nil
}

func (t *ProjectTool) detectLanguage(workspace string) string {
	// Check for language-specific files to auto-detect the project type
	languageFiles := map[string][]string{
		"go":         {"go.mod"},
		"python":     {"requirements.txt", "setup.py", "pyproject.toml"},
		"javascript": {"package.json"},
		"typescript": {"package.json", "tsconfig.json"},
		"rust":       {"Cargo.toml"},
		"ruby":       {"Gemfile"},
		"php":        {"composer.json"},
		"csharp":     {"*.sln", "*.csproj"},
		"java":       {"pom.xml", "build.gradle"},
		"kotlin":     {"build.gradle.kts"},
		"swift":      {"Package.swift"},
		"c":          {"Makefile"},
		"cpp":        {"CMakeLists.txt", "Makefile"},
	}

	for lang, files := range languageFiles {
		for _, file := range files {
			// Check for exact file matches
			if file[0] != '*' {
				fullPath := filepath.Join(workspace, file)
				if _, err := os.Stat(fullPath); err == nil {
					return lang
				}
			} else {
				// Check for pattern matches
				matches, err := filepath.Glob(filepath.Join(workspace, file))
				if err == nil && len(matches) > 0 {
					return lang
				}
			}
		}
	}

	return "" // No language detected
}

// matchPattern matches a file path against a glob pattern, supporting ** for recursive matching
func matchPattern(pattern, path string) (bool, error) {
	// For patterns without **, use standard filepath.Match
	if !strings.Contains(pattern, "**") {
		return filepath.Match(pattern, path)
	}

	// For ** patterns, use a simple implementation
	return matchGlobPattern(pattern, path)
}

// matchGlobPattern implements basic ** support
func matchGlobPattern(pattern, path string) (bool, error) {
	// Split pattern and path by "/"
	patternParts := strings.Split(pattern, "/")
	pathParts := strings.Split(path, "/")

	return matchGlobParts(patternParts, pathParts, 0, 0)
}

// matchGlobParts recursively matches pattern parts against path parts
func matchGlobParts(patternParts, pathParts []string, patternIdx, pathIdx int) (bool, error) {
	// If we've consumed all pattern parts, we need to have consumed all path parts too
	if patternIdx >= len(patternParts) {
		return pathIdx >= len(pathParts), nil
	}

	// If we've consumed all path parts but still have pattern parts, no match
	if pathIdx >= len(pathParts) {
		// Special case: if the remaining pattern is just **, it matches empty
		if patternIdx < len(patternParts) && patternParts[patternIdx] == "**" {
			return matchGlobParts(patternParts, pathParts, patternIdx+1, pathIdx)
		}
		return false, nil
	}

	patternPart := patternParts[patternIdx]
	pathPart := pathParts[pathIdx]

	switch {
	case patternPart == "**":
		// ** can match zero or more directories
		// Try matching zero directories first
		if matched, _ := matchGlobParts(patternParts, pathParts, patternIdx+1, pathIdx); matched {
			return true, nil
		}
		// Try matching one or more directories
		return matchGlobParts(patternParts, pathParts, patternIdx, pathIdx+1)

	case strings.Contains(patternPart, "*"):
		// Pattern contains wildcards, use filepath.Match
		matched, err := filepath.Match(patternPart, pathPart)
		if err != nil {
			return false, err
		}
		if matched {
			return matchGlobParts(patternParts, pathParts, patternIdx+1, pathIdx+1)
		}
		return false, nil

	default:
		// Exact match
		if patternPart == pathPart {
			return matchGlobParts(patternParts, pathParts, patternIdx+1, pathIdx+1)
		}
		return false, nil
	}
}

func (t *ProjectTool) executeGetSource(workspace, language string, customFilters []string, maxFileSize, maxTotalSize int) (string, error) {
	defaultFilters := map[string][]string{
		"all":        {"**/*"},
		"shell":      {"**/*.sh", "**/*.bash", "**.zsh", "**/*.pwsh"},
		"assembly":   {"**/*.asm", "**/*.s"},
		"c":          {"**/*.c", "**/*.h", "Makefile"},
		"cpp":        {"**/*.cpp", "**/*.hpp", "**/*.h", "CMakeLists.txt"},
		"rust":       {"**/*.rs", "Cargo.toml"},
		"go":         {"**/*.go", "go.mod"},
		"csharp":     {"**/*.cs", "**/*.csproj", "*.sln"},
		"swift":      {"**/*.swift", "Package.swift"},
		"java":       {"**/*.java", "**/*.xml"},
		"kotlin":     {"**/*.kt", "**/*.kts", "build.gradle.kts"},
		"clojure":    {"**/*.clj", "**/*.cljs", "project.clj", "deps.edn"},
		"lua":        {"**/*.lua"},
		"elixir":     {"**/*.ex", "**/*.exs", "mix.exs"},
		"scala":      {"**/*.scala", "build.sbt"},
		"dart":       {"**/*.dart", "pubspec.yaml"},
		"haskell":    {"**/*.hs", "stack.yaml", "cabal.project"},
		"javascript": {"**/*.js", "**/*.ts", "package.json"},
		"typescript": {"**/*.ts", "**/*.tsx", "package.json"},
		"python":     {"**/*.py", "requirements.txt", "setup.py"},
		"ruby":       {"**/*.rb", "Gemfile"},
		"php":        {"**/*.php", "composer.json"},
		"perl":       {"**/*.pl", "**/*.pm", "Makefile.PL"},
		"r":          {"**/*.R", "**/*.r", "DESCRIPTION"},
		"html":       {"**/*.html", "**/*.htm"},
		"stylesheet": {"**/*.css", "**/*.scss", "**/*.less"},
	}

	filters := customFilters
	if len(filters) == 0 {
		if language != "" {
			var ok bool
			filters, ok = defaultFilters[language]
			if !ok {
				t.logger.Error("Unsupported language", zap.String("language", language))
				return "", fmt.Errorf("unsupported language: %s", language)
			}
		} else {
			// Auto-detect language or use source-focused default
			detectedLanguage := t.detectLanguage(workspace)
			if detectedLanguage != "" {
				t.logger.Info("Auto-detected language", zap.String("language", detectedLanguage))
				filters = defaultFilters[detectedLanguage]
			} else {
				// Use source-focused default instead of "all"
				filters = []string{
					"**/*.go", "**/*.py", "**/*.js", "**/*.ts", "**/*.java", "**/*.cpp", "**/*.c", "**/*.h",
					"**/*.rs", "**/*.rb", "**/*.php", "**/*.cs", "**/*.swift", "**/*.kt", "**/*.scala",
					"go.mod", "go.sum", "package.json", "Cargo.toml", "Gemfile", "composer.json",
					"requirements.txt", "setup.py", "Makefile", "CMakeLists.txt", "*.sln", "**/*.csproj",
				}
			}
		}
	}

	// Build directory tree (ignores .git)
	tree, err := buildDirectoryTree(workspace)
	if err != nil {
		return "", err
	}

	// Collect matching files (ignores .git and common non-source directories)
	var files []struct {
		Path    string
		Content string
	}
	err = filepath.Walk(workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip .git, .aiagent, and common non-source directories
			if info.Name() == ".git" || info.Name() == ".aiagent" ||
				info.Name() == "node_modules" || info.Name() == "bin" ||
				info.Name() == "dist" || info.Name() == "build" ||
				info.Name() == "target" || info.Name() == ".next" ||
				info.Name() == ".nuxt" || info.Name() == ".vuepress" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return err
		}

		for _, pattern := range filters {
			matched, err := matchPattern(pattern, relPath)
			if err != nil {
				return err
			}
			if matched {
				var content string
				if maxFileSize > 0 && info.Size() > int64(maxFileSize) {
					content = fmt.Sprintf("File too large (%d bytes, max: %d)", info.Size(), maxFileSize)
				} else {
					byteContent, err := os.ReadFile(path)
					if err != nil {
						return err
					}
					content = string(byteContent)
				}
				files = append(files, struct {
					Path    string
					Content string
				}{relPath, content})
				break
			}
		}
		return nil
	})
	if err != nil {
		t.logger.Error("Failed to walk directory", zap.Error(err))
		return "", fmt.Errorf("failed to walk directory: %v", err)
	}

	// Sort files by path
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})

	// Limit total size
	totalSize := len(tree)
	fileContents := make(map[string]string)
	for _, file := range files {
		contentSize := len(file.Content)
		if totalSize+contentSize > maxTotalSize {
			t.logger.Info("Stopping file inclusion due to size limit", zap.Int("current_size", totalSize), zap.Int("max_size", maxTotalSize))
			break
		}
		fileContents[file.Path] = file.Content
		totalSize += contentSize
	}

	// Create TUI-friendly summary
	var summary strings.Builder
	summary.WriteString(fmt.Sprintf("📁 Reading %d files", len(files)))

	// Create JSON response with summary for TUI and full data for AI
	response := struct {
		Summary      string            `json:"summary"`
		FileMap      string            `json:"file_map"`
		FileContents map[string]string `json:"file_contents"`
		TotalFiles   int               `json:"total_files"`
	}{
		Summary:      summary.String(),
		FileMap:      tree,
		FileContents: fileContents,
		TotalFiles:   len(files),
	}

	jsonResult, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal project response", zap.Error(err))
		return summary.String(), nil // Fallback to summary only
	}

	t.logger.Info("Source code retrieved successfully", zap.String("workspace", workspace), zap.Int("files", len(files)))
	return string(jsonResult), nil
}

// buildDirectoryTree generates a tree representation of the directory (ignores .git)
func buildDirectoryTree(root string) (string, error) {
	var buf bytes.Buffer
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			buf.WriteString(".\n")
			return nil
		}

		// Skip .git and .aiagent directories
		if info.IsDir() && (info.Name() == ".git" || info.Name() == ".aiagent") {
			return filepath.SkipDir
		}

		depth := strings.Count(relPath, string(os.PathSeparator))
		prefix := strings.Repeat("    ", depth)
		if info.IsDir() {
			buf.WriteString(fmt.Sprintf("%s└── %s/\n", prefix, info.Name()))
			// To make it simple, we list dirs but don't use full tree lines
		} else {
			buf.WriteString(fmt.Sprintf("%s├── %s\n", prefix, info.Name()))
		}
		return nil
	})
	if err != nil {
		return "", err
	}
	return buf.String(), nil
}

var _ entities.Tool = (*ProjectTool)(nil)
