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
	DefaultTemperature    float64 `json:"default_temperature"`
	DefaultMaxTokensRatio float64 `json:"default_max_tokens_ratio"`
}

// DefaultGlobalConfig returns the default global configuration
func DefaultGlobalConfig() *GlobalConfig {
	return &GlobalConfig{
		DefaultTemperature:    0.7,
		DefaultMaxTokensRatio: 0.25,
	}
}

// LoadGlobalConfig loads the global configuration from ~/.config/aiagent/aiagent.json
func LoadGlobalConfig(logger *zap.Logger) (*GlobalConfig, error) {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "aiagent")
	configPath := filepath.Join(configDir, "aiagent.json")

	// Start with defaults
	config := DefaultGlobalConfig()

	// Try to read the config file
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			logger.Debug("Global config file does not exist, using defaults", zap.String("path", configPath))
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

// SaveGlobalConfig saves the global configuration to ~/.config/aiagent/aiagent.json
func SaveGlobalConfig(config *GlobalConfig, logger *zap.Logger) error {
	configDir := filepath.Join(os.Getenv("HOME"), ".config", "aiagent")
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
