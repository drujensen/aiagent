package entities

import (
	"time"

	"github.com/google/uuid"
)

// ToolEventType represents different types of tool call events
type ToolEventType uint32

const (
	ToolCallStarted ToolEventType = iota + 1
	ToolCallProgress
	ToolCallCompleted
	ToolCallError
)

// ToolCallEventData represents the data for a tool call event
type ToolCallEventData struct {
	EventID    string            `json:"event_id"`
	ChatID     string            `json:"chat_id"`
	ToolCallID string            `json:"tool_call_id"`
	ToolName   string            `json:"tool_name"`
	Arguments  string            `json:"arguments,omitempty"`
	Result     string            `json:"result,omitempty"`
	Error      string            `json:"error,omitempty"`
	Progress   string            `json:"progress,omitempty"`
	Timestamp  time.Time         `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

// Type returns the event type for the event dispatcher
func (e ToolCallEventData) Type() uint32 {
	return uint32(e.EventID[0]) // Simple hash for event type - could be improved
}

// NewToolCallStartedEvent creates a new tool call started event
func NewToolCallStartedEvent(chatID, toolCallID, toolName, arguments string) ToolCallEventData {
	return ToolCallEventData{
		EventID:    uuid.New().String(),
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Arguments:  arguments,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}
}

// NewToolCallProgressEvent creates a new tool call progress event
func NewToolCallProgressEvent(chatID, toolCallID, toolName, progress string) ToolCallEventData {
	return ToolCallEventData{
		EventID:    uuid.New().String(),
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Progress:   progress,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}
}

// NewToolCallCompletedEvent creates a new tool call completed event
func NewToolCallCompletedEvent(chatID, toolCallID, toolName, result string) ToolCallEventData {
	return ToolCallEventData{
		EventID:    uuid.New().String(),
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Result:     result,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}
}

// NewToolCallErrorEvent creates a new tool call error event
func NewToolCallErrorEvent(chatID, toolCallID, toolName, errorMsg string) ToolCallEventData {
	return ToolCallEventData{
		EventID:    uuid.New().String(),
		ChatID:     chatID,
		ToolCallID: toolCallID,
		ToolName:   toolName,
		Error:      errorMsg,
		Timestamp:  time.Now(),
		Metadata:   make(map[string]string),
	}
}
