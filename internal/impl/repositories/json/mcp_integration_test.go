//go:build mcp_integration

// See internal/impl/tools/mcp/integration_test.go for why this is
// build-tagged and excluded from the default go test ./... gate. This file
// verifies the repository-level merge/fallback path specifically (ListTools,
// GetToolByName, GetToolForChat all consulting a real MCP ToolSource), which
// the mcp package's own test does not exercise since it talks to ToolSource
// directly.
package repositories_json

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drujensen/aiagent/internal/impl/config"
	"github.com/drujensen/aiagent/internal/impl/tools"
	mcptools "github.com/drujensen/aiagent/internal/impl/tools/mcp"

	"go.uber.org/zap"
)

func TestJsonToolRepository_MCPFallback(t *testing.T) {
	fixtureDir := t.TempDir()
	fixtureFile := filepath.Join(fixtureDir, "fixture.txt")
	const knownContent = "hello from the aiagent repository-level MCP test"
	if err := os.WriteFile(fixtureFile, []byte(knownContent), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	source := mcptools.NewToolSource(config.MCPServerConfig{
		Name:    "filesystem",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", fixtureDir},
	}, zap.NewNop())
	defer source.Close()

	storageDir := t.TempDir()
	toolFactory, err := newTestToolFactory(t)
	if err != nil {
		t.Fatalf("failed to create tool factory: %v", err)
	}
	repo, err := NewJSONToolRepository(storageDir, toolFactory, identityConfigResolver{}, []tools.ToolSource{source}, testLogger())
	if err != nil {
		t.Fatalf("failed to create repo: %v", err)
	}

	ctx := context.Background()

	// ListTools must include the MCP-sourced tools alongside native ones.
	allTools, err := repo.ListTools()
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	var readToolName string
	for _, tl := range allTools {
		if strings.Contains(tl.Name(), "read") {
			readToolName = tl.Name()
			break
		}
	}
	if readToolName == "" {
		t.Fatal("expected a read-capable MCP tool in the merged ListTools() result")
	}

	// GetToolByName must find it too (the fallback path).
	byName, err := repo.GetToolByName(readToolName)
	if err != nil {
		t.Fatalf("GetToolByName failed: %v", err)
	}
	if byName == nil {
		t.Fatalf("GetToolByName(%q) returned nil - expected the MCP fallback to find it", readToolName)
	}

	// GetToolForChat must find it too (the actual per-chat-turn path chat_service uses).
	forChat, err := repo.GetToolForChat(ctx, readToolName, nil)
	if err != nil {
		t.Fatalf("GetToolForChat failed: %v", err)
	}

	result, err := forChat.Execute(ctx, `{"path": "`+fixtureFile+`"}`)
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if !strings.Contains(result, knownContent) {
		t.Fatalf("expected result to contain fixture content %q, got: %s", knownContent, result)
	}
}
