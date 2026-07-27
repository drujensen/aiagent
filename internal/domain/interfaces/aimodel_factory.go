package interfaces

import "github.com/drujensen/aiagent/internal/domain/entities"

// AIModelFactory creates provider-specific AI model integrations.
// Implemented by internal/impl/integrations.AIModelFactory.
type AIModelFactory interface {
	CreateModelIntegration(model *entities.Model, provider *entities.Provider, apiKey string) (AIModelIntegration, error)
}
