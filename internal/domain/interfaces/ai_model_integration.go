package interfaces

import "aiagent/internal/domain/entities"

// AIModelIntegration defines the interface for AI model providers
type AIModelIntegration interface {
	// GenerateResponse generates response(s) from the AI model
	GenerateResponse(messages []*entities.Message, toolList []*ToolIntegration, options map[string]interface{}) ([]*entities.Message, error)
	
	// GetUsage returns token usage information for billing/reporting
	GetUsage() (*entities.Usage, error)
	
	// ModelName returns the name of the model being used
	ModelName() string
	
	// ProviderType returns the type of provider
	ProviderType() entities.ProviderType
}
