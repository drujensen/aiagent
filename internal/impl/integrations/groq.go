package integrations

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// GroqIntegration implements the Groq API
// For now, we'll use Base implementation as DeepSeek uses an Base-compatible API,
// but in the future this could have DeepSeek-specific customizations
type GroqIntegration struct {
	*AIModelIntegration
}

// NewGroqIntegration creates a new DeepSeek integration
func NewGroqIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*GroqIntegration, error) {
	togetherIntegration, err := NewAIModelIntegration(baseURL+"/openai/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &GroqIntegration{
		AIModelIntegration: togetherIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *GroqIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderGroq
}

// Ensure GroqIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*GroqIntegration)(nil)
