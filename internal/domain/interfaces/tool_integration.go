package interfaces

type Parameter struct {
	Name        string
	Type        string
	Enum        []string
	Description string
	Required    bool
}

type ToolIntegration interface {
	Name() string
	Description() string
	Configuration() []string
	Parameters() []Parameter
	Execute(arguments string) (string, error)
}
