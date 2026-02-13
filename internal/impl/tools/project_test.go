package tools

import (
	"testing"

	"go.uber.org/zap"
)

func TestProjectTool_Read(t *testing.T) {
	// Test that the ProjectTool can be instantiated and has the correct schema
	// This is a basic unit test to ensure the tool works after removing get_source

	logger := zap.NewNop()
	config := map[string]string{"project_file": "AGENTS.md"}

	tool := NewProjectTool("test-project", "Test project tool", config, logger)

	if tool.Name() != "test-project" {
		t.Errorf("Expected name 'test-project', got '%s'", tool.Name())
	}

	if tool.Description() == "" {
		t.Error("Expected non-empty description")
	}

	// Test schema only has 'read' operation
	schema := tool.Schema()
	if schema["type"] != "object" {
		t.Error("Expected schema type 'object'")
	}

	properties, ok := schema["properties"].(map[string]any)
	if !ok {
		t.Fatal("Expected properties in schema")
	}

	operation, exists := properties["operation"]
	if !exists {
		t.Error("Expected 'operation' property in schema")
	}

	opProps, ok := operation.(map[string]any)
	if !ok {
		t.Fatal("Expected operation to be a map")
	}

	enum, exists := opProps["enum"]
	if !exists {
		t.Error("Expected enum in operation")
	}

	enumValues, ok := enum.([]string)
	if !ok {
		t.Fatal("Expected enum to be string array")
	}

	if len(enumValues) != 2 || enumValues[0] != "read" || enumValues[1] != "structure" {
		t.Errorf("Expected 'read' and 'structure' operations, got %v", enumValues)
	}
}
