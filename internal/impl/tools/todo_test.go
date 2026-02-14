package tools

import (
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
	writeArgs1 := `{"action": "write", "todos": [{"content": "Task for session1"}], "session_id": "` + session1 + `"}`
	_, err := tool.Execute(writeArgs1)
	if err != nil {
		t.Fatal(err)
	}

	readArgs1 := `{"action": "read", "session_id": "` + session1 + `"}`
	read1, err := tool.Execute(readArgs1)
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
	writeArgs2 := `{"action": "write", "todos": [{"content": "Task for session2"}], "session_id": "` + session2 + `"}`
	_, err = tool.Execute(writeArgs2)
	if err != nil {
		t.Fatal(err)
	}

	readArgs2 := `{"action": "read", "session_id": "` + session2 + `"}`
	read2, err := tool.Execute(readArgs2)
	if err != nil {
		t.Fatal(err)
	}
	var readResult2 map[string]interface{}
	json.Unmarshal([]byte(read2), &readResult2)
	if len(readResult2["todos"].([]interface{})) != 1 {
		t.Errorf("Expected 1 todo for session2, got %d", len(readResult2["todos"].([]interface{})))
	}

	// Verify session1 unchanged
	read1Again, _ := tool.Execute(readArgs1)
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
	writeArgs := `{"action": "write", "todos": [{"content": "To be cleared"}], "session_id": "` + session + `"}`
	_, err := tool.Execute(writeArgs)
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
	_, err = tool.Execute(clearArgs)
	if err != nil {
		t.Fatal(err)
	}

	// Verify gone
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("Todo file should be deleted after clear")
	}

	// Test clear on non-existent
	clearArgs2 := `{"action": "clear", "session_id": "nonexistent"}`
	_, err = tool.Execute(clearArgs2)
	if err != nil {
		t.Fatal(err)
	}
}

func TestTodoTool_AlwaysSessionScoped(t *testing.T) {
	wd, _ := os.Getwd()
	config := map[string]string{"workspace": wd}
	observedZapCore, _ := observer.New(zap.DebugLevel)
	logger := zap.New(observedZapCore)

	tool := NewTodoTool("Todo", "test", config, logger)

	// Global write should fail (no session_id)
	writeArgsNoSession := `{"action": "write", "todos": [{"content": "Global task"}]}`
	_, err := tool.Execute(writeArgsNoSession)
	if err == nil || err.Error() != "session_id is required" {
		t.Errorf("Expected session_id required error, got %v", err)
	}

	// Session write succeeds
	session := "test-session"
	writeArgs := `{"action": "write", "todos": [{"content": "Session task"}], "session_id": "` + session + `"}`
	_, err = tool.Execute(writeArgs)
	if err != nil {
		t.Fatal(err)
	}

	// Cleanup
	os.Remove(filepath.Join(wd, ".aiagent", "todos_"+session+".json"))
}
