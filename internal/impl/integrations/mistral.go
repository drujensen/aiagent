package integrations

import (
	"aiagent/internal/domain/entities"
	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type MistralIntegration struct {
	*AIModelIntegration
}

func NewMistralIntegration(baseURL, apiKey, model string, toolRepo interfaces.ToolRepository, logger *zap.Logger) (*MistralIntegration, error) {
	mistralIntegration, err := NewAIModelIntegration(baseURL+"/v1/chat/completions", apiKey, model, toolRepo, logger)
	if err != nil {
		return nil, err
	}

	return &MistralIntegration{
		AIModelIntegration: mistralIntegration,
	}, nil
}

func (m *MistralIntegration) ProviderType() entities.ProviderType {
	return entities.ProviderMistral
}

var _ interfaces.AIModelIntegration = (*MistralIntegration)(nil)
