package integrations

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// XAIIntegration implements the X.AI (Grok) API
// For now, we'll use OpenAI implementation as the API is similar,
// but in the future, this would have X.AI-specific customizations
type XAIIntegration struct {
	*OpenAIIntegration
}

// NewXAIIntegration creates a new X.AI integration
func NewXAIIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*XAIIntegration, error) {
	openAIIntegration, err := NewOpenAIIntegration(baseURL, apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &XAIIntegration{
		OpenAIIntegration: openAIIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *XAIIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderXAI
}

// Ensure XAIIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*XAIIntegration)(nil)
