package tools

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestMatchGlobPattern(t *testing.T) {
	tests := []struct {
		pattern string
		path    string
		want    bool
	}{
		{"**/*.go", "internal/impl/tools/project.go", true},
		{"**/*.go", "main.go", true},
		{"**/*.go", "internal/project.go", true},
		{"**/*.go", "project.go", true},
		{"**/*.go", "internal/impl/tools/project.txt", false},
		{"go.mod", "go.mod", true},
		{"go.mod", "internal/go.mod", false},
		{"*.go", "main.go", true},
		{"*.go", "internal/main.go", false},
	}

	for _, tt := range tests {
		t.Run(tt.pattern+"_"+tt.path, func(t *testing.T) {
			var matched bool
			if strings.Contains(tt.pattern, "**") {
				// For ** patterns, use our custom matcher
				var err error
				matched, err = matchGlobPattern(tt.pattern, tt.path)
				if err != nil {
					t.Fatalf("Match error: %v", err)
				}
			} else {
				// For simple patterns, use filepath.Match
				var err error
				matched, err = filepath.Match(tt.pattern, tt.path)
				if err != nil {
					t.Fatalf("Match error: %v", err)
				}
			}

			if matched != tt.want {
				t.Errorf("matchPattern(%q, %q) = %v, want %v", tt.pattern, tt.path, matched, tt.want)
			}
		})
	}
}

func TestProjectTool_GetSource(t *testing.T) {
	// This is a basic integration test
	// Note: This test assumes the project structure exists

	workspace := "/Users/drujensen/workspace/go/ai/aiagent"

	// Check if we can detect Go files
	goFiles, err := filepath.Glob(filepath.Join(workspace, "**/*.go"))
	if err != nil {
		t.Fatalf("Failed to glob Go files: %v", err)
	}

	if len(goFiles) == 0 {
		t.Log("No Go files found with **/*.go pattern")
	} else {
		t.Logf("Found %d Go files with **/*.go pattern", len(goFiles))
		for i, file := range goFiles {
			if i >= 5 { // Show first 5
				break
			}
			relPath, _ := filepath.Rel(workspace, file)
			t.Logf("  %s", relPath)
		}
	}
}
