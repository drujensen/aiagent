package entities

import (
	"time"

	"github.com/google/uuid"
)

type ChatUsage struct {
	TotalPromptTokens     int     `json:"total_prompt_tokens" bson:"total_prompt_tokens"`
	TotalCompletionTokens int     `json:"total_completion_tokens" bson:"total_completion_tokens"`
	TotalTokens           int     `json:"total_tokens" bson:"total_tokens"`
	TotalCost             float64 `json:"total_cost" bson:"total_cost"` // Cost in USD
}

type Chat struct {
	ID        string     `json:"id" bson:"_id"`
	AgentID   string     `json:"agent_id" bson:"agent_id"`
	ModelID   string     `json:"model_id" bson:"model_id"`
	Name      string     `json:"name" bson:"name"`
	Messages  []Message  `json:"messages" bson:"messages"`
	Usage     *ChatUsage `json:"usage,omitempty" bson:"usage,omitempty"`
	CreatedAt time.Time  `json:"created_at" bson:"created_at"`
	UpdatedAt time.Time  `json:"updated_at" bson:"updated_at"`
	Active    bool       `json:"active" bson:"active"`
}

func NewChat(agentID, modelID, name string) *Chat {
	return &Chat{
		ID:        uuid.New().String(),
		AgentID:   agentID,
		ModelID:   modelID,
		Name:      name,
		Messages:  make([]Message, 0),
		Usage:     &ChatUsage{},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		Active:    true,
	}
}

// UpdateUsage recalculates the total usage for this chat
func (c *Chat) UpdateUsage() {
	if c.Usage == nil {
		c.Usage = &ChatUsage{}
	}

	var totalPromptTokens, totalCompletionTokens int
	var totalCost float64

	for _, msg := range c.Messages {
		if msg.Usage != nil {
			totalPromptTokens += msg.Usage.PromptTokens
			totalCompletionTokens += msg.Usage.CompletionTokens
			totalCost += msg.Usage.Cost
		}
	}

	c.Usage.TotalPromptTokens = totalPromptTokens
	c.Usage.TotalCompletionTokens = totalCompletionTokens
	c.Usage.TotalTokens = totalPromptTokens + totalCompletionTokens
	c.Usage.TotalCost = totalCost
}

// Implement list.Item interface for Bubble Tea
func (c *Chat) FilterValue() string {
	return c.Name
}

func (c *Chat) Title() string {
	return c.Name
}

func (c *Chat) Description() string {
	return c.CreatedAt.Format("2006-01-02 15:04")
}
