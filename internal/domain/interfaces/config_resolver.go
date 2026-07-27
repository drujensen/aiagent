package interfaces

// ConfigResolver resolves environment-variable references and tool
// configuration maps. Implemented by internal/impl/config.Config.
type ConfigResolver interface {
	ResolveEnvironmentVariable(value string) (string, error)
	ResolveConfiguration(config map[string]string) (map[string]string, error)
}
