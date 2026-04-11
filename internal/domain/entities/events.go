package entities

import (
	"time"

	"github.com/google/uuid"
)

// ToolCallEvent represents an event when a tool is called
type ToolCallEvent struct {
	ID         string            `json:"id" bson:"_id"`
	ChatID     string            `json:"chat_id,omitempty" bson:"chat_id,omitempty"`
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
func NewToolCallEvent(toolCallID, toolName, arguments, result, errorMsg, diff, chatID string, metadata map[string]string) *ToolCallEvent {
	return &ToolCallEvent{
		ID:         uuid.New().String(),
		ChatID:     chatID,
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

// ProcessFinishedEvent represents successful completion of message processing
type ProcessFinishedEvent struct {
	ID        string    `json:"id" bson:"_id"`
	ChatID    string    `json:"chat_id" bson:"chat_id"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

// NewProcessFinishedEvent creates a new process finished event
func NewProcessFinishedEvent(chatID string) *ProcessFinishedEvent {
	return &ProcessFinishedEvent{
		ID:        uuid.New().String(),
		ChatID:    chatID,
		Timestamp: time.Now(),
	}
}

// ProcessFailedEvent represents failed message processing
type ProcessFailedEvent struct {
	ID        string    `json:"id" bson:"_id"`
	ChatID    string    `json:"chat_id" bson:"chat_id"`
	Error     string    `json:"error" bson:"error"`
	Timestamp time.Time `json:"timestamp" bson:"timestamp"`
}

// NewProcessFailedEvent creates a new process failed event
func NewProcessFailedEvent(chatID, errorMsg string) *ProcessFailedEvent {
	return &ProcessFailedEvent{
		ID:        uuid.New().String(),
		ChatID:    chatID,
		Error:     errorMsg,
		Timestamp: time.Now(),
	}
}

// ChatUpdateEvent represents updates to chat state (usage, messages, etc.)
type ChatUpdateEvent struct {
	ID         string                 `json:"id"`
	ChatID     string                 `json:"chat_id"`
	UpdateType string                 `json:"update_type"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
}

// SubAgentEvent reports the lifecycle of a sub-agent launched by the Agent tool.
type SubAgentEvent struct {
	ID           string    `json:"id"`
	AgentName    string    `json:"agent_name"`
	Task         string    `json:"task"`
	SubChatID    string    `json:"sub_chat_id"`
	ParentChatID string    `json:"parent_chat_id"`
	Status       string    `json:"status"` // "started", "finished", "failed"
	Error        string    `json:"error,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// NewSubAgentEvent creates a SubAgentEvent.
func NewSubAgentEvent(agentName, task, subChatID, parentChatID, status, errorMsg string) *SubAgentEvent {
	return &SubAgentEvent{
		ID:           uuid.New().String(),
		AgentName:    agentName,
		Task:         task,
		SubChatID:    subChatID,
		ParentChatID: parentChatID,
		Status:       status,
		Error:        errorMsg,
		Timestamp:    time.Now(),
	}
}

// NewChatUpdateEvent creates a new chat update event
func NewChatUpdateEvent(chatID, updateType string, data map[string]interface{}) *ChatUpdateEvent {
	return &ChatUpdateEvent{
		ID:         uuid.New().String(),
		ChatID:     chatID,
		UpdateType: updateType,
		Data:       data,
		Timestamp:  time.Now(),
	}
}
