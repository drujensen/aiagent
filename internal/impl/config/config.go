package config

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/joho/godotenv"
	"go.uber.org/zap"
)

type Config struct {
	MongoURI string
	logger   *zap.Logger
}

var (
	configInstance *Config
	once           sync.Once
)

func InitConfig() (*Config, error) {
	var initErr error

	once.Do(func() {
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zap.WarnLevel)
		logger, err := config.Build()
		if err != nil {
			logger = zap.NewNop()
			initErr = fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer logger.Sync()

		// Load .env file
		if err := godotenv.Load(); err != nil {
			if os.IsNotExist(err) {
				logger.Warn("No .env file found; falling back to system environment variables")
			} else {
				initErr = fmt.Errorf("failed to load .env file: %w", err)
				logger.Error("Config file load error", zap.Error(err))
				return
			}
		} else {
			logger.Debug("Successfully loaded .env file")
		}

		// Read MONGO_URI from environment variables
		mongoURI := os.Getenv("MONGO_URI")
		if mongoURI == "" {
			logger.Warn("MONGO_URI not set in environment variables")
		}

		configInstance = &Config{
			MongoURI: mongoURI,
			logger:   logger,
		}
	})

	if initErr != nil {
		return nil, initErr
	}
	if configInstance == nil {
		return nil, fmt.Errorf("configuration initialization failed unexpectedly")
	}

	return configInstance, nil
}

func (c *Config) ResolveEnvironmentVariable(value string) (string, error) {
	const prefix, suffix = "#{", "}#"
	if strings.HasPrefix(value, prefix) && strings.HasSuffix(value, suffix) {
		varName := strings.TrimSuffix(strings.TrimPrefix(value, prefix), suffix)
		if varName == "" {
			return "", fmt.Errorf("empty variable name in reference: %s", value)
		}

		resolved := os.Getenv(varName)
		if resolved == "" {
			c.logger.Warn("Environment variable not found for reference",
				zap.String("reference", value),
				zap.String("var_name", varName))
			return "", fmt.Errorf("environment variable '%s' not found", varName)
		}

		c.logger.Debug("Resolved environment variable",
			zap.String("var_name", varName),
			zap.String("resolved", maskKey(resolved)))
		return resolved, nil
	}

	c.logger.Debug("Using raw value", zap.String("value", maskKey(value)))
	return value, nil
}

func (c *Config) ResolveConfiguration(config map[string]string) (map[string]string, error) {
	resolvedConfig := make(map[string]string)
	for key, value := range config {
		resolvedValue, err := c.ResolveEnvironmentVariable(value)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve configuration for key '%s': %w", key, err)
		}
		resolvedConfig[key] = resolvedValue
	}
	return resolvedConfig, nil
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}
