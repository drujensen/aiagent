package integrations

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// DeepseekIntegration implements the Deepseek API
// For now, we'll use Base implementation as DeepSeek uses an Base-compatible API,
// but in the future this could have DeepSeek-specific customizations
type DeepseekIntegration struct {
	*BaseIntegration
}

// NewDeepseekIntegration creates a new DeepSeek integration
func NewDeepseekIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*DeepseekIntegration, error) {
	openAIIntegration, err := NewBaseIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &DeepseekIntegration{
		BaseIntegration: openAIIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *DeepseekIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderDeepseek
}

// Ensure DeepseekIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*DeepseekIntegration)(nil)
