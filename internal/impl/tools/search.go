package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"aiagent/internal/domain/entities"

	"go.uber.org/zap"
)

// SearchTool represents a tool for searching using the Tavily API.
type SearchTool struct {
	name          string
	description   string
	configuration map[string]string
	logger        *zap.Logger
}

// NewSearchTool creates a new instance of SearchTool.
func NewSearchTool(name, description string, configuration map[string]string, logger *zap.Logger) *SearchTool {
	return &SearchTool{
		name:          name,
		description:   description,
		configuration: configuration,
		logger:        logger,
	}
}

func (t *SearchTool) Name() string {
	return t.name
}

func (t *SearchTool) Description() string {
	return t.description
}

func (t *SearchTool) Configuration() map[string]string {
	return t.configuration
}

func (t *SearchTool) Parameters() []entities.Parameter {
	return []entities.Parameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	}
}

// Execute performs the search and returns both the answer and results.
func (t *SearchTool) Execute(arguments string) (string, error) {
	// Log the search query
	t.logger.Debug("Executing search", zap.String("arguments", arguments))

	// Parse the arguments to extract the query
	type args struct {
		Query string `json:"query"`
	}
	var query string
	var argumentsArgs args

	if err := json.Unmarshal([]byte(arguments), &argumentsArgs); err != nil {
		// If unmarshaling into struct fails, try as a simple string
		if err := json.Unmarshal([]byte(arguments), &query); err != nil {
			t.logger.Error("Failed to parse arguments", zap.Error(err))
			return "", err
		}
	} else {
		query = argumentsArgs.Query
	}

	t.logger.Info("Search query", zap.String("query", query))
	if query == "" {
		t.logger.Error("Search query cannot be empty")
		return "", fmt.Errorf("search query cannot be empty")
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
		t.logger.Error("Failed to parse search response", zap.Error(err))
		return "", err
	}

	// Build the output string
	var output string
	if result.Answer != "" {
		output = "Answer: " + result.Answer + "\n\n"
	}
	if len(result.Results) > 0 {
		if output != "" {
			output += "Top Results:\n"
		} else {
			output = "Top Results:\n"
		}
		for i, res := range result.Results {
			if i >= 3 {
				break
			}
			output += fmt.Sprintf("%d. %s - %s\n   %s\n", i+1, res.Title, res.URL, res.Content)
		}
	} else if output == "" {
		output = "No results found"
	}

	t.logger.Info("Search completed", zap.String("result", output))
	return output, nil
}

var _ entities.Tool = (*SearchTool)(nil)
