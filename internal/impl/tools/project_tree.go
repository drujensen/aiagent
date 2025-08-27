//go:build !windows
// +build !windows

package tools

import (
	"context"
	"fmt"
	"strings"

	treesitter "github.com/smacker/go-tree-sitter"
	"github.com/smacker/go-tree-sitter/c"
	"github.com/smacker/go-tree-sitter/cpp"
	"github.com/smacker/go-tree-sitter/csharp"
	"github.com/smacker/go-tree-sitter/golang"
	"github.com/smacker/go-tree-sitter/java"
	"github.com/smacker/go-tree-sitter/javascript"
	"github.com/smacker/go-tree-sitter/kotlin"
	"github.com/smacker/go-tree-sitter/python"
	"github.com/smacker/go-tree-sitter/ruby"
	"github.com/smacker/go-tree-sitter/rust"
	"github.com/smacker/go-tree-sitter/swift"
	"github.com/smacker/go-tree-sitter/typescript/typescript"
)

// Map of language names to Tree-sitter language objects
var languageMap = map[string]*treesitter.Language{
	"c":          c.GetLanguage(),
	"cpp":        cpp.GetLanguage(),
	"go":         golang.GetLanguage(),
	"csharp":     csharp.GetLanguage(),
	"java":       java.GetLanguage(),
	"javascript": javascript.GetLanguage(),
	"kotlin":     kotlin.GetLanguage(),
	"python":     python.GetLanguage(),
	"ruby":       ruby.GetLanguage(),
	"rust":       rust.GetLanguage(),
	"swift":      swift.GetLanguage(),
	"typescript": typescript.GetLanguage(),
}

// Map of language to tags query (copy from tree-sitter-<lang>/queries/tags.scm)
var tagsQueries = map[string]string{
	"c": `
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @definition.function))
(struct_specifier
  name: (type_identifier) @definition.type)
	`,
	"cpp": `
(function_definition
  declarator: (function_declarator
    declarator: (identifier) @definition.function))
(class_specifier
  name: (type_identifier) @definition.class)
(struct_specifier
  name: (type_identifier) @definition.type)
	`,
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
	"java": `
(class_declaration
  name: (identifier) @definition.class)
(interface_declaration
  name: (identifier) @definition.interface)
(method_declaration
  name: (identifier) @definition.method)
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
	"kotlin": `
(class_declaration
  name: (simple_identifier) @definition.class)
(function_declaration
  name: (simple_identifier) @definition.function)
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
	"rust": `
(function_item
  name: (identifier) @definition.function)
(struct_item
  name: (type_identifier) @definition.type)
(enum_item
  name: (type_identifier) @definition.type)
	`,
	"swift": `
(function_declaration
  name: (identifier) @definition.function)
(class_declaration
  name: (type_identifier) @definition.class)
(struct_declaration
  name: (type_identifier) @definition.type)
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
	case "c", "cpp":
		queryStr = `(preproc_include path: (string_literal) @name)`
	case "go":
		queryStr = `(import_spec path: (package_identifier) @name)`
	case "csharp":
		queryStr = `(using_directive (qualified_name) @name)`
	case "java":
		queryStr = `(import_declaration (scoped_identifier) @name)`
	case "javascript", "typescript":
		queryStr = `(import_declaration source: (string (string_fragment) @name))`
	case "kotlin":
		queryStr = `(import_header (identifier) @name)`
	case "python":
		queryStr = `
(import_statement name: (dotted_name) @name)
(import_from_statement module_name: (dotted_name) @name)`
	case "ruby":
		queryStr = `(call method: (identifier) @method (#eq? @method "require") receiver: (constant)? argument: (string (string_content) @name))`
	case "rust":
		queryStr = `(use_declaration argument: (scoped_identifier) @name)`
	case "swift":
		queryStr = `(import_declaration (identifier) @name)`
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
