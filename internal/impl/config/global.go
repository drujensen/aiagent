package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"go.uber.org/zap"
)

// GlobalConfig represents the global configuration settings
type GlobalConfig struct {
	DefaultTemperature    float64                         `json:"default_temperature"`
	DefaultMaxTokensRatio float64                         `json:"default_max_tokens_ratio"`
	LastUsedAgent         string                          `json:"last_used_agent"` // Agent name (not ID)
	LastUsedModel         string                          `json:"last_used_model"` // Model name (not ID)
	Providers             map[string]CustomProviderConfig `json:"providers,omitempty"`
}

// CustomProviderConfig represents a custom provider configuration
type CustomProviderConfig struct {
	Name       string                       `json:"name"`
	Type       string                       `json:"type"`
	BaseURL    string                       `json:"base_url"`
	APIKeyName string                       `json:"api_key_name"`
	Models     map[string]CustomModelConfig `json:"models"`
}

// CustomModelConfig represents a custom model configuration
type CustomModelConfig struct {
	Name                string  `json:"name"`
	ContextWindow       int     `json:"context_window"`
	InputPricePerMille  float64 `json:"input_price_per_mille"`
	OutputPricePerMille float64 `json:"output_price_per_mille"`
	MaxOutputTokens     int     `json:"max_output_tokens,omitempty"`
}

// DefaultGlobalConfig returns the default global configuration
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		DefaultTemperature:    0.7,
		DefaultMaxTokensRatio: 0.25,
		LastUsedAgent:         "",
		LastUsedModel:         "",
		Providers: map[string]CustomProviderConfig{
			"drujensen": {
				Name:       "Drujensen",
				Type:       "generic",
				BaseURL:    "https://ai.drujensen.com",
				APIKeyName: "DRUJENSEN_API_KEY",
				Models: map[string]CustomModelConfig{
					"qwen3-coder:latest": {
						Name:                "Qwen3 Coder",
						ContextWindow:       64000,
						InputPricePerMille:  0,
						OutputPricePerMille: 0,
					},
				},
			},
			"ollama": {
				Name:       "Ollama",
				Type:       "generic",
				BaseURL:    "http://localhost:11434",
				APIKeyName: "",
				Models: map[string]CustomModelConfig{
					"llama2:7b": {
						Name:                "Llama 2 7B",
						ContextWindow:       4096,
						InputPricePerMille:  0,
						OutputPricePerMille: 0,
					},
				},
			},
		},
	}
}

// LoadGlobalConfig loads the global configuration from ~/.aiagent/aiagent.json
func LoadGlobalConfig(logger *zap.Logger) (*GlobalConfig, error) {
	configDir := filepath.Join(os.Getenv("HOME"), ".aiagent")
	configPath := filepath.Join(configDir, "aiagent.json")

	// Start with defaults
	config := DefaultGlobalConfig()

	// Try to read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("Global config file does not exist, using defaults", zap.String("path", configPath))
			SaveGlobalConfig(config, logger) // Save defaults for future use
			return config, nil
		}
		return nil, fmt.Errorf("failed to read global config file: %w", err)
	}

	// Parse the JSON
	if err := json.Unmarshal(data, &config); err != nil {
		logger.Warn("Failed to parse global config file, using defaults", zap.Error(err), zap.String("path", configPath))
		return DefaultGlobalConfig(), nil
	}

	logger.Debug("Loaded global config", zap.String("path", configPath))
	return config, nil
}

// SaveGlobalConfig saves the global configuration to ~/.aiagent/aiagent.json
func SaveGlobalConfig(config *GlobalConfig, logger *zap.Logger) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".aiagent")
	configPath := filepath.Join(configDir, "aiagent.json")

	// Create directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Marshal to JSON
	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Write file
	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	logger.Debug("Saved global config", zap.String("path", configPath))
	return nil
}
