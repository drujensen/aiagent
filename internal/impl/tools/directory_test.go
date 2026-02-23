package tools

import (
	"testing"

	"go.uber.org/zap"
)

func TestDirectoryTool_ValidatePath_Absolute(t *testing.T) {
	logger := zap.NewNop()
	tool := NewDirectoryTool("test", "Test directory tool", map[string]string{
		"workspace": "/tmp/test",
	}, logger)

	// Test absolute path within workspace
	validPath, err := tool.validatePath("/tmp/test/subdir")
	if err != nil {
		t.Errorf("Expected valid absolute path, got error: %v", err)
	}
	if validPath != "/tmp/test/subdir" {
		t.Errorf("Expected /tmp/test/subdir, got %s", validPath)
	}

	// Test absolute path outside workspace
	_, err = tool.validatePath("/tmp/other")
	if err == nil {
		t.Error("Expected error for absolute path outside workspace")
	}

	// Test relative path
	validPath, err = tool.validatePath("subdir")
	if err != nil {
		t.Errorf("Expected valid relative path, got error: %v", err)
	}
	if validPath != "/tmp/test/subdir" {
		t.Errorf("Expected /tmp/test/subdir, got %s", validPath)
	}
}
