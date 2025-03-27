package tools

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"aiagent/internal/domain/interfaces"

	"github.com/gorilla/websocket"
	"go.uber.org/zap"
)

// MCPTool represents a tool that interacts with an MCP server.
type MCPTool struct {
	configuration map[string]string
	logger        *zap.Logger
	conn          *websocket.Conn
	enabled       bool // Tracks if the tool is usable based on config
}

// NewMCPTool creates a new instance of MCPTool, disabling it if MCP_SERVER_URL is missing.
func NewMCPTool(configuration map[string]string, logger *zap.Logger) *MCPTool {
	tool := &MCPTool{
		configuration: configuration,
		logger:        logger,
	}

	// Check for MCP_SERVER_URL at initialization
	url, ok := configuration["mcp_server_url"]
	if !ok || url == "" {
		logger.Warn("MCP tool disabled: MCP_SERVER_URL is missing from configuration")
		tool.enabled = false
	} else {
		tool.enabled = true
		if err := tool.connect(); err != nil {
			logger.Error("Failed to initialize MCP WebSocket connection", zap.Error(err))
			tool.enabled = false // Disable if connection fails
		}
	}
	return tool
}

// Name returns the name of the tool.
func (t *MCPTool) Name() string {
	return "MCP"
}

// Description returns a description of the tool.
func (t *MCPTool) Description() string {
	if !t.enabled {
		return "A tool to interact with an MCP server (currently disabled due to missing MCP_SERVER_URL)"
	}
	return "A tool to interact with an MCP server, with dynamic method and parameter discovery"
}

// Configuration returns the required configuration keys.
func (t *MCPTool) Configuration() []string {
	return []string{"mcp_server_url"}
}

// Parameters returns the parameters required by the tool.
func (t *MCPTool) Parameters() []interfaces.Parameter {
	if !t.enabled {
		return []interfaces.Parameter{} // No parameters if disabled
	}
	return []interfaces.Parameter{
		{
			Name:        "method",
			Type:        "string",
			Description: "The MCP method to call (e.g., 'search', 'listMethods', 'describeMethod')",
			Required:    true,
		},
		{
			Name:        "params",
			Type:        "object",
			Description: "Parameters for the MCP method as a JSON object",
			Required:    false,
		},
		{
			Name:        "timeout",
			Type:        "integer",
			Description: "Timeout in seconds for the request (default: 30)",
			Required:    false,
		},
	}
}

// MCPArgs defines the structure of the arguments for MCP execution.
type MCPArgs struct {
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
	Timeout int                    `json:"timeout"`
}

// MCPRequest represents a JSON-RPC 2.0 request to the MCP server.
type MCPRequest struct {
	JSONRPC string                 `json:"jsonrpc"`
	ID      int                    `json:"id"`
	Method  string                 `json:"method"`
	Params  map[string]interface{} `json:"params,omitempty"`
}

