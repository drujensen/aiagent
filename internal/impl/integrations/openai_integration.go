package integrations

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// OpenAIIntegration implements the OpenAI API
// For now, we'll use Base implementation as DeepSeek uses an Base-compatible API,
// but in the future this could have DeepSeek-specific customizations
type OpenAIIntegration struct {
	*BaseIntegration
}

// NewOpenAIIntegration creates a new DeepSeek integration
func NewOpenAIIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*OpenAIIntegration, error) {
	openAIIntegration, err := NewBaseIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &OpenAIIntegration{
		BaseIntegration: openAIIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *OpenAIIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderOpenAI
}

// Ensure OpenAIIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*OpenAIIntegration)(nil)
