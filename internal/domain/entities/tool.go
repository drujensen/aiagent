package entities

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

type Tool interface {
	Name() string
	Description() string
	Parameters() []Parameter
	Execute(arguments string) (string, error)
}
