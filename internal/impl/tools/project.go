package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	treesitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/typescript/typescript"

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
	var b strings.Builder
	b.WriteString(t.Description())
	b.WriteString("\n\n")
	b.WriteString("## Usage Instructions\n")
	b.WriteString("This tool provides project details and source code/structure in JSON format.\n")
	b.WriteString("- 'read': Reads the project markdown file.\n")
	b.WriteString("- 'get_source': Provides file map (directory tree) and full file contents for relevant files based on language or custom filters.\n")
	b.WriteString("- 'get_structure': Provides file map and parsed source structure (e.g., classes, methods, functions, imports) using Tree-sitter for an index-like view. Use this to reduce context size; fetch details with FileRead tool.\n")
	b.WriteString("  - Use 'language' to set default file patterns (e.g., 'go' for **/*.go and go.mod) and parsing rules.\n")
	b.WriteString("  - Override with 'filters' array for custom patterns.\n")
	b.WriteString("  - Use 'max_file_size' to limit individual file parsing (in bytes; if exceeded, structure is replaced with a message).\n")
	b.WriteString("\n## Supported Languages and Default Filters\n")
	b.WriteString("- go: **/*.go, go.mod\n")
	b.WriteString("- csharp: **/*.cs, **/*.csproj, *.sln\n")
	b.WriteString("- python: **/*.py\n")
	b.WriteString("- javascript: **/*.js, **/*.ts, package.json\n")
	b.WriteString("- typescript: **/*.ts, **/*.tsx, package.json\n")
	b.WriteString("- ruby: **/*.rb, Gemfile\n")
	b.WriteString("\n## Configuration\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *ProjectTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "operation",
			Type:        "string",
			Enum:        []string{"read", "get_source", "get_structure"},
			Description: "The operation to perform ('read' for project file, 'get_source' for file map and source code, 'get_structure' for file map and parsed structure)",
			Required:    true,
		},
		{
			Name:        "language",
			Type:        "string",
			Description: "Programming language to determine default file patterns and parsing (e.g., 'go', 'csharp')",
			Required:    false,
			Enum:        []string{"all", "shell", "assembly", "c", "cpp", "rust", "zig", "go", "csharp", "objective-c", "swift", "java", "kotlin", "clojure", "groovy", "lua", "elixir", "scala", "dart", "haskell", "javascript", "typescript", "python", "ruby", "php", "perl", "r", "html", "stylesheet"},
		},
		{
			Name:        "filters",
			Type:        "array",
			Items:       []entities.Item{{Type: "string"}},
			Description: "List of file patterns to include (e.g., '**/*.go', 'go.mod'). Overrides language defaults.",
			Required:    false,
		},
		{
			Name:        "max_file_size",
			Type:        "integer",
			Description: "Maximum file size in bytes; if exceeded, content/structure is replaced with a message (default: 0, no limit)",
			Required:    false,
		},
	}
}

