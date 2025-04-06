package interfaces

type Item struct {
	Type string
}

type Parameter struct {
	Name        string
	Type        string
	Enum        []string
	Items       []Item
	Description string
	Required    bool
}

type ToolIntegration interface {
	Name() string
	Description() string
	Parameters() []Parameter
	Execute(arguments string) (string, error)
}

type ToolRepository interface {
	ListTools() ([]*ToolIntegration, error)
	GetToolByName(name string) (*ToolIntegration, error)
	RegisterTool(name string, tool *ToolIntegration) error
}
