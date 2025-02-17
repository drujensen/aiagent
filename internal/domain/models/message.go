package models

// IMessage defines the interface for messages
// It provides methods to access common message properties

type IMessage interface {
    GetID() string
    GetContent() string
}

// ChatMessage represents a basic chat message
// between a user and the system

type ChatMessage struct {
    ID        string
    Content   string
    Sender    string
    Timestamp int64
}

func (m ChatMessage) GetID() string {
    return m.ID
}

func (m ChatMessage) GetContent() string {
    return m.Content
}

// ToolMessage represents a message involving a tool call
// It includes the tool name, request, and result

type ToolMessage struct {
    ID        string
    ToolName  string
    Request   string
    Result    string
    Timestamp int64
}

func (m ToolMessage) GetID() string {
    return m.ID
}

func (m ToolMessage) GetContent() string {
    return m.Request
}
