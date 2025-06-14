package integrations

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// OllamaIntegration implements the Ollama API
// For now, we'll use Base implementation as Ollama can use an Base-compatible API,
// but in the future this would have Ollama-specific customizations
type OllamaIntegration struct {
	*AIModelIntegration
}

// NewOllamaIntegration creates a new Ollama integration
func NewOllamaIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*OllamaIntegration, error) {
	openAIIntegration, err := NewAIModelIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &OllamaIntegration{
		AIModelIntegration: openAIIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *OllamaIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderOllama
}

// Ensure OllamaIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*OllamaIntegration)(nil)
