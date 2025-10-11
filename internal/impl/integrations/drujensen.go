package integrations

import (
	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

// DrujensenIntegration implements the Drujensen API
// Uses OpenAI-compatible API
type DrujensenIntegration struct {
	*AIModelIntegration
}

// NewDrujensenIntegration creates a new Drujensen integration
func NewDrujensenIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*DrujensenIntegration, error) {
	drujensenIntegration, err := NewAIModelIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &DrujensenIntegration{
		AIModelIntegration: drujensenIntegration,
	}, nil
}

// ProviderType returns the type of provider
func (m *DrujensenIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderDrujensen
}

// Ensure DrujensenIntegration implements AIModelIntegration
var _ interfaces.AIModelIntegration = (*DrujensenIntegration)(nil)
