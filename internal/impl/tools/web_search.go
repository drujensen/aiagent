package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/drujensen/aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

// WebSearchTool represents a tool for searching using the Tavily API.
type WebSearchTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

// NewWebSearchTool creates a new instance of WebSearchTool.
func NewWebSearchTool(name, description string, configuration map[string]string, logger *zap.Logger) *WebSearchTool {
	return &WebSearchTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *WebSearchTool) Name() string {
	return t.name
}

func (t *WebSearchTool) Description() string {
	return t.description
}

func (t *WebSearchTool) Configuration() map[string]string {
	return t.configuration
}

func (t *WebSearchTool) UpdateConfiguration(config map[string]string) {
	t.configuration = config
}

func (t *WebSearchTool) FullDescription() string {
	var b strings.Builder

	// Add description
	b.WriteString(t.Description())
	b.WriteString("\n\n")

	// Add configuration header
	b.WriteString("Configuration for this tool:\n")
	b.WriteString("| Key           | Value         |\n")
	b.WriteString("|---------------|---------------|\n")

	// Loop through configuration and add key-value pairs to the table
	for key, value := range t.Configuration() {
		b.WriteString(fmt.Sprintf("| %-13s | %-13s |\n", key, value))
	}

	return b.String()
}

func (t *WebSearchTool) Schema() map[string]any {
	return map[string]any{
		"type": "object",
		"properties": map[string]any{
			"query": map[string]any{
				"type":        "string",
				"description": "Search query",
			},
			"num_results": map[string]any{
				"type":        "integer",
				"description": "Number of results",
				"default":     10,
			},
		},
		"required": []string{"query"},
	}
}

// Execute performs the search and returns both the answer and results.
func (t *WebSearchTool) Execute(arguments string) (string, error) {
	// Log the search query
	t.logger.Debug("Executing search", zap.String("arguments", arguments))

	// Parse the arguments
	type args struct {
		Query      string `json:"query"`
		NumResults int    `json:"num_results,omitempty"`
	}
	var argumentsArgs args

	if err := json.Unmarshal([]byte(arguments), &argumentsArgs); err != nil {
		return `{"results": [], "error": "failed to parse arguments"}`, nil
	}

	query := argumentsArgs.Query
	numResults := argumentsArgs.NumResults
	if numResults == 0 {
		numResults = 10
	}
	if numResults > 30 {
		numResults = 30
	}

	if query == "" {
		return `{"results": [], "error": "query is required"}`, nil
	}

	// Get the Tavily API key from configuration
	apiKey, ok := t.configuration["tavily_api_key"]
	if !ok {
		t.logger.Error("TAVILY_API_KEY not found in configuration")
		return "", fmt.Errorf("tavily_api_key not found in configuration")
	}

	// Create JSON payload for Tavily API
	payload := map[string]string{"query": query}
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		t.logger.Error("Failed to marshal payload", zap.Error(err))
		return "", err
	}

	// Set up the HTTP request
	apiURL := "https://api.tavily.com/search"
	req, err := http.NewRequest("POST", apiURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		t.logger.Error("Failed to create HTTP request", zap.Error(err))
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)

	// Execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		t.logger.Error("Failed to execute search request", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	// Read the response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.Error("Failed to read response body", zap.Error(err))
		return "", err
	}

	bodyString := string(bodyBytes)
	t.logger.Debug("Search API response body", zap.String("body", bodyString))

	// Check the response status
	if resp.StatusCode != http.StatusOK {
		t.logger.Error("Search API request failed", zap.Int("status_code", resp.StatusCode))
		return "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Define the structure for Tavily's response
	type TavilyResponse struct {
		Answer  string `json:"answer"`
		Results []struct {
			Title   string  `json:"title"`
			URL     string  `json:"url"`
			Content string  `json:"content"`
			Score   float64 `json:"score"`
		} `json:"results"`
	}

	// Parse the response
	var result TavilyResponse
	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		return `{"results": [], "error": "failed to parse API response"}`, nil
	}

	// Limit results
	if len(result.Results) > numResults {
		result.Results = result.Results[:numResults]
	}

	var grokResults []map[string]any
	for _, res := range result.Results {
		grokResults = append(grokResults, map[string]any{
			"title":   res.Title,
			"url":     res.URL,
			"snippet": res.Content,
		})
	}

	jsonResult, err := json.Marshal(map[string]any{
		"results": grokResults,
		"error":   "",
	})
	if err != nil {
		return `{"results": [], "error": "failed to marshal response"}`, nil
	}

	t.logger.Info("Web search completed", zap.String("query", query), zap.Int("results", len(grokResults)))
	return string(jsonResult), nil
}

var _ entities.Tool = (*WebSearchTool)(nil)