// MCPResponse represents a JSON-RPC 2.0 response from the MCP server.
type MCPResponse struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      int             `json:"id"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *MCPError       `json:"error,omitempty"`
}

// MCPError represents an error response from the MCP server.
type MCPError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPMethodDescription defines the structure of a method's metadata.
type MCPMethodDescription struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  []interfaces.Parameter `json:"parameters"`
}

// Execute sends a request to the MCP server and returns the response.
func (t *MCPTool) Execute(arguments string) (string, error) {
	if !t.enabled {
		t.logger.Info("MCP tool execution attempted but tool is disabled due to missing MCP_SERVER_URL")
		return "", fmt.Errorf("MCP tool is disabled: MCP_SERVER_URL not configured")
	}

	t.logger.Debug("Executing MCP request", zap.String("arguments", arguments))

	var args MCPArgs
	if err := json.Unmarshal([]byte(arguments), &args); err != nil {
		t.logger.Error("Failed to parse arguments", zap.Error(err))
		return "", err
	}
	if args.Method == "" {
		t.logger.Error("Method is required")
		return "", fmt.Errorf("method is required")
	}
	if args.Timeout == 0 {
		args.Timeout = 30
	}

	switch args.Method {
	case "listMethods":
		return t.DiscoverMethods(args.Timeout)
	case "describeMethod":
		return t.DescribeMethod(args.Params, args.Timeout)
	}

	if t.conn == nil {
		if err := t.connect(); err != nil {
			return "", err
		}
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      int(time.Now().UnixNano()),
		Method:  args.Method,
		Params:  args.Params,
	}
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.logger.Error("Failed to marshal MCP request", zap.Error(err))
		return "", err
	}

	err = t.conn.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		t.logger.Error("Failed to send MCP request", zap.Error(err))
		t.conn = nil
		return "", err
	}

	t.conn.SetReadDeadline(time.Now().Add(time.Duration(args.Timeout) * time.Second))
	_, respBytes, err := t.conn.ReadMessage()
	if err != nil {
		t.logger.Error("Failed to read MCP response", zap.Error(err))
		t.conn = nil
		return "", err
	}

	var resp MCPResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.logger.Error("Failed to parse MCP response", zap.Error(err))
		return "", err
	}

	if resp.Error != nil {
		t.logger.Error("MCP server returned an error",
			zap.Int("code", resp.Error.Code),
			zap.String("message", resp.Error.Message))
		return "", fmt.Errorf("MCP error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	resultStr := string(resp.Result)
	t.logger.Info("MCP request completed",
		zap.String("method", args.Method),
		zap.String("result", resultStr))
	return resultStr, nil
}

// DiscoverMethods queries the MCP server for available methods.
func (t *MCPTool) DiscoverMethods(timeout int) (string, error) {
	if !t.enabled {
		return "", fmt.Errorf("MCP tool is disabled: MCP_SERVER_URL not configured")
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      int(time.Now().UnixNano()),
		Method:  "listMethods",
	}
	respBytes, err := t.sendRequest(req, timeout)
	if err != nil {
		return "", err
	}

	var resp MCPResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.logger.Error("Failed to parse discovery response", zap.Error(err))
		return "", err
	}

	if resp.Error != nil {
		t.logger.Warn("MCP server does not support listMethods",
			zap.Int("code", resp.Error.Code),
			zap.String("message", resp.Error.Message))
		return "", fmt.Errorf("discovery failed: %s", resp.Error.Message)
	}

	var methods []string
	if err := json.Unmarshal(resp.Result, &methods); err != nil {
		t.logger.Error("Failed to parse methods list", zap.Error(err))
		return "", err
	}

	resultStr := "Available MCP methods:\n" + string(resp.Result)
	t.logger.Info("Discovered MCP methods", zap.Strings("methods", methods))
	return resultStr, nil
}

// DescribeMethod queries the MCP server for a method's details.
func (t *MCPTool) DescribeMethod(params map[string]interface{}, timeout int) (string, error) {
	if !t.enabled {
		return "", fmt.Errorf("MCP tool is disabled: MCP_SERVER_URL not configured")
	}

	method, ok := params["method"].(string)
	if !ok || method == "" {
		t.logger.Error("describeMethod requires a 'method' parameter")
		return "", fmt.Errorf("describeMethod requires a 'method' parameter")
	}

	req := MCPRequest{
		JSONRPC: "2.0",
		ID:      int(time.Now().UnixNano()),
		Method:  "describeMethod",
		Params:  map[string]interface{}{"method": method},
	}
	respBytes, err := t.sendRequest(req, timeout)
	if err != nil {
		return "", err
	}

	var resp MCPResponse
	if err := json.Unmarshal(respBytes, &resp); err != nil {
		t.logger.Error("Failed to parse describeMethod response", zap.Error(err))
		return "", err
	}

	if resp.Error != nil {
		t.logger.Warn("MCP server does not support describeMethod",
			zap.Int("code", resp.Error.Code),
			zap.String("message", resp.Error.Message))
		return "", fmt.Errorf("description failed: %s", resp.Error.Message)
	}

	var desc MCPMethodDescription
	if err := json.Unmarshal(resp.Result, &desc); err != nil {
		t.logger.Error("Failed to parse method description", zap.Error(err))
		return "", err
	}

	resultStr := fmt.Sprintf("Method: %s\nDescription: %s\nParameters:\n%s",
		desc.Name, desc.Description, formatParameters(desc.Parameters))
	t.logger.Info("Described MCP method",
		zap.String("method", desc.Name),
		zap.String("description", desc.Description))
	return resultStr, nil
}

// sendRequest is a helper to send an MCP request and get a response.
func (t *MCPTool) sendRequest(req MCPRequest, timeout int) ([]byte, error) {
	reqBytes, err := json.Marshal(req)
	if err != nil {
		t.logger.Error("Failed to marshal request", zap.Error(err))
		return nil, err
	}

	err = t.conn.WriteMessage(websocket.TextMessage, reqBytes)
	if err != nil {
		t.logger.Error("Failed to send request", zap.Error(err))
		t.conn = nil
		return nil, err
	}

	t.conn.SetReadDeadline(time.Now().Add(time.Duration(timeout) * time.Second))
	_, respBytes, err := t.conn.ReadMessage()
	if err != nil {
		t.logger.Error("Failed to read response", zap.Error(err))
		t.conn = nil
		return nil, err
	}
	return respBytes, nil
}

// formatParameters formats a list of parameters into a readable string.
func formatParameters(params []interfaces.Parameter) string {
	if len(params) == 0 {
		return "  None"
	}
	var lines []string
	for _, p := range params {
		req := ""
		if p.Required {
			req = " (required)"
		}
		lines = append(lines, fmt.Sprintf("  - %s (%s)%s: %s", p.Name, p.Type, req, p.Description))
	}
	return strings.Join(lines, "\n")
}

// connect establishes a WebSocket connection to the MCP server.
func (t *MCPTool) connect() error {
	url, ok := t.configuration["mcp_server_url"]
	if !ok || url == "" {
		// This should never happen since enabled is checked, but included for safety
		t.logger.Error("MCP server URL not configured")
		return fmt.Errorf("mcp_server_url not found in configuration")
	}

	conn, _, err := websocket.DefaultDialer.Dial(url, nil)
	if err != nil {
		t.logger.Error("Failed to connect to MCP server",
			zap.String("url", url),
			zap.Error(err))
		return err
	}

	t.conn = conn
	t.logger.Info("Connected to MCP server", zap.String("url", url))
	return nil
}
