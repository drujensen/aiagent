package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Config holds the application-wide configuration settings loaded from the .env file.
// It provides structured access to critical environment variables used across the application.
//
// Key features:
// - Centralized configuration: Manages global settings like MongoDB URI and local API key.
// - Thread-safe access: Uses a singleton pattern with sync.Once for initialization.
//
// Dependencies:
// - github.com/spf13/viper: Used for loading and parsing the .env file.
// - go.uber.org/zap: Used for logging configuration events and errors.
type Config struct {
	MongoURI    string      `mapstructure:"MONGO_URI"`     // Connection string for MongoDB
	LocalAPIKey string      `mapstructure:"LOCAL_API_KEY"` // API key for local access control
	logger      *zap.Logger // Logger instance for config-related logs
}

// singleton instance and sync mechanism
var (
	configInstance *Config
	once           sync.Once
)

// InitConfig initializes the Viper configuration and returns a Config instance.
// It loads the .env file from the project root, binds environment variables to the Config struct,
// and validates required fields. This function is thread-safe and ensures only one instance is created.
//
// Returns:
// - *Config: The initialized configuration instance.
// - error: Any error encountered during initialization (e.g., file not found, missing required vars).
//
// Notes:
// - The .env file is expected at the project root (e.g., /path/to/aiagent/.env).
// - Required variables (MONGO_URI, LOCAL_API_KEY) must be present; otherwise, an error is returned.
// - Edge case: If .env is missing, Viper falls back to system environment variables.
// - Assumption: Logger is initialized as a production logger; can be swapped for development mode if needed.
func InitConfig() (*Config, error) {
	var initErr error

	once.Do(func() {
		// Initialize zap logger
		logger, err := zap.NewProduction()
		if err != nil {
			// Fallback to a no-op logger if initialization fails
			logger = zap.NewNop()
			initErr = fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer logger.Sync()

		// Initialize Viper
		v := viper.New()
		v.SetConfigName(".env")
		v.SetConfigType("env")
		v.AddConfigPath(".") // Look for .env in the project root

		// Automatically bind environment variables (e.g., MONGO_URI from system env if .env is absent)
		v.AutomaticEnv()

		// Attempt to read the .env file
		if err := v.ReadInConfig(); err != nil {
			if _, ok := err.(viper.ConfigFileNotFoundError); ok {
				logger.Warn("No .env file found; falling back to system environment variables")
			} else {
				initErr = fmt.Errorf("failed to read .env file: %w", err)
				logger.Error("Config file read error", zap.Error(err))
				return
			}
		} else {
			logger.Info("Successfully loaded .env file", zap.String("file", v.ConfigFileUsed()))
		}

		// Create Config instance
		configInstance = &Config{
			logger: logger,
		}

		// Unmarshal environment variables into the Config struct
		if err := v.Unmarshal(configInstance); err != nil {
			initErr = fmt.Errorf("failed to unmarshal config: %w", err)
			logger.Error("Config unmarshal error", zap.Error(err))
			return
		}

		// Validate required fields
		if configInstance.MongoURI == "" {
			initErr = fmt.Errorf("MONGO_URI is required but not set")
			logger.Error("Missing required configuration", zap.String("field", "MONGO_URI"))
			return
		}
		if configInstance.LocalAPIKey == "" {
			initErr = fmt.Errorf("LOCAL_API_KEY is required but not set")
			logger.Error("Missing required configuration", zap.String("field", "LOCAL_API_KEY"))
			return
		}

		logger.Info("Configuration initialized successfully",
			zap.String("mongo_uri", configInstance.MongoURI),
			zap.String("local_api_key", maskKey(configInstance.LocalAPIKey)))
	})

	if initErr != nil {
		return nil, initErr
	}
	if configInstance == nil {
		return nil, fmt.Errorf("configuration initialization failed unexpectedly")
	}

	return configInstance, nil
}

// GetConfig returns the singleton Config instance, initializing it if not already done.
// This is a convenience method for accessing the config without explicit initialization.
//
// Returns:
// - *Config: The configuration instance.
// - error: Any error from the initial setup (persisted from InitConfig).
func GetConfig() (*Config, error) {
	return InitConfig()
}

// ResolveAPIKey resolves an API key value, handling the #{VAR_NAME}# syntax.
// It checks if the input matches the syntax and retrieves the corresponding environment variable
// via Viper, falling back to the raw value if it's not a reference.
//
// Parameters:
// - apiKey: The API key input (e.g., "#{OPENAI_API_KEY}#" or "rawkey123").
//
// Returns:
// - string: The resolved API key value.
// - error: If the referenced variable is not found.
//
// Notes:
// - Used for agent configurations to allow dynamic API key references.
// - Edge case: Malformed syntax (e.g., "#{VAR") is treated as a raw value.
// - Assumption: Environment variables are loaded by Viper during initialization.
func (c *Config) ResolveAPIKey(apiKey string) (string, error) {
	const prefix, suffix = "#{", "}#"
	if strings.HasPrefix(apiKey, prefix) && strings.HasSuffix(apiKey, suffix) {
		varName := strings.TrimSuffix(strings.TrimPrefix(apiKey, prefix), suffix)
		if varName == "" {
			return "", fmt.Errorf("empty variable name in API key reference: %s", apiKey)
		}

		resolved := viper.GetString(varName)
		if resolved == "" {
			c.logger.Warn("Environment variable not found for API key reference",
				zap.String("reference", apiKey),
				zap.String("var_name", varName))
			return "", fmt.Errorf("environment variable '%s' not found for API key reference", varName)
		}

		c.logger.Debug("Resolved API key from environment variable",
			zap.String("var_name", varName),
			zap.String("resolved", maskKey(resolved)))
		return resolved, nil
	}

	// If no #{...}# syntax, return the raw value
	c.logger.Debug("Using raw API key value", zap.String("value", maskKey(apiKey)))
	return apiKey, nil
}

// maskKey masks all but the last 4 characters of a key for logging purposes.
// This prevents sensitive data from appearing in logs.
//
// Parameters:
// - key: The API key or sensitive string to mask.
//
// Returns:
// - string: The masked string (e.g., "****1234").
func maskKey(key string) string {
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}

// Notes:
// - The singleton pattern ensures thread-safe initialization, suitable for concurrent goroutine usage.
// - ResolveAPIKey supports the technical spec's requirement for #{VAR_NAME}# syntax in agent configs.
// - Error handling ensures missing .env files or variables are logged and reported gracefully.
// - Limitation: Does not support nested config files; only .env at root is loaded.
