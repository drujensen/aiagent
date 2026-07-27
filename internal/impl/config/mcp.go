package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// MCPServerConfig describes one configured MCP server that aiagent should
// connect to as a client (stdio transport only for Milestone 1).
type MCPServerConfig struct {
	Name    string   `json:"name"`
	Command string   `json:"command"`
	Args    []string `json:"args,omitempty"`
	Env     []string `json:"env,omitempty"`
}

// LoadMCPServers reads the configured MCP servers from
// ~/.aiagent/mcp_servers.json. A missing file is not an error - it means no
// MCP servers are configured, which is the default and most common case.
func LoadMCPServers(logger *zap.Logger) ([]MCPServerConfig, error) {
	path := filepath.Join(os.Getenv("HOME"), ".aiagent", "mcp_servers.json")

	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read MCP server config at %s: %w", path, err)
	}

	var servers []MCPServerConfig
	if err := json.Unmarshal(data, &servers); err != nil {
		return nil, fmt.Errorf("failed to parse MCP server config at %s: %w", path, err)
	}

	logger.Debug("Loaded MCP server configuration", zap.Int("count", len(servers)), zap.String("path", path))
	return servers, nil
}
