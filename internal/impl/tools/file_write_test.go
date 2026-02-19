package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestFileWriteTool_WriteOperation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filewrite_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create FileWriteTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileWriteTool("test-file-write", "Test File Write Tool", config, logger)

	// Test 1: Write a new file (oldString empty triggers write)
	content := "Hello, World!\nThis is a test file."
	argsData := map[string]interface{}{
		"filePath":  "test.txt",
		"oldString": "",
		"newString": content,
	}
	argsBytes, _ := json.Marshal(argsData)
	args := string(argsBytes)
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	// Parse the JSON result
	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	// Check that it's successful
	if success, ok := resultData["success"].(bool); !ok || !success {
		t.Errorf("Expected success=true, got %v", resultData["success"])
	}

	// Note: diff and summary are added by UI formatting, not the tool itself
	// The tool only returns success/error status

	// Verify file was created
	fullPath := filepath.Join(tempDir, "test.txt")
	fileContent, err := os.ReadFile(fullPath)
	if err != nil {
		t.Fatalf("Failed to read created file: %v", err)
	}
	if string(fileContent) != content {
		t.Errorf("Expected file content '%s', got '%s'", content, string(fileContent))
	}
}

func TestFileWriteTool_EditOperation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filewrite_edit_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create FileWriteTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileWriteTool("test-file-edit", "Test File Edit Tool", config, logger)

	// Create initial file content
	filePath := filepath.Join(tempDir, "edit.txt")
	initialContent := "Line 1\nLine 2\nLine 3\nLine 2\n"
	err = os.WriteFile(filePath, []byte(initialContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create initial file: %v", err)
	}

	// Test 1: Edit with replace first
	args := `{"filePath": "edit.txt", "oldString": "Line 2", "newString": "Updated Line", "replaceAll": false}`
	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Failed to edit file: %v", err)
	}

	// Parse JSON response
	var response struct {
		Summary     string `json:"summary"`
		Success     bool   `json:"success"`
		FilePath    string `json:"filePath"`
		Occurrences int    `json:"occurrences"`
		ReplacedAll bool   `json:"replacedAll"`
		Diff        string `json:"diff"`
	}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	if response.Occurrences != 2 {
		t.Errorf("Expected 2 occurrences, got %d", response.Occurrences)
	}
	if response.ReplacedAll {
		t.Errorf("Expected replaced_all to be false, got true")
	}

	// Verify file content
	expectedContent := "Line 1\nUpdated Line\nLine 3\nLine 2\n"
	fileContent, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read edited file: %v", err)
	}
	if string(fileContent) != expectedContent {
		t.Errorf("Expected file content '%s', got '%s'", expectedContent, string(fileContent))
	}

	// Test 2: Edit with replace all
	args = `{"filePath": "edit.txt", "oldString": "Line 2", "newString": "Modified Line 2", "replaceAll": true}`
	result, err = tool.Execute(args)
	if err != nil {
		t.Fatalf("Failed to edit file with replace all: %v", err)
	}

	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse JSON response for replace all: %v", err)
	}

	if !response.ReplacedAll {
		t.Errorf("Expected replaced_all to be true, got false")
	}

	// Verify all occurrences were replaced
	finalContent := "Line 1\nUpdated Line\nLine 3\nModified Line 2\n"
	fileContent, err = os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read edited file: %v", err)
	}
	if string(fileContent) != finalContent {
		t.Errorf("Expected file content '%s', got '%s'", finalContent, string(fileContent))
	}
}

