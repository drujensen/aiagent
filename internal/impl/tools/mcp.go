package tools

import (
	"aiagent/internal/domain/entities"
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"maps"
	"os"
	"os/exec"
	"strings"
	"time"

	"go.uber.org/zap"
)

type MCPClient struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	stderr io.ReadCloser
	pid    int
	ctx    context.Context
	cancel context.CancelFunc
}

func NewMCPClient(command string, workspace string, args []string) *MCPClient {
	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, command, args...)
	cmd.Dir = workspace
	cmd.Env = os.Environ()
	stdin, err := cmd.StdinPipe()
	if err != nil {
		panic(fmt.Sprintf("Error creating stdin pipe: %v", err))
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		panic(fmt.Sprintf("Error creating stdout pipe: %v", err))
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		panic(fmt.Sprintf("Error creating stderr pipe: %v", err))
	}

	// Start the command and store PID
	if err := cmd.Start(); err != nil {
		panic(fmt.Sprintf("Error starting command: %v", err))
	}

	go func() {
		scanner := bufio.NewScanner(stderr)
		for scanner.Scan() {
			select {
			case <-ctx.Done():
				return
			default:
				fmt.Printf("Server stderr: %s\n", scanner.Text())
			}
		}
		if err := scanner.Err(); err != nil && ctx.Err() == nil {
			fmt.Printf("Error reading server stderr: %v\n", err)
		}
	}()

	return &MCPClient{
		cmd:    cmd,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		pid:    cmd.Process.Pid,
		ctx:    ctx,
		cancel: cancel,
	}
}

func (c *MCPClient) InvokeMethod(method string, params any) (any, error) {

	// JSON-RPC request
	req := map[string]any{
		"jsonrpc": "2.0",
		"method":  method,
		"params":  params,
		"id":      1,
	}
	reqJSON, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("error marshaling request: %w", err)
	}

	// Write request to stdin with newline
	if _, err := fmt.Fprintln(c.stdin, string(reqJSON)); err != nil {
		return nil, fmt.Errorf("error writing to stdin: %w", err)
	}

	// Set a read timeout using context (e.g., 5 seconds)
	ctx, cancel := context.WithTimeout(c.ctx, 5*time.Second)
	defer cancel()

	// Read response with timeout using a goroutine
	reader := bufio.NewReader(c.stdout)
	var responseStr string
	done := make(chan error, 1)
	go func() {
		res, err := reader.ReadString('\n') // Read until newline
		responseStr = res
		done <- err
	}()

	select {
	case err := <-done:
		if err != nil && err != io.EOF {
			return nil, fmt.Errorf("error reading from stdout: %w", err)
		}
	case <-ctx.Done():
		return nil, fmt.Errorf("timeout reading response: %w", ctx.Err())
	}

	fmt.Printf("Received response: %s\n", responseStr)

	// Parse JSON-RPC response
	var resp map[string]any
	if err := json.Unmarshal([]byte(responseStr), &resp); err != nil {
		return nil, fmt.Errorf("error unmarshaling response: %w", err)
	}

	// Check for error in response
	if errorVal, ok := resp["error"]; ok {
		return nil, fmt.Errorf("MCP server error: %v", errorVal)
	}

	// Return the result
	result, ok := resp["result"]
	if !ok {
		return nil, fmt.Errorf("no 'result' in response")
	}
	return result, nil
}

// Close shuts down the MCP server process and cancels context
func (c *MCPClient) Close() {
	if c.cancel != nil {
		c.cancel()
	}
	if c.cmd != nil {
		if err := c.cmd.Process.Kill(); err != nil {
			fmt.Printf("Error killing process: %v\n", err)
		}
		c.cmd.Wait()
	}
	if c.stdin != nil {
		c.stdin.Close()
	}
	if c.stdout != nil {
		c.stdout.Close()
	}
	if c.stderr != nil {
		c.stderr.Close()
	}
}

// MCPTool implements the Tool interface for dynamic MCP-based tools.
type MCPTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
	mcpClient     *MCPClient
}

