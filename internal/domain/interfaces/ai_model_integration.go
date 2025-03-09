package interfaces

import "aiagent/internal/domain/entities"

type AIModelIntegration interface {
	GenerateResponse(messages []map[string]string, toolList []*ToolIntegration, options map[string]interface{}) ([]*entities.Message, error)
	GetTokenUsage() (int, error)
}
