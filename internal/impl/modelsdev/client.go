package modelsdev

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"go.uber.org/zap"
)

type ModelsDevClient struct {
	cachePath string
	logger    *zap.Logger
	client    *http.Client
}

type ModelsDevResponse map[string]ProviderData

type ProviderData struct {
	ID      string               `json:"id"`
	Name    string               `json:"name"`
	Type    string               `json:"type"`
	BaseURL string               `json:"api"`
	Env     []string             `json:"env"`
	Models  map[string]ModelData `json:"models"`
}

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

type CostData struct {
	Input      float64  `json:"input"`
	Output     float64  `json:"output"`
	CacheRead  *float64 `json:"cache_read,omitempty"`
	CacheWrite *float64 `json:"cache_write,omitempty"`
}

type LimitData struct {
	Context int `json:"context"`
	Output  int `json:"output"`
}

type ModelsDevCache struct {
	Version     int                     `json:"version"`
	Providers   map[string]ProviderData `json:"providers"`
	LastRefresh time.Time               `json:"last_refresh"`
}

const cacheVersion = 1
const cacheRefreshInterval = 24 * time.Hour

func NewModelsDevClient(logger *zap.Logger) *ModelsDevClient {
	cacheDir := filepath.Join(os.Getenv("HOME"), ".aiagent")
	return &ModelsDevClient{
		cachePath: filepath.Join(cacheDir, "providers-cache.json"),
		logger:    logger,
		client:    &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *ModelsDevClient) Fetch() (*ModelsDevResponse, error) {
	req, err := http.NewRequest("GET", "https://models.dev/api.json", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("User-Agent", "aiagent/1.0.0")
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch models.dev data: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.logger.Error("models.dev API request failed",
			zap.Int("status_code", resp.StatusCode),
			zap.String("response_body", string(body)))
		return nil, fmt.Errorf("models.dev API request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	c.logger.Debug("Raw models.dev response", zap.String("body", string(body)))

	var result ModelsDevResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to decode models.dev response: %w", err)
	}

	c.logger.Info("Successfully fetched models.dev data", zap.Int("providers", len(result)))

	// Debug: log provider keys
	if len(result) == 0 {
		c.logger.Warn("No providers found in models.dev response")
	} else {
		keys := make([]string, 0, len(result))
		for k := range result {
			keys = append(keys, k)
		}
		c.logger.Info("Available provider keys from models.dev", zap.Strings("keys", keys))
	}

	return &result, nil
}

func (c *ModelsDevClient) GetCached() (*ModelsDevResponse, error) {
	data, err := os.ReadFile(c.cachePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache ModelsDevCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	if cache.Version != cacheVersion {
		c.logger.Warn("Cache version mismatch, will refresh")
		return nil, nil
	}

	c.logger.Debug("Loaded cache", zap.Int("providers", len(cache.Providers)))
	result := ModelsDevResponse(cache.Providers)
	return &result, nil
}

func (c *ModelsDevClient) Refresh() error {
	fetched, err := c.Fetch()
	if err != nil {
		return err
	}

	cacheData := ModelsDevCache{
		Providers:   *fetched,
		LastRefresh: time.Now(),
	}

	data, err := json.MarshalIndent(cacheData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	if err := os.MkdirAll(filepath.Dir(c.cachePath), 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	if err := os.WriteFile(c.cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	c.logger.Info("Successfully refreshed provider cache", zap.Int("providers", len(*fetched)))
	return nil
}

func (c *ModelsDevClient) ShouldRefresh() bool {
	data, err := os.ReadFile(c.cachePath)
	if os.IsNotExist(err) {
		return true
	}
	if err != nil {
		c.logger.Warn("Failed to read cache, forcing refresh", zap.Error(err))
		return true
	}

	var cache ModelsDevCache
	if err := json.Unmarshal(data, &cache); err != nil {
		c.logger.Warn("Failed to unmarshal cache, forcing refresh", zap.Error(err))
		return true
	}

	refreshAge := time.Since(cache.LastRefresh)
	return refreshAge > cacheRefreshInterval
}

func (c *ModelsDevClient) GetLastRefreshTime() (*time.Time, error) {
	data, err := os.ReadFile(c.cachePath)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache ModelsDevCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	return &cache.LastRefresh, nil
}

var _ *ModelsDevClient = (*ModelsDevClient)(nil)
