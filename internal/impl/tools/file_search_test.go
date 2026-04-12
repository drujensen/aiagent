package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestFileSearchTool_SearchOperation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filesearch_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create FileSearchTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileSearchTool("test-file-search", "Test File Search Tool", config, logger)

	// Create test files
	file1Path := filepath.Join(tempDir, "test1.txt")
	content1 := "Hello world\nThis is a test file\nAnother line"
	err = os.WriteFile(file1Path, []byte(content1), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	file2Path := filepath.Join(tempDir, "test2.txt")
	content2 := "Another world\nDifferent content\nHello again"
	err = os.WriteFile(file2Path, []byte(content2), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test 1: Search for "Hello"
	args := `{"pattern": "Hello"}`
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to search: %v", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); ok && errorStr != "" {
		t.Errorf("Expected no error, got: %s", errorStr)
	}

	results, ok := resultData["results"].([]interface{})
	if !ok {
		t.Fatalf("Results not found in response")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(results))
	}

	// Test 2: Search with include pattern
	args = `{"pattern": "world", "include": "*.txt"}`
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to search with include: %v", err)
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	results, ok = resultData["results"].([]interface{})
	if !ok {
		t.Fatalf("Results not found in response")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for 'world', got %d", len(results))
	}

	// Test 3: Absolute path search
	subDir := filepath.Join(tempDir, "subdir")
	err = os.Mkdir(subDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create subdir: %v", err)
	}

	absFilePath := filepath.Join(subDir, "abs_test.txt")
	absContent := "Absolute search test\nHello absolute"
	err = os.WriteFile(absFilePath, []byte(absContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create absolute test file: %v", err)
	}

	argsData := map[string]interface{}{
		"pattern": "absolute",
		"path":    subDir,
	}
	argsBytes, _ := json.Marshal(argsData)
	args = string(argsBytes)
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to search with absolute path: %v", err)
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	results, ok = resultData["results"].([]interface{})
	if !ok {
		t.Fatalf("Results not found in response")
	}

	if len(results) != 2 {
		t.Errorf("Expected 2 results for absolute search, got %d", len(results))
	}
}

func TestFileSearchTool_ErrorCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "filesearch_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create FileSearchTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileSearchTool("test-file-search-error", "Test File Search Error Tool", config, logger)

	// Test 1: Missing pattern
	args := `{"path": "."}`
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return error for missing pattern, got: %v", err)
	}

	var resultData map[string]interface{}
	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); !ok || !strings.Contains(errorStr, "pattern is required") {
		t.Errorf("Expected pattern required error, got: %s", errorStr)
	}

	// Test 2: Path outside workspace
	args = `{"pattern": "test", "path": "../outside"}`
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return error for invalid path, got: %v", err)
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); !ok || !strings.Contains(errorStr, "invalid path") {
		t.Errorf("Expected invalid path error, got: %s", errorStr)
	}

	// Test 3: Absolute path outside workspace
	args = `{"pattern": "test", "path": "/outside"}`
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return error for absolute path outside, got: %v", err)
	}

	if err := json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); !ok || !strings.Contains(errorStr, "absolute path is outside workspace") {
		t.Errorf("Expected absolute path outside error, got: %s", errorStr)
	}
}
