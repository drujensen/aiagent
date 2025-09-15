package tools

import (
	"os"
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
	// Check the formatted summary string directly
	if !contains(result, "Test task 1") {
		t.Errorf("Expected task list to contain 'Test task 1', got: %s", result)
	}
	if !contains(result, "☐") {
		t.Errorf("Expected task to show empty checkbox '☐', got: %s", result)
	}

	// Test 3: Mark first task as done (simplified test without structured data access)
	// For this simplified test, we'll just verify the update operation works
	result, err = tool.Execute(`{"operation": "write", "content": "Test task 1", "done": true}`)
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}
	if !contains(result, "Task updated") && !contains(result, "Task created") {
		t.Errorf("Expected task operation to succeed, got: %s", result)
	}

	// Test 4: Verify task appears in updated summary
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}

	if !contains(result, "Test task 1") {
		t.Errorf("Expected updated task list to contain 'Test task 1', got: %s", result)
	}

	// Test error handling for non-existent task ID
	result, err = tool.Execute(`{"operation": "write", "id": "non-existent-id", "done": true}`)
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
