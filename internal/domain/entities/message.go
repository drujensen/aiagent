package entities

import (
	"time"

	"github.com/google/uuid"
)

type ToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type Message struct {
	ID        string     `json:"id" bson:"id"`
	Role      string     `json:"role" bson:"role"`
	Content   string     `json:"content" bson:"content"`
	ToolCalls []ToolCall `json:"tool_calls" bson:"tool_calls"`
	Timestamp time.Time  `json:"timestamp" bson:"timestamp"`
}

func NewMessage(role, content string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}
