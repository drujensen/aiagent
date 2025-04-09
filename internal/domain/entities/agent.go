package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type Agent struct {
	ID            primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name          string             `json:"name" bson:"name"`
	ProviderID    primitive.ObjectID `json:"provider_id" bson:"provider_id"`
	ProviderType  ProviderType       `json:"provider_type" bson:"provider_type"` // Denormalized for easier access
	Endpoint      string             `json:"endpoint" bson:"endpoint"`           // Will be populated automatically for known providers
	Model         string             `json:"model" bson:"model"`
	APIKey        string             `json:"api_key" bson:"api_key"`
	SystemPrompt  string             `json:"system_prompt" bson:"system_prompt"`
	Temperature   *float64           `json:"temperature,omitempty" bson:"temperature,omitempty"`
	MaxTokens     *int               `json:"max_tokens,omitempty" bson:"max_tokens,omitempty"`
	ContextWindow *int               `json:"context_window,omitempty" bson:"context_window,omitempty"`
	Tools         []string           `json:"tools,omitempty" bson:"tools,omitempty"`
	CreatedAt     time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time          `json:"updated_at" bson:"updated_at"`
}

func NewAgent(name, role string, providerID primitive.ObjectID, providerType ProviderType, endpoint, model, apiKey, systemPrompt string, tools []string) *Agent {
	return &Agent{
		ID:           primitive.NewObjectID(),
		Name:         name,
		ProviderID:   providerID,
		ProviderType: providerType,
		Endpoint:     endpoint,
		Model:        model,
		APIKey:       apiKey,
		SystemPrompt: systemPrompt,
		Tools:        tools,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func (a *Agent) FullSystemPrompt() string {
	return "Your name is " + a.Name + ". " + a.SystemPrompt
}
