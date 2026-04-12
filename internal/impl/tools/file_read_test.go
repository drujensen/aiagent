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

func TestFileReadTool_ReadOperation(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileread_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	var expectedContent string

	// Create FileReadTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileReadTool("test-file-read", "Test File Read Tool", config, logger)

	// Create a test file
	filePath := filepath.Join(tempDir, "test.txt")
	content := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	err = os.WriteFile(filePath, []byte(content), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Test 1: Read entire file
	args := `{"filePath": "test.txt"}`
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	var resultData map[string]interface{}
	if err = json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	expectedContent = content
	if contentResult, ok := resultData["content"].(string); !ok || contentResult != expectedContent {
		t.Errorf("Expected content '%s', got '%s'", expectedContent, contentResult)
	}

	// Test 3: Absolute path within workspace
	absPath := filepath.Join(tempDir, "abs_read.txt")
	absContent := "Absolute content"
	err = os.WriteFile(absPath, []byte(absContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create absolute test file: %v", err)
	}

	argsData := map[string]interface{}{
		"filePath": absPath,
	}
	argsBytes, _ := json.Marshal(argsData)
	args = string(argsBytes)
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Failed to read with absolute path: %v", err)
	}

	if err = json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if contentResult, ok := resultData["content"].(string); !ok || contentResult != absContent {
		t.Errorf("Expected content '%s', got '%s'", absContent, contentResult)
	}
}

func TestFileReadTool_ErrorCases(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "fileread_error_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create FileReadTool
	config := map[string]string{"workspace": tempDir}
	tool := NewFileReadTool("test-file-read-error", "Test File Read Error Tool", config, logger)

	// Test 1: Missing filePath
	args := `{}`
	result, err := tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return error for missing filePath, got: %v", err)
	}

	var resultData map[string]interface{}
	if err = json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); !ok || !strings.Contains(errorStr, "filePath is required") {
		t.Errorf("Expected error for missing filePath, got: %s", errorStr)
	}

	// Test 2: File not found
	args = `{"filePath": "nonexistent.txt"}`
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return error for file not found, got: %v", err)
	}

	if err = json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); !ok || !strings.Contains(errorStr, "no such file") {
		t.Errorf("Expected file open error, got: %s", errorStr)
	}

	// Test 3: Path outside workspace
	args = `{"filePath": "../outside.txt"}`
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return error for path outside workspace, got: %v", err)
	}

	if err = json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); !ok || !strings.Contains(errorStr, "path is outside workspace") {
		t.Errorf("Expected error for path outside workspace, got: %s", errorStr)
	}

	// Test 4: Absolute path outside workspace
	args = `{"filePath": "/outside.txt"}`
	result, err = tool.Execute(context.Background(), args)
	if err != nil {
		t.Fatalf("Execute should not return error for absolute path outside workspace, got: %v", err)
	}

	if err = json.Unmarshal([]byte(result), &resultData); err != nil {
		t.Fatalf("Failed to parse JSON result: %v", err)
	}

	if errorStr, ok := resultData["error"].(string); !ok || !strings.Contains(errorStr, "absolute path is outside workspace") {
		t.Errorf("Expected error for absolute path outside workspace, got: %s", errorStr)
	}
}
