package services

import (
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"
)

// ModelFilterService provides filtering logic for models based on their capabilities
type ModelFilterService struct{}

// NewModelFilterService creates a new model filter service
func NewModelFilterService() *ModelFilterService {
	return &ModelFilterService{}
}

// IsChatCompatibleModel determines if a model is compatible with the chat system
// based on its capabilities and metadata
func (s *ModelFilterService) IsChatCompatibleModel(model *entities.Model) bool {
	// If model doesn't have metadata (empty Family field), assume it's compatible
	// This handles existing models that were created before metadata was added
	if model.Family == "" {
		// For models without metadata, only filter out obviously incompatible ones
		modelNameLower := strings.ToLower(model.ModelName)
		// Skip codex models based on name
		if strings.Contains(modelNameLower, "codex") {
			return false
		}
		// Skip embedding models based on name
		if strings.Contains(modelNameLower, "embedding") {
			return false
		}
		// Skip vision models based on name
		if strings.Contains(modelNameLower, "vision") {
			return false
		}
		return true
	}

	// For models with metadata, apply full filtering
	// Must support tool calls (required for chat functionality)
	if !model.ToolCall {
		return false
	}

	// Must support temperature (for now - future enhancement to handle models without temperature)
	if !model.TemperatureCap {
		return false
	}

	// Skip OpenAI o1 models (use /v1/responses API instead of /v1/chat/completions)
	if model.ProviderType == entities.ProviderOpenAI && model.Family == "o" {
		return false
	}

	// Skip codex models (code completion focused, not suitable for chat)
	if strings.Contains(model.Family, "codex") {
		return false
	}

	// Skip embedding models (not for chat)
	if strings.Contains(model.Family, "embedding") {
		return false
	}

	// Skip text-embedding models
	if strings.Contains(model.Family, "text-embedding") {
		return false
	}

	// Also skip models with "vision" in the name as a fallback
	if strings.Contains(strings.ToLower(model.ModelName), "vision") {
		return false
	}

	return true
}

// FilterChatCompatibleModels filters a list of models to only include chat-compatible ones
func (s *ModelFilterService) FilterChatCompatibleModels(models []*entities.Model) []*entities.Model {
	var filtered []*entities.Model
	for _, model := range models {
		if s.IsChatCompatibleModel(model) {
			filtered = append(filtered, model)
		}
	}
	return filtered
}
