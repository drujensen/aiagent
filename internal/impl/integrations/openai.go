package integrations

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// OpenAIIntegration implements the OpenAI API
// For now, we'll use Base implementation as OpenAI uses an OpenAI-compatible API,
// but in the future this could have OpenAI-specific customizations
type OpenAIIntegration struct {
	*AIModelIntegration
}

// NewOpenAIIntegration creates a new OpenAI integration
func NewOpenAIIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*OpenAIIntegration, error) {
	openAIIntegration, err := NewAIModelIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &OpenAIIntegration{
		AIModelIntegration: openAIIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *OpenAIIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderOpenAI
}

// Ensure OpenAIIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*OpenAIIntegration)(nil)
