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

type Usage struct {
	PromptTokens     int     `json:"prompt_tokens" bson:"prompt_tokens"`         // Input tokens
	CompletionTokens int     `json:"completion_tokens" bson:"completion_tokens"` // Output tokens
	TotalTokens      int     `json:"total_tokens" bson:"total_tokens"`           // Total tokens processed
	Cost             float64 `json:"cost" bson:"cost"`                           // Cost in USD
}

type Message struct {
	ID         string     `json:"id" bson:"id"`
	Role       string     `json:"role" bson:"role"`
	Content    string     `json:"content" bson:"content"`
	ImageURL   string     `json:"image_url,omitempty" bson:"image_url,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty" bson:"tool_call_id,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls" bson:"tool_calls"`
	Usage      *Usage     `json:"usage,omitempty" bson:"usage,omitempty"`
	Timestamp  time.Time  `json:"timestamp" bson:"timestamp"`
}

func NewMessage(role, content string) *Message {
	return &Message{
		ID:        uuid.New().String(),
		Role:      role,
		Content:   content,
		Timestamp: time.Now(),
	}
}

// AddUsage adds usage information to the message
func (m *Message) AddUsage(promptTokens, completionTokens int, inputCostPerMille, outputCostPerMille float64) {
	totalTokens := promptTokens + completionTokens

	// Calculate cost
	inputCost := float64(promptTokens) * inputCostPerMille / 1000000.0
	outputCost := float64(completionTokens) * outputCostPerMille / 1000000.0
	totalCost := inputCost + outputCost

	// Create or update usage
	if m.Usage == nil {
		m.Usage = &Usage{}
	}

	m.Usage.PromptTokens = promptTokens
	m.Usage.CompletionTokens = completionTokens
	m.Usage.TotalTokens = totalTokens
	m.Usage.Cost = totalCost
}
