package integrations

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// GoogleIntegration implements the Google Gemini API
// For now, we'll use Base implementation as a temporary measure,
// but in a real implementation this would use the Gemini API
type GoogleIntegration struct {
	*BaseIntegration
}

// NewGoogleIntegration creates a new Google integration
func NewGoogleIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*GoogleIntegration, error) {
	openAIIntegration, err := NewBaseIntegration(baseURL+"/v1beta/openai/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &GoogleIntegration{
		BaseIntegration: openAIIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *GoogleIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderGoogle
}

// Ensure GoogleIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*GoogleIntegration)(nil)
