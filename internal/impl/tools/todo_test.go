package tools

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestTodoTool_PerSessionIsolation(t *testing.T) {
	// Setup
	wd, _ := os.Getwd()
	config := map[string]string{"workspace": wd}
	observedZapCore, _ := observer.New(zap.DebugLevel)
	logger := zap.New(observedZapCore)

	tool := NewTodoTool("Todo", "test", config, logger)

	// Test write/read for session1
	session1 := "test-session-1"
	writeArgs1 := `{"action": "write", "todos": ["Task for session1"], "session_id": "` + session1 + `"}`
	_, err := tool.Execute(context.Background(), writeArgs1)
	if err != nil {
		t.Fatal(err)
	}

	readArgs1 := `{"action": "read", "session_id": "` + session1 + `"}`
	read1, err := tool.Execute(context.Background(), readArgs1)
	if err != nil {
		t.Fatal(err)
	}
	var readResult1 map[string]interface{}
	json.Unmarshal([]byte(read1), &readResult1)
	if len(readResult1["todos"].([]interface{})) != 1 {
		t.Errorf("Expected 1 todo for session1, got %d", len(readResult1["todos"].([]interface{})))
	}

	// Test write/read for session2
	session2 := "test-session-2"
	writeArgs2 := `{"action": "write", "todos": ["Task for session2"], "session_id": "` + session2 + `"}`
	_, err = tool.Execute(context.Background(), writeArgs2)
	if err != nil {
		t.Fatal(err)
	}

	readArgs2 := `{"action": "read", "session_id": "` + session2 + `"}`
	read2, err := tool.Execute(context.Background(), readArgs2)
	if err != nil {
		t.Fatal(err)
	}
	var readResult2 map[string]interface{}
	json.Unmarshal([]byte(read2), &readResult2)
	if len(readResult2["todos"].([]interface{})) != 1 {
		t.Errorf("Expected 1 todo for session2, got %d", len(readResult2["todos"].([]interface{})))
	}

	// Verify session1 unchanged
	read1Again, _ := tool.Execute(context.Background(), readArgs1)
	var readResult1Again map[string]interface{}
	json.Unmarshal([]byte(read1Again), &readResult1Again)
	if len(readResult1Again["todos"].([]interface{})) != 1 {
		t.Errorf("Session1 changed unexpectedly, got %d todos", len(readResult1Again["todos"].([]interface{})))
	}

	// Cleanup
	todoDir := filepath.Join(wd, ".aiagent")
	os.Remove(filepath.Join(todoDir, "todos_test-session-1.json"))
	os.Remove(filepath.Join(todoDir, "todos_test-session-2.json"))
}

func TestTodoTool_Clear(t *testing.T) {
	// Setup
	wd, _ := os.Getwd()
	config := map[string]string{"workspace": wd}
	observedZapCore, _ := observer.New(zap.DebugLevel)
	logger := zap.New(observedZapCore)

	tool := NewTodoTool("Todo", "test", config, logger)

	session := "test-clear-session"
	writeArgs := `{"action": "write", "todos": ["To be cleared"], "session_id": "` + session + `"}`
	_, err := tool.Execute(context.Background(), writeArgs)
	if err != nil {
		t.Fatal(err)
	}

	// Verify exists
	path := filepath.Join(wd, ".aiagent", "todos_"+session+".json")
	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Fatal("Todo file should exist before clear")
	}

	// Clear
	clearArgs := `{"action": "clear", "session_id": "` + session + `"}`
	_, err = tool.Execute(context.Background(), clearArgs)
	if err != nil {
		t.Fatal(err)
	}

	// Verify gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Todo file should be deleted after clear")
	}

	// Test clear on non-existent
	clearArgs2 := `{"action": "clear", "session_id": "nonexistent"}`
	_, err = tool.Execute(context.Background(), clearArgs2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTodoTool_UpdateStatus(t *testing.T) {
	// Setup
	wd, _ := os.Getwd()
	config := map[string]string{"workspace": wd}
	observedZapCore, _ := observer.New(zap.DebugLevel)
	logger := zap.New(observedZapCore)

	tool := NewTodoTool("Todo", "test", config, logger)

	session := "test-update-session"
	writeArgs := `{"action": "write", "todos": ["Task 1", "Task 2"], "session_id": "` + session + `"}`
	_, err := tool.Execute(context.Background(), writeArgs)
	if err != nil {
		t.Fatal(err)
	}

	// Update status of first task (index 1) to in_progress
	updateArgs := `{"action": "update_status", "index": 1, "status": "in_progress", "session_id": "` + session + `"}`
	result, err := tool.Execute(context.Background(), updateArgs)
	if err != nil {
		t.Fatal(err)
	}

	// Verify
	var response map[string]interface{}
	json.Unmarshal([]byte(result), &response)
	todos := response["todos"].([]interface{})
	if len(todos) != 2 {
		t.Errorf("Expected 2 todos, got %d", len(todos))
	}
	task1 := todos[0].(map[string]interface{})
	if task1["status"] != "in_progress" {
		t.Errorf("Expected status in_progress, got %s", task1["status"])
	}
	task2 := todos[1].(map[string]interface{})
	if task2["status"] != "pending" {
		t.Errorf("Expected status pending, got %s", task2["status"])
	}

	// Cleanup
	os.Remove(filepath.Join(wd, ".aiagent", "todos_"+session+".json"))
}

func TestTodoTool_UpdateStatusInvalidIndex(t *testing.T) {
	// Setup
	wd, _ := os.Getwd()
	config := map[string]string{"workspace": wd}
	observedZapCore, _ := observer.New(zap.DebugLevel)
	logger := zap.New(observedZapCore)

	tool := NewTodoTool("Todo", "test", config, logger)

	session := "test-invalid-index-session"

	// Try to update status with index 0 (should fail)
	updateArgs := `{"action": "update_status", "index": 0, "status": "completed", "session_id": "` + session + `"}`
	_, err := tool.Execute(context.Background(), updateArgs)
	if err == nil {
		t.Fatal("Expected error for invalid index 0")
	}
	expectedError := "invalid index 0, must be between 1 and 0"
	if err.Error() != expectedError {
		t.Errorf("Expected error '%s', got '%s'", expectedError, err.Error())
	}

	// Now add todos and try invalid index
	writeArgs := `{"action": "write", "todos": ["Task 1"], "session_id": "` + session + `"}`
	_, err = tool.Execute(context.Background(), writeArgs)
	if err != nil {
		t.Fatal(err)
	}

	// Try index 2 (out of range)
	updateArgs2 := `{"action": "update_status", "index": 2, "status": "completed", "session_id": "` + session + `"}`
	_, err = tool.Execute(context.Background(), updateArgs2)
	if err == nil {
		t.Fatal("Expected error for index 2 when only 1 todo")
	}

	// Cleanup
	os.Remove(filepath.Join(wd, ".aiagent", "todos_"+session+".json"))
}
