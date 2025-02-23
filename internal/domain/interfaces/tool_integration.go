package interfaces

type ToolIntegration interface {
	Name() string
	Execute(input string) (string, error)
}
