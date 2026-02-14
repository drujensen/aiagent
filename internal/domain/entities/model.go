package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Model struct {
	ID              string       `json:"id" bson:"_id"`
	Name            string       `json:"name" bson:"name"`
	ProviderID      string       `json:"provider_id" bson:"provider_id"`
	ProviderType    ProviderType `json:"provider_type" bson:"provider_type"`
	ModelName       string       `json:"model_name" bson:"model_name"`
	APIKey          string       `json:"api_key" bson:"api_key"`
	Temperature     *float64     `json:"temperature,omitempty" bson:"temperature,omitempty"`
	MaxTokens       *int         `json:"max_tokens,omitempty" bson:"max_tokens,omitempty"`
	ContextWindow   *int         `json:"context_window,omitempty" bson:"context_window,omitempty"`
	ReasoningEffort string       `json:"reasoning_effort" bson:"reasoning_effort"`
	// Model capability metadata from models.dev
	Family           string    `json:"family" bson:"family"`
	Reasoning        bool      `json:"reasoning" bson:"reasoning"`
	ToolCall         bool      `json:"tool_call" bson:"tool_call"`
	TemperatureCap   bool      `json:"temperature_cap" bson:"temperature_cap"` // Renamed to avoid conflict with Temperature field
	Attachment       bool      `json:"attachment" bson:"attachment"`
	StructuredOutput bool      `json:"structured_output" bson:"structured_output"`
	CreatedAt        time.Time `json:"created_at" bson:"created_at"`
	UpdatedAt        time.Time `json:"updated_at" bson:"updated_at"`
}

func NewModel(name, providerID string, providerType ProviderType, modelName, apiKey string, temperature *float64, maxTokens, contextWindow *int, reasoningEffort string, family string, reasoning, toolCall, temperatureCap, attachment, structuredOutput bool) *Model {
	return &Model{
		ID:               uuid.New().String(),
		Name:             name,
		ProviderID:       providerID,
		ProviderType:     providerType,
		ModelName:        modelName,
		APIKey:           apiKey,
		Temperature:      temperature,
		MaxTokens:        maxTokens,
		ContextWindow:    contextWindow,
		ReasoningEffort:  reasoningEffort,
		Family:           family,
		Reasoning:        reasoning,
		ToolCall:         toolCall,
		TemperatureCap:   temperatureCap,
		Attachment:       attachment,
		StructuredOutput: structuredOutput,
		CreatedAt:        time.Now(),
		UpdatedAt:        time.Now(),
	}
}

func (m *Model) FilterValue() string {
	return m.Name + " - " + m.ModelName
}

func (m *Model) Title() string {
	return m.Name
}

func (m *Model) Description() string {
	providerInfo := fmt.Sprintf("%s/%s", m.ProviderType, m.ModelName)
	var params []string
	if m.Temperature != nil {
		params = append(params, fmt.Sprintf("temp: %.2f", *m.Temperature))
	}
	if m.MaxTokens != nil {
		params = append(params, fmt.Sprintf("max: %d", *m.MaxTokens))
	}
	if m.ContextWindow != nil {
		params = append(params, fmt.Sprintf("ctx: %d", *m.ContextWindow))
	}
	if m.ReasoningEffort != "" && m.ReasoningEffort != "none" {
		params = append(params, fmt.Sprintf("reasoning: %s", m.ReasoningEffort))
	}

	if len(params) > 0 {
		return providerInfo
	}
	return fmt.Sprintf("%s | %s", providerInfo, fmt.Sprintf("%v", params))
}
