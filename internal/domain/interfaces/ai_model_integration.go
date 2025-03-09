package interfaces

import "aiagent/internal/domain/entities"

type AIModelIntegration interface {
	GenerateResponse(messages []*entities.Message, toolList []*ToolIntegration, options map[string]interface{}) ([]*entities.Message, error)
	GetTokenUsage() (int, error)
}
