package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Agent struct {
	ID              string       `json:"id" bson:"_id"`
	Name            string       `json:"name" bson:"name"`
	ProviderID      string       `json:"provider_id" bson:"provider_id"`
	ProviderType    ProviderType `json:"provider_type" bson:"provider_type"` // Denormalized for easier access
	Endpoint        string       `json:"endpoint" bson:"endpoint"`           // Will be populated automatically for known providers
	Model           string       `json:"model" bson:"model"`
	APIKey          string       `json:"api_key" bson:"api_key"`
	SystemPrompt    string       `json:"system_prompt" bson:"system_prompt"`
	Temperature     *float64     `json:"temperature,omitempty" bson:"temperature,omitempty"`
	MaxTokens       *int         `json:"max_tokens,omitempty" bson:"max_tokens,omitempty"`
	ContextWindow   *int         `json:"context_window,omitempty" bson:"context_window,omitempty"`
	ReasoningEffort string       `json:"reasoning_effort" bson:"reasoning_effort"` // low, medium, high, or none
	Tools           []string     `json:"tools,omitempty" bson:"tools,omitempty"`
	CreatedAt       time.Time    `json:"created_at" bson:"created_at"`
	UpdatedAt       time.Time    `json:"updated_at" bson:"updated_at"`
}

func NewAgent(name, role string, providerID string, providerType ProviderType, endpoint, model, apiKey, systemPrompt string, tools []string) *Agent {
	return &Agent{
		ID:           uuid.New().String(),
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

// Implement the list.Item interface
func (a *Agent) FilterValue() string {
	return a.Name + " - " + a.Model
}

func (a *Agent) Title() string {
	return a.Name
}

func (a *Agent) Description() string {
	return fmt.Sprintf("Model: %s | Provider: %s | Prompt: %s", a.Model, a.ProviderType, a.SystemPrompt)
}

func (a *Agent) FullSystemPrompt() string {
	currentTime := time.Now()
	formattedTime := currentTime.Format("2006-01-02 15:04:05")
	return "Your name is " + a.Name + "\nCurrent date and time is " + formattedTime + "\n" + a.SystemPrompt
}
