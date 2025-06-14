package integrations

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// TogetherIntegration implements the Together API
// For now, we'll use Base implementation as DeepSeek uses an Base-compatible API,
// but in the future this could have DeepSeek-specific customizations
type TogetherIntegration struct {
	*AIModelIntegration
}

// NewTogetherIntegration creates a new DeepSeek integration
func NewTogetherIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*TogetherIntegration, error) {
	togetherIntegration, err := NewAIModelIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &TogetherIntegration{
		AIModelIntegration: togetherIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *TogetherIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderTogether
}

// Ensure TogetherIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*TogetherIntegration)(nil)
