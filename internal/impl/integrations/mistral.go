package integrations

import (
	"crypto/rand"

	"github.com/drujensen/aiagent/internal/domain/entities"
	"github.com/drujensen/aiagent/internal/domain/interfaces"

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

// customizeToolCallID generates a valid 9-character alphanumeric tool call ID for Mistral
func (m *MistralIntegration) customizeToolCallID(originalID string) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, 9)
	rand.Read(b)
	for i := range b {
		b[i] = charset[b[i]%byte(len(charset))]
	}
	return string(b)
}

var _ interfaces.AIModelIntegration = (*MistralIntegration)(nil)
