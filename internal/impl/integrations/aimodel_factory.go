package integrations

import (
	"fmt"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// AIModelFactory creates AI model integrations based on provider type
type AIModelFactory struct {
	toolRepo interfaces.ToolRepository
	logger   *zap.Logger
}

// NewAIModelFactory creates a new AI model factory
func NewAIModelFactory(toolRepo interfaces.ToolRepository, logger *zap.Logger) *AIModelFactory {
	return &AIModelFactory{
		toolRepo: toolRepo,
		logger:   logger,
	}
}

// CreateModelIntegration creates an AI model integration based on the model configuration
func (f *AIModelFactory) CreateModelIntegration(model *entities.Model, provider *entities.Provider, apiKey string) (interfaces.AIModelIntegration, error) {
	// Use provider's base URL as endpoint
	endpoint := provider.BaseURL

	// Create provider-specific integration
	switch provider.Type {
	case entities.ProviderOpenAI:
		return NewOpenAIIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderAnthropic:
		return NewAnthropicIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderXAI:
		return NewXAIIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderGoogle:
		return NewGoogleIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderDeepseek:
		return NewDeepseekIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderTogether:
		return NewTogetherIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderGroq:
		return NewGroqIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderMistral:
		return NewMistralIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderOllama:
		return NewOllamaIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderDrujensen:
		return NewDrujensenIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	case entities.ProviderGeneric:
		// For generic providers, use the OpenAI-compatible API
		return NewGenericIntegration(endpoint, apiKey, model.ModelName, f.toolRepo, f.logger)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}
