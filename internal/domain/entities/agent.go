package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Agent struct {
	ID           primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name         string             `json:"name" bson:"name"`
	Endpoint     string             `json:"endpoint" bson:"endpoint"`
	Model        string             `json:"model" bson:"model"`
	APIKey       string             `json:"api_key" bson:"api_key"`
	SystemPrompt string             `json:"system_prompt" bson:"system_prompt"`
	Temperature  *float64           `json:"temperature,omitempty" bson:"temperature,omitempty"`
	MaxTokens    *int               `json:"max_tokens,omitempty" bson:"max_tokens,omitempty"`
	Tools        []string           `json:"tools,omitempty" bson:"tools,omitempty"`
	CreatedAt    time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at" bson:"updated_at"`
}

func NewAgent(name, endpoint, model, apiKey, systemPrompt string, tools []string) *Agent {
	return &Agent{
		ID:           primitive.NewObjectID(),
		Name:         name,
		Endpoint:     endpoint,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}
