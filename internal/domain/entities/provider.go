package entities

import (
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
)

type ProviderType string

const (
	ProviderOpenAI    ProviderType = "openai"
	ProviderAnthropic ProviderType = "anthropic"
	ProviderXAI       ProviderType = "xai"
	ProviderGoogle    ProviderType = "google"
	ProviderDeepseek  ProviderType = "deepseek"
	ProviderTogether  ProviderType = "together"
	ProviderOllama    ProviderType = "ollama"
	ProviderGeneric   ProviderType = "generic"
)

// ModelPricing represents the cost structure for a specific model
type ModelPricing struct {
	Name                string  `json:"name" bson:"name"`                                     // Model name (e.g., "gpt-4o", "claude-3-opus")
	InputPricePerMille  float64 `json:"input_price_per_mille" bson:"input_price_per_mille"`   // Cost per 1000 input tokens
	OutputPricePerMille float64 `json:"output_price_per_mille" bson:"output_price_per_mille"` // Cost per 1000 output tokens
	ContextWindow       int     `json:"context_window" bson:"context_window"`                 // Maximum context length in tokens
}

// Provider represents an AI model provider
type Provider struct {
	ID         primitive.ObjectID `json:"id" bson:"_id,omitempty"`
	Name       string             `json:"name" bson:"name"`
	Type       ProviderType       `json:"type" bson:"type"`
	BaseURL    string             `json:"base_url" bson:"base_url"`
	APIKeyName string             `json:"api_key_name" bson:"api_key_name"` // Name to display for the API key field
	Models     []ModelPricing     `json:"models" bson:"models"`
	CreatedAt  time.Time          `json:"created_at" bson:"created_at"`
	UpdatedAt  time.Time          `json:"updated_at" bson:"updated_at"`
}

// NewProvider creates a new provider with the specified attributes
func NewProvider(name string, providerType ProviderType, baseURL, apiKeyName string, models []ModelPricing) *Provider {
	return &Provider{
		ID:         primitive.NewObjectID(),
		Name:       name,
		Type:       providerType,
		BaseURL:    baseURL,
		APIKeyName: apiKeyName,
		Models:     models,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

// GetModelPricing returns pricing information for a specific model
func (p *Provider) GetModelPricing(modelName string) *ModelPricing {
	for _, model := range p.Models {
		if model.Name == modelName {
			return &model
		}
	}
	return nil
}
