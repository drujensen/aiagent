package tools

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"aiagent/internal/domain/interfaces"

	"go.uber.org/zap"
)

type SearchTool struct {
	configuration map[string]string
	logger        *zap.Logger
}

func NewSearchTool(configuration map[string]string, logger *zap.Logger) *SearchTool {
	return &SearchTool{
		configuration: configuration,
		logger:        logger,
	}
}

func (t *SearchTool) Name() string {
	return "Search"
}

func (t *SearchTool) Description() string {
	return "A tool to search for information using DuckDuckGo API"
}

func (t *SearchTool) Configuration() []string {
	return []string{}
}

func (t *SearchTool) Parameters() []interfaces.Parameter {
	return []interfaces.Parameter{
		{
			Name:        "query",
			Type:        "string",
			Description: "The search query",
			Required:    true,
		},
	}
}

func (t *SearchTool) Execute(arguments string) (string, error) {
	// Log the search query
	t.logger.Debug("Executing search", zap.String("arguments", arguments))

	var query string
	// Try to unmarshal as a plain string
	if err := json.Unmarshal([]byte(arguments), &query); err != nil {
		// If that fails, try unmarshaling as a JSON object
		var args struct {
			Query string `json:"query"`
		}
		if err := json.Unmarshal([]byte(arguments), &args); err != nil {
			t.logger.Error("Failed to parse arguments", zap.Error(err))
			return "", err
		}
		query = args.Query
	}
	t.logger.Info("Search query", zap.String("query", query))
	if query == "" {
		t.logger.Error("Search query cannot be empty")
		return "", nil
	}

	apiURL := "https://api.duckduckgo.com/?q=" + url.QueryEscape(query) + "&format=json"

	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		t.logger.Error("Failed to create HTTP request", zap.Error(err))
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		t.logger.Error("Failed to execute search request", zap.Error(err))
		return "", err
	}
	defer resp.Body.Close()

	// Read the entire response body
	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		t.logger.Error("Failed to read response body", zap.Error(err))
		return "", err
	}
	bodyString := string(bodyBytes)
	t.logger.Debug("Search API response body", zap.String("body", bodyString))

	t.logger.Debug("Search API response", zap.Int("status_code", resp.StatusCode))
	t.logger.Debug("Search API response headers", zap.Any("headers", resp.Header))

	if resp.StatusCode == http.StatusFound || resp.StatusCode == http.StatusMovedPermanently {
		location, err := resp.Location()
		if err != nil {
			t.logger.Error("Failed to get redirect location", zap.Error(err))
			return "", err
		}
		t.logger.Info("Search resulted in redirect", zap.String("location", location.String()))
		return fmt.Sprintf("Redirect detected to: %s", location), nil
	}

	if resp.StatusCode != http.StatusOK {
		t.logger.Error("Search API request failed", zap.Int("status_code", resp.StatusCode))
		return "", fmt.Errorf("API request failed with status code: %d", resp.StatusCode)
	}

	// Check Content-Type
	contentType := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(contentType, "application/json") {
		t.logger.Error("Unexpected Content-Type", zap.String("content_type", contentType))
		return "", fmt.Errorf("unexpected Content-Type: %s", contentType)
	}

	var result struct {
		AbstractText  string `json:"AbstractText"`
		RelatedTopics []struct {
			Text string `json:"Text"`
		} `json:"RelatedTopics"`
	}

	if err := json.Unmarshal(bodyBytes, &result); err != nil {
		t.logger.Error("Failed to parse search response", zap.Error(err))
		return "", err
	}

	if result.AbstractText != "" {
		t.logger.Info("Search completed with abstract", zap.String("result", result.AbstractText))
		return result.AbstractText, nil
	}

	if len(result.RelatedTopics) > 0 {
		var topics []string
		for _, topic := range result.RelatedTopics {
			topics = append(topics, topic.Text)
		}
		resultStr := "Related topics: " + fmt.Sprint(topics)
		t.logger.Info("Search completed with related topics", zap.String("result", resultStr))
		return resultStr, nil
	}

	t.logger.Info("Search completed with no results")
	return "No results found", nil
}
