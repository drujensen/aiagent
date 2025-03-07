package interfaces

type AIModelIntegration interface {
	GenerateResponse(messages []map[string]string, toolList []*ToolIntegration, options map[string]interface{}) (string, error)
	GetTokenUsage() (int, error)
}