// NewMCPTool creates a new MCPTool instance. It expects ToolData to have a "mcp_command" in Configuration.
func NewMCPTool(name, description string, configuration map[string]string, logger *zap.Logger) *MCPTool {

	return &MCPTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}

}

func (t *MCPTool) StartMCPClient() error {
	command, ok := t.configuration["command"]
	if !ok {
		return fmt.Errorf("mcp_command not found in configuration")
	}
	workspace, ok := t.configuration["workspace"]
	if !ok {
		var err error
		workspace, err = os.Getwd()
		if err != nil {
			return fmt.Errorf("could not get current directory: %v", err)
		}
	}
	args := []string{}
	if argStr, ok := t.configuration["args"]; ok {
		args = strings.Split(argStr, " ")
	}
	t.mcpClient = NewMCPClient(command, workspace, args)
	return nil
}

func (t *MCPTool) Name() string {
	return t.name
}

func (t *MCPTool) Description() string {
	return t.description
}

func (t *MCPTool) FullDescription() string {
	var b strings.Builder
	b.WriteString(t.Description() + "\n\n")
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}
	return b.String()
}

func (t *MCPTool) Configuration() map[string]string {
	t.StartMCPClient()
	return t.configuration
}

func (t *MCPTool) UpdateConfiguration(config map[string]string) {
	maps.Copy(t.configuration, config)
	t.StartMCPClient()
}

func (t *MCPTool) Parameters() []entities.Parameter {
	result, err := t.mcpClient.InvokeMethod("list_tools", nil)
	if err != nil {
		t.logger.Error("Error invoking list_tools", zap.Error(err))
		return nil
	}

	toolsArray, ok := result.([]any)
	if !ok {
		t.logger.Error("Invalid response format for list_tools")
		return nil
	}

	for _, toolInterface := range toolsArray {
		tool, ok := toolInterface.(map[string]any)
		if !ok {
			continue
		}
		if toolName, ok := tool["name"].(string); ok && toolName == t.Name() {
			// Found the matching tool, extract parameters
			paramsSchema, ok := tool["parameters"].(map[string]any)
			if !ok {
				t.logger.Error("Invalid parameters schema for tool", zap.String("tool_name", t.Name()))
				return nil
			}
			// Assume parameters schema has "properties" and "required"
			properties, propOk := paramsSchema["properties"].(map[string]any)
			requiredArray, reqOk := paramsSchema["required"].([]any)
			if !propOk || !reqOk {
				t.logger.Error("Invalid properties or required fields in parameters schema")
				return nil
			}

			// Create a set of required fields for easy lookup
			requiredSet := make(map[string]bool)
			for _, reqItem := range requiredArray {
				if reqStr, ok := reqItem.(string); ok {
					requiredSet[reqStr] = true
				}
			}

			// Extract and convert properties to []Parameter
			var parameters []entities.Parameter
			for name, propInterface := range properties {
				prop, ok := propInterface.(map[string]any)
				if !ok {
					continue // Skip invalid property
				}
				paramType, _ := prop["type"].(string)
				description, _ := prop["description"].(string)
				param := entities.Parameter{
					Name:        name,
					Type:        paramType,
					Description: description,
					Required:    requiredSet[name],
				} // Note: Enum and Items not handled for simplicity; extend if needed
				parameters = append(parameters, param)
			}
			return parameters
		}
	}
	t.logger.Error("Tool not found in MCP server", zap.String("tool_name", t.Name()))
	return nil
}

func (t *MCPTool) Execute(arguments string) (string, error) {
	t.logger.Debug("Executing MCP tool", zap.String("arguments", arguments))
	fmt.Println("\rExecuting MCP tool", arguments)

	// Parse the arguments string as JSON to get params map
	var params map[string]any
	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return "", fmt.Errorf("error parsing arguments JSON: %w", err)
	}

	// Call the MCP method with the tool's name
	result, err := t.mcpClient.InvokeMethod(t.Name(), params)
	if err != nil {
		return "", fmt.Errorf("error executing MCP method: %w", err)
	}

	// Marshal the result to string and return
	resultJSON, err := json.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("error marshaling result: %w", err)
	}
	return string(resultJSON), nil
}

var _ entities.Tool = (*MCPTool)(nil)
