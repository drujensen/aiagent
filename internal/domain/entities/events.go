package entities

import (
	"time"

	"github.com/google/uuid"
)

// ToolCallEvent represents an event when a tool is called
type ToolCallEvent struct {
	ID         string            `json:"id" bson:"_id"`
	ToolCallID string            `json:"tool_call_id" bson:"tool_call_id"`
	ToolName   string            `json:"tool_name" bson:"tool_name"`
	Arguments  string            `json:"arguments" bson:"arguments"`
	Result     string            `json:"result" bson:"result"`
	Error      string            `json:"error,omitempty" bson:"error,omitempty"`
	Diff       string            `json:"diff,omitempty" bson:"diff,omitempty"`
	Timestamp  time.Time         `json:"timestamp" bson:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty" bson:"metadata,omitempty"`
}

// NewToolCallEvent creates a new tool call event
func NewToolCallEvent(toolCallID, toolName, arguments, result, errorMsg, diff string, metadata map[string]string) *ToolCallEvent {
	return &ToolCallEvent{
		ID:         uuid.New().String(),
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  arguments,
		Result:     result,
		Error:      errorMsg,
		Diff:       diff,
		Timestamp:  time.Now(),
		Metadata:   metadata,
	}
}
