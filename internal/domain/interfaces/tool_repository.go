package interfaces

type ToolRepository interface {
	GetToolByName(name string) (*ToolIntegration, error)
	ListTools() ([]*ToolIntegration, error)
}
