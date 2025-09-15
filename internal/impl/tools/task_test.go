package tools

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"go.uber.org/zap"
)

func TestTaskTool_SimplifiedInterface(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tasktool_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create TaskTool
	config := map[string]string{"data_dir": tempDir}
	tool := NewTaskTool("test-task", "Test Task Tool", config, logger)

	// Test 1: Create a new task
	result, err := tool.Execute(`{"operation": "write", "content": "Test task 1"}`)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}
	if result != "✅ Task created: Test task 1" {
		t.Errorf("Expected '✅ Task created: Test task 1', got '%s'", result)
	}

	// Test 2: Read tasks
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}
	if !contains(result, "Test task 1") {
		t.Errorf("Expected task list to contain 'Test task 1', got: %s", result)
	}
	if !contains(result, "☐") {
		t.Errorf("Expected task to show empty checkbox '☐', got: %s", result)
	}

	// Test 3: Extract ID from output and mark task as done
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}

	// Extract ID from the output (should contain "ID: <uuid>")
	idStart := strings.Index(result, "ID: ")
	if idStart == -1 {
		t.Fatalf("Expected to find ID in output, got: %s", result)
	}
	idEnd := strings.Index(result[idStart:], "\n")
	if idEnd == -1 {
		idEnd = len(result)
	} else {
		idEnd += idStart
	}
	taskID := strings.TrimSpace(result[idStart+4 : idEnd])

	// Test 4: Mark task as done using the extracted ID
	result, err = tool.Execute(fmt.Sprintf(`{"operation": "write", "id": "%s", "done": true}`, taskID))
	if err != nil {
		t.Fatalf("Failed to mark task as done: %v", err)
	}
	if !contains(result, "Task updated") {
		t.Errorf("Expected task to be updated, got: %s", result)
	}

	// Test 5: Verify task is marked as done
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}
	if !contains(result, "☑") {
		t.Errorf("Expected task to show checked checkbox '☑', got: %s", result)
	}

	// Read tasks to get the ID
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}

	// Extract ID from result (this is a simple test, in real usage the agent would parse this)
	// For now, we'll just verify the interface works by creating and marking done in one call
	result, err = tool.Execute(`{"operation": "write", "id": "test-id", "done": true}`)
	if err == nil {
		t.Errorf("Expected error for non-existent ID, but got success")
	}

	// Test 4: Verify parameters
	params := tool.Parameters()
	if len(params) != 4 {
		t.Errorf("Expected 4 parameters, got %d", len(params))
	}

	// Check parameter names
	expectedNames := []string{"operation", "content", "done", "id"}
	for i, param := range params {
		if param.Name != expectedNames[i] {
			t.Errorf("Expected parameter %d to be '%s', got '%s'", i, expectedNames[i], param.Name)
		}
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsAt(s, substr)))
}

func containsAt(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
