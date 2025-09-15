package tools

import (
	"encoding/json"
	"fmt"
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

	// Parse JSON response
	var response struct {
		Summary   string `json:"summary"`
		FullTasks []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"full_tasks"`
	}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if response.Summary != "✅ Task created: Test task 1" {
		t.Errorf("Expected summary '✅ Task created: Test task 1', got '%s'", response.Summary)
	}
	if len(response.FullTasks) != 1 || response.FullTasks[0].Content != "Test task 1" {
		t.Errorf("Expected task content 'Test task 1', got '%v'", response.FullTasks)
	}

	taskID := response.FullTasks[0].ID

	// Test 2: Read tasks
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}

	// Parse JSON response
	var readResponse struct {
		Summary   string `json:"summary"`
		FullTasks []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"full_tasks"`
	}
	if err := json.Unmarshal([]byte(result), &readResponse); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if !contains(readResponse.Summary, "Test task 1") {
		t.Errorf("Expected task list summary to contain 'Test task 1', got: %s", readResponse.Summary)
	}
	if !contains(readResponse.Summary, "☐") {
		t.Errorf("Expected task summary to show empty checkbox '☐', got: %s", readResponse.Summary)
	}
	if len(readResponse.FullTasks) != 1 || readResponse.FullTasks[0].Content != "Test task 1" {
		t.Errorf("Expected full tasks to contain 'Test task 1', got '%v'", readResponse.FullTasks)
	}

	// Test 3: Mark first task as done
	result, err = tool.Execute(fmt.Sprintf(`{"operation": "write", "id": "%s", "done": true}`, taskID))
	if err != nil {
		t.Fatalf("Failed to update task: %v", err)
	}

	// Parse JSON response
	var updateResponse struct {
		Summary   string `json:"summary"`
		FullTasks []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"full_tasks"`
	}
	if err := json.Unmarshal([]byte(result), &updateResponse); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if !contains(updateResponse.Summary, "Task updated") {
		t.Errorf("Expected task update to succeed, got summary: %s", updateResponse.Summary)
	}

	// Test 4: Verify task appears in updated summary
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}

	// Parse JSON response
	var finalReadResponse struct {
		Summary   string `json:"summary"`
		FullTasks []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"full_tasks"`
	}
	if err := json.Unmarshal([]byte(result), &finalReadResponse); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}
	if !contains(finalReadResponse.Summary, "Test task 1") {
		t.Errorf("Expected updated task list summary to contain 'Test task 1', got: %s", finalReadResponse.Summary)
	}
	if len(finalReadResponse.FullTasks) != 1 || finalReadResponse.FullTasks[0].Content != "Test task 1" {
		t.Errorf("Expected full tasks to contain 'Test task 1', got '%v'", finalReadResponse.FullTasks)
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

func TestTaskTool_CreateBuyMilkTask(t *testing.T) {
	// Create a temporary directory for testing
	tempDir, err := os.MkdirTemp("", "tasktool_buymilk_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Create logger
	logger := zap.NewNop()

	// Create TaskTool
	config := map[string]string{"data_dir": tempDir}
	tool := NewTaskTool("test-task", "Test Task Tool", config, logger)

	// Create a new task "Buy Milk"
	result, err := tool.Execute(`{"operation": "write", "content": "Buy Milk"}`)
	if err != nil {
		t.Fatalf("Failed to create task: %v", err)
	}

	// Parse JSON response
	var response struct {
		Summary   string `json:"summary"`
		FullTasks []struct {
			ID      string `json:"id"`
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"full_tasks"`
	}
	if err := json.Unmarshal([]byte(result), &response); err != nil {
		t.Fatalf("Failed to parse JSON response: %v", err)
	}

	// Verify the task was created correctly
	if response.Summary != "✅ Task created: Buy Milk" {
		t.Errorf("Expected summary '✅ Task created: Buy Milk', got '%s'", response.Summary)
	}
	if len(response.FullTasks) != 1 {
		t.Errorf("Expected 1 task, got %d", len(response.FullTasks))
	}
	if response.FullTasks[0].Content != "Buy Milk" {
		t.Errorf("Expected content 'Buy Milk', got '%s'", response.FullTasks[0].Content)
	}
	if response.FullTasks[0].Status != "not done" {
		t.Errorf("Expected status 'not done', got '%s'", response.FullTasks[0].Status)
	}

	// Read tasks to verify it's listed
	result, err = tool.Execute(`{"operation": "read"}`)
	if err != nil {
		t.Fatalf("Failed to read tasks: %v", err)
	}

	var readResponse struct {
		Summary   string `json:"summary"`
		FullTasks []struct {
			Content string `json:"content"`
			Status  string `json:"status"`
		} `json:"full_tasks"`
	}
	if err := json.Unmarshal([]byte(result), &readResponse); err != nil {
		t.Fatalf("Failed to parse read JSON response: %v", err)
	}

	if !contains(readResponse.Summary, "Buy Milk") {
		t.Errorf("Expected read summary to contain 'Buy Milk', got: %s", readResponse.Summary)
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
