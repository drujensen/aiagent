package tools

import (
	"testing"

	"go.uber.org/zap/zaptest"
)

func TestCompressionTool_Basic(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := map[string]string{"workspace": "/tmp"}

	tool := NewCompressionTool("test-compression", "Test compression tool", config, logger)

	// Test basic properties
	if tool.Name() != "test-compression" {
		t.Errorf("Expected name 'test-compression', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}

	// Test schema validation
	schema := tool.Schema()
	if schema["type"] != "object" {
		t.Error("Expected schema type 'object'")
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties in schema")
	}

	if _, exists := properties["action"]; !exists {
		t.Error("Expected 'action' property in schema")
	}
}

func TestCompressionTool_Execute(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := map[string]string{"workspace": "/tmp"}

	tool := NewCompressionTool("test-compression", "Test compression tool", config, logger)

	// Test valid compression request
	args := `{
		"action": "compress_range",
		"start_message_index": 10,
		"end_message_index": 20,
		"summary_type": "task_cleanup",
		"description": "Test compression"
	}`

	result, err := tool.Execute(args)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	// Should contain compression_instruction
	if !contains(result, "compression_instruction") {
		t.Error("Expected compression_instruction in result")
	}

	// Test invalid action
	invalidArgs := `{
		"action": "invalid_action",
		"start_message_index": 10,
		"end_message_index": 20
	}`

	_, err = tool.Execute(invalidArgs)
	if err == nil {
		t.Error("Expected error for invalid action")
	}
}

func TestCompressionTool_InvalidRange(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := map[string]string{"workspace": "/tmp"}

	tool := NewCompressionTool("test-compression", "Test compression tool", config, logger)

	// Test invalid range (start > end)
	invalidArgs := `{
		"action": "compress_range",
		"start_message_index": 20,
		"end_message_index": 10,
		"summary_type": "task_cleanup"
	}`

	_, err := tool.Execute(invalidArgs)
	if err == nil {
		t.Error("Expected error for invalid range")
	}
}

// Helper function to check if string contains substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsHelper(s, substr)))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestCompressionTool_FormatResult(t *testing.T) {
	logger := zaptest.NewLogger(t)
	config := map[string]string{"workspace": "/tmp"}

	tool := NewCompressionTool("test-compression", "Test compression tool", config, logger)

	// Test TUI formatting with valid result
	result := `{
		"compression_instruction": {
			"action": "compress_range",
			"start_message_index": 5,
			"end_message_index": 15,
			"summary_type": "task_cleanup",
			"description": "Cleaning up old task messages"
		},
		"message": "Requested compression of messages 5-15 with type 'task_cleanup'"
	}`

	formatted := tool.FormatResult("tui", result, "", "")
	expected := "🗜️ Compressed messages 5-15 with type 'task_cleanup' (Cleaning up old task messages)"
	if formatted != expected {
		t.Errorf("Expected TUI format '%s', got '%s'", expected, formatted)
	}

	// Test webui formatting (should return the message field)
	webFormatted := tool.FormatResult("webui", result, "", "")
	expectedWeb := "Requested compression of messages 5-15 with type 'task_cleanup'"
	if webFormatted != expectedWeb {
		t.Errorf("Expected webui format '%s', got '%s'", expectedWeb, webFormatted)
	}

	// Test with invalid JSON (should return original result)
	invalidResult := "invalid json"
	fallback := tool.FormatResult("tui", invalidResult, "", "")
	if fallback != invalidResult {
		t.Errorf("Expected fallback to return original result, got '%s'", fallback)
	}
}
