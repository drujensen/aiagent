// Package mcp adapts external Model Context Protocol servers into
// entities.Tool instances, via the tools.ToolSource interface. This keeps
// MCP-specific SDK types (github.com/modelcontextprotocol/go-sdk/mcp)
// confined to this package - the rest of aiagent only ever sees
// entities.Tool and tools.ToolSource.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/impl/config"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"go.uber.org/zap"
)

// ToolSource connects to one configured MCP server over stdio and exposes
// its tools. Milestone 1 supports the stdio transport only.
type ToolSource struct {
	serverConfig config.MCPServerConfig
	logger       *zap.Logger

	mu      sync.Mutex
	client  *mcpsdk.Client
	session *mcpsdk.ClientSession
}

// NewToolSource creates a ToolSource for the given server configuration. It
// does not connect immediately - the connection is established lazily on
// first use and reused thereafter, so a misconfigured or temporarily
// unreachable server doesn't fail aiagent startup.
func NewToolSource(serverConfig config.MCPServerConfig, logger *zap.Logger) *ToolSource {
	return &ToolSource{
		serverConfig: serverConfig,
		logger:       logger,
	}
}

func (s *ToolSource) Name() string {
	return s.serverConfig.Name
}

// connect establishes the client session if not already connected. Callers
// must hold s.mu.
func (s *ToolSource) connect(ctx context.Context) (*mcpsdk.ClientSession, error) {
	if s.session != nil {
		return s.session, nil
	}

	client := mcpsdk.NewClient(&mcpsdk.Implementation{Name: "aiagent", Version: "1.0.0"}, nil)

	cmd := exec.Command(s.serverConfig.Command, s.serverConfig.Args...)
	if len(s.serverConfig.Env) > 0 {
		cmd.Env = append(os.Environ(), s.serverConfig.Env...)
	}

	session, err := client.Connect(ctx, &mcpsdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MCP server %q: %w", s.serverConfig.Name, err)
	}

	s.client = client
	s.session = session
	return session, nil
}

// Close disconnects from the MCP server, if connected.
func (s *ToolSource) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.session == nil {
		return nil
	}
	err := s.session.Close()
	s.session = nil
	s.client = nil
	return err
}

func (s *ToolSource) ListTools(ctx context.Context) ([]entities.Tool, error) {
	s.mu.Lock()
	session, err := s.connect(ctx)
	s.mu.Unlock()
	if err != nil {
		return nil, err
	}

	result, err := session.ListTools(ctx, &mcpsdk.ListToolsParams{})
	if err != nil {
		return nil, fmt.Errorf("failed to list tools from MCP server %q: %w", s.serverConfig.Name, err)
	}

	toolList := make([]entities.Tool, 0, len(result.Tools))
	for _, t := range result.Tools {
		toolList = append(toolList, newMCPTool(s, t))
	}
	return toolList, nil
}

// mcpTool adapts one MCP-discovered tool into entities.Tool. It re-resolves
// its session from the owning ToolSource on every Execute call rather than
// caching the *mcpsdk.ClientSession directly, so a session recreated after a
// Close (e.g. server restart) is picked up automatically.
type mcpTool struct {
	source *ToolSource
	tool   *mcpsdk.Tool
	config map[string]string
}

func newMCPTool(source *ToolSource, tool *mcpsdk.Tool) *mcpTool {
	return &mcpTool{source: source, tool: tool, config: map[string]string{}}
}

func (t *mcpTool) Name() string {
	return t.tool.Name
}

func (t *mcpTool) Description() string {
	return t.tool.Description
}

func (t *mcpTool) FullDescription() string {
	return fmt.Sprintf("%s (via MCP server %q)", t.tool.Description, t.source.Name())
}

func (t *mcpTool) Configuration() map[string]string {
	return t.config
}

func (t *mcpTool) UpdateConfiguration(c map[string]string) {
	t.config = c
}

// Schema returns the MCP tool's input schema. On the client side,
// mcpsdk.Tool.InputSchema is documented to hold the server's schema decoded
// as map[string]any, so this is a direct type assertion.
func (t *mcpTool) Schema() map[string]any {
	schema, ok := t.tool.InputSchema.(map[string]any)
	if !ok {
		t.source.logger.Warn("MCP tool input schema was not a map[string]any as documented",
			zap.String("tool", t.tool.Name), zap.String("server", t.source.Name()))
		return map[string]any{}
	}
	return schema
}

func (t *mcpTool) Execute(ctx context.Context, arguments string) (string, error) {
	var args map[string]any
	if arguments != "" {
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			return "", fmt.Errorf("failed to parse arguments for MCP tool %s: %w", t.tool.Name, err)
		}
	}

	t.source.mu.Lock()
	session, err := t.source.connect(ctx)
	t.source.mu.Unlock()
	if err != nil {
		return "", err
	}

	result, err := session.CallTool(ctx, &mcpsdk.CallToolParams{
		Name:      t.tool.Name,
		Arguments: args,
	})
	if err != nil {
		return "", fmt.Errorf("failed to call MCP tool %s: %w", t.tool.Name, err)
	}

	var sb strings.Builder
	for _, c := range result.Content {
		if tc, ok := c.(*mcpsdk.TextContent); ok {
			sb.WriteString(tc.Text)
		}
	}

	if result.IsError {
		return sb.String(), fmt.Errorf("MCP tool %s returned an error", t.tool.Name)
	}
	return sb.String(), nil
}

func (t *mcpTool) FormatResult(ui string, result, diff, arguments string) string {
	return result
}

func (t *mcpTool) DisplayName(ui string, arguments string) (string, string) {
	return t.tool.Name, ""
}

var _ entities.Tool = (*mcpTool)(nil)
