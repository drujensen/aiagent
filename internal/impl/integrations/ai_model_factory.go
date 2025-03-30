package integrations

import (
	"fmt"

	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

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

// CreateModelIntegration creates an AI model integration based on the agent configuration
func (f *AIModelFactory) CreateModelIntegration(agent *entities.Agent, provider *entities.Provider, apiKey string) (interfaces.AIModelIntegration, error) {
	// Get the endpoint from the provider if not explicitly defined in the agent
	endpoint := agent.Endpoint
	if endpoint == "" {
		endpoint = provider.BaseURL
	}

	// Create provider-specific integration
	switch provider.Type {
	case entities.ProviderOpenAI:
		return NewOpenAIIntegration(endpoint, apiKey, agent.Model, f.toolRepo, f.logger)
	case entities.ProviderAnthropic:
		return NewAnthropicIntegration(endpoint, apiKey, agent.Model, f.toolRepo, f.logger)
	case entities.ProviderXAI:
		return NewXAIIntegration(endpoint, apiKey, agent.Model, f.toolRepo, f.logger)
	case entities.ProviderGoogle:
		return NewGoogleIntegration(endpoint, apiKey, agent.Model, f.toolRepo, f.logger)
	case entities.ProviderDeepseek:
		return NewDeepseekIntegration(endpoint, apiKey, agent.Model, f.toolRepo, f.logger)
	case entities.ProviderOllama:
		return NewOllamaIntegration(endpoint, apiKey, agent.Model, f.toolRepo, f.logger)
	case entities.ProviderGeneric:
		// For generic providers, use the OpenAI-compatible API
		return NewGenericOpenAIIntegration(endpoint, apiKey, agent.Model, f.toolRepo, f.logger)
	default:
		return nil, fmt.Errorf("unsupported provider type: %s", provider.Type)
	}
}
