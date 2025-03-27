package config

import (
	"fmt"
	"strings"
	"sync"

	"github.com/spf13/viper"
	"go.uber.org/zap"
)

type Config struct {
	MongoURI      string `mapstructure:"MONGO_URI"`
	Workspace     string `mapstructure:"WORKSPACE"`
	TavilyAPIKey  string `mapstructure:"TAVILY_API_KEY"`
	BasicAuthUser string `mapstructure:"BASIC_AUTH_USER"`
	BasicAuthPass string `mapstructure:"BASIC_AUTH_PASS"`
	MCPServerURL  string `mapstructure:"MCP_SERVER_URL"`
	logger        *zap.Logger
	viper         *viper.Viper
}

var (
	configInstance *Config
	once           sync.Once
)

func InitConfig() (*Config, error) {
	var initErr error

	once.Do(func() {
		logger, err := zap.NewProduction()
		if err != nil {
			logger = zap.NewNop()
			initErr = fmt.Errorf("failed to initialize logger: %w", err)
		}
		defer logger.Sync()

		v := viper.New()
		v.SetConfigName(".env")
		v.SetConfigType("env")
		v.AddConfigPath(".")
		v.AutomaticEnv()

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

		configInstance = &Config{
			logger: logger,
			viper:  v,
		}

		if err := v.Unmarshal(configInstance); err != nil {
			initErr = fmt.Errorf("failed to unmarshal config: %w", err)
			logger.Error("Config unmarshal error", zap.Error(err))
			return
		}

		if configInstance.MongoURI == "" {
			initErr = fmt.Errorf("MONGO_URI is required but not set")
			logger.Error("Missing required configuration", zap.String("field", "MONGO_URI"))
			return
		}

		if configInstance.TavilyAPIKey == "" {
			logger.Warn("TAVILY_API_KEY not set. Search tool will not work")
		}

		if configInstance.Workspace == "" {
			logger.Warn("WORKSPACE not set. Search tool will not work")
		}

		if configInstance.MCPServerURL == "" {
			logger.Warn("MCP_SERVER_URL not set. MCP tool will not function")
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

func (c *Config) ResolveAPIKey(apiKey string) (string, error) {
	const prefix, suffix = "#{", "}#"
	if strings.HasPrefix(apiKey, prefix) && strings.HasSuffix(apiKey, suffix) {
		varName := strings.TrimSuffix(strings.TrimPrefix(apiKey, prefix), suffix)
		if varName == "" {
			return "", fmt.Errorf("empty variable name in API key reference: %s", apiKey)
		}

		resolved := c.viper.GetString(varName) // Use c.viper instead of viper
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

	c.logger.Debug("Using raw API key value", zap.String("value", maskKey(apiKey)))
	return apiKey, nil
}

func maskKey(key string) string {
	if len(key) <= 4 {
		return strings.Repeat("*", len(key))
	}
	return strings.Repeat("*", len(key)-4) + key[len(key)-4:]
}
