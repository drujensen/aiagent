//go:build mcp_integration

// This file is AC2's deliverable test: a real, named reference MCP server,
// exercised end-to-end (tools/list then tools/call). It requires network
// access and npx/node to fetch and run
// @modelcontextprotocol/server-filesystem, so it is excluded from the
// default `go test ./...` gate via the mcp_integration build tag and run
// explicitly with:
//
//	go test -tags=mcp_integration ./internal/impl/tools/mcp/...
package mcp

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/impl/config"

	"go.uber.org/zap"
)

func findToolByNameContains(toolList []entities.Tool, substr string) entities.Tool {
	for _, tl := range toolList {
		if strings.Contains(tl.Name(), substr) {
			return tl
		}
	}
	return nil
}

// TestMCPToolSource_DiscoverAndInvoke is AC2: connects to a real, named
// reference MCP server (the official filesystem server), discovers its
// tools via tools/list, and successfully invokes a read tool via
// tools/call, verifying the returned content matches a fixture file's known
// contents.
func TestMCPToolSource_DiscoverAndInvoke(t *testing.T) {
	fixtureDir := t.TempDir()
	fixtureFile := filepath.Join(fixtureDir, "fixture.txt")
	const knownContent = "hello from the aiagent MCP integration test"
	if err := os.WriteFile(fixtureFile, []byte(knownContent), 0644); err != nil {
		t.Fatalf("failed to write fixture file: %v", err)
	}

	source := NewToolSource(config.MCPServerConfig{
		Name:    "filesystem",
		Command: "npx",
		Args:    []string{"-y", "@modelcontextprotocol/server-filesystem", fixtureDir},
	}, zap.NewNop())
	defer source.Close()

	ctx := context.Background()

	toolList, err := source.ListTools(ctx)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}
	if len(toolList) == 0 {
		t.Fatal("expected at least one tool from the reference filesystem server, got none")
	}

	// Find whichever read-tool this server version exposes (older versions
	// call it "read_file", newer ones "read_text_file") rather than
	// hardcoding one name.
	var readTool = findToolByNameContains(toolList, "read")
	if readTool == nil {
		names := make([]string, len(toolList))
		for i, tl := range toolList {
			names[i] = tl.Name()
		}
		t.Fatalf("no read-capable tool found among: %s", strings.Join(names, ", "))
	}

	argsJSON := `{"path": "` + fixtureFile + `"}`
	result, err := readTool.Execute(ctx, argsJSON)
	if err != nil {
		t.Fatalf("Execute on tool %s failed: %v", readTool.Name(), err)
	}

	if !strings.Contains(result, knownContent) {
		t.Fatalf("expected tool result to contain fixture content %q, got: %s", knownContent, result)
	}
}
