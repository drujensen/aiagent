package tools

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

func TestFileTool_SearchFiles(t *testing.T) {
	// Initialize a FileTool instance with a logger
	logger, _ := zap.NewDevelopment()
	fileTool := NewFileTool("file", "description", map[string]string{"workspace": "."}, logger)

	// Test case: No matches found
	results, err := fileTool.searchFiles(".", "nonexistent", nil)
	assert.Error(t, err)
	assert.Nil(t, results)
	assert.Contains(t, err.Error(), "no matches found for pattern 'nonexistent' in path '.'")

	// Test case: Matches found
	results, err = fileTool.searchFiles(".", "file.go", nil)
	assert.NoError(t, err)
	assert.NotNil(t, results)
	assert.Greater(t, len(results), 0)
}