func (t *ProjectTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing project tool", zap.String("arguments", arguments))

	var args struct {
		Operation   string   `json:"operation"`
		Language    string   `json:"language,omitempty"`
		Filters     []string `json:"filters,omitempty"`
		MaxFileSize int      `json:"max_file_size,omitempty"`
	}
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", fmt.Errorf("failed to parse arguments: %v", err)
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
		return t.executeGetSource(workspace, args.Language, args.Filters, args.MaxFileSize)
	case "get_structure":
		return t.executeGetStructure(workspace, args.Language, args.Filters, args.MaxFileSize)
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

	// Return the file contents as a JSON response
	response := struct {
		Content string `json:"content"`
	}{
		Content: string(content),
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	t.logger.Info("Project file read successfully", zap.String("path", fullPath))
	return string(jsonResponse), nil
}

func (t *ProjectTool) executeGetSource(workspace, language string, customFilters []string, maxFileSize int) (string, error) {
	defaultFilters := map[string][]string{
		"all":        {"**/*"},
		"shell":      {"**/*.sh", "**/*.bash", "**.zsh", "**/*.pwsh"},
		"assembly":   {"**/*.asm", "**/*.s"},
		"c":          {"**/*.c", "**/*.h", "Makefile"},
		"cpp":        {"**/*.cpp", "**/*.hpp", "**/*.h", "CMakeLists.txt"},
		"rust":       {"**/*.rs", "Cargo.toml"},
		"zig":        {"**/*.zig", "build.zig"},
		"go":         {"**/*.go"},
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
			filters = defaultFilters["all"]
		}
	}

	// Build directory tree (ignores .git)
	tree, err := buildDirectoryTree(workspace)
	if err != nil {
		return "", err
	}

	// Collect matching files (ignores .git)
	var files []struct {
		Path    string
		Content string
	}
	err = filepath.Walk(workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			// Skip .git directory
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return err
		}

		for _, pattern := range filters {
			matched, err := filepath.Match(pattern, relPath)
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

	// Build JSON response
	response := struct {
		FileMap      string            `json:"file_map"`
		FileContents map[string]string `json:"file_contents"`
	}{
		FileMap:      tree,
		FileContents: make(map[string]string),
	}
	for _, file := range files {
		response.FileContents[file.Path] = file.Content
	}

	jsonResponse, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	t.logger.Info("Source code retrieved successfully", zap.String("workspace", workspace), zap.Int("files", len(files)))
	return string(jsonResponse), nil
}

func (t *ProjectTool) executeGetStructure(workspace, language string, customFilters []string, maxFileSize int) (string, error) {
	defaultFilters := map[string][]string{
		"all":        {"**/*"},
		"assembly":   {"**/*.asm", "**/*.s"},
		"c":          {"**/*.c", "**/*.h"},
		"cpp":        {"**/*.cpp", "**/*.hpp", "**/*.h"},
		"rust":       {"**/*.rs"},
		"zig":        {"**/*.zig", "build.zig"},
		"go":         {"**/*.go"},
		"csharp":     {"**/*.cs"},
		"swift":      {"**/*.swift", "Package.swift"},
		"java":       {"**/*.java"},
		"kotlin":     {"**/*.kt", "**/*.kts"},
		"clojure":    {"**/*.clj", "**/*.cljs"},
		"lua":        {"**/*.lua"},
		"elixir":     {"**/*.ex"},
		"scala":      {"**/*.scala"},
		"dart":       {"**/*.dart"},
		"haskell":    {"**/*.hs"},
		"javascript": {"**/*.js"},
		"typescript": {"**/*.ts", "**/*.tsx"},
		"python":     {"**/*.py"},
		"ruby":       {"**/*.rb"},
		"php":        {"**/*.php"},
		"perl":       {"**/*.pl"},
		"r":          {"**/*.R"},
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
			filters = defaultFilters["all"]
		}
	}

	// Build directory tree (ignores .git)
	tree, err := buildDirectoryTree(workspace)
	if err != nil {
		return "", err
	}

	// Collect matching files (ignores .git)
	var filePaths []string
	err = filepath.Walk(workspace, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}

		relPath, err := filepath.Rel(workspace, path)
		if err != nil {
			return err
		}

		for _, pattern := range filters {
			matched, err := filepath.Match(pattern, relPath)
			if err != nil {
				return err
			}
			if matched {
				filePaths = append(filePaths, path)
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
	sort.Strings(filePaths)

	// Extract structures
	structures := make(map[string]Structure)
	for _, path := range filePaths {
		relPath, _ := filepath.Rel(workspace, path)

		// Read content
		info, _ := os.Stat(path)
		var content string
		if maxFileSize > 0 && info.Size() > int64(maxFileSize) {
			structures[relPath] = Structure{Error: fmt.Sprintf("File too large (%d bytes, max: %d)", info.Size(), maxFileSize)}
			continue
		}
		byteContent, err := os.ReadFile(path)
		if err != nil {
			structures[relPath] = Structure{Error: err.Error()}
			continue
		}
		content = string(byteContent)

		// Extract structure
		structure, err := extractStructure(language, content)
		if err != nil {
			t.logger.Error("Failed to extract structure", zap.String("path", relPath), zap.String("language", language), zap.Error(err))
			structures[relPath] = Structure{Error: err.Error()}
			continue
		}
		structures[relPath] = structure
	}

	// Build JSON response
	response := struct {
		FileMap         string               `json:"file_map"`
		SourceStructure map[string]Structure `json:"source_structure"`
	}{
		FileMap:         tree,
		SourceStructure: structures,
	}
	jsonResponse, err := json.Marshal(response)
	if err != nil {
		t.logger.Error("Failed to marshal response", zap.Error(err))
		return "", fmt.Errorf("failed to marshal response: %v", err)
	}

	t.logger.Info("Source structure retrieved successfully", zap.String("workspace", workspace), zap.Int("files", len(filePaths)))
	return string(jsonResponse), nil
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

		// Skip .git directories
		if info.IsDir() && info.Name() == ".git" {
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

// Structure represents the parsed structure of a file
type Structure struct {
	Language string       `json:"language"`
	Imports  []string     `json:"imports,omitempty"`
	Entities []CodeEntity `json:"entities,omitempty"`
	Error    string       `json:"error,omitempty"` // If parsing failed
}

// CodeEntity represents a code entity (e.g., struct, function)
type CodeEntity struct {
	Type      string `json:"type"` // e.g., "function", "type", "method", "class"
	Name      string `json:"name"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
	// Add more fields as needed, e.g., Fields []string for struct fields
}

// Map of language names to Tree-sitter language objects
var languageMap = map[string]*treesitter.Language{
	"go":         golang.GetLanguage(),
	"csharp":     csharp.GetLanguage(),
	"javascript": javascript.GetLanguage(),
	"typescript": typescript.GetLanguage(),
	"python":     python.GetLanguage(),
	"ruby":       ruby.GetLanguage(),
}

// Map of language to tags query (copy from tree-sitter-<lang>/queries/tags.scm)
var tagsQueries = map[string]string{
	"go": `
(source_file
  (package_clause
   (package_identifier) @package)
   (function_declaration
     name: (identifier) @definition.function))

(source_file
  (package_clause
   (package_identifier) @package)
   (type_declaration
     (type_spec
       name: (type_identifier) @definition.type)))
	`,
	"csharp": `
(class_declaration
  name: (identifier) @definition.class)
(interface_declaration
  name: (identifier) @definition.interface)
(struct_declaration
  name: (identifier) @definition.struct)
(enum_declaration
  name: (identifier) @definition.enum)
(method_declaration
  name: (identifier) @definition.method)
(property_declaration
  name: (identifier) @definition.property)
(field_declaration
  (variable_declaration
    (variable_declarator
      name: (identifier) @definition.field)))
	`,
	"javascript": `
(function_declaration
  name: (identifier) @definition.function)
(method_definition
  name: (property_identifier) @definition.method)
(class_declaration
  name: (identifier) @definition.class)
(variable_declarator
  name: (identifier) @definition.var)
	`,
	"typescript": `
(function_declaration
  name: (identifier) @definition.function)
(method_definition
  name: (property_identifier) @definition.method)
(class_declaration
  name: (type_identifier) @definition.class)
(interface_declaration
  name: (type_identifier) @definition.interface)
(type_alias_declaration
  name: (type_identifier) @definition.type)
(variable_declarator
  name: (identifier) @definition.var)
	`,
	"python": `
(class_definition
  name: (identifier) @definition.class)
(function_definition
  name: (identifier) @definition.function)
	`,
	"ruby": `
(class
  name: (constant) @definition.class)
(module
  name: (constant) @definition.module)
(method
  name: (identifier) @definition.method)
(singleton_method
  name: (identifier) @definition.method)
	`,
}

// extractStructure parses the content and extracts structure using Tree-sitter
func extractStructure(language, content string) (Structure, error) {
	lang, ok := languageMap[language]
	if !ok {
		return Structure{Language: language}, fmt.Errorf("unsupported language for parsing: %s", language)
	}

	parser := treesitter.NewParser()
	parser.SetLanguage(lang)
	tree, err := parser.ParseCtx(context.Background(), nil, []byte(content))
	if err != nil {
		return Structure{}, err
	}

	// Extract imports (example for Go; customize per language)
	imports, err := extractImports(language, tree, content)
	if err != nil {
		imports = nil
	}

	// Extract entities using tags query
	queryStr, ok := tagsQueries[language]
	if !ok {
		return Structure{Imports: imports, Language: language}, nil // No entities if no query
	}
	q, err := treesitter.NewQuery([]byte(queryStr), lang)
	if err != nil {
		return Structure{}, err
	}
	cursor := treesitter.NewQueryCursor()
	cursor.Exec(q, tree.RootNode())

	var entities []CodeEntity
	for {
		match, found := cursor.NextMatch()
		if !found {
			break
		}
		for _, capture := range match.Captures {
			captureName := q.CaptureNameForId(capture.Index)
			if strings.HasPrefix(captureName, "definition.") {
				kind := strings.TrimPrefix(captureName, "definition.")
				entityName := capture.Node.Content([]byte(content))
				startLine := int(capture.Node.StartPoint().Row) + 1
				endLine := int(capture.Node.EndPoint().Row) + 1
				entities = append(entities, CodeEntity{
					Type:      kind,
					Name:      entityName,
					StartLine: startLine,
					EndLine:   endLine,
				})
				// Optional: For types like "type" (struct), traverse node for fields, etc.
			}
		}
	}

	return Structure{
		Language: language,
		Imports:  imports,
		Entities: entities,
	}, nil
}

// extractImports extracts import statements (customize per language)
func extractImports(language string, tree *treesitter.Tree, content string) ([]string, error) {
	var queryStr string
	switch language {
	case "go":
		queryStr = `(import_spec path: (package_identifier) @name)`
	case "csharp":
		queryStr = `(using_directive (qualified_name) @name)`
	case "javascript", "typescript":
		queryStr = `(import_declaration source: (string (string_fragment) @name))`
	case "python":
		queryStr = `
(import_statement name: (dotted_name) @name)
(import_from_statement module_name: (dotted_name) @name)`
	case "ruby":
		queryStr = `(call method: (identifier) @method (#eq? @method "require") receiver: (constant)? argument: (string (string_content) @name))`
	default:
		return nil, nil // Add support for other languages
	}

	lang := languageMap[language]
	q, err := treesitter.NewQuery([]byte(queryStr), lang)
	if err != nil {
		return nil, err
	}
	cursor := treesitter.NewQueryCursor()
	cursor.Exec(q, tree.RootNode())

	var imports []string
	for {
		match, found := cursor.NextMatch()
		if !found {
			break
		}
		for _, capture := range match.Captures {
			if q.CaptureNameForId(capture.Index) == "name" {
				imports = append(imports, capture.Node.Content([]byte(content)))
			}
		}
	}
	return imports, nil
}

var _ entities.Tool = (*ProjectTool)(nil)
