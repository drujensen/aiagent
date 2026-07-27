package interfaces

// GlobalConfig represents the global configuration settings. Owned by the
// domain layer so services can depend on it without importing internal/impl.
type GlobalConfig struct {
	DefaultTemperature    float64                         `json:"default_temperature"`
	DefaultMaxTokensRatio float64                         `json:"default_max_tokens_ratio"`
	LastUsedAgent         string                          `json:"last_used_agent"` // Agent name (not ID)
	LastUsedModel         string                          `json:"last_used_model"` // Model name (not ID)
	Providers             map[string]CustomProviderConfig `json:"providers,omitempty"`
}

// CustomProviderConfig represents a custom provider configuration.
type CustomProviderConfig struct {
	Name       string                       `json:"name"`
	Type       string                       `json:"type"`
	BaseURL    string                       `json:"base_url"`
	APIKeyName string                       `json:"api_key_name"`
	Models     map[string]CustomModelConfig `json:"models"`
}

// CustomModelConfig represents a custom model configuration.
type CustomModelConfig struct {
	Name                string  `json:"name"`
	ContextWindow       int     `json:"context_window"`
	InputPricePerMille  float64 `json:"input_price_per_mille"`
	OutputPricePerMille float64 `json:"output_price_per_mille"`
	MaxOutputTokens     int     `json:"max_output_tokens,omitempty"`
}