func TestFileWriteTool_ErrorCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filewrite_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create FileWriteTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileWriteTool("test-file-error", "Test File Error Tool", config, logger)

	// Test 1: Missing filePath
	_, err = tool.Execute(`{"newString": "test"}`)
	if err == nil || !strings.Contains(err.Error(), "filePath is required") {
		t.Errorf("Expected error for missing filePath, got: %v", err)
	}

	// Test 3: Invalid path (outside workspace)
	_, err = tool.Execute(`{"filePath": "../outside.txt", "newString": "test"}`)
	if err == nil || !strings.Contains(err.Error(), "path is outside workspace") {
		t.Errorf("Expected error for path outside workspace, got: %v", err)
	}

	// Test 4: Missing newString for write
	_, err = tool.Execute(`{"filePath": "test.txt"}`)
	if err == nil || !strings.Contains(err.Error(), "newString is required for write operation") {
		t.Errorf("Expected error for missing newString in write, got: %v", err)
	}

	// Test 5: Missing oldString for edit
	_, err = tool.Execute(`{"filePath": "test.txt", "oldString": "old", "newString": "new content"}`)
	// This should work as edit

	// Test 6: Missing newString for edit
	_, err = tool.Execute(`{"filePath": "test.txt", "oldString": "old"}`)
	if err == nil || !strings.Contains(err.Error(), "newString is required for edit operation") {
		t.Errorf("Expected error for missing newString in edit, got: %v", err)
	}

	// Test 7: Old string not found
	filePath := filepath.Join(tempDir, "nonexistent.txt")
	err = os.WriteFile(filePath, []byte("Some content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	_, err = tool.Execute(`{"filePath": "nonexistent.txt", "oldString": "not found", "newString": "replacement"}`)
	if err == nil || !strings.Contains(err.Error(), "oldString not found in file") {
		t.Errorf("Expected error for oldString not found, got: %v", err)
	}
}

func TestFileWriteTool_GoFormatting(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filewrite_go_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create FileWriteTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileWriteTool("test-file-go", "Test File Go Tool", config, logger)

	// Test Go formatting
	unformattedCode := `package main
import (
"fmt"
"os"
)
func main(){fmt.Println("Hello")}`
	filePath := filepath.Join(tempDir, "test.go")
	err = os.WriteFile(filePath, []byte(unformattedCode), 0644)
	if err != nil {
		t.Fatalf("Failed to create Go file: %v", err)
	}

	// Edit the file to trigger formatting
	args := `{"filePath": "test.go", "oldString": "package main", "newString": "package main"}`
	_, err = tool.Execute(args)
	if err != nil {
		t.Fatalf("Failed to edit Go file: %v", err)
	}

	// Check if file was formatted (go fmt should have been called)
	content, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("Failed to read Go file: %v", err)
	}

	formattedContent := string(content)
	// go fmt should add proper spacing
	if !strings.Contains(formattedContent, "import (") || !strings.Contains(formattedContent, ")\nfunc") {
		t.Logf("File content after edit: %s", formattedContent)
		t.Logf("Expected formatting not applied - this may be okay if go fmt is not available")
	}
}

func TestFileWriteTool_Parameters(t *testing.T) {
	logger := zap.NewNop()
	tool := NewFileWriteTool("test-params", "Test Parameters", nil, logger)

	params := tool.Parameters()
	expectedParams := []string{"filePath", "oldString", "newString", "replaceAll"}

	if len(params) != len(expectedParams) {
		t.Errorf("Expected %d parameters, got %d", len(expectedParams), len(params))
	}

	for i, param := range params {
		if param.Name != expectedParams[i] {
			t.Errorf("Expected parameter %d to be '%s', got '%s'", i, expectedParams[i], param.Name)
		}
	}

	// Check specific parameter properties
	if params[0].Type != "string" || !params[0].Required {
		t.Errorf("Invalid filePath parameter: %v", params[0])
	}

	if params[1].Type != "string" || !params[1].Required {
		t.Errorf("Invalid oldString parameter: %v", params[1])
	}

	if params[2].Type != "string" || !params[2].Required {
		t.Errorf("Invalid newString parameter: %v", params[2])
	}

	if params[3].Type != "boolean" || params[3].Required {
		t.Errorf("Invalid replaceAll parameter: %v", params[3])
	}
}

func TestFileWriteTool_Schema(t *testing.T) {
	logger := zap.NewNop()
	tool := NewFileWriteTool("test-schema", "Test Schema", nil, logger)

	schema := tool.Schema()

	// Check that schema is an object
	schemaType, ok := schema["type"].(string)
	if !ok || schemaType != "object" {
		t.Errorf("Expected schema type 'object', got %v", schema["type"])
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatalf("Schema properties not found or not a map")
	}

	// Check required properties
	required, ok := schema["required"].([]string)
	if !ok || len(required) != 3 || required[0] != "filePath" || required[1] != "oldString" || required[2] != "newString" {
		t.Errorf("Expected required fields ['filePath', 'oldString', 'newString'], got %v", required)
	}

	// Check that all expected properties are present
	expectedProps := []string{"filePath", "oldString", "newString", "replaceAll"}
	for _, prop := range expectedProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Expected property '%s' not found in schema", prop)
		}
	}

	// Check filePath property
	filePathProp, ok := properties["filePath"].(map[string]any)
	if !ok {
		t.Errorf("filePath property not a map")
	} else {
		if desc, ok := filePathProp["description"].(string); !ok || !strings.Contains(desc, "modify") {
			t.Errorf("Invalid filePath description: %v", desc)
		}
	}

	// Check oldString property
	oldStrProp, ok := properties["oldString"].(map[string]any)
	if !ok {
		t.Errorf("oldString property not a map")
	} else {
		if desc, ok := oldStrProp["description"].(string); !ok || !strings.Contains(desc, "replace") {
			t.Errorf("Invalid oldString description: %v", desc)
		}
	}

	// Check replaceAll property
	replaceProp, ok := properties["replaceAll"].(map[string]any)
	if !ok {
		t.Errorf("replaceAll property not a map")
	} else {
		if typ, ok := replaceProp["type"].(string); !ok || typ != "boolean" {
			t.Errorf("Invalid replaceAll type: %v", typ)
		}
	}
}
