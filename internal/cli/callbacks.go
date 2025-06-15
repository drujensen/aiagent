package cli

import (
	"context"
	"encoding/json"
)

// CallbackHandler defines the interface for handling tool execution callbacks
type CallbackHandler interface {
	OnStart(ctx context.Context, toolName string, input map[string]interface{}) context.Context
	OnEnd(ctx context.Context, toolName string, output map[string]interface{}) context.Context
	OnError(ctx context.Context, toolName string, err error) context.Context
}

// ToolCallbackHandler implements the CallbackHandler interface
type ToolCallbackHandler struct {
	cli *CLI
}

// NewToolCallbackHandler creates a new ToolCallbackHandler
func NewToolCallbackHandler(cli *CLI) *ToolCallbackHandler {
	return &ToolCallbackHandler{cli: cli}
}

// OnStart is called when a tool execution starts
func (h *ToolCallbackHandler) OnStart(ctx context.Context, toolName string, input map[string]interface{}) context.Context {
	argsJSON, err := json.Marshal(input)
	if err != nil {
		argsJSON = []byte("{}")
	}
	h.cli.DisplayToolCallMessage(toolName, string(argsJSON))
	return ctx
}

// OnEnd is called when a tool execution completes successfully
func (h *ToolCallbackHandler) OnEnd(ctx context.Context, toolName string, output map[string]interface{}) context.Context {
	resultJSON, err := json.Marshal(output)
	if err != nil {
		resultJSON = []byte("{}")
	}
	h.cli.DisplayToolMessage(toolName, "", string(resultJSON), false)
	return ctx
}

// OnError is called when a tool execution fails
func (h *ToolCallbackHandler) OnError(ctx context.Context, toolName string, err error) context.Context {
	h.cli.DisplayToolMessage(toolName, "", err.Error(), true)
	return ctx
}

// CreateCallbackHandler creates a callback handler for the CLI
func (c *CLI) CreateCallbackHandler() CallbackHandler {
	return NewToolCallbackHandler(c)
}

// RegisterCallbackHandler registers the callback handler with the tool service
func (c *CLI) RegisterCallbackHandler() {
	// handler := c.CreateCallbackHandler()
	//
	//	c.toolService.RegisterCallback(func(ctx context.Context, tool *entities.Tool, input map[string]interface{}) (map[string]interface{}, error) {
	//		ctx = handler.OnStart(ctx, tool.Name(), input)
	//		output, err := tool.Execute(ctx, input)
	//		if err != nil {
	//			return nil, handler.OnError(ctx, tool.Name(), err)
	//		}
	//		return output, handler.OnEnd(ctx, tool.Name(), output)
	//	})
}
