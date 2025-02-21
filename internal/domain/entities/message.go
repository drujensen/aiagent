package entities

import "time"

// MessageType represents the type of a message in a conversation.
// It is stored as a string in MongoDB for simplicity and readability.
type MessageType string

// Constants defining valid MessageType values
const (
	ChatMessageType MessageType = "chat" // Message for user-agent communication
	ToolMessageType MessageType = "tool" // Message for tool invocation results
)

// Message represents a single message within a conversation in the workflow automation platform.
// It is embedded as a subdocument in the 'conversations' collection in MongoDB and supports
// both chat interactions and tool execution logs.
//
// Key features:
// - Dual-purpose: Handles chat messages (Content, Sender) and tool messages (ToolName, Request, Result).
// - Type safety: Uses MessageType enum to differentiate message purposes.
//
// Relationships:
// - Embedded in Conversation (many-to-one).
type Message struct {
	ID        string      `bson:"id"`                  // Unique identifier for the message within its conversation
	Type      MessageType `bson:"type"`                // Type of message (chat or tool), required
	Content   string      `bson:"content,omitempty"`   // Text content for chat messages, optional
	Sender    string      `bson:"sender,omitempty"`    // ID of the sender (user or AIAgent), optional for chat messages
	ToolName  string      `bson:"tool_name,omitempty"` // Name of the tool used, optional for tool messages
	Request   string      `bson:"request,omitempty"`   // Input provided to the tool, optional for tool messages
	Result    string      `bson:"result,omitempty"`    // Output from the tool execution, optional for tool messages
	Timestamp time.Time   `bson:"timestamp"`           // Time the message was created, required
}

// Notes:
// - For Type="chat", Content and Sender are populated; for Type="tool", ToolName, Request, and Result are used.
// - ID is included for tracking but not indexed uniquely in MongoDB as it’s scoped to a Conversation.
// - Edge cases: Sanitization of Content occurs in the ChatService to prevent HTML injection.
// - Assumption: Timestamp is in UTC for consistency across the application.
// - Limitation: No explicit support for message edits; new messages are created for corrections.
