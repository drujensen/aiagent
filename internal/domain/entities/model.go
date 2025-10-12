package entities

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ModelCapabilities represents the capabilities of a specific model
type ModelCapabilities struct {
	SupportsTools  bool `json:"supports_tools" bson:"supports_tools"`
	SupportsImages bool `json:"supports_images" bson:"supports_images"`
	SupportsVision bool `json:"supports_vision" bson:"supports_vision"`
}

// Model represents an AI model that can be selected independently of agents
type Model struct {
	ID            string            `json:"id" bson:"_id"`
	Name          string            `json:"name" bson:"name"`
	ProviderID    string            `json:"provider_id" bson:"provider_id"`
	ProviderType  ProviderType      `json:"provider_type" bson:"provider_type"` // Denormalized for easier access
	ContextWindow int               `json:"context_window" bson:"context_window"`
	Capabilities  ModelCapabilities `json:"capabilities" bson:"capabilities"`
	CreatedAt     time.Time         `json:"created_at" bson:"created_at"`
	UpdatedAt     time.Time         `json:"updated_at" bson:"updated_at"`
}

// NewModel creates a new model with the specified attributes
func NewModel(name, providerID string, providerType ProviderType, contextWindow int, capabilities ModelCapabilities) *Model {
	return &Model{
		ID:            uuid.New().String(),
		Name:          name,
		ProviderID:    providerID,
		ProviderType:  providerType,
		ContextWindow: contextWindow,
		Capabilities:  capabilities,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}
}

// Implement the list.Item interface
func (m *Model) FilterValue() string {
	return m.Name + " - " + string(m.ProviderType)
}

func (m *Model) Title() string {
	return m.Name
}

func (m *Model) Description() string {
	return fmt.Sprintf("Provider: %s | Context: %d tokens", m.ProviderType, m.ContextWindow)
}
