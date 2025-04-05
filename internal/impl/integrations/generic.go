package integrations

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// GenericIntegration implements the Generic API
// For now, we'll use Base implementation as DeepSeek uses an Base-compatible API,
// but in the future this could have DeepSeek-specific customizations
type GenericIntegration struct {
	*AIModelIntegration
}

// NewGenericIntegration creates a new DeepSeek integration
func NewGenericIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*GenericIntegration, error) {
	openAIIntegration, err := NewAIModelIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &GenericIntegration{
		AIModelIntegration: openAIIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *GenericIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderGeneric
}

// Ensure GenericIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*GenericIntegration)(nil)
