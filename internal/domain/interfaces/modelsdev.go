package interfaces

import "time"

// ModelsDevClient abstracts access to the models.dev model catalog so domain
// services can depend on it without importing internal/impl/modelsdev.
// Implemented by internal/impl/modelsdev.ModelsDevClient.
type ModelsDevClient interface {
	Fetch() (*ModelsDevResponse, error)
	GetCached() (*ModelsDevResponse, error)
	Refresh() error
	ShouldRefresh() bool
	GetLastRefreshTime() (*time.Time, error)
}

// ModelsDevResponse is the models.dev catalog response, keyed by provider ID.
type ModelsDevResponse map[string]ProviderData

// ProviderData describes one provider's entry in the models.dev catalog.
type ProviderData struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Type    string               `json:"type"`
	BaseURL string               `json:"api"`
	Env     []string             `json:"env"`
	Models  map[string]ModelData `json:"models"`
}

// ModelData describes one model's entry in the models.dev catalog.
type ModelData struct {
	ID               string    `json:"id"`
	Name             string    `json:"name"`
	Family           string    `json:"family"`
	Attachment       bool      `json:"attachment"`
	Reasoning        bool      `json:"reasoning"`
	ToolCall         bool      `json:"tool_call"`
	Temperature      bool      `json:"temperature"`
	StructuredOutput bool      `json:"structured_output"`
	Cost             CostData  `json:"cost"`
	Limit            LimitData `json:"limit"`
	ReleaseDate      string    `json:"release_date"`
}

// CostData describes a model's per-token pricing.
type CostData struct {
	Input      float64  `json:"input"`
	Output     float64  `json:"output"`
	CacheRead  *float64 `json:"cache_read,omitempty"`
	CacheWrite *float64 `json:"cache_write,omitempty"`
}

// LimitData describes a model's context/output token limits.
type LimitData struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}